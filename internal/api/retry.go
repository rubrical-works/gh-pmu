package api

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

// DefaultRetryDelays defines the exponential backoff delays for retry attempts
var DefaultRetryDelays = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
}

// WithRetry executes a function with automatic retry on retryable errors.
// Retryable errors are rate limits (429, rate-limited 403) and transient
// 5xx responses (502/503/504); see IsRetryable. Uses exponential backoff
// with positive jitter to avoid thundering herd on coordinated retries.
// Returns the function's error if not retryable, or after max retries exceeded.
func WithRetry(fn func() error, maxRetries int) error {
	return WithRetryDelays(fn, maxRetries, DefaultRetryDelays)
}

// WithRetryDelays executes a function with automatic retry using custom delays.
// This variant is primarily for testing to allow faster tests.
// If the error carries a Retry-After header, that value overrides the default
// delay (and is not jittered — server-mandated delay is honored exactly).
func WithRetryDelays(fn func() error, maxRetries int, delays []time.Duration) error {
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = fn()
		if err == nil || !IsRetryable(err) {
			return err
		}
		if attempt < maxRetries {
			delay := delays[min(attempt, len(delays)-1)]
			// Retry-After header overrides default delay (no jitter on server-mandated delay)
			if retryAfter := GetRetryAfter(err); retryAfter > 0 {
				delay = retryAfter
			} else {
				delay = applyJitter(delay)
			}
			reason := "transient error"
			if IsRateLimited(err) {
				reason = "rate limited"
			}
			fmt.Fprintf(os.Stderr, "Warning: %s, retrying in %v...\n", reason, delay)
			time.Sleep(delay)
		}
	}
	return err
}

// applyJitter returns a duration in [d, d*1.25]. The positive-only bias
// preserves the configured base as a lower bound (callers never wait less
// than they asked for) while spreading retries enough to avoid thundering
// herd in coordinated-client scenarios. Returns 0 unchanged.
func applyJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	return d + time.Duration(float64(d)*0.25*rand.Float64())
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
