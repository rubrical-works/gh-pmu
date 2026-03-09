package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
)

func TestInitCommand_Exists(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("init command should exist: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("init")) {
		t.Error("Expected help output to mention 'init'")
	}
}

func TestDetectRepository_FromGitRemote(t *testing.T) {
	// Test with a known git remote URL
	tests := []struct {
		name     string
		remote   string
		expected string
	}{
		{
			name:     "HTTPS URL",
			remote:   "https://github.com/owner/repo.git",
			expected: "owner/repo",
		},
		{
			name:     "HTTPS URL without .git",
			remote:   "https://github.com/owner/repo",
			expected: "owner/repo",
		},
		{
			name:     "SSH URL",
			remote:   "git@github.com:owner/repo.git",
			expected: "owner/repo",
		},
		{
			name:     "SSH URL without .git",
			remote:   "git@github.com:owner/repo",
			expected: "owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitRemote(tt.remote)
			if result != tt.expected {
				t.Errorf("parseGitRemote(%q) = %q, want %q", tt.remote, result, tt.expected)
			}
		})
	}
}

func TestDetectRepository_InvalidRemote(t *testing.T) {
	tests := []string{
		"",
		"not-a-url",
		"https://gitlab.com/owner/repo",
	}

	for _, remote := range tests {
		t.Run(remote, func(t *testing.T) {
			result := parseGitRemote(remote)
			if result != "" {
				t.Errorf("parseGitRemote(%q) = %q, want empty string", remote, result)
			}
		})
	}
}

func TestWriteConfig_CreatesValidYAML(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectOwner:  "test-owner",
		ProjectNumber: 5,
		Repositories:  []string{"test-owner/test-repo"},
	}

	err := writeConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	// Verify file was created
	configPath := tmpDir + "/.gh-pmu.yml"
	content, err := readFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// Check content contains expected values
	if !bytes.Contains(content, []byte("owner: test-owner")) {
		t.Error("Config should contain owner")
	}
	if !bytes.Contains(content, []byte("number: 5")) {
		t.Error("Config should contain project number")
	}
	if !bytes.Contains(content, []byte("test-owner/test-repo")) {
		t.Error("Config should contain repository")
	}
}

func TestWriteConfig_WithDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	err := writeConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	content, _ := readFile(tmpDir + "/.gh-pmu.yml")

	// Should have default status field mapping
	if !bytes.Contains(content, []byte("status:")) {
		t.Error("Config should have default status field")
	}
}

func TestWriteConfig_IncludesTriageAndLabels(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectName:   "Test Project",
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	err := writeConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	content, _ := readFile(tmpDir + "/.gh-pmu.yml")

	// Should have project name
	if !bytes.Contains(content, []byte("name: Test Project")) {
		t.Error("Config should have project name")
	}

	// Should have triage section
	if !bytes.Contains(content, []byte("triage:")) {
		t.Error("Config should have triage section")
	}

	// Should have estimate triage rule
	if !bytes.Contains(content, []byte("estimate:")) {
		t.Error("Config should have estimate triage rule")
	}
}

// Helper to read file for tests
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func TestValidateProject_Success(t *testing.T) {
	// Mock client that returns a valid project
	mockClient := &MockAPIClient{
		project: &MockProject{
			ID:    "PVT_test123",
			Title: "Test Project",
		},
	}

	err := validateProject(mockClient, "owner", 1)
	if err != nil {
		t.Errorf("validateProject should succeed for valid project: %v", err)
	}
}

func TestValidateProject_NotFound(t *testing.T) {
	// Mock client that returns not found error
	mockClient := &MockAPIClient{
		err: ErrProjectNotFound,
	}

	err := validateProject(mockClient, "owner", 999)
	if err == nil {
		t.Error("validateProject should fail for non-existent project")
	}
}

// MockProject represents a mock project for testing
type MockProject struct {
	ID    string
	Title string
}

// MockAPIClient is a mock implementation for testing
type MockAPIClient struct {
	project *MockProject
	err     error
}

// GetProject implements ProjectValidator interface
func (m *MockAPIClient) GetProject(owner string, number int) (interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.project, nil
}

// ErrProjectNotFound is returned when project doesn't exist
var ErrProjectNotFound = fmt.Errorf("project not found")

func TestWriteConfigWithMetadata_IncludesFields(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	metadata := &ProjectMetadata{
		ProjectID: "PVT_test123",
		Fields: []FieldMetadata{
			{
				ID:       "PVTF_status",
				Name:     "Status",
				DataType: "SINGLE_SELECT",
				Options: []OptionMetadata{
					{ID: "opt1", Name: "Backlog"},
					{ID: "opt2", Name: "Done"},
				},
			},
			{
				ID:       "PVTF_priority",
				Name:     "Priority",
				DataType: "SINGLE_SELECT",
				Options: []OptionMetadata{
					{ID: "opt3", Name: "High"},
					{ID: "opt4", Name: "Low"},
				},
			},
		},
	}

	err := writeConfigWithMetadata(tmpDir, cfg, metadata)
	if err != nil {
		t.Fatalf("writeConfigWithMetadata failed: %v", err)
	}

	content, _ := readFile(tmpDir + "/.gh-pmu.yml")

	// Should contain metadata section with project ID
	if !bytes.Contains(content, []byte("metadata:")) {
		t.Error("Config should have metadata section")
	}
	if !bytes.Contains(content, []byte("PVT_test123")) {
		t.Error("Config should contain project ID")
	}
	// Should contain field IDs
	if !bytes.Contains(content, []byte("PVTF_status")) {
		t.Error("Config should contain field IDs")
	}
}

