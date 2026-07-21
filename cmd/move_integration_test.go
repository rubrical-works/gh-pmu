//go:build integration

package cmd

import (
	"fmt"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/testutil"
)

// TestRunMove_Integration_ChangeStatus tests changing --status
func TestRunMove_Integration_ChangeStatus(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create a test issue
	title := fmt.Sprintf("Test Issue - MoveStatus - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Move to in_progress
	moveResult := testutil.RunCommand(t, "move", fmt.Sprintf("%d", issueNum), "--status", "in_progress")
	testutil.AssertExitCode(t, moveResult, 0)
	testutil.AssertContains(t, moveResult.Stdout, "Updated issue")
	testutil.AssertContains(t, moveResult.Stdout, "Status")
	testutil.AssertContains(t, moveResult.Stdout, "In progress")

	// Verify change via view
	testutil.AssertIssueField(t, issueNum, "status", "In progress")
}

// TestRunMove_Integration_ChangePriority tests changing --priority
func TestRunMove_Integration_ChangePriority(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - MovePriority - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--priority", "p2")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Move to P0
	moveResult := testutil.RunCommand(t, "move", fmt.Sprintf("%d", issueNum), "--priority", "p0")
	testutil.AssertExitCode(t, moveResult, 0)
	testutil.AssertContains(t, moveResult.Stdout, "Priority")
	testutil.AssertContains(t, moveResult.Stdout, "P0")

	// Verify change
	testutil.AssertIssueField(t, issueNum, "priority", "P0")
}

// TestRunMove_Integration_MultipleFields tests changing multiple fields
func TestRunMove_Integration_MultipleFields(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - MoveMultiple - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog", "--priority", "p2")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Move both status and priority
	moveResult := testutil.RunCommand(t, "move", fmt.Sprintf("%d", issueNum),
		"--status", "in_progress",
		"--priority", "p1",
	)
	testutil.AssertExitCode(t, moveResult, 0)
	testutil.AssertContains(t, moveResult.Stdout, "Status")
	testutil.AssertContains(t, moveResult.Stdout, "Priority")

	// Verify changes
	testutil.AssertIssueFields(t, issueNum, map[string]string{
		"status":   "In progress",
		"priority": "P1",
	})
}

// TestRunMove_Integration_FieldAliases tests field value aliases
func TestRunMove_Integration_FieldAliases(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - MoveAliases - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title)
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Use aliases: in_review -> "In review"
	moveResult := testutil.RunCommand(t, "move", fmt.Sprintf("%d", issueNum), "--status", "in_review")
	testutil.AssertExitCode(t, moveResult, 0)
	testutil.AssertContains(t, moveResult.Stdout, "In review")

	// Verify alias resolved correctly
	testutil.AssertIssueField(t, issueNum, "status", "In review")
}

// TestRunMove_Integration_NotInProject tests issue not in project error
func TestRunMove_Integration_NotInProject(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Try to move a non-existent issue number
	result := testutil.RunCommand(t, "move", "99999", "--status", "backlog")

	// Should fail
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for non-existent issue")
	}

	// Should show error message
	if result.Stderr == "" && result.Stdout == "" {
		t.Error("expected error message")
	}
}

// TestRunMove_Integration_NoFlags tests error when no flags provided
func TestRunMove_Integration_NoFlags(t *testing.T) {
	testutil.RequireTestEnv(t)

	result := testutil.RunCommand(t, "move", "1")

	// Should fail
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code when no flags provided")
	}

	testutil.AssertContains(t, result.Stderr, "at least one of --status or --priority is required")
}

// TestRunMove_Integration_DryRun tests --dry-run flag
func TestRunMove_Integration_DryRun(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - MoveDryRun - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Dry run - should not change anything
	moveResult := testutil.RunCommand(t, "move", fmt.Sprintf("%d", issueNum),
		"--status", "done",
		"--dry-run",
	)
	testutil.AssertExitCode(t, moveResult, 0)
	testutil.AssertContains(t, moveResult.Stdout, "Dry run")

	// Verify status is still backlog — the dry run must not have applied Done.
	// Comparing the decoded field replaces the old pair of substring checks,
	// whose AssertNotContains was keyed on a "Status" casing that view never
	// emits (it emits "status") and so could never have failed (#891).
	testutil.AssertIssueField(t, issueNum, "status", "Backlog")
}

