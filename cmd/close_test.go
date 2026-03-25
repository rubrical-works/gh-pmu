package cmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
)

// mockCloseClient implements closeClient for testing
type mockCloseClient struct {
	project *api.Project
	itemID  string

	// Error injection
	getProjectErr          error
	getProjectItemIDErr    error
	setProjectItemFieldErr error
	getIssueErr            error
	closeIssueErr          error
	addIssueCommentErr     error

	// Tracking
	issue            *api.Issue
	closedIssueID    string
	closedReason     string
	commentIssueID   string
	commentBody      string
}

func newMockCloseClient() *mockCloseClient {
	return &mockCloseClient{
		project: &api.Project{
			ID:    "proj-1",
			Title: "Test Project",
		},
		itemID: "item-123",
		issue: &api.Issue{
			ID:     "issue-node-123",
			Number: 42,
			Title:  "Test Issue",
		},
	}
}

func (m *mockCloseClient) GetProject(owner string, number int) (*api.Project, error) {
	if m.getProjectErr != nil {
		return nil, m.getProjectErr
	}
	return m.project, nil
}

func (m *mockCloseClient) GetProjectItemIDForIssue(projectID, owner, repo string, number int) (string, error) {
	if m.getProjectItemIDErr != nil {
		return "", m.getProjectItemIDErr
	}
	return m.itemID, nil
}

func (m *mockCloseClient) SetProjectItemField(projectID, itemID, fieldName, value string) error {
	return m.setProjectItemFieldErr
}

func (m *mockCloseClient) GetIssue(owner, repo string, number int) (*api.Issue, error) {
	if m.getIssueErr != nil {
		return nil, m.getIssueErr
	}
	return m.issue, nil
}

func (m *mockCloseClient) CloseIssue(issueID string, stateReason string) error {
	m.closedIssueID = issueID
	m.closedReason = stateReason
	return m.closeIssueErr
}

func (m *mockCloseClient) AddIssueComment(issueID, body string) (*api.Comment, error) {
	m.commentIssueID = issueID
	m.commentBody = body
	if m.addIssueCommentErr != nil {
		return nil, m.addIssueCommentErr
	}
	return &api.Comment{ID: "comment-1", Body: body}, nil
}

// ============================================================================
// updateStatusToDoneWithDeps Tests
// ============================================================================

func TestUpdateStatusToDoneWithDeps_Success(t *testing.T) {
	mock := newMockCloseClient()
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/test-repo"},
		Fields: map[string]config.Field{
			"status": {
				Field:  "Status",
				Values: map[string]string{"done": "Done"},
			},
		},
	}

	// Create a temp file to capture output
	stdout, _ := os.CreateTemp("", "stdout")
	defer os.Remove(stdout.Name())

	err := updateStatusToDoneWithDeps(42, "", cfg, mock, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read output
	_, _ = stdout.Seek(0, 0)
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(stdout)
	output := buf.String()

	if !strings.Contains(output, "#42") {
		t.Error("expected issue number in output")
	}
	if !strings.Contains(output, "Done") {
		t.Error("expected 'Done' status in output")
	}
}

func TestUpdateStatusToDoneWithDeps_WithRepoOverride(t *testing.T) {
	mock := newMockCloseClient()
	cfg := &config.Config{
		Project: config.Project{Owner: "test-org", Number: 1},
		Fields: map[string]config.Field{
			"status": {
				Field:  "Status",
				Values: map[string]string{"done": "Done"},
			},
		},
	}

	stdout, _ := os.CreateTemp("", "stdout")
	defer os.Remove(stdout.Name())

	err := updateStatusToDoneWithDeps(42, "other-org/other-repo", cfg, mock, stdout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStatusToDoneWithDeps_InvalidRepoFormat(t *testing.T) {
	mock := newMockCloseClient()
	cfg := &config.Config{
		Project: config.Project{Owner: "test-org", Number: 1},
	}

	stdout, _ := os.CreateTemp("", "stdout")
	defer os.Remove(stdout.Name())

	err := updateStatusToDoneWithDeps(42, "invalid-format", cfg, mock, stdout)
	if err == nil {
		t.Fatal("expected error for invalid repo format")
	}
	if !strings.Contains(err.Error(), "invalid --repo format") {
		t.Errorf("expected 'invalid --repo format' error, got: %v", err)
	}
}

func TestUpdateStatusToDoneWithDeps_NoRepoConfigured(t *testing.T) {
	mock := newMockCloseClient()
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{},
	}

	stdout, _ := os.CreateTemp("", "stdout")
	defer os.Remove(stdout.Name())

	err := updateStatusToDoneWithDeps(42, "", cfg, mock, stdout)
	if err == nil {
		t.Fatal("expected error when no repo configured")
	}
	if !strings.Contains(err.Error(), "no repository specified") {
		t.Errorf("expected 'no repository specified' error, got: %v", err)
	}
}

func TestUpdateStatusToDoneWithDeps_GetProjectError(t *testing.T) {
	mock := newMockCloseClient()
	mock.getProjectErr = errors.New("project not found")

	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/test-repo"},
	}

	stdout, _ := os.CreateTemp("", "stdout")
	defer os.Remove(stdout.Name())

	err := updateStatusToDoneWithDeps(42, "", cfg, mock, stdout)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get project") {
		t.Errorf("expected 'failed to get project' error, got: %v", err)
	}
}

