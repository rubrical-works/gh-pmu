package cmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/spf13/cobra"
)

// mockFilterClient implements filterClient for testing
type mockFilterClient struct {
	project      *api.Project
	projectItems []api.ProjectItem

	// Call tracking
	getProjectItemsCalls         int
	getProjectItemsByIssuesCalls int
	lastIssueRefs                []api.IssueRef

	// Error injection
	getProjectErr              error
	getProjectItemsErr         error
	getProjectItemsByIssuesErr error
}

func newMockFilterClient() *mockFilterClient {
	return &mockFilterClient{
		project: &api.Project{
			ID:    "proj-1",
			Title: "Test Project",
		},
		projectItems: []api.ProjectItem{},
	}
}

func (m *mockFilterClient) GetProject(owner string, number int) (*api.Project, error) {
	if m.getProjectErr != nil {
		return nil, m.getProjectErr
	}
	return m.project, nil
}

func (m *mockFilterClient) GetProjectItems(projectID string, filter *api.ProjectItemsFilter) ([]api.ProjectItem, error) {
	m.getProjectItemsCalls++
	if m.getProjectItemsErr != nil {
		return nil, m.getProjectItemsErr
	}
	return m.projectItems, nil
}

func (m *mockFilterClient) GetProjectItemsByIssues(projectID string, refs []api.IssueRef) ([]api.ProjectItem, error) {
	m.getProjectItemsByIssuesCalls++
	m.lastIssueRefs = refs
	if m.getProjectItemsByIssuesErr != nil {
		return nil, m.getProjectItemsByIssuesErr
	}
	// Return only items matching the refs
	var result []api.ProjectItem
	refSet := make(map[int]bool)
	for _, ref := range refs {
		refSet[ref.Number] = true
	}
	for _, item := range m.projectItems {
		if item.Issue != nil && refSet[item.Issue.Number] {
			result = append(result, item)
		}
	}
	return result, nil
}

func TestFilterCommand_Exists(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"filter", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("filter command should exist: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "filter") {
		t.Error("Expected help output to mention 'filter'")
	}
}

func TestFilterCommand_HasStatusFlag(t *testing.T) {
	cmd := NewRootCommand()
	filterCmd, _, err := cmd.Find([]string{"filter"})
	if err != nil {
		t.Fatalf("filter command not found: %v", err)
	}

	flag := filterCmd.Flags().Lookup("status")
	if flag == nil {
		t.Fatal("Expected --status flag to exist")
	}
	if flag.Shorthand != "s" {
		t.Errorf("Expected shorthand 's', got '%s'", flag.Shorthand)
	}
}

func TestFilterCommand_HasPriorityFlag(t *testing.T) {
	cmd := NewRootCommand()
	filterCmd, _, err := cmd.Find([]string{"filter"})
	if err != nil {
		t.Fatalf("filter command not found: %v", err)
	}

	flag := filterCmd.Flags().Lookup("priority")
	if flag == nil {
		t.Fatal("Expected --priority flag to exist")
	}
	if flag.Shorthand != "p" {
		t.Errorf("Expected shorthand 'p', got '%s'", flag.Shorthand)
	}
}

func TestFilterCommand_HasAssigneeFlag(t *testing.T) {
	cmd := NewRootCommand()
	filterCmd, _, err := cmd.Find([]string{"filter"})
	if err != nil {
		t.Fatalf("filter command not found: %v", err)
	}

	flag := filterCmd.Flags().Lookup("assignee")
	if flag == nil {
		t.Fatal("Expected --assignee flag to exist")
	}
	if flag.Shorthand != "a" {
		t.Errorf("Expected shorthand 'a', got '%s'", flag.Shorthand)
	}
}

func TestFilterCommand_HasLabelFlag(t *testing.T) {
	cmd := NewRootCommand()
	filterCmd, _, err := cmd.Find([]string{"filter"})
	if err != nil {
		t.Fatalf("filter command not found: %v", err)
	}

	flag := filterCmd.Flags().Lookup("label")
	if flag == nil {
		t.Fatal("Expected --label flag to exist")
	}
	if flag.Shorthand != "l" {
		t.Errorf("Expected shorthand 'l', got '%s'", flag.Shorthand)
	}
}

