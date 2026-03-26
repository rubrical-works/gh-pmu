package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/spf13/cobra"
)

// mockEditClient implements editClient for testing
type mockEditClient struct {
	issue            *api.Issue
	getIssueErr      error
	updateBodyErr    error
	updateTitleErr   error
	addLabelErr      error
	removeLabelErr   error
	updateBodyCalls  []string
	updateTitleCalls []string
	addLabelCalls    []string
	removeLabelCalls []string
}

func (m *mockEditClient) GetIssueByNumber(owner, repo string, number int) (*api.Issue, error) {
	if m.getIssueErr != nil {
		return nil, m.getIssueErr
	}
	return m.issue, nil
}

func (m *mockEditClient) UpdateIssueBody(issueID, body string) error {
	m.updateBodyCalls = append(m.updateBodyCalls, body)
	return m.updateBodyErr
}

func (m *mockEditClient) UpdateIssueTitle(issueID, title string) error {
	m.updateTitleCalls = append(m.updateTitleCalls, title)
	return m.updateTitleErr
}

func (m *mockEditClient) AddLabelToIssue(owner, repo, issueID, labelName string) error {
	m.addLabelCalls = append(m.addLabelCalls, labelName)
	return m.addLabelErr
}

func (m *mockEditClient) RemoveLabelFromIssue(owner, repo, issueID, labelName string) error {
	m.removeLabelCalls = append(m.removeLabelCalls, labelName)
	return m.removeLabelErr
}

func setupMockForEdit() *mockEditClient {
	return &mockEditClient{
		issue: &api.Issue{
			ID:     "ISSUE_123",
			Number: 123,
			Title:  "Test Issue",
			URL:    "https://github.com/testowner/testrepo/issues/123",
		},
	}
}

func testEditConfig() *config.Config {
	return &config.Config{
		Project: config.Project{
			Owner:  "testowner",
			Number: 1,
		},
		Repositories: []string{"testowner/testrepo"},
	}
}

func newTestEditCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{Use: "edit"}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd, buf
}

// ============================================================================
// Edit Command Structure Tests
// ============================================================================

func TestEditCommand_Structure(t *testing.T) {
	cmd := NewRootCommand()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("edit command not found: %v", err)
	}

	if editCmd.Use != "edit <issue-number>" {
		t.Errorf("Expected Use 'edit <issue-number>', got '%s'", editCmd.Use)
	}

	// Requires exactly 1 argument
	if err := editCmd.Args(editCmd, []string{}); err == nil {
		t.Error("Expected error when no arguments provided")
	}
	if err := editCmd.Args(editCmd, []string{"123"}); err != nil {
		t.Errorf("Unexpected error with one argument: %v", err)
	}
}

func TestEditCommand_Flags(t *testing.T) {
	cmd := NewRootCommand()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("edit command not found: %v", err)
	}

	tests := []struct {
		flag      string
		shorthand string
	}{
		{"title", "t"},
		{"body", "b"},
		{"body-file", "F"},
		{"label", "l"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := editCmd.Flags().Lookup(tt.flag)
			if flag == nil {
				t.Fatalf("Expected --%s flag to exist", tt.flag)
			}
			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("Expected --%s shorthand to be '%s', got '%s'", tt.flag, tt.shorthand, flag.Shorthand)
			}
		})
	}
}

// ============================================================================
// Edit Command Behavior Tests
// ============================================================================

func TestRunEditWithDeps_UpdatesTitle(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, buf := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		title:       "New Title",
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(mock.updateTitleCalls) != 1 || mock.updateTitleCalls[0] != "New Title" {
		t.Errorf("Expected title update to 'New Title', got: %v", mock.updateTitleCalls)
	}
	output := buf.String()
	if !contains(output, "Updated issue #123") {
		t.Errorf("Expected output to contain 'Updated issue #123', got: %s", output)
	}
}

func TestRunEditWithDeps_UpdatesBody(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		body:        "New body content",
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(mock.updateBodyCalls) != 1 || mock.updateBodyCalls[0] != "New body content" {
		t.Errorf("Expected body update to 'New body content', got: %v", mock.updateBodyCalls)
	}
}

func TestRunEditWithDeps_AddsLabels(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, buf := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		addLabels:   []string{"bug", "urgent"},
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(mock.addLabelCalls) != 2 {
		t.Errorf("Expected 2 label calls, got: %d", len(mock.addLabelCalls))
	}
	output := buf.String()
	if !contains(output, "2 label(s)") {
		t.Errorf("Expected output to contain '2 label(s)', got: %s", output)
	}
}