func TestUpdateStatusToDoneWithDeps_IssueNotInProject(t *testing.T) {
	mock := newMockCloseClient()
	mock.getProjectItemIDErr = errors.New("issue not found in project")

	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/test-repo"},
	}

	stdout, _ := os.CreateTemp("", "stdout")
	defer os.Remove(stdout.Name())

	err := updateStatusToDoneWithDeps(42, "", cfg, mock, stdout)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to find issue in project") {
		t.Errorf("expected 'failed to find issue in project' error, got: %v", err)
	}
}

func TestUpdateStatusToDoneWithDeps_SetFieldError(t *testing.T) {
	mock := newMockCloseClient()
	mock.setProjectItemFieldErr = errors.New("API error")

	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/test-repo"},
		Fields: map[string]config.Field{
			"status": {
				Field:  "Status",
				Values: map[string]string{"done": "Done"},
			},
		},
	}

	stdout, _ := os.CreateTemp("", "stdout")
	defer os.Remove(stdout.Name())

	err := updateStatusToDoneWithDeps(42, "", cfg, mock, stdout)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to update status") {
		t.Errorf("expected 'failed to update status' error, got: %v", err)
	}
}

func TestNormalizeCloseReason(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		// not_planned variations
		{
			name:     "underscore not_planned",
			input:    "not_planned",
			expected: "not planned",
		},
		{
			name:     "space not planned",
			input:    "not planned",
			expected: "not planned",
		},
		{
			name:     "uppercase NOT_PLANNED",
			input:    "NOT_PLANNED",
			expected: "not planned",
		},
		{
			name:     "mixed case Not_Planned",
			input:    "Not_Planned",
			expected: "not planned",
		},
		{
			name:     "notplanned no separator",
			input:    "notplanned",
			expected: "not planned",
		},

		// completed variations
		{
			name:     "completed",
			input:    "completed",
			expected: "completed",
		},
		{
			name:     "COMPLETED uppercase",
			input:    "COMPLETED",
			expected: "completed",
		},
		{
			name:     "complete shorthand",
			input:    "complete",
			expected: "completed",
		},
		{
			name:     "done alias",
			input:    "done",
			expected: "completed",
		},

		// empty
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "  ",
			expected: "",
		},

		// invalid
		{
			name:      "invalid reason",
			input:     "invalid",
			expectErr: true,
		},
		{
			name:      "wontfix invalid",
			input:     "wontfix",
			expectErr: true,
		},
		{
			name:      "cancelled invalid",
			input:     "cancelled",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeCloseReason(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("normalizeCloseReason(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewCloseCommand(t *testing.T) {
	cmd := newCloseCommand()

	// Verify command basics
	if cmd.Use != "close <issue-number>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Verify flags exist
	flags := []struct {
		name      string
		shorthand string
	}{
		{"reason", "r"},
		{"comment", "c"},
		{"update-status", ""},
	}

	for _, f := range flags {
		flag := cmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("flag --%s not found", f.name)
			continue
		}
		if f.shorthand != "" && flag.Shorthand != f.shorthand {
			t.Errorf("flag --%s shorthand = %q, want %q", f.name, flag.Shorthand, f.shorthand)
		}
	}
}

func TestNewCloseCommand_RequiresArg(t *testing.T) {
	cmd := newCloseCommand()

	// Command requires exactly 1 argument
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("expected error when no arguments provided")
	}

	err = cmd.Args(cmd, []string{"123"})
	if err != nil {
		t.Errorf("unexpected error with one argument: %v", err)
	}

	err = cmd.Args(cmd, []string{"123", "456"})
	if err == nil {
		t.Error("expected error when too many arguments provided")
	}
}

// ============================================================================
// runCloseWithClient Tests
// ============================================================================

