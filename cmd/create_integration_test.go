//go:build integration

package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/testutil"
)

// TestRunCreate_Integration_TitleAndBody tests creating issue with --title and --body
func TestRunCreate_Integration_TitleAndBody(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - TitleAndBody - %d", testUniqueID())
	body := "This is a test issue body created by integration tests."

	result := testutil.RunCommand(t, "create", "--title", title, "--body", body)

	testutil.AssertExitCode(t, result, 0)
	testutil.AssertContains(t, result.Stdout, "Created issue #")
	testutil.AssertContains(t, result.Stdout, title)

	// Cleanup: extract issue number and delete
	issueNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Verify the issue exists by viewing it
	viewResult := testutil.RunCommand(t, "view", fmt.Sprintf("%d", issueNum))
	testutil.AssertExitCode(t, viewResult, 0)
	testutil.AssertContains(t, viewResult.Stdout, title)
	testutil.AssertContains(t, viewResult.Stdout, body)
}

// TestRunCreate_Integration_StatusAndPriority tests setting --status and --priority
func TestRunCreate_Integration_StatusAndPriority(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - StatusPriority - %d", testUniqueID())

	result := testutil.RunCommand(t, "create",
		"--title", title,
		"--status", "in_progress",
		"--priority", "p0",
	)

	testutil.AssertExitCode(t, result, 0)
	testutil.AssertContains(t, result.Stdout, "Created issue #")

	issueNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Verify field values via view --json=<fields>
	testutil.AssertIssueFields(t, issueNum, map[string]string{
		"status":   "In progress",
		"priority": "P0",
	})
}

// TestRunCreate_Integration_Labels tests applying --label flags
func TestRunCreate_Integration_Labels(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - Labels - %d", testUniqueID())

	// Note: The label must exist in the test repo
	// We use "bug" which is a standard GitHub label
	result := testutil.RunCommand(t, "create",
		"--title", title,
		"--label", "bug",
	)

	testutil.AssertExitCode(t, result, 0)

	issueNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Verify label was applied
	testutil.AssertIssueHasLabel(t, issueNum, "bug")
}

// TestRunCreate_Integration_ConfigDefaults tests applying defaults from config
func TestRunCreate_Integration_ConfigDefaults(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - ConfigDefaults - %d", testUniqueID())

	// Create without explicit status/priority - should use config defaults
	// Config has: status: backlog, priority: p2
	result := testutil.RunCommand(t, "create", "--title", title)

	testutil.AssertExitCode(t, result, 0)

	issueNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Should have default status (Backlog) and priority (P2)
	testutil.AssertIssueFields(t, issueNum, map[string]string{
		"status":   "Backlog",
		"priority": "P2",
	})
}

// TestRunCreate_Integration_OverrideDefaults tests overriding config defaults with CLI flags
func TestRunCreate_Integration_OverrideDefaults(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - OverrideDefaults - %d", testUniqueID())

	// Override default status (backlog) with in_progress
	// Override default priority (p2) with p1
	result := testutil.RunCommand(t, "create",
		"--title", title,
		"--status", "in_progress",
		"--priority", "p1",
	)

	testutil.AssertExitCode(t, result, 0)

	issueNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Verify CLI flags override defaults
	testutil.AssertIssueFields(t, issueNum, map[string]string{
		"status":   "In progress",
		"priority": "P1",
	})
}

// TestRunCreate_Integration_FieldAliases tests field value aliases
func TestRunCreate_Integration_FieldAliases(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - FieldAliases - %d", testUniqueID())

	// Use aliases: in_progress -> "In progress", p0 -> "P0"
	result := testutil.RunCommand(t, "create",
		"--title", title,
		"--status", "in_review", // alias for "In review"
		"--priority", "p1", // alias for "P1"
	)

	testutil.AssertExitCode(t, result, 0)

	issueNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Verify aliases resolved correctly
	testutil.AssertIssueFields(t, issueNum, map[string]string{
		"status":   "In review",
		"priority": "P1",
	})
}

// TestRunCreate_Integration_NoTitle tests error when title is missing
func TestRunCreate_Integration_NoTitle(t *testing.T) {
	testutil.RequireTestEnv(t)

	result := testutil.RunCommand(t, "create", "--body", "body without title")

	// Should fail
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code when title is missing")
	}

	testutil.AssertContains(t, result.Stderr, "--title is required")
}

// TestRunCreate_Integration_OutputFormat tests output format
func TestRunCreate_Integration_OutputFormat(t *testing.T) {
	testutil.RequireTestEnv(t)

	title := fmt.Sprintf("Test Issue - OutputFormat - %d", testUniqueID())

	result := testutil.RunCommand(t, "create", "--title", title)

	testutil.AssertExitCode(t, result, 0)

	issueNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Output should include "Created issue #N: Title"
	testutil.AssertContains(t, result.Stdout, fmt.Sprintf("Created issue #%d:", issueNum))
	testutil.AssertContains(t, result.Stdout, title)

	// Output should include URL
	testutil.AssertContains(t, result.Stdout, "https://github.com/")
}

// testUniqueID returns a unique identifier for test issues
// Uses a simple counter-based approach
var testCounter int

func testUniqueID() int {
	testCounter++
	return testCounter + int(strings.Count("integration", "i")*1000000)
}
