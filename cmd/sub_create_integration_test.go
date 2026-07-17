//go:build integration

package cmd

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/testutil"
)

// TestRunSubCreate_Integration_WithTitle tests creating a sub-issue with --title
func TestRunSubCreate_Integration_WithTitle(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create parent issue
	parentTitle := fmt.Sprintf("Test SubCreate Parent - Title - %d", subCreateTestID())
	parentResult := testutil.RunCommand(t, "create", "--title", parentTitle)
	testutil.AssertExitCode(t, parentResult, 0)

	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	// Create sub-issue
	subTitle := fmt.Sprintf("Test SubCreate Child - Title - %d", subCreateTestID())
	result := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", subTitle,
	)
	testutil.AssertExitCode(t, result, 0)

	// Verify success message
	testutil.AssertContains(t, result.Stdout, "Created sub-issue")
	testutil.AssertContains(t, result.Stdout, subTitle)

	// Extract sub-issue number for cleanup
	subNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, subNum)

	// Verify the sub-issue is linked
	listResult := testutil.RunCommand(t, "sub", "list", fmt.Sprintf("%d", parentNum))
	testutil.AssertExitCode(t, listResult, 0)
	testutil.AssertContains(t, listResult.Stdout, subTitle)
}

// TestRunSubCreate_Integration_WithTitleAndBody tests creating sub-issue with --title and --body
func TestRunSubCreate_Integration_WithTitleAndBody(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create parent issue
	parentTitle := fmt.Sprintf("Test SubCreate Parent - TitleBody - %d", subCreateTestID())
	parentResult := testutil.RunCommand(t, "create", "--title", parentTitle)
	testutil.AssertExitCode(t, parentResult, 0)

	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	// Create sub-issue with body
	subTitle := fmt.Sprintf("Test SubCreate Child - TitleBody - %d", subCreateTestID())
	subBody := "This is the body of the sub-issue created by integration test."

	result := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", subTitle,
		"--body", subBody,
	)
	testutil.AssertExitCode(t, result, 0)

	// Extract sub-issue number for cleanup
	subNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, subNum)

	// Verify the sub-issue body via view command
	viewResult := testutil.RunCommand(t, "view", fmt.Sprintf("%d", subNum))
	testutil.AssertExitCode(t, viewResult, 0)
	testutil.AssertContains(t, viewResult.Stdout, subBody)
}

// TestRunSubCreate_Integration_InheritLabels tests --inherit-labels=true
func TestRunSubCreate_Integration_InheritLabels(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Create parent issue with a label
	parentTitle := fmt.Sprintf("Test SubCreate Parent - InheritLabels - %d", subCreateTestID())
	parentResult := testutil.RunCommand(t, "create",
		"--title", parentTitle,
		"--label", "bug",
	)
	testutil.AssertExitCode(t, parentResult, 0)

	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	// Create sub-issue with --inherit-labels=true. The flag defaults to false, so it
	// must be set explicitly for inheritance to occur.
	subTitle := fmt.Sprintf("Test SubCreate Child - InheritLabels - %d", subCreateTestID())
	result := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", subTitle,
		"--inherit-labels=true",
	)
	testutil.AssertExitCode(t, result, 0)

	// Extract sub-issue number for cleanup
	subNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, subNum)

	// Assert the sub-issue's REAL labels include the inherited parent label, read via
	// `gh issue view --json labels` rather than the command's echoed stdout. The old
	// stdout check passed on the command's own output even if the label was never
	// applied to the created issue.
	labels := issueLabels(t, env, subNum)
	if !containsExact(labels, "bug") {
		t.Errorf("expected sub-issue #%d to inherit parent label %q, got labels: %v", subNum, "bug", labels)
	}
}

// TestRunSubCreate_Integration_NoInheritLabels tests --inherit-labels=false
func TestRunSubCreate_Integration_NoInheritLabels(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Create parent issue with a label
	parentTitle := fmt.Sprintf("Test SubCreate Parent - NoInheritLabels - %d", subCreateTestID())
	parentResult := testutil.RunCommand(t, "create",
		"--title", parentTitle,
		"--label", "bug",
	)
	testutil.AssertExitCode(t, parentResult, 0)

	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	// Create sub-issue with --inherit-labels=false
	subTitle := fmt.Sprintf("Test SubCreate Child - NoInheritLabels - %d", subCreateTestID())
	result := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", subTitle,
		"--inherit-labels=false",
	)
	testutil.AssertExitCode(t, result, 0)

	// Extract sub-issue number for cleanup
	subNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, subNum)

	// The parent's "bug" label must NOT appear on the sub-issue's real labels when
	// --inherit-labels=false. Reading the created issue's actual labels (not stdout)
	// fails if the flag is ignored and the parent label is inherited anyway.
	labels := issueLabels(t, env, subNum)
	if containsExact(labels, "bug") {
		t.Errorf("expected sub-issue #%d NOT to inherit parent label %q, got labels: %v", subNum, "bug", labels)
	}
}

