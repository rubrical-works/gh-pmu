package api

import "time"

// defaultClientMaxRetries is the retry budget applied to every API call routed
// through a retrying decorator. Three attempts after the initial call.
const defaultClientMaxRetries = 3

// retryingGraphQL decorates a GraphQLClient so every Mutate / Query call is
// transparently retried on retryable errors (rate limits + transient 5xx).
// All gh-pmu commands inherit this without per-call-site wrapping.
type retryingGraphQL struct {
	inner      GraphQLClient
	maxRetries int
	delays     []time.Duration
}

// newRetryingGraphQL wraps inner. delays may be nil to use DefaultRetryDelays.
func newRetryingGraphQL(inner GraphQLClient, maxRetries int, delays []time.Duration) *retryingGraphQL {
	if delays == nil {
		delays = DefaultRetryDelays
	}
	return &retryingGraphQL{inner: inner, maxRetries: maxRetries, delays: delays}
}

func (r *retryingGraphQL) Mutate(name string, mutation interface{}, vars map[string]interface{}) error {
	return WithRetryDelays(func() error {
		return r.inner.Mutate(name, mutation, vars)
	}, r.maxRetries, r.delays)
}

func (r *retryingGraphQL) Query(name string, query interface{}, vars map[string]interface{}) error {
	return WithRetryDelays(func() error {
		return r.inner.Query(name, query, vars)
	}, r.maxRetries, r.delays)
}

// retryingRawGraphQL is the analog for raw / typed-batch GraphQL calls.
type retryingRawGraphQL struct {
	inner      RawGraphQLDoer
	maxRetries int
	delays     []time.Duration
}

func newRetryingRawGraphQL(inner RawGraphQLDoer, maxRetries int, delays []time.Duration) *retryingRawGraphQL {
	if delays == nil {
		delays = DefaultRetryDelays
	}
	return &retryingRawGraphQL{inner: inner, maxRetries: maxRetries, delays: delays}
}

func (r *retryingRawGraphQL) DoRaw(query string, headers map[string]string) ([]byte, error) {
	var data []byte
	err := WithRetryDelays(func() error {
		var inner error
		data, inner = r.inner.DoRaw(query, headers)
		return inner
	}, r.maxRetries, r.delays)
	return data, err
}

func (r *retryingRawGraphQL) DoRawBody(body []byte, headers map[string]string) ([]byte, error) {
	var data []byte
	err := WithRetryDelays(func() error {
		var inner error
		data, inner = r.inner.DoRawBody(body, headers)
		return inner
	}, r.maxRetries, r.delays)
	return data, err
}
