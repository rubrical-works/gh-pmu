package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
)

// mockSplitClient implements splitClient for testing
type mockSplitClient struct {
	issue         *api.Issue
	createdIssues []*api.Issue
	createIndex   int

	// Error injection
	getIssueErr    error
	createIssueErr error
	addSubIssueErr error
}

func newMockSplitClient() *mockSplitClient {
	return &mockSplitClient{
		issue: &api.Issue{
			ID:     "issue-1",
			Number: 42,
			Title:  "Parent Issue",
			Body:   "- [ ] Task 1\n- [ ] Task 2\n- [x] Done task",
		},
		createdIssues: []*api.Issue{},
	}
}

func (m *mockSplitClient) GetIssue(owner, repo string, number int) (*api.Issue, error) {
	if m.getIssueErr != nil {
		return nil, m.getIssueErr
	}
	return m.issue, nil
}

func (m *mockSplitClient) CreateIssue(owner, repo, title, body string, labels []string) (*api.Issue, error) {
	if m.createIssueErr != nil {
		return nil, m.createIssueErr
	}
	if m.createIndex < len(m.createdIssues) {
		issue := m.createdIssues[m.createIndex]
		m.createIndex++
		return issue, nil
	}
	return &api.Issue{
		ID:     "new-issue",
		Number: 100 + m.createIndex,
		Title:  title,
	}, nil
}

func (m *mockSplitClient) AddSubIssue(parentID, issueID string) error {
	return m.addSubIssueErr
}

// ============================================================================
// runSplitWithDeps Tests
// ============================================================================

func TestRunSplitWithDeps_DryRun(t *testing.T) {
	mock := newMockSplitClient()
	mock.issue = &api.Issue{
		Number: 42,
		Title:  "Parent Issue",
		Body:   "- [ ] Task 1\n- [ ] Task 2",
	}

	cmd := newSplitCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &splitOptions{from: "body", dryRun: true}
	args := []string{"42"}
	err := runSplitWithDeps(cmd, args, opts, mock, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Would create") {
		t.Error("expected 'Would create' in dry-run output")
	}
}

func TestRunSplitWithDeps_GetIssueError(t *testing.T) {
	mock := newMockSplitClient()
	mock.getIssueErr = errors.New("issue not found")

	cmd := newSplitCommand()
	opts := &splitOptions{from: "body"}
	args := []string{"42"}
	err := runSplitWithDeps(cmd, args, opts, mock, "owner", "repo", 42)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get issue") {
		t.Errorf("expected 'failed to get issue' error, got: %v", err)
	}
}

func TestRunSplitWithDeps_NoTasks(t *testing.T) {
	mock := newMockSplitClient()
	mock.issue = &api.Issue{
		Number: 42,
		Title:  "Empty Issue",
		Body:   "No checklist items here",
	}

	cmd := newSplitCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &splitOptions{from: "body"}
	args := []string{"42"}
	err := runSplitWithDeps(cmd, args, opts, mock, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No tasks found") {
		t.Error("expected 'No tasks found' in output")
	}
}

func TestRunSplitWithDeps_NoSource(t *testing.T) {
	mock := newMockSplitClient()

	cmd := newSplitCommand()
	opts := &splitOptions{}
	args := []string{"42"} // No --from and no task arguments
	err := runSplitWithDeps(cmd, args, opts, mock, "owner", "repo", 42)

	if err == nil {
		t.Fatal("expected error for no source, got nil")
	}
	if !strings.Contains(err.Error(), "no tasks specified") {
		t.Errorf("expected 'no tasks specified' error, got: %v", err)
	}
}

func TestRunSplitWithDeps_WithTaskArgs(t *testing.T) {
	mock := newMockSplitClient()
	mock.issue = &api.Issue{
		ID:     "issue-42",
		Number: 42,
		Title:  "Parent Issue",
	}
	mock.createdIssues = []*api.Issue{
		{ID: "new-1", Number: 43, Title: "Task 1"},
		{ID: "new-2", Number: 44, Title: "Task 2"},
	}

	cmd := newSplitCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &splitOptions{}
	args := []string{"42", "Task 1", "Task 2"}
	err := runSplitWithDeps(cmd, args, opts, mock, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Created sub-issue") {
		t.Error("expected 'Created sub-issue' in output")
	}
}

