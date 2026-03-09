package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ValidConfig_ReturnsProjectDetails(t *testing.T) {
	// ARRANGE: Path to valid test config
	configPath := filepath.Join("..", "..", "testdata", "config", "valid.gh-pmu.yml")

	// ACT: Load the configuration
	cfg, err := Load(configPath)

	// ASSERT: No error and correct values
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Project.Owner != "rubrical-works" {
		t.Errorf("Expected owner 'rubrical-works', got '%s'", cfg.Project.Owner)
	}

	if cfg.Project.Number != 13 {
		t.Errorf("Expected project number 13, got %d", cfg.Project.Number)
	}
}

func TestLoad_MinimalConfig_ReturnsRequiredFields(t *testing.T) {
	// ARRANGE: Path to minimal test config
	configPath := filepath.Join("..", "..", "testdata", "config", "minimal.gh-pmu.yml")

	// ACT: Load the configuration
	cfg, err := Load(configPath)

	// ASSERT: No error and required fields present
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Project.Owner != "rubrical-works" {
		t.Errorf("Expected owner 'rubrical-works', got '%s'", cfg.Project.Owner)
	}

	if cfg.Project.Number != 13 {
		t.Errorf("Expected project number 13, got %d", cfg.Project.Number)
	}

	if len(cfg.Repositories) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(cfg.Repositories))
	}
}