// TestRunMove_Integration_Recursive tests --recursive flag with sub-issues
func TestRunMove_Integration_Recursive(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create a parent issue
	parentTitle := fmt.Sprintf("Test Parent - Recursive - %d", testUniqueID())
	parentResult := testutil.RunCommand(t, "create", "--title", parentTitle, "--status", "backlog")
	testutil.AssertExitCode(t, parentResult, 0)
	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	// Create a sub-issue
	childTitle := fmt.Sprintf("Test Child - Recursive - %d", testUniqueID())
	childResult := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", childTitle,
	)
	testutil.AssertExitCode(t, childResult, 0)
	childNum := testutil.ExtractIssueNumber(t, childResult.Stdout)

	// Add sub-issue to project
	testutil.RunCommand(t, "move", fmt.Sprintf("%d", childNum), "--status", "backlog")
	defer testutil.DeleteTestIssue(t, childNum)

	// Move with dry-run to verify recursive detection
	result := testutil.RunCommand(t, "move", fmt.Sprintf("%d", parentNum),
		"--status", "done",
		"--recursive",
		"--dry-run",
	)

	testutil.AssertExitCode(t, result, 0)
	testutil.AssertContains(t, result.Stdout, "Dry run")
	testutil.AssertContains(t, result.Stdout, "Issues to update")
	testutil.AssertContains(t, result.Stdout, fmt.Sprintf("#%d", parentNum))
	testutil.AssertContains(t, result.Stdout, fmt.Sprintf("#%d", childNum))
}

// TestRunMove_Integration_BatchMove tests moving multiple issues at once
func TestRunMove_Integration_BatchMove(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create two test issues
	title1 := fmt.Sprintf("Test Issue - Batch1 - %d", testUniqueID())
	create1 := testutil.RunCommand(t, "create", "--title", title1, "--status", "backlog")
	testutil.AssertExitCode(t, create1, 0)
	num1 := testutil.ExtractIssueNumber(t, create1.Stdout)
	defer testutil.DeleteTestIssue(t, num1)

	title2 := fmt.Sprintf("Test Issue - Batch2 - %d", testUniqueID())
	create2 := testutil.RunCommand(t, "create", "--title", title2, "--status", "backlog")
	testutil.AssertExitCode(t, create2, 0)
	num2 := testutil.ExtractIssueNumber(t, create2.Stdout)
	defer testutil.DeleteTestIssue(t, num2)

	// Move both issues at once (--yes skips confirmation prompt)
	moveResult := testutil.RunCommand(t, "move",
		fmt.Sprintf("%d", num1), fmt.Sprintf("%d", num2),
		"--status", "in_progress", "--yes",
	)
	testutil.AssertExitCode(t, moveResult, 0)
	testutil.AssertContains(t, moveResult.Stdout, fmt.Sprintf("#%d", num1))
	testutil.AssertContains(t, moveResult.Stdout, fmt.Sprintf("#%d", num2))

	// Verify both issues moved
	view1 := testutil.RunCommand(t, "view", fmt.Sprintf("%d", num1), "--json=status")
	testutil.AssertExitCode(t, view1, 0)
	testutil.AssertContains(t, view1.Stdout, "In progress")

	view2 := testutil.RunCommand(t, "view", fmt.Sprintf("%d", num2), "--json=status")
	testutil.AssertExitCode(t, view2, 0)
	testutil.AssertContains(t, view2.Stdout, "In progress")
}

// TestRunMove_Integration_BacklogFlag tests --backlog flag clears branch field
func TestRunMove_Integration_BacklogFlag(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create issue and set a branch field
	title := fmt.Sprintf("Test Issue - Backlog - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Set a branch field first
	branchResult := testutil.RunCommand(t, "move", fmt.Sprintf("%d", issueNum),
		"--branch", "release/v99.0.0-test")
	testutil.AssertExitCode(t, branchResult, 0)

	// Verify branch was set
	viewBefore := testutil.RunCommand(t, "view", fmt.Sprintf("%d", issueNum), "--json=branch")
	testutil.AssertExitCode(t, viewBefore, 0)
	testutil.AssertContains(t, viewBefore.Stdout, "release/v99.0.0-test")

	// Clear branch using --backlog flag
	moveResult := testutil.RunCommand(t, "move", fmt.Sprintf("%d", issueNum), "--backlog")
	testutil.AssertExitCode(t, moveResult, 0)
	testutil.AssertContains(t, moveResult.Stdout, "Branch -> (cleared)")

	// Verify branch is cleared
	viewAfter := testutil.RunCommand(t, "view", fmt.Sprintf("%d", issueNum), "--json=branch")
	testutil.AssertExitCode(t, viewAfter, 0)
	testutil.AssertNotContains(t, viewAfter.Stdout, "release/v99.0.0-test")
}

// TestRunMove_Integration_BranchFlag tests --branch flag sets branch field
func TestRunMove_Integration_BranchFlag(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - Branch - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Set branch field
	moveResult := testutil.RunCommand(t, "move", fmt.Sprintf("%d", issueNum),
		"--branch", "release/v99.0.0-test")
	testutil.AssertExitCode(t, moveResult, 0)
	testutil.AssertContains(t, moveResult.Stdout, "Branch")

	// Verify branch was set
	viewResult := testutil.RunCommand(t, "view", fmt.Sprintf("%d", issueNum), "--json=branch")
	testutil.AssertExitCode(t, viewResult, 0)
	testutil.AssertContains(t, viewResult.Stdout, "release/v99.0.0-test")
}