func TestRunSplitWithDeps_JSONOutput(t *testing.T) {
	mock := newMockSplitClient()
	mock.issue = &api.Issue{
		Number: 42,
		Title:  "Parent Issue",
		Body:   "- [ ] Task 1",
	}

	cmd := newSplitCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	opts := &splitOptions{from: "body", dryRun: true, json: true}
	args := []string{"42"}
	err := runSplitWithDeps(cmd, args, opts, mock, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSplitCommand(t *testing.T) {
	t.Run("has correct command structure", func(t *testing.T) {
		cmd := newSplitCommand()

		if cmd.Use != "split <issue> [tasks...]" {
			t.Errorf("expected Use to be 'split <issue> [tasks...]', got %s", cmd.Use)
		}

		if cmd.Short == "" {
			t.Error("expected Short description to be set")
		}
	})

	t.Run("has required flags", func(t *testing.T) {
		cmd := newSplitCommand()

		// Check --from flag
		fromFlag := cmd.Flags().Lookup("from")
		if fromFlag == nil {
			t.Error("expected --from flag")
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
	})

	t.Run("command is registered in root", func(t *testing.T) {
		root := NewRootCommand()
		buf := new(bytes.Buffer)
		root.SetOut(buf)
		root.SetArgs([]string{"split", "--help"})
		err := root.Execute()
		if err != nil {
			t.Errorf("split command not registered: %v", err)
		}
	})
}

func TestParseChecklist(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "simple checklist",
			input: `# Epic Title

Some description here.

## Tasks
- [ ] Task one
- [ ] Task two
- [ ] Task three
`,
			expected: []string{"Task one", "Task two", "Task three"},
		},
		{
			name: "mixed checked and unchecked",
			input: `- [x] Completed task
- [ ] Pending task
- [ ] Another pending
`,
			expected: []string{"Pending task", "Another pending"},
		},
		{
			name: "with nested content",
			input: `- [ ] Main task
  - Some notes
  - More notes
- [ ] Second task
`,
			expected: []string{"Main task", "Second task"},
		},
		{
			name:     "no checklist items",
			input:    "Just some text without any checklist",
			expected: []string{},
		},
		{
			name: "checklist with extra whitespace",
			input: `- [ ]   Task with leading space
- [ ]	Task with tab
`,
			expected: []string{"Task with leading space", "Task with tab"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseChecklist(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d items, got %d", len(tt.expected), len(result))
				t.Errorf("got: %v", result)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("item %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestOutputSplitJSON(t *testing.T) {
	t.Run("includes parent issue info", func(t *testing.T) {
		cmd := newSplitCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		parent := &api.Issue{
			Number: 123,
			Title:  "Parent Epic",
			URL:    "https://github.com/owner/repo/issues/123",
		}
		tasks := []string{"Task 1", "Task 2", "Task 3"}

		err := outputSplitJSON(cmd, parent, tasks, "dry-run")
		if err != nil {
			t.Fatalf("outputSplitJSON failed: %v", err)
		}

		var decoded struct {
			Status string `json:"status"`
			Parent struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
				URL    string `json:"url"`
			} `json:"parent"`
			TaskCount int      `json:"taskCount"`
			Tasks     []string `json:"tasks"`
		}
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("output was not valid JSON: %v\noutput: %s", err, buf.String())
		}

		if decoded.Status != "dry-run" {
			t.Errorf("expected status 'dry-run', got %q", decoded.Status)
		}
		if decoded.Parent.Number != 123 {
			t.Errorf("expected parent number 123, got %d", decoded.Parent.Number)
		}
		if decoded.Parent.Title != "Parent Epic" {
			t.Errorf("expected parent title 'Parent Epic', got %q", decoded.Parent.Title)
		}
		if decoded.Parent.URL != "https://github.com/owner/repo/issues/123" {
			t.Errorf("unexpected parent url: %q", decoded.Parent.URL)
		}
		if decoded.TaskCount != 3 {
			t.Errorf("expected taskCount 3, got %d", decoded.TaskCount)
		}
		if len(decoded.Tasks) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(decoded.Tasks))
		}
		if decoded.Tasks[0] != "Task 1" || decoded.Tasks[2] != "Task 3" {
			t.Errorf("unexpected tasks content: %v", decoded.Tasks)
		}
	})

	t.Run("handles nil tasks", func(t *testing.T) {
		cmd := newSplitCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		parent := &api.Issue{
			Number: 1,
			Title:  "No tasks",
			URL:    "https://github.com/owner/repo/issues/1",
		}

		err := outputSplitJSON(cmd, parent, nil, "no-tasks")
		if err != nil {
			t.Fatalf("outputSplitJSON failed with nil tasks: %v", err)
		}

		var decoded struct {
			Status    string   `json:"status"`
			TaskCount int      `json:"taskCount"`
			Tasks     []string `json:"tasks"`
		}
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("output was not valid JSON: %v\noutput: %s", err, buf.String())
		}

		if decoded.Status != "no-tasks" {
			t.Errorf("expected status 'no-tasks', got %q", decoded.Status)
		}
		if decoded.TaskCount != 0 {
			t.Errorf("expected taskCount 0 for nil tasks, got %d", decoded.TaskCount)
		}
		if len(decoded.Tasks) != 0 {
			t.Errorf("expected empty tasks, got %v", decoded.Tasks)
		}
	})

	t.Run("status field is preserved", func(t *testing.T) {
		parent := &api.Issue{Number: 1, Title: "Test"}

		statuses := []string{"dry-run", "no-tasks", "completed"}
		for _, status := range statuses {
			cmd := newSplitCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)

			err := outputSplitJSON(cmd, parent, []string{}, status)
			if err != nil {
				t.Fatalf("outputSplitJSON failed with status %q: %v", status, err)
			}

			var decoded struct {
				Status string `json:"status"`
			}
			if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
				t.Fatalf("output was not valid JSON for status %q: %v\noutput: %s", status, err, buf.String())
			}
			if decoded.Status != status {
				t.Errorf("expected status %q in output, got %q", status, decoded.Status)
			}
		}
	})
}

