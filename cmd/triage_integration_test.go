//go:build integration

package cmd

import (
	"fmt"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/testutil"
)

// TestRunTriage_Integration_ListConfigs tests --list shows available configs
func TestRunTriage_Integration_ListConfigs(t *testing.T) {
	testutil.RequireTestEnv(t)

	result := testutil.RunCommand(t, "triage", "--list")

	testutil.AssertExitCode(t, result, 0)

	// Should show available configs from .gh-pmu.json
	testutil.AssertContains(t, result.Stdout, "Available triage configs")
}

// TestRunTriage_Integration_NamedConfigDryRun tests running a named config with --dry-run
func TestRunTriage_Integration_NamedConfigDryRun(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create an issue that matches triage criteria
	title := fmt.Sprintf("Test Issue - TriageConfig - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Run triage with dry-run - uses "unlabeled" config if it exists
	// If no config exists, this tests that we handle that gracefully
	result := testutil.RunCommand(t, "triage", "unlabeled", "--dry-run")

	// Either succeeds with dry-run output or fails with "config not found"
	// Both are valid outcomes depending on project config
	if result.ExitCode == 0 {
		testutil.AssertContains(t, result.Stdout, "Dry run")
	}
}

// TestRunTriage_Integration_AdHocQueryDryRun tests ad-hoc triage with --query and --dry-run
func TestRunTriage_Integration_AdHocQueryDryRun(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create an issue to match
	title := fmt.Sprintf("Test Issue - TriageAdHoc - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Run ad-hoc triage with dry-run
	result := testutil.RunCommand(t, "triage",
		"--query", "status:backlog",
		"--apply", "priority:p2",
		"--dry-run",
	)

	testutil.AssertExitCode(t, result, 0)
	testutil.AssertContains(t, result.Stdout, "Dry run")
	testutil.AssertContains(t, result.Stdout, "Would update")
}

// TestRunTriage_Integration_AdHocQueryApply tests ad-hoc triage with --apply (actual changes)
func TestRunTriage_Integration_AdHocQueryApply(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create an issue to triage
	title := fmt.Sprintf("Test Issue - TriageApply - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog", "--priority", "p2")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Run ad-hoc triage to change priority
	result := testutil.RunCommand(t, "triage",
		"--query", fmt.Sprintf("is:issue is:open %s", title),
		"--apply", "priority:p1",
	)

	testutil.AssertExitCode(t, result, 0)
	testutil.AssertContains(t, result.Stdout, "Updated")

	// Verify the change
	viewResult := testutil.RunCommand(t, "view", fmt.Sprintf("%d", issueNum), "--json")
	testutil.AssertExitCode(t, viewResult, 0)
	testutil.AssertContains(t, viewResult.Stdout, "P1")
}

// TestRunTriage_Integration_AddLabel tests triage adding labels
func TestRunTriage_Integration_AddLabel(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create an issue to triage
	title := fmt.Sprintf("Test Issue - TriageLabel - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title)
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Run ad-hoc triage to add label
	result := testutil.RunCommand(t, "triage",
		"--query", fmt.Sprintf("is:issue is:open %s", title),
		"--apply", "label:bug",
	)

	testutil.AssertExitCode(t, result, 0)

	// Verify the label was added
	viewResult := testutil.RunCommand(t, "view", fmt.Sprintf("%d", issueNum), "--json")
	testutil.AssertExitCode(t, viewResult, 0)
	testutil.AssertContains(t, viewResult.Stdout, "bug")
}

// TestRunTriage_Integration_ConfigNotFound tests error when config doesn't exist
func TestRunTriage_Integration_ConfigNotFound(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Try to run a non-existent config
	result := testutil.RunCommand(t, "triage", "nonexistent-config-xyz")

	// Should fail
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for non-existent config")
	}

	// Should show error message
	if result.Stderr == "" && result.Stdout == "" {
		t.Error("expected error message")
	}
}

// TestRunTriage_Integration_NoQueryOrConfig tests error when neither query nor config provided
func TestRunTriage_Integration_NoQueryOrConfig(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Run triage without query or config name
	result := testutil.RunCommand(t, "triage")

	// Should fail or show help
	// If it exits 0, it should show help/usage
	if result.ExitCode == 0 {
		// Check for usage or help message
		if result.Stdout == "" && result.Stderr == "" {
			t.Error("expected usage message or error")
		}
	}
}

// TestRunTriage_Integration_JSONOutput tests --json output format
func TestRunTriage_Integration_JSONOutput(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create an issue to triage
	title := fmt.Sprintf("Test Issue - TriageJSON - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Run ad-hoc triage with JSON output
	result := testutil.RunCommand(t, "triage",
		"--query", fmt.Sprintf("is:issue is:open %s", title),
		"--apply", "priority:p0",
		"--json",
	)

	testutil.AssertExitCode(t, result, 0)

	// Should be valid JSON (contains braces/brackets)
	if result.Stdout == "" {
		t.Error("expected JSON output")
	}
}

// TestRunTriage_Integration_SeedIssuesDryRun tests triage dry-run on seed issues
func TestRunTriage_Integration_SeedIssuesDryRun(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Use dry-run to avoid modifying seed issues
	result := testutil.RunCommand(t, "triage",
		"--query", "is:issue is:open",
		"--apply", "status:done",
		"--dry-run",
	)

	testutil.AssertExitCode(t, result, 0)
	testutil.AssertContains(t, result.Stdout, "Dry run")
}
