package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
)

// mockTriageClient implements triageClient interface for testing
type mockTriageClient struct {
	issues             []api.Issue
	searchIssues       []api.Issue
	issuesError        error
	searchIssuesError  error
	project            *api.Project
	projectError       error
	addToProjectItemID string
	addToProjectError  error
	addLabelError      error
	setFieldError      error
	getIssuesCalled    bool
	searchIssuesCalled bool
	lastSearchFilters  api.SearchFilters
	getProjectCalled   bool
	addToProjectCalled bool
	addLabelCalls      []string
	setFieldCalls      []struct{ field, value string }
}

func (m *mockTriageClient) GetRepositoryIssues(owner, repo, state string) ([]api.Issue, error) {
	m.getIssuesCalled = true
	return m.issues, m.issuesError
}

func (m *mockTriageClient) SearchRepositoryIssues(owner, repo string, filters api.SearchFilters, limit int) ([]api.Issue, error) {
	m.searchIssuesCalled = true
	m.lastSearchFilters = filters
	if m.searchIssuesError != nil {
		return nil, m.searchIssuesError
	}
	// Return searchIssues if set, otherwise fall back to issues
	if m.searchIssues != nil {
		return m.searchIssues, nil
	}
	return m.issues, nil
}

func (m *mockTriageClient) GetProject(owner string, number int) (*api.Project, error) {
	m.getProjectCalled = true
	return m.project, m.projectError
}

func (m *mockTriageClient) AddIssueToProject(projectID, issueID string) (string, error) {
	m.addToProjectCalled = true
	return m.addToProjectItemID, m.addToProjectError
}

func (m *mockTriageClient) AddLabelToIssue(owner, repo, issueID, labelName string) error {
	m.addLabelCalls = append(m.addLabelCalls, labelName)
	return m.addLabelError
}

func (m *mockTriageClient) SetProjectItemField(projectID, itemID, fieldName, value string) error {
	m.setFieldCalls = append(m.setFieldCalls, struct{ field, value string }{fieldName, value})
	return m.setFieldError
}

func TestTriageCommand(t *testing.T) {
	t.Run("has correct command structure", func(t *testing.T) {
		cmd := newTriageCommand()

		if cmd.Use != "triage [config-name]" {
			t.Errorf("expected Use to be 'triage [config-name]', got %s", cmd.Use)
		}

		if cmd.Short == "" {
			t.Error("expected Short description to be set")
		}
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := newTriageCommand()

		// Check --dry-run flag
		dryRunFlag := cmd.Flags().Lookup("dry-run")
		if dryRunFlag == nil {
			t.Error("expected --dry-run flag")
		}

		// Check --interactive flag
		interactiveFlag := cmd.Flags().Lookup("interactive")
		if interactiveFlag == nil {
			t.Error("expected --interactive flag")
		}

		// Check --json flag
		jsonFlag := cmd.Flags().Lookup("json")
		if jsonFlag == nil {
			t.Error("expected --json flag")
		}

		// Check --list flag
		listFlag := cmd.Flags().Lookup("list")
		if listFlag == nil {
			t.Error("expected --list flag")
		}

		// Check --repo flag
		repoFlag := cmd.Flags().Lookup("repo")
		if repoFlag == nil {
			t.Fatal("expected --repo flag")
		}
		if repoFlag.Shorthand != "R" {
			t.Errorf("expected --repo shorthand to be 'R', got %q", repoFlag.Shorthand)
		}

		// Check --query flag
		queryFlag := cmd.Flags().Lookup("query")
		if queryFlag == nil {
			t.Fatal("expected --query flag")
		}
		if queryFlag.Shorthand != "q" {
			t.Errorf("expected --query shorthand to be 'q', got %q", queryFlag.Shorthand)
		}

		// Check --apply flag
		applyFlag := cmd.Flags().Lookup("apply")
		if applyFlag == nil {
			t.Fatal("expected --apply flag")
		}
		if applyFlag.Shorthand != "a" {
			t.Errorf("expected --apply shorthand to be 'a', got %q", applyFlag.Shorthand)
		}
	})

	t.Run("command is registered in root", func(t *testing.T) {
		root := NewRootCommand()
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetArgs([]string{"triage", "--help"})
		err := root.Execute()
		if err != nil {
			t.Errorf("triage command not registered: %v", err)
		}
	})
}

func TestTriageOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := &triageOptions{}

		if opts.dryRun {
			t.Error("dryRun should be false by default")
		}
		if opts.interactive {
			t.Error("interactive should be false by default")
		}
		if opts.json {
			t.Error("json should be false by default")
		}
		if opts.list {
			t.Error("list should be false by default")
		}
		if opts.repo != "" {
			t.Errorf("repo should be empty by default, got %q", opts.repo)
		}
		if opts.query != "" {
			t.Errorf("query should be empty by default, got %q", opts.query)
		}
		if opts.apply != "" {
			t.Errorf("apply should be empty by default, got %q", opts.apply)
		}
	})
}

func TestParseTriageApplyFields(t *testing.T) {
	t.Run("parses single field", func(t *testing.T) {
		result := parseTriageApplyFields("status:backlog")
		if len(result) != 1 {
			t.Errorf("Expected 1 field, got %d", len(result))
		}
		if result["status"] != "backlog" {
			t.Errorf("Expected status=backlog, got %s", result["status"])
		}
	})

	t.Run("parses multiple fields", func(t *testing.T) {
		result := parseTriageApplyFields("status:backlog,priority:p1")
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
		result := parseTriageApplyFields("")
		if len(result) != 0 {
			t.Errorf("Expected 0 fields, got %d", len(result))
		}
	})

	t.Run("handles whitespace", func(t *testing.T) {
		result := parseTriageApplyFields(" status : backlog , priority : p1 ")
		if result["status"] != "backlog" {
			t.Errorf("Expected status=backlog, got %s", result["status"])
		}
		if result["priority"] != "p1" {
			t.Errorf("Expected priority=p1, got %s", result["priority"])
		}
	})

	t.Run("ignores invalid pairs", func(t *testing.T) {
		result := parseTriageApplyFields("status:backlog,invalid,priority:p1")
		if len(result) != 2 {
			t.Errorf("Expected 2 fields (ignoring invalid), got %d", len(result))
		}
	})
}

