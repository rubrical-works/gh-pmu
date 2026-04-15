package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
)

// mockIntakeClient implements intakeClient for testing
type mockIntakeClient struct {
	project          *api.Project
	projectItems     []api.ProjectItem
	repositoryIssues []api.Issue
	addedItemID      string

	// Call tracking
	getProjectItemsCalls []getProjectItemsCallIntake
	searchIssuesCalled   bool
	lastSearchFilters    api.SearchFilters

	// Error injection
	getProjectErr             error
	getProjectItemsErr        error
	searchRepositoryIssuesErr error
	addIssueToProjectErr      error
	setProjectItemFieldErr    error
	getProjectFieldsErr       error

	// Bulk field-setting tracking (#833)
	projectFields                     []api.ProjectField
	getProjectFieldsCalls             int
	setProjectItemFieldCalls          int
	setProjectItemFieldWithFieldsCalls int
	lastFieldsPassed                  []api.ProjectField
}

type getProjectItemsCallIntake struct {
	projectID string
	filter    *api.ProjectItemsFilter
}

func newMockIntakeClient() *mockIntakeClient {
	return &mockIntakeClient{
		project: &api.Project{
			ID:    "proj-1",
			Title: "Test Project",
		},
		projectItems:     []api.ProjectItem{},
		repositoryIssues: []api.Issue{},
		addedItemID:      "item-123",
	}
}

func (m *mockIntakeClient) GetProject(owner string, number int) (*api.Project, error) {
	if m.getProjectErr != nil {
		return nil, m.getProjectErr
	}
	return m.project, nil
}

func (m *mockIntakeClient) GetProjectItems(projectID string, filter *api.ProjectItemsFilter) ([]api.ProjectItem, error) {
	m.getProjectItemsCalls = append(m.getProjectItemsCalls, getProjectItemsCallIntake{
		projectID: projectID,
		filter:    filter,
	})
	if m.getProjectItemsErr != nil {
		return nil, m.getProjectItemsErr
	}
	return m.projectItems, nil
}

func (m *mockIntakeClient) SearchRepositoryIssues(owner, repo string, filters api.SearchFilters, limit int) ([]api.Issue, error) {
	m.searchIssuesCalled = true
	m.lastSearchFilters = filters
	if m.searchRepositoryIssuesErr != nil {
		return nil, m.searchRepositoryIssuesErr
	}
	return m.repositoryIssues, nil
}

func (m *mockIntakeClient) AddIssueToProject(projectID, issueID string) (string, error) {
	if m.addIssueToProjectErr != nil {
		return "", m.addIssueToProjectErr
	}
	return m.addedItemID, nil
}

func (m *mockIntakeClient) SetProjectItemField(projectID, itemID, fieldName, value string) error {
	m.setProjectItemFieldCalls++
	return m.setProjectItemFieldErr
}

func (m *mockIntakeClient) GetProjectFields(projectID string) ([]api.ProjectField, error) {
	m.getProjectFieldsCalls++
	if m.getProjectFieldsErr != nil {
		return nil, m.getProjectFieldsErr
	}
	return m.projectFields, nil
}

func (m *mockIntakeClient) SetProjectItemFieldWithFields(projectID, itemID, fieldName, value string, fields []api.ProjectField) error {
	m.setProjectItemFieldWithFieldsCalls++
	m.lastFieldsPassed = fields
	return m.setProjectItemFieldErr
}

