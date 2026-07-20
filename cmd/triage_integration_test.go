//go:build integration

package cmd

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/testutil"
)

// viewIssueFields reads the requested project fields back off an issue via
// `gh pmu view --json=<fields>` and returns the decoded object. Reading state
// back is what makes a mutation (or the absence of one, under --dry-run)
// observable to a test.
func viewIssueFields(t *testing.T, issueNum int, fields string) map[string]interface{} {
	t.Helper()

	result := testutil.RunCommand(t, "view", fmt.Sprintf("%d", issueNum), fmt.Sprintf("--json=%s", fields))
	testutil.AssertExitCode(t, result, 0)

	var out map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &out); err != nil {
		t.Fatalf("failed to parse view JSON for #%d: %v\nOutput: %s", issueNum, err, result.Stdout)
	}

	return out
}

// assertIssueField asserts that a single project field on an issue has the
// expected value.
func assertIssueField(t *testing.T, issueNum int, field, want string) {
	t.Helper()

	got := viewIssueFields(t, issueNum, fmt.Sprintf("number,%s", field))[field]
	if got != want {
		t.Errorf("issue #%d: expected %s %q, got %v", issueNum, field, want, got)
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
	assertIssueField(t, issueNum, "priority", "P0")
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
