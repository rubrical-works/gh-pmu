package api

import (
	"bytes"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// fakeRoundTripper returns canned responses in order. After the slice is
// exhausted it returns a sentinel error to make over-call faults visible.
type fakeRoundTripper struct {
	calls     atomic.Int32
	responses []*http.Response
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := int(f.calls.Add(1)) - 1
	if idx >= len(f.responses) {
		return nil, io.EOF
	}
	return f.responses[idx], nil
}

func makeResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

// TestRetryIntegration_RawGraphQL504ThenSuccess verifies the end-to-end retry
// behaviour through the raw GraphQL path: a 504 response on the first attempt
// triggers exactly one retry, and the second 200 response is returned to the
// caller without surfacing the 504.
func TestRetryIntegration_RawGraphQL504ThenSuccess(t *testing.T) {
	transport := &fakeRoundTripper{
		responses: []*http.Response{
			makeResp(504, `{"message":"Gateway Timeout"}`),
			makeResp(200, `{"data":{"ok":true}}`),
		},
	}
	SetTestTransport(transport)
	defer SetTestTransport(nil)

	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Override the raw decorator's delays so the test runs in milliseconds, not seconds.
	// Construction wraps with DefaultRetryDelays; for testability we re-wrap here.
	rawInner := c.rawGQL.(*retryingRawGraphQL).inner
	c.rawGQL = newRetryingRawGraphQL(rawInner, 3, []time.Duration{1 * time.Millisecond})

	data, err := c.rawGQL.DoRaw("{ ok }", nil)
	if err != nil {
		t.Fatalf("Expected nil error after retry recovers from 504, got: %v", err)
	}
	if string(data) != `{"data":{"ok":true}}` {
		t.Errorf("Expected success payload, got %q", string(data))
	}
	if got := transport.calls.Load(); got != 2 {
		t.Errorf("Expected exactly 2 transport calls (1 504 + 1 200), got %d", got)
	}
}

// TestRetryIntegration_RawGraphQLNonRetryable surfaces the 404 immediately
// without consuming additional retries.
func TestRetryIntegration_RawGraphQLNonRetryable(t *testing.T) {
	transport := &fakeRoundTripper{
		responses: []*http.Response{
			makeResp(404, `{"message":"Not Found"}`),
		},
	}
	SetTestTransport(transport)
	defer SetTestTransport(nil)

	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	rawInner := c.rawGQL.(*retryingRawGraphQL).inner
	c.rawGQL = newRetryingRawGraphQL(rawInner, 3, []time.Duration{1 * time.Millisecond})

	_, err = c.rawGQL.DoRaw("{ q }", nil)
	if err == nil {
		t.Fatal("Expected error from 404, got nil")
	}
	if got := transport.calls.Load(); got != 1 {
		t.Errorf("Expected 1 transport call (no retry on 404), got %d", got)
	}
}
