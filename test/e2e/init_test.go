//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"encoding/json"
)

// TestInitNonInteractiveMode tests the init command in non-interactive mode.
// This copies from the source project #41 configured in config_test.go.
func TestInitNonInteractiveMode(t *testing.T) {
	// Create a temp directory for the init test
	tmpDir := t.TempDir()

	// Initialize git repo (some init validation may need it)
	initGitRepo(t, tmpDir)

	t.Run("non-interactive_creates_config", func(t *testing.T) {
		// Run init in non-interactive mode
		result := runPMU(t, tmpDir, "init",
			"--non-interactive",
			"--source-project", "41",
			"--repo", "rubrical-worker/gh-pmu-e2e-test",
		)

		assertExitCode(t, result, 0)

		// Clean up the created project
		if projNum := extractProjectNumber(t, result.Stdout); projNum > 0 {
			t.Cleanup(func() { deleteTestProject(t, projNum) })
		}

		// Verify config file was created
		configPath := filepath.Join(tmpDir, ".gh-pmu.json")
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

		// Verify project section
		project, ok := config["project"].(map[string]interface{})
		if !ok {
			t.Error("Config missing 'project' section")
		} else {
			// Non-interactive mode copies from source project, so the new
			// project number will differ from the source (41). Just verify
			// a positive number was assigned.
			if num, ok := project["number"].(float64); !ok || num <= 0 {
				t.Errorf("Expected positive project number, got %v", project["number"])
			}
			if project["owner"] != "rubrical-worker" {
				t.Errorf("Expected owner 'rubrical-worker', got %v", project["owner"])
			}
		}

		// Verify repositories section
		repos, ok := config["repositories"].([]interface{})
		if !ok {
			t.Error("Config missing 'repositories' section")
		} else if len(repos) == 0 {
			t.Error("No repositories configured")
		} else if repos[0] != "rubrical-worker/gh-pmu-e2e-test" {
			t.Errorf("Expected repo 'rubrical-worker/gh-pmu-e2e-test', got %v", repos[0])
		}

		// Verify framework defaults to IDPF
		if framework, ok := config["framework"].(string); ok {
			if framework != "IDPF" {
				t.Errorf("Expected framework 'IDPF', got %q", framework)
			}
		}

		// Verify metadata section exists
		if _, ok := config["metadata"]; !ok {
			t.Error("Config missing 'metadata' section")
		}
	})
}

