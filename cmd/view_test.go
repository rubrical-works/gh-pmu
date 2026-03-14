package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/ui"
	"github.com/spf13/cobra"
)

// mockViewClient implements viewClient for testing
type mockViewClient struct {
	issue       *api.Issue
	fieldValues []api.FieldValue
	subIssues   []api.SubIssue
	parentIssue *api.Issue
	comments    []api.Comment

	// Error injection
	getIssueErr                  error
	getIssueWithProjectFieldsErr error
	getSubIssuesErr              error
	getParentIssueErr            error
	getIssueCommentsErr          error

	// Multi-issue support
	issues          map[int]*api.Issue
	fieldValuesMap  map[int][]api.FieldValue
	subIssuesMap    map[int][]api.SubIssue
	parentIssuesMap map[int]*api.Issue
	issueErrors     map[int]error
}

func newMockViewClient() *mockViewClient {
	return &mockViewClient{
		issue: &api.Issue{
			Number: 42,
			Title:  "Test Issue",
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/issues/42",
			Author: api.Actor{Login: "testuser"},
		},
		fieldValues: []api.FieldValue{},
		subIssues:   []api.SubIssue{},
	}
}

func (m *mockViewClient) GetIssue(owner, repo string, number int) (*api.Issue, error) {
	if m.getIssueErr != nil {
		return nil, m.getIssueErr
	}
	return m.issue, nil
}

func (m *mockViewClient) GetIssueWithProjectFields(owner, repo string, number int) (*api.Issue, []api.FieldValue, error) {
	if m.getIssueWithProjectFieldsErr != nil {
		return nil, nil, m.getIssueWithProjectFieldsErr
	}
	return m.issue, m.fieldValues, nil
}

func (m *mockViewClient) GetSubIssues(owner, repo string, number int) ([]api.SubIssue, error) {
	if m.getSubIssuesErr != nil {
		return nil, m.getSubIssuesErr
	}
	return m.subIssues, nil
}

func (m *mockViewClient) GetParentIssue(owner, repo string, number int) (*api.Issue, error) {
	if m.getParentIssueErr != nil {
		return nil, m.getParentIssueErr
	}
	return m.parentIssue, nil
}

func (m *mockViewClient) GetIssueComments(owner, repo string, number int) ([]api.Comment, error) {
	if m.getIssueCommentsErr != nil {
		return nil, m.getIssueCommentsErr
	}
	return m.comments, nil
}

func (m *mockViewClient) GetIssuesWithProjectFieldsBatch(owner, repo string, numbers []int) (map[int]*api.Issue, map[int][]api.FieldValue, map[int]error, error) {
	issues := make(map[int]*api.Issue)
	fvs := make(map[int][]api.FieldValue)
	errs := make(map[int]error)
	for _, n := range numbers {
		if m.issueErrors != nil {
			if e, ok := m.issueErrors[n]; ok {
				errs[n] = e
				continue
			}
		}
		if m.issues != nil {
			if iss, ok := m.issues[n]; ok {
				issues[n] = iss
			}
		}
		if m.fieldValuesMap != nil {
			if fv, ok := m.fieldValuesMap[n]; ok {
				fvs[n] = fv
			}
		}
	}
	return issues, fvs, errs, nil
}

func (m *mockViewClient) GetSubIssuesBatch(owner, repo string, numbers []int) (map[int][]api.SubIssue, error) {
	result := make(map[int][]api.SubIssue)
	for _, n := range numbers {
		if m.subIssuesMap != nil {
			if subs, ok := m.subIssuesMap[n]; ok {
				result[n] = subs
			}
		}
	}
	return result, nil
}

func (m *mockViewClient) GetParentIssueBatch(owner, repo string, numbers []int) (map[int]*api.Issue, error) {
	result := make(map[int]*api.Issue)
	for _, n := range numbers {
		if m.parentIssuesMap != nil {
			if parent, ok := m.parentIssuesMap[n]; ok {
				result[n] = parent
			}
		}
	}
	return result, nil
}

// ============================================================================
// runViewWithDeps Tests
// ============================================================================

func TestRunViewWithDeps_Success(t *testing.T) {
	mock := newMockViewClient()
	mock.issue = &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "testuser"},
	}
	mock.fieldValues = []api.FieldValue{
		{Field: "Status", Value: "In Progress"},
	}

	cmd := newViewCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &viewOptions{}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunViewWithDeps_GetIssueError(t *testing.T) {
	mock := newMockViewClient()
	mock.getIssueWithProjectFieldsErr = errors.New("issue not found")

	cmd := newViewCommand()
	opts := &viewOptions{}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get issue") {
		t.Errorf("expected 'failed to get issue' error, got: %v", err)
	}
}

func TestRunViewWithDeps_JSONOutput(t *testing.T) {
	mock := newMockViewClient()
	mock.issue = &api.Issue{
		Number: 42,
		Title:  "JSON Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "testuser"},
	}

	cmd := newViewCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &viewOptions{jsonFields: "number,title,state,body,url,author,assignees,labels,fieldValues"}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunViewWithDeps_WithSubIssues(t *testing.T) {
	mock := newMockViewClient()
	mock.issue = &api.Issue{
		Number: 42,
		Title:  "Parent Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "testuser"},
	}
	mock.subIssues = []api.SubIssue{
		{Number: 43, Title: "Sub 1", State: "CLOSED"},
		{Number: 44, Title: "Sub 2", State: "OPEN"},
	}

	cmd := newViewCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &viewOptions{}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunViewWithDeps_WithParentIssue(t *testing.T) {
	mock := newMockViewClient()
	mock.issue = &api.Issue{
		Number: 43,
		Title:  "Sub-Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/43",
		Author: api.Actor{Login: "testuser"},
	}
	mock.parentIssue = &api.Issue{
		Number: 42,
		Title:  "Parent Issue",
		URL:    "https://github.com/owner/repo/issues/42",
	}

	cmd := newViewCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &viewOptions{}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 43)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunViewWithDeps_WithComments(t *testing.T) {
	mock := newMockViewClient()
	mock.issue = &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "testuser"},
	}
	mock.comments = []api.Comment{
		{Author: "user1", Body: "Comment 1", CreatedAt: "2024-01-01T10:00:00Z"},
		{Author: "user2", Body: "Comment 2", CreatedAt: "2024-01-02T11:00:00Z"},
	}

	cmd := newViewCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &viewOptions{comments: true}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestViewCommand_Exists(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"view", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("view command should exist: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("view")) {
		t.Error("Expected help output to mention 'view'")
	}
}

func TestViewCommand_RequiresIssueNumber(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"view"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when issue number not provided")
	}
}

