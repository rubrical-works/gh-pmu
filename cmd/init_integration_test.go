//go:build integration

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"encoding/json"

	"github.com/rubrical-works/gh-pmu/internal/testutil"
)

// TestRunInit_Integration_NonInteractiveWithPipedInput tests init with piped stdin
// This simulates automated/scripted initialization
func TestRunInit_Integration_NonInteractiveWithPipedInput(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "gh-pmu-init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repo in temp dir (needed for repo detection)
	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set up git remote
	gitRemote := exec.Command("git", "remote", "add", "origin",
		fmt.Sprintf("https://github.com/%s/%s.git", env.RepoOwner, env.RepoName))
	gitRemote.Dir = tempDir
	if err := gitRemote.Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Run init with piped input
	// Input: project selection (1 for first project), repository confirm (empty for default)
	cmd := exec.Command("gh", "pmu", "init")
	cmd.Dir = tempDir
	cmd.Stdin = strings.NewReader("1\n\n") // Select project 1, confirm repo

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Logf("Init stdout: %s", stdout.String())
		t.Logf("Init stderr: %s", stderr.String())
		t.Fatalf("Init failed: %v", err)
	}

	// Verify config file was created
	configPath := filepath.Join(tempDir, ".gh-pmu.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Read and validate config structure
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify required sections exist
	if _, ok := config["project"]; !ok {
		t.Error("Config missing 'project' section")
	}
	if _, ok := config["repositories"]; !ok {
		t.Error("Config missing 'repositories' section")
	}
	if _, ok := config["fields"]; !ok {
		t.Error("Config missing 'fields' section")
	}
	if _, ok := config["metadata"]; !ok {
		t.Error("Config missing 'metadata' section")
	}
}

// TestRunInit_Integration_ProjectValidation tests that init validates project exists
func TestRunInit_Integration_ProjectValidation(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "gh-pmu-init-validation-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repo
	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set up git remote
	gitRemote := exec.Command("git", "remote", "add", "origin",
		fmt.Sprintf("https://github.com/%s/%s.git", env.RepoOwner, env.RepoName))
	gitRemote.Dir = tempDir
	if err := gitRemote.Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Run init - this tests that project fetching/validation works
	// Input: select first project (or manual entry 0 then valid project)
	cmd := exec.Command("gh", "pmu", "init")
	cmd.Dir = tempDir
	cmd.Stdin = strings.NewReader("1\n\n")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// The command should either succeed (found projects) or fail gracefully
	// We mainly want to ensure it doesn't crash and validates properly
	t.Logf("Init stdout: %s", stdout.String())
	t.Logf("Init stderr: %s", stderr.String())

	// Check for expected output patterns
	output := stdout.String()
	// Should mention fetching projects or finding projects
	if !strings.Contains(output, "Fetching projects") && !strings.Contains(output, "Found") {
		t.Error("Expected project discovery output")
	}
}

// TestRunInit_Integration_FieldMetadataFetching tests that field metadata is fetched
func TestRunInit_Integration_FieldMetadataFetching(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "gh-pmu-init-fields-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repo
	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set up git remote
	gitRemote := exec.Command("git", "remote", "add", "origin",
		fmt.Sprintf("https://github.com/%s/%s.git", env.RepoOwner, env.RepoName))
	gitRemote.Dir = tempDir
	if err := gitRemote.Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Run init
	cmd := exec.Command("gh", "pmu", "init")
	cmd.Dir = tempDir
	cmd.Stdin = strings.NewReader("1\n\n")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Skipf("Init failed (may not have access to projects): %v", err)
	}

	// Check that field fetching happened
	output := stdout.String()
	if !strings.Contains(output, "Fetching project fields") {
		t.Log("Warning: Field fetching message not found in output")
	}

	// Read config and check metadata section
	configPath := filepath.Join(tempDir, ".gh-pmu.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Check metadata section contains fields
	metadata, ok := config["metadata"].(map[string]interface{})
	if !ok {
		t.Error("Config metadata section is missing or invalid")
		return
	}

	fields, ok := metadata["fields"].([]interface{})
	if !ok {
		t.Error("Metadata fields section is missing")
		return
	}

	// Should have at least some fields (Status, Priority, etc.)
	if len(fields) == 0 {
		t.Error("Expected at least some field metadata")
	}
}