// splitCreatedOutput mirrors the JSON shape emitted by outputSplitJSONCreated.
type splitCreatedOutput struct {
	Status string `json:"status"`
	Parent struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		URL    string `json:"url"`
	} `json:"parent"`
	CreatedCount int `json:"createdCount"`
	FailedCount  int `json:"failedCount"`
	Created      []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		URL    string `json:"url"`
	} `json:"created"`
	Failed []string `json:"failed"`
}

func TestOutputSplitJSONCreated(t *testing.T) {
	t.Run("tracks created vs failed counts", func(t *testing.T) {
		cmd := newSplitCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		parent := &api.Issue{
			Number: 100,
			Title:  "Parent Issue",
			URL:    "https://github.com/owner/repo/issues/100",
		}

		created := []api.Issue{
			{Number: 101, Title: "Sub 1", URL: "https://github.com/owner/repo/issues/101"},
			{Number: 102, Title: "Sub 2", URL: "https://github.com/owner/repo/issues/102"},
			{Number: 103, Title: "Sub 3", URL: "https://github.com/owner/repo/issues/103"},
		}
		failed := []string{"Failed task 1"}

		err := outputSplitJSONCreated(cmd, parent, created, failed)
		if err != nil {
			t.Fatalf("outputSplitJSONCreated failed: %v", err)
		}

		var decoded splitCreatedOutput
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("output was not valid JSON: %v\noutput: %s", err, buf.String())
		}

		if decoded.Status != "completed" {
			t.Errorf("expected status 'completed', got %q", decoded.Status)
		}
		if decoded.Parent.Number != 100 {
			t.Errorf("expected parent number 100, got %d", decoded.Parent.Number)
		}
		if decoded.CreatedCount != 3 {
			t.Errorf("expected createdCount 3, got %d", decoded.CreatedCount)
		}
		if decoded.FailedCount != 1 {
			t.Errorf("expected failedCount 1, got %d", decoded.FailedCount)
		}
		if len(decoded.Created) != 3 {
			t.Fatalf("expected 3 created entries, got %d", len(decoded.Created))
		}
		if decoded.Created[0].Number != 101 || decoded.Created[1].Number != 102 || decoded.Created[2].Number != 103 {
			t.Errorf("unexpected created issue numbers: %+v", decoded.Created)
		}
		if decoded.Created[0].Title != "Sub 1" {
			t.Errorf("expected first created title 'Sub 1', got %q", decoded.Created[0].Title)
		}
		if decoded.Created[2].URL != "https://github.com/owner/repo/issues/103" {
			t.Errorf("unexpected created url: %q", decoded.Created[2].URL)
		}
		if len(decoded.Failed) != 1 || decoded.Failed[0] != "Failed task 1" {
			t.Errorf("unexpected failed content: %v", decoded.Failed)
		}
	})

	t.Run("handles empty created list", func(t *testing.T) {
		cmd := newSplitCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		parent := &api.Issue{Number: 1, Title: "Parent"}

		err := outputSplitJSONCreated(cmd, parent, []api.Issue{}, []string{"all", "failed"})
		if err != nil {
			t.Fatalf("outputSplitJSONCreated failed with empty created: %v", err)
		}

		var decoded splitCreatedOutput
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("output was not valid JSON: %v\noutput: %s", err, buf.String())
		}

		if decoded.Status != "completed" {
			t.Errorf("expected status 'completed', got %q", decoded.Status)
		}
		if decoded.CreatedCount != 0 {
			t.Errorf("expected createdCount 0, got %d", decoded.CreatedCount)
		}
		if len(decoded.Created) != 0 {
			t.Errorf("expected no created entries, got %+v", decoded.Created)
		}
		if decoded.FailedCount != 2 {
			t.Errorf("expected failedCount 2, got %d", decoded.FailedCount)
		}
		if len(decoded.Failed) != 2 {
			t.Fatalf("expected 2 failed entries, got %d", len(decoded.Failed))
		}
		if decoded.Failed[0] != "all" || decoded.Failed[1] != "failed" {
			t.Errorf("unexpected failed content: %v", decoded.Failed)
		}
	})

	t.Run("handles empty failed list", func(t *testing.T) {
		cmd := newSplitCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		parent := &api.Issue{Number: 1, Title: "Parent"}

		created := []api.Issue{
			{Number: 2, Title: "Sub", URL: "url"},
		}

		err := outputSplitJSONCreated(cmd, parent, created, []string{})
		if err != nil {
			t.Fatalf("outputSplitJSONCreated failed with empty failed: %v", err)
		}

		var decoded splitCreatedOutput
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("output was not valid JSON: %v\noutput: %s", err, buf.String())
		}

		if decoded.CreatedCount != 1 {
			t.Errorf("expected createdCount 1, got %d", decoded.CreatedCount)
		}
		if len(decoded.Created) != 1 {
			t.Fatalf("expected 1 created entry, got %d", len(decoded.Created))
		}
		if decoded.Created[0].Number != 2 || decoded.Created[0].Title != "Sub" {
			t.Errorf("unexpected created entry: %+v", decoded.Created[0])
		}
		if decoded.FailedCount != 0 {
			t.Errorf("expected failedCount 0, got %d", decoded.FailedCount)
		}
		if len(decoded.Failed) != 0 {
			t.Errorf("expected no failed entries, got %v", decoded.Failed)
		}
	})
}