func TestRunCloseWithClient_BasicClose(t *testing.T) {
	mock := newMockCloseClient()
	cmd := newCloseCommand()

	err := runCloseWithClient(cmd, mock, "test-org", "test-repo", 42, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.closedIssueID != "issue-node-123" {
		t.Errorf("expected closedIssueID 'issue-node-123', got %q", mock.closedIssueID)
	}
	if mock.closedReason != "" {
		t.Errorf("expected empty reason, got %q", mock.closedReason)
	}
}

func TestRunCloseWithClient_WithCompletedReason(t *testing.T) {
	mock := newMockCloseClient()
	cmd := newCloseCommand()

	err := runCloseWithClient(cmd, mock, "test-org", "test-repo", 42, "completed", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.closedReason != "COMPLETED" {
		t.Errorf("expected reason 'COMPLETED', got %q", mock.closedReason)
	}
}

func TestRunCloseWithClient_WithNotPlannedReason(t *testing.T) {
	mock := newMockCloseClient()
	cmd := newCloseCommand()

	err := runCloseWithClient(cmd, mock, "test-org", "test-repo", 42, "not planned", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.closedReason != "NOT_PLANNED" {
		t.Errorf("expected reason 'NOT_PLANNED', got %q", mock.closedReason)
	}
}

func TestRunCloseWithClient_WithComment(t *testing.T) {
	mock := newMockCloseClient()
	cmd := newCloseCommand()

	err := runCloseWithClient(cmd, mock, "test-org", "test-repo", 42, "", "Fixed in v1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.commentIssueID != "issue-node-123" {
		t.Errorf("expected comment on issue-node-123, got %q", mock.commentIssueID)
	}
	if mock.commentBody != "Fixed in v1.0" {
		t.Errorf("expected comment body 'Fixed in v1.0', got %q", mock.commentBody)
	}
	if mock.closedIssueID != "issue-node-123" {
		t.Errorf("expected issue closed after comment")
	}
}

func TestRunCloseWithClient_WithReasonAndComment(t *testing.T) {
	mock := newMockCloseClient()
	cmd := newCloseCommand()

	err := runCloseWithClient(cmd, mock, "test-org", "test-repo", 42, "not planned", "Duplicate of #100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.closedReason != "NOT_PLANNED" {
		t.Errorf("expected reason 'NOT_PLANNED', got %q", mock.closedReason)
	}
	if mock.commentBody != "Duplicate of #100" {
		t.Errorf("expected comment body, got %q", mock.commentBody)
	}
}

func TestRunCloseWithClient_GetIssueError(t *testing.T) {
	mock := newMockCloseClient()
	mock.getIssueErr = errors.New("issue not found")
	cmd := newCloseCommand()

	err := runCloseWithClient(cmd, mock, "test-org", "test-repo", 999, "", "")
	if err == nil {
		t.Fatal("expected error when issue not found")
	}
	if !strings.Contains(err.Error(), "failed to get issue #999") {
		t.Errorf("expected 'failed to get issue' error, got: %v", err)
	}
}

func TestRunCloseWithClient_CloseError(t *testing.T) {
	mock := newMockCloseClient()
	mock.closeIssueErr = errors.New("mutation failed")
	cmd := newCloseCommand()

	err := runCloseWithClient(cmd, mock, "test-org", "test-repo", 42, "", "")
	if err == nil {
		t.Fatal("expected error when close fails")
	}
	if !strings.Contains(err.Error(), "failed to close issue #42") {
		t.Errorf("expected 'failed to close issue' error, got: %v", err)
	}
}

func TestRunCloseWithClient_CommentError(t *testing.T) {
	mock := newMockCloseClient()
	mock.addIssueCommentErr = errors.New("comment failed")
	cmd := newCloseCommand()

	err := runCloseWithClient(cmd, mock, "test-org", "test-repo", 42, "", "some comment")
	if err == nil {
		t.Fatal("expected error when comment fails")
	}
	if !strings.Contains(err.Error(), "failed to add closing comment") {
		t.Errorf("expected 'failed to add closing comment' error, got: %v", err)
	}
	// Issue should NOT be closed if comment failed
	if mock.closedIssueID != "" {
		t.Error("issue should not be closed when comment fails")
	}
}

// ============================================================================
// reasonToGraphQL Tests
// ============================================================================

func TestReasonToGraphQL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"completed", "COMPLETED"},
		{"not planned", "NOT_PLANNED"},
		{"", ""},
		{"unknown", ""},
	}

	for _, tt := range tests {
		result := reasonToGraphQL(tt.input)
		if result != tt.expected {
			t.Errorf("reasonToGraphQL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRunClose_InvalidIssueNumber(t *testing.T) {
	cmd := newCloseCommand()
	opts := &closeOptions{}

	err := runClose(cmd, []string{"not-a-number"}, opts)
	if err == nil {
		t.Error("expected error for non-numeric issue number")
	}
}

func TestRunClose_InvalidReason(t *testing.T) {
	cmd := newCloseCommand()
	opts := &closeOptions{
		reason: "invalid_reason",
	}

	err := runClose(cmd, []string{"123"}, opts)
	if err == nil {
		t.Error("expected error for invalid close reason")
	}
}