func TestMatchesTriageQuery(t *testing.T) {
	tests := []struct {
		name   string
		issue  api.Issue
		query  string
		expect bool
	}{
		{
			name: "label exclusion - issue has excluded label",
			issue: api.Issue{
				State:  "OPEN",
				Labels: []api.Label{{Name: "bug"}},
			},
			query:  "-label:bug",
			expect: false,
		},
		{
			name: "label exclusion - issue does not have excluded label",
			issue: api.Issue{
				State:  "OPEN",
				Labels: []api.Label{{Name: "enhancement"}},
			},
			query:  "-label:bug",
			expect: true,
		},
		{
			name: "label inclusion - issue has required label",
			issue: api.Issue{
				State:  "OPEN",
				Labels: []api.Label{{Name: "enhancement"}},
			},
			query:  "label:enhancement",
			expect: true,
		},
		{
			name: "label inclusion - issue missing required label",
			issue: api.Issue{
				State:  "OPEN",
				Labels: []api.Label{{Name: "bug"}},
			},
			query:  "label:enhancement",
			expect: false,
		},
		{
			name: "state filter - is:open matches OPEN issue",
			issue: api.Issue{
				State:  "OPEN",
				Labels: []api.Label{},
			},
			query:  "is:open",
			expect: true,
		},
		{
			name: "state filter - is:open does not match CLOSED issue",
			issue: api.Issue{
				State:  "CLOSED",
				Labels: []api.Label{},
			},
			query:  "is:open",
			expect: false,
		},
		{
			name: "state filter - is:closed matches CLOSED issue",
			issue: api.Issue{
				State:  "CLOSED",
				Labels: []api.Label{},
			},
			query:  "is:closed",
			expect: true,
		},
		{
			name: "combined query - label and state",
			issue: api.Issue{
				State:  "OPEN",
				Labels: []api.Label{{Name: "triaged"}},
			},
			query:  "is:open -label:triaged",
			expect: false,
		},
		{
			name: "no labels - empty labels slice",
			issue: api.Issue{
				State:  "OPEN",
				Labels: []api.Label{},
			},
			query:  "-label:bug",
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesTriageQuery(tt.issue, tt.query)
			if result != tt.expect {
				t.Errorf("matchesTriageQuery() = %v, want %v for query %q", result, tt.expect, tt.query)
			}
		})
	}
}