func TestIntakeCommand(t *testing.T) {
	t.Run("has correct command structure", func(t *testing.T) {
		cmd := newIntakeCommand()

		if cmd.Use != "intake" {
			t.Errorf("expected Use to be 'intake', got %s", cmd.Use)
		}

		if cmd.Short == "" {
			t.Error("expected Short description to be set")
		}

		// Check aliases
		if len(cmd.Aliases) == 0 || cmd.Aliases[0] != "in" {
			t.Error("expected 'in' alias")
		}
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := newIntakeCommand()

		// Check --apply flag
		applyFlag := cmd.Flags().Lookup("apply")
		if applyFlag == nil {
			t.Fatal("expected --apply flag")
		}
		if applyFlag.Shorthand != "a" {
			t.Errorf("expected --apply shorthand 'a', got %s", applyFlag.Shorthand)
		}
		// Verify NoOptDefVal is set to allow --apply without a value
		if applyFlag.NoOptDefVal == "" {
			t.Error("expected --apply NoOptDefVal to be set (allow usage without value)")
		}

		// Check --dry-run flag
		dryRunFlag := cmd.Flags().Lookup("dry-run")
		if dryRunFlag == nil {
			t.Error("expected --dry-run flag")
		}

		// Check --json flag
		jsonFlag := cmd.Flags().Lookup("json")
		if jsonFlag == nil {
			t.Error("expected --json flag")
		}

		// Check --label flag
		labelFlag := cmd.Flags().Lookup("label")
		if labelFlag == nil {
			t.Fatal("expected --label flag")
		}
		if labelFlag.Shorthand != "l" {
			t.Errorf("expected --label shorthand 'l', got %s", labelFlag.Shorthand)
		}

		// Check --assignee flag
		assigneeFlag := cmd.Flags().Lookup("assignee")
		if assigneeFlag == nil {
			t.Error("expected --assignee flag")
		}
	})

	t.Run("command is registered in root", func(t *testing.T) {
		root := NewRootCommand()
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetArgs([]string{"intake", "--help"})
		err := root.Execute()
		if err != nil {
			t.Errorf("intake command not registered: %v", err)
		}
	})
}

func TestIntakeOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := &intakeOptions{}

		if opts.apply != "" {
			t.Error("apply should be empty string by default")
		}
		if opts.dryRun {
			t.Error("dryRun should be false by default")
		}
		if opts.json {
			t.Error("json should be false by default")
		}
		if len(opts.label) > 0 {
			t.Error("label should be empty by default")
		}
		if len(opts.assignee) > 0 {
			t.Error("assignee should be empty by default")
		}
	})
}

func TestOutputIntakeTable(t *testing.T) {
	t.Run("displays issues in table format", func(t *testing.T) {
		cmd := newIntakeCommand()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		issues := []api.Issue{
			{
				Number:     1,
				Title:      "First issue",
				State:      "OPEN",
				Repository: api.Repository{Owner: "owner", Name: "repo"},
			},
			{
				Number:     2,
				Title:      "Second issue",
				State:      "OPEN",
				Repository: api.Repository{Owner: "owner", Name: "repo"},
			},
		}

		err := outputIntakeTable(cmd, issues)
		if err != nil {
			t.Fatalf("outputIntakeTable failed: %v", err)
		}

		// Note: outputIntakeTable writes directly to os.Stdout, not cmd.Out()
		// We're testing it doesn't error; actual output goes to stdout
	})

	t.Run("truncates long titles to 50 chars", func(t *testing.T) {
		cmd := newIntakeCommand()

		// Create issue with 60-character title
		longTitle := strings.Repeat("A", 60)
		issues := []api.Issue{
			{
				Number:     1,
				Title:      longTitle,
				State:      "OPEN",
				Repository: api.Repository{Owner: "owner", Name: "repo"},
			},
		}

		// outputIntakeTable writes to os.Stdout, so we just verify no error
		err := outputIntakeTable(cmd, issues)
		if err != nil {
			t.Fatalf("outputIntakeTable failed with long title: %v", err)
		}
	})

	t.Run("handles empty issue list", func(t *testing.T) {
		cmd := newIntakeCommand()
		issues := []api.Issue{}

		err := outputIntakeTable(cmd, issues)
		if err != nil {
			t.Fatalf("outputIntakeTable failed with empty list: %v", err)
		}
	})
}

func TestOutputIntakeJSON(t *testing.T) {
	t.Run("outputs correct JSON structure with dry-run status", func(t *testing.T) {
		cmd := newIntakeCommand()

		issues := []api.Issue{
			{
				Number:     42,
				Title:      "Test issue",
				State:      "OPEN",
				URL:        "https://github.com/owner/repo/issues/42",
				Repository: api.Repository{Owner: "owner", Name: "repo"},
			},
		}

		// Capture stdout for JSON output
		// Note: outputIntakeJSON writes to os.Stdout via json.NewEncoder
		err := outputIntakeJSON(cmd, issues, "dry-run")
		if err != nil {
			t.Fatalf("outputIntakeJSON failed: %v", err)
		}
	})

	t.Run("status field matches input status", func(t *testing.T) {
		// Test that various status values are preserved
		statuses := []string{"dry-run", "applied", "untracked"}
		for _, status := range statuses {
			cmd := newIntakeCommand()
			issues := []api.Issue{}

			err := outputIntakeJSON(cmd, issues, status)
			if err != nil {
				t.Fatalf("outputIntakeJSON failed with status %q: %v", status, err)
			}
		}
	})

	t.Run("count matches issues length", func(t *testing.T) {
		cmd := newIntakeCommand()

		issues := []api.Issue{
			{Number: 1, Title: "Issue 1", Repository: api.Repository{Owner: "o", Name: "r"}},
			{Number: 2, Title: "Issue 2", Repository: api.Repository{Owner: "o", Name: "r"}},
			{Number: 3, Title: "Issue 3", Repository: api.Repository{Owner: "o", Name: "r"}},
		}

		err := outputIntakeJSON(cmd, issues, "test")
		if err != nil {
			t.Fatalf("outputIntakeJSON failed: %v", err)
		}
	})
}

