package api

import (
	"errors"
	"testing"
	"time"
)

func TestWithRetryDelays_SuccessOnFirstAttempt(t *testing.T) {
	callCount := 0
	err := WithRetryDelays(func() error {
		callCount++
		return nil
	}, 3, []time.Duration{1 * time.Millisecond})

	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestWithRetryDelays_SuccessAfterRetry(t *testing.T) {
	callCount := 0
	err := WithRetryDelays(func() error {
		callCount++
		if callCount < 3 {
			return ErrRateLimited
		}
		return nil
	}, 3, []time.Duration{1 * time.Millisecond, 1 * time.Millisecond})

	if err != nil {
		t.Errorf("Expected nil error after retry, got: %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls (2 failures + 1 success), got %d", callCount)
	}
}

func TestWithRetryDelays_GivesUpAfterMaxRetries(t *testing.T) {
	callCount := 0
	err := WithRetryDelays(func() error {
		callCount++
		return ErrRateLimited
	}, 3, []time.Duration{1 * time.Millisecond})

	if err == nil {
		t.Fatal("Expected error after max retries, got nil")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("Expected ErrRateLimited, got: %v", err)
	}
	// 4 attempts: initial + 3 retries
	if callCount != 4 {
		t.Errorf("Expected 4 calls (1 initial + 3 retries), got %d", callCount)
	}
}

func TestWithRetryDelays_NonRateLimitedErrorReturnsImmediately(t *testing.T) {
	callCount := 0
	otherErr := errors.New("some other error")
	err := WithRetryDelays(func() error {
		callCount++
		return otherErr
	}, 3, []time.Duration{1 * time.Millisecond})

	if err != otherErr {
		t.Errorf("Expected original error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call (no retries for non-rate-limit errors), got %d", callCount)
	}
}

func TestWithRetryDelays_UsesLastDelayWhenAttemptsExceedDelayLength(t *testing.T) {
	callCount := 0
	delays := []time.Duration{1 * time.Millisecond, 2 * time.Millisecond}

	start := time.Now()
	_ = WithRetryDelays(func() error {
		callCount++
		return ErrRateLimited
	}, 3, delays)
	elapsed := time.Since(start)

	// Should have delays: 1ms, 2ms, 2ms (uses last delay when index exceeds length)
	// Total: 5ms minimum
	if elapsed < 5*time.Millisecond {
		t.Errorf("Expected at least 5ms total delay, got %v", elapsed)
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{3, 3, 3},
		{0, 5, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestWithRetry_CallsWithRetryDelays(t *testing.T) {
	callCount := 0
	err := WithRetry(func() error {
		callCount++
		return nil
	}, 0)

	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestWithRetryDelays_403RateLimitTriggersRetry(t *testing.T) {
	callCount := 0
	rateLimitErr := &httpStatusError{code: 403, msg: "API rate limit exceeded"}
	err := WithRetryDelays(func() error {
		callCount++
		if callCount < 3 {
			return rateLimitErr
		}
		return nil
	}, 3, []time.Duration{1 * time.Millisecond})

	if err != nil {
		t.Errorf("Expected nil error after retry, got: %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestWithRetryDelays_429WithRetryAfterUsesHeaderValue(t *testing.T) {
	callCount := 0
	retryAfterErr := &httpStatusRetryAfterError{code: 429, msg: "Too Many Requests", retryAfter: "1"}
	start := time.Now()
	err := WithRetryDelays(func() error {
		callCount++
		if callCount < 2 {
			return retryAfterErr
		}
		return nil
	}, 3, []time.Duration{100 * time.Millisecond})
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Expected nil error after retry, got: %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}
	// Retry-After says 1s, which should override the 100ms default delay
	if elapsed < 900*time.Millisecond {
		t.Errorf("Expected at least ~1s delay from Retry-After, got %v", elapsed)
	}
}

func TestWithRetryDelays_NonRateLimitedPermissionDeniedNoRetry(t *testing.T) {
	callCount := 0
	permErr := &httpStatusError{code: 403, msg: "Resource not accessible by integration"}
	err := WithRetryDelays(func() error {
		callCount++
		return permErr
	}, 3, []time.Duration{1 * time.Millisecond})

	if err != permErr {
		t.Errorf("Expected original permission error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call (no retry for permission denied), got %d", callCount)
	}
}

func TestWithRetryDelays_MaxRetriesExhaustedClearMessage(t *testing.T) {
	rateLimitErr := &httpStatusError{code: 429, msg: "Too Many Requests"}
	callCount := 0
	err := WithRetryDelays(func() error {
		callCount++
		return rateLimitErr
	}, 3, []time.Duration{1 * time.Millisecond})

	if err == nil {
		t.Fatal("Expected error after max retries, got nil")
	}
	if !IsRateLimited(err) {
		t.Errorf("Expected rate-limited error, got: %v", err)
	}
	if callCount != 4 {
		t.Errorf("Expected 4 calls (1 initial + 3 retries), got %d", callCount)
	}
}

func TestWithRetryDelays_504TriggersRetry(t *testing.T) {
	callCount := 0
	gatewayErr := &httpStatusError{code: 504, msg: "Gateway Timeout"}
	err := WithRetryDelays(func() error {
		callCount++
		if callCount < 3 {
			return gatewayErr
		}
		return nil
	}, 3, []time.Duration{1 * time.Millisecond})

	if err != nil {
		t.Errorf("Expected nil error after retry, got: %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls (2 504s + 1 success), got %d", callCount)
	}
}

func TestWithRetryDelays_502TriggersRetry(t *testing.T) {
	callCount := 0
	badGatewayErr := &httpStatusError{code: 502, msg: "Bad Gateway"}
	err := WithRetryDelays(func() error {
		callCount++
		if callCount < 2 {
			return badGatewayErr
		}
		return nil
	}, 3, []time.Duration{1 * time.Millisecond})
	if err != nil {
		t.Errorf("Expected nil error after retry, got: %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}
}

func TestWithRetryDelays_503TriggersRetry(t *testing.T) {
	callCount := 0
	unavailErr := &httpStatusError{code: 503, msg: "Service Unavailable"}
	err := WithRetryDelays(func() error {
		callCount++
		if callCount < 2 {
			return unavailErr
		}
		return nil
	}, 3, []time.Duration{1 * time.Millisecond})
	if err != nil {
		t.Errorf("Expected nil error after retry, got: %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}
}

func TestWithRetryDelays_500DoesNotRetry(t *testing.T) {
	// 500 is not classified as transient — likely a real server-side bug.
	callCount := 0
	internalErr := &httpStatusError{code: 500, msg: "Internal Server Error"}
	err := WithRetryDelays(func() error {
		callCount++
		return internalErr
	}, 3, []time.Duration{1 * time.Millisecond})
	if err != internalErr {
		t.Errorf("Expected original 500 error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call (no retry for 500), got %d", callCount)
	}
}

func TestApplyJitter_PositiveBiasRange(t *testing.T) {
	// Positive-only jitter: result lies in [base, base*1.25]. Positive bias
	// preserves the "delay at least the configured base" guarantee that the
	// existing TestWithRetryDelays_UsesLastDelayWhenAttemptsExceedDelayLength
	// test relies on, while still spreading retries to avoid thundering herd.
	base := 100 * time.Millisecond
	high := time.Duration(float64(base) * 1.25)
	sawAbove := false
	for i := 0; i < 200; i++ {
		got := applyJitter(base)
		if got < base || got > high {
			t.Fatalf("applyJitter(%v) = %v, outside [%v, %v]", base, got, base, high)
		}
		if got > base {
			sawAbove = true
		}
	}
	if !sawAbove {
		t.Errorf("Expected at least one jittered value strictly above base over 200 iterations; got all == base")
	}
}

func TestApplyJitter_ZeroBaseReturnsZero(t *testing.T) {
	if got := applyJitter(0); got != 0 {
		t.Errorf("applyJitter(0) = %v, want 0", got)
	}
}

// httpStatusError simulates an HTTP error with a status code for testing.
type httpStatusError struct {
	code int
	msg  string
}

func (e *httpStatusError) Error() string {
	return e.msg
}

func (e *httpStatusError) HTTPStatusCode() int {
	return e.code
}

// httpStatusRetryAfterError adds Retry-After header support for testing.
type httpStatusRetryAfterError struct {
	code       int
	msg        string
	retryAfter string
}

func (e *httpStatusRetryAfterError) Error() string {
	return e.msg
}

func (e *httpStatusRetryAfterError) HTTPStatusCode() int {
	return e.code
}

func (e *httpStatusRetryAfterError) RetryAfterSeconds() string {
	return e.retryAfter
}
