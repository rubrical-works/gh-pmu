//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// CommandResult holds the result of running a command
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runPMU executes the local binary with the given arguments.
// The command runs in the specified working directory (typically a temp dir with config).
// Returns stdout on success, or logs stderr and fails the test on error.
func runPMU(t *testing.T, workDir string, args ...string) *CommandResult {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		// Log stderr on command failure for debugging
		if result.Stderr != "" {
			t.Logf("Command stderr: %s", result.Stderr)
		}
	}

	return result
}

// assertContains checks that the output contains the expected substring.
// Fails the test if the substring is not found.
func assertContains(t *testing.T, output, expected string) {
	t.Helper()

	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain %q\nGot: %s", expected, output)
	}
}

// assertNotContains checks that the output does not contain the substring.
// Fails the test if the substring is found.
func assertNotContains(t *testing.T, output, notExpected string) {
	t.Helper()

	if strings.Contains(output, notExpected) {
		t.Errorf("Expected output to NOT contain %q\nGot: %s", notExpected, output)
	}
}

// assertExitCode checks that the command result has the expected exit code.
func assertExitCode(t *testing.T, result *CommandResult, expected int) {
	t.Helper()

	if result.ExitCode != expected {
		t.Errorf("Expected exit code %d, got %d\nStdout: %s\nStderr: %s",
			expected, result.ExitCode, result.Stdout, result.Stderr)
	}
}

// createTestIssue creates a test issue with the [E2E] prefix in the title.
// Returns the issue number and a cleanup function.
func createTestIssue(t *testing.T, cfg *TestConfig, title string) int {
	t.Helper()

	// Prefix with [E2E] for cleanup identification
	fullTitle := fmt.Sprintf("[E2E] %s", title)

	result := runPMU(t, cfg.Dir, "create",
		"--title", fullTitle,
		"--status", "backlog",
		"--priority", "p2",
	)

	if result.ExitCode != 0 {
		t.Fatalf("Failed to create test issue: %s\nStderr: %s", result.Stdout, result.Stderr)
	}

	issueNum := extractIssueNumber(t, result.Stdout)
	return issueNum
}

// extractIssueNumber parses an issue number from command output.
// Handles both URL format (https://github.com/owner/repo/issues/123)
// and plain formats like "Created issue #123" or just the number.
func extractIssueNumber(t *testing.T, output string) int {
	t.Helper()

	// Try URL format first
	urlPattern := regexp.MustCompile(`/issues/(\d+)`)
	if matches := urlPattern.FindStringSubmatch(output); len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return num
		}
	}

	// Try "#123" format
	hashPattern := regexp.MustCompile(`#(\d+)`)
	if matches := hashPattern.FindStringSubmatch(output); len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return num
		}
	}

	// Try plain number at end of line
	numberPattern := regexp.MustCompile(`(\d+)\s*$`)
	if matches := numberPattern.FindStringSubmatch(output); len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return num
		}
	}

	t.Fatalf("Could not extract issue number from output: %s", output)
	return 0
}

// waitForProjectSync waits for GitHub's eventual consistency by retrying
// a command until the expected content appears or timeout is reached.
// Uses 2-second intervals between retries for GitHub API eventual consistency.
func waitForProjectSync(t *testing.T, cfg *TestConfig, maxRetries int, args []string, expectedContent string) *CommandResult {
	t.Helper()

	for i := 0; i < maxRetries; i++ {
		result := runPMU(t, cfg.Dir, args...)
		if result.ExitCode == 0 && strings.Contains(result.Stdout, expectedContent) {
			return result
		}
		if i < maxRetries-1 {
			t.Logf("Retry %d/%d: waiting for project sync...", i+1, maxRetries)
			// Sleep 2 seconds between retries for GitHub eventual consistency
			time.Sleep(2 * time.Second)
		}
	}

	// Return last result even if it doesn't contain expected content
	return runPMU(t, cfg.Dir, args...)
}