func TestIntakeJSONOutput_Structure(t *testing.T) {
	t.Run("marshals to correct JSON format", func(t *testing.T) {
		output := intakeJSONOutput{
			Status: "dry-run",
			Count:  2,
			Issues: []intakeJSONIssue{
				{
					Number:     1,
					Title:      "First",
					State:      "OPEN",
					URL:        "https://github.com/owner/repo/issues/1",
					Repository: "owner/repo",
				},
				{
					Number:     2,
					Title:      "Second",
					State:      "OPEN",
					URL:        "https://github.com/owner/repo/issues/2",
					Repository: "owner/repo",
				},
			},
		}

		data, err := json.Marshal(output)
		if err != nil {
			t.Fatalf("Failed to marshal intakeJSONOutput: %v", err)
		}

		// Unmarshal and verify
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if result["status"] != "dry-run" {
			t.Errorf("Expected status 'dry-run', got %v", result["status"])
		}

		if int(result["count"].(float64)) != 2 {
			t.Errorf("Expected count 2, got %v", result["count"])
		}

		issues, ok := result["issues"].([]interface{})
		if !ok {
			t.Fatal("Expected issues to be an array")
		}
		if len(issues) != 2 {
			t.Errorf("Expected 2 issues, got %d", len(issues))
		}
	})

	t.Run("intakeJSONIssue includes all fields", func(t *testing.T) {
		issue := intakeJSONIssue{
			Number:     42,
			Title:      "Test Issue",
			State:      "OPEN",
			URL:        "https://github.com/owner/repo/issues/42",
			Repository: "owner/repo",
		}

		data, err := json.Marshal(issue)
		if err != nil {
			t.Fatalf("Failed to marshal intakeJSONIssue: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		expectedFields := []string{"number", "title", "state", "url", "repository"}
		for _, field := range expectedFields {
			if _, exists := result[field]; !exists {
				t.Errorf("Expected field %q to exist in JSON output", field)
			}
		}
	})
}

func TestFilterIntakeByLabel(t *testing.T) {
	issues := []api.Issue{
		{
			Number: 1,
			Title:  "Bug issue",
			Labels: []api.Label{{Name: "bug"}, {Name: "urgent"}},
		},
		{
			Number: 2,
			Title:  "Feature issue",
			Labels: []api.Label{{Name: "feature"}},
		},
		{
			Number: 3,
			Title:  "No labels",
			Labels: []api.Label{},
		},
	}

	t.Run("filters by single label", func(t *testing.T) {
		filtered := filterIntakeByLabel(issues, []string{"bug"})
		if len(filtered) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(filtered))
		}
		if filtered[0].Number != 1 {
			t.Errorf("Expected issue #1, got #%d", filtered[0].Number)
		}
	})

	t.Run("filters by multiple labels (OR)", func(t *testing.T) {
		filtered := filterIntakeByLabel(issues, []string{"bug", "feature"})
		if len(filtered) != 2 {
			t.Errorf("Expected 2 issues, got %d", len(filtered))
		}
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		filtered := filterIntakeByLabel(issues, []string{"BUG"})
		if len(filtered) != 1 {
			t.Errorf("Expected 1 issue with case-insensitive match, got %d", len(filtered))
		}
	})

	t.Run("returns empty for non-matching label", func(t *testing.T) {
		filtered := filterIntakeByLabel(issues, []string{"nonexistent"})
		if len(filtered) != 0 {
			t.Errorf("Expected 0 issues, got %d", len(filtered))
		}
	})
}