func TestRunEditWithDeps_MultipleUpdates(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, buf := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		title:       "New Title",
		body:        "New body",
		addLabels:   []string{"fix"},
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(mock.updateTitleCalls) != 1 {
		t.Errorf("Expected 1 title update, got: %d", len(mock.updateTitleCalls))
	}
	if len(mock.updateBodyCalls) != 1 {
		t.Errorf("Expected 1 body update, got: %d", len(mock.updateBodyCalls))
	}
	if len(mock.addLabelCalls) != 1 {
		t.Errorf("Expected 1 label call, got: %d", len(mock.addLabelCalls))
	}
	output := buf.String()
	if !contains(output, "title") || !contains(output, "body") || !contains(output, "label") {
		t.Errorf("Expected output to contain all update types, got: %s", output)
	}
}

func TestRunEditWithDeps_RequiresAtLeastOneOption(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		// No title, body, bodyFile, or labels
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err == nil {
		t.Fatal("Expected error when no options provided")
	}
	if !contains(err.Error(), "at least one of") {
		t.Errorf("Expected 'at least one of' error, got: %s", err.Error())
	}
}

func TestRunEditWithDeps_CannotUseBothBodyAndBodyFile(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		body:        "inline body",
		bodyFile:    "file.md",
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err == nil {
		t.Fatal("Expected error when both body and body-file provided")
	}
	if !contains(err.Error(), "cannot use --body and --body-file together") {
		t.Errorf("Expected 'cannot use --body and --body-file together' error, got: %s", err.Error())
	}
}

// ============================================================================
// Edit Command Error Handling Tests
// ============================================================================

func TestRunEditWithDeps_GetIssueError(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	mock.getIssueErr = fmt.Errorf("issue not found")
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 999,
		title:       "New Title",
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err == nil {
		t.Fatal("Expected error when GetIssueByNumber fails")
	}
	if !contains(err.Error(), "failed to get issue") {
		t.Errorf("Expected 'failed to get issue' error, got: %s", err.Error())
	}
}

func TestRunEditWithDeps_UpdateTitleError(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	mock.updateTitleErr = fmt.Errorf("update failed")
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		title:       "New Title",
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err == nil {
		t.Fatal("Expected error when UpdateIssueTitle fails")
	}
	if !contains(err.Error(), "failed to update title") {
		t.Errorf("Expected 'failed to update title' error, got: %s", err.Error())
	}
}

func TestRunEditWithDeps_UpdateBodyError(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	mock.updateBodyErr = fmt.Errorf("update failed")
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		body:        "New body",
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err == nil {
		t.Fatal("Expected error when UpdateIssueBody fails")
	}
	if !contains(err.Error(), "failed to update body") {
		t.Errorf("Expected 'failed to update body' error, got: %s", err.Error())
	}
}

func TestRunEditWithDeps_AddLabelError(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	mock.addLabelErr = fmt.Errorf("label not found")
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		addLabels:   []string{"nonexistent"},
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err == nil {
		t.Fatal("Expected error when AddLabelToIssue fails")
	}
	if !contains(err.Error(), "failed to apply labels") {
		t.Errorf("Expected 'failed to apply labels' error, got: %s", err.Error())
	}
}

func TestRunEditWithDeps_RemovesLabels(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, buf := newTestEditCmd()
	opts := &editOptions{
		issueNumber:  123,
		removeLabels: []string{"old-label", "deprecated"},
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(mock.removeLabelCalls) != 2 {
		t.Errorf("Expected 2 remove label calls, got: %d", len(mock.removeLabelCalls))
	}
	output := buf.String()
	if !contains(output, "2 label(s) removed") {
		t.Errorf("Expected output to contain '2 label(s) removed', got: %s", output)
	}
}

func TestRunEditWithDeps_RemoveLabelError(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	mock.removeLabelErr = fmt.Errorf("label not found")
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber:  123,
		removeLabels: []string{"nonexistent"},
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err == nil {
		t.Fatal("Expected error when RemoveLabelFromIssue fails")
	}
	if !contains(err.Error(), "failed to apply labels") {
		t.Errorf("Expected 'failed to apply labels' error, got: %s", err.Error())
	}
}

func TestRunEditWithDeps_AddAndRemoveLabels(t *testing.T) {
	// ARRANGE
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, buf := newTestEditCmd()
	opts := &editOptions{
		issueNumber:  123,
		addLabels:    []string{"new-label"},
		removeLabels: []string{"old-label"},
	}

	// ACT
	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(mock.addLabelCalls) != 1 {
		t.Errorf("Expected 1 add label call, got: %d", len(mock.addLabelCalls))
	}
	if len(mock.removeLabelCalls) != 1 {
		t.Errorf("Expected 1 remove label call, got: %d", len(mock.removeLabelCalls))
	}
	output := buf.String()
	if !contains(output, "added") || !contains(output, "removed") {
		t.Errorf("Expected output to contain both 'added' and 'removed', got: %s", output)
	}
}

func TestEditCommand_HasRemoveLabelFlag(t *testing.T) {
	cmd := NewRootCommand()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("edit command not found: %v", err)
	}

	flag := editCmd.Flags().Lookup("remove-label")
	if flag == nil {
		t.Fatal("Expected --remove-label flag to exist")
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// Body Stdin Tests
// ============================================================================

func TestEditCommand_HasBodyStdinFlag(t *testing.T) {
	cmd := NewRootCommand()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("edit command not found: %v", err)
	}

	flag := editCmd.Flags().Lookup("body-stdin")
	if flag == nil {
		t.Fatal("Expected --body-stdin flag to exist")
	}

	// body-stdin has no shorthand
	if flag.Shorthand != "" {
		t.Errorf("Expected --body-stdin to have no shorthand, got %s", flag.Shorthand)
	}
}

func TestRunEditWithDeps_BodyStdinSetsBodyFile(t *testing.T) {
	// When bodyStdin is true, runEditWithDeps converts it to bodyFile="-"
	// which reads from os.Stdin via readBodyFile. Verify the conversion
	// happens by checking opts.bodyFile after the call.
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		bodyStdin:   true,
	}

	// Verify initial state
	if opts.bodyFile != "" {
		t.Fatalf("Expected bodyFile to be empty initially, got %q", opts.bodyFile)
	}

	// Call runEditWithDeps — will fail reading stdin (EOF) but conversion happens first
	_ = runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// Verify bodyStdin was converted to bodyFile="-"
	if opts.bodyFile != "-" {
		t.Errorf("Expected bodyFile to be converted to \"-\" from bodyStdin, got %q", opts.bodyFile)
	}
}