// TestInitNonInteractiveFrameworkNone tests init with --framework none.
func TestInitNonInteractiveFrameworkNone(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	result := runPMU(t, tmpDir, "init",
		"--non-interactive",
		"--source-project", "41",
		"--repo", "rubrical-worker/gh-pmu-e2e-test",
		"--framework", "none",
	)

	assertExitCode(t, result, 0)

	// Clean up the created project
	if projNum := extractProjectNumber(t, result.Stdout); projNum > 0 {
		t.Cleanup(func() { deleteTestProject(t, projNum) })
	}

	// Read and validate config
	configPath := filepath.Join(tmpDir, ".gh-pmu.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify framework is none
	if framework, ok := config["framework"].(string); ok {
		if framework != "none" {
			t.Errorf("Expected framework 'none', got %q", framework)
		}
	} else {
		t.Error("Framework field not found in config")
	}
}

// TestInitNonInteractiveWithOwner tests that --owner is honored over the owner
// that would otherwise be inferred from --repo.
//
// The e2e infra is single-owner: the source template project #41 and project
// creation only work under "rubrical-worker", so --owner must be rubrical-worker.
// To make the flag's effect observable, --repo deliberately points at a repo under
// a DIFFERENT owner (octocat/Hello-World). Repository linking and label creation are
// warn-only in init (init.go / init_atomic.go), so the unrelated repo does not fail
// init. If --owner were ignored, the resolved owner would fall back to the --repo
// owner "octocat", source-project #41 lookup under octocat would fail, and init would
// exit non-zero. Honoring --owner yields exit 0 and project.owner == "rubrical-worker"
// — distinguishable from the repo owner, unlike the previous same-owner assertion.
func TestInitNonInteractiveWithOwner(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	result := runPMU(t, tmpDir, "init",
		"--non-interactive",
		"--source-project", "41",
		"--repo", "octocat/Hello-World", // repo owner deliberately != --owner
		"--owner", "rubrical-worker",
	)

	assertExitCode(t, result, 0)

	// Clean up the created project (created under the --owner, rubrical-worker)
	if projNum := extractProjectNumber(t, result.Stdout); projNum > 0 {
		t.Cleanup(func() { deleteTestProject(t, projNum) })
	}

	// Parse the generated config and assert project.owner reflects --owner, not the
	// --repo owner. A silently-ignored --owner could not produce this value.
	configPath := filepath.Join(tmpDir, ".gh-pmu.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	project, ok := config["project"].(map[string]interface{})
	if !ok {
		t.Fatalf("Config missing 'project' section")
	}
	if project["owner"] != "rubrical-worker" {
		t.Errorf("Expected project.owner 'rubrical-worker' (from --owner), got %v — --owner not honored", project["owner"])
	}
}

// TestInitNonInteractiveOverwrite tests the --yes + --force combo for overwriting
// when the existing config declares a project number. Since #847 added a guard
// that refuses --source-project against a repo whose .gh-pmu.json declares a
// project (steering users toward --project <existing>), the explicit-replace
// path now requires --force in addition to --yes.
func TestInitNonInteractiveOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create existing config with a project number — triggers the #847 guard.
	existingConfig := `{"project":{"name":"existing","owner":"test","number":999}}`
	configPath := filepath.Join(tmpDir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("Failed to write existing config: %v", err)
	}

	// Run init with --yes + --force to overwrite (--yes alone is no longer enough
	// when the existing config has a project number; --force confirms the user
	// really wants to replace the project pointer rather than re-link via --project).
	result := runPMU(t, tmpDir, "init",
		"--non-interactive",
		"--source-project", "41",
		"--repo", "rubrical-worker/gh-pmu-e2e-test",
		"--yes",
		"--force",
	)

	assertExitCode(t, result, 0)

	// Clean up the created project
	if projNum := extractProjectNumber(t, result.Stdout); projNum > 0 {
		t.Cleanup(func() { deleteTestProject(t, projNum) })
	}

	// Verify config was overwritten
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if strings.Contains(string(configData), "existing") {
		t.Error("Expected existing config to be overwritten")
	}
	if !strings.Contains(string(configData), "rubrical-worker") {
		t.Error("Expected new config to contain repo owner")
	}
}

// TestInitNonInteractiveMissingFlags tests error handling for missing flags.
func TestInitNonInteractiveMissingFlags(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("missing_both_flags", func(t *testing.T) {
		result := runPMU(t, tmpDir, "init", "--non-interactive")
		assertExitCode(t, result, 1)
		assertContains(t, result.Stderr, "--source-project")
		assertContains(t, result.Stderr, "--repo")
	})

	t.Run("missing_project", func(t *testing.T) {
		result := runPMU(t, tmpDir, "init",
			"--non-interactive",
			"--repo", "owner/repo",
		)
		assertExitCode(t, result, 1)
		assertContains(t, result.Stderr, "--source-project")
	})

	t.Run("missing_repo", func(t *testing.T) {
		result := runPMU(t, tmpDir, "init",
			"--non-interactive",
			"--source-project", "41",
		)
		assertExitCode(t, result, 1)
		assertContains(t, result.Stderr, "--repo")
	})
}

// TestInitNonInteractiveInvalidRepoFormat tests error handling for invalid repo format.
func TestInitNonInteractiveInvalidRepoFormat(t *testing.T) {
	tmpDir := t.TempDir()

	result := runPMU(t, tmpDir, "init",
		"--non-interactive",
		"--source-project", "41",
		"--repo", "invalid-repo-format",
	)

	assertExitCode(t, result, 1)
	// Should mention owner/repo format in error
	if !strings.Contains(result.Stderr, "owner/repo") && !strings.Contains(result.Stderr, "invalid") {
		t.Errorf("Expected error about repo format, got: %s", result.Stderr)
	}
}

// TestInitNonInteractiveExistingConfigNoYes tests error when config exists without --yes.
func TestInitNonInteractiveExistingConfigNoYes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing config in both formats (init checks JSON first, YAML as fallback)
	jsonPath := filepath.Join(tmpDir, ".gh-pmu.json")
	if err := os.WriteFile(jsonPath, []byte(`{"project":{"owner":"test"}}`), 0644); err != nil {
		t.Fatalf("Failed to write existing JSON config: %v", err)
	}
	yamlPath := filepath.Join(tmpDir, ".gh-pmu.json")
	if err := os.WriteFile(yamlPath, []byte("project:\n  owner: test\n"), 0644); err != nil {
		t.Fatalf("Failed to write existing YAML config: %v", err)
	}

	result := runPMU(t, tmpDir, "init",
		"--non-interactive",
		"--source-project", "41",
		"--repo", "rubrical-worker/gh-pmu-e2e-test",
	)

	assertExitCode(t, result, 1)
	// Should mention --yes in error
	if !strings.Contains(result.Stderr, "--yes") && !strings.Contains(result.Stderr, "already exists") {
		t.Errorf("Expected error about --yes or already exists, got: %s", result.Stderr)
	}
}