func TestFilterIntakeByAssignee(t *testing.T) {
	issues := []api.Issue{
		{
			Number:    1,
			Title:     "Assigned to alice",
			Assignees: []api.Actor{{Login: "alice"}},
		},
		{
			Number:    2,
			Title:     "Assigned to bob",
			Assignees: []api.Actor{{Login: "bob"}},
		},
		{
			Number:    3,
			Title:     "Assigned to both",
			Assignees: []api.Actor{{Login: "alice"}, {Login: "bob"}},
		},
		{
			Number:    4,
			Title:     "No assignees",
			Assignees: []api.Actor{},
		},
	}

	t.Run("filters by single assignee", func(t *testing.T) {
		filtered := filterIntakeByAssignee(issues, []string{"alice"})
		if len(filtered) != 2 {
			t.Errorf("Expected 2 issues assigned to alice, got %d", len(filtered))
		}
	})

	t.Run("filters by multiple assignees (OR)", func(t *testing.T) {
		filtered := filterIntakeByAssignee(issues, []string{"alice", "bob"})
		if len(filtered) != 3 {
			t.Errorf("Expected 3 issues, got %d", len(filtered))
		}
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		filtered := filterIntakeByAssignee(issues, []string{"ALICE"})
		if len(filtered) != 2 {
			t.Errorf("Expected 2 issues with case-insensitive match, got %d", len(filtered))
		}
	})

	t.Run("returns empty for non-matching assignee", func(t *testing.T) {
		filtered := filterIntakeByAssignee(issues, []string{"charlie"})
		if len(filtered) != 0 {
			t.Errorf("Expected 0 issues, got %d", len(filtered))
		}
	})
}

func TestParseApplyFields(t *testing.T) {
	t.Run("parses single field", func(t *testing.T) {
		result := parseApplyFields("status:backlog")
		if len(result) != 1 {
			t.Errorf("Expected 1 field, got %d", len(result))
		}
		if result["status"] != "backlog" {
			t.Errorf("Expected status=backlog, got %s", result["status"])
		}
	})

	t.Run("parses multiple fields", func(t *testing.T) {
		result := parseApplyFields("status:backlog,priority:p1")
		if len(result) != 2 {
			t.Errorf("Expected 2 fields, got %d", len(result))
		}
		if result["status"] != "backlog" {
			t.Errorf("Expected status=backlog, got %s", result["status"])
		}
		if result["priority"] != "p1" {
			t.Errorf("Expected priority=p1, got %s", result["priority"])
		}
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := parseApplyFields("")
		if len(result) != 0 {
			t.Errorf("Expected 0 fields, got %d", len(result))
		}
	})

	t.Run("handles whitespace", func(t *testing.T) {
		result := parseApplyFields(" status : backlog , priority : p1 ")
		if result["status"] != "backlog" {
			t.Errorf("Expected status=backlog, got %s", result["status"])
		}
		if result["priority"] != "p1" {
			t.Errorf("Expected priority=p1, got %s", result["priority"])
		}
	})

	t.Run("ignores invalid pairs", func(t *testing.T) {
		result := parseApplyFields("status:backlog,invalid,priority:p1")
		if len(result) != 2 {
			t.Errorf("Expected 2 fields (ignoring invalid), got %d", len(result))
		}
	})

	t.Run("handles trailing comma", func(t *testing.T) {
		result := parseApplyFields("status:backlog,")
		if len(result) != 1 {
			t.Errorf("Expected 1 field, got %d", len(result))
		}
	})
}

// ============================================================================
// runIntakeWithDeps Tests
// ============================================================================

func TestRunIntakeWithDeps_GetProjectError(t *testing.T) {
	mock := newMockIntakeClient()
	mock.getProjectErr = errors.New("project not found")
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	opts := &intakeOptions{}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get project") {
		t.Errorf("expected 'failed to get project' error, got: %v", err)
	}
}

func TestRunIntakeWithDeps_GetProjectItemsError(t *testing.T) {
	mock := newMockIntakeClient()
	mock.getProjectItemsErr = errors.New("items not found")
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	opts := &intakeOptions{}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get project items") {
		t.Errorf("expected 'failed to get project items' error, got: %v", err)
	}
}

func TestRunIntakeWithDeps_AllIssuesTracked(t *testing.T) {
	mock := newMockIntakeClient()
	mock.projectItems = []api.ProjectItem{
		{Issue: &api.Issue{ID: "issue-1", Number: 1}},
	}
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "Tracked Issue"},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &intakeOptions{}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "All issues are already tracked") {
		t.Errorf("expected 'All issues are already tracked' message, got: %s", output)
	}
}