func TestFilterCommand_HasJSONFlag(t *testing.T) {
	cmd := NewRootCommand()
	filterCmd, _, err := cmd.Find([]string{"filter"})
	if err != nil {
		t.Fatalf("filter command not found: %v", err)
	}

	flag := filterCmd.Flags().Lookup("json")
	if flag == nil {
		t.Fatal("Expected --json flag to exist")
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("Expected --json to be bool, got %s", flag.Value.Type())
	}
}

func TestFilterCommand_HelpText(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"filter", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("filter help failed: %v", err)
	}

	output := buf.String()

	// Should mention piping from gh issue list
	if !strings.Contains(output, "gh issue list") {
		t.Error("Expected help to mention 'gh issue list'")
	}

	// Should have example usage
	if !strings.Contains(output, "Example") {
		t.Error("Expected help to have Example section")
	}
}

// ============================================================================
// hasFieldValue Tests
// ============================================================================

func TestHasFieldValue_Filter(t *testing.T) {
	tests := []struct {
		name      string
		item      api.ProjectItem
		fieldName string
		value     string
		want      bool
	}{
		{
			name: "exact match",
			item: api.ProjectItem{
				FieldValues: []api.FieldValue{
					{Field: "Status", Value: "In Progress"},
				},
			},
			fieldName: "Status",
			value:     "In Progress",
			want:      true,
		},
		{
			name: "case insensitive field",
			item: api.ProjectItem{
				FieldValues: []api.FieldValue{
					{Field: "Status", Value: "Ready"},
				},
			},
			fieldName: "status",
			value:     "Ready",
			want:      true,
		},
		{
			name: "case insensitive value",
			item: api.ProjectItem{
				FieldValues: []api.FieldValue{
					{Field: "Status", Value: "In Progress"},
				},
			},
			fieldName: "Status",
			value:     "in progress",
			want:      true,
		},
		{
			name: "no match",
			item: api.ProjectItem{
				FieldValues: []api.FieldValue{
					{Field: "Status", Value: "Done"},
				},
			},
			fieldName: "Status",
			value:     "Ready",
			want:      false,
		},
		{
			name:      "empty field values",
			item:      api.ProjectItem{},
			fieldName: "Status",
			value:     "Ready",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasFieldValue(tt.item, tt.fieldName, tt.value)
			if got != tt.want {
				t.Errorf("hasFieldValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// hasAssignee Tests
// ============================================================================

func TestHasAssignee(t *testing.T) {
	tests := []struct {
		name     string
		issue    FilterInput
		assignee string
		want     bool
	}{
		{
			name: "exact match",
			issue: FilterInput{
				Assignees: []User{{Login: "user1"}},
			},
			assignee: "user1",
			want:     true,
		},
		{
			name: "case insensitive",
			issue: FilterInput{
				Assignees: []User{{Login: "User1"}},
			},
			assignee: "user1",
			want:     true,
		},
		{
			name: "multiple assignees",
			issue: FilterInput{
				Assignees: []User{{Login: "user1"}, {Login: "user2"}},
			},
			assignee: "user2",
			want:     true,
		},
		{
			name: "no match",
			issue: FilterInput{
				Assignees: []User{{Login: "user1"}},
			},
			assignee: "user3",
			want:     false,
		},
		{
			name:     "empty assignees",
			issue:    FilterInput{},
			assignee: "user1",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAssignee(tt.issue, tt.assignee)
			if got != tt.want {
				t.Errorf("hasAssignee() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// hasLabel Tests
// ============================================================================

func TestHasLabel(t *testing.T) {
	tests := []struct {
		name  string
		issue FilterInput
		label string
		want  bool
	}{
		{
			name: "exact match",
			issue: FilterInput{
				Labels: []Label{{Name: "bug"}},
			},
			label: "bug",
			want:  true,
		},
		{
			name: "case insensitive",
			issue: FilterInput{
				Labels: []Label{{Name: "Bug"}},
			},
			label: "bug",
			want:  true,
		},
		{
			name: "multiple labels",
			issue: FilterInput{
				Labels: []Label{{Name: "bug"}, {Name: "enhancement"}},
			},
			label: "enhancement",
			want:  true,
		},
		{
			name: "no match",
			issue: FilterInput{
				Labels: []Label{{Name: "bug"}},
			},
			label: "feature",
			want:  false,
		},
		{
			name:  "empty labels",
			issue: FilterInput{},
			label: "bug",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasLabel(tt.issue, tt.label)
			if got != tt.want {
				t.Errorf("hasLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// outputFilterTable Tests
// ============================================================================

func TestOutputFilterTable_EmptyIssues(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createTestCmd(buf)

	err := outputFilterTable(cmd, []FilterInput{})
	if err != nil {
		t.Fatalf("outputFilterTable() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No matching issues found") {
		t.Errorf("Expected 'No matching issues found', got: %s", output)
	}
}

func TestOutputFilterTable_WithIssues(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createTestCmd(buf)

	issues := []FilterInput{
		{
			Number: 42,
			Title:  "Test Issue",
			State:  "open",
		},
		{
			Number: 43,
			Title:  "Another Issue",
			State:  "closed",
		},
	}

	err := outputFilterTable(cmd, issues)
	if err != nil {
		t.Fatalf("outputFilterTable() error = %v", err)
	}
	// Function writes to os.Stdout, not buf, so we just verify no error
}

func TestOutputFilterTable_TitleTruncation(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createTestCmd(buf)

	longTitle := "This is a very long title that exceeds fifty characters and should be truncated by the table output"
	issues := []FilterInput{
		{
			Number: 42,
			Title:  longTitle,
			State:  "open",
		},
	}

	err := outputFilterTable(cmd, issues)
	if err != nil {
		t.Fatalf("outputFilterTable() error = %v", err)
	}
	// Function writes to os.Stdout, not buf, so we just verify no error
}

// ============================================================================
// outputFilterJSON Tests
// ============================================================================

func TestOutputFilterJSON_EmptyIssues(t *testing.T) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := outputFilterJSON(cmd, []FilterInput{})
	if err != nil {
		t.Fatalf("outputFilterJSON() error = %v", err)
	}
}

func TestOutputFilterJSON_WithIssues(t *testing.T) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	issues := []FilterInput{
		{
			Number: 42,
			Title:  "Test Issue",
			State:  "open",
			URL:    "https://github.com/owner/repo/issues/42",
		},
	}

	err := outputFilterJSON(cmd, issues)
	if err != nil {
		t.Fatalf("outputFilterJSON() error = %v", err)
	}
}

// ============================================================================
// runFilterWithDeps Tests
// ============================================================================

// createTempStdin creates a temp file with the given content for use as stdin
func createTempStdin(t *testing.T, content string) *os.File {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "stdin-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	if _, err := tmpfile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek temp file: %v", err)
	}
	return tmpfile
}

func TestRunFilterWithDeps_GetProjectError(t *testing.T) {
	mock := newMockFilterClient()
	mock.getProjectErr = errors.New("project not found")
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/repo"},
	}

	stdin := createTempStdin(t, `[{"number": 1, "title": "Test"}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{}
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get project") {
		t.Errorf("expected 'failed to get project' error, got: %v", err)
	}
}

func TestRunFilterWithDeps_GetProjectItemsError_OnFallback(t *testing.T) {
	mock := newMockFilterClient()
	mock.getProjectItemsByIssuesErr = errors.New("targeted query failed")
	mock.getProjectItemsErr = errors.New("items not found")
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	stdin := createTempStdin(t, `[{"number": 1, "title": "Test"}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{}
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get project items") {
		t.Errorf("expected 'failed to get project items' error, got: %v", err)
	}
}

func TestRunFilterWithDeps_EmptyInput(t *testing.T) {
	mock := newMockFilterClient()
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/repo"},
	}

	stdin := createTempStdin(t, "")
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{}
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)

	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "empty input") {
		t.Errorf("expected 'empty input' error, got: %v", err)
	}
}

func TestRunFilterWithDeps_InvalidJSON(t *testing.T) {
	mock := newMockFilterClient()
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/repo"},
	}

	stdin := createTempStdin(t, "not valid json")
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{}
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse JSON") {
		t.Errorf("expected 'failed to parse JSON' error, got: %v", err)
	}
}

func TestRunFilterWithDeps_NoMatchingIssues(t *testing.T) {
	mock := newMockFilterClient()
	mock.projectItems = []api.ProjectItem{
		{
			Issue: &api.Issue{Number: 99}, // Different issue number
			FieldValues: []api.FieldValue{
				{Field: "Status", Value: "Ready"},
			},
		},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/repo"},
	}

	stdin := createTempStdin(t, `[{"number": 1, "title": "Test Issue", "state": "open"}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &filterOptions{}
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No matching issues found") {
		t.Errorf("expected 'No matching issues found', got: %s", output)
	}
}

func TestRunFilterWithDeps_FilterByStatus(t *testing.T) {
	mock := newMockFilterClient()
	mock.projectItems = []api.ProjectItem{
		{
			Issue: &api.Issue{Number: 1},
			FieldValues: []api.FieldValue{
				{Field: "Status", Value: "Ready"},
			},
		},
		{
			Issue: &api.Issue{Number: 2},
			FieldValues: []api.FieldValue{
				{Field: "Status", Value: "In Progress"},
			},
		},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/repo"},
		Fields: map[string]config.Field{
			"status": {
				Field:  "Status",
				Values: map[string]string{"ready": "Ready"},
			},
		},
	}

	stdin := createTempStdin(t, `[{"number": 1, "title": "Issue 1", "state": "open"}, {"number": 2, "title": "Issue 2", "state": "open"}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{status: "ready"}
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Output goes to stdout - verified by no error
}

func TestRunFilterWithDeps_FilterByPriority(t *testing.T) {
	mock := newMockFilterClient()
	mock.projectItems = []api.ProjectItem{
		{
			Issue: &api.Issue{Number: 1},
			FieldValues: []api.FieldValue{
				{Field: "Priority", Value: "P1"},
			},
		},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/repo"},
		Fields: map[string]config.Field{
			"priority": {
				Field:  "Priority",
				Values: map[string]string{"p1": "P1"},
			},
		},
	}

	stdin := createTempStdin(t, `[{"number": 1, "title": "Issue 1", "state": "open"}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{priority: "p1"}
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFilterWithDeps_FilterByAssignee(t *testing.T) {
	mock := newMockFilterClient()
	mock.projectItems = []api.ProjectItem{
		{
			Issue: &api.Issue{Number: 1},
		},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/repo"},
	}

	stdin := createTempStdin(t, `[{"number": 1, "title": "Issue 1", "state": "open", "assignees": [{"login": "user1"}]}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{assignee: "user1"}
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFilterWithDeps_FilterByLabel(t *testing.T) {
	mock := newMockFilterClient()
	mock.projectItems = []api.ProjectItem{
		{
			Issue: &api.Issue{Number: 1},
		},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/repo"},
	}

	stdin := createTempStdin(t, `[{"number": 1, "title": "Issue 1", "state": "open", "labels": [{"name": "bug"}]}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{label: "bug"}
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFilterWithDeps_JSONOutput(t *testing.T) {
	mock := newMockFilterClient()
	mock.projectItems = []api.ProjectItem{
		{
			Issue: &api.Issue{Number: 1},
		},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/repo"},
	}

	stdin := createTempStdin(t, `[{"number": 1, "title": "Issue 1", "state": "open"}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{json: true}
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Targeted Query Optimization Tests (Issue #545)
// ============================================================================

func TestRunFilterWithDeps_UsesTargetedQuery_WithURL(t *testing.T) {
	// ARRANGE - Input with URL should use targeted query
	mock := newMockFilterClient()
	mock.projectItems = []api.ProjectItem{
		{Issue: &api.Issue{Number: 1}},
		{Issue: &api.Issue{Number: 2}},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"test-org/repo"},
	}

	// Input with URLs allows targeted query
	stdin := createTempStdin(t, `[
		{"number": 1, "title": "Issue 1", "state": "open", "url": "https://github.com/owner/repo/issues/1"},
		{"number": 2, "title": "Issue 2", "state": "open", "url": "https://github.com/owner/repo/issues/2"}
	]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{}

	// ACT
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Should use targeted query, not full fetch
	if mock.getProjectItemsByIssuesCalls != 1 {
		t.Errorf("Expected 1 GetProjectItemsByIssues call, got %d", mock.getProjectItemsByIssuesCalls)
	}
	if mock.getProjectItemsCalls != 0 {
		t.Errorf("Expected 0 GetProjectItems calls (should use targeted), got %d", mock.getProjectItemsCalls)
	}
	// Should have built refs from URLs
	if len(mock.lastIssueRefs) != 2 {
		t.Errorf("Expected 2 issue refs, got %d", len(mock.lastIssueRefs))
	}
}

func TestRunFilterWithDeps_UsesTargetedQuery_WithConfigRepo(t *testing.T) {
	// ARRANGE - Input without URL falls back to config repo
	mock := newMockFilterClient()
	mock.projectItems = []api.ProjectItem{
		{Issue: &api.Issue{Number: 1}},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"fallback-owner/fallback-repo"},
	}

	// Input without URLs - should fall back to config repo
	stdin := createTempStdin(t, `[{"number": 1, "title": "Issue 1", "state": "open"}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{}

	// ACT
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Should use targeted query with fallback repo
	if mock.getProjectItemsByIssuesCalls != 1 {
		t.Errorf("Expected 1 GetProjectItemsByIssues call, got %d", mock.getProjectItemsByIssuesCalls)
	}
	if len(mock.lastIssueRefs) != 1 {
		t.Errorf("Expected 1 issue ref, got %d", len(mock.lastIssueRefs))
	}
	if mock.lastIssueRefs[0].Owner != "fallback-owner" {
		t.Errorf("Expected owner 'fallback-owner', got %s", mock.lastIssueRefs[0].Owner)
	}
}

func TestRunFilterWithDeps_FallsBackToFullFetch_OnTargetedError(t *testing.T) {
	// ARRANGE - Targeted query fails, should fall back to full fetch
	mock := newMockFilterClient()
	mock.projectItems = []api.ProjectItem{
		{Issue: &api.Issue{Number: 1}},
	}
	mock.getProjectItemsByIssuesErr = errors.New("targeted query failed")
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	stdin := createTempStdin(t, `[{"number": 1, "title": "Issue 1", "state": "open"}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{}

	// ACT
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Should have tried targeted first, then fallen back
	if mock.getProjectItemsByIssuesCalls != 1 {
		t.Errorf("Expected 1 GetProjectItemsByIssues call, got %d", mock.getProjectItemsByIssuesCalls)
	}
	if mock.getProjectItemsCalls != 1 {
		t.Errorf("Expected 1 GetProjectItems call (fallback), got %d", mock.getProjectItemsCalls)
	}
}

func TestRunFilterWithDeps_ErrorsWhenNoRefsBuilt(t *testing.T) {
	// ARRANGE - No URL and no config repo - should error (not silently fall back)
	mock := newMockFilterClient()
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{}, // No repos configured
	}

	// No URL in input
	stdin := createTempStdin(t, `[{"number": 1, "title": "Issue 1", "state": "open"}]`)
	defer os.Remove(stdin.Name())
	defer stdin.Close()

	cmd := newFilterCommand()
	opts := &filterOptions{}

	// ACT
	err := runFilterWithDeps(cmd, opts, cfg, mock, stdin)

	// ASSERT - should return error instead of silently falling back to full scan
	if err == nil {
		t.Fatal("Expected error when no issue refs can be built")
	}
	if !strings.Contains(err.Error(), "no repositories configured") {
		t.Errorf("Expected 'no repositories configured' error, got: %v", err)
	}
	// Should NOT have called GetProjectItems (no silent fallback)
	if mock.getProjectItemsCalls != 0 {
		t.Errorf("Expected 0 GetProjectItems calls (no silent fallback), got %d", mock.getProjectItemsCalls)
	}
}

func TestParseRepoFromURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "standard URL",
			url:       "https://github.com/owner/repo/issues/123",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTP URL",
			url:       "http://github.com/owner/repo/issues/123",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "empty URL",
			url:       "",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "invalid URL",
			url:       "not-a-url",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "non-github URL",
			url:       "https://gitlab.com/owner/repo/issues/123",
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo := parseRepoFromURL(tt.url)
			if gotOwner != tt.wantOwner {
				t.Errorf("parseRepoFromURL() owner = %v, want %v", gotOwner, tt.wantOwner)
			}
			if gotRepo != tt.wantRepo {
				t.Errorf("parseRepoFromURL() repo = %v, want %v", gotRepo, tt.wantRepo)
			}
		})
	}
}

func TestBuildIssueRefsFromInput(t *testing.T) {
	tests := []struct {
		name     string
		issues   []FilterInput
		cfg      *config.Config
		wantLen  int
		wantRepo string
	}{
		{
			name: "with URL",
			issues: []FilterInput{
				{Number: 1, URL: "https://github.com/url-owner/url-repo/issues/1"},
			},
			cfg:      &config.Config{Repositories: []string{"config-owner/config-repo"}},
			wantLen:  1,
			wantRepo: "url-repo",
		},
		{
			name: "fallback to config",
			issues: []FilterInput{
				{Number: 1, URL: ""},
			},
			cfg:      &config.Config{Repositories: []string{"config-owner/config-repo"}},
			wantLen:  1,
			wantRepo: "config-repo",
		},
		{
			name: "no repo available",
			issues: []FilterInput{
				{Number: 1, URL: ""},
			},
			cfg:      &config.Config{Repositories: []string{}},
			wantLen:  0,
			wantRepo: "",
		},
		{
			name: "skip zero number",
			issues: []FilterInput{
				{Number: 0, URL: "https://github.com/owner/repo/issues/0"},
			},
			cfg:      &config.Config{Repositories: []string{"owner/repo"}},
			wantLen:  0,
			wantRepo: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := buildIssueRefsFromInput(tt.issues, tt.cfg)
			if len(refs) != tt.wantLen {
				t.Errorf("buildIssueRefsFromInput() len = %v, want %v", len(refs), tt.wantLen)
			}
			if tt.wantLen > 0 && refs[0].Repo != tt.wantRepo {
				t.Errorf("buildIssueRefsFromInput() repo = %v, want %v", refs[0].Repo, tt.wantRepo)
			}
		})
	}
}

// ============================================================================
// hasPipedInput Tests
// ============================================================================

func TestHasPipedInput_NilFile(t *testing.T) {
	// Simulates os.Stdin.Stat() returning error (nil stat)
	result := hasPipedInput(nil)
	if result != false {
		t.Error("Expected false for nil file (graceful fallback)")
	}
}

func TestHasPipedInput_RegularFile(t *testing.T) {
	// A temp file is not a character device, so it simulates piped input
	tmpfile, err := os.CreateTemp("", "test-stdin-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	tmpfile.WriteString("test data")
	tmpfile.Seek(0, 0)

	result := hasPipedInput(tmpfile)
	if result != true {
		t.Error("Expected true for regular file (piped input)")
	}
}
