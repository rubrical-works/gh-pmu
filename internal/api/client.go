package api

import (
	"fmt"
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

// Client wraps the GitHub GraphQL API client with project management features
type Client struct {
	gql  GraphQLClient
	opts ClientOptions
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

	return &Client{
		gql:  gql,
		opts: opts,
	}, nil
}

// NewClientWithGraphQL creates a Client with a custom GraphQL client (for testing)
func NewClientWithGraphQL(gql GraphQLClient) *Client {
	return &Client{gql: gql}
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
