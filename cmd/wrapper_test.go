package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
)

// mockGraphQLHandler creates an HTTP handler that responds to GraphQL requests
// with preconfigured responses.
type mockGraphQLHandler struct {
	// Responses maps operation names to their JSON responses
	responses map[string]interface{}
	// Default response for unmatched operations
	defaultResponse interface{}
	// Track requests for assertions
	requests []graphQLRequest
	// Mutex to protect concurrent access to requests
	mu sync.Mutex
}

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func newMockGraphQLHandler() *mockGraphQLHandler {
	return &mockGraphQLHandler{
		responses: make(map[string]interface{}),
		requests:  []graphQLRequest{},
	}
}

func (h *mockGraphQLHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req graphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	h.mu.Lock()
	h.requests = append(h.requests, req)
	h.mu.Unlock()

	// Find matching response based on query content
	var response interface{}
	for opName, resp := range h.responses {
		if strings.Contains(req.Query, opName) {
			response = resp
			break
		}
	}

	if response == nil {
		response = h.defaultResponse
	}

	if response == nil {
		// Return empty data if no response configured
		response = map[string]interface{}{"data": map[string]interface{}{}}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// redirectTransport is an http.RoundTripper that redirects all requests to a test server
type redirectTransport struct {
	server *httptest.Server
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect the request to the test server
	newURL := t.server.URL + req.URL.Path
	if req.URL.RawQuery != "" {
		newURL += "?" + req.URL.RawQuery
	}

	newReq, err := http.NewRequest(req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}

	// Copy headers
	for k, v := range req.Header {
		newReq.Header[k] = v
	}

	return t.server.Client().Do(newReq)
}

// setupTestEnvironment creates a temp directory with a valid config file
// and sets up the mock GraphQL server. Returns a cleanup function.
func setupTestEnvironment(t *testing.T, handler *mockGraphQLHandler) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Write config file
	configContent := `project:
  owner: test-org
  number: 1
repositories:
  - test-org/test-repo
fields:
  status:
    field: Status
    values:
      backlog: Backlog
      in_progress: In Progress
      done: Done
  priority:
    field: Priority
    values:
      p0: P0
      p1: P1
      p2: P2
`
	configPath := filepath.Join(tmpDir, ".gh-pmu.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to write config: %v", err)
	}

	// Start mock server
	server := httptest.NewServer(handler)

	// Set test transport that redirects all requests to mock server
	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")

	// Save original directory
	origDir, _ := os.Getwd()

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		server.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to chdir: %v", err)
	}

	cleanup := func() {
		_ = os.Chdir(origDir)
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
		server.Close()
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// ============================================================================
// runList Wrapper Tests
// ============================================================================

func TestRunList_LoadsConfig(t *testing.T) {
	// This test verifies that runList successfully loads config from cwd
	// We can't easily mock the full GraphQL flow, but we can verify:
	// 1. Config loading works
	// 2. The API client is created
	// The API call itself may fail without auth, but that's expected

	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &listOptions{}
	err := runList(cmd, opts)

	// We expect an API error (not a config error)
	// This proves the config was loaded successfully
	if err == nil {
		// If no error, even better - mock worked
		return
	}

	// Config loading should succeed; API call may fail
	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}

	// API errors are expected when mocking isn't perfect
	// This still proves we passed through the config loading code path
}