// deleteTestIssue closes and deletes a test issue.
// Uses gh issue commands directly since we need to delete from the repository.
func deleteTestIssue(t *testing.T, issueNum int) {
	t.Helper()

	repo := "rubrical-works/gh-pmu-e2e-test"

	// Close the issue first
	closeCmd := exec.Command("gh", "issue", "close",
		"--repo", repo,
		strconv.Itoa(issueNum),
		"--reason", "not planned",
	)
	output, err := closeCmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: failed to close test issue #%d: %v\nOutput: %s", issueNum, err, output)
	}

	// Try to delete (requires admin permissions)
	deleteCmd := exec.Command("gh", "issue", "delete",
		"--repo", repo,
		strconv.Itoa(issueNum),
		"--yes",
	)
	_ = deleteCmd.Run() // Ignore errors - deletion may not be available
}

// getIssueLabels retrieves labels for an issue using gh CLI.
// Returns a slice of label names.
func getIssueLabels(t *testing.T, issueNum int) []string {
	t.Helper()

	repo := "rubrical-works/gh-pmu-e2e-test"
	cmd := exec.Command("gh", "issue", "view",
		"--repo", repo,
		strconv.Itoa(issueNum),
		"--json", "labels",
		"--jq", ".labels[].name",
	)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get labels for issue #%d: %v", issueNum, err)
	}
	labelsStr := strings.TrimSpace(string(output))
	if labelsStr == "" {
		return []string{}
	}
	return strings.Split(labelsStr, "\n")
}

// assertHasLabel verifies an issue has a specific label.
// Fails the test if the label is not found.
func assertHasLabel(t *testing.T, issueNum int, expectedLabel string) {
	t.Helper()

	labels := getIssueLabels(t, issueNum)
	for _, label := range labels {
		if label == expectedLabel {
			return
		}
	}
	t.Errorf("Issue #%d missing expected label %q, has: %v", issueNum, expectedLabel, labels)
}

// extractProjectNumber parses a project number from init command output.
// Handles format: "Created project: name (#NN)"
func extractProjectNumber(t *testing.T, output string) int {
	t.Helper()

	pattern := regexp.MustCompile(`#(\d+)\)`)
	if matches := pattern.FindStringSubmatch(output); len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return num
		}
	}
	return 0
}

// deleteTestProject deletes a GitHub project created during testing.
func deleteTestProject(t *testing.T, projectNumber int) {
	t.Helper()

	if projectNumber <= 0 {
		return
	}

	cmd := exec.Command("gh", "project", "delete",
		strconv.Itoa(projectNumber),
		"--owner", "rubrical-works",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("Warning: failed to delete test project #%d: %v\nOutput: %s", projectNumber, err, output)
	} else {
		t.Logf("Deleted test project #%d", projectNumber)
	}
}

// TestBranchInfo holds information about a test branch
type TestBranchInfo struct {
	Name         string
	TrackerIssue int
}

// setupTestBranch creates a test branch for tests that need to move issues.
// Returns branch info and a cleanup function. The cleanup function should be
// deferred to ensure the branch is closed even if the test fails.
func setupTestBranch(t *testing.T, cfg *TestConfig) (*TestBranchInfo, func()) {
	t.Helper()

	// Generate unique branch name with timestamp
	branchName := fmt.Sprintf("release/e2e-%d", time.Now().UnixNano())

	// Start the branch
	result := runPMU(t, cfg.Dir, "branch", "start", "--name", branchName)
	if result.ExitCode != 0 {
		t.Fatalf("Failed to start test branch: %s\nStderr: %s", result.Stdout, result.Stderr)
	}

	trackerNum := extractIssueNumber(t, result.Stdout)

	info := &TestBranchInfo{
		Name:         branchName,
		TrackerIssue: trackerNum,
	}

	cleanup := func() {
		// Close the branch
		runPMU(t, cfg.Dir, "branch", "close", "--yes")
		// Clean up tracker issue
		if trackerNum > 0 {
			runCleanupAfterTest(t, trackerNum)
		}
	}

	return info, cleanup
}

// assignIssueToBranch assigns an issue to the current branch.
// This is required before moving issues to in_progress when IDPF validation is enabled.
func assignIssueToBranch(t *testing.T, cfg *TestConfig, issueNum int) {
	t.Helper()

	result := runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", issueNum), "--branch", "current")
	if result.ExitCode != 0 {
		t.Fatalf("Failed to assign issue #%d to branch: %s\nStderr: %s", issueNum, result.Stdout, result.Stderr)
	}
}