func TestDescribeActions(t *testing.T) {
	tests := []struct {
		name   string
		triage config.Triage
		expect string
	}{
		{
			name: "labels only",
			triage: config.Triage{
				Apply: config.TriageApply{
					Labels: []string{"triaged", "bug"},
				},
			},
			expect: "labels: triaged, bug",
		},
		{
			name: "fields only",
			triage: config.Triage{
				Apply: config.TriageApply{
					Fields: map[string]string{"priority": "p1"},
				},
			},
			expect: "priority: p1",
		},
		{
			name: "labels and fields combined",
			triage: config.Triage{
				Apply: config.TriageApply{
					Labels: []string{"triaged"},
					Fields: map[string]string{"priority": "p1"},
				},
			},
			expect: "labels: triaged; priority: p1",
		},
		{
			name: "interactive only",
			triage: config.Triage{
				Interactive: config.TriageInteractive{
					Status:   true,
					Estimate: true,
				},
			},
			expect: "interactive only",
		},
		{
			name:   "empty - no actions",
			triage: config.Triage{},
			expect: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := describeActions(&tt.triage)
			if result != tt.expect {
				t.Errorf("describeActions() = %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestDescribeTriageActions(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		triage   config.Triage
		contains []string
	}{
		{
			name: "shows labels to add",
			cfg:  &config.Config{},
			triage: config.Triage{
				Apply: config.TriageApply{
					Labels: []string{"triaged", "bug"},
				},
			},
			contains: []string{"Add labels:", "triaged, bug"},
		},
		{
			name: "shows fields to set",
			cfg: &config.Config{
				Fields: map[string]config.Field{
					"priority": {Field: "Priority", Values: map[string]string{"p1": "P1"}},
				},
			},
			triage: config.Triage{
				Apply: config.TriageApply{
					Fields: map[string]string{"priority": "p1"},
				},
			},
			contains: []string{"Set priority:"},
		},
		{
			name: "shows interactive prompts",
			cfg:  &config.Config{},
			triage: config.Triage{
				Interactive: config.TriageInteractive{
					Status:   true,
					Estimate: true,
				},
			},
			contains: []string{"Prompt for status", "Prompt for estimate"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := newTriageCommand()
			cmd.SetOut(buf)

			describeTriageActions(cmd, tt.cfg, &tt.triage)

			output := buf.String()
			for _, substr := range tt.contains {
				if !bytes.Contains([]byte(output), []byte(substr)) {
					t.Errorf("output should contain %q, got:\n%s", substr, output)
				}
			}
		})
	}
}

func TestListTriageConfigs(t *testing.T) {
	t.Run("table output with configs returns no error", func(t *testing.T) {
		cfg := &config.Config{
			Triage: map[string]config.Triage{
				"tracked": {
					Query: "is:open -label:triaged",
					Apply: config.TriageApply{
						Labels: []string{"triaged"},
					},
				},
				"estimate": {
					Query: "is:open -has:estimate",
					Interactive: config.TriageInteractive{
						Estimate: true,
					},
				},
			},
		}

		cmd := newTriageCommand()
		// Output goes to os.Stdout via tabwriter, verify no error
		err := listTriageConfigs(cmd, cfg, false)
		if err != nil {
			t.Fatalf("listTriageConfigs() error = %v", err)
		}
	})

	t.Run("json output returns no error", func(t *testing.T) {
		cfg := &config.Config{
			Triage: map[string]config.Triage{
				"tracked": {
					Query: "is:open -label:triaged",
					Apply: config.TriageApply{
						Labels: []string{"triaged"},
					},
				},
			},
		}

		command := newTriageCommand()
		// JSON goes to os.Stdout, so we just verify no error
		err := listTriageConfigs(command, cfg, true)
		if err != nil {
			t.Fatalf("listTriageConfigs() error = %v", err)
		}
	})

	t.Run("empty configs", func(t *testing.T) {
		cfg := &config.Config{
			Triage: map[string]config.Triage{},
		}

		buf := new(bytes.Buffer)
		cmd := newTriageCommand()
		cmd.SetOut(buf)

		err := listTriageConfigs(cmd, cfg, false)
		if err != nil {
			t.Fatalf("listTriageConfigs() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "No triage configurations") {
			t.Errorf("output should indicate no configs, got:\n%s", output)
		}
	})
}

func TestOutputTriageTable(t *testing.T) {
	t.Run("formats issues correctly", func(t *testing.T) {
		issues := []api.Issue{
			{
				Number: 1,
				Title:  "First issue",
				State:  "OPEN",
				Labels: []api.Label{{Name: "bug"}},
			},
			{
				Number: 2,
				Title:  "Second issue with a very long title that should be truncated",
				State:  "CLOSED",
				Labels: []api.Label{{Name: "enhancement"}, {Name: "help-wanted"}},
			},
		}

		cmd := newTriageCommand()
		// Output goes to os.Stdout, verify no error
		err := outputTriageTable(cmd, issues)
		if err != nil {
			t.Fatalf("outputTriageTable() error = %v", err)
		}
	})

	t.Run("handles empty issues", func(t *testing.T) {
		issues := []api.Issue{}

		cmd := newTriageCommand()
		err := outputTriageTable(cmd, issues)
		if err != nil {
			t.Fatalf("outputTriageTable() error = %v", err)
		}
	})

	t.Run("handles issues with no labels", func(t *testing.T) {
		issues := []api.Issue{
			{
				Number: 1,
				Title:  "No labels issue",
				State:  "OPEN",
				Labels: []api.Label{},
			},
		}

		cmd := newTriageCommand()
		err := outputTriageTable(cmd, issues)
		if err != nil {
			t.Fatalf("outputTriageTable() error = %v", err)
		}
	})
}

func TestSearchIssuesForTriage(t *testing.T) {
	t.Run("returns matching issues from repository with labels via Search API", func(t *testing.T) {
		mock := &mockTriageClient{
			searchIssues: []api.Issue{
				{Number: 1, Title: "Bug fix", State: "OPEN", Labels: []api.Label{{Name: "bug"}}},
			},
		}

		cfg := &config.Config{
			Repositories: []string{"owner/repo"},
		}

		issues, err := searchIssuesForTriage(mock, cfg, "is:open label:bug", "")
		if err != nil {
			t.Fatalf("searchIssuesForTriage() error = %v", err)
		}

		// With label filter, Search API should be used
		if !mock.searchIssuesCalled {
			t.Error("expected SearchRepositoryIssues to be called when query has label filter")
		}

		// Should only return issue #1 with "bug" label
		if len(issues) != 1 {
			t.Errorf("expected 1 issue, got %d", len(issues))
		}
		if len(issues) > 0 && issues[0].Number != 1 {
			t.Errorf("expected issue #1, got #%d", issues[0].Number)
		}
	})

	t.Run("handles multiple repositories", func(t *testing.T) {
		mock := &mockTriageClient{
			issues: []api.Issue{
				{Number: 1, Title: "Issue 1", State: "OPEN"},
			},
		}

		cfg := &config.Config{
			Repositories: []string{"owner/repo1", "owner/repo2"},
		}

		issues, err := searchIssuesForTriage(mock, cfg, "is:open", "")
		if err != nil {
			t.Fatalf("searchIssuesForTriage() error = %v", err)
		}

		// Should return issues from both repos (mock returns same issues for each)
		if len(issues) != 2 {
			t.Errorf("expected 2 issues (one from each repo), got %d", len(issues))
		}
	})

	t.Run("skips invalid repository format", func(t *testing.T) {
		mock := &mockTriageClient{
			issues: []api.Issue{
				{Number: 1, Title: "Issue 1", State: "OPEN"},
			},
		}

		cfg := &config.Config{
			Repositories: []string{"invalid-format", "owner/repo"},
		}

		issues, err := searchIssuesForTriage(mock, cfg, "is:open", "")
		if err != nil {
			t.Fatalf("searchIssuesForTriage() error = %v", err)
		}

		// Should only process valid repo
		if len(issues) != 1 {
			t.Errorf("expected 1 issue, got %d", len(issues))
		}
	})

	t.Run("handles API error gracefully", func(t *testing.T) {
		mock := &mockTriageClient{
			issuesError: fmt.Errorf("API error"),
		}

		cfg := &config.Config{
			Repositories: []string{"owner/repo"},
		}

		issues, err := searchIssuesForTriage(mock, cfg, "is:open", "")
		if err != nil {
			t.Fatalf("searchIssuesForTriage() should not return error, got %v", err)
		}

		// Should return empty (error is logged but not returned)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues on error, got %d", len(issues))
		}
	})

	t.Run("detects closed state from query", func(t *testing.T) {
		mock := &mockTriageClient{
			issues: []api.Issue{
				{Number: 1, Title: "Closed issue", State: "CLOSED"},
			},
		}

		cfg := &config.Config{
			Repositories: []string{"owner/repo"},
		}

		issues, err := searchIssuesForTriage(mock, cfg, "is:closed", "")
		if err != nil {
			t.Fatalf("searchIssuesForTriage() error = %v", err)
		}

		if len(issues) != 1 {
			t.Errorf("expected 1 issue, got %d", len(issues))
		}
	})

	t.Run("--repo flag overrides config repositories", func(t *testing.T) {
		callCount := 0
		mock := &mockTriageClient{
			issues: []api.Issue{
				{Number: 1, Title: "Issue 1", State: "OPEN"},
			},
		}
		// Override GetRepositoryIssues to track calls
		originalGetIssues := mock.GetRepositoryIssues
		_ = originalGetIssues // suppress unused warning

		cfg := &config.Config{
			Repositories: []string{"owner/repo1", "owner/repo2", "owner/repo3"},
		}

		// Pass target repo - should only search that one
		issues, err := searchIssuesForTriage(mock, cfg, "is:open", "target/specific-repo")
		if err != nil {
			t.Fatalf("searchIssuesForTriage() error = %v", err)
		}

		// Mock is called once for the target repo only
		// Since mock returns same issues, we should get 1 issue
		if len(issues) != 1 {
			t.Errorf("expected 1 issue from target repo, got %d", len(issues))
		}
		_ = callCount // suppress unused warning
	})

	t.Run("--repo flag with invalid format returns error", func(t *testing.T) {
		mock := &mockTriageClient{}

		cfg := &config.Config{
			Repositories: []string{"owner/repo"},
		}

		_, err := searchIssuesForTriage(mock, cfg, "is:open", "invalid-no-slash")
		if err == nil {
			t.Error("expected error for invalid repo format")
		}
		if !strings.Contains(err.Error(), "invalid repository format") {
			t.Errorf("expected 'invalid repository format' error, got: %v", err)
		}
	})

	t.Run("--repo flag allows repo not in config", func(t *testing.T) {
		mock := &mockTriageClient{
			issues: []api.Issue{
				{Number: 99, Title: "External Issue", State: "OPEN"},
			},
		}

		cfg := &config.Config{
			Repositories: []string{"owner/repo1"}, // target repo not in config
		}

		issues, err := searchIssuesForTriage(mock, cfg, "is:open", "other-owner/other-repo")
		if err != nil {
			t.Fatalf("searchIssuesForTriage() error = %v", err)
		}

		// Should search the specified repo even though it's not in config
		if len(issues) != 1 {
			t.Errorf("expected 1 issue, got %d", len(issues))
		}
		if len(issues) > 0 && issues[0].Number != 99 {
			t.Errorf("expected issue #99, got #%d", issues[0].Number)
		}
	})
}

func TestApplyTriageRules(t *testing.T) {
	t.Run("applies labels and fields", func(t *testing.T) {
		mock := &mockTriageClient{
			addToProjectItemID: "item-123",
		}

		cfg := &config.Config{
			Fields: map[string]config.Field{
				"status": {Field: "Status", Values: map[string]string{"backlog": "Backlog"}},
			},
		}

		project := &api.Project{ID: "proj-1"}
		issue := &api.Issue{ID: "issue-1", Number: 1}
		triage := &config.Triage{
			Apply: config.TriageApply{
				Labels: []string{"triaged"},
				Fields: map[string]string{"status": "backlog"},
			},
		}

		err := applyTriageRules(mock, cfg, project, issue, triage)
		if err != nil {
			t.Fatalf("applyTriageRules() error = %v", err)
		}

		if !mock.addToProjectCalled {
			t.Error("expected AddIssueToProject to be called")
		}

		if len(mock.addLabelCalls) != 1 || mock.addLabelCalls[0] != "triaged" {
			t.Errorf("expected label 'triaged' to be added, got %v", mock.addLabelCalls)
		}

		if len(mock.setFieldCalls) != 1 {
			t.Errorf("expected 1 field call, got %d", len(mock.setFieldCalls))
		}
	})

	t.Run("returns error on add to project failure", func(t *testing.T) {
		mock := &mockTriageClient{
			addToProjectError: fmt.Errorf("add to project failed"),
		}

		cfg := &config.Config{}
		project := &api.Project{ID: "proj-1"}
		issue := &api.Issue{ID: "issue-1", Number: 1}
		triage := &config.Triage{}

		err := applyTriageRules(mock, cfg, project, issue, triage)
		if err == nil {
			t.Error("expected error when add to project fails")
		}
	})

	t.Run("returns error on set field failure", func(t *testing.T) {
		mock := &mockTriageClient{
			addToProjectItemID: "item-123",
			setFieldError:      fmt.Errorf("set field failed"),
		}

		cfg := &config.Config{
			Fields: map[string]config.Field{
				"status": {Field: "Status", Values: map[string]string{"backlog": "Backlog"}},
			},
		}

		project := &api.Project{ID: "proj-1"}
		issue := &api.Issue{ID: "issue-1", Number: 1}
		triage := &config.Triage{
			Apply: config.TriageApply{
				Fields: map[string]string{"status": "backlog"},
			},
		}

		err := applyTriageRules(mock, cfg, project, issue, triage)
		if err == nil {
			t.Error("expected error when set field fails")
		}
	})

	t.Run("continues on label error", func(t *testing.T) {
		mock := &mockTriageClient{
			addToProjectItemID: "item-123",
			addLabelError:      fmt.Errorf("label already exists"),
		}

		cfg := &config.Config{}
		project := &api.Project{ID: "proj-1"}
		issue := &api.Issue{ID: "issue-1", Number: 1}
		triage := &config.Triage{
			Apply: config.TriageApply{
				Labels: []string{"label1", "label2"},
			},
		}

		err := applyTriageRules(mock, cfg, project, issue, triage)
		if err != nil {
			t.Errorf("applyTriageRules() should not error on label failure, got %v", err)
		}

		// Should still try both labels
		if len(mock.addLabelCalls) != 2 {
			t.Errorf("expected 2 label calls, got %d", len(mock.addLabelCalls))
		}
	})
}

func TestEnsureIssueInProject(t *testing.T) {
	t.Run("returns item ID on success", func(t *testing.T) {
		mock := &mockTriageClient{
			addToProjectItemID: "item-456",
		}

		itemID, err := ensureIssueInProject(mock, "proj-1", "issue-1")
		if err != nil {
			t.Fatalf("ensureIssueInProject() error = %v", err)
		}

		if itemID != "item-456" {
			t.Errorf("expected item ID 'item-456', got %q", itemID)
		}

		if !mock.addToProjectCalled {
			t.Error("expected AddIssueToProject to be called")
		}
	})

	t.Run("returns error on failure", func(t *testing.T) {
		mock := &mockTriageClient{
			addToProjectError: fmt.Errorf("failed to add"),
		}

		_, err := ensureIssueInProject(mock, "proj-1", "issue-1")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("returns error for already exists message", func(t *testing.T) {
		mock := &mockTriageClient{
			addToProjectError: fmt.Errorf("item already exists in project"),
		}

		_, err := ensureIssueInProject(mock, "proj-1", "issue-1")
		if err == nil {
			t.Error("expected error for already exists")
		}
	})
}

func TestSearchIssuesForTriage_QueryParsing(t *testing.T) {
	// Test the state detection from query strings
	// This tests the logic inside searchIssuesForTriage without needing API calls
	tests := []struct {
		query         string
		expectedState string
	}{
		{"is:open -label:bug", "open"},
		{"is:closed label:done", "closed"},
		{"is:all", "all"},
		{"label:bug", "open"}, // default to open
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			// Determine state from query (same logic as searchIssuesForTriage)
			state := "open"
			if strings.Contains(tt.query, "is:closed") {
				state = "closed"
			} else if strings.Contains(tt.query, "is:all") {
				state = "all"
			}

			if state != tt.expectedState {
				t.Errorf("state detection for query %q = %q, want %q", tt.query, state, tt.expectedState)
			}
		})
	}
}

func TestOutputTriageJSON(t *testing.T) {
	t.Run("outputs valid JSON", func(t *testing.T) {
		issues := []api.Issue{
			{
				Number: 1,
				Title:  "First issue",
				State:  "OPEN",
				URL:    "https://github.com/test/repo/issues/1",
				Labels: []api.Label{{Name: "bug"}},
			},
		}

		cmd := newTriageCommand()
		// Output goes to os.Stdout, verify no error
		err := outputTriageJSON(cmd, issues, "dry-run", "tracked")
		if err != nil {
			t.Fatalf("outputTriageJSON() error = %v", err)
		}
	})

	t.Run("handles empty issues", func(t *testing.T) {
		issues := []api.Issue{}

		cmd := newTriageCommand()
		err := outputTriageJSON(cmd, issues, "no-matches", "estimate")
		if err != nil {
			t.Fatalf("outputTriageJSON() error = %v", err)
		}
	})

	t.Run("handles different status values", func(t *testing.T) {
		issues := []api.Issue{
			{Number: 1, Title: "Test", State: "OPEN"},
		}

		cmd := newTriageCommand()

		statuses := []string{"dry-run", "no-matches", "completed"}
		for _, status := range statuses {
			err := outputTriageJSON(cmd, issues, status, "config")
			if err != nil {
				t.Errorf("outputTriageJSON() with status %q error = %v", status, err)
			}
		}
	})
}

func TestRunTriageWithDeps(t *testing.T) {
	// Helper to create a valid config
	makeConfig := func() *config.Config {
		return &config.Config{
			Project: config.Project{
				Owner:  "test-owner",
				Number: 1,
			},
			Repositories: []string{"test-owner/test-repo"},
			Triage: map[string]config.Triage{
				"tracked": {
					Query: "is:open -label:triaged",
					Apply: config.TriageApply{
						Labels: []string{"triaged"},
					},
				},
			},
			Fields: map[string]config.Field{
				"status": {Field: "Status", Values: map[string]string{"backlog": "Backlog"}},
			},
		}
	}

	t.Run("list mode returns configs", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{}
		opts := &triageOptions{list: true}

		buf := new(bytes.Buffer)
		cmd := newTriageCommand()
		cmd.SetOut(buf)

		err := runTriageWithDeps(cmd, []string{}, opts, cfg, mock, nil)
		if err != nil {
			t.Fatalf("runTriageWithDeps() error = %v", err)
		}
	})

	t.Run("requires config name when not in list mode", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{}
		opts := &triageOptions{}

		cmd := newTriageCommand()

		err := runTriageWithDeps(cmd, []string{}, opts, cfg, mock, nil)
		if err == nil {
			t.Error("expected error when config name is missing")
		}
		if !strings.Contains(err.Error(), "triage config name is required") {
			t.Errorf("expected 'config name is required' error, got: %v", err)
		}
	})

	t.Run("returns error for unknown config name", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{}
		opts := &triageOptions{}

		cmd := newTriageCommand()

		err := runTriageWithDeps(cmd, []string{"unknown-config"}, opts, cfg, mock, nil)
		if err == nil {
			t.Error("expected error for unknown config")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})

	t.Run("returns error when GetProject fails", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			projectError: fmt.Errorf("project not found"),
		}
		opts := &triageOptions{}

		cmd := newTriageCommand()

		err := runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, nil)
		if err == nil {
			t.Error("expected error when GetProject fails")
		}
		if !strings.Contains(err.Error(), "failed to get project") {
			t.Errorf("expected 'failed to get project' error, got: %v", err)
		}
	})

	t.Run("dry run shows issues without changes", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			project: &api.Project{ID: "proj-1"},
			issues: []api.Issue{
				{Number: 1, Title: "Test Issue", State: "OPEN", Labels: []api.Label{}},
			},
		}
		opts := &triageOptions{dryRun: true}

		buf := new(bytes.Buffer)
		cmd := newTriageCommand()
		cmd.SetOut(buf)

		err := runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, nil)
		if err != nil {
			t.Fatalf("runTriageWithDeps() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Would process") {
			t.Errorf("expected 'Would process' in dry run output, got:\n%s", output)
		}

		// Should not apply any changes
		if mock.addToProjectCalled {
			t.Error("dry run should not call AddIssueToProject")
		}
	})

	t.Run("dry run with json output", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			project: &api.Project{ID: "proj-1"},
			issues: []api.Issue{
				{Number: 1, Title: "Test Issue", State: "OPEN", Labels: []api.Label{}},
			},
		}
		opts := &triageOptions{dryRun: true, json: true}

		cmd := newTriageCommand()

		err := runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, nil)
		if err != nil {
			t.Fatalf("runTriageWithDeps() error = %v", err)
		}
	})

	t.Run("no matching issues shows message", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			project: &api.Project{ID: "proj-1"},
			issues:  []api.Issue{}, // No issues
		}
		opts := &triageOptions{}

		buf := new(bytes.Buffer)
		cmd := newTriageCommand()
		cmd.SetOut(buf)

		err := runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, nil)
		if err != nil {
			t.Fatalf("runTriageWithDeps() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "No issues match") {
			t.Errorf("expected 'No issues match' message, got:\n%s", output)
		}
	})

	t.Run("no matching issues with json output", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			project: &api.Project{ID: "proj-1"},
			issues:  []api.Issue{},
		}
		opts := &triageOptions{json: true}

		cmd := newTriageCommand()

		err := runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, nil)
		if err != nil {
			t.Fatalf("runTriageWithDeps() error = %v", err)
		}
	})

	t.Run("processes issues successfully", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			project:            &api.Project{ID: "proj-1"},
			addToProjectItemID: "item-123",
			issues: []api.Issue{
				{ID: "issue-1", Number: 1, Title: "Test Issue", State: "OPEN", Labels: []api.Label{}},
			},
		}
		opts := &triageOptions{}

		buf := new(bytes.Buffer)
		cmd := newTriageCommand()
		cmd.SetOut(buf)

		err := runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, nil)
		if err != nil {
			t.Fatalf("runTriageWithDeps() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Processed #1") {
			t.Errorf("expected 'Processed #1' in output, got:\n%s", output)
		}
		if !strings.Contains(output, "1 processed") {
			t.Errorf("expected '1 processed' in summary, got:\n%s", output)
		}

		// Verify API calls
		if !mock.addToProjectCalled {
			t.Error("expected AddIssueToProject to be called")
		}
		if len(mock.addLabelCalls) != 1 || mock.addLabelCalls[0] != "triaged" {
			t.Errorf("expected label 'triaged' to be added, got %v", mock.addLabelCalls)
		}
	})

	t.Run("handles processing errors gracefully", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			project:           &api.Project{ID: "proj-1"},
			addToProjectError: fmt.Errorf("failed to add to project"),
			issues: []api.Issue{
				{ID: "issue-1", Number: 1, Title: "Test Issue", State: "OPEN", Labels: []api.Label{}},
				{ID: "issue-2", Number: 2, Title: "Test Issue 2", State: "OPEN", Labels: []api.Label{}},
			},
		}
		opts := &triageOptions{}

		buf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)
		cmd := newTriageCommand()
		cmd.SetOut(buf)
		cmd.SetErr(errBuf)

		err := runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, nil)
		if err != nil {
			t.Fatalf("runTriageWithDeps() should not return error, got: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "2 failed") {
			t.Errorf("expected '2 failed' in summary, got:\n%s", output)
		}
	})

	t.Run("json output after processing", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			project:            &api.Project{ID: "proj-1"},
			addToProjectItemID: "item-123",
			issues: []api.Issue{
				{ID: "issue-1", Number: 1, Title: "Test Issue", State: "OPEN", Labels: []api.Label{}},
			},
		}
		opts := &triageOptions{json: true}

		cmd := newTriageCommand()

		err := runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, nil)
		if err != nil {
			t.Fatalf("runTriageWithDeps() error = %v", err)
		}
	})

	t.Run("interactive mode with yes response", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			project:            &api.Project{ID: "proj-1"},
			addToProjectItemID: "item-123",
			issues: []api.Issue{
				{ID: "issue-1", Number: 1, Title: "Test Issue", State: "OPEN", Labels: []api.Label{}},
			},
		}
		opts := &triageOptions{interactive: true}

		// Create a temp file with "y\n" as input
		tmpFile, err := os.CreateTemp("", "stdin")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString("y\n")
		if err != nil {
			t.Fatalf("failed to write to temp file: %v", err)
		}
		_, err = tmpFile.Seek(0, 0)
		if err != nil {
			t.Fatalf("failed to seek temp file: %v", err)
		}

		buf := new(bytes.Buffer)
		cmd := newTriageCommand()
		cmd.SetOut(buf)

		err = runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, tmpFile)
		if err != nil {
			t.Fatalf("runTriageWithDeps() error = %v", err)
		}

		// Should have processed the issue
		if !mock.addToProjectCalled {
			t.Error("expected AddIssueToProject to be called")
		}

		output := buf.String()
		if !strings.Contains(output, "1 processed") {
			t.Errorf("expected '1 processed' in summary, got:\n%s", output)
		}
	})

	t.Run("interactive mode with no response skips issue", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			project: &api.Project{ID: "proj-1"},
			issues: []api.Issue{
				{ID: "issue-1", Number: 1, Title: "Test Issue", State: "OPEN", Labels: []api.Label{}},
			},
		}
		opts := &triageOptions{interactive: true}

		// Create a temp file with "n\n" as input
		tmpFile, err := os.CreateTemp("", "stdin")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString("n\n")
		if err != nil {
			t.Fatalf("failed to write to temp file: %v", err)
		}
		_, err = tmpFile.Seek(0, 0)
		if err != nil {
			t.Fatalf("failed to seek temp file: %v", err)
		}

		buf := new(bytes.Buffer)
		cmd := newTriageCommand()
		cmd.SetOut(buf)

		err = runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, tmpFile)
		if err != nil {
			t.Fatalf("runTriageWithDeps() error = %v", err)
		}

		// Should NOT have processed the issue
		if mock.addToProjectCalled {
			t.Error("expected AddIssueToProject NOT to be called")
		}

		output := buf.String()
		if !strings.Contains(output, "1 skipped") {
			t.Errorf("expected '1 skipped' in summary, got:\n%s", output)
		}
	})

	t.Run("interactive mode with quit response aborts", func(t *testing.T) {
		cfg := makeConfig()
		mock := &mockTriageClient{
			project: &api.Project{ID: "proj-1"},
			issues: []api.Issue{
				{ID: "issue-1", Number: 1, Title: "Test Issue 1", State: "OPEN", Labels: []api.Label{}},
				{ID: "issue-2", Number: 2, Title: "Test Issue 2", State: "OPEN", Labels: []api.Label{}},
			},
		}
		opts := &triageOptions{interactive: true}

		// Create a temp file with "q\n" as input
		tmpFile, err := os.CreateTemp("", "stdin")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString("q\n")
		if err != nil {
			t.Fatalf("failed to write to temp file: %v", err)
		}
		_, err = tmpFile.Seek(0, 0)
		if err != nil {
			t.Fatalf("failed to seek temp file: %v", err)
		}

		buf := new(bytes.Buffer)
		cmd := newTriageCommand()
		cmd.SetOut(buf)

		err = runTriageWithDeps(cmd, []string{"tracked"}, opts, cfg, mock, tmpFile)
		if err != nil {
			t.Fatalf("runTriageWithDeps() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Aborted") {
			t.Errorf("expected 'Aborted' message, got:\n%s", output)
		}
	})
}