func TestSplitRepository(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedOwner string
		expectedName  string
	}{
		{
			name:          "valid owner/repo format",
			input:         "rubrical-works/gh-pmu",
			expectedOwner: "rubrical-works",
			expectedName:  "gh-pmu",
		},
		{
			name:          "simple owner/repo",
			input:         "owner/repo",
			expectedOwner: "owner",
			expectedName:  "repo",
		},
		{
			name:          "no slash - invalid input",
			input:         "noslash",
			expectedOwner: "",
			expectedName:  "",
		},
		{
			name:          "empty string",
			input:         "",
			expectedOwner: "",
			expectedName:  "",
		},
		{
			name:          "multiple slashes - takes first split",
			input:         "owner/repo/extra",
			expectedOwner: "owner",
			expectedName:  "repo/extra",
		},
		{
			name:          "only slash",
			input:         "/",
			expectedOwner: "",
			expectedName:  "",
		},
		{
			name:          "owner with trailing slash",
			input:         "owner/",
			expectedOwner: "owner",
			expectedName:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, name := splitRepository(tt.input)
			if owner != tt.expectedOwner {
				t.Errorf("splitRepository(%q) owner = %q, want %q", tt.input, owner, tt.expectedOwner)
			}
			if name != tt.expectedName {
				t.Errorf("splitRepository(%q) name = %q, want %q", tt.input, name, tt.expectedName)
			}
		})
	}
}

func TestWriteConfigWithMetadata_EmptyMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectName:   "Test",
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	// Empty metadata with no fields
	metadata := &ProjectMetadata{
		ProjectID: "PVT_empty",
		Fields:    []FieldMetadata{},
	}

	err := writeConfigWithMetadata(tmpDir, cfg, metadata)
	if err != nil {
		t.Fatalf("writeConfigWithMetadata failed with empty fields: %v", err)
	}

	content, err := readFile(tmpDir + "/.gh-pmu.yml")
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// Should still have metadata section
	if !bytes.Contains(content, []byte("metadata:")) {
		t.Error("Config should have metadata section even with empty fields")
	}
	if !bytes.Contains(content, []byte("PVT_empty")) {
		t.Error("Config should contain project ID")
	}
}

func TestWriteConfigWithMetadata_FieldOptions(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	metadata := &ProjectMetadata{
		ProjectID: "PVT_test",
		Fields: []FieldMetadata{
			{
				ID:       "PVTF_size",
				Name:     "Size",
				DataType: "SINGLE_SELECT",
				Options: []OptionMetadata{
					{ID: "size_xs", Name: "XS"},
					{ID: "size_s", Name: "S"},
					{ID: "size_m", Name: "M"},
					{ID: "size_l", Name: "L"},
					{ID: "size_xl", Name: "XL"},
				},
			},
		},
	}

	err := writeConfigWithMetadata(tmpDir, cfg, metadata)
	if err != nil {
		t.Fatalf("writeConfigWithMetadata failed: %v", err)
	}

	content, _ := readFile(tmpDir + "/.gh-pmu.yml")

	// Check all options are written
	options := []string{"XS", "S", "M", "L", "XL"}
	for _, opt := range options {
		if !bytes.Contains(content, []byte(opt)) {
			t.Errorf("Config should contain option %q", opt)
		}
	}
}

func TestWriteConfig_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	err := writeConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	// Check file exists and is readable
	info, err := os.Stat(tmpDir + "/.gh-pmu.yml")
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	// File should not be a directory
	if info.IsDir() {
		t.Error("Config file should not be a directory")
	}

	// File should have some content
	if info.Size() == 0 {
		t.Error("Config file should not be empty")
	}
}

// ============================================================================
// writeConfig Error Path Tests (IT-3.4)
// ============================================================================

func TestWriteConfig_InvalidDirectory(t *testing.T) {
	// Try to write to a non-existent directory
	nonExistentDir := "/nonexistent/path/that/does/not/exist"

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	err := writeConfig(nonExistentDir, cfg)
	if err == nil {
		t.Error("Expected error when writing to non-existent directory")
	}

	// Check error message mentions file write failure
	if !strings.Contains(err.Error(), "failed to write config file") {
		t.Errorf("Expected 'failed to write config file' error, got: %v", err)
	}
}

func TestWriteConfig_ReadOnlyDirectory(t *testing.T) {
	// Skip on Windows as permission handling differs
	if os.Getenv("OS") == "Windows_NT" || strings.Contains(os.Getenv("OS"), "Windows") {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()

	// Make directory read-only
	if err := os.Chmod(tmpDir, 0444); err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}
	// Restore permissions for cleanup
	defer func() { _ = os.Chmod(tmpDir, 0755) }()

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	err := writeConfig(tmpDir, cfg)
	if err == nil {
		t.Error("Expected error when writing to read-only directory")
	}
}

