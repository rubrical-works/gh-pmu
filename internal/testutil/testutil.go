//go:build integration

// Package testutil provides utilities for integration tests against the GitHub API.
// All functions in this package require the integration build tag.
package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
)

var (
	testConfigDir     string
	testConfigDirOnce sync.Once
)

// getTestConfigDir finds the testdata/integration directory for test configs.
// This ensures integration tests use the test config, not the repo root config.
func getTestConfigDir() string {
	testConfigDirOnce.Do(func() {
		// Start from current directory and walk up to find repo root
		dir, err := os.Getwd()
		if err != nil {
			return
		}

		for {
			// Check for go.mod (repo root marker)
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				// Found repo root, use testdata/integration config
				testDir := filepath.Join(dir, "testdata", "integration")
				if _, err := os.Stat(filepath.Join(testDir, ".gh-pmu.yml")); err == nil {
					testConfigDir = testDir
					return
				}
				// Fallback to repo root if testdata config doesn't exist
				testConfigDir = dir
				return
			}

			// Move to parent directory
			parent := filepath.Dir(dir)
			if parent == dir {
				// Reached root without finding markers
				break
			}
			dir = parent
		}
	})
	return testConfigDir
}

// TestEnv holds environment configuration for integration tests
type TestEnv struct {
	ProjectOwner  string
	ProjectNumber int
	RepoOwner     string
	RepoName      string
	Token         string
	WorkDir       string
}

// RequireTestEnv skips the test if required environment variables are not set.
// Returns the test environment configuration.
func RequireTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	projectOwner := os.Getenv("TEST_PROJECT_OWNER")
	if projectOwner == "" {
		t.Skip("TEST_PROJECT_OWNER not set")
	}

	projectNumberStr := os.Getenv("TEST_PROJECT_NUMBER")
	if projectNumberStr == "" {
		t.Skip("TEST_PROJECT_NUMBER not set")
	}

	projectNumber, err := strconv.Atoi(projectNumberStr)
	if err != nil {
		t.Fatalf("TEST_PROJECT_NUMBER is not a valid integer: %v", err)
	}

	repoOwner := os.Getenv("TEST_REPO_OWNER")
	if repoOwner == "" {
		t.Skip("TEST_REPO_OWNER not set")
	}

	repoName := os.Getenv("TEST_REPO_NAME")
	if repoName == "" {
		t.Skip("TEST_REPO_NAME not set")
	}

	// Token is optional - will use gh auth token if not set
	token := os.Getenv("TEST_GH_TOKEN")

	return &TestEnv{
		ProjectOwner:  projectOwner,
		ProjectNumber: projectNumber,
		RepoOwner:     repoOwner,
		RepoName:      repoName,
		Token:         token,
		WorkDir:       getTestConfigDir(),
	}
}

// SetupTestClient creates an authenticated API client for integration tests.
// Uses TEST_GH_TOKEN if set, otherwise uses the default gh auth token.
func SetupTestClient(t *testing.T) *api.Client {
	t.Helper()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("failed to create API client: %v", err)
	}

	return client
}

// CleanupFunc is a function that cleans up test resources
type CleanupFunc func()

// CreateTestIssue creates a test issue and returns its number and a cleanup function.
// The cleanup function should be deferred to ensure the issue is deleted after the test.
func CreateTestIssue(t *testing.T, title string) (int, CleanupFunc) {
	t.Helper()

	env := RequireTestEnv(t)

	// Create issue using gh CLI
	cmd := exec.Command("gh", "issue", "create",
		"--repo", fmt.Sprintf("%s/%s", env.RepoOwner, env.RepoName),
		"--title", title,
		"--body", "Test issue created by integration test. Will be deleted automatically.",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to create test issue: %v\nOutput: %s", err, output)
	}

	// Extract issue number from URL
	issueNum := ExtractIssueNumber(t, string(output))

	cleanup := func() {
		DeleteTestIssue(t, issueNum)
	}

	return issueNum, cleanup
}

// DeleteTestIssue deletes a test issue by number.
func DeleteTestIssue(t *testing.T, issueNum int) {
	t.Helper()

	env := RequireTestEnv(t)

	cmd := exec.Command("gh", "issue", "close",
		"--repo", fmt.Sprintf("%s/%s", env.RepoOwner, env.RepoName),
		strconv.Itoa(issueNum),
		"--reason", "not planned",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Don't fail the test if cleanup fails, just log it
		t.Logf("Warning: failed to delete test issue #%d: %v\nOutput: %s", issueNum, err, output)
	}

	// Also try to delete the issue entirely (requires admin permissions)
	// This is best-effort since not all test accounts have admin access
	cmd = exec.Command("gh", "issue", "delete",
		"--repo", fmt.Sprintf("%s/%s", env.RepoOwner, env.RepoName),
		strconv.Itoa(issueNum),
		"--yes",
	)
	_ = cmd.Run() // Ignore errors - deletion may not be available
}