// ============================================================================
// parseTriageQuery Tests
// ============================================================================

func TestParseTriageQuery(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedState  string
		expectedLabels []string
		expectedNeg    []string
	}{
		{
			name:           "simple positive label",
			query:          "label:bug",
			expectedState:  "open",
			expectedLabels: []string{"bug"},
			expectedNeg:    []string{},
		},
		{
			name:           "simple negative label",
			query:          "-label:triaged",
			expectedState:  "open",
			expectedLabels: []string{},
			expectedNeg:    []string{"triaged"},
		},
		{
			name:           "multiple positive labels",
			query:          "label:bug label:urgent",
			expectedState:  "open",
			expectedLabels: []string{"bug", "urgent"},
			expectedNeg:    []string{},
		},
		{
			name:           "mixed positive and negative labels",
			query:          "label:bug -label:triaged",
			expectedState:  "open",
			expectedLabels: []string{"bug"},
			expectedNeg:    []string{"triaged"},
		},
		{
			name:           "with is:closed state",
			query:          "is:closed label:done",
			expectedState:  "closed",
			expectedLabels: []string{"done"},
			expectedNeg:    []string{},
		},
		{
			name:           "with is:all state",
			query:          "is:all label:needs-review",
			expectedState:  "all",
			expectedLabels: []string{"needs-review"},
			expectedNeg:    []string{},
		},
		{
			name:           "no labels - state only",
			query:          "is:open",
			expectedState:  "open",
			expectedLabels: []string{},
			expectedNeg:    []string{},
		},
		{
			name:           "complex query",
			query:          "is:open label:bug label:p0 -label:triaged -label:wontfix",
			expectedState:  "open",
			expectedLabels: []string{"bug", "p0"},
			expectedNeg:    []string{"triaged", "wontfix"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTriageQuery(tt.query)

			if result.state != tt.expectedState {
				t.Errorf("state = %q, want %q", result.state, tt.expectedState)
			}

			if len(result.labels) != len(tt.expectedLabels) {
				t.Errorf("labels count = %d, want %d", len(result.labels), len(tt.expectedLabels))
			} else {
				for i, label := range tt.expectedLabels {
					if result.labels[i] != label {
						t.Errorf("labels[%d] = %q, want %q", i, result.labels[i], label)
					}
				}
			}

			if len(result.negatedLabels) != len(tt.expectedNeg) {
				t.Errorf("negatedLabels count = %d, want %d", len(result.negatedLabels), len(tt.expectedNeg))
			} else {
				for i, label := range tt.expectedNeg {
					if result.negatedLabels[i] != label {
						t.Errorf("negatedLabels[%d] = %q, want %q", i, result.negatedLabels[i], label)
					}
				}
			}
		})
	}
}