func TestViewCommand_HasJSONFlag(t *testing.T) {
	cmd := NewRootCommand()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("view command not found: %v", err)
	}

	flag := viewCmd.Flags().Lookup("json")
	if flag == nil {
		t.Fatal("Expected --json flag to exist")
	}
}

func TestViewCommand_HasWebFlag(t *testing.T) {
	cmd := NewRootCommand()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("view command not found: %v", err)
	}

	flag := viewCmd.Flags().Lookup("web")
	if flag == nil {
		t.Fatal("Expected --web flag to exist")
	}

	// Check shorthand
	if flag.Shorthand != "w" {
		t.Errorf("Expected --web shorthand to be 'w', got %s", flag.Shorthand)
	}
}

func TestViewCommand_HasCommentsFlag(t *testing.T) {
	cmd := NewRootCommand()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("view command not found: %v", err)
	}

	flag := viewCmd.Flags().Lookup("comments")
	if flag == nil {
		t.Fatal("Expected --comments flag to exist")
	}

	// Check shorthand
	if flag.Shorthand != "c" {
		t.Errorf("Expected --comments shorthand to be 'c', got %s", flag.Shorthand)
	}
}

func TestViewCommand_HasRepoFlag(t *testing.T) {
	cmd := NewRootCommand()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("view command not found: %v", err)
	}

	flag := viewCmd.Flags().Lookup("repo")
	if flag == nil {
		t.Fatal("Expected --repo flag to exist")
	}

	// Check shorthand
	if flag.Shorthand != "R" {
		t.Errorf("Expected --repo shorthand to be 'R', got %s", flag.Shorthand)
	}

	// Check type
	if flag.Value.Type() != "string" {
		t.Errorf("Expected --repo to be string, got %s", flag.Value.Type())
	}
}

func TestViewCommand_AcceptsIssueNumber(t *testing.T) {
	cmd := NewRootCommand()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("view command not found: %v", err)
	}

	// Verify the command accepts exactly 1 argument
	if viewCmd.Args == nil {
		t.Error("Expected Args validator to be set")
	}
}

func TestViewCommand_ParsesIssueNumber(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		wantErr bool
	}{
		{"valid number", "123", false},
		{"with hash", "#123", false},
		{"invalid string", "abc", true},
		{"negative number", "-1", true},
		{"zero", "0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseIssueNumber(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIssueNumber(%q) error = %v, wantErr %v", tt.arg, err, tt.wantErr)
			}
		})
	}
}

func TestViewCommand_ParsesIssueReference(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		wantOwner  string
		wantRepo   string
		wantNumber int
		wantErr    bool
	}{
		{"number only", "123", "", "", 123, false},
		{"with hash", "#123", "", "", 123, false},
		{"full reference", "owner/repo#123", "owner", "repo", 123, false},
		{"invalid", "invalid", "", "", 0, true},
		// URL formats
		{"https URL", "https://github.com/owner/repo/issues/123", "owner", "repo", 123, false},
		{"http URL", "http://github.com/owner/repo/issues/123", "owner", "repo", 123, false},
		{"URL with anchor", "https://github.com/owner/repo/issues/123#issuecomment-456", "owner", "repo", 123, false},
		{"invalid URL - not issues", "https://github.com/owner/repo/pulls/123", "", "", 0, true},
		{"invalid URL - too short", "https://github.com/owner", "", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, number, err := parseIssueReference(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIssueReference(%q) error = %v, wantErr %v", tt.arg, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if owner != tt.wantOwner {
					t.Errorf("parseIssueReference(%q) owner = %v, want %v", tt.arg, owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("parseIssueReference(%q) repo = %v, want %v", tt.arg, repo, tt.wantRepo)
				}
				if number != tt.wantNumber {
					t.Errorf("parseIssueReference(%q) number = %v, want %v", tt.arg, number, tt.wantNumber)
				}
			}
		})
	}
}

// Progress bar tests

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name      string
		completed int
		total     int
		width     int
		want      string
	}{
		{"empty", 0, 10, 10, "[░░░░░░░░░░]"},
		{"half", 5, 10, 10, "[█████░░░░░]"},
		{"full", 10, 10, 10, "[██████████]"},
		{"quarter", 1, 4, 8, "[██░░░░░░]"},
		{"zero total", 0, 0, 10, "[░░░░░░░░░░]"},
		{"60 percent", 3, 5, 10, "[██████░░░░]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderProgressBar(tt.completed, tt.total, tt.width)
			if got != tt.want {
				t.Errorf("renderProgressBar(%d, %d, %d) = %q, want %q",
					tt.completed, tt.total, tt.width, got, tt.want)
			}
		})
	}
}

func TestRenderProgressBar_OverflowProtection(t *testing.T) {
	// Test that completed > total doesn't overflow
	result := renderProgressBar(15, 10, 10)
	// Should cap at full
	if result != "[██████████]" {
		t.Errorf("renderProgressBar with overflow should cap at full, got %q", result)
	}
}

