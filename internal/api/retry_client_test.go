package api

import (
	"testing"
	"time"
)

// fakeGQL records call counts and returns a sequence of errors.
type fakeGQL struct {
	mutateCalls int
	queryCalls  int
	mutateErrs  []error
	queryErrs   []error
}

func (f *fakeGQL) Mutate(name string, mutation interface{}, vars map[string]interface{}) error {
	defer func() { f.mutateCalls++ }()
	if f.mutateCalls < len(f.mutateErrs) {
		return f.mutateErrs[f.mutateCalls]
	}
	return nil
}

func (f *fakeGQL) Query(name string, query interface{}, vars map[string]interface{}) error {
	defer func() { f.queryCalls++ }()
	if f.queryCalls < len(f.queryErrs) {
		return f.queryErrs[f.queryCalls]
	}
	return nil
}

func TestRetryingGraphQL_MutateRetriesOn504(t *testing.T) {
	gw := &httpStatusError{code: 504, msg: "Gateway Timeout"}
	inner := &fakeGQL{mutateErrs: []error{gw, gw}}
	dec := newRetryingGraphQL(inner, 3, []time.Duration{1 * time.Millisecond})

	err := dec.Mutate("op", nil, nil)
	if err != nil {
		t.Fatalf("Expected nil after 2 retries + success, got: %v", err)
	}
	if inner.mutateCalls != 3 {
		t.Errorf("Expected 3 Mutate calls (2 504 + 1 success), got %d", inner.mutateCalls)
	}
}

func TestRetryingGraphQL_QueryRetriesOn503(t *testing.T) {
	un := &httpStatusError{code: 503, msg: "Service Unavailable"}
	inner := &fakeGQL{queryErrs: []error{un}}
	dec := newRetryingGraphQL(inner, 3, []time.Duration{1 * time.Millisecond})

	err := dec.Query("op", nil, nil)
	if err != nil {
		t.Fatalf("Expected nil after 1 retry + success, got: %v", err)
	}
	if inner.queryCalls != 2 {
		t.Errorf("Expected 2 Query calls, got %d", inner.queryCalls)
	}
}

func TestRetryingGraphQL_NonRetryableErrorReturnsImmediately(t *testing.T) {
	notFound := &httpStatusError{code: 404, msg: "Not Found"}
	inner := &fakeGQL{mutateErrs: []error{notFound}}
	dec := newRetryingGraphQL(inner, 3, []time.Duration{1 * time.Millisecond})

	err := dec.Mutate("op", nil, nil)
	if err != notFound {
		t.Errorf("Expected original 404 error, got: %v", err)
	}
	if inner.mutateCalls != 1 {
		t.Errorf("Expected 1 Mutate call (no retry), got %d", inner.mutateCalls)
	}
}

// fakeRaw records call counts and returns a sequence of error/payload pairs.
type fakeRaw struct {
	calls int
	errs  []error
	body  []byte
}

func (f *fakeRaw) DoRaw(query string, headers map[string]string) ([]byte, error) {
	defer func() { f.calls++ }()
	if f.calls < len(f.errs) {
		return nil, f.errs[f.calls]
	}
	return f.body, nil
}

func (f *fakeRaw) DoRawBody(body []byte, headers map[string]string) ([]byte, error) {
	return f.DoRaw("", headers)
}

func TestRetryingRawGraphQL_DoRawRetriesOn502(t *testing.T) {
	bg := &httpStatusError{code: 502, msg: "Bad Gateway"}
	inner := &fakeRaw{errs: []error{bg}, body: []byte("ok")}
	dec := newRetryingRawGraphQL(inner, 3, []time.Duration{1 * time.Millisecond})

	data, err := dec.DoRaw("query", nil)
	if err != nil {
		t.Fatalf("Expected nil error after retry, got: %v", err)
	}
	if string(data) != "ok" {
		t.Errorf("Expected payload 'ok', got %q", string(data))
	}
	if inner.calls != 2 {
		t.Errorf("Expected 2 DoRaw calls, got %d", inner.calls)
	}
}

func TestRetryingRawGraphQL_DoRawBodyRetriesOn504(t *testing.T) {
	gw := &httpStatusError{code: 504, msg: "Gateway Timeout"}
	inner := &fakeRaw{errs: []error{gw}, body: []byte("payload")}
	dec := newRetryingRawGraphQL(inner, 3, []time.Duration{1 * time.Millisecond})

	data, err := dec.DoRawBody([]byte("body"), nil)
	if err != nil {
		t.Fatalf("Expected nil error after retry, got: %v", err)
	}
	if string(data) != "payload" {
		t.Errorf("Expected 'payload', got %q", string(data))
	}
	if inner.calls != 2 {
		t.Errorf("Expected 2 calls, got %d", inner.calls)
	}
}

