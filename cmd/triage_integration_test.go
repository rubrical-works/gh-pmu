//go:build integration

package cmd

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/testutil"
)

// assertDryRunLeftIssueUnchanged asserts that a --dry-run triage sweep did not
// touch the issue's Status or Priority.
//
// The generic read-back helpers this used to define now live in
// internal/testutil so all five integration files share one implementation
// (#891).
func assertDryRunLeftIssueUnchanged(t *testing.T, issueNum int, wantStatus, wantPriority string) {
	t.Helper()

	fields := testutil.ViewIssueFields(t, issueNum, "number,status,priority")
	if fields["status"] != wantStatus {
		t.Errorf("dry-run mutated status on #%d: expected %q, got %v", issueNum, wantStatus, fields["status"])
	}
	if fields["priority"] != wantPriority {
		t.Errorf("dry-run mutated priority on #%d: expected %q, got %v", issueNum, wantPriority, fields["priority"])
	}
}

// TestRunTriage_Integration_ListConfigs tests --list shows available configs
func TestRunTriage_Integration_ListConfigs(t *testing.T) {
	testutil.RequireTestEnv(t)

	result := testutil.RunCommand(t, "triage", "--list")

	testutil.AssertExitCode(t, result, 0)

	// Should show available configs from .gh-pmu.json
	testutil.AssertContains(t, result.Stdout, "Available triage configs")
}

// integrationTriageConfig is a triage config seeded in testdata/integration/.gh-pmu.json
// specifically for these tests. It matches every open issue and applies priority:p2,
// so the named-config path is deterministic instead of depending on whatever configs
// happen to exist in the project.
const integrationTriageConfig = "integration-dryrun"