// ============================================================================
// outputViewTable Tests
// ============================================================================

// createViewTestCmd creates a cobra command for testing view output
func createViewTestCmd(buf *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd
}

func TestOutputViewTable_BasicIssue(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Test Issue Title",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "testuser"},
	}

	err := outputViewTable(cmd, issue, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}

	// Note: outputViewTable writes to os.Stdout, not cmd buffer
	// We verify no error occurred
}

func TestOutputViewTable_WithAssignees(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
		Assignees: []api.Actor{
			{Login: "user1"},
			{Login: "user2"},
		},
	}

	err := outputViewTable(cmd, issue, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}
}

func TestOutputViewTable_WithLabels(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
		Labels: []api.Label{
			{Name: "bug"},
			{Name: "priority:high"},
		},
	}

	err := outputViewTable(cmd, issue, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}
}

func TestOutputViewTable_WithMilestone(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number:    42,
		Title:     "Test Issue",
		State:     "OPEN",
		URL:       "https://github.com/owner/repo/issues/42",
		Author:    api.Actor{Login: "author"},
		Milestone: &api.Milestone{Title: "v1.0.0"},
	}

	err := outputViewTable(cmd, issue, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}
}

func TestOutputViewTable_WithFieldValues(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
	}

	fieldValues := []api.FieldValue{
		{Field: "Status", Value: "In Progress"},
		{Field: "Priority", Value: "High"},
	}

	err := outputViewTable(cmd, issue, fieldValues, nil, nil, nil)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}
}

func TestOutputViewTable_WithParentIssue(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Sub-Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
	}

	parentIssue := &api.Issue{
		Number: 10,
		Title:  "Parent Issue",
		URL:    "https://github.com/owner/repo/issues/10",
	}

	err := outputViewTable(cmd, issue, nil, nil, parentIssue, nil)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}
}

func TestOutputViewTable_WithSubIssues(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Parent Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
		Repository: api.Repository{
			Owner: "owner",
			Name:  "repo",
		},
	}

	subIssues := []api.SubIssue{
		{Number: 43, Title: "Sub 1", State: "CLOSED", URL: "https://github.com/owner/repo/issues/43"},
		{Number: 44, Title: "Sub 2", State: "OPEN", URL: "https://github.com/owner/repo/issues/44"},
		{Number: 45, Title: "Sub 3", State: "CLOSED", URL: "https://github.com/owner/repo/issues/45"},
	}

	err := outputViewTable(cmd, issue, nil, subIssues, nil, nil)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}
}

func TestOutputViewTable_WithCrossRepoSubIssues(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Parent Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
		Repository: api.Repository{
			Owner: "owner",
			Name:  "repo",
		},
	}

	subIssues := []api.SubIssue{
		{
			Number: 43,
			Title:  "Same Repo Sub",
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/issues/43",
			Repository: api.Repository{
				Owner: "owner",
				Name:  "repo",
			},
		},
		{
			Number: 10,
			Title:  "Cross Repo Sub",
			State:  "CLOSED",
			URL:    "https://github.com/owner/other-repo/issues/10",
			Repository: api.Repository{
				Owner: "owner",
				Name:  "other-repo",
			},
		},
	}

	err := outputViewTable(cmd, issue, nil, subIssues, nil, nil)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}
}

func TestOutputViewTable_WithBody(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
		Body:   "This is the issue body with some content.\n\nMultiple paragraphs.",
	}

	err := outputViewTable(cmd, issue, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}
}

func TestOutputViewTable_FullIssue(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number:    42,
		Title:     "Full Featured Issue",
		State:     "OPEN",
		URL:       "https://github.com/owner/repo/issues/42",
		Body:      "Issue body content",
		Author:    api.Actor{Login: "author"},
		Assignees: []api.Actor{{Login: "dev1"}, {Login: "dev2"}},
		Labels:    []api.Label{{Name: "bug"}, {Name: "urgent"}},
		Milestone: &api.Milestone{Title: "v2.0"},
		Repository: api.Repository{
			Owner: "owner",
			Name:  "repo",
		},
	}

	fieldValues := []api.FieldValue{
		{Field: "Status", Value: "In Progress"},
		{Field: "Priority", Value: "P1"},
	}

	subIssues := []api.SubIssue{
		{Number: 43, Title: "Task 1", State: "CLOSED"},
		{Number: 44, Title: "Task 2", State: "OPEN"},
	}

	parentIssue := &api.Issue{
		Number: 10,
		Title:  "Epic",
		URL:    "https://github.com/owner/repo/issues/10",
	}

	err := outputViewTable(cmd, issue, fieldValues, subIssues, parentIssue, nil)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}
}

// ============================================================================
// outputViewJSON Tests
// ============================================================================

func TestOutputViewJSON_BasicIssue(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "testuser"},
	}

	opts := &viewOptions{jsonFields: "number,title,state,body,url,author,assignees,labels,fieldValues"}
	err := outputViewJSON(cmd, opts, issue, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("outputViewJSON() error = %v", err)
	}
}

func TestOutputViewJSON_WithAllFields(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number:    42,
		Title:     "Full Issue",
		State:     "OPEN",
		Body:      "Issue description",
		URL:       "https://github.com/owner/repo/issues/42",
		Author:    api.Actor{Login: "author"},
		Assignees: []api.Actor{{Login: "dev1"}, {Login: "dev2"}},
		Labels:    []api.Label{{Name: "bug"}, {Name: "priority:high"}},
		Milestone: &api.Milestone{Title: "v1.0"},
	}

	fieldValues := []api.FieldValue{
		{Field: "Status", Value: "In Progress"},
		{Field: "Priority", Value: "High"},
	}

	opts := &viewOptions{jsonFields: "number,title,state,body,url,author,assignees,labels,milestone,fieldValues"}
	err := outputViewJSON(cmd, opts, issue, fieldValues, nil, nil, nil)
	if err != nil {
		t.Fatalf("outputViewJSON() error = %v", err)
	}
}