func TestRunEditWithDeps_CannotUseBodyStdinWithBody(t *testing.T) {
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		body:        "Some body",
		bodyStdin:   true,
	}

	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	if err == nil {
		t.Fatal("Expected error when using --body-stdin with --body")
	}
	if !contains(err.Error(), "cannot use --body-stdin with --body or --body-file") {
		t.Errorf("Expected mutual exclusion error, got: %s", err.Error())
	}
}

func TestRunEditWithDeps_CannotUseBodyStdinWithBodyFile(t *testing.T) {
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		bodyFile:    "some-file.md",
		bodyStdin:   true,
	}

	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	if err == nil {
		t.Fatal("Expected error when using --body-stdin with --body-file")
	}
	if !contains(err.Error(), "cannot use --body-stdin with --body or --body-file") {
		t.Errorf("Expected mutual exclusion error, got: %s", err.Error())
	}
}

func TestRunEditWithDeps_BodyStdinCountsAsOption(t *testing.T) {
	// Verify --body-stdin counts as a valid option (no "at least one option" error)
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, _ := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		bodyStdin:   true,
	}

	err := runEditWithDeps(cmd, opts, cfg, mock, "testowner", "testrepo")

	// Should NOT get "at least one of" error - will get stdin EOF error instead
	if err != nil && contains(err.Error(), "at least one of") {
		t.Errorf("--body-stdin should count as a valid option, got: %s", err.Error())
	}
}

// ============================================================================
// Issue #492: --repo flag Tests
// ============================================================================

func TestEditCommand_HasRepoFlag(t *testing.T) {
	cmd := NewRootCommand()
	editCmd, _, err := cmd.Find([]string{"edit"})
	if err != nil {
		t.Fatalf("edit command not found: %v", err)
	}

	flag := editCmd.Flags().Lookup("repo")
	if flag == nil {
		t.Fatal("Expected --repo flag to exist")
	}
	if flag.Shorthand != "R" {
		t.Errorf("Expected shorthand 'R', got '%s'", flag.Shorthand)
	}
}

func TestRunEditWithDeps_UsesRepoFromOptions(t *testing.T) {
	// When --repo is provided, it should use that repo instead of config
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, buf := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 123,
		title:       "New Title",
		repo:        "otherowner/otherrepo",
	}

	// The mock will be called with otherowner/otherrepo
	// (we verify by checking success - the mock doesn't care about owner/repo)
	err := runEditWithDeps(cmd, opts, cfg, mock, "otherowner", "otherrepo")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()
	if !contains(output, "Updated issue #123") {
		t.Errorf("Expected success output, got: %s", output)
	}
}

func TestRunEditWithDeps_CrossRepoEditing(t *testing.T) {
	// Verify cross-repo editing works with different owner/repo
	mock := setupMockForEdit()
	cfg := testEditConfig()
	cmd, buf := newTestEditCmd()
	opts := &editOptions{
		issueNumber: 456,
		body:        "Updated body content",
	}

	// Call with a different repo than config
	err := runEditWithDeps(cmd, opts, cfg, mock, "differentowner", "differentrepo")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()
	if !contains(output, "Updated issue #456") {
		t.Errorf("Expected success output, got: %s", output)
	}
	if len(mock.updateBodyCalls) != 1 || mock.updateBodyCalls[0] != "Updated body content" {
		t.Errorf("Expected body update call, got: %v", mock.updateBodyCalls)
	}
}