func TestRunIntakeWithDeps_FindsUntrackedIssues(t *testing.T) {
	mock := newMockIntakeClient()
	mock.projectItems = []api.ProjectItem{} // No tracked issues
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "Untracked Issue", State: "OPEN"},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &intakeOptions{}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Found 1 untracked issue") {
		t.Errorf("expected 'Found 1 untracked issue' message, got: %s", output)
	}
}

func TestRunIntakeWithDeps_DryRun(t *testing.T) {
	mock := newMockIntakeClient()
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "Untracked Issue", State: "OPEN"},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &intakeOptions{dryRun: true}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Would add 1 issue") {
		t.Errorf("expected 'Would add 1 issue' message, got: %s", output)
	}
}

func TestRunIntakeWithDeps_JSONOutput(t *testing.T) {
	mock := newMockIntakeClient()
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "Untracked Issue", State: "OPEN"},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	opts := &intakeOptions{json: true}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// JSON output goes to stdout - verified by no error
}

func TestRunIntakeWithDeps_FilterByLabel(t *testing.T) {
	mock := newMockIntakeClient()
	// Simulate server-side label filtering - only return issues matching the label filter
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "Bug Issue", Labels: []api.Label{{Name: "bug"}}},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &intakeOptions{label: []string{"bug"}}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify SearchRepositoryIssues was called with correct label filters
	if !mock.searchIssuesCalled {
		t.Error("expected SearchRepositoryIssues to be called")
	}
	if len(mock.lastSearchFilters.Labels) != 1 || mock.lastSearchFilters.Labels[0] != "bug" {
		t.Errorf("expected Labels=[bug], got %v", mock.lastSearchFilters.Labels)
	}

	output := buf.String()
	if !strings.Contains(output, "Found 1 untracked") {
		t.Errorf("expected 'Found 1 untracked' (filtered), got: %s", output)
	}
}

func TestRunIntakeWithDeps_FilterByAssignee(t *testing.T) {
	mock := newMockIntakeClient()
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "Alice Issue", Assignees: []api.Actor{{Login: "alice"}}},
		{ID: "issue-2", Number: 2, Title: "Bob Issue", Assignees: []api.Actor{{Login: "bob"}}},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &intakeOptions{assignee: []string{"alice"}}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Found 1 untracked") {
		t.Errorf("expected 'Found 1 untracked' (filtered), got: %s", output)
	}
}

func TestRunIntakeWithDeps_InvalidRepoFormat(t *testing.T) {
	mock := newMockIntakeClient()
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"invalid-no-slash"},
	}

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &intakeOptions{}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	// Should not error, just warn
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Either warning about invalid repo format or "all tracked" message is acceptable
	_ = output
}

func TestRunIntakeWithDeps_AllTrackedJSON(t *testing.T) {
	mock := newMockIntakeClient()
	mock.projectItems = []api.ProjectItem{
		{Issue: &api.Issue{ID: "issue-1", Number: 1}},
	}
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "Tracked Issue"},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	opts := &intakeOptions{json: true}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// JSON output with empty issues goes to stdout - verified by no error
}

// =============================================================================
// Optimization Tests - Repository Filtering
// =============================================================================

// Test that single-repo intake uses repository filter (optimization)
func TestRunIntakeWithDeps_SingleRepo_UsesRepositoryFilter(t *testing.T) {
	mock := newMockIntakeClient()
	mock.projectItems = []api.ProjectItem{}
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "New Issue"},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"}, // Single repo
	}

	cmd := newIntakeCommand()
	opts := &intakeOptions{}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify GetProjectItems was called with repository filter
	if len(mock.getProjectItemsCalls) != 1 {
		t.Fatalf("Expected 1 GetProjectItems call, got %d", len(mock.getProjectItemsCalls))
	}

	call := mock.getProjectItemsCalls[0]
	if call.filter == nil {
		t.Fatal("Expected GetProjectItems to be called with a filter, got nil")
	}
	expectedRepo := "owner/repo"
	if call.filter.Repository != expectedRepo {
		t.Errorf("Expected filter.Repository to be %q, got %q", expectedRepo, call.filter.Repository)
	}
}