func TestOutputViewJSON_WithSubIssues(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Parent Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
	}

	subIssues := []api.SubIssue{
		{Number: 43, Title: "Sub 1", State: "CLOSED", URL: "https://github.com/owner/repo/issues/43"},
		{Number: 44, Title: "Sub 2", State: "OPEN", URL: "https://github.com/owner/repo/issues/44"},
		{Number: 45, Title: "Sub 3", State: "CLOSED", URL: "https://github.com/owner/repo/issues/45"},
	}

	opts := &viewOptions{jsonFields: "number,title,state,body,url,author,assignees,labels,fieldValues,subIssues,subProgress"}
	err := outputViewJSON(cmd, opts, issue, nil, subIssues, nil, nil)
	if err != nil {
		t.Fatalf("outputViewJSON() error = %v", err)
	}
}

func TestOutputViewJSON_WithParentIssue(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Sub-Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
	}

	parentIssue := &api.Issue{
		Number: 10,
		Title:  "Parent Issue",
		URL:    "https://github.com/owner/repo/issues/10",
	}

	opts := &viewOptions{jsonFields: "number,title,state,body,url,author,assignees,labels,fieldValues,parentIssue"}
	err := outputViewJSON(cmd, opts, issue, nil, nil, parentIssue, nil)
	if err != nil {
		t.Fatalf("outputViewJSON() error = %v", err)
	}
}

func TestOutputViewJSON_SubIssueProgress(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Parent Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
	}

	// 3 closed out of 5 = 60%
	subIssues := []api.SubIssue{
		{Number: 1, Title: "Task 1", State: "CLOSED"},
		{Number: 2, Title: "Task 2", State: "CLOSED"},
		{Number: 3, Title: "Task 3", State: "OPEN"},
		{Number: 4, Title: "Task 4", State: "CLOSED"},
		{Number: 5, Title: "Task 5", State: "OPEN"},
	}

	opts := &viewOptions{jsonFields: "number,title,state,body,url,author,assignees,labels,fieldValues,subIssues,subProgress"}
	err := outputViewJSON(cmd, opts, issue, nil, subIssues, nil, nil)
	if err != nil {
		t.Fatalf("outputViewJSON() error = %v", err)
	}
}

func TestOpenInBrowser(t *testing.T) {
	// Test that function exists and handles URL parameter
	// We can't actually test browser opening in unit tests
	_ = ui.OpenInBrowser
}

func TestOutputViewTable_WithComments(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
	}

	comments := []api.Comment{
		{Author: "user1", Body: "First comment", CreatedAt: "2024-01-01T10:00:00Z"},
		{Author: "user2", Body: "Second comment", CreatedAt: "2024-01-02T11:00:00Z"},
	}

	err := outputViewTable(cmd, issue, nil, nil, nil, comments)
	if err != nil {
		t.Fatalf("outputViewTable() error = %v", err)
	}
}

func TestOutputViewJSON_WithComments(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := createViewTestCmd(buf)

	issue := &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "author"},
	}

	comments := []api.Comment{
		{Author: "user1", Body: "First comment", CreatedAt: "2024-01-01T10:00:00Z"},
		{Author: "user2", Body: "Second comment", CreatedAt: "2024-01-02T11:00:00Z"},
	}

	opts := &viewOptions{jsonFields: "number,title,state,body,url,author,assignees,labels,fieldValues,comments"}
	err := outputViewJSON(cmd, opts, issue, nil, nil, nil, comments)
	if err != nil {
		t.Fatalf("outputViewJSON() error = %v", err)
	}
}

// ============================================================================
// ViewJSONOutput Structure Tests
// ============================================================================

func TestViewJSONOutput_Structure(t *testing.T) {
	output := ViewJSONOutput{
		Number:    42,
		Title:     "Test Issue",
		State:     "OPEN",
		Body:      "Issue body",
		URL:       "https://github.com/owner/repo/issues/42",
		Author:    "testuser",
		Assignees: []string{"user1", "user2"},
		Labels:    []string{"bug", "urgent"},
		Milestone: "v1.0",
		FieldValues: map[string]string{
			"Status":   "In Progress",
			"Priority": "High",
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal ViewJSONOutput: %v", err)
	}

	var parsed ViewJSONOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal ViewJSONOutput: %v", err)
	}

	if parsed.Number != 42 {
		t.Errorf("Expected Number 42, got %d", parsed.Number)
	}
	if parsed.Title != "Test Issue" {
		t.Errorf("Expected Title 'Test Issue', got %s", parsed.Title)
	}
	if len(parsed.Assignees) != 2 {
		t.Errorf("Expected 2 assignees, got %d", len(parsed.Assignees))
	}
	if parsed.FieldValues["Status"] != "In Progress" {
		t.Errorf("Expected Status 'In Progress', got %s", parsed.FieldValues["Status"])
	}
}