func TestWriteConfigWithMetadata_InvalidDirectory(t *testing.T) {
	// Try to write to a non-existent directory
	nonExistentDir := "/nonexistent/path/that/does/not/exist"

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	metadata := &ProjectMetadata{
		ProjectID: "test-id",
		Fields:    []FieldMetadata{},
	}

	err := writeConfigWithMetadata(nonExistentDir, cfg, metadata)
	if err == nil {
		t.Error("Expected error when writing to non-existent directory")
	}

	// Check error message mentions file write failure
	if !strings.Contains(err.Error(), "failed to write config file") {
		t.Errorf("Expected 'failed to write config file' error, got: %v", err)
	}
}

func TestWriteConfig_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty config should still work (though with empty/default values)
	cfg := &InitConfig{}

	err := writeConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("writeConfig with empty config failed: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tmpDir, ".gh-pmu.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestWriteConfig_OverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Write initial config
	cfg1 := &InitConfig{
		ProjectOwner:  "owner1",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo1"},
	}
	if err := writeConfig(tmpDir, cfg1); err != nil {
		t.Fatalf("Initial writeConfig failed: %v", err)
	}

	// Write second config (should overwrite)
	cfg2 := &InitConfig{
		ProjectOwner:  "owner2",
		ProjectNumber: 2,
		Repositories:  []string{"owner/repo2"},
	}
	if err := writeConfig(tmpDir, cfg2); err != nil {
		t.Fatalf("Second writeConfig failed: %v", err)
	}

	// Read file and verify it has new content
	configPath := filepath.Join(tmpDir, ".gh-pmu.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "owner2") {
		t.Error("Expected config to contain 'owner2' (new value)")
	}
	if strings.Contains(content, "owner1") {
		t.Error("Expected old 'owner1' to be overwritten")
	}
}

func TestWriteConfig_IncludesVersion(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	err := writeConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	content, _ := readFile(tmpDir + "/.gh-pmu.yml")

	// Version field should be present and match the source constant
	expected := getVersion()
	if !bytes.Contains(content, []byte("version: "+expected)) {
		t.Errorf("Config should contain version: %s, got:\n%s", expected, string(content))
	}
}

func TestWriteConfigWithMetadata_IncludesVersion(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}
	meta := &ProjectMetadata{
		ProjectID: "test-project-id",
		Fields:    []FieldMetadata{},
	}

	err := writeConfigWithMetadata(tmpDir, cfg, meta)
	if err != nil {
		t.Fatalf("writeConfigWithMetadata failed: %v", err)
	}

	content, _ := readFile(tmpDir + "/.gh-pmu.yml")

	expected := getVersion()
	if !bytes.Contains(content, []byte("version: "+expected)) {
		t.Errorf("Config should contain version: %s, got:\n%s", expected, string(content))
	}
}

func TestWriteConfigWithMetadata_NilMetadataPanics(t *testing.T) {
	// Document that nil metadata causes a panic
	// This test verifies the current behavior - the function does not handle nil metadata
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when metadata is nil, but function didn't panic")
		}
	}()

	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	// This should panic because metadata is nil
	// Note: In production, metadata is always provided by the caller
	_ = writeConfigWithMetadata(tmpDir, cfg, nil)
}