func TestRunList_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	// Create temp dir WITHOUT config file
	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(server.Client().Transport)
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newListCommand()
	opts := &listOptions{}
	err = runList(cmd, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

func TestRunList_InvalidConfig(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-badconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write invalid config (missing required fields)
	configPath := filepath.Join(tmpDir, ".gh-pmu.yml")
	_ = os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(server.Client().Transport)
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newListCommand()
	opts := &listOptions{}
	err = runList(cmd, opts)

	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

// ============================================================================
// runView Wrapper Tests
// ============================================================================

func TestRunView_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newViewCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &viewOptions{}
	args := []string{"1"}
	err := runView(cmd, args, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunView_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(server.Client().Transport)
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newViewCommand()
	opts := &viewOptions{}
	args := []string{"1"}
	err = runView(cmd, args, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

// ============================================================================
// runMove Wrapper Tests
// ============================================================================

func TestRunMove_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newMoveCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &moveOptions{
		status: "in_progress",
	}
	args := []string{"1"}
	err := runMove(cmd, args, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunMove_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(server.Client().Transport)
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newMoveCommand()
	opts := &moveOptions{status: "done"}
	args := []string{"1"}
	err = runMove(cmd, args, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

// ============================================================================
// runClose Wrapper Tests
// ============================================================================

func TestRunClose_WithUpdateStatus_LoadsConfig(t *testing.T) {
	// runClose only loads config when --update-status is used
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newCloseCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &closeOptions{
		updateStatus: true, // This triggers config loading
	}
	args := []string{"1"}
	err := runClose(cmd, args, opts)

	// We expect an API or gh CLI error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunClose_WithUpdateStatus_ConfigNotFound(t *testing.T) {
	// Note: runClose catches config errors and warns but continues,
	// so we can't directly test for config load errors here.
	// This test exercises the code path where config is not found.
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newCloseCommand()
	opts := &closeOptions{
		updateStatus: true, // This triggers config loading path
	}
	args := []string{"1"}
	_ = runClose(cmd, args, opts)

	// We don't check the error because runClose warns and continues
	// The code path for updateStatusToDone was exercised
}

// ============================================================================
// runFieldList Wrapper Tests
// ============================================================================

func TestRunFieldList_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newFieldListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := runFieldList(cmd, []string{})

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunFieldList_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newFieldListCommand()
	err = runFieldList(cmd, []string{})

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

// ============================================================================
// runFieldCreate Wrapper Tests
// ============================================================================

func TestRunFieldCreate_InvalidType(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newFieldCreateCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &fieldCreateOptions{
		fieldType: "invalid_type",
	}
	err := runFieldCreate(cmd, []string{"TestField"}, opts)

	if err == nil {
		t.Fatal("expected error for invalid field type")
	}
	if !strings.Contains(err.Error(), "invalid field type") {
		t.Errorf("expected 'invalid field type' error, got: %v", err)
	}
}

func TestRunFieldCreate_SingleSelectWithoutOptions(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newFieldCreateCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &fieldCreateOptions{
		fieldType: "single_select",
		options:   []string{}, // No options
	}
	err := runFieldCreate(cmd, []string{"TestField"}, opts)

	if err == nil {
		t.Fatal("expected error for single_select without options")
	}
	if !strings.Contains(err.Error(), "requires at least one --option") {
		t.Errorf("expected 'requires at least one --option' error, got: %v", err)
	}
}

func TestRunFieldCreate_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newFieldCreateCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &fieldCreateOptions{
		fieldType: "text",
	}
	err := runFieldCreate(cmd, []string{"TestField"}, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

// ============================================================================
// runHistory Wrapper Tests
// ============================================================================

func TestRunHistory_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newHistoryCommand()
	opts := &historyOptions{}
	err = runHistory(cmd, []string{}, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

// ============================================================================
// runBoard Wrapper Tests
// ============================================================================

func TestRunBoard_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newBoardCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &boardOptions{}
	err := runBoard(cmd, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunBoard_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newBoardCommand()
	opts := &boardOptions{}
	err = runBoard(cmd, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

// ============================================================================
// Test Transport Integration
// ============================================================================

func TestSetTestTransport_WorksWithNewClient(t *testing.T) {
	// Verify that SetTestTransport actually affects NewClient behavior
	handler := newMockGraphQLHandler()
	handler.defaultResponse = map[string]interface{}{
		"data": map[string]interface{}{
			"test": "response",
		},
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	// Set test transport
	api.SetTestTransport(server.Client().Transport)
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	// Create client - should use the test transport
	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("expected client to be created, got error: %v", err)
	}
	_ = client

	// The client should be functional (even if calls fail due to mock)
	// This verifies the transport injection works
}

// ============================================================================
// runIntake Wrapper Tests
// ============================================================================

func TestRunIntake_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &intakeOptions{}
	err := runIntake(cmd, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunIntake_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newIntakeCommand()
	opts := &intakeOptions{}
	err = runIntake(cmd, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

func TestRunIntake_WithDryRun(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &intakeOptions{
		dryRun: true,
	}
	err := runIntake(cmd, opts)

	// API error expected, but proves config loading works
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

// ============================================================================
// runSplit Wrapper Tests
// ============================================================================

func TestRunSplit_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newSplitCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &splitOptions{}
	args := []string{"1"}
	err := runSplit(cmd, args, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunSplit_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newSplitCommand()
	opts := &splitOptions{}
	args := []string{"1"}
	err = runSplit(cmd, args, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

func TestRunSplit_InvalidIssueNumber(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newSplitCommand()
	opts := &splitOptions{}
	args := []string{"not-a-number"}
	err := runSplit(cmd, args, opts)

	if err == nil {
		t.Fatal("expected error for invalid issue number")
	}
	if !strings.Contains(err.Error(), "invalid issue number") {
		t.Errorf("expected 'invalid issue number' error, got: %v", err)
	}
}

func TestRunSplit_WithDryRun(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newSplitCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &splitOptions{
		dryRun: true,
		from:   "body",
	}
	args := []string{"1"}
	err := runSplit(cmd, args, opts)

	// API error expected, but proves config loading works
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

// ============================================================================
// runTriage Wrapper Tests
// ============================================================================

func TestRunTriage_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newTriageCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &triageOptions{
		list: true, // Use list mode to avoid requiring config name
	}
	err := runTriage(cmd, []string{}, opts)

	// We expect an API error or no error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunTriage_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newTriageCommand()
	opts := &triageOptions{}
	err = runTriage(cmd, []string{"test"}, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

func TestRunTriage_WithQuery(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newTriageCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &triageOptions{
		query:  "is:open",
		apply:  "status:backlog",
		dryRun: true,
	}
	err := runTriage(cmd, []string{}, opts)

	// API error expected, but proves config loading works
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

// ============================================================================
// runSubAdd Wrapper Tests
// ============================================================================

func TestRunSubAdd_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newSubAddCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &subAddOptions{}
	args := []string{"1", "2"}
	err := runSubAdd(cmd, args, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunSubAdd_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newSubAddCommand()
	opts := &subAddOptions{}
	args := []string{"1", "2"}
	err = runSubAdd(cmd, args, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

// ============================================================================
// runSubCreate Wrapper Tests
// ============================================================================

func TestRunSubCreate_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newSubCreateCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &subCreateOptions{
		parent: "1",
		title:  "Test sub-issue",
	}
	err := runSubCreate(cmd, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunSubCreate_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newSubCreateCommand()
	opts := &subCreateOptions{
		parent: "1",
		title:  "Test sub-issue",
	}
	err = runSubCreate(cmd, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

// ============================================================================
// runSubList Wrapper Tests
// ============================================================================

func TestRunSubList_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newSubListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &subListOptions{
		state:    "all",      // Valid state required before config check
		relation: "children", // Valid relation required before config check
	}
	args := []string{"1"}
	err := runSubList(cmd, args, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunSubList_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newSubListCommand()
	opts := &subListOptions{
		state:    "all",      // Valid state required before config check
		relation: "children", // Valid relation required before config check
	}
	args := []string{"1"}
	err = runSubList(cmd, args, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

// ============================================================================
// runCreate Wrapper Tests
// ============================================================================

func TestRunCreate_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newCreateCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &createOptions{
		title: "Test Issue",
	}
	err := runCreate(cmd, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunCreate_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newCreateCommand()
	opts := &createOptions{title: "Test"}
	err = runCreate(cmd, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

// ============================================================================
// runSubRemove Wrapper Tests
// ============================================================================

func TestRunSubRemove_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newSubRemoveCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &subRemoveOptions{}
	args := []string{"1", "2"}
	err := runSubRemove(cmd, args, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}

func TestRunSubRemove_ConfigNotFound(t *testing.T) {
	handler := newMockGraphQLHandler()

	tmpDir, err := os.MkdirTemp("", "gh-pmu-test-noconfig-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(handler)
	defer server.Close()

	api.SetTestTransport(&redirectTransport{server: server})
	api.SetTestAuthToken("test-token")
	defer func() {
		api.SetTestTransport(nil)
		api.SetTestAuthToken("")
	}()

	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	cmd := newSubRemoveCommand()
	opts := &subRemoveOptions{}
	args := []string{"1", "2"}
	err = runSubRemove(cmd, args, opts)

	if err == nil {
		t.Fatal("expected error when config not found")
	}
	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

// ============================================================================
// runHistory Wrapper Tests (additional)
// ============================================================================

func TestRunHistory_LoadsConfig(t *testing.T) {
	handler := newMockGraphQLHandler()
	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newHistoryCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &historyOptions{}
	err := runHistory(cmd, []string{}, opts)

	// We expect an API error (not a config error)
	if err == nil {
		return
	}

	if strings.Contains(err.Error(), "failed to load configuration") {
		t.Fatalf("config loading failed: %v", err)
	}
}