// TestRunTriage_Integration_NamedConfigDryRun tests running a named config with --dry-run
func TestRunTriage_Integration_NamedConfigDryRun(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create an issue that the seeded config's query ("is:issue is:open") matches
	title := fmt.Sprintf("Test Issue - TriageConfig - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// The config is seeded in testdata, so a broken named-config path is a failure —
	// not an "acceptable outcome".
	result := testutil.RunCommand(t, "triage", integrationTriageConfig, "--dry-run")

	testutil.AssertExitCode(t, result, 0)
	testutil.AssertContains(t, result.Stdout, "Would process")
	testutil.AssertContains(t, result.Stdout, integrationTriageConfig)

	// The created issue must appear in the dry-run preview — otherwise the setup
	// above is dead weight and the query was never really applied.
	testutil.AssertContains(t, result.Stdout, fmt.Sprintf("#%d", issueNum))
	testutil.AssertContains(t, result.Stdout, title)

	// The config's apply rules must be described, resolved through cfg.Fields.
	testutil.AssertContains(t, result.Stdout, "Actions to apply:")
	testutil.AssertContains(t, result.Stdout, "Set priority: P2")
}

// TestRunTriage_Integration_AdHocQueryDryRun tests ad-hoc triage with --query and --dry-run
func TestRunTriage_Integration_AdHocQueryDryRun(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create an issue to match. Priority is pinned to P2 and the dry-run below
	// applies P0, so a dry-run that actually mutates is observable.
	title := fmt.Sprintf("Test Issue - TriageAdHoc - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog", "--priority", "p2")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Run ad-hoc triage with dry-run
	result := testutil.RunCommand(t, "triage",
		"--query", "is:issue is:open",
		"--apply", "priority:p0",
		"--dry-run",
	)

	testutil.AssertExitCode(t, result, 0)
	testutil.AssertContains(t, result.Stdout, "Would process")
	testutil.AssertContains(t, result.Stdout, fmt.Sprintf("#%d", issueNum))
	testutil.AssertContains(t, result.Stdout, "Actions to apply:")
	testutil.AssertContains(t, result.Stdout, "Set priority: P0")

	// The whole point of --dry-run: nothing changed.
	assertDryRunLeftIssueUnchanged(t, issueNum, "Backlog", "P2")
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
	testutil.AssertIssueField(t, issueNum, "priority", "P1")
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
	testutil.AssertIssueHasLabel(t, issueNum, "bug")
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

	// Pinned behavior: cobra returns the RunE error, main exits 1, and the
	// message plus usage go to stderr. Anything else is a regression.
	testutil.AssertExitCode(t, result, 1)
	testutil.AssertContains(t, result.Stderr, "triage config name is required")
	testutil.AssertContains(t, result.Stderr, "Use --list to see available configs, or use --query for ad-hoc triage")
	testutil.AssertContains(t, result.Stderr, "Usage:")

	// Nothing is written to stdout, so `--json`-style consumers piping stdout
	// never see a half-formed payload.
	if result.Stdout != "" {
		t.Errorf("expected empty stdout, got: %s", result.Stdout)
	}
}

// TestRunTriage_Integration_JSONOutput tests --json output format
func TestRunTriage_Integration_JSONOutput(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create an issue to triage
	title := fmt.Sprintf("Test Issue - TriageJSON - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog", "--priority", "p2")
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

	// A plain-text error dump on stdout must not pass — decode it.
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &out); err != nil {
		t.Fatalf("failed to parse triage JSON output: %v\nOutput: %s", err, result.Stdout)
	}

	if out["status"] != "completed" {
		t.Errorf("expected status \"completed\", got %v", out["status"])
	}
	if out["configName"] != "ad-hoc" {
		t.Errorf("expected configName \"ad-hoc\", got %v", out["configName"])
	}

	count, ok := out["count"].(float64)
	if !ok {
		t.Fatalf("expected numeric count, got %v", out["count"])
	}

	issues, ok := out["issues"].([]interface{})
	if !ok {
		t.Fatalf("expected issues to be an array, got %v", out["issues"])
	}
	if len(issues) != int(count) {
		t.Errorf("count %d disagrees with %d issues in the array", int(count), len(issues))
	}

	// The triaged issue must be reported in the payload with its real fields.
	found := false
	for _, raw := range issues {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			t.Fatalf("expected issue entries to be objects, got %v", raw)
		}
		num, ok := entry["number"].(float64)
		if !ok || int(num) != issueNum {
			continue
		}
		found = true
		if entry["title"] != title {
			t.Errorf("expected title %q in JSON payload, got %v", title, entry["title"])
		}
		if entry["state"] != "OPEN" {
			t.Errorf("expected state \"OPEN\" in JSON payload, got %v", entry["state"])
		}
		if _, ok := entry["url"].(string); !ok {
			t.Errorf("expected a url string in JSON payload, got %v", entry["url"])
		}
	}
	if !found {
		t.Errorf("issue #%d missing from triage JSON payload: %s", issueNum, result.Stdout)
	}

	// --apply is a real mutation: verify it landed rather than trusting the report.
	testutil.AssertIssueField(t, issueNum, "priority", "P0")
}

// TestRunTriage_Integration_SeedIssuesDryRun tests triage dry-run on seed issues
func TestRunTriage_Integration_SeedIssuesDryRun(t *testing.T) {
	testutil.RequireTestEnv(t)

	// A canary issue that the broad sweep below matches alongside the seed
	// issues. If the dry-run mutates anything, it mutates this too — and unlike
	// the shared seed fixtures, this one has a known starting state.
	title := fmt.Sprintf("Test Issue - TriageSeedDryRun - %d", testUniqueID())
	createResult := testutil.RunCommand(t, "create", "--title", title, "--status", "backlog", "--priority", "p2")
	testutil.AssertExitCode(t, createResult, 0)

	issueNum := testutil.ExtractIssueNumber(t, createResult.Stdout)
	defer testutil.DeleteTestIssue(t, issueNum)

	// Use dry-run to avoid modifying seed issues
	result := testutil.RunCommand(t, "triage",
		"--query", "is:issue is:open",
		"--apply", "status:done",
		"--dry-run",
	)

	testutil.AssertExitCode(t, result, 0)
	testutil.AssertContains(t, result.Stdout, "Would process")
	testutil.AssertContains(t, result.Stdout, fmt.Sprintf("#%d", issueNum))
	testutil.AssertContains(t, result.Stdout, "Set status: Done")

	// No issue in the sweep may have been moved to Done.
	assertDryRunLeftIssueUnchanged(t, issueNum, "Backlog", "P2")
}