func TestParseGitRemote_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		remote   string
		expected string
	}{
		{
			name:     "GitHub enterprise HTTPS - not supported",
			remote:   "https://github.example.com/owner/repo.git",
			expected: "",
		},
		{
			name:     "GitLab URL - not supported",
			remote:   "https://gitlab.com/owner/repo.git",
			expected: "",
		},
		{
			name:     "Bitbucket URL - not supported",
			remote:   "https://bitbucket.org/owner/repo.git",
			expected: "",
		},
		{
			name:     "SSH with port - not standard GitHub",
			remote:   "ssh://git@github.com:22/owner/repo.git",
			expected: "",
		},
		{
			name:     "file protocol",
			remote:   "file:///path/to/repo.git",
			expected: "",
		},
		{
			name:     "random string",
			remote:   "not-a-valid-url",
			expected: "",
		},
		{
			name:     "empty string",
			remote:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitRemote(tt.remote)
			if result != tt.expected {
				t.Errorf("parseGitRemote(%q) = %q, want %q", tt.remote, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// loadExistingFramework Tests
// ============================================================================

func TestLoadExistingFramework_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a config file
	configContent := `
framework: IDPF
project:
  owner: test
  number: 1
`
	configPath := filepath.Join(tmpDir, ".gh-pmu.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load and verify
	result, err := loadExistingFramework(tmpDir)
	if err != nil {
		t.Fatalf("loadExistingFramework failed: %v", err)
	}

	if result != "IDPF" {
		t.Errorf("Expected framework 'IDPF', got %q", result)
	}
}

func TestLoadExistingFramework_NoFramework(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
project:
  owner: test
  number: 1
`
	configPath := filepath.Join(tmpDir, ".gh-pmu.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	result, err := loadExistingFramework(tmpDir)
	if err != nil {
		t.Fatalf("loadExistingFramework failed: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty framework, got %q", result)
	}
}

func TestLoadExistingFramework_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create any config file

	_, err := loadExistingFramework(tmpDir)
	if err == nil {
		t.Error("Expected error for missing config file")
	}
}

func TestLoadExistingFramework_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
not valid: yaml:
  - bad indent
    really bad
`
	configPath := filepath.Join(tmpDir, ".gh-pmu.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	_, err := loadExistingFramework(tmpDir)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

// ============================================================================
// isRepoRoot Tests
// ============================================================================

func TestIsRepoRoot_WithGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a go.mod file
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	result := isRepoRoot(tmpDir)
	if !result {
		t.Error("Expected isRepoRoot to return true when go.mod exists")
	}
}

func TestIsRepoRoot_WithoutGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	result := isRepoRoot(tmpDir)
	if result {
		t.Error("Expected isRepoRoot to return false when go.mod doesn't exist")
	}
}

func TestIsRepoRoot_InvalidPath(t *testing.T) {
	result := isRepoRoot("/nonexistent/path/12345")
	if result {
		t.Error("Expected isRepoRoot to return false for invalid path")
	}
}

// ============================================================================
// SetRepoRootProtection Tests
// ============================================================================

func TestSetRepoRootProtection_EnablesProtection(t *testing.T) {
	// Reset protection state after test
	defer SetRepoRootProtection(false)

	SetRepoRootProtection(true)

	tmpDir := t.TempDir()
	// Create go.mod to simulate repo root
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	err := writeConfig(tmpDir, cfg)
	if err != ErrRepoRootProtected {
		t.Errorf("Expected ErrRepoRootProtected, got: %v", err)
	}
}

func TestSetRepoRootProtection_DisabledAllowsWrite(t *testing.T) {
	SetRepoRootProtection(false)

	tmpDir := t.TempDir()
	// Create go.mod to simulate repo root
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	err := writeConfig(tmpDir, cfg)
	if err != nil {
		t.Errorf("Expected write to succeed with protection disabled, got: %v", err)
	}
}

func TestWriteConfigWithMetadata_RepoRootProtection(t *testing.T) {
	// Reset protection state after test
	defer SetRepoRootProtection(false)

	SetRepoRootProtection(true)

	tmpDir := t.TempDir()
	// Create go.mod to simulate repo root
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	metadata := &ProjectMetadata{
		ProjectID: "PVT_test",
		Fields:    []FieldMetadata{},
	}

	err := writeConfigWithMetadata(tmpDir, cfg, metadata)
	if err != ErrRepoRootProtected {
		t.Errorf("Expected ErrRepoRootProtected, got: %v", err)
	}
}

// ============================================================================
// optionNameToAlias Tests (Issue #442)
// ============================================================================

func TestOptionNameToAlias(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lowercase",
			input:    "Backlog",
			expected: "backlog",
		},
		{
			name:     "space to underscore",
			input:    "In progress",
			expected: "in_progress",
		},
		{
			name:     "multiple spaces",
			input:    "In Review",
			expected: "in_review",
		},
		{
			name:     "emoji prefix with space",
			input:    "🅿️ Parking Lot",
			expected: "parking_lot",
		},
		{
			name:     "emoji prefix no space",
			input:    "🚀Ready",
			expected: "ready",
		},
		{
			name:     "multiple emojis",
			input:    "✅ ✓ Done",
			expected: "done",
		},
		{
			name:     "emoji only",
			input:    "🔥",
			expected: "",
		},
		{
			name:     "already lowercase underscore",
			input:    "in_progress",
			expected: "in_progress",
		},
		{
			name:     "uppercase with underscore",
			input:    "IN_PROGRESS",
			expected: "in_progress",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  Backlog  ",
			expected: "backlog",
		},
		{
			name:     "P0 priority",
			input:    "P0",
			expected: "p0",
		},
		{
			name:     "complex emoji",
			input:    "🏃‍♂️ Running",
			expected: "running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := optionNameToAlias(tt.input)
			if result != tt.expected {
				t.Errorf("optionNameToAlias(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// buildFieldMappingsFromMetadata Tests (Issue #442)
// ============================================================================

func TestBuildFieldMappingsFromMetadata_StatusField(t *testing.T) {
	metadata := &ProjectMetadata{
		ProjectID: "PVT_test",
		Fields: []FieldMetadata{
			{
				ID:       "PVTF_status",
				Name:     "Status",
				DataType: "SINGLE_SELECT",
				Options: []OptionMetadata{
					{ID: "opt1", Name: "Backlog"},
					{ID: "opt2", Name: "In progress"},
					{ID: "opt3", Name: "🅿️ Parking Lot"},
					{ID: "opt4", Name: "Done"},
				},
			},
		},
	}

	mappings := buildFieldMappingsFromMetadata(metadata)

	// Check status field exists
	status, ok := mappings["status"]
	if !ok {
		t.Fatal("Expected 'status' field mapping")
	}

	if status.Field != "Status" {
		t.Errorf("Expected field name 'Status', got %q", status.Field)
	}

	// Check all values are mapped
	expectedMappings := map[string]string{
		"backlog":     "Backlog",
		"in_progress": "In progress",
		"parking_lot": "🅿️ Parking Lot",
		"done":        "Done",
	}

	for alias, expected := range expectedMappings {
		if actual, ok := status.Values[alias]; !ok {
			t.Errorf("Missing alias %q in status values", alias)
		} else if actual != expected {
			t.Errorf("status.Values[%q] = %q, want %q", alias, actual, expected)
		}
	}
}

func TestBuildFieldMappingsFromMetadata_PriorityField(t *testing.T) {
	metadata := &ProjectMetadata{
		ProjectID: "PVT_test",
		Fields: []FieldMetadata{
			{
				ID:       "PVTF_priority",
				Name:     "Priority",
				DataType: "SINGLE_SELECT",
				Options: []OptionMetadata{
					{ID: "opt1", Name: "P0"},
					{ID: "opt2", Name: "P1"},
					{ID: "opt3", Name: "P2"},
					{ID: "opt4", Name: "P3"},
				},
			},
		},
	}

	mappings := buildFieldMappingsFromMetadata(metadata)

	priority, ok := mappings["priority"]
	if !ok {
		t.Fatal("Expected 'priority' field mapping")
	}

	// Check P3 is included (not hardcoded)
	if _, ok := priority.Values["p3"]; !ok {
		t.Error("Expected 'p3' to be included in priority values")
	}
}

func TestBuildFieldMappingsFromMetadata_FallbackDefaults(t *testing.T) {
	// Empty metadata - should use fallback defaults
	metadata := &ProjectMetadata{
		ProjectID: "PVT_test",
		Fields:    []FieldMetadata{},
	}

	mappings := buildFieldMappingsFromMetadata(metadata)

	// Should have default status
	status, ok := mappings["status"]
	if !ok {
		t.Fatal("Expected default 'status' field mapping")
	}
	if _, ok := status.Values["backlog"]; !ok {
		t.Error("Expected default 'backlog' in status values")
	}

	// Should have default priority
	priority, ok := mappings["priority"]
	if !ok {
		t.Fatal("Expected default 'priority' field mapping")
	}
	if _, ok := priority.Values["p0"]; !ok {
		t.Error("Expected default 'p0' in priority values")
	}
}

func TestBuildFieldMappingsFromMetadata_NoOptions(t *testing.T) {
	// Fields exist but have no options - should fall back to defaults
	metadata := &ProjectMetadata{
		ProjectID: "PVT_test",
		Fields: []FieldMetadata{
			{
				ID:       "PVTF_status",
				Name:     "Status",
				DataType: "SINGLE_SELECT",
				Options:  []OptionMetadata{}, // Empty options
			},
		},
	}

	mappings := buildFieldMappingsFromMetadata(metadata)

	// Should fall back to default status values
	status := mappings["status"]
	if _, ok := status.Values["backlog"]; !ok {
		t.Error("Expected fallback to default 'backlog' when no options")
	}
}

func TestBuildFieldMappingsFromMetadata_CaseInsensitiveFieldName(t *testing.T) {
	metadata := &ProjectMetadata{
		ProjectID: "PVT_test",
		Fields: []FieldMetadata{
			{
				ID:       "PVTF_status",
				Name:     "STATUS", // Uppercase
				DataType: "SINGLE_SELECT",
				Options: []OptionMetadata{
					{ID: "opt1", Name: "Active"},
				},
			},
		},
	}

	mappings := buildFieldMappingsFromMetadata(metadata)

	status, ok := mappings["status"]
	if !ok {
		t.Fatal("Expected 'status' field mapping for uppercase 'STATUS'")
	}
	if status.Field != "STATUS" {
		t.Errorf("Expected field name to preserve case 'STATUS', got %q", status.Field)
	}
}

func TestWriteConfigWithMetadata_IncludesParkingLot(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}

	metadata := &ProjectMetadata{
		ProjectID: "PVT_test",
		Fields: []FieldMetadata{
			{
				ID:       "PVTF_status",
				Name:     "Status",
				DataType: "SINGLE_SELECT",
				Options: []OptionMetadata{
					{ID: "opt1", Name: "Backlog"},
					{ID: "opt2", Name: "🅿️ Parking Lot"},
					{ID: "opt3", Name: "Done"},
				},
			},
		},
	}

	err := writeConfigWithMetadata(tmpDir, cfg, metadata)
	if err != nil {
		t.Fatalf("writeConfigWithMetadata failed: %v", err)
	}

	content, _ := readFile(tmpDir + "/.gh-pmu.yml")

	// Should contain parking_lot alias
	if !bytes.Contains(content, []byte("parking_lot:")) {
		t.Error("Config should contain 'parking_lot:' alias")
	}

	// Should contain the original name with emoji
	if !bytes.Contains(content, []byte("Parking Lot")) {
		t.Error("Config should contain 'Parking Lot' value")
	}
}

func TestFindFieldByName(t *testing.T) {
	fields := []api.ProjectField{
		{ID: "1", Name: "Status", DataType: "SINGLE_SELECT"},
		{ID: "2", Name: "Priority", DataType: "SINGLE_SELECT"},
		{ID: "3", Name: "Branch", DataType: "TEXT"},
	}

	tests := []struct {
		name      string
		fieldName string
		wantID    string
		wantNil   bool
	}{
		{
			name:      "find Status field",
			fieldName: "Status",
			wantID:    "1",
			wantNil:   false,
		},
		{
			name:      "find Priority field",
			fieldName: "Priority",
			wantID:    "2",
			wantNil:   false,
		},
		{
			name:      "find Branch field",
			fieldName: "Branch",
			wantID:    "3",
			wantNil:   false,
		},
		{
			name:      "field not found",
			fieldName: "NonExistent",
			wantNil:   true,
		},
		{
			name:      "case sensitive - lowercase",
			fieldName: "status",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findFieldByName(fields, tt.fieldName)
			if tt.wantNil {
				if result != nil {
					t.Errorf("findFieldByName() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("findFieldByName() = nil, want field with ID %s", tt.wantID)
				} else if result.ID != tt.wantID {
					t.Errorf("findFieldByName().ID = %s, want %s", result.ID, tt.wantID)
				}
			}
		})
	}
}

func TestFindFieldByName_EmptySlice(t *testing.T) {
	result := findFieldByName([]api.ProjectField{}, "Status")
	if result != nil {
		t.Errorf("findFieldByName() on empty slice = %v, want nil", result)
	}
}

// ============================================================================
// Non-Interactive Mode Tests (Issue #609)
// ============================================================================

func TestInitCommand_HasNonInteractiveFlag(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("init --help should work: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--non-interactive") {
		t.Error("Expected help output to mention '--non-interactive'")
	}
}

func TestInitCommand_HasSourceProjectFlag(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	_ = cmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "--source-project") {
		t.Error("Expected help output to mention '--source-project'")
	}
}

func TestInitCommand_HasRepoFlag(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	_ = cmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "--repo") {
		t.Error("Expected help output to mention '--repo'")
	}
}

func TestInitCommand_HasOwnerFlag(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	_ = cmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "--owner") {
		t.Error("Expected help output to mention '--owner'")
	}
}

func TestInitCommand_HasFrameworkFlag(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	_ = cmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "--framework") {
		t.Error("Expected help output to mention '--framework'")
	}
}

func TestInitCommand_HasYesFlag(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	_ = cmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "--yes") {
		t.Error("Expected help output to mention '--yes'")
	}
	if !strings.Contains(output, "-y") {
		t.Error("Expected help output to mention '-y' shorthand")
	}
}

func TestInitOptions_DefaultFramework(t *testing.T) {
	// Verify the default framework is IDPF
	cmd := newInitCommand()

	// Get the framework flag
	frameworkFlag := cmd.Flags().Lookup("framework")
	if frameworkFlag == nil {
		t.Fatal("Expected --framework flag to exist")
	}

	if frameworkFlag.DefValue != "IDPF" {
		t.Errorf("Expected default framework to be 'IDPF', got %q", frameworkFlag.DefValue)
	}
}

func TestInitNonInteractive_MissingSourceProjectFlag(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--non-interactive", "--repo", "owner/repo"})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when --source-project is missing in non-interactive mode")
	}

	// Check error mentions --source-project
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "--source-project") {
		t.Errorf("Expected error to mention --source-project, got: %s", errOutput)
	}
}