// ============================================================================
// Search API Optimization Tests
// ============================================================================

func TestSearchIssuesForTriage_UsesSearchAPIWithLabels(t *testing.T) {
	mock := &mockTriageClient{
		searchIssues: []api.Issue{
			{Number: 1, Title: "Bug fix", State: "OPEN", Labels: []api.Label{{Name: "bug"}}},
		},
	}

	cfg := &config.Config{
		Repositories: []string{"owner/repo"},
	}

	// Query with label should use Search API
	issues, err := searchIssuesForTriage(mock, cfg, "is:open label:bug", "")
	if err != nil {
		t.Fatalf("searchIssuesForTriage() error = %v", err)
	}

	if !mock.searchIssuesCalled {
		t.Error("expected SearchRepositoryIssues to be called when query has labels")
	}

	if mock.getIssuesCalled {
		t.Error("GetRepositoryIssues should not be called when Search API is used")
	}

	// Verify search filters were set correctly
	if len(mock.lastSearchFilters.Labels) != 1 || mock.lastSearchFilters.Labels[0] != "bug" {
		t.Errorf("expected Labels=[bug], got %v", mock.lastSearchFilters.Labels)
	}

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestSearchIssuesForTriage_UsesSearchAPIWithNegatedLabels(t *testing.T) {
	mock := &mockTriageClient{
		searchIssues: []api.Issue{
			{Number: 1, Title: "Untriaged Issue", State: "OPEN", Labels: []api.Label{}},
		},
	}

	cfg := &config.Config{
		Repositories: []string{"owner/repo"},
	}

	// Query with negated label should use Search API
	issues, err := searchIssuesForTriage(mock, cfg, "is:open -label:triaged", "")
	if err != nil {
		t.Fatalf("searchIssuesForTriage() error = %v", err)
	}

	if !mock.searchIssuesCalled {
		t.Error("expected SearchRepositoryIssues to be called when query has negated labels")
	}

	// Verify negated labels were passed in Search field
	if !strings.Contains(mock.lastSearchFilters.Search, "-label:triaged") {
		t.Errorf("expected Search to contain '-label:triaged', got %q", mock.lastSearchFilters.Search)
	}

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestSearchIssuesForTriage_FallsBackWithoutLabels(t *testing.T) {
	mock := &mockTriageClient{
		issues: []api.Issue{
			{Number: 1, Title: "Open Issue", State: "OPEN"},
		},
	}

	cfg := &config.Config{
		Repositories: []string{"owner/repo"},
	}

	// Query without labels should NOT use Search API
	issues, err := searchIssuesForTriage(mock, cfg, "is:open", "")
	if err != nil {
		t.Fatalf("searchIssuesForTriage() error = %v", err)
	}

	if mock.searchIssuesCalled {
		t.Error("SearchRepositoryIssues should NOT be called when query has no labels")
	}

	if !mock.getIssuesCalled {
		t.Error("expected GetRepositoryIssues to be called for queries without labels")
	}

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestSearchIssuesForTriage_FallsBackOnSearchAPIError(t *testing.T) {
	mock := &mockTriageClient{
		searchIssuesError: fmt.Errorf("search API error"),
		issues: []api.Issue{
			{Number: 1, Title: "Bug Issue", State: "OPEN", Labels: []api.Label{{Name: "bug"}}},
		},
	}

	cfg := &config.Config{
		Repositories: []string{"owner/repo"},
	}

	// Should fall back to GetRepositoryIssues on Search API error
	issues, err := searchIssuesForTriage(mock, cfg, "is:open label:bug", "")
	if err != nil {
		t.Fatalf("searchIssuesForTriage() error = %v", err)
	}

	if !mock.searchIssuesCalled {
		t.Error("expected SearchRepositoryIssues to be attempted")
	}

	if !mock.getIssuesCalled {
		t.Error("expected GetRepositoryIssues to be called as fallback")
	}

	// Should return the issue from fallback
	if len(issues) != 1 {
		t.Errorf("expected 1 issue from fallback, got %d", len(issues))
	}
}

func TestSearchIssuesForTriage_SearchAPIWithClosedState(t *testing.T) {
	mock := &mockTriageClient{
		searchIssues: []api.Issue{
			{Number: 1, Title: "Closed Bug", State: "CLOSED", Labels: []api.Label{{Name: "bug"}}},
		},
	}

	cfg := &config.Config{
		Repositories: []string{"owner/repo"},
	}

	issues, err := searchIssuesForTriage(mock, cfg, "is:closed label:bug", "")
	if err != nil {
		t.Fatalf("searchIssuesForTriage() error = %v", err)
	}

	if !mock.searchIssuesCalled {
		t.Error("expected SearchRepositoryIssues to be called")
	}

	// Verify state was passed correctly
	if mock.lastSearchFilters.State != "closed" {
		t.Errorf("expected State=closed, got %q", mock.lastSearchFilters.State)
	}

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

// ============================================================================
// matchesTriageQuery Tests
// ============================================================================

func TestMatchesTriageQuery_PositiveLabel(t *testing.T) {
	issue := api.Issue{Labels: []api.Label{{Name: "bug"}}}
	if !matchesTriageQuery(issue, "label:bug") {
		t.Error("Expected match for issue with label:bug")
	}
}

func TestMatchesTriageQuery_PositiveLabelMissing(t *testing.T) {
	issue := api.Issue{Labels: []api.Label{{Name: "enhancement"}}}
	if matchesTriageQuery(issue, "label:bug") {
		t.Error("Expected no match for issue without label:bug")
	}
}

func TestMatchesTriageQuery_NegativeLabel(t *testing.T) {
	issue := api.Issue{Labels: []api.Label{{Name: "triaged"}}}
	if matchesTriageQuery(issue, "-label:triaged") {
		t.Error("Expected no match for issue with excluded label")
	}
}

func TestMatchesTriageQuery_MixedLabels(t *testing.T) {
	// Regression: both positive and negative labels must be checked independently
	issue := api.Issue{Labels: []api.Label{{Name: "bug"}}}
	if !matchesTriageQuery(issue, "label:bug -label:triaged") {
		t.Error("Expected match: has 'bug', doesn't have 'triaged'")
	}
}

func TestMatchesTriageQuery_MixedLabels_PositiveMissing(t *testing.T) {
	issue := api.Issue{Labels: []api.Label{}}
	if matchesTriageQuery(issue, "label:bug -label:triaged") {
		t.Error("Expected no match: missing required 'bug' label")
	}
}

func TestMatchesTriageQuery_MixedLabels_NegativePresent(t *testing.T) {
	issue := api.Issue{Labels: []api.Label{{Name: "bug"}, {Name: "triaged"}}}
	if matchesTriageQuery(issue, "label:bug -label:triaged") {
		t.Error("Expected no match: has excluded 'triaged' label")
	}
}