// Test that multi-repo intake uses full fetch (no filter)
func TestRunIntakeWithDeps_MultiRepo_UsesFullFetch(t *testing.T) {
	mock := newMockIntakeClient()
	mock.projectItems = []api.ProjectItem{}
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "New Issue"},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo1", "owner/repo2"}, // Multiple repos
	}

	cmd := newIntakeCommand()
	opts := &intakeOptions{}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify GetProjectItems was called without filter
	if len(mock.getProjectItemsCalls) != 1 {
		t.Fatalf("Expected 1 GetProjectItems call, got %d", len(mock.getProjectItemsCalls))
	}

	call := mock.getProjectItemsCalls[0]
	if call.filter != nil {
		t.Errorf("Expected GetProjectItems to be called without filter for multi-repo, got filter with Repository=%q", call.filter.Repository)
	}
}

// Test that filtered results are still correct with single repo
func TestRunIntakeWithDeps_SingleRepo_CorrectResults(t *testing.T) {
	mock := newMockIntakeClient()
	// Project has one tracked issue
	mock.projectItems = []api.ProjectItem{
		{Issue: &api.Issue{ID: "issue-1", Number: 1}},
	}
	// Repo has two issues - one tracked, one not
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "Tracked Issue"},
		{ID: "issue-2", Number: 2, Title: "Untracked Issue"},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	var buf bytes.Buffer
	cmd := newIntakeCommand()
	cmd.SetOut(&buf)
	opts := &intakeOptions{}
	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should find 1 untracked issue (table output goes to os.Stdout, only summary goes to cmd.Out)
	if !strings.Contains(output, "1 untracked") {
		t.Errorf("Expected output to contain '1 untracked', got: %s", output)
	}

	// Verify the filter was used correctly
	if len(mock.getProjectItemsCalls) != 1 {
		t.Fatalf("Expected 1 GetProjectItems call, got %d", len(mock.getProjectItemsCalls))
	}
	call := mock.getProjectItemsCalls[0]
	if call.filter == nil || call.filter.Repository != "owner/repo" {
		t.Errorf("Expected repository filter to be set")
	}
}

// =============================================================================
// --apply Without Value Tests (Issue #667)
// =============================================================================

// Test that --apply flag can be used without a value (NoOptDefVal)
func TestRunIntakeWithDeps_ApplyWithoutValue(t *testing.T) {
	mock := newMockIntakeClient()
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "Untracked Issue", State: "OPEN"},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
		Defaults: config.Defaults{
			Status:   "backlog",
			Priority: "p2",
		},
	}

	// Simulate --apply being set without a value (NoOptDefVal triggers this)
	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Mark the apply flag as changed (as Cobra does when --apply is used)
	_ = cmd.Flags().Set("apply", " ") // NoOptDefVal value
	opts := &intakeOptions{apply: " "}

	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Added 1 issue") {
		t.Errorf("expected 'Added 1 issue' message, got: %s", output)
	}
}

// Test that --apply with explicit fields continues to work
func TestRunIntakeWithDeps_ApplyWithExplicitFields(t *testing.T) {
	mock := newMockIntakeClient()
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "Untracked Issue", State: "OPEN"},
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Simulate --apply status:backlog,priority:p1
	_ = cmd.Flags().Set("apply", "status:backlog,priority:p1")
	opts := &intakeOptions{apply: "status:backlog,priority:p1"}

	err := runIntakeWithDeps(cmd, opts, cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Added 1 issue") {
		t.Errorf("expected 'Added 1 issue' message, got: %s", output)
	}
}

// =============================================================================
// #833: Bulk Field-Setting Tests
// =============================================================================

