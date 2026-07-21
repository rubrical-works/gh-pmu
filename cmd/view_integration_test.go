//go:build integration

package cmd

import (
	"encoding/json"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/testutil"
)

// TestRunView_Integration_ByNumber tests viewing issue by number
func TestRunView_Integration_ByNumber(t *testing.T) {
	testutil.RequireTestEnv(t)

	// View seed issue #1
	result := testutil.RunCommand(t, "view", "1")

	testutil.AssertExitCode(t, result, 0)

	// Should show issue details
	testutil.AssertContains(t, result.Stdout, "Seed Issue 1")
	testutil.AssertContains(t, result.Stdout, "#1")
	testutil.AssertContains(t, result.Stdout, "State:")
	testutil.AssertContains(t, result.Stdout, "URL:")
}

// TestRunView_Integration_ByNumberWithHash tests viewing issue by #number format
func TestRunView_Integration_ByNumberWithHash(t *testing.T) {
	testutil.RequireTestEnv(t)

	// View seed issue #2 with # prefix
	result := testutil.RunCommand(t, "view", "#2")

	testutil.AssertExitCode(t, result, 0)

	testutil.AssertContains(t, result.Stdout, "Seed Issue 2")
	testutil.AssertContains(t, result.Stdout, "#2")
}

// TestRunView_Integration_ByURL tests viewing issue by URL
func TestRunView_Integration_ByURL(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// View seed issue #3 by URL
	url := "https://github.com/" + env.RepoOwner + "/" + env.RepoName + "/issues/3"
	result := testutil.RunCommand(t, "view", url)

	testutil.AssertExitCode(t, result, 0)

	testutil.AssertContains(t, result.Stdout, "Seed Issue 3")
	testutil.AssertContains(t, result.Stdout, "#3")
}

// TestRunView_Integration_WithSubIssues tests viewing issue with sub-issues
func TestRunView_Integration_WithSubIssues(t *testing.T) {
	testutil.RequireTestEnv(t)

	// View seed issue #4 which has sub-issue #5
	result := testutil.RunCommand(t, "view", "4")

	testutil.AssertExitCode(t, result, 0)

	// Should show issue details
	testutil.AssertContains(t, result.Stdout, "Seed Issue 4")

	// Should show sub-issues section
	testutil.AssertContains(t, result.Stdout, "Sub-Issues:")
	testutil.AssertContains(t, result.Stdout, "#5")

	// Should show progress
	testutil.AssertContains(t, result.Stdout, "sub-issues complete")
}

// TestRunView_Integration_SubIssueShowsParent tests viewing sub-issue shows parent
func TestRunView_Integration_SubIssueShowsParent(t *testing.T) {
	testutil.RequireTestEnv(t)

	// View seed issue #5 which is a sub-issue of #4
	result := testutil.RunCommand(t, "view", "5")

	testutil.AssertExitCode(t, result, 0)

	// Should show issue details
	testutil.AssertContains(t, result.Stdout, "Seed Issue 5")

	// Should show parent issue
	testutil.AssertContains(t, result.Stdout, "Parent Issue:")
	testutil.AssertContains(t, result.Stdout, "#4")
}

// TestRunView_Integration_JSONOutput tests --json output format
func TestRunView_Integration_JSONOutput(t *testing.T) {
	testutil.RequireTestEnv(t)

	result := testutil.RunCommand(t, "view", "1", "--json=number,title,url,fieldValues")

	testutil.AssertExitCode(t, result, 0)

	// Parse JSON output
	var output struct {
		Number      int               `json:"number"`
		Title       string            `json:"title"`
		State       string            `json:"state"`
		Body        string            `json:"body"`
		URL         string            `json:"url"`
		Author      string            `json:"author"`
		Assignees   []string          `json:"assignees"`
		Labels      []string          `json:"labels"`
		FieldValues map[string]string `json:"fieldValues"`
	}

	err := json.Unmarshal([]byte(result.Stdout), &output)
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, result.Stdout)
	}

	// Verify structure
	if output.Number != 1 {
		t.Errorf("expected issue number 1, got %d", output.Number)
	}

	if output.Title == "" {
		t.Error("expected non-empty title")
	}

	if output.URL == "" {
		t.Error("expected non-empty URL")
	}

	// Should have field values from project
	if len(output.FieldValues) == 0 {
		t.Error("expected field values in JSON output")
	}
}

