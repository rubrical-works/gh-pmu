package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/config"
)

func TestAcceptCommand_WritesAcceptanceToConfig(t *testing.T) {
	// ARRANGE: Create temp dir with minimal config
	tmpDir := t.TempDir()
	cfg := config.Config{
		Project: config.Project{
			Owner:  "test-owner",
			Number: 1,
		},
		Repositories: []string{"test-owner/test-repo"},
	}
	data, _ := json.MarshalIndent(&cfg, "", "  ")
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// ACT: Run accept command with --yes flag
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"accept", "--yes", "--dir", tmpDir})
	err := cmd.Execute()

	// ASSERT: No error and acceptance recorded
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Read back config
	updatedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if updatedCfg.Acceptance == nil {
		t.Fatal("Expected acceptance to be set")
	}

	if !updatedCfg.Acceptance.Accepted {
		t.Error("Expected accepted to be true")
	}

	if updatedCfg.Acceptance.Version == "" {
		t.Error("Expected version to be set")
	}

	if updatedCfg.Acceptance.Date == "" {
		t.Error("Expected date to be set")
	}

	if updatedCfg.Acceptance.User == "" {
		t.Error("Expected user to be set")
	}
}

func TestAcceptCommand_ShowsTermsText(t *testing.T) {
	// ARRANGE: Create temp dir with minimal config
	tmpDir := t.TempDir()
	cfg := config.Config{
		Project: config.Project{
			Owner:  "test-owner",
			Number: 1,
		},
		Repositories: []string{"test-owner/test-repo"},
	}
	data, _ := json.MarshalIndent(&cfg, "", "  ")
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// ACT: Run accept command with --yes flag
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"accept", "--yes", "--dir", tmpDir})
	_ = cmd.Execute()

	// ASSERT: Output contains terms text
	output := buf.String()
	if !strings.Contains(output, "Terms and Conditions") {
		t.Error("Expected output to contain terms text")
	}
}

func TestAcceptCommand_ShowsSharedAcceptanceNotice(t *testing.T) {
	// ARRANGE: Create temp dir with minimal config
	tmpDir := t.TempDir()
	cfg := config.Config{
		Project: config.Project{
			Owner:  "test-owner",
			Number: 1,
		},
		Repositories: []string{"test-owner/test-repo"},
	}
	data, _ := json.MarshalIndent(&cfg, "", "  ")
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// ACT: Run accept with --yes
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"accept", "--yes", "--dir", tmpDir})
	_ = cmd.Execute()

	// ASSERT: Output contains shared acceptance notice
	output := buf.String()
	if !strings.Contains(output, "all users") || !strings.Contains(output, "collaborator") {
		t.Error("Expected shared acceptance notice mentioning all users/collaborators")
	}
}

func TestAcceptCommand_LongDescriptionReferencesPraxis(t *testing.T) {
	cmd := NewRootCommand()
	acceptCmd, _, _ := cmd.Find([]string{"accept"})
	if acceptCmd == nil {
		t.Fatal("Expected to find accept subcommand")
	}

	long := acceptCmd.Long
	if !strings.Contains(long, "Praxis Management Utility") {
		t.Errorf("Expected accept Long description to reference 'Praxis Management Utility', got: %s", long)
	}
}

func TestAcceptCommand_RecordsVersion(t *testing.T) {
	// ARRANGE: Create temp dir with minimal config
	tmpDir := t.TempDir()
	cfg := config.Config{
		Project: config.Project{
			Owner:  "test-owner",
			Number: 1,
		},
		Repositories: []string{"test-owner/test-repo"},
	}
	data, _ := json.MarshalIndent(&cfg, "", "  ")
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// ACT
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"accept", "--yes", "--dir", tmpDir})
	_ = cmd.Execute()

	// ASSERT: Version matches effective version (getVersion uses source constant when ldflags empty)
	updatedCfg, _ := config.Load(configPath)
	expected := getVersion()
	if updatedCfg.Acceptance.Version != expected {
		t.Errorf("Expected version %q, got %q", expected, updatedCfg.Acceptance.Version)
	}
}

func TestAcceptCommand_SavesAcceptanceToJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Config{
		Project: config.Project{
			Owner:  "test-owner",
			Number: 1,
		},
		Repositories: []string{"test-owner/test-repo"},
	}
	data, _ := json.MarshalIndent(&cfg, "", "  ")
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"accept", "--yes", "--dir", tmpDir})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify JSON config was updated with acceptance
	jsonCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}
	if jsonCfg.Acceptance == nil || !jsonCfg.Acceptance.Accepted {
		t.Error("Expected JSON config to contain acceptance")
	}

	// Verify no YAML companion was created
	yamlPath := filepath.Join(tmpDir, config.ConfigFileNameYAML)
	if _, err := os.Stat(yamlPath); !os.IsNotExist(err) {
		t.Error("Expected no YAML companion to be created")
	}
}