func TestInitNonInteractive_MissingRepoFlag(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--non-interactive", "--source-project", "5"})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when --repo is missing in non-interactive mode")
	}

	// Check error mentions --repo
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "--repo") {
		t.Errorf("Expected error to mention --repo, got: %s", errOutput)
	}
}

func TestInitNonInteractive_MissingBothFlags(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--non-interactive"})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when both --source-project and --repo are missing")
	}

	// Check error mentions both flags
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "--source-project") || !strings.Contains(errOutput, "--repo") {
		t.Errorf("Expected error to mention both --source-project and --repo, got: %s", errOutput)
	}
}

func TestInitNonInteractive_InvalidRepoFormat(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--non-interactive", "--source-project", "5", "--repo", "invalidrepo"})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid repo format")
	}

	// Check error mentions format issue
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "owner/repo") && !strings.Contains(errOutput, "invalid repo format") {
		t.Errorf("Expected error to mention format issue, got: %s", errOutput)
	}
}

func TestInitNonInteractive_OwnerInferredFromRepo(t *testing.T) {
	// Test that owner is correctly inferred from repo
	// This tests the splitRepository function which is used internally
	tests := []struct {
		repo          string
		expectedOwner string
	}{
		{"myorg/myrepo", "myorg"},
		{"company/repo-name", "company"},
		{"user123/project", "user123"},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			owner, _ := splitRepository(tt.repo)
			if owner != tt.expectedOwner {
				t.Errorf("Expected owner %q from repo %q, got %q", tt.expectedOwner, tt.repo, owner)
			}
		})
	}
}