func TestViewJSONOutput_WithSubProgress(t *testing.T) {
	output := ViewJSONOutput{
		Number: 42,
		Title:  "Parent",
		State:  "OPEN",
		URL:    "https://example.com",
		Author: "user",
		SubIssues: []SubIssueJSON{
			{Number: 1, Title: "Sub 1", State: "CLOSED"},
			{Number: 2, Title: "Sub 2", State: "OPEN"},
		},
		SubProgress: &SubProgressJSON{
			Total:      2,
			Completed:  1,
			Percentage: 50,
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed ViewJSONOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.SubProgress == nil {
		t.Fatal("Expected SubProgress to be present")
	}
	if parsed.SubProgress.Percentage != 50 {
		t.Errorf("Expected 50%% progress, got %d%%", parsed.SubProgress.Percentage)
	}
}

func TestViewJSONOutput_WithParentIssue(t *testing.T) {
	output := ViewJSONOutput{
		Number: 42,
		Title:  "Sub-Issue",
		State:  "OPEN",
		URL:    "https://example.com",
		Author: "user",
		ParentIssue: &ParentIssueJSON{
			Number: 10,
			Title:  "Parent Issue",
			URL:    "https://example.com/10",
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed ViewJSONOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.ParentIssue == nil {
		t.Fatal("Expected ParentIssue to be present")
	}
	if parsed.ParentIssue.Number != 10 {
		t.Errorf("Expected parent number 10, got %d", parsed.ParentIssue.Number)
	}
}

func TestSubIssueJSON_Structure(t *testing.T) {
	sub := SubIssueJSON{
		Number: 43,
		Title:  "Sub-Issue Title",
		State:  "CLOSED",
		URL:    "https://github.com/owner/repo/issues/43",
	}

	data, err := json.Marshal(sub)
	if err != nil {
		t.Fatalf("Failed to marshal SubIssueJSON: %v", err)
	}

	jsonStr := string(data)
	expectedFields := []string{"number", "title", "state", "url"}
	for _, field := range expectedFields {
		if !bytes.Contains(data, []byte(field)) {
			t.Errorf("Expected JSON to contain field %q, got: %s", field, jsonStr)
		}
	}
}

func TestSubProgressJSON_Structure(t *testing.T) {
	progress := SubProgressJSON{
		Total:      10,
		Completed:  6,
		Percentage: 60,
	}

	data, err := json.Marshal(progress)
	if err != nil {
		t.Fatalf("Failed to marshal SubProgressJSON: %v", err)
	}

	var parsed SubProgressJSON
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal SubProgressJSON: %v", err)
	}

	if parsed.Total != 10 {
		t.Errorf("Expected Total 10, got %d", parsed.Total)
	}
	if parsed.Completed != 6 {
		t.Errorf("Expected Completed 6, got %d", parsed.Completed)
	}
	if parsed.Percentage != 60 {
		t.Errorf("Expected Percentage 60, got %d", parsed.Percentage)
	}
}

// ============================================================================
// Body File Tests
// ============================================================================

func TestViewCommand_HasBodyFileFlag(t *testing.T) {
	cmd := NewRootCommand()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("view command not found: %v", err)
	}

	flag := viewCmd.Flags().Lookup("body-file")
	if flag == nil {
		t.Fatal("Expected --body-file flag to exist")
	}

	// Check shorthand
	if flag.Shorthand != "b" {
		t.Errorf("Expected --body-file shorthand to be 'b', got %s", flag.Shorthand)
	}
}

func TestWriteBodyToFile_Success(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	var buf bytes.Buffer
	err := writeBodyToFile(&buf, 42, "Test body content\n\nWith multiple lines.")
	if err != nil {
		t.Fatalf("writeBodyToFile() error = %v", err)
	}

	// Verify file was created
	filePath := filepath.Join("tmp", "issue-42.md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	expected := "Test body content\n\nWith multiple lines."
	if string(content) != expected {
		t.Errorf("File content = %q, want %q", string(content), expected)
	}
}

func TestWriteBodyToFile_CreatesTmpDirectory(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Verify tmp doesn't exist
	if _, err := os.Stat("tmp"); !os.IsNotExist(err) {
		t.Fatal("tmp directory should not exist before test")
	}

	var buf bytes.Buffer
	err := writeBodyToFile(&buf, 123, "Body content")
	if err != nil {
		t.Fatalf("writeBodyToFile() error = %v", err)
	}

	// Verify tmp directory was created
	info, err := os.Stat("tmp")
	if err != nil {
		t.Fatalf("tmp directory should exist after writeBodyToFile: %v", err)
	}
	if !info.IsDir() {
		t.Error("tmp should be a directory")
	}
}

func TestWriteBodyToFile_EmptyBody(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	var buf bytes.Buffer
	err := writeBodyToFile(&buf, 99, "")
	if err != nil {
		t.Fatalf("writeBodyToFile() error = %v", err)
	}

	// Verify file was created with empty content
	filePath := filepath.Join("tmp", "issue-99.md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != "" {
		t.Errorf("File content = %q, want empty string", string(content))
	}
}

func TestRunViewWithDeps_BodyFile(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	mock := newMockViewClient()
	mock.issue = &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "testuser"},
		Body:   "This is the issue body for testing.",
	}

	cmd := newViewCommand()
	opts := &viewOptions{bodyFile: true}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created with correct content
	filePath := filepath.Join("tmp", "issue-42.md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	expected := "This is the issue body for testing."
	if string(content) != expected {
		t.Errorf("File content = %q, want %q", string(content), expected)
	}
}

// ============================================================================
// Body Stdout Tests
// ============================================================================

func TestViewCommand_HasBodyStdoutFlag(t *testing.T) {
	cmd := NewRootCommand()
	viewCmd, _, err := cmd.Find([]string{"view"})
	if err != nil {
		t.Fatalf("view command not found: %v", err)
	}

	flag := viewCmd.Flags().Lookup("body-stdout")
	if flag == nil {
		t.Fatal("Expected --body-stdout flag to exist")
	}

	// body-stdout has no shorthand (b is taken by body-file)
	if flag.Shorthand != "" {
		t.Errorf("Expected --body-stdout to have no shorthand, got %s", flag.Shorthand)
	}
}

func TestRunViewWithDeps_BodyStdout(t *testing.T) {
	mock := newMockViewClient()
	mock.issue = &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "testuser"},
		Body:   "This is the issue body for stdout testing.",
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newViewCommand()
	opts := &viewOptions{bodyStdout: true}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "This is the issue body for stdout testing."
	if buf.String() != expected {
		t.Errorf("stdout output = %q, want %q", buf.String(), expected)
	}
}

func TestRunViewWithDeps_BodyStdout_EmptyBody(t *testing.T) {
	mock := newMockViewClient()
	mock.issue = &api.Issue{
		Number: 99,
		Title:  "Empty Body Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/99",
		Author: api.Actor{Login: "testuser"},
		Body:   "",
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newViewCommand()
	opts := &viewOptions{bodyStdout: true}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 99)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "" {
		t.Errorf("stdout output = %q, want empty string", buf.String())
	}
}

// ============================================================================
// Flag Mutual Exclusivity Tests
// ============================================================================

func TestRunViewWithDeps_JSONWithBodyFileError(t *testing.T) {
	mock := newMockViewClient()
	cmd := newViewCommand()

	opts := &viewOptions{jsonFields: "title", bodyFile: true}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)

	if err == nil {
		t.Fatal("expected error for --json with --body-file")
	}
	if !strings.Contains(err.Error(), "cannot use --json with --body-file") {
		t.Errorf("expected mutual exclusivity error, got: %v", err)
	}
}

