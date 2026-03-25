package framework

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// REQ-029: Framework Detection
// =============================================================================

// AC-029-1: Given `.gh-pmu.json` with `framework: IDPF-Agile`, Then framework detected as Agile
func TestDetectFramework_FromGhPmuJson(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()

	// Create .gh-pmu.json with framework field
	ghPmuContent := `{"project":{"owner":"testowner","number":1},"repositories":["testowner/testrepo"],"framework":"IDPF-Agile"}`
	err := os.WriteFile(filepath.Join(tempDir, ".gh-pmu.json"), []byte(ghPmuContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// ACT
	framework, err := DetectFramework(tempDir)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if framework != "IDPF-Agile" {
		t.Errorf("Expected framework 'IDPF-Agile', got '%s'", framework)
	}
}

// AC-029-2: Given `framework-config.json` with `processFramework: "IDPF-Structured"`, Then framework detected
func TestDetectFramework_FromFrameworkConfigJson(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()

	// Create framework-config.json with processFramework
	frameworkConfigContent := `{
  "projectType": {
    "processFramework": "IDPF-Structured"
  }
}`
	err := os.WriteFile(filepath.Join(tempDir, "framework-config.json"), []byte(frameworkConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// ACT
	framework, err := DetectFramework(tempDir)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if framework != "IDPF-Structured" {
		t.Errorf("Expected framework 'IDPF-Structured', got '%s'", framework)
	}
}

// AC-029-3: Given no framework config, Then no framework restriction applied (empty string)
func TestDetectFramework_NoConfig_ReturnsEmpty(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()

	// No config files created - empty directory

	// ACT
	framework, err := DetectFramework(tempDir)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if framework != "" {
		t.Errorf("Expected empty framework (no restriction), got '%s'", framework)
	}
}

// Test priority: .gh-pmu.json takes precedence over framework-config.json
func TestDetectFramework_GhPmuJsonTakesPrecedence(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()

	// Create both config files with different values
	ghPmuContent := `{"project":{"owner":"testowner","number":1},"repositories":["testowner/testrepo"],"framework":"IDPF-Agile"}`
	err := os.WriteFile(filepath.Join(tempDir, ".gh-pmu.json"), []byte(ghPmuContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gh-pmu.json: %v", err)
	}

	frameworkConfigContent := `{
  "projectType": {
    "processFramework": "IDPF-Structured"
  }
}`
	err = os.WriteFile(filepath.Join(tempDir, "framework-config.json"), []byte(frameworkConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create framework-config.json: %v", err)
	}

	// ACT
	framework, err := DetectFramework(tempDir)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// .gh-pmu.json should take precedence
	if framework != "IDPF-Agile" {
		t.Errorf("Expected framework 'IDPF-Agile' (from .gh-pmu.json), got '%s'", framework)
	}
}

// Test .gh-pmu.json without framework field falls back to framework-config.json
func TestDetectFramework_GhPmuJsonWithoutFramework_FallsBackToFrameworkConfig(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()

	// Create .gh-pmu.json WITHOUT framework field
	ghPmuContent := `{"project":{"owner":"testowner","number":1},"repositories":["testowner/testrepo"]}`
	err := os.WriteFile(filepath.Join(tempDir, ".gh-pmu.json"), []byte(ghPmuContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gh-pmu.json: %v", err)
	}

	frameworkConfigContent := `{
  "projectType": {
    "processFramework": "IDPF-Structured"
  }
}`
	err = os.WriteFile(filepath.Join(tempDir, "framework-config.json"), []byte(frameworkConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create framework-config.json: %v", err)
	}

	// ACT
	framework, err := DetectFramework(tempDir)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should fall back to framework-config.json
	if framework != "IDPF-Structured" {
		t.Errorf("Expected framework 'IDPF-Structured' (from framework-config.json), got '%s'", framework)
	}
}

// =============================================================================
// REQ-030: Command Restrictions
// =============================================================================

// AC-030-1: Given IDPF-Agile framework, When running `patch start`,
// Then error suggesting `branch` command
func TestValidateCommand_AgileFramework_PatchBlocked(t *testing.T) {
	// ARRANGE
	framework := "IDPF-Agile"
	command := "patch"

	// ACT
	err := ValidateCommand(framework, command)

	// ASSERT
	if err == nil {
		t.Fatalf("Expected error for patch on IDPF-Agile, got nil")
	}

	// Verify error message contains helpful suggestion
	errMsg := err.Error()
	if !strings.Contains(errMsg, "not applicable") {
		t.Errorf("Expected error to mention 'not applicable', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "branch") {
		t.Errorf("Expected error to suggest 'branch' command, got: %s", errMsg)
	}
}

// Test that valid commands pass validation
func TestValidateCommand_AgileFramework_BranchAllowed(t *testing.T) {
	// ARRANGE
	framework := "IDPF-Agile"
	command := "branch"

	// ACT
	err := ValidateCommand(framework, command)

	// ASSERT
	if err != nil {
		t.Errorf("Expected branch to be allowed on IDPF-Agile, got error: %v", err)
	}
}

func TestValidateCommand_StructuredFramework_ReleaseAllowed(t *testing.T) {
	// ARRANGE
	framework := "IDPF-Structured"
	command := "release"

	// ACT
	err := ValidateCommand(framework, command)

	// ASSERT
	if err != nil {
		t.Errorf("Expected release to be allowed on IDPF-Structured, got error: %v", err)
	}
}

func TestValidateCommand_LTSFramework_PatchAllowed(t *testing.T) {
	// ARRANGE
	framework := "IDPF-LTS"
	command := "patch"

	// ACT
	err := ValidateCommand(framework, command)

	// ASSERT
	if err != nil {
		t.Errorf("Expected patch to be allowed on IDPF-LTS, got error: %v", err)
	}
}

// =============================================================================
// REQ-031: No Framework Fallback
// =============================================================================

// AC-031-1: Given no framework config, When running any workflow command,
// Then command executes without restriction
func TestValidateCommand_NoFramework_AllCommandsAllowed(t *testing.T) {
	// ARRANGE
	framework := "" // No framework detected
	commands := []string{"branch", "release", "patch"}

	for _, command := range commands {
		// ACT
		err := ValidateCommand(framework, command)

		// ASSERT
		if err != nil {
			t.Errorf("Expected %s to be allowed with no framework, got error: %v", command, err)
		}
	}
}

// AC-031-2: Given no framework, Then branch, release, and patch commands all available
func TestValidateCommand_NoFramework_BranchAvailable(t *testing.T) {
	// ARRANGE
	framework := ""
	command := "branch"

	// ACT
	err := ValidateCommand(framework, command)

	// ASSERT
	if err != nil {
		t.Errorf("Expected branch to be available with no framework, got error: %v", err)
	}
}

func TestValidateCommand_NoFramework_ReleaseAvailable(t *testing.T) {
	// ARRANGE
	framework := ""
	command := "release"

	// ACT
	err := ValidateCommand(framework, command)

	// ASSERT
	if err != nil {
		t.Errorf("Expected release to be available with no framework, got error: %v", err)
	}
}

func TestValidateCommand_NoFramework_PatchAvailable(t *testing.T) {
	// ARRANGE
	framework := ""
	command := "patch"

	// ACT
	err := ValidateCommand(framework, command)

	// ASSERT
	if err != nil {
		t.Errorf("Expected patch to be available with no framework, got error: %v", err)
	}
}