func TestInitNonInteractive_ExistingConfigWithoutYes(t *testing.T) {
	// Create temp directory with existing config
	tmpDir := t.TempDir()

	// Write existing config
	configPath := filepath.Join(tmpDir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, []byte(`{"project":{"owner":"test"}}`), 0644); err != nil {
		t.Fatalf("Failed to write existing config: %v", err)
	}

	// Change to temp directory for the test
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--non-interactive", "--source-project", "5", "--repo", "owner/repo"})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when config exists without --yes flag")
	}

	// Check error mentions --yes or already exists
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "--yes") && !strings.Contains(errOutput, "already exists") {
		t.Errorf("Expected error to mention --yes or already exists, got: %s", errOutput)
	}
}

func TestInitNonInteractive_ConfigUsesNewProjectNumber(t *testing.T) {
	// Verify that when non-interactive mode creates a project by copying,
	// the config is written with the NEW project number, not the source number.
	tmpDir := t.TempDir()

	// The source project is #30, but the new project should get a different number.
	// We test this by writing config with a known new project number
	// and verifying it doesn't use the source number.
	sourceNumber := 30
	newNumber := 99
	cfg := &InitConfig{
		ProjectName:   "testrepo",
		ProjectOwner:  "testowner",
		ProjectNumber: newNumber, // Must be the NEW project number
		Repositories:  []string{"testowner/testrepo"},
		Framework:     "IDPF",
	}

	metadata := &ProjectMetadata{
		ProjectID: "PVT_new99",
	}

	err := writeConfigWithMetadata(tmpDir, cfg, metadata)
	if err != nil {
		t.Fatalf("writeConfigWithMetadata failed: %v", err)
	}

	// Read the config and verify it uses the new project number
	configData, err := os.ReadFile(filepath.Join(tmpDir, ".gh-pmu.json"))
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var configFile ConfigFileWithMetadata
	if err := json.Unmarshal(configData, &configFile); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if configFile.Project.Number == sourceNumber {
		t.Errorf("Config should use new project number (%d), not source number (%d)", newNumber, sourceNumber)
	}
	if configFile.Project.Number != newNumber {
		t.Errorf("Expected project number %d, got %d", newNumber, configFile.Project.Number)
	}
}

