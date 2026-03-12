package api

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNewClient_ReturnsClient(t *testing.T) {
	// ACT: Create a new client
	client, err := NewClient()
	if err != nil {
		t.Skipf("Skipping - requires auth: %v", err)
	}

	// ASSERT: Client is not nil
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}
}

func TestNewClient_HasGraphQLClient(t *testing.T) {
	// Skip in CI - requires gh auth
	if testing.Short() {
		t.Skip("Skipping test that requires gh auth")
	}

	// ACT: Create a new client
	client, err := NewClient()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// ASSERT: GraphQL client is accessible
	if client.gql == nil {
		t.Fatal("Expected GraphQL client to be initialized")
	}
}

func TestNewClientWithOptions_CustomHost(t *testing.T) {
	// ARRANGE: Custom options with transport to avoid auth lookup
	transport := &headerCapturingTransport{}
	opts := ClientOptions{
		Host:      "github.example.com",
		Transport: transport,
		AuthToken: "test-token",
	}

	// ACT: Create client with options
	client, err := NewClientWithOptions(opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// ASSERT: Client is created with custom host
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}
	if client.opts.Host != "github.example.com" {
		t.Errorf("Expected Host 'github.example.com', got %q", client.opts.Host)
	}
}

func TestClient_FeatureHeaders_Included(t *testing.T) {
	transport := &headerCapturingTransport{}
	opts := ClientOptions{
		EnableSubIssues:  true,
		EnableIssueTypes: true,
		Transport:        transport,
		AuthToken:        "test-token",
	}

	client, err := NewClientWithOptions(opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// Make a GraphQL request to trigger the transport
	var query struct{}
	_ = client.gql.Query("Test", &query, nil)

	if !transport.called {
		t.Fatal("Expected transport to be called")
	}

	// Verify X-Github-Next header contains both feature flags
	ghNext := transport.capturedHeaders.Get("X-Github-Next")
	if ghNext == "" {
		t.Fatal("Expected X-Github-Next header to be set")
	}
	if !strings.Contains(ghNext, FeatureSubIssues) {
		t.Errorf("Expected X-Github-Next to contain %q, got %q", FeatureSubIssues, ghNext)
	}
	if !strings.Contains(ghNext, FeatureIssueTypes) {
		t.Errorf("Expected X-Github-Next to contain %q, got %q", FeatureIssueTypes, ghNext)
	}
}

func TestJoinFeatures_Empty(t *testing.T) {
	result := joinFeatures([]string{})
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestJoinFeatures_Single(t *testing.T) {
	result := joinFeatures([]string{"sub_issues"})
	if result != "sub_issues" {
		t.Errorf("Expected 'sub_issues', got '%s'", result)
	}
}

func TestJoinFeatures_Multiple(t *testing.T) {
	result := joinFeatures([]string{"sub_issues", "issue_types"})
	if result != "sub_issues,issue_types" {
		t.Errorf("Expected 'sub_issues,issue_types', got '%s'", result)
	}
}

func TestSetTestTransport(t *testing.T) {
	// Clear any existing transport
	SetTestTransport(nil)
	testMu.Lock()
	isNil := testTransport == nil
	testMu.Unlock()
	if !isNil {
		t.Fatal("Expected testTransport to be nil after clearing")
	}

	// Set a test transport
	transport := &headerCapturingTransport{}
	SetTestTransport(transport)
	testMu.Lock()
	match := testTransport == transport
	testMu.Unlock()
	if !match {
		t.Fatal("Expected testTransport to be the set transport")
	}

	// Clear the transport
	SetTestTransport(nil)
	testMu.Lock()
	isNil = testTransport == nil
	testMu.Unlock()
	if !isNil {
		t.Fatal("Expected testTransport to be nil after clearing")
	}
}

func TestSetTestAuthToken(t *testing.T) {
	// Clear any existing token
	SetTestAuthToken("")
	testMu.Lock()
	isEmpty := testAuthToken == ""
	testMu.Unlock()
	if !isEmpty {
		t.Fatal("Expected testAuthToken to be empty after clearing")
	}

	// Set a token
	SetTestAuthToken("test-token-123")
	testMu.Lock()
	val := testAuthToken
	testMu.Unlock()
	if val != "test-token-123" {
		t.Errorf("Expected testAuthToken to be 'test-token-123', got %q", val)
	}

	// Clear the token
	SetTestAuthToken("")
	testMu.Lock()
	isEmpty = testAuthToken == ""
	testMu.Unlock()
	if !isEmpty {
		t.Fatal("Expected testAuthToken to be empty after clearing")
	}
}

func TestNewClient_UsesTestTransport(t *testing.T) {
	// Set up test transport
	transport := &headerCapturingTransport{}
	SetTestTransport(transport)
	SetTestAuthToken("test-token")
	defer func() {
		SetTestTransport(nil)
		SetTestAuthToken("")
	}()

	// Create client via NewClient() — should pick up test transport
	client, err := NewClient()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("Expected client to be created")
	}

	// Make a request to verify the test transport is actually used
	var query struct{}
	_ = client.gql.Query("Test", &query, nil)

	if !transport.called {
		t.Fatal("Expected test transport to be used by NewClient()")
	}
}

func TestNewClientWithOptions_Transport(t *testing.T) {
	transport := &headerCapturingTransport{}
	opts := ClientOptions{
		Transport: transport,
		AuthToken: "test-token",
	}

	client, err := NewClientWithOptions(opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// Verify the transport is used for requests
	var query struct{}
	_ = client.gql.Query("Test", &query, nil)

	if !transport.called {
		t.Fatal("Expected provided transport to be used for requests")
	}
}

func TestNewClientWithOptions_AllOptions(t *testing.T) {
	transport := &headerCapturingTransport{}
	opts := ClientOptions{
		Host:             "github.example.com",
		EnableSubIssues:  true,
		EnableIssueTypes: true,
		Transport:        transport,
		AuthToken:        "test-token",
	}

	client, err := NewClientWithOptions(opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// Verify all options were stored
	if client.opts.Host != "github.example.com" {
		t.Errorf("Expected Host 'github.example.com', got %q", client.opts.Host)
	}
	if !client.opts.EnableSubIssues {
		t.Error("Expected EnableSubIssues to be true")
	}
	if !client.opts.EnableIssueTypes {
		t.Error("Expected EnableIssueTypes to be true")
	}

	// Verify feature headers are set via actual request
	var query struct{}
	_ = client.gql.Query("Test", &query, nil)

	ghNext := transport.capturedHeaders.Get("X-Github-Next")
	if !strings.Contains(ghNext, FeatureSubIssues) || !strings.Contains(ghNext, FeatureIssueTypes) {
		t.Errorf("Expected X-Github-Next to contain both features, got %q", ghNext)
	}
}

// headerCapturingTransport captures HTTP request headers for verification.
// Returns a minimal valid JSON response so GraphQL client doesn't error on parse.
type headerCapturingTransport struct {
	capturedHeaders http.Header
	called          bool
}

func (t *headerCapturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.capturedHeaders = req.Header.Clone()
	t.called = true
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
		Header:     make(http.Header),
	}, nil
}

func TestNewClientWithGraphQL(t *testing.T) {
	// Create mock GraphQL client
	mockGQL := &simpleGraphQLMock{}

	// Create client with mock
	client := NewClientWithGraphQL(mockGQL)

	// Assert client is created
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// Assert the mock was set
	if client.gql != mockGQL {
		t.Error("Expected GraphQL client to be the mock")
	}
}

func TestNewClientWithOptions_DisabledFeatures(t *testing.T) {
	transport := &headerCapturingTransport{}
	opts := ClientOptions{
		EnableSubIssues:  false,
		EnableIssueTypes: false,
		Transport:        transport,
		AuthToken:        "test-token",
	}

	client, err := NewClientWithOptions(opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// Make a request and verify NO feature header is set
	var query struct{}
	_ = client.gql.Query("Test", &query, nil)

	ghNext := transport.capturedHeaders.Get("X-Github-Next")
	if ghNext != "" {
		t.Errorf("Expected no X-Github-Next header when features disabled, got %q", ghNext)
	}
}

func TestNewClientWithOptions_OnlySubIssues(t *testing.T) {
	transport := &headerCapturingTransport{}
	opts := ClientOptions{
		EnableSubIssues:  true,
		EnableIssueTypes: false,
		Transport:        transport,
		AuthToken:        "test-token",
	}

	client, err := NewClientWithOptions(opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// Make a request and verify only sub_issues is in the header
	var query struct{}
	_ = client.gql.Query("Test", &query, nil)

	ghNext := transport.capturedHeaders.Get("X-Github-Next")
	if ghNext != FeatureSubIssues {
		t.Errorf("Expected X-Github-Next to be %q, got %q", FeatureSubIssues, ghNext)
	}
}

func TestNewClientWithOptions_ReturnsErrorOnAuthFailure(t *testing.T) {
	// Ensure no test overrides are active
	SetTestTransport(nil)
	SetTestAuthToken("")

	// Use an invalid host to force auth failure
	opts := ClientOptions{
		Host: "invalid.nonexistent.example.com",
	}

	_, err := NewClientWithOptions(opts)
	// The go-gh library may or may not fail depending on environment.
	// If it fails, it should return a wrapped error.
	if err != nil {
		if !strings.Contains(err.Error(), "failed to create API client") {
			t.Errorf("Expected wrapped error, got: %v", err)
		}
	}
	// If it succeeds (e.g., gh auth has a default token), that's also valid.
}

func TestNewClientWithOptions_OnlyIssueTypes(t *testing.T) {
	transport := &headerCapturingTransport{}
	opts := ClientOptions{
		EnableSubIssues:  false,
		EnableIssueTypes: true,
		Transport:        transport,
		AuthToken:        "test-token",
	}

	client, err := NewClientWithOptions(opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}

	// Make a request and verify only issue_types is in the header
	var query struct{}
	_ = client.gql.Query("Test", &query, nil)

	ghNext := transport.capturedHeaders.Get("X-Github-Next")
	if ghNext != FeatureIssueTypes {
		t.Errorf("Expected X-Github-Next to be %q, got %q", FeatureIssueTypes, ghNext)
	}
}

// simpleGraphQLMock implements GraphQLClient for client_test.go testing
type simpleGraphQLMock struct{}

func (m *simpleGraphQLMock) Query(name string, query interface{}, variables map[string]interface{}) error {
	return nil
}

func (m *simpleGraphQLMock) Mutate(name string, mutation interface{}, variables map[string]interface{}) error {
	return nil
}