// TestRunInit_Integration_ConfigFileCreation tests the structure of created config
func TestRunInit_Integration_ConfigFileCreation(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "gh-pmu-init-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repo
	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set up git remote
	gitRemote := exec.Command("git", "remote", "add", "origin",
		fmt.Sprintf("https://github.com/%s/%s.git", env.RepoOwner, env.RepoName))
	gitRemote.Dir = tempDir
	if err := gitRemote.Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Run init
	cmd := exec.Command("gh", "pmu", "init")
	cmd.Dir = tempDir
	cmd.Stdin = strings.NewReader("1\n\n")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Skipf("Init failed (may not have access to projects): %v", err)
	}

	// Read and validate full config structure
	configPath := filepath.Join(tempDir, ".gh-pmu.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Check project section
	project, ok := config["project"].(map[string]interface{})
	if !ok {
		t.Error("Project section is invalid")
	} else {
		if project["owner"] == nil {
			t.Error("Project owner is missing")
		}
		if project["number"] == nil {
			t.Error("Project number is missing")
		}
	}

	// Check repositories section
	repos, ok := config["repositories"].([]interface{})
	if !ok {
		t.Error("Repositories section is invalid")
	} else if len(repos) == 0 {
		t.Error("No repositories configured")
	}

	// Check defaults section
	defaults, ok := config["defaults"].(map[string]interface{})
	if !ok {
		t.Error("Defaults section is invalid")
	} else {
		if defaults["status"] == nil {
			t.Error("Default status is missing")
		}
		if defaults["priority"] == nil {
			t.Error("Default priority is missing")
		}
	}

	// Check fields section (should have status and priority mappings)
	fields, ok := config["fields"].(map[string]interface{})
	if !ok {
		t.Error("Fields section is invalid")
	} else {
		if fields["status"] == nil {
			t.Error("Status field mapping is missing")
		}
		if fields["priority"] == nil {
			t.Error("Priority field mapping is missing")
		}
	}
}

// TestRunInit_Integration_ExistingConfigHandling tests overwrite behavior
func TestRunInit_Integration_ExistingConfigHandling(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "gh-pmu-init-existing-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repo
	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set up git remote
	gitRemote := exec.Command("git", "remote", "add", "origin",
		fmt.Sprintf("https://github.com/%s/%s.git", env.RepoOwner, env.RepoName))
	gitRemote.Dir = tempDir
	if err := gitRemote.Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Create an existing config file
	existingConfig := `{"project":{"name":"existing","owner":"test","number":999},"repositories":["test/repo"]}`
	existingPath := filepath.Join(tempDir, ".gh-pmu.json")
	if err := os.WriteFile(existingPath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("Failed to write existing config: %v", err)
	}

	// Run init and decline to overwrite (answer "n")
	cmd := exec.Command("gh", "pmu", "init")
	cmd.Dir = tempDir
	cmd.Stdin = strings.NewReader("n\n") // Don't overwrite

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	// Check that existing config was NOT overwritten
	configData, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Should still contain "existing" from original
	if !strings.Contains(string(configData), "existing") {
		t.Error("Expected existing config to be preserved when declining overwrite")
	}

	// Check output mentions abort
	output := stdout.String()
	if !strings.Contains(output, "Aborted") && !strings.Contains(output, "already exists") {
		t.Log("Expected abort message in output")
	}

	_ = env // satisfy unused warning
}

// TestRunInit_Integration_OutputMessages tests that expected messages are displayed
func TestRunInit_Integration_OutputMessages(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "gh-pmu-init-output-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repo
	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set up git remote
	gitRemote := exec.Command("git", "remote", "add", "origin",
		fmt.Sprintf("https://github.com/%s/%s.git", env.RepoOwner, env.RepoName))
	gitRemote.Dir = tempDir
	if err := gitRemote.Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Run init
	cmd := exec.Command("gh", "pmu", "init")
	cmd.Dir = tempDir
	cmd.Stdin = strings.NewReader("1\n\n")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	output := stdout.String()

	// Should show header
	if !strings.Contains(output, "gh-pmu init") {
		t.Error("Expected header in output")
	}

	// Should show detected repository
	if !strings.Contains(output, "Detected repository") {
		t.Error("Expected repository detection message")
	}

	// If successful, should show config saved
	if err == nil {
		if !strings.Contains(output, "Configuration saved") && !strings.Contains(output, "saved") {
			t.Log("Warning: Expected 'saved' message in successful output")
		}
	}

	_ = env // satisfy unused warning
}

// ============================================================================
// Non-Interactive Mode Integration Tests (Issue #609)
// ============================================================================