func TestInitNonInteractive_HelpDescribesSourceProjectCopy(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	_ = cmd.Execute()

	output := buf.String()

	// Verify help mentions source project copying semantics
	if !strings.Contains(output, "source") {
		t.Error("Expected help to mention 'source' project")
	}
	if !strings.Contains(output, "copy") {
		t.Error("Expected help to mention 'copy' behavior")
	}
}

func TestInitNonInteractive_ErrorMentionsSourceProject(t *testing.T) {
	// When --source-project is missing, error should mention --source-project
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"init", "--non-interactive", "--repo", "owner/repo"})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Expected error when --source-project is missing")
	}

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "--source-project") {
		t.Errorf("Error should mention --source-project, got: %s", errOutput)
	}
	// Should NOT mention the old --project flag
	if strings.Contains(errOutput, " --project") && !strings.Contains(errOutput, "--source-project") {
		t.Errorf("Error should not mention old --project flag, got: %s", errOutput)
	}
}

func TestDeriveProjectTitle_NoSuffix(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		expected string
	}{
		{
			name:     "simple repo name",
			repoName: "my-app",
			expected: "my-app",
		},
		{
			name:     "hyphenated repo name",
			repoName: "gh-pmu",
			expected: "gh-pmu",
		},
		{
			name:     "single word repo name",
			repoName: "project",
			expected: "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveProjectTitle(tt.repoName)
			if got != tt.expected {
				t.Errorf("deriveProjectTitle(%q) = %q, want %q", tt.repoName, got, tt.expected)
			}
			if strings.Contains(got, "Board") {
				t.Errorf("deriveProjectTitle(%q) should not contain 'Board', got %q", tt.repoName, got)
			}
		})
	}
}

func TestWriteConfig_CreatesJSONCompanion(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &InitConfig{
		ProjectOwner:  "test-owner",
		ProjectNumber: 5,
		Repositories:  []string{"test-owner/test-repo"},
	}

	err := writeConfig(tmpDir, cfg)
	if err != nil {
		t.Fatalf("writeConfig failed: %v", err)
	}

	// Verify JSON companion exists
	jsonPath := filepath.Join(tmpDir, config.ConfigFileName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Expected JSON companion to exist: %v", err)
	}

	// Verify it's valid JSON with expected content
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON companion is not valid JSON: %v", err)
	}

	project, ok := parsed["project"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected JSON to contain project object")
	}
	if project["owner"] != "test-owner" {
		t.Errorf("Expected owner 'test-owner', got %v", project["owner"])
	}
}

// --- Acceptance Preservation Tests ---

// seedAcceptance writes a .gh-pmu.json with acceptance data and the given version.
func seedAcceptance(t *testing.T, dir, version string) {
	t.Helper()
	acc := map[string]interface{}{
		"version": version,
		"project": map[string]interface{}{
			"owner":  "owner",
			"number": 1,
		},
		"repositories": []string{"owner/repo"},
		"acceptance": map[string]interface{}{
			"accepted": true,
			"user":     "testuser",
			"date":     "2026-01-15",
			"version":  version,
		},
	}
	data, err := json.MarshalIndent(acc, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal seed config: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, config.ConfigFileName), data, 0644); err != nil {
		t.Fatalf("Failed to write seed config: %v", err)
	}
}