// TestRunSubCreate_Integration_InheritAssignees tests --inherit-assignees=true
func TestRunSubCreate_Integration_InheritAssignees(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Create parent issue and assign the authenticated user so there is a real
	// assignee to inherit.
	parentTitle := fmt.Sprintf("Test SubCreate Parent - InheritAssignees - %d", subCreateTestID())
	parentResult := testutil.RunCommand(t, "create", "--title", parentTitle)
	testutil.AssertExitCode(t, parentResult, 0)

	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	login := ghCurrentLogin(t)
	assignUserToIssue(t, env, parentNum, login)

	// Create sub-issue with --inherit-assignees=true
	subTitle := fmt.Sprintf("Test SubCreate Child - InheritAssignees - %d", subCreateTestID())
	result := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", subTitle,
		"--inherit-assignees=true",
	)
	testutil.AssertExitCode(t, result, 0)

	// Extract sub-issue number for cleanup
	subNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, subNum)

	// The created sub-issue must actually carry the inherited assignee. Asserting on
	// the sub-issue's real assignees (not just exit 0) fails if --inherit-assignees
	// is ignored, since the parent previously had no assignees to inherit.
	if !containsExact(issueAssignees(t, env, subNum), login) {
		t.Errorf("expected sub-issue #%d to inherit assignee %q, got assignees: %v",
			subNum, login, issueAssignees(t, env, subNum))
	}
}

// TestRunSubCreate_Integration_InheritMilestone tests --inherit-milestone (default true)
func TestRunSubCreate_Integration_InheritMilestone(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Seed a repo milestone and attach it to the parent so there is one to inherit.
	msTitle := fmt.Sprintf("SubCreate Inherit Milestone %d", subCreateTestID())
	msNum := createMilestone(t, env, msTitle)
	defer deleteMilestone(t, env, msNum)

	parentTitle := fmt.Sprintf("Test SubCreate Parent - InheritMilestone - %d", subCreateTestID())
	parentResult := testutil.RunCommand(t, "create", "--title", parentTitle)
	testutil.AssertExitCode(t, parentResult, 0)

	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	setIssueMilestone(t, env, parentNum, msTitle)

	// Create sub-issue — --inherit-milestone defaults to true, so no flag is needed.
	subTitle := fmt.Sprintf("Test SubCreate Child - InheritMilestone - %d", subCreateTestID())
	result := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", subTitle,
	)
	testutil.AssertExitCode(t, result, 0)

	subNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, subNum)

	// The sub-issue must inherit the parent's milestone. `.milestone.title // ""`
	// yields "" when the sub-issue has no milestone, so a dropped inheritance fails.
	got := issueJSONField(t, env, subNum, "milestone", ".milestone.title // \"\"")
	if got != msTitle {
		t.Errorf("expected sub-issue #%d to inherit milestone %q, got %q", subNum, msTitle, got)
	}
}

// TestRunSubCreate_Integration_SubIssueLinkedToParent tests that sub-issue is properly linked
func TestRunSubCreate_Integration_SubIssueLinkedToParent(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create parent issue
	parentTitle := fmt.Sprintf("Test SubCreate Parent - Linked - %d", subCreateTestID())
	parentResult := testutil.RunCommand(t, "create", "--title", parentTitle)
	testutil.AssertExitCode(t, parentResult, 0)

	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	// Create sub-issue
	subTitle := fmt.Sprintf("Test SubCreate Child - Linked - %d", subCreateTestID())
	result := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", subTitle,
	)
	testutil.AssertExitCode(t, result, 0)

	subNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, subNum)

	// Verify output shows parent info
	testutil.AssertContains(t, result.Stdout, fmt.Sprintf("parent #%d", parentNum))

	// Verify link by checking sub list
	listResult := testutil.RunCommand(t, "sub", "list", fmt.Sprintf("%d", parentNum))
	testutil.AssertExitCode(t, listResult, 0)
	testutil.AssertContains(t, listResult.Stdout, subTitle)
	testutil.AssertContains(t, listResult.Stdout, fmt.Sprintf("#%d", subNum))

	// Verify via --relation parent from child's perspective
	parentRelation := testutil.RunCommand(t, "sub", "list", fmt.Sprintf("%d", subNum), "--relation", "parent")
	testutil.AssertExitCode(t, parentRelation, 0)
	testutil.AssertContains(t, parentRelation.Stdout, parentTitle)
}