// TestRunInit_Integration_NonInteractiveMode tests init with --non-interactive flag
func TestRunInit_Integration_NonInteractiveMode(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "gh-pmu-init-noninteractive-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repo (not strictly needed for non-interactive, but good practice)
	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Run init in non-interactive mode
	cmd := exec.Command("gh", "pmu", "init",
		"--non-interactive",
		"--source-project", fmt.Sprintf("%d", env.ProjectNumber),
		"--repo", fmt.Sprintf("%s/%s", env.RepoOwner, env.RepoName))
	cmd.Dir = tempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Logf("Init stdout: %s", stdout.String())
		t.Logf("Init stderr: %s", stderr.String())
		t.Fatalf("Non-interactive init failed: %v", err)
	}

	// Verify config file was created
	configPath := filepath.Join(tempDir, ".gh-pmu.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Read and validate config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify required sections
	if _, ok := config["project"]; !ok {
		t.Error("Config missing 'project' section")
	}
	if _, ok := config["repositories"]; !ok {
		t.Error("Config missing 'repositories' section")
	}

	// Verify framework defaults to IDPF
	if framework, ok := config["framework"].(string); ok {
		if framework != "IDPF" {
			t.Errorf("Expected framework 'IDPF', got %q", framework)
		}
	}
}

// TestRunInit_Integration_NonInteractiveWithOwner tests --owner flag in non-interactive mode
func TestRunInit_Integration_NonInteractiveWithOwner(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	tempDir, err := os.MkdirTemp("", "gh-pmu-init-owner-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Run init with explicit --owner (same as repo owner in this test)
	cmd := exec.Command("gh", "pmu", "init",
		"--non-interactive",
		"--source-project", fmt.Sprintf("%d", env.ProjectNumber),
		"--repo", fmt.Sprintf("%s/%s", env.RepoOwner, env.RepoName),
		"--owner", env.RepoOwner)
	cmd.Dir = tempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Logf("Init stderr: %s", stderr.String())
		t.Fatalf("Non-interactive init with --owner failed: %v", err)
	}

	// Verify config was created
	configPath := filepath.Join(tempDir, ".gh-pmu.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

// TestRunInit_Integration_NonInteractiveFrameworkNone tests --framework none
func TestRunInit_Integration_NonInteractiveFrameworkNone(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	tempDir, err := os.MkdirTemp("", "gh-pmu-init-framework-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Run init with --framework none
	cmd := exec.Command("gh", "pmu", "init",
		"--non-interactive",
		"--source-project", fmt.Sprintf("%d", env.ProjectNumber),
		"--repo", fmt.Sprintf("%s/%s", env.RepoOwner, env.RepoName),
		"--framework", "none")
	cmd.Dir = tempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Logf("Init stderr: %s", stderr.String())
		t.Fatalf("Non-interactive init with --framework none failed: %v", err)
	}

	// Verify config was created with framework: none
	configPath := filepath.Join(tempDir, ".gh-pmu.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if framework, ok := config["framework"].(string); ok {
		if framework != "none" {
			t.Errorf("Expected framework 'none', got %q", framework)
		}
	} else {
		t.Error("Framework field not found in config")
	}
}

// TestRunInit_Integration_NonInteractiveOverwrite tests --yes flag for overwriting
func TestRunInit_Integration_NonInteractiveOverwrite(t *testing.T) {
	env := testutil.RequireTestEnv(t)

	tempDir, err := os.MkdirTemp("", "gh-pmu-init-overwrite-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create existing config
	existingConfig := `{"project":{"name":"existing","owner":"test","number":999}}`
	configPath := filepath.Join(tempDir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("Failed to write existing config: %v", err)
	}

	// Run init with --yes to overwrite
	cmd := exec.Command("gh", "pmu", "init",
		"--non-interactive",
		"--source-project", fmt.Sprintf("%d", env.ProjectNumber),
		"--repo", fmt.Sprintf("%s/%s", env.RepoOwner, env.RepoName),
		"--yes")
	cmd.Dir = tempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Logf("Init stderr: %s", stderr.String())
		t.Fatalf("Non-interactive init with --yes failed: %v", err)
	}

	// Verify config was overwritten (should not contain "existing")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if strings.Contains(string(configData), "existing") {
		t.Error("Expected existing config to be overwritten")
	}
	if !strings.Contains(string(configData), env.RepoOwner) {
		t.Error("Expected new config to contain repo owner")
	}
}

// TestRunInit_Integration_NonInteractiveMissingFlags tests error handling
func TestRunInit_Integration_NonInteractiveMissingFlags(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gh-pmu-init-missing-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Run init without required flags
	cmd := exec.Command("gh", "pmu", "init", "--non-interactive")
	cmd.Dir = tempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		t.Error("Expected error when required flags are missing")
	}

	// Check stderr contains error about missing flags
	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, "--source-project") || !strings.Contains(stderrOutput, "--repo") {
		t.Errorf("Expected stderr to mention missing flags, got: %s", stderrOutput)
	}
}