func TestLoad_MissingFile_ReturnsError(t *testing.T) {
	// ARRANGE: Path to non-existent file
	configPath := filepath.Join("..", "..", "testdata", "config", "does-not-exist.yml")

	// ACT: Load the configuration
	_, err := Load(configPath)

	// ASSERT: Error is returned
	if err == nil {
		t.Fatal("Expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML_ReturnsError(t *testing.T) {
	// ARRANGE: Path to invalid YAML
	configPath := filepath.Join("..", "..", "testdata", "config", "invalid-yaml-syntax.gh-pmu.yml")

	// ACT: Load the configuration
	_, err := Load(configPath)

	// ASSERT: Error is returned
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

func TestValidate_MissingOwner_ReturnsError(t *testing.T) {
	// ARRANGE: Config with missing owner
	cfg := &Config{
		Project: Project{
			Number: 13,
			// Owner is missing
		},
		Repositories: []string{"rubrical-works/gh-pm-test"},
	}

	// ACT: Validate the config
	err := cfg.Validate()

	// ASSERT: Error mentions owner
	if err == nil {
		t.Fatal("Expected validation error for missing owner, got nil")
	}
}

func TestValidate_MissingNumber_ReturnsError(t *testing.T) {
	// ARRANGE: Config with missing project number
	cfg := &Config{
		Project: Project{
			Owner: "rubrical-works",
			// Number is missing (zero value)
		},
		Repositories: []string{"rubrical-works/gh-pm-test"},
	}

	// ACT: Validate the config
	err := cfg.Validate()

	// ASSERT: Error mentions number
	if err == nil {
		t.Fatal("Expected validation error for missing project number, got nil")
	}
}

func TestValidate_MissingRepositories_ReturnsError(t *testing.T) {
	// ARRANGE: Config with no repositories
	cfg := &Config{
		Project: Project{
			Owner:  "rubrical-works",
			Number: 13,
		},
		Repositories: []string{}, // Empty
	}

	// ACT: Validate the config
	err := cfg.Validate()

	// ASSERT: Error mentions repositories
	if err == nil {
		t.Fatal("Expected validation error for missing repositories, got nil")
	}
}

func TestValidate_ValidConfig_ReturnsNil(t *testing.T) {
	// ARRANGE: Valid config
	cfg := &Config{
		Project: Project{
			Owner:  "rubrical-works",
			Number: 13,
		},
		Repositories: []string{"rubrical-works/gh-pm-test"},
	}

	// ACT: Validate the config
	err := cfg.Validate()

	// ASSERT: No error
	if err != nil {
		t.Fatalf("Expected no error for valid config, got: %v", err)
	}
}

func TestResolveFieldValue_WithAlias_ReturnsActualValue(t *testing.T) {
	// ARRANGE: Config with field aliases
	cfg := &Config{
		Fields: map[string]Field{
			"priority": {
				Field: "Priority",
				Values: map[string]string{
					"p0": "P0",
					"p1": "P1",
					"p2": "P2",
				},
			},
		},
	}

	// ACT: Resolve alias
	value := cfg.ResolveFieldValue("priority", "p1")

	// ASSERT: Returns actual value
	if value != "P1" {
		t.Errorf("Expected 'P1', got '%s'", value)
	}
}

func TestResolveFieldValue_NoAlias_ReturnsOriginal(t *testing.T) {
	// ARRANGE: Config with field aliases
	cfg := &Config{
		Fields: map[string]Field{
			"priority": {
				Field: "Priority",
				Values: map[string]string{
					"p0": "P0",
					"p1": "P1",
				},
			},
		},
	}

	// ACT: Try to resolve value that has no alias
	value := cfg.ResolveFieldValue("priority", "Unknown")

	// ASSERT: Returns original value unchanged
	if value != "Unknown" {
		t.Errorf("Expected 'Unknown', got '%s'", value)
	}
}

func TestResolveFieldValue_UnknownField_ReturnsOriginal(t *testing.T) {
	// ARRANGE: Config with no fields configured
	cfg := &Config{
		Fields: map[string]Field{},
	}

	// ACT: Try to resolve unknown field
	value := cfg.ResolveFieldValue("unknown", "some-value")

	// ASSERT: Returns original value unchanged
	if value != "some-value" {
		t.Errorf("Expected 'some-value', got '%s'", value)
	}
}

func TestValidateFieldValue_ValidAlias_ReturnsNil(t *testing.T) {
	// ARRANGE: Config with field aliases
	cfg := &Config{
		Fields: map[string]Field{
			"status": {
				Field: "Status",
				Values: map[string]string{
					"backlog":     "Backlog",
					"in_progress": "In progress",
					"done":        "Done",
				},
			},
		},
	}

	// ACT: Validate valid alias
	err := cfg.ValidateFieldValue("status", "backlog")

	// ASSERT: Returns nil (valid)
	if err != nil {
		t.Errorf("Expected nil error for valid alias, got: %v", err)
	}
}

func TestValidateFieldValue_InvalidValue_ReturnsError(t *testing.T) {
	// ARRANGE: Config with field aliases
	cfg := &Config{
		Fields: map[string]Field{
			"status": {
				Field: "Status",
				Values: map[string]string{
					"backlog":     "Backlog",
					"in_progress": "In progress",
					"done":        "Done",
				},
			},
		},
	}

	// ACT: Validate invalid value
	err := cfg.ValidateFieldValue("status", "nonexistent")

	// ASSERT: Returns error with available values
	if err == nil {
		t.Fatal("Expected error for invalid value, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, `invalid status value "nonexistent"`) {
		t.Errorf("Expected error to contain invalid value message, got: %s", errStr)
	}
	if !strings.Contains(errStr, "Available values:") {
		t.Errorf("Expected error to list available values, got: %s", errStr)
	}
}

func TestValidateFieldValue_FieldNotConfigured_ReturnsNil(t *testing.T) {
	// ARRANGE: Config without the field
	cfg := &Config{
		Fields: map[string]Field{},
	}

	// ACT: Validate value for unconfigured field
	err := cfg.ValidateFieldValue("status", "anything")

	// ASSERT: Returns nil (pass-through behavior)
	if err != nil {
		t.Errorf("Expected nil error for unconfigured field, got: %v", err)
	}
}

func TestValidateFieldValue_CaseInsensitive(t *testing.T) {
	// ARRANGE: Config with lowercase aliases
	cfg := &Config{
		Fields: map[string]Field{
			"status": {
				Field: "Status",
				Values: map[string]string{
					"backlog": "Backlog",
				},
			},
		},
	}

	// ACT: Validate with uppercase input
	err := cfg.ValidateFieldValue("status", "BACKLOG")

	// ASSERT: Returns nil (case-insensitive match)
	if err != nil {
		t.Errorf("Expected nil error for case-insensitive match, got: %v", err)
	}
}

func TestValidateFieldValue_FieldWithNoValues_ReturnsNil(t *testing.T) {
	// ARRANGE: Config with field but no values defined
	cfg := &Config{
		Fields: map[string]Field{
			"status": {
				Field:  "Status",
				Values: map[string]string{}, // Empty values
			},
		},
	}

	// ACT: Validate any value
	err := cfg.ValidateFieldValue("status", "anything")

	// ASSERT: Returns nil (no values to validate against)
	if err != nil {
		t.Errorf("Expected nil error when no values defined, got: %v", err)
	}
}

func TestGetFieldName_WithMapping_ReturnsActualName(t *testing.T) {
	// ARRANGE: Config with field mapping
	cfg := &Config{
		Fields: map[string]Field{
			"priority": {
				Field: "Priority",
			},
			"status": {
				Field: "Status",
			},
		},
	}

	// ACT: Get actual field name
	name := cfg.GetFieldName("priority")

	// ASSERT: Returns mapped name
	if name != "Priority" {
		t.Errorf("Expected 'Priority', got '%s'", name)
	}
}

func TestGetFieldName_NoMapping_ReturnsOriginal(t *testing.T) {
	// ARRANGE: Config with no field mapping
	cfg := &Config{
		Fields: map[string]Field{},
	}

	// ACT: Get field name for unmapped field
	name := cfg.GetFieldName("SomeField")

	// ASSERT: Returns original name
	if name != "SomeField" {
		t.Errorf("Expected 'SomeField', got '%s'", name)
	}
}

func TestLoadFromDirectory_FindsConfigFile(t *testing.T) {
	// ARRANGE: Directory containing valid config
	dir := filepath.Join("..", "..", "testdata", "config")

	// Create a temporary .gh-pmu.yml in testdata/config for this test
	// (We'll use the valid.gh-pmu.yml by copying it)
	testDir := t.TempDir()
	srcPath := filepath.Join(dir, "valid.gh-pmu.yml")
	dstPath := filepath.Join(testDir, ".gh-pmu.yml")

	// Copy the file
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}
	if err := os.WriteFile(dstPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// ACT: Load from directory
	cfg, err := LoadFromDirectory(testDir)

	// ASSERT: Config loaded successfully
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Project.Owner != "rubrical-works" {
		t.Errorf("Expected owner 'rubrical-works', got '%s'", cfg.Project.Owner)
	}
}

func TestLoadFromDirectory_NoConfigFile_ReturnsError(t *testing.T) {
	// ARRANGE: Empty directory
	testDir := t.TempDir()

	// ACT: Try to load from directory with no config
	_, err := LoadFromDirectory(testDir)

	// ASSERT: Error is returned
	if err == nil {
		t.Fatal("Expected error for missing config file, got nil")
	}
}

func TestApplyEnvOverrides_OverridesOwner(t *testing.T) {
	// ARRANGE: Config and env var
	cfg := &Config{
		Project: Project{
			Owner:  "original-owner",
			Number: 13,
		},
	}
	t.Setenv("GH_PM_PROJECT_OWNER", "env-owner")

	// ACT: Apply overrides
	cfg.ApplyEnvOverrides()

	// ASSERT: Owner is overridden
	if cfg.Project.Owner != "env-owner" {
		t.Errorf("Expected owner 'env-owner', got '%s'", cfg.Project.Owner)
	}
}

func TestApplyEnvOverrides_OverridesNumber(t *testing.T) {
	// ARRANGE: Config and env var
	cfg := &Config{
		Project: Project{
			Owner:  "rubrical-works",
			Number: 13,
		},
	}
	t.Setenv("GH_PM_PROJECT_NUMBER", "99")

	// ACT: Apply overrides
	cfg.ApplyEnvOverrides()

	// ASSERT: Number is overridden
	if cfg.Project.Number != 99 {
		t.Errorf("Expected project number 99, got %d", cfg.Project.Number)
	}
}

func TestApplyEnvOverrides_InvalidNumber_Ignored(t *testing.T) {
	// ARRANGE: Config and invalid env var
	cfg := &Config{
		Project: Project{
			Owner:  "rubrical-works",
			Number: 13,
		},
	}
	t.Setenv("GH_PM_PROJECT_NUMBER", "not-a-number")

	// ACT: Apply overrides
	cfg.ApplyEnvOverrides()

	// ASSERT: Number unchanged
	if cfg.Project.Number != 13 {
		t.Errorf("Expected project number 13 (unchanged), got %d", cfg.Project.Number)
	}
}

func TestApplyEnvOverrides_NoEnvVars_Unchanged(t *testing.T) {
	// ARRANGE: Config with no env vars set
	cfg := &Config{
		Project: Project{
			Owner:  "original-owner",
			Number: 13,
		},
	}
	// Ensure env vars are not set
	os.Unsetenv("GH_PM_PROJECT_OWNER")
	os.Unsetenv("GH_PM_PROJECT_NUMBER")

	// ACT: Apply overrides
	cfg.ApplyEnvOverrides()

	// ASSERT: Values unchanged
	if cfg.Project.Owner != "original-owner" {
		t.Errorf("Expected owner 'original-owner', got '%s'", cfg.Project.Owner)
	}
	if cfg.Project.Number != 13 {
		t.Errorf("Expected project number 13, got %d", cfg.Project.Number)
	}
}

func TestFindConfigFile_InCurrentDir_ReturnsPath(t *testing.T) {
	// ARRANGE: Create temp dir with config file
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"project":{"owner":"test","number":1}}`), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// ACT: Find config starting from same dir
	found, err := FindConfigFile(testDir)

	// ASSERT: Found in current dir
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if found != configPath {
		t.Errorf("Expected %s, got %s", configPath, found)
	}
}

func TestFindConfigFile_InParentDir_ReturnsPath(t *testing.T) {
	// ARRANGE: Create nested dirs, config in parent
	parentDir := t.TempDir()
	childDir := filepath.Join(parentDir, "subdir")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	configPath := filepath.Join(parentDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"project":{"owner":"test","number":1}}`), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// ACT: Find config starting from child dir
	found, err := FindConfigFile(childDir)

	// ASSERT: Found in parent dir
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if found != configPath {
		t.Errorf("Expected %s, got %s", configPath, found)
	}
}

func TestFindConfigFile_InGrandparentDir_ReturnsPath(t *testing.T) {
	// ARRANGE: Create deeply nested dirs, config in grandparent
	grandparentDir := t.TempDir()
	parentDir := filepath.Join(grandparentDir, "parent")
	childDir := filepath.Join(parentDir, "child")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dirs: %v", err)
	}
	configPath := filepath.Join(grandparentDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"project":{"owner":"test","number":1}}`), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// ACT: Find config starting from grandchild dir
	found, err := FindConfigFile(childDir)

	// ASSERT: Found in grandparent dir
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if found != configPath {
		t.Errorf("Expected %s, got %s", configPath, found)
	}
}

func TestFindConfigFile_NotFound_ReturnsError(t *testing.T) {
	// ARRANGE: Empty temp dir (no config anywhere in tree)
	testDir := t.TempDir()
	childDir := filepath.Join(testDir, "subdir")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// ACT: Try to find config
	_, err := FindConfigFile(childDir)

	// ASSERT: Error returned
	if err == nil {
		t.Fatal("Expected error when no config file exists, got nil")
	}
}

func TestLoadFromDirectory_FromSubdir_FindsParentConfig(t *testing.T) {
	// ARRANGE: Create nested structure with config in parent
	parentDir := t.TempDir()
	childDir := filepath.Join(parentDir, "subdir", "nested")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dirs: %v", err)
	}

	configContent := `{"project":{"owner":"rubrical-works","number":13},"repositories":["rubrical-works/gh-pmu"]}`
	configPath := filepath.Join(parentDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// ACT: Load from nested child dir
	cfg, err := LoadFromDirectory(childDir)

	// ASSERT: Config loaded from parent
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cfg.Project.Owner != "rubrical-works" {
		t.Errorf("Expected owner 'rubrical-works', got '%s'", cfg.Project.Owner)
	}
	if cfg.Project.Number != 13 {
		t.Errorf("Expected number 13, got %d", cfg.Project.Number)
	}
}

// ============================================================================
// Save Tests
// ============================================================================

func TestConfig_Save_Success(t *testing.T) {
	// ARRANGE: Create temp dir and config
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)

	cfg := &Config{
		Project: Project{
			Owner:  "test-owner",
			Number: 42,
		},
		Repositories: []string{"test-owner/test-repo"},
	}

	// ACT: Save config
	err := cfg.Save(configPath)

	// ASSERT: File saved correctly
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file was created and can be loaded
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}
	if loadedCfg.Project.Owner != "test-owner" {
		t.Errorf("Expected owner 'test-owner', got '%s'", loadedCfg.Project.Owner)
	}
	if loadedCfg.Project.Number != 42 {
		t.Errorf("Expected number 42, got %d", loadedCfg.Project.Number)
	}
}

func TestConfig_Save_WithMetadata(t *testing.T) {
	// ARRANGE: Config with metadata
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)

	cfg := &Config{
		Project: Project{
			Owner:  "test-owner",
			Number: 1,
		},
		Repositories: []string{"test-owner/test-repo"},
		Metadata: &Metadata{
			Project: ProjectMetadata{ID: "PVT_test123"},
			Fields: []FieldMetadata{
				{Name: "Status", ID: "F1", DataType: "SINGLE_SELECT"},
				{Name: "PRD", ID: "F2", DataType: "TEXT"},
			},
		},
	}

	// ACT: Save config
	err := cfg.Save(configPath)

	// ASSERT: Metadata preserved
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}
	if loadedCfg.Metadata == nil {
		t.Fatal("Expected metadata to be preserved")
	}
	if loadedCfg.Metadata.Project.ID != "PVT_test123" {
		t.Errorf("Expected project ID 'PVT_test123', got '%s'", loadedCfg.Metadata.Project.ID)
	}
	if len(loadedCfg.Metadata.Fields) != 2 {
		t.Errorf("Expected 2 fields in metadata, got %d", len(loadedCfg.Metadata.Fields))
	}
}

func TestConfig_Save_WithVersion_RoundTrip(t *testing.T) {
	// ARRANGE: Config with version field
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)

	cfg := &Config{
		Version: "0.16.0",
		Project: Project{
			Owner:  "test-owner",
			Number: 1,
		},
		Repositories: []string{"test-owner/test-repo"},
	}

	// ACT: Save and reload
	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	// ASSERT: Version preserved through round-trip
	if loadedCfg.Version != "0.16.0" {
		t.Errorf("Expected version '0.16.0', got '%s'", loadedCfg.Version)
	}
}

func TestConfig_Load_WithoutVersion_BackwardCompatible(t *testing.T) {
	// ARRANGE: Config JSON without version field (existing configs)
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	configContent := `{"project":{"owner":"test-owner","number":1},"repositories":["test-owner/test-repo"]}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// ACT: Load config without version field
	cfg, err := Load(configPath)

	// ASSERT: Loads without error, version is empty string
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cfg.Version != "" {
		t.Errorf("Expected empty version for config without version field, got '%s'", cfg.Version)
	}
}

func TestConfig_Load_WithVersion_ReadsCorrectly(t *testing.T) {
	// ARRANGE: Config JSON with version field
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	configContent := `{"version":"1.0.0","project":{"owner":"test-owner","number":1},"repositories":["test-owner/test-repo"]}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// ACT: Load config with version field
	cfg, err := Load(configPath)

	// ASSERT: Version read correctly
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cfg.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", cfg.Version)
	}
}

func TestConfig_Save_InvalidPath(t *testing.T) {
	// ARRANGE: Config with invalid path
	cfg := &Config{
		Project: Project{Owner: "test", Number: 1},
	}

	// ACT: Try to save to invalid path
	err := cfg.Save("/nonexistent/directory/config.yml")

	// ASSERT: Error returned
	if err == nil {
		t.Fatal("Expected error for invalid path, got nil")
	}
}

// ============================================================================
// AddFieldMetadata Tests
// ============================================================================

func TestConfig_AddFieldMetadata_NewField(t *testing.T) {
	// ARRANGE: Config without metadata
	cfg := &Config{
		Project: Project{Owner: "test", Number: 1},
	}

	field := FieldMetadata{
		Name:     "PRD",
		ID:       "PVTF_test",
		DataType: "TEXT",
	}

	// ACT: Add field metadata
	cfg.AddFieldMetadata(field)

	// ASSERT: Metadata created and field added
	if cfg.Metadata == nil {
		t.Fatal("Expected metadata to be created")
	}
	if len(cfg.Metadata.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(cfg.Metadata.Fields))
	}
	if cfg.Metadata.Fields[0].Name != "PRD" {
		t.Errorf("Expected field name 'PRD', got '%s'", cfg.Metadata.Fields[0].Name)
	}
}

func TestConfig_AddFieldMetadata_UpdateExisting(t *testing.T) {
	// ARRANGE: Config with existing field
	cfg := &Config{
		Project: Project{Owner: "test", Number: 1},
		Metadata: &Metadata{
			Fields: []FieldMetadata{
				{Name: "PRD", ID: "old-id", DataType: "TEXT"},
			},
		},
	}

	updatedField := FieldMetadata{
		Name:     "PRD",
		ID:       "new-id",
		DataType: "TEXT",
	}

	// ACT: Add same field with different ID
	cfg.AddFieldMetadata(updatedField)

	// ASSERT: Field updated, not duplicated
	if len(cfg.Metadata.Fields) != 1 {
		t.Fatalf("Expected 1 field (no duplicates), got %d", len(cfg.Metadata.Fields))
	}
	if cfg.Metadata.Fields[0].ID != "new-id" {
		t.Errorf("Expected field ID 'new-id', got '%s'", cfg.Metadata.Fields[0].ID)
	}
}

func TestConfig_AddFieldMetadata_MultipleFields(t *testing.T) {
	// ARRANGE: Empty config
	cfg := &Config{
		Project: Project{Owner: "test", Number: 1},
	}

	// ACT: Add multiple fields
	cfg.AddFieldMetadata(FieldMetadata{Name: "Field1", ID: "F1", DataType: "TEXT"})
	cfg.AddFieldMetadata(FieldMetadata{Name: "Field2", ID: "F2", DataType: "NUMBER"})
	cfg.AddFieldMetadata(FieldMetadata{Name: "Field3", ID: "F3", DataType: "SINGLE_SELECT"})

	// ASSERT: All fields added
	if len(cfg.Metadata.Fields) != 3 {
		t.Fatalf("Expected 3 fields, got %d", len(cfg.Metadata.Fields))
	}
}

func TestConfig_AddFieldMetadata_WithOptions(t *testing.T) {
	// ARRANGE: Empty config
	cfg := &Config{
		Project: Project{Owner: "test", Number: 1},
	}

	field := FieldMetadata{
		Name:     "Environment",
		ID:       "PVTSSF_test",
		DataType: "SINGLE_SELECT",
		Options: []OptionMetadata{
			{Name: "Development", ID: "opt1"},
			{Name: "Production", ID: "opt2"},
		},
	}

	// ACT: Add field with options
	cfg.AddFieldMetadata(field)

	// ASSERT: Options preserved
	if len(cfg.Metadata.Fields[0].Options) != 2 {
		t.Fatalf("Expected 2 options, got %d", len(cfg.Metadata.Fields[0].Options))
	}
	if cfg.Metadata.Fields[0].Options[0].Name != "Development" {
		t.Errorf("Expected first option 'Development', got '%s'", cfg.Metadata.Fields[0].Options[0].Name)
	}
}

func TestGetTrackPrefix_DefaultStable(t *testing.T) {
	cfg := &Config{}
	prefix := cfg.GetTrackPrefix("stable")
	if prefix != "v" {
		t.Errorf("expected 'v' for stable track, got '%s'", prefix)
	}
}

func TestGetTrackPrefix_DefaultPatch(t *testing.T) {
	cfg := &Config{}
	prefix := cfg.GetTrackPrefix("patch")
	if prefix != "patch/" {
		t.Errorf("expected 'patch/' for patch track, got '%s'", prefix)
	}
}

func TestGetTrackPrefix_Configured(t *testing.T) {
	cfg := &Config{
		Release: Release{
			Tracks: map[string]TrackConfig{
				"beta": {Prefix: "beta-"},
			},
		},
	}
	prefix := cfg.GetTrackPrefix("beta")
	if prefix != "beta-" {
		t.Errorf("expected 'beta-' for beta track, got '%s'", prefix)
	}
}

func TestGetDefaultTrack_NoConfig(t *testing.T) {
	cfg := &Config{}
	track := cfg.GetDefaultTrack()
	if track != "stable" {
		t.Errorf("expected 'stable' as default track, got '%s'", track)
	}
}

func TestGetDefaultTrack_Configured(t *testing.T) {
	cfg := &Config{
		Release: Release{
			Tracks: map[string]TrackConfig{
				"stable": {Prefix: "v"},
				"beta":   {Prefix: "beta/", Default: true},
			},
		},
	}
	track := cfg.GetDefaultTrack()
	if track != "beta" {
		t.Errorf("expected 'beta' as default track, got '%s'", track)
	}
}

func TestFormatReleaseFieldValue(t *testing.T) {
	cfg := &Config{
		Release: Release{
			Tracks: map[string]TrackConfig{
				"stable": {Prefix: "v"},
				"patch":  {Prefix: "patch/"},
			},
		},
	}

	tests := []struct {
		version string
		track   string
		want    string
	}{
		{"1.2.0", "stable", "v1.2.0"},
		{"1.1.1", "patch", "patch/1.1.1"},
	}

	for _, tt := range tests {
		result := cfg.FormatReleaseFieldValue(tt.version, tt.track)
		if result != tt.want {
			t.Errorf("FormatReleaseFieldValue(%s, %s) = %s, want %s", tt.version, tt.track, result, tt.want)
		}
	}
}

// TestGetArtifactDirectory tests artifact directory configuration
func TestGetArtifactDirectory(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected string
	}{
		{
			name:     "default directory when nil",
			cfg:      &Config{},
			expected: "Releases",
		},
		{
			name: "custom directory",
			cfg: &Config{
				Release: Release{
					Artifacts: &ArtifactConfig{
						Directory: "dist/releases",
					},
				},
			},
			expected: "dist/releases",
		},
		{
			name: "default when empty string",
			cfg: &Config{
				Release: Release{
					Artifacts: &ArtifactConfig{
						Directory: "",
					},
				},
			},
			expected: "Releases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.GetArtifactDirectory()
			if result != tt.expected {
				t.Errorf("GetArtifactDirectory() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestShouldGenerateReleaseNotes tests release notes generation config
func TestShouldGenerateReleaseNotes(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected bool
	}{
		{
			name:     "default true when nil",
			cfg:      &Config{},
			expected: true,
		},
		{
			name: "explicit true",
			cfg: &Config{
				Release: Release{
					Artifacts: &ArtifactConfig{
						ReleaseNotes: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "explicit false",
			cfg: &Config{
				Release: Release{
					Artifacts: &ArtifactConfig{
						ReleaseNotes: false,
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.ShouldGenerateReleaseNotes()
			if result != tt.expected {
				t.Errorf("ShouldGenerateReleaseNotes() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestShouldGenerateChangelog tests changelog generation config
func TestShouldGenerateChangelog(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected bool
	}{
		{
			name:     "default true when nil",
			cfg:      &Config{},
			expected: true,
		},
		{
			name: "explicit true",
			cfg: &Config{
				Release: Release{
					Artifacts: &ArtifactConfig{
						Changelog: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "explicit false",
			cfg: &Config{
				Release: Release{
					Artifacts: &ArtifactConfig{
						Changelog: false,
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.ShouldGenerateChangelog()
			if result != tt.expected {
				t.Errorf("ShouldGenerateChangelog() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ============================================================================
// IsIDPF Tests
// ============================================================================

func TestConfig_IsIDPF_WithIDPF(t *testing.T) {
	cfg := &Config{Framework: "IDPF"}
	if !cfg.IsIDPF() {
		t.Error("Expected IsIDPF() to return true for 'IDPF'")
	}
}

func TestConfig_IsIDPF_WithLowercase(t *testing.T) {
	cfg := &Config{Framework: "idpf"}
	if !cfg.IsIDPF() {
		t.Error("Expected IsIDPF() to return true for 'idpf'")
	}
}

func TestConfig_IsIDPF_WithNone(t *testing.T) {
	cfg := &Config{Framework: "none"}
	if cfg.IsIDPF() {
		t.Error("Expected IsIDPF() to return false for 'none'")
	}
}

func TestConfig_IsIDPF_WithEmpty(t *testing.T) {
	cfg := &Config{Framework: ""}
	if cfg.IsIDPF() {
		t.Error("Expected IsIDPF() to return false for empty string")
	}
}

func TestConfig_IsIDPF_WithIDPFAgile(t *testing.T) {
	cfg := &Config{Framework: "IDPF-Agile"}
	if !cfg.IsIDPF() {
		t.Error("Expected IsIDPF() to return true for 'IDPF-Agile'")
	}
}

func TestConfig_IsIDPF_WithIDPFAgileLowercase(t *testing.T) {
	cfg := &Config{Framework: "idpf-agile"}
	if !cfg.IsIDPF() {
		t.Error("Expected IsIDPF() to return true for 'idpf-agile'")
	}
}

func TestConfig_IsIDPF_WithMixedCase(t *testing.T) {
	cfg := &Config{Framework: "Idpf"}
	if !cfg.IsIDPF() {
		t.Error("Expected IsIDPF() to return true for 'Idpf'")
	}
}

// ============================================================================
// LoadFromDirectoryAndNormalize Tests
// ============================================================================

func TestLoadFromDirectoryAndNormalize_NormalizesEmptyFramework(t *testing.T) {
	// ARRANGE: Create temp dir with config without framework
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	configContent := `{"project":{"owner":"test-owner","number":1},"repositories":["test-owner/test-repo"]}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// ACT: Load and normalize
	cfg, err := LoadFromDirectoryAndNormalize(testDir)

	// ASSERT: Framework is set to IDPF
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cfg.Framework != "IDPF" {
		t.Errorf("Expected framework 'IDPF', got '%s'", cfg.Framework)
	}

	// ASSERT: File was updated
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
	if loadedCfg.Framework != "IDPF" {
		t.Errorf("Expected saved framework 'IDPF', got '%s'", loadedCfg.Framework)
	}
}

func TestLoadFromDirectoryAndNormalize_PreservesExistingFramework(t *testing.T) {
	// ARRANGE: Create temp dir with config with framework: none
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	configContent := `{"project":{"owner":"test-owner","number":1},"repositories":["test-owner/test-repo"],"framework":"none"}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// ACT: Load and normalize
	cfg, err := LoadFromDirectoryAndNormalize(testDir)

	// ASSERT: Framework is preserved as 'none'
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cfg.Framework != "none" {
		t.Errorf("Expected framework 'none', got '%s'", cfg.Framework)
	}
}

// ============================================================================
// Temp File Handling Tests
// ============================================================================

func TestGetTempDir_CreatesTmpDirectory(t *testing.T) {
	// ARRANGE: Create temp dir with config file and change to it
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"project":{"owner":"test","number":1}}`), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Save current dir and change to test dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// ACT: Get temp dir
	tempDir, err := GetTempDir()

	// ASSERT: Temp dir created
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedPath := filepath.Join(testDir, TempDirName)
	if tempDir != expectedPath {
		t.Errorf("Expected temp dir '%s', got '%s'", expectedPath, tempDir)
	}

	// Verify directory exists
	info, err := os.Stat(tempDir)
	if err != nil {
		t.Fatalf("Temp directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected temp path to be a directory")
	}
}

