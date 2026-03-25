//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// testConfig holds the configuration for E2E test project #41
const testConfig = `{
  "project": {
    "name": "IDPF-gh-pmu-testing",
    "number": 41,
    "owner": "rubrical-works"
  },
  "framework": "IDPF-Agile",
  "repositories": ["rubrical-works/gh-pmu-e2e-test"],
  "defaults": {
    "priority": "p2",
    "status": "backlog"
  },
  "fields": {
    "priority": {
      "field": "Priority",
      "values": {"p0": "P0", "p1": "P1", "p2": "P2"}
    },
    "status": {
      "field": "Status",
      "values": {"backlog": "Backlog", "done": "Done", "in_progress": "In progress", "in_review": "In review", "ready": "Ready"}
    },
    "release": {
      "field": "Release"
    }
  }
}`

// TestConfig holds the paths for a test configuration
type TestConfig struct {
	// Dir is the temporary directory containing the config
	Dir string
	// ConfigPath is the full path to the .gh-pmu.json file
	ConfigPath string
}

// setupTestConfig creates a temporary directory with a .gh-pmu.json file
// configured for test project #41. The directory is automatically cleaned up
// when the test completes. Also initializes a git repository for tests that
// require git operations (like branch commands).
func setupTestConfig(t *testing.T) *TestConfig {
	t.Helper()

	// Create temp directory - automatically cleaned up by t.TempDir()
	tmpDir := t.TempDir()

	// Write config file
	configPath := filepath.Join(tmpDir, ".gh-pmu.json")
	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Initialize git repo for tests that require git operations
	initGitRepo(t, tmpDir)

	return &TestConfig{
		Dir:        tmpDir,
		ConfigPath: configPath,
	}
}

// initGitRepo initializes a git repository in the given directory
// with minimal configuration for testing.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	// Run git init
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for commits (required for some git operations)
	cmd = exec.Command("git", "config", "user.email", "test@e2e.local")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "E2E Test")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git name: %v", err)
	}

	// Create initial commit (some git operations require at least one commit)
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# E2E Test Repository\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
}
