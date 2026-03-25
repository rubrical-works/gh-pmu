package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestConfigVerify_NoConfig_ReturnsError(t *testing.T) {
	// ARRANGE: Empty dir with no config
	dir := t.TempDir()

	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"config", "verify", "--dir", dir})

	// ACT
	err := cmd.Execute()

	// ASSERT
	if err == nil {
		t.Fatal("Expected error when no config file exists")
	}
}

func TestConfigVerify_CleanConfig_ReportsNoIssues(t *testing.T) {
	// ARRANGE: Create a temp dir with a config and init a git repo
	dir := t.TempDir()
	configContent := []byte(`{"project":{"owner":"test","number":1},"repositories":["test/repo"]}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Init git repo and commit config
	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"config", "verify", "--dir", dir})

	// ACT
	err := cmd.Execute()

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v\nOutput: %s", err, buf.String())
	}
	output := buf.String()
	if !containsStr(output, "No drift detected") {
		t.Errorf("Expected 'No drift detected' in output, got: %s", output)
	}
}

func TestConfigVerify_DriftedConfig_ReportsChanges(t *testing.T) {
	// ARRANGE: Create config, commit, then modify
	dir := t.TempDir()
	original := []byte(`{"project":{"owner":"original","number":1},"repositories":["test/repo"]}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	// Modify config locally
	modified := []byte(`{"project":{"owner":"changed","number":1},"repositories":["test/repo"]}`)
	if err := os.WriteFile(configPath, modified, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"config", "verify", "--dir", dir})

	// ACT
	err := cmd.Execute()

	// ASSERT: Should report drift (not error — verify reports findings)
	if err != nil {
		t.Fatalf("Expected no error (drift is reported, not errored), got: %v", err)
	}
	output := buf.String()
	if !containsStr(output, "Drift detected") {
		t.Errorf("Expected 'Drift detected' in output, got: %s", output)
	}
	if !containsStr(output, "project.owner") {
		t.Errorf("Expected change detail mentioning 'project.owner', got: %s", output)
	}
	// Verify unchanged sections are shown
	if !containsStr(output, "Unchanged:") {
		t.Errorf("Expected 'Unchanged:' section in drift report, got: %s", output)
	}
	if !containsStr(output, "repositories") {
		t.Errorf("Expected 'repositories' in unchanged list, got: %s", output)
	}
	// Verify changed/unchanged visual distinction
	if !containsStr(output, "Changed:") {
		t.Errorf("Expected 'Changed:' header in drift report, got: %s", output)
	}
}

func TestConfigVerify_StrictMode_ErrorsOnDrift(t *testing.T) {
	// ARRANGE: Config with strict integrity setting, drifted
	dir := t.TempDir()
	original := []byte(`{"project":{"owner":"original","number":1},"repositories":["test/repo"],"configIntegrity":"strict"}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	// Modify config
	modified := []byte(`{"project":{"owner":"changed","number":1},"repositories":["test/repo"],"configIntegrity":"strict"}`)
	if err := os.WriteFile(configPath, modified, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"config", "verify", "--dir", dir})

	// ACT
	err := cmd.Execute()

	// ASSERT: Strict mode returns error on drift
	if err == nil {
		t.Fatal("Expected error in strict mode with drift")
	}
}

// runGit is a test helper to run git commands in a directory.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func containsStr(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