func TestRunViewWithDeps_JSONWithBodyStdoutError(t *testing.T) {
	mock := newMockViewClient()
	cmd := newViewCommand()

	opts := &viewOptions{jsonFields: "title", bodyStdout: true}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)

	if err == nil {
		t.Fatal("expected error for --json with --body-stdout")
	}
	if !strings.Contains(err.Error(), "cannot use --json with --body-stdout") {
		t.Errorf("expected mutual exclusivity error, got: %v", err)
	}
}

func TestRunViewWithDeps_JSONWithWebError(t *testing.T) {
	mock := newMockViewClient()
	cmd := newViewCommand()

	opts := &viewOptions{jsonFields: "title", web: true}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)

	if err == nil {
		t.Fatal("expected error for --json with --web")
	}
	if !strings.Contains(err.Error(), "cannot use --json with --web") {
		t.Errorf("expected mutual exclusivity error, got: %v", err)
	}
}

func TestRunViewWithDeps_JSONWithCommentsError(t *testing.T) {
	mock := newMockViewClient()
	cmd := newViewCommand()

	opts := &viewOptions{jsonFields: "title", comments: true}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)

	if err == nil {
		t.Fatal("expected error for --json with --comments")
	}
	if !strings.Contains(err.Error(), "cannot use --json with --comments") {
		t.Errorf("expected mutual exclusivity error, got: %v", err)
	}
}

func TestRunViewWithDeps_JQWithoutJSONError(t *testing.T) {
	mock := newMockViewClient()
	cmd := newViewCommand()

	opts := &viewOptions{jq: ".title"}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)

	if err == nil {
		t.Fatal("expected error for --jq without --json")
	}
	if !strings.Contains(err.Error(), "--jq requires --json") {
		t.Errorf("expected jq requires json error, got: %v", err)
	}
}

func TestRunViewWithDeps_TemplateNotSupportedError(t *testing.T) {
	mock := newMockViewClient()
	cmd := newViewCommand()

	opts := &viewOptions{template: "{{.title}}"}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)

	if err == nil {
		t.Fatal("expected error for --template")
	}
	// Should mention --template is not supported
	if !strings.Contains(err.Error(), "--template is not supported") {
		t.Errorf("expected '--template is not supported' error, got: %v", err)
	}
	// Should mention gh issue view as alternative
	if !strings.Contains(err.Error(), "gh issue view") {
		t.Errorf("expected error to mention 'gh issue view', got: %v", err)
	}
	// Should mention --jq for project fields
	if !strings.Contains(err.Error(), "--jq") {
		t.Errorf("expected error to mention '--jq', got: %v", err)
	}
}

// ============================================================================
// JSON Output with Project Fields Tests
// ============================================================================

func TestRunViewWithDeps_JSONWithProjectFields(t *testing.T) {
	mock := newMockViewClient()
	mock.issue = &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "testuser"},
	}
	mock.fieldValues = []api.FieldValue{
		{Field: "Status", Value: "In Progress"},
		{Field: "Priority", Value: "P1"},
		{Field: "Branch", Value: "release/v1.0"},
	}

	cmd := newViewCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &viewOptions{jsonFields: "fieldValues"}
	err := runViewWithDeps(cmd, opts, mock, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Note: JSON is written to stdout, not the buffer
	// We verify no error occurred and the function completed
}

func TestBuildViewJSONOutput_FieldValues(t *testing.T) {
	issue := &api.Issue{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://github.com/owner/repo/issues/42",
		Author: api.Actor{Login: "testuser"},
	}
	fieldValues := []api.FieldValue{
		{Field: "Status", Value: "In Progress"},
		{Field: "Priority", Value: "P1"},
		{Field: "Branch", Value: "release/v1.0"},
	}

	output := buildViewJSONOutput(issue, fieldValues, nil, nil, nil)

	if output.FieldValues["Status"] != "In Progress" {
		t.Errorf("expected Status 'In Progress', got %q", output.FieldValues["Status"])
	}
	if output.FieldValues["Priority"] != "P1" {
		t.Errorf("expected Priority 'P1', got %q", output.FieldValues["Priority"])
	}
	if output.FieldValues["Branch"] != "release/v1.0" {
		t.Errorf("expected Branch 'release/v1.0', got %q", output.FieldValues["Branch"])
	}
}

func TestFilterViewJSONFields_SelectsRequestedFields(t *testing.T) {
	output := ViewJSONOutput{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		Body:   "Body text",
		URL:    "https://example.com",
		Author: "testuser",
	}

	result := filterViewJSONFields(output, []string{"number", "title"})

	if result["number"] != 42 {
		t.Errorf("expected number 42, got %v", result["number"])
	}
	if result["title"] != "Test Issue" {
		t.Errorf("expected title 'Test Issue', got %v", result["title"])
	}
	if _, exists := result["body"]; exists {
		t.Error("expected body to be excluded")
	}
	if _, exists := result["state"]; exists {
		t.Error("expected state to be excluded")
	}
}