func TestGetTempDir_AddsToGitignore(t *testing.T) {
	// ARRANGE: Create temp dir with config file
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"project":{"owner":"test","number":1}}`), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Save current dir and change to test dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// ACT: Get temp dir (should create .gitignore entry)
	_, err = GetTempDir()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// ASSERT: .gitignore contains tmp/
	gitignorePath := filepath.Join(testDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Expected .gitignore to be created: %v", err)
	}

	content := string(data)
	if content != "tmp/\n" {
		t.Errorf("Expected .gitignore to contain 'tmp/', got '%s'", content)
	}
}

func TestGetTempDir_DoesNotDuplicateGitignoreEntry(t *testing.T) {
	// ARRANGE: Create temp dir with config file and existing .gitignore
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"project":{"owner":"test","number":1}}`), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	gitignorePath := filepath.Join(testDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("tmp/\n"), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Save current dir and change to test dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// ACT: Get temp dir twice
	_, _ = GetTempDir()
	_, _ = GetTempDir()

	// ASSERT: .gitignore still only has one entry
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	content := string(data)
	if content != "tmp/\n" {
		t.Errorf("Expected .gitignore to contain only one 'tmp/' entry, got '%s'", content)
	}
}

func TestGetTempDir_AppendsToExistingGitignore(t *testing.T) {
	// ARRANGE: Create temp dir with config file and existing .gitignore with other entries
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"project":{"owner":"test","number":1}}`), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	gitignorePath := filepath.Join(testDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("node_modules/\n.env\n"), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Save current dir and change to test dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// ACT: Get temp dir
	_, err = GetTempDir()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// ASSERT: .gitignore has original entries plus tmp/
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	content := string(data)
	expected := "node_modules/\n.env\ntmp/\n"
	if content != expected {
		t.Errorf("Expected .gitignore content '%s', got '%s'", expected, content)
	}
}

func TestCreateTempFile_CreatesFileInTmpDir(t *testing.T) {
	// ARRANGE: Create temp dir with config file
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"project":{"owner":"test","number":1}}`), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Save current dir and change to test dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// ACT: Create temp file
	file, err := CreateTempFile("test-*.txt")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	// ASSERT: File is in tmp directory
	expectedDir := filepath.Join(testDir, TempDirName)
	if filepath.Dir(file.Name()) != expectedDir {
		t.Errorf("Expected file in '%s', got '%s'", expectedDir, filepath.Dir(file.Name()))
	}

	// Verify file exists
	if _, err := os.Stat(file.Name()); err != nil {
		t.Errorf("Temp file should exist: %v", err)
	}
}

