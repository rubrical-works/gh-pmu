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

func TestConfigVerify_CriticalFieldChange_SingleField(t *testing.T) {
	dir := t.TempDir()
	original := []byte(`{"project":{"owner":"test","number":1},"repositories":["test/repo"]}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	// Change only project.number
	modified := []byte(`{"project":{"owner":"test","number":42},"repositories":["test/repo"]}`)
	if err := os.WriteFile(configPath, modified, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"config", "verify", "--dir", dir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Alert should be on stderr
	errOutput := stderr.String()
	if !containsStr(errOutput, "CRITICAL CONFIG CHANGE DETECTED") {
		t.Errorf("Expected critical alert on stderr, got: %s", errOutput)
	}
	if !containsStr(errOutput, "project.number") {
		t.Errorf("Expected 'project.number' in alert, got: %s", errOutput)
	}
	if !containsStr(errOutput, "1") || !containsStr(errOutput, "42") {
		t.Errorf("Expected old (1) and new (42) values in alert, got: %s", errOutput)
	}
	// Should NOT mention owner or repositories since those didn't change
	if containsStr(errOutput, "project.owner") {
		t.Errorf("Should not mention project.owner when it didn't change")
	}

	// Stdout should still have the normal drift report
	if !containsStr(stdout.String(), "Drift detected") {
		t.Errorf("Expected drift report on stdout")
	}
}

func TestConfigVerify_CriticalFieldChange_MultipleFields(t *testing.T) {
	dir := t.TempDir()
	original := []byte(`{"project":{"owner":"old-owner","number":1},"repositories":["old/repo"]}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	modified := []byte(`{"project":{"owner":"new-owner","number":99},"repositories":["new/repo"]}`)
	if err := os.WriteFile(configPath, modified, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"config", "verify", "--dir", dir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	errOutput := stderr.String()
	if !containsStr(errOutput, "project.owner") {
		t.Errorf("Expected 'project.owner' in alert")
	}
	if !containsStr(errOutput, "project.number") {
		t.Errorf("Expected 'project.number' in alert")
	}
	if !containsStr(errOutput, "repositories[0]") {
		t.Errorf("Expected 'repositories[0]' in alert")
	}
}

func TestConfigVerify_NoCriticalChange_WithGeneralDrift(t *testing.T) {
	dir := t.TempDir()
	original := []byte(`{"project":{"owner":"test","number":1},"repositories":["test/repo"],"defaults":{"status":"backlog"}}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	// Only change defaults (non-critical)
	modified := []byte(`{"project":{"owner":"test","number":1},"repositories":["test/repo"],"defaults":{"status":"ready"}}`)
	if err := os.WriteFile(configPath, modified, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"config", "verify", "--dir", dir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should report general drift on stdout
	if !containsStr(stdout.String(), "Drift detected") {
		t.Errorf("Expected drift report on stdout")
	}

	// Should NOT have critical alert on stderr
	if containsStr(stderr.String(), "CRITICAL CONFIG CHANGE DETECTED") {
		t.Errorf("Should not show critical alert when only non-critical fields changed")
	}
}

func TestConfigVerify_StrictMode_CriticalDrift_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	original := []byte(`{"project":{"owner":"test","number":1},"repositories":["test/repo"],"configIntegrity":"strict"}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	modified := []byte(`{"project":{"owner":"changed","number":1},"repositories":["test/repo"],"configIntegrity":"strict"}`)
	if err := os.WriteFile(configPath, modified, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"config", "verify", "--dir", dir})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error in strict mode with critical drift")
	}

	// Should also have the critical alert on stderr
	if !containsStr(stderr.String(), "CRITICAL CONFIG CHANGE DETECTED") {
		t.Errorf("Expected critical alert on stderr in strict mode")
	}
}