func TestWriteConfigWithMetadata_PreservesAcceptance_PatchVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Derive seed version from current version with same major.minor but patch 0
	// so that writeConfigWithMetadata sees a patch-only difference and preserves acceptance.
	cur := getVersion()
	parts := strings.SplitN(cur, ".", 3)
	seedVer := "1.0.0" // fallback for dev builds
	if len(parts) >= 2 {
		seedVer = parts[0] + "." + parts[1] + ".0"
	}
	seedAcceptance(t, tmpDir, seedVer)

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}
	meta := &ProjectMetadata{
		ProjectID: "test-project-id",
		Fields:    []FieldMetadata{},
	}

	err := writeConfigWithMetadata(tmpDir, cfg, meta)
	if err != nil {
		t.Fatalf("writeConfigWithMetadata failed: %v", err)
	}

	// Read the JSON output and check acceptance
	jsonPath := filepath.Join(tmpDir, config.ConfigFileName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON is not valid: %v", err)
	}

	acc, ok := parsed["acceptance"]
	if !ok {
		t.Fatal("Expected acceptance section to be preserved on patch version change, but it was missing")
	}

	accMap, ok := acc.(map[string]interface{})
	if !ok {
		t.Fatal("acceptance is not an object")
	}

	if accMap["accepted"] != true {
		t.Errorf("Expected accepted=true, got %v", accMap["accepted"])
	}
	if accMap["user"] != "testuser" {
		t.Errorf("Expected user=testuser, got %v", accMap["user"])
	}
}

func TestWriteConfigWithMetadata_ClearsAcceptance_MajorVersion(t *testing.T) {
	tmpDir := t.TempDir()
	// Seed with version 0.9.0 — any current version 1.x.x is a major bump
	seedAcceptance(t, tmpDir, "0.9.0")

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}
	meta := &ProjectMetadata{
		ProjectID: "test-project-id",
		Fields:    []FieldMetadata{},
	}

	err := writeConfigWithMetadata(tmpDir, cfg, meta)
	if err != nil {
		t.Fatalf("writeConfigWithMetadata failed: %v", err)
	}

	jsonPath := filepath.Join(tmpDir, config.ConfigFileName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON is not valid: %v", err)
	}

	if _, ok := parsed["acceptance"]; ok {
		t.Error("Expected acceptance to be cleared on major version change, but it was present")
	}
}

func TestWriteConfigWithMetadata_ClearsAcceptance_MinorVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Seed with a version that differs in minor from current.
	currentVersion := getVersion()
	// Parse manually to construct a different-minor version
	parts := strings.SplitN(currentVersion, ".", 3)
	if len(parts) >= 2 {
		// Same major, different minor (add 1)
		major := parts[0]
		minor := 999 // guaranteed different
		differentMinor := fmt.Sprintf("%s.%d.0", major, minor)
		seedAcceptance(t, tmpDir, differentMinor)
	} else {
		// Dev build or unparseable — seed with "1.0.0" to guarantee diff
		seedAcceptance(t, tmpDir, "1.0.0")
	}

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}
	meta := &ProjectMetadata{
		ProjectID: "test-project-id",
		Fields:    []FieldMetadata{},
	}

	err := writeConfigWithMetadata(tmpDir, cfg, meta)
	if err != nil {
		t.Fatalf("writeConfigWithMetadata failed: %v", err)
	}

	jsonPath := filepath.Join(tmpDir, config.ConfigFileName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON is not valid: %v", err)
	}

	if _, ok := parsed["acceptance"]; ok {
		t.Error("Expected acceptance to be cleared on minor version change, but it was present")
	}
}

func TestWriteConfigWithMetadata_PreservesAcceptance_SameVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Seed with exactly the current version
	currentVersion := getVersion()
	seedAcceptance(t, tmpDir, currentVersion)

	cfg := &InitConfig{
		ProjectOwner:  "owner",
		ProjectNumber: 1,
		Repositories:  []string{"owner/repo"},
	}
	meta := &ProjectMetadata{
		ProjectID: "test-project-id",
		Fields:    []FieldMetadata{},
	}

	err := writeConfigWithMetadata(tmpDir, cfg, meta)
	if err != nil {
		t.Fatalf("writeConfigWithMetadata failed: %v", err)
	}

	jsonPath := filepath.Join(tmpDir, config.ConfigFileName)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON config: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON is not valid: %v", err)
	}

	acc, ok := parsed["acceptance"]
	if !ok {
		t.Fatal("Expected acceptance section to be preserved on same version re-init, but it was missing")
	}

	accMap, ok := acc.(map[string]interface{})
	if !ok {
		t.Fatal("acceptance is not an object")
	}

	if accMap["accepted"] != true {
		t.Errorf("Expected accepted=true, got %v", accMap["accepted"])
	}
	if accMap["version"] != currentVersion {
		t.Errorf("Expected version=%s, got %v", currentVersion, accMap["version"])
	}
}