func TestCreateTempFile_UsesPattern(t *testing.T) {
	// ARRANGE: Create temp dir with config file
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"project":{"owner":"test","number":1}}`), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Save current dir and change to test dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// ACT: Create temp file with pattern
	file, err := CreateTempFile("gh-pmu-issue-*.md")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	// ASSERT: File name matches pattern
	filename := filepath.Base(file.Name())
	if len(filename) < len("gh-pmu-issue-.md") {
		t.Errorf("Filename should be longer than pattern base, got '%s'", filename)
	}
	if filename[:13] != "gh-pmu-issue-" {
		t.Errorf("Filename should start with 'gh-pmu-issue-', got '%s'", filename)
	}
	if filename[len(filename)-3:] != ".md" {
		t.Errorf("Filename should end with '.md', got '%s'", filename)
	}
}

func TestGetTempDir_NoConfigFile_ReturnsError(t *testing.T) {
	// ARRANGE: Empty temp dir with no config file
	testDir := t.TempDir()

	// Save current dir and change to test dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// ACT: Try to get temp dir
	_, err = GetTempDir()

	// ASSERT: Error returned
	if err == nil {
		t.Fatal("Expected error when no config file exists, got nil")
	}
}

func TestCreateTempFile_NoConfigFile_ReturnsError(t *testing.T) {
	// ARRANGE: Empty temp dir with no config file
	testDir := t.TempDir()

	// Save current dir and change to test dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// ACT: Try to create temp file
	_, err = CreateTempFile("test-*.txt")

	// ASSERT: Error returned
	if err == nil {
		t.Fatal("Expected error when no config file exists, got nil")
	}
}

func TestEnsureGitignore_HandlesNoTrailingNewline(t *testing.T) {
	// ARRANGE: Create temp dir with .gitignore without trailing newline
	testDir := t.TempDir()
	gitignorePath := filepath.Join(testDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("node_modules/"), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// ACT: Call ensureGitignore
	err := ensureGitignore(testDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// ASSERT: .gitignore has newline before tmp/
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	content := string(data)
	expected := "node_modules/\ntmp/\n"
	if content != expected {
		t.Errorf("Expected .gitignore content '%s', got '%s'", expected, content)
	}
}

func TestEnsureGitignore_RecognizesTmpWithoutSlash(t *testing.T) {
	// ARRANGE: Create temp dir with .gitignore containing "tmp" without slash
	testDir := t.TempDir()
	gitignorePath := filepath.Join(testDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("tmp\n"), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// ACT: Call ensureGitignore
	err := ensureGitignore(testDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// ASSERT: .gitignore is not modified (tmp already present)
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	content := string(data)
	if content != "tmp\n" {
		t.Errorf("Expected .gitignore to remain unchanged, got '%s'", content)
	}
}

// ============================================================================
// Coverage Configuration Tests
// ============================================================================

func TestIsCoverageGateEnabled_DefaultsToTrue(t *testing.T) {
	// ARRANGE: Config with no coverage section
	cfg := &Config{}

	// ACT & ASSERT: Should default to true
	if !cfg.IsCoverageGateEnabled() {
		t.Error("Expected coverage gate to be enabled by default")
	}
}

func TestIsCoverageGateEnabled_WithNilEnabled(t *testing.T) {
	// ARRANGE: Config with coverage section but nil Enabled
	cfg := &Config{
		Release: Release{
			Coverage: &CoverageConfig{
				Threshold: 85,
			},
		},
	}

	// ACT & ASSERT: Should default to true when Enabled is nil
	if !cfg.IsCoverageGateEnabled() {
		t.Error("Expected coverage gate to be enabled when Enabled is nil")
	}
}

func TestIsCoverageGateEnabled_ExplicitlyDisabled(t *testing.T) {
	// ARRANGE: Config with coverage explicitly disabled
	enabled := false
	cfg := &Config{
		Release: Release{
			Coverage: &CoverageConfig{
				Enabled: &enabled,
			},
		},
	}

	// ACT & ASSERT: Should be disabled
	if cfg.IsCoverageGateEnabled() {
		t.Error("Expected coverage gate to be disabled")
	}
}

func TestIsCoverageGateEnabled_ExplicitlyEnabled(t *testing.T) {
	// ARRANGE: Config with coverage explicitly enabled
	enabled := true
	cfg := &Config{
		Release: Release{
			Coverage: &CoverageConfig{
				Enabled: &enabled,
			},
		},
	}

	// ACT & ASSERT: Should be enabled
	if !cfg.IsCoverageGateEnabled() {
		t.Error("Expected coverage gate to be enabled")
	}
}

func TestGetCoverageThreshold_DefaultsTo80(t *testing.T) {
	// ARRANGE: Config with no coverage section
	cfg := &Config{}

	// ACT & ASSERT: Should default to 80
	if threshold := cfg.GetCoverageThreshold(); threshold != 80 {
		t.Errorf("Expected default threshold 80, got %d", threshold)
	}
}

func TestGetCoverageThreshold_WithZeroValue(t *testing.T) {
	// ARRANGE: Config with coverage section but zero threshold
	cfg := &Config{
		Release: Release{
			Coverage: &CoverageConfig{
				Threshold: 0,
			},
		},
	}

	// ACT & ASSERT: Should default to 80 when threshold is 0
	if threshold := cfg.GetCoverageThreshold(); threshold != 80 {
		t.Errorf("Expected default threshold 80, got %d", threshold)
	}
}

func TestGetCoverageThreshold_CustomValue(t *testing.T) {
	// ARRANGE: Config with custom threshold
	cfg := &Config{
		Release: Release{
			Coverage: &CoverageConfig{
				Threshold: 90,
			},
		},
	}

	// ACT & ASSERT: Should return custom value
	if threshold := cfg.GetCoverageThreshold(); threshold != 90 {
		t.Errorf("Expected threshold 90, got %d", threshold)
	}
}

func TestGetCoverageSkipPatterns_DefaultPatterns(t *testing.T) {
	// ARRANGE: Config with no coverage section
	cfg := &Config{}

	// ACT
	patterns := cfg.GetCoverageSkipPatterns()

	// ASSERT: Should return default patterns
	if len(patterns) != 2 {
		t.Errorf("Expected 2 default patterns, got %d", len(patterns))
	}
	if patterns[0] != "*_test.go" {
		t.Errorf("Expected first pattern '*_test.go', got '%s'", patterns[0])
	}
	if patterns[1] != "mock_*.go" {
		t.Errorf("Expected second pattern 'mock_*.go', got '%s'", patterns[1])
	}
}

func TestGetCoverageSkipPatterns_EmptyPatterns(t *testing.T) {
	// ARRANGE: Config with coverage section but empty patterns
	cfg := &Config{
		Release: Release{
			Coverage: &CoverageConfig{
				SkipPatterns: []string{},
			},
		},
	}

	// ACT
	patterns := cfg.GetCoverageSkipPatterns()

	// ASSERT: Should return default patterns when empty
	if len(patterns) != 2 {
		t.Errorf("Expected 2 default patterns, got %d", len(patterns))
	}
}

func TestGetCoverageSkipPatterns_CustomPatterns(t *testing.T) {
	// ARRANGE: Config with custom patterns
	cfg := &Config{
		Release: Release{
			Coverage: &CoverageConfig{
				SkipPatterns: []string{"*_generated.go", "vendor/*"},
			},
		},
	}

	// ACT
	patterns := cfg.GetCoverageSkipPatterns()

	// ASSERT: Should return custom patterns
	if len(patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(patterns))
	}
	if patterns[0] != "*_generated.go" {
		t.Errorf("Expected first pattern '*_generated.go', got '%s'", patterns[0])
	}
	if patterns[1] != "vendor/*" {
		t.Errorf("Expected second pattern 'vendor/*', got '%s'", patterns[1])
	}
}

func TestCoverageConfig_JSONParsing(t *testing.T) {
	// ARRANGE: JSON config with coverage section
	jsonContent := `{"project":{"owner":"test","number":1},"repositories":["test/repo"],"release":{"coverage":{"enabled":false,"threshold":85,"skip_patterns":["*_generated.go","mock_*.go"]}}}`
	testDir := t.TempDir()
	configPath := filepath.Join(testDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// ACT
	cfg, err := Load(configPath)

	// ASSERT
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.IsCoverageGateEnabled() {
		t.Error("Expected coverage gate to be disabled")
	}

	if threshold := cfg.GetCoverageThreshold(); threshold != 85 {
		t.Errorf("Expected threshold 85, got %d", threshold)
	}

	patterns := cfg.GetCoverageSkipPatterns()
	if len(patterns) != 2 || patterns[0] != "*_generated.go" {
		t.Errorf("Unexpected patterns: %v", patterns)
	}
}

// ============================================================================
// Config File Protection Test
// ============================================================================

// TestRealConfigFileNotCorrupted verifies that the real .gh-pmu.json file
// at the project root has not been corrupted by tests writing test data to it.
// This test acts as a canary to detect when test isolation fails.
func TestRealConfigFileNotCorrupted(t *testing.T) {
	// Find the real config file at project root
	cwd, err := os.Getwd()
	if err != nil {
		t.Skipf("Could not get current directory: %v", err)
	}

	// Walk up to find project root (where .gh-pmu.json should be)
	configPath, err := FindConfigFile(cwd)
	if err != nil {
		t.Skipf("No .gh-pmu.json found in path: %v", err)
	}

	// Read the config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Skipf("Could not read config file: %v", err)
	}

	// Verify it contains the real project owner, not test data
	if strings.Contains(string(content), "testowner") {
		t.Error("Real .gh-pmu.json contains 'testowner' - tests have corrupted the config file! " +
			"Tests that call cfg.Save() must use setupBranchTestDir for isolation.")
	}

	// Verify it contains expected owner
	if !strings.Contains(string(content), "rubrical-works") {
		t.Error("Real config does not contain 'rubrical-works' - the config may be corrupted")
	}
}

func TestSave_WritesBothJSONAndYAML(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, ConfigFileName)

	cfg := &Config{
		Project: Project{
			Owner:  "test-owner",
			Number: 1,
		},
		Repositories: []string{"test-owner/test-repo"},
	}

	// ACT: Save config
	err := cfg.Save(jsonPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// ASSERT: Both files exist
	yamlPath := filepath.Join(tmpDir, ConfigFileNameYAML)
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("Expected .gh-pmu.json to exist")
	}
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		t.Error("Expected .gh-pmu.yml to exist")
	}
}

func TestSave_BothFormatsContainExpectedData(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, ConfigFileName)

	cfg := &Config{
		Project: Project{
			Owner:  "test-owner",
			Number: 42,
		},
		Repositories: []string{"test-owner/test-repo"},
		Framework:    "IDPF-Agile",
	}

	// ACT: Save config
	if err := cfg.Save(jsonPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// ASSERT: JSON primary contains expected fields
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}

	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "test-owner") {
		t.Error("JSON file should contain project owner")
	}
	if !strings.Contains(jsonStr, "42") {
		t.Error("JSON file should contain project number")
	}
	if !strings.Contains(jsonStr, "IDPF-Agile") {
		t.Error("JSON file should contain framework")
	}

	// ASSERT: YAML companion contains expected fields
	yamlPath := filepath.Join(tmpDir, ConfigFileNameYAML)
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}

	yamlStr := string(yamlData)
	if !strings.Contains(yamlStr, "test-owner") {
		t.Error("YAML file should contain project owner")
	}
}

func TestFindConfigFile_FallsBackToYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Only create YAML file, no JSON
	yamlPath := filepath.Join(tmpDir, ".gh-pmu.yml")
	if err := os.WriteFile(yamlPath, []byte("project:\n  owner: test\n  number: 1\n"), 0644); err != nil {
		t.Fatalf("Failed to write YAML config: %v", err)
	}

	// ACT: FindConfigFile should find YAML as fallback
	found, err := FindConfigFile(tmpDir)
	if err != nil {
		t.Fatalf("Expected FindConfigFile to find YAML fallback, got error: %v", err)
	}

	if !strings.HasSuffix(found, ".gh-pmu.yml") {
		t.Errorf("Expected YAML path, got: %s", found)
	}
}

func TestFindConfigFile_JSONTakesPrecedence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create both files
	yamlPath := filepath.Join(tmpDir, ".gh-pmu.yml")
	jsonPath := filepath.Join(tmpDir, ".gh-pmu.json")
	if err := os.WriteFile(yamlPath, []byte("project:\n  owner: yaml-owner\n  number: 1\n"), 0644); err != nil {
		t.Fatalf("Failed to write YAML: %v", err)
	}
	if err := os.WriteFile(jsonPath, []byte(`{"project":{"owner":"json-owner","number":1}}`), 0644); err != nil {
		t.Fatalf("Failed to write JSON: %v", err)
	}

	// ACT: FindConfigFile should find JSON (primary)
	found, err := FindConfigFile(tmpDir)
	if err != nil {
		t.Fatalf("FindConfigFile failed: %v", err)
	}

	if !strings.HasSuffix(found, ".gh-pmu.json") {
		t.Errorf("Expected JSON to take precedence, got: %s", found)
	}
}