// TestRunSubCreate_Integration_ParentNotFound tests error when parent doesn't exist
func TestRunSubCreate_Integration_ParentNotFound(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Try to create sub-issue for non-existent parent
	subTitle := fmt.Sprintf("Test SubCreate Child - ParentNotFound - %d", subCreateTestID())
	result := testutil.RunCommand(t, "sub", "create",
		"--parent", "999999",
		"--title", subTitle,
	)

	// Should fail
	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code for non-existent parent")
	}

	testutil.AssertContains(t, result.Stderr, "failed to get parent issue")
}

// TestRunSubCreate_Integration_OutputFormat tests output includes expected information
func TestRunSubCreate_Integration_OutputFormat(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create parent issue
	parentTitle := fmt.Sprintf("Test SubCreate Parent - Output - %d", subCreateTestID())
	parentResult := testutil.RunCommand(t, "create", "--title", parentTitle)
	testutil.AssertExitCode(t, parentResult, 0)

	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	// Create sub-issue
	subTitle := fmt.Sprintf("Test SubCreate Child - Output - %d", subCreateTestID())
	result := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", subTitle,
	)
	testutil.AssertExitCode(t, result, 0)

	subNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, subNum)

	// Verify output format
	testutil.AssertContains(t, result.Stdout, "Created sub-issue")
	testutil.AssertContains(t, result.Stdout, fmt.Sprintf("#%d", subNum))
	testutil.AssertContains(t, result.Stdout, "Title:")
	testutil.AssertContains(t, result.Stdout, subTitle)
	testutil.AssertContains(t, result.Stdout, "Parent:")
	testutil.AssertContains(t, result.Stdout, parentTitle)
	// Should include URL
	testutil.AssertContains(t, result.Stdout, "https://github.com/")
}

// TestRunSubCreate_Integration_MultipleSubIssues tests creating multiple sub-issues
func TestRunSubCreate_Integration_MultipleSubIssues(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create parent issue
	parentTitle := fmt.Sprintf("Test SubCreate Parent - Multiple - %d", subCreateTestID())
	parentResult := testutil.RunCommand(t, "create", "--title", parentTitle)
	testutil.AssertExitCode(t, parentResult, 0)

	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	// Create first sub-issue
	sub1Title := fmt.Sprintf("Test SubCreate Child1 - Multiple - %d", subCreateTestID())
	result1 := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", sub1Title,
	)
	testutil.AssertExitCode(t, result1, 0)

	sub1Num := testutil.ExtractIssueNumber(t, result1.Stdout)
	defer testutil.DeleteTestIssue(t, sub1Num)

	// Create second sub-issue
	sub2Title := fmt.Sprintf("Test SubCreate Child2 - Multiple - %d", subCreateTestID())
	result2 := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("%d", parentNum),
		"--title", sub2Title,
	)
	testutil.AssertExitCode(t, result2, 0)

	sub2Num := testutil.ExtractIssueNumber(t, result2.Stdout)
	defer testutil.DeleteTestIssue(t, sub2Num)

	// Verify both are linked
	listResult := testutil.RunCommand(t, "sub", "list", fmt.Sprintf("%d", parentNum))
	testutil.AssertExitCode(t, listResult, 0)
	testutil.AssertContains(t, listResult.Stdout, sub1Title)
	testutil.AssertContains(t, listResult.Stdout, sub2Title)
	testutil.AssertContains(t, listResult.Stdout, "0/2 complete")
}

// TestRunSubCreate_Integration_WithHashPrefix tests using # prefix on parent
func TestRunSubCreate_Integration_WithHashPrefix(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Create parent issue
	parentTitle := fmt.Sprintf("Test SubCreate Parent - Hash - %d", subCreateTestID())
	parentResult := testutil.RunCommand(t, "create", "--title", parentTitle)
	testutil.AssertExitCode(t, parentResult, 0)

	parentNum := testutil.ExtractIssueNumber(t, parentResult.Stdout)
	defer testutil.DeleteTestIssue(t, parentNum)

	// Create sub-issue using # prefix
	subTitle := fmt.Sprintf("Test SubCreate Child - Hash - %d", subCreateTestID())
	result := testutil.RunCommand(t, "sub", "create",
		"--parent", fmt.Sprintf("#%d", parentNum),
		"--title", subTitle,
	)
	testutil.AssertExitCode(t, result, 0)

	subNum := testutil.ExtractIssueNumber(t, result.Stdout)
	defer testutil.DeleteTestIssue(t, subNum)

	testutil.AssertContains(t, result.Stdout, "Created sub-issue")
}