func TestParseJSONFields_CommaSeparated(t *testing.T) {
	result := parseJSONFields("number,title,state", viewAvailableFields)

	if len(result) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(result))
	}
	if result[0] != "number" || result[1] != "title" || result[2] != "state" {
		t.Errorf("unexpected fields: %v", result)
	}
}

func TestParseJSONFields_EmptyReturnsAll(t *testing.T) {
	result := parseJSONFields("", viewAvailableFields)

	if len(result) != len(viewAvailableFields) {
		t.Errorf("expected all %d fields, got %d", len(viewAvailableFields), len(result))
	}
}

// ============================================================================
// Project Field Shorthand Tests (Issue #668)
// ============================================================================

func TestViewAvailableFields_IncludesProjectFields(t *testing.T) {
	// Verify status, priority, branch are in available fields
	projectFields := []string{"status", "priority", "branch"}
	for _, field := range projectFields {
		found := false
		for _, available := range viewAvailableFields {
			if available == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q to be in viewAvailableFields", field)
		}
	}
}

func TestFilterViewJSONFields_StatusFromFieldValues(t *testing.T) {
	output := ViewJSONOutput{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://example.com",
		Author: "testuser",
		FieldValues: map[string]string{
			"Status":   "In Progress",
			"Priority": "P1",
		},
	}

	result := filterViewJSONFields(output, []string{"status"})

	if result["status"] != "In Progress" {
		t.Errorf("expected status 'In Progress', got %v", result["status"])
	}
}

func TestFilterViewJSONFields_PriorityFromFieldValues(t *testing.T) {
	output := ViewJSONOutput{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://example.com",
		Author: "testuser",
		FieldValues: map[string]string{
			"Status":   "In Progress",
			"Priority": "P1",
		},
	}

	result := filterViewJSONFields(output, []string{"priority"})

	if result["priority"] != "P1" {
		t.Errorf("expected priority 'P1', got %v", result["priority"])
	}
}

func TestFilterViewJSONFields_BranchFromFieldValues(t *testing.T) {
	output := ViewJSONOutput{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://example.com",
		Author: "testuser",
		FieldValues: map[string]string{
			"Status": "In Progress",
			"Branch": "release/v1.0",
		},
	}

	result := filterViewJSONFields(output, []string{"branch"})

	if result["branch"] != "release/v1.0" {
		t.Errorf("expected branch 'release/v1.0', got %v", result["branch"])
	}
}

func TestFilterViewJSONFields_StatusNullWhenNotInProject(t *testing.T) {
	output := ViewJSONOutput{
		Number:      42,
		Title:       "Test Issue",
		State:       "OPEN",
		URL:         "https://example.com",
		Author:      "testuser",
		FieldValues: map[string]string{}, // No project fields
	}

	result := filterViewJSONFields(output, []string{"status"})

	if result["status"] != nil {
		t.Errorf("expected status nil when not in project, got %v", result["status"])
	}
}

func TestFilterViewJSONFields_PriorityNullWhenNotSet(t *testing.T) {
	output := ViewJSONOutput{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://example.com",
		Author: "testuser",
		FieldValues: map[string]string{
			"Status": "In Progress",
			// Priority not set
		},
	}

	result := filterViewJSONFields(output, []string{"priority"})

	if result["priority"] != nil {
		t.Errorf("expected priority nil when not set, got %v", result["priority"])
	}
}

func TestFilterViewJSONFields_MultipleProjectFields(t *testing.T) {
	output := ViewJSONOutput{
		Number: 42,
		Title:  "Test Issue",
		State:  "OPEN",
		URL:    "https://example.com",
		Author: "testuser",
		FieldValues: map[string]string{
			"Status":   "Done",
			"Priority": "P0",
			"Branch":   "patch/v1.1.5",
		},
	}

	result := filterViewJSONFields(output, []string{"status", "priority", "branch"})

	if result["status"] != "Done" {
		t.Errorf("expected status 'Done', got %v", result["status"])
	}
	if result["priority"] != "P0" {
		t.Errorf("expected priority 'P0', got %v", result["priority"])
	}
	if result["branch"] != "patch/v1.1.5" {
		t.Errorf("expected branch 'patch/v1.1.5', got %v", result["branch"])
	}
}

func TestFilterViewJSONFields_AllStatusValues(t *testing.T) {
	// Test that all standard status values work
	statuses := []string{"Backlog", "In Progress", "In Review", "Done"}
	for _, status := range statuses {
		output := ViewJSONOutput{
			Number: 42,
			Title:  "Test Issue",
			State:  "OPEN",
			URL:    "https://example.com",
			Author: "testuser",
			FieldValues: map[string]string{
				"Status": status,
			},
		}

		result := filterViewJSONFields(output, []string{"status"})

		if result["status"] != status {
			t.Errorf("expected status %q, got %v", status, result["status"])
		}
	}
}

// ============================================================================
// Multi-Issue Tests
// ============================================================================

func newMultiMockViewClient() *mockViewClient {
	return &mockViewClient{
		issues: map[int]*api.Issue{
			42: {Number: 42, Title: "First Issue", State: "OPEN", URL: "https://github.com/o/r/issues/42", Author: api.Actor{Login: "user1"}},
			43: {Number: 43, Title: "Second Issue", State: "CLOSED", URL: "https://github.com/o/r/issues/43", Author: api.Actor{Login: "user2"}},
		},
		fieldValuesMap: map[int][]api.FieldValue{
			42: {{Field: "Status", Value: "In progress"}},
			43: {{Field: "Status", Value: "Done"}},
		},
		subIssuesMap:    map[int][]api.SubIssue{},
		parentIssuesMap: map[int]*api.Issue{},
	}
}