// TestRunView_Integration_JSONWithSubIssues tests --json output with sub-issues
func TestRunView_Integration_JSONWithSubIssues(t *testing.T) {
	testutil.RequireTestEnv(t)

	result := testutil.RunCommand(t, "view", "4", "--json=number,subIssues,subProgress")

	testutil.AssertExitCode(t, result, 0)

	// Parse JSON output
	var output struct {
		Number    int `json:"number"`
		SubIssues []struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			State  string `json:"state"`
		} `json:"subIssues"`
		SubProgress *struct {
			Total      int `json:"total"`
			Completed  int `json:"completed"`
			Percentage int `json:"percentage"`
		} `json:"subProgress"`
	}

	err := json.Unmarshal([]byte(result.Stdout), &output)
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, result.Stdout)
	}

	// Should have sub-issues
	if len(output.SubIssues) == 0 {
		t.Error("expected sub-issues in JSON output for issue #4")
	}

	// Should have progress
	if output.SubProgress == nil {
		t.Error("expected subProgress in JSON output")
	}
}

// TestRunView_Integration_JSONSubIssueWithParent tests --json output for sub-issue with parent
func TestRunView_Integration_JSONSubIssueWithParent(t *testing.T) {
	testutil.RequireTestEnv(t)

	result := testutil.RunCommand(t, "view", "5", "--json=number,parentIssue")

	testutil.AssertExitCode(t, result, 0)

	// Parse JSON output
	var output struct {
		Number      int `json:"number"`
		ParentIssue *struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
		} `json:"parentIssue"`
	}

	err := json.Unmarshal([]byte(result.Stdout), &output)
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, result.Stdout)
	}

	// Should have parent issue
	if output.ParentIssue == nil {
		t.Error("expected parentIssue in JSON output for issue #5")
	} else if output.ParentIssue.Number != 4 {
		t.Errorf("expected parent issue #4, got #%d", output.ParentIssue.Number)
	}
}

// TestRunView_Integration_NotFound tests issue not found error
func TestRunView_Integration_NotFound(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Try to view non-existent issue
	result := testutil.RunCommand(t, "view", "99999")

	// Should fail
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for non-existent issue")
	}

	// Should show error message
	testutil.AssertContains(t, result.Stderr, "failed to get issue")
}

// TestRunView_Integration_ProjectFields tests that project fields are shown
func TestRunView_Integration_ProjectFields(t *testing.T) {
	testutil.RequireTestEnv(t)

	result := testutil.RunCommand(t, "view", "1")

	testutil.AssertExitCode(t, result, 0)

	// Should show project fields section
	testutil.AssertContains(t, result.Stdout, "Project Fields:")

	// Seed issue #1 has Status=Backlog and Priority=P0
	testutil.AssertContains(t, result.Stdout, "Status:")
	testutil.AssertContains(t, result.Stdout, "Priority:")
}

// TestRunView_Integration_BodyStdout tests --body-stdout flag outputs raw body
func TestRunView_Integration_BodyStdout(t *testing.T) {
	testutil.RequireTestEnv(t)

	result := testutil.RunCommand(t, "view", "1", "--body-stdout")

	testutil.AssertExitCode(t, result, 0)

	// Output should be raw body content only (no headers, no formatting)
	// Should NOT contain view metadata like "State:", "URL:", "Project Fields:"
	testutil.AssertNotContains(t, result.Stdout, "State:")
	testutil.AssertNotContains(t, result.Stdout, "URL:")
	testutil.AssertNotContains(t, result.Stdout, "Project Fields:")

	// Seed issue #1 has body content "Body of seed issue 1"
	testutil.AssertContains(t, result.Stdout, "Body of seed issue 1")
}