// TestRunSubCreate_Integration_RequiredFlags tests that required flags are enforced
func TestRunSubCreate_Integration_RequiredFlags(t *testing.T) {
	testutil.RequireTestEnv(t)

	// Missing --title
	result1 := testutil.RunCommand(t, "sub", "create", "--parent", "123")
	if result1.ExitCode == 0 {
		t.Error("Expected non-zero exit code when --title is missing")
	}

	// Missing --parent
	result2 := testutil.RunCommand(t, "sub", "create", "--title", "Test")
	if result2.ExitCode == 0 {
		t.Error("Expected non-zero exit code when --parent is missing")
	}
}

// --- integration helpers for inheritance verification (#880) ---

// ghCurrentLogin returns the login of the authenticated gh user, used as a known
// assignee that can be applied to a parent and then checked on the sub-issue.
func ghCurrentLogin(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("gh", "api", "user", "--jq", ".login").Output()
	if err != nil {
		t.Fatalf("failed to resolve authenticated gh user: %v", err)
	}
	login := strings.TrimSpace(string(out))
	if login == "" {
		t.Fatal("authenticated gh user login was empty")
	}
	return login
}

// assignUserToIssue assigns a user to an issue via gh issue edit.
func assignUserToIssue(t *testing.T, env *testutil.TestEnv, issueNum int, login string) {
	t.Helper()
	out, err := exec.Command("gh", "issue", "edit", strconv.Itoa(issueNum),
		"--repo", env.GetTestRepo(), "--add-assignee", login).CombinedOutput()
	if err != nil {
		t.Fatalf("failed to assign %q to #%d: %v\nOutput: %s", login, issueNum, err, out)
	}
}

// issueJSONField returns a jq-extracted field from `gh issue view --json` for the
// created issue — i.e. its real GitHub state, not any command's echoed output.
func issueJSONField(t *testing.T, env *testutil.TestEnv, issueNum int, jsonField, jq string) string {
	t.Helper()
	out, err := exec.Command("gh", "issue", "view", strconv.Itoa(issueNum),
		"--repo", env.GetTestRepo(), "--json", jsonField, "--jq", jq).CombinedOutput()
	if err != nil {
		t.Fatalf("gh issue view #%d --json %s failed: %v\nOutput: %s", issueNum, jsonField, err, out)
	}
	return strings.TrimSpace(string(out))
}

// splitNonEmptyLines splits jq newline-delimited output into a slice, dropping the
// single empty element that TrimSpace leaves for no matches.
func splitNonEmptyLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// issueLabels returns the label names on an issue.
func issueLabels(t *testing.T, env *testutil.TestEnv, issueNum int) []string {
	t.Helper()
	return splitNonEmptyLines(issueJSONField(t, env, issueNum, "labels", ".labels[].name"))
}

// issueAssignees returns the assignee logins on an issue.
func issueAssignees(t *testing.T, env *testutil.TestEnv, issueNum int) []string {
	t.Helper()
	return splitNonEmptyLines(issueJSONField(t, env, issueNum, "assignees", ".assignees[].login"))
}

// containsExact reports whether list has an element equal to want.
func containsExact(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// createMilestone creates a repo milestone and returns its number.
func createMilestone(t *testing.T, env *testutil.TestEnv, title string) int {
	t.Helper()
	path := fmt.Sprintf("repos/%s/milestones", env.GetTestRepo())
	out, err := exec.Command("gh", "api", path, "-f", "title="+title, "--jq", ".number").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to create milestone %q: %v\nOutput: %s", title, err, out)
	}
	num, convErr := strconv.Atoi(strings.TrimSpace(string(out)))
	if convErr != nil {
		t.Fatalf("could not parse milestone number from %q: %v", out, convErr)
	}
	return num
}

// deleteMilestone removes a repo milestone by number (best-effort cleanup).
func deleteMilestone(t *testing.T, env *testutil.TestEnv, number int) {
	t.Helper()
	path := fmt.Sprintf("repos/%s/milestones/%d", env.GetTestRepo(), number)
	if out, err := exec.Command("gh", "api", "--method", "DELETE", path).CombinedOutput(); err != nil {
		t.Logf("warning: failed to delete milestone %d: %v\nOutput: %s", number, err, out)
	}
}

// setIssueMilestone sets an issue's milestone by title via gh issue edit.
func setIssueMilestone(t *testing.T, env *testutil.TestEnv, issueNum int, title string) {
	t.Helper()
	out, err := exec.Command("gh", "issue", "edit", strconv.Itoa(issueNum),
		"--repo", env.GetTestRepo(), "--milestone", title).CombinedOutput()
	if err != nil {
		t.Fatalf("failed to set milestone %q on #%d: %v\nOutput: %s", title, issueNum, err, out)
	}
}

// subCreateTestID returns a unique identifier for sub create test issues
var subCreateTestCounter int

func subCreateTestID() int {
	subCreateTestCounter++
	return subCreateTestCounter + int(strings.Count("sub-create-integration", "c")*1000000)
}