func TestRunViewMulti_JSONArray(t *testing.T) {
	mock := newMultiMockViewClient()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetOut(w)
	opts := &viewOptions{jsonFields: "number,title,state"}
	refs := []viewIssueRef{
		{owner: "o", repo: "r", number: 42},
		{owner: "o", repo: "r", number: 43},
	}

	err := runViewMulti(cmd, opts, mock, refs, nil)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("expected valid JSON array, got error: %v\noutput: %s", err, output)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 items in array, got %d", len(result))
	}

	if result[0]["number"].(float64) != 42 {
		t.Errorf("expected first issue number 42, got %v", result[0]["number"])
	}
	if result[1]["number"].(float64) != 43 {
		t.Errorf("expected second issue number 43, got %v", result[1]["number"])
	}
}

func TestRunViewMulti_TableWithSeparator(t *testing.T) {
	mock := newMultiMockViewClient()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetOut(w)
	opts := &viewOptions{}
	refs := []viewIssueRef{
		{owner: "o", repo: "r", number: 42},
		{owner: "o", repo: "r", number: 43},
	}

	err := runViewMulti(cmd, opts, mock, refs, nil)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should contain both issue titles
	if !strings.Contains(output, "First Issue") {
		t.Error("expected output to contain 'First Issue'")
	}
	if !strings.Contains(output, "Second Issue") {
		t.Error("expected output to contain 'Second Issue'")
	}
	// Should contain separator
	if !strings.Contains(output, "\u2550") {
		t.Error("expected output to contain separator line")
	}
}

func TestRunViewMulti_InvalidIssuePartialSuccess(t *testing.T) {
	mock := newMultiMockViewClient()
	mock.issueErrors = map[int]error{
		99: errors.New("issue not found"),
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Capture stderr too
	oldErr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	cmd := &cobra.Command{}
	cmd.SetOut(w)
	opts := &viewOptions{jsonFields: "number,title"}
	refs := []viewIssueRef{
		{owner: "o", repo: "r", number: 42},
		{owner: "o", repo: "r", number: 99},
	}

	err := runViewMulti(cmd, opts, mock, refs, nil)

	w.Close()
	wErr.Close()
	os.Stdout = old
	os.Stderr = oldErr
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	var errBuf bytes.Buffer
	_, _ = io.Copy(&errBuf, rErr)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should output valid JSON with just issue 42
	var result []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON array: %v\noutput: %s", err, buf.String())
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 item (partial success), got %d", len(result))
	}
	if result[0]["number"].(float64) != 42 {
		t.Errorf("expected issue 42, got %v", result[0]["number"])
	}

	// Should warn on stderr about issue 99
	if !strings.Contains(errBuf.String(), "#99") {
		t.Errorf("expected stderr warning about #99, got: %s", errBuf.String())
	}
}

func TestRunViewMulti_BodyStdoutError(t *testing.T) {
	mock := newMultiMockViewClient()
	cmd := &cobra.Command{}
	opts := &viewOptions{bodyStdout: true}
	refs := []viewIssueRef{
		{owner: "o", repo: "r", number: 42},
		{owner: "o", repo: "r", number: 43},
	}

	err := runViewMulti(cmd, opts, mock, refs, nil)
	if err == nil {
		t.Fatal("expected error for --body-stdout with multiple issues")
	}
	if !strings.Contains(err.Error(), "only supported for single issue") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunViewMulti_BodyFileError(t *testing.T) {
	mock := newMultiMockViewClient()
	cmd := &cobra.Command{}
	opts := &viewOptions{bodyFile: true}
	refs := []viewIssueRef{
		{owner: "o", repo: "r", number: 42},
		{owner: "o", repo: "r", number: 43},
	}

	err := runViewMulti(cmd, opts, mock, refs, nil)
	if err == nil {
		t.Fatal("expected error for --body-file with multiple issues")
	}
	if !strings.Contains(err.Error(), "only supported for single issue") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunViewMulti_WebError(t *testing.T) {
	mock := newMultiMockViewClient()
	cmd := &cobra.Command{}
	opts := &viewOptions{web: true}
	refs := []viewIssueRef{
		{owner: "o", repo: "r", number: 42},
		{owner: "o", repo: "r", number: 43},
	}

	err := runViewMulti(cmd, opts, mock, refs, nil)
	if err == nil {
		t.Fatal("expected error for --web with multiple issues")
	}
	if !strings.Contains(err.Error(), "only supported for single issue") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunViewMulti_JQWithArray(t *testing.T) {
	mock := newMultiMockViewClient()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetOut(w)
	opts := &viewOptions{jsonFields: "number,title", jq: ".[].number"}
	refs := []viewIssueRef{
		{owner: "o", repo: "r", number: 42},
		{owner: "o", repo: "r", number: 43},
	}

	err := runViewMulti(cmd, opts, mock, refs, nil)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	// jq .[].number should output "42\n43"
	if !strings.Contains(output, "42") || !strings.Contains(output, "43") {
		t.Errorf("expected jq output to contain 42 and 43, got: %s", output)
	}
}

func TestViewCommand_AcceptsMultipleArgs(t *testing.T) {
	cmd := newViewCommand()
	// The args validator should accept 2+ args
	err := cmd.Args(cmd, []string{"42", "43"})
	if err != nil {
		t.Errorf("expected no error for multiple args, got: %v", err)
	}
}

func TestViewCommand_AcceptsSingleArg(t *testing.T) {
	cmd := newViewCommand()
	err := cmd.Args(cmd, []string{"42"})
	if err != nil {
		t.Errorf("expected no error for single arg, got: %v", err)
	}
}

func TestViewCommand_RejectsZeroArgs(t *testing.T) {
	cmd := newViewCommand()
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("expected error for zero args")
	}
}