// GetProjectItem fetches a project item by issue number.
// Returns the project item with its field values.
func GetProjectItem(t *testing.T, issueNum int) *api.ProjectItem {
	t.Helper()

	env := RequireTestEnv(t)
	client := SetupTestClient(t)

	// Get the project
	project, err := client.GetProject(env.ProjectOwner, env.ProjectNumber)
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}

	// Get all project items
	items, err := client.GetProjectItems(project.ID, &api.ProjectItemsFilter{
		Repository: fmt.Sprintf("%s/%s", env.RepoOwner, env.RepoName),
	})
	if err != nil {
		t.Fatalf("failed to get project items: %v", err)
	}

	// Find the item for the issue
	for _, item := range items {
		if item.Issue != nil && item.Issue.Number == issueNum {
			return &item
		}
	}

	return nil
}

// CommandResult holds the result of running a gh pmu command
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// RunCommand executes a gh pmu command and returns the result.
// The command should not include "gh pmu" - just the subcommand and args.
// Example: RunCommand(t, "list", "--status", "backlog")
func RunCommand(t *testing.T, args ...string) *CommandResult {
	t.Helper()

	// Prepend "gh pmu" to the args
	fullArgs := append([]string{"pmu"}, args...)

	cmd := exec.Command("gh", fullArgs...)

	// Set working directory to test config dir so gh pmu uses test config
	if testDir := getTestConfigDir(); testDir != "" {
		cmd.Dir = testDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	return result
}

// RunCommandWithEnv executes a gh pmu command with custom environment variables.
func RunCommandWithEnv(t *testing.T, env map[string]string, args ...string) *CommandResult {
	t.Helper()

	fullArgs := append([]string{"pmu"}, args...)

	cmd := exec.Command("gh", fullArgs...)

	// Set working directory to test config dir so gh pmu uses test config
	if testDir := getTestConfigDir(); testDir != "" {
		cmd.Dir = testDir
	}

	// Copy current environment and add custom variables
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	return result
}

// ExtractIssueNumber parses an issue number from command output.
// Handles both URL format (https://github.com/owner/repo/issues/123) and plain number.
func ExtractIssueNumber(t *testing.T, output string) int {
	t.Helper()

	// Try URL format first
	urlPattern := regexp.MustCompile(`/issues/(\d+)`)
	if matches := urlPattern.FindStringSubmatch(output); len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return num
		}
	}

	// Try plain number format (e.g., "Created issue #123" or just "123")
	numberPattern := regexp.MustCompile(`#?(\d+)`)
	if matches := numberPattern.FindStringSubmatch(output); len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return num
		}
	}

	t.Fatalf("could not extract issue number from output: %s", output)
	return 0
}

// AssertExitCode asserts that the command result has the expected exit code.
func AssertExitCode(t *testing.T, result *CommandResult, expected int) {
	t.Helper()

	if result.ExitCode != expected {
		t.Errorf("expected exit code %d, got %d\nStdout: %s\nStderr: %s",
			expected, result.ExitCode, result.Stdout, result.Stderr)
	}
}

// AssertContains asserts that the output contains the expected substring.
func AssertContains(t *testing.T, output, expected string) {
	t.Helper()

	if !strings.Contains(output, expected) {
		t.Errorf("expected output to contain %q, got: %s", expected, output)
	}
}

// AssertNotContains asserts that the output does not contain the substring.
func AssertNotContains(t *testing.T, output, notExpected string) {
	t.Helper()

	if strings.Contains(output, notExpected) {
		t.Errorf("expected output to not contain %q, got: %s", notExpected, output)
	}
}

// GetTestRepo returns the full repository name (owner/repo) for tests.
func (env *TestEnv) GetTestRepo() string {
	return fmt.Sprintf("%s/%s", env.RepoOwner, env.RepoName)
}

// GetProjectID returns the project ID for the test project.
func (env *TestEnv) GetProjectID(t *testing.T) string {
	t.Helper()

	client := SetupTestClient(t)
	project, err := client.GetProject(env.ProjectOwner, env.ProjectNumber)
	if err != nil {
		t.Fatalf("failed to get project ID: %v", err)
	}

	return project.ID
}

// AddIssueToProject adds an issue to the test project.
func AddIssueToProject(t *testing.T, issueNum int) string {
	t.Helper()

	env := RequireTestEnv(t)
	client := SetupTestClient(t)

	// Get issue ID
	issue, err := client.GetIssue(env.RepoOwner, env.RepoName, issueNum)
	if err != nil {
		t.Fatalf("failed to get issue: %v", err)
	}

	// Get project ID
	projectID := env.GetProjectID(t)

	// Add to project
	itemID, err := client.AddIssueToProject(projectID, issue.ID)
	if err != nil {
		t.Fatalf("failed to add issue to project: %v", err)
	}

	return itemID
}

// SetIssueField sets a field value on an issue in the test project.
func SetIssueField(t *testing.T, issueNum int, field, value string) {
	t.Helper()

	env := RequireTestEnv(t)
	client := SetupTestClient(t)

	// Get project item
	item := GetProjectItem(t, issueNum)
	if item == nil {
		// Add to project first
		AddIssueToProject(t, issueNum)
		item = GetProjectItem(t, issueNum)
		if item == nil {
			t.Fatalf("failed to find project item for issue #%d", issueNum)
		}
	}

	// Set field value
	projectID := env.GetProjectID(t)
	err := client.SetProjectItemField(projectID, item.ID, field, value)
	if err != nil {
		t.Fatalf("failed to set field %s to %s: %v", field, value, err)
	}
}
