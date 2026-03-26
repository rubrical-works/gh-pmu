package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/cli/go-gh/v2/pkg/api"
)

// FeatureSubIssues is the GitHub API preview header for sub-issues
const FeatureSubIssues = "sub_issues"

// FeatureIssueTypes is the GitHub API preview header for issue types
const FeatureIssueTypes = "issue_types"

// testMu guards testTransport and testAuthToken against concurrent access.
var testMu sync.Mutex

// testTransport is a package-level transport override for testing.
// When set, NewClient() will use this transport instead of http.DefaultTransport.
// This allows integration tests to mock the HTTP layer without modifying production code.
var testTransport http.RoundTripper

// testAuthToken is a package-level auth token override for testing.
var testAuthToken string

// SetTestTransport sets a custom transport for testing purposes.
// Call with nil to clear the test transport.
func SetTestTransport(t http.RoundTripper) {
	testMu.Lock()
	defer testMu.Unlock()
	testTransport = t
}

// SetTestAuthToken sets a custom auth token for testing purposes.
// Call with empty string to clear the test token.
func SetTestAuthToken(token string) {
	testMu.Lock()
	defer testMu.Unlock()
	testAuthToken = token
}

// GraphQLClient interface allows mocking the GitHub GraphQL client for testing
type GraphQLClient interface {
	Query(name string, query interface{}, variables map[string]interface{}) error
	Mutate(name string, mutation interface{}, variables map[string]interface{}) error
}

// RawGraphQLDoer executes raw GraphQL queries and returns the full response bytes.
// This is used for dynamic/batch queries with aliased fields that don't fit typed clients.
type RawGraphQLDoer interface {
	DoRaw(query string, headers map[string]string) ([]byte, error)
	DoRawBody(body []byte, headers map[string]string) ([]byte, error)
}

// Client wraps the GitHub GraphQL API client with project management features
type Client struct {
	gql    GraphQLClient
	rawGQL RawGraphQLDoer
	opts   ClientOptions
}

// ClientOptions configures the API client
type ClientOptions struct {
	// Host is the GitHub hostname (default: github.com)
	Host string

	// EnableSubIssues enables the sub_issues feature preview
	EnableSubIssues bool

	// EnableIssueTypes enables the issue_types feature preview
	EnableIssueTypes bool

	// Transport specifies the HTTP transport for API requests (for testing)
	Transport http.RoundTripper

	// AuthToken is the authorization token (for testing)
	AuthToken string
}

// NewClient creates a new API client with default options
func NewClient() (*Client, error) {
	opts := ClientOptions{
		EnableSubIssues:  true,
		EnableIssueTypes: true,
	}
	// Apply test overrides if set
	testMu.Lock()
	if testTransport != nil {
		opts.Transport = testTransport
	}
	if testAuthToken != "" {
		opts.AuthToken = testAuthToken
	}
	testMu.Unlock()
	return NewClientWithOptions(opts)
}

// NewClientWithOptions creates a new API client with custom options
func NewClientWithOptions(opts ClientOptions) (*Client, error) {
	// Build headers with feature previews
	headers := make(map[string]string)

	// Add GraphQL feature preview headers
	// These enable beta features in the GitHub API
	featureHeaders := []string{}
	if opts.EnableSubIssues {
		featureHeaders = append(featureHeaders, FeatureSubIssues)
	}
	if opts.EnableIssueTypes {
		featureHeaders = append(featureHeaders, FeatureIssueTypes)
	}

	if len(featureHeaders) > 0 {
		// GitHub uses X-Github-Next for feature previews
		headers["X-Github-Next"] = joinFeatures(featureHeaders)
	}

	// Create GraphQL client options
	apiOpts := api.ClientOptions{
		Headers: headers,
	}

	if opts.Host != "" {
		apiOpts.Host = opts.Host
	}
	if opts.Transport != nil {
		apiOpts.Transport = opts.Transport
	}
	if opts.AuthToken != "" {
		apiOpts.AuthToken = opts.AuthToken
	}

	// Create the GraphQL client
	gql, err := api.NewGraphQLClient(apiOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Create raw HTTP client for dynamic/batch GraphQL queries
	httpClient, err := api.NewHTTPClient(apiOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Client{
		gql:    gql,
		rawGQL: &httpRawGraphQL{httpClient: httpClient, host: opts.Host},
		opts:   opts,
	}, nil
}

// NewClientWithGraphQL creates a Client with a custom GraphQL client (for testing)
func NewClientWithGraphQL(gql GraphQLClient) *Client {
	return &Client{gql: gql}
}

// httpRawGraphQL implements RawGraphQLDoer using go-gh's HTTP client.
type httpRawGraphQL struct {
	httpClient *http.Client
	host       string
}

// DoRaw executes a raw GraphQL query and returns the full JSON response bytes.
// Extra headers (e.g. feature previews) are merged into the request.
func (h *httpRawGraphQL) DoRaw(query string, headers map[string]string) ([]byte, error) {
	body, err := json.Marshal(map[string]interface{}{"query": query})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}
	return h.DoRawBody(body, headers)
}

// DoRawBody sends a pre-built JSON request body to the GraphQL endpoint.
// Use this for mutations with variables where the body is already constructed.
func (h *httpRawGraphQL) DoRawBody(body []byte, headers map[string]string) ([]byte, error) {
	host := h.host
	if host == "" {
		host = "github.com"
	}
	url := fmt.Sprintf("https://api.%s/graphql", host)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GraphQL request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read GraphQL response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GraphQL request returned status %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// joinFeatures joins feature names with commas
func joinFeatures(features []string) string {
	if len(features) == 0 {
		return ""
	}
	result := features[0]
	for i := 1; i < len(features); i++ {
		result += "," + features[i]
	}
	return result
}

// GetLatestGitTag returns the latest git tag using git describe
func (c *Client) GetLatestGitTag() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no git tags found: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