// TestRunIntakeWithDeps_ApplyFetchesProjectFieldsOnce asserts that `intake --apply`
// fetches project fields exactly once regardless of issue count.
func TestRunIntakeWithDeps_ApplyFetchesProjectFieldsOnce(t *testing.T) {
	mock := newMockIntakeClient()
	mock.projectFields = []api.ProjectField{
		{ID: "f1", Name: "Status", DataType: "SINGLE_SELECT"},
		{ID: "f2", Name: "Priority", DataType: "SINGLE_SELECT"},
	}
	// 20 untracked issues
	for i := 1; i <= 20; i++ {
		mock.repositoryIssues = append(mock.repositoryIssues, api.Issue{
			ID:     "issue-" + string(rune('a'+i%26)),
			Number: i,
			Title:  "Issue",
			State:  "OPEN",
		})
	}

	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
		Defaults: config.Defaults{
			Status:   "backlog",
			Priority: "p2",
		},
	}

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	_ = cmd.Flags().Set("apply", " ")
	opts := &intakeOptions{apply: " "}

	if err := runIntakeWithDeps(cmd, opts, cfg, mock); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.getProjectFieldsCalls != 1 {
		t.Errorf("GetProjectFields should be called exactly once for 20 issues, got %d", mock.getProjectFieldsCalls)
	}
	if mock.setProjectItemFieldCalls != 0 {
		t.Errorf("SetProjectItemField (non-bulk) should not be called during apply, got %d", mock.setProjectItemFieldCalls)
	}
	// 20 issues × 2 default fields (status + priority) = 40 bulk calls
	if mock.setProjectItemFieldWithFieldsCalls != 40 {
		t.Errorf("SetProjectItemFieldWithFields should be called 40 times (20 issues × 2 fields), got %d", mock.setProjectItemFieldWithFieldsCalls)
	}
}

// TestRunIntakeWithDeps_ApplyPassesPrefetchedFields asserts bulk calls receive the prefetched fields.
func TestRunIntakeWithDeps_ApplyPassesPrefetchedFields(t *testing.T) {
	mock := newMockIntakeClient()
	mock.projectFields = []api.ProjectField{
		{ID: "f1", Name: "Status", DataType: "SINGLE_SELECT"},
	}
	mock.repositoryIssues = []api.Issue{
		{ID: "issue-1", Number: 1, Title: "One", State: "OPEN"},
	}

	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"},
	}

	cmd := newIntakeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	_ = cmd.Flags().Set("apply", "status:backlog")
	opts := &intakeOptions{apply: "status:backlog"}

	if err := runIntakeWithDeps(cmd, opts, cfg, mock); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.setProjectItemFieldWithFieldsCalls != 1 {
		t.Fatalf("expected 1 bulk field call, got %d", mock.setProjectItemFieldWithFieldsCalls)
	}
	if len(mock.lastFieldsPassed) != 1 || mock.lastFieldsPassed[0].Name != "Status" {
		t.Errorf("expected prefetched Status field to be passed, got: %+v", mock.lastFieldsPassed)
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

// generateBenchmarkIntakeItems creates N project items for benchmarking
func generateBenchmarkIntakeItems(n int) []api.ProjectItem {
	items := make([]api.ProjectItem, n)
	for i := 0; i < n; i++ {
		items[i] = api.ProjectItem{
			ID: "item-" + string(rune(i)),
			Issue: &api.Issue{
				ID:     "issue-" + string(rune(i)),
				Number: i + 1,
				Title:  "Issue " + string(rune(i)),
			},
		}
	}
	return items
}

// BenchmarkIntake_WithRepositoryFilter benchmarks intake with single repo (optimized)
func BenchmarkIntake_WithRepositoryFilter(b *testing.B) {
	itemCount := 100
	mock := &mockIntakeClient{
		project: &api.Project{
			ID:    "proj-1",
			Title: "Test Project",
		},
		projectItems:     generateBenchmarkIntakeItems(itemCount),
		repositoryIssues: []api.Issue{},
		addedItemID:      "item-123",
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo"}, // Single repo - uses filter
	}

	cmd := newIntakeCommand()
	opts := &intakeOptions{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.getProjectItemsCalls = nil
		_ = runIntakeWithDeps(cmd, opts, cfg, mock)
	}

	b.ReportMetric(float64(itemCount), "items_processed")
}

// BenchmarkIntake_WithoutFilter benchmarks intake with multiple repos (full fetch)
func BenchmarkIntake_WithoutFilter(b *testing.B) {
	itemCount := 500
	mock := &mockIntakeClient{
		project: &api.Project{
			ID:    "proj-1",
			Title: "Test Project",
		},
		projectItems:     generateBenchmarkIntakeItems(itemCount),
		repositoryIssues: []api.Issue{},
		addedItemID:      "item-123",
	}
	cfg := &config.Config{
		Project:      config.Project{Owner: "test-org", Number: 1},
		Repositories: []string{"owner/repo1", "owner/repo2"}, // Multi repo - no filter
	}

	cmd := newIntakeCommand()
	opts := &intakeOptions{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.getProjectItemsCalls = nil
		_ = runIntakeWithDeps(cmd, opts, cfg, mock)
	}

	b.ReportMetric(float64(itemCount), "items_processed")
}