// TestSplitJSONOutput_Structure calls the real output functions with the cobra
// writer redirected to a buffer and decodes what they emitted, so the split
// JSON shape is verified against production rather than a hand-built copy.
func TestSplitJSONOutput_Structure(t *testing.T) {
	t.Run("outputSplitJSON produces valid JSON", func(t *testing.T) {
		parent := &api.Issue{
			Number: 123,
			Title:  "Parent Epic",
			URL:    "https://github.com/owner/repo/issues/123",
		}
		tasks := []string{"Task 1", "Task 2", "Task 3"}

		cmd := newSplitCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		if err := outputSplitJSON(cmd, parent, tasks, "dry-run"); err != nil {
			t.Fatalf("outputSplitJSON() error = %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to decode emitted JSON: %v\nOutput: %s", err, buf.String())
		}

		if result["status"] != "dry-run" {
			t.Errorf("Expected status 'dry-run', got %v", result["status"])
		}
		if int(result["taskCount"].(float64)) != 3 {
			t.Errorf("Expected taskCount 3, got %v", result["taskCount"])
		}

		emittedTasks, ok := result["tasks"].([]interface{})
		if !ok {
			t.Fatal("Expected tasks to be an array")
		}
		if len(emittedTasks) != 3 || emittedTasks[0] != "Task 1" {
			t.Errorf("Expected [Task 1 Task 2 Task 3], got %v", emittedTasks)
		}

		parentJSON, ok := result["parent"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected parent to be an object")
		}
		if int(parentJSON["number"].(float64)) != 123 {
			t.Errorf("Expected parent number 123, got %v", parentJSON["number"])
		}
		if parentJSON["title"] != "Parent Epic" {
			t.Errorf("Expected parent title 'Parent Epic', got %v", parentJSON["title"])
		}
		if parentJSON["url"] != "https://github.com/owner/repo/issues/123" {
			t.Errorf("Expected parent url to match, got %v", parentJSON["url"])
		}
	})

	t.Run("outputSplitJSON reports taskCount matching the tasks emitted", func(t *testing.T) {
		parent := &api.Issue{Number: 1, Title: "Parent", URL: "url"}

		cmd := newSplitCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		if err := outputSplitJSON(cmd, parent, []string{}, "no-tasks"); err != nil {
			t.Fatalf("outputSplitJSON() error = %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to decode emitted JSON: %v", err)
		}

		if int(result["taskCount"].(float64)) != 0 {
			t.Errorf("Expected taskCount 0, got %v", result["taskCount"])
		}
		if result["status"] != "no-tasks" {
			t.Errorf("Expected status 'no-tasks', got %v", result["status"])
		}
	})

	t.Run("outputSplitJSONCreated produces valid JSON with counts", func(t *testing.T) {
		parent := &api.Issue{Number: 100, Title: "Parent", URL: "url"}
		created := []api.Issue{
			{Number: 101, Title: "Sub 1", URL: "url1"},
			{Number: 102, Title: "Sub 2", URL: "url2"},
			{Number: 103, Title: "Sub 3", URL: "url3"},
		}
		failed := []string{"Failed task"}

		cmd := newSplitCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		if err := outputSplitJSONCreated(cmd, parent, created, failed); err != nil {
			t.Fatalf("outputSplitJSONCreated() error = %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to decode emitted JSON: %v\nOutput: %s", err, buf.String())
		}

		if result["status"] != "completed" {
			t.Errorf("Expected status 'completed', got %v", result["status"])
		}
		if int(result["createdCount"].(float64)) != 3 {
			t.Errorf("Expected createdCount 3, got %v", result["createdCount"])
		}
		if int(result["failedCount"].(float64)) != 1 {
			t.Errorf("Expected failedCount 1, got %v", result["failedCount"])
		}

		createdJSON, ok := result["created"].([]interface{})
		if !ok {
			t.Fatal("Expected created to be an array")
		}
		if len(createdJSON) != 3 {
			t.Fatalf("Expected 3 created items, got %d", len(createdJSON))
		}
		first, ok := createdJSON[0].(map[string]interface{})
		if !ok {
			t.Fatal("Expected created[0] to be an object")
		}
		if int(first["number"].(float64)) != 101 || first["title"] != "Sub 1" {
			t.Errorf("Expected created[0] to be #101 'Sub 1', got %v", first)
		}

		failedJSON, ok := result["failed"].([]interface{})
		if !ok {
			t.Fatal("Expected failed to be an array")
		}
		if len(failedJSON) != 1 || failedJSON[0] != "Failed task" {
			t.Errorf("Expected failed [Failed task], got %v", failedJSON)
		}
	})

	t.Run("outputSplitJSONCreated emits created as an array when empty", func(t *testing.T) {
		parent := &api.Issue{Number: 100, Title: "Parent", URL: "url"}

		cmd := newSplitCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		if err := outputSplitJSONCreated(cmd, parent, nil, nil); err != nil {
			t.Fatalf("outputSplitJSONCreated() error = %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to decode emitted JSON: %v", err)
		}

		// created is built with make(...,0,len) so it must encode as [] not null.
		if _, ok := result["created"].([]interface{}); !ok {
			t.Errorf("Expected created to be an array, got %T (%v)", result["created"], result["created"])
		}
		if int(result["createdCount"].(float64)) != 0 {
			t.Errorf("Expected createdCount 0, got %v", result["createdCount"])
		}
	})
}