func TestNewClientWithOptions_GraphQLClientIsRetrying(t *testing.T) {
	// Provide a stub auth token so go-gh's NewGraphQLClient doesn't fail on
	// auth-discovery in CI environments without `gh auth status`.
	c, err := NewClientWithOptions(ClientOptions{AuthToken: "test-token"})
	if err != nil {
		t.Fatalf("NewClientWithOptions failed: %v", err)
	}
	if _, ok := c.gql.(*retryingGraphQL); !ok {
		t.Errorf("Expected c.gql to be *retryingGraphQL, got %T", c.gql)
	}
	if _, ok := c.rawGQL.(*retryingRawGraphQL); !ok {
		t.Errorf("Expected c.rawGQL to be *retryingRawGraphQL, got %T", c.rawGQL)
	}
}

// ============================================================================
// Non-idempotent mutation retry safety (#858)
// ============================================================================
//
// A 502/504 on a create-style mutation is ambiguous: the write may have applied
// server-side before the response was lost, so a transparent retry can create a
// duplicate issue/comment/label/field/project. Create-style mutations must
// therefore retry only on rate limits (rejected before processing), never on
// 5xx. Idempotent mutations and all queries keep full 5xx retry.

func TestRetryingGraphQL_NonIdempotentMutationSkips5xxRetry(t *testing.T) {
	gw := &httpStatusError{code: 504, msg: "Gateway Timeout"}
	inner := &fakeGQL{mutateErrs: []error{gw, gw, gw, gw}}
	dec := newRetryingGraphQL(inner, 3, []time.Duration{1 * time.Millisecond})

	err := dec.Mutate("CreateIssue", nil, nil)
	if err != gw {
		t.Errorf("expected the 504 to surface without retry, got %v", err)
	}
	if inner.mutateCalls != 1 {
		t.Errorf("expected exactly 1 call (no 5xx retry for a create), got %d", inner.mutateCalls)
	}
}

func TestRetryingGraphQL_AllNonIdempotentMutationsSkip5xxRetry(t *testing.T) {
	for _, name := range []string{"CreateIssue", "AddComment", "CreateLabel", "CreateProjectV2Field", "CopyProjectV2"} {
		t.Run(name, func(t *testing.T) {
			bg := &httpStatusError{code: 502, msg: "Bad Gateway"}
			inner := &fakeGQL{mutateErrs: []error{bg, bg, bg, bg}}
			dec := newRetryingGraphQL(inner, 3, []time.Duration{1 * time.Millisecond})

			err := dec.Mutate(name, nil, nil)
			if err == nil {
				t.Fatalf("%s: expected the 5xx error to surface", name)
			}
			if inner.mutateCalls != 1 {
				t.Errorf("%s: expected 1 call (no 5xx retry), got %d", name, inner.mutateCalls)
			}
		})
	}
}

func TestRetryingGraphQL_NonIdempotentMutationRetriesRateLimit(t *testing.T) {
	rl := &httpStatusError{code: 429, msg: "Too Many Requests"}
	inner := &fakeGQL{mutateErrs: []error{rl}} // then success
	dec := newRetryingGraphQL(inner, 3, []time.Duration{1 * time.Millisecond})

	err := dec.Mutate("CreateIssue", nil, nil)
	if err != nil {
		t.Fatalf("expected success after rate-limit retry, got %v", err)
	}
	if inner.mutateCalls != 2 {
		t.Errorf("expected 2 calls (429 is safe to retry for creates), got %d", inner.mutateCalls)
	}
}

func TestRetryingGraphQL_IdempotentMutationStillRetries5xx(t *testing.T) {
	bg := &httpStatusError{code: 502, msg: "Bad Gateway"}
	inner := &fakeGQL{mutateErrs: []error{bg}} // then success
	dec := newRetryingGraphQL(inner, 3, []time.Duration{1 * time.Millisecond})

	err := dec.Mutate("UpdateProjectV2ItemFieldValue", nil, nil)
	if err != nil {
		t.Fatalf("expected idempotent mutation to retry 5xx and succeed, got %v", err)
	}
	if inner.mutateCalls != 2 {
		t.Errorf("expected 2 calls (idempotent 5xx retry unchanged), got %d", inner.mutateCalls)
	}
}
