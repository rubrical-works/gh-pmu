package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the .gh-pmu.yml configuration file
type Config struct {
	Version      string            `yaml:"version,omitempty" json:"version,omitempty"`
	Project      Project           `yaml:"project" json:"project"`
	Repositories []string          `yaml:"repositories" json:"repositories"`
	Framework    string            `yaml:"framework,omitempty" json:"framework,omitempty"`
	Defaults     Defaults          `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Fields       map[string]Field  `yaml:"fields,omitempty" json:"fields,omitempty"`
	Triage       map[string]Triage `yaml:"triage,omitempty" json:"triage,omitempty"`
	Release      Release           `yaml:"release,omitempty" json:"release,omitempty"`
	Acceptance   *Acceptance       `yaml:"acceptance,omitempty" json:"acceptance,omitempty"`
	Metadata     *Metadata         `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// Project contains GitHub project configuration
type Project struct {
	Name   string `yaml:"name,omitempty" json:"name,omitempty"`
	Number int    `yaml:"number" json:"number"`
	Owner  string `yaml:"owner" json:"owner"`
}

// Defaults contains default values for new issues
type Defaults struct {
	Priority string   `yaml:"priority,omitempty" json:"priority,omitempty"`
	Status   string   `yaml:"status,omitempty" json:"status,omitempty"`
	Labels   []string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// Field maps field aliases to GitHub project field names and values
type Field struct {
	Field  string            `yaml:"field" json:"field"`
	Values map[string]string `yaml:"values,omitempty" json:"values,omitempty"`
}

// Triage contains configuration for triage rules
type Triage struct {
	Query       string            `yaml:"query" json:"query"`
	Apply       TriageApply       `yaml:"apply,omitempty" json:"apply,omitempty"`
	Interactive TriageInteractive `yaml:"interactive,omitempty" json:"interactive,omitempty"`
}

// TriageApply contains fields to apply during triage
type TriageApply struct {
	Labels []string          `yaml:"labels,omitempty" json:"labels,omitempty"`
	Fields map[string]string `yaml:"fields,omitempty" json:"fields,omitempty"`
}

// TriageInteractive contains interactive prompts for triage
type TriageInteractive struct {
	Status   bool `yaml:"status,omitempty" json:"status,omitempty"`
	Estimate bool `yaml:"estimate,omitempty" json:"estimate,omitempty"`
}

// Metadata contains cached project metadata from GitHub API
type Metadata struct {
	Project ProjectMetadata `yaml:"project,omitempty" json:"project,omitempty"`
	Fields  []FieldMetadata `yaml:"fields,omitempty" json:"fields,omitempty"`
}

// ProjectMetadata contains cached project info
type ProjectMetadata struct {
	ID string `yaml:"id,omitempty" json:"id,omitempty"`
}

// FieldMetadata contains cached field info
type FieldMetadata struct {
	Name     string           `yaml:"name" json:"name"`
	ID       string           `yaml:"id" json:"id"`
	DataType string           `yaml:"data_type" json:"data_type"`
	Options  []OptionMetadata `yaml:"options,omitempty" json:"options,omitempty"`
}

// OptionMetadata contains cached field option info
type OptionMetadata struct {
	Name string `yaml:"name" json:"name"`
	ID   string `yaml:"id" json:"id"`
}

// ConfigFileName is the default (primary) configuration file name
const ConfigFileName = ".gh-pmu.json"

// ConfigFileNameYAML is the legacy YAML configuration file name (fallback)
const ConfigFileNameYAML = ".gh-pmu.yml"

// Load reads and parses a configuration file from the given path.
// Detects format (YAML or JSON) based on file extension.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config file: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	return &cfg, nil
}

// LoadFromDirectory finds and loads the config file from the given directory.
// It searches up the directory tree until it finds a .gh-pmu.json file or
// reaches the filesystem root.
func LoadFromDirectory(dir string) (*Config, error) {
	configPath, err := FindConfigFile(dir)
	if err != nil {
		return nil, err
	}
	return Load(configPath)
}

// LoadFromDirectoryAndNormalize loads the config and normalizes the framework field.
// If the framework field is empty, it sets it to "IDPF" and saves the config.
// This ensures the config file is self-documenting about which framework is in use.
func LoadFromDirectoryAndNormalize(dir string) (*Config, error) {
	configPath, err := FindConfigFile(dir)
	if err != nil {
		return nil, err
	}

	cfg, err := Load(configPath)
	if err != nil {
		return nil, err
	}

	// Normalize: missing framework defaults to IDPF
	if cfg.Framework == "" {
		cfg.Framework = "IDPF"
		if err := cfg.Save(configPath); err != nil {
			// Log warning but don't fail - config is still usable
			// The next save operation will include the framework
			return cfg, nil
		}
	}

	return cfg, nil
}

// FindConfigFile searches for .gh-pmu.json starting from dir and walking up
// the directory tree until found or filesystem root is reached.
// Falls back to .gh-pmu.yml if no JSON file is found.
func FindConfigFile(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// First pass: look for JSON (primary)
	searchDir := dir
	for {
		configPath := filepath.Join(searchDir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			break
		}
		searchDir = parent
	}

	// Second pass: look for YAML fallback
	searchDir = dir
	for {
		configPath := filepath.Join(searchDir, ConfigFileNameYAML)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			break
		}
		searchDir = parent
	}

	return "", fmt.Errorf("no %s found in %s or any parent directory", ConfigFileName, startDir)
}

// Validate checks that required configuration fields are present
func (c *Config) Validate() error {
	if c.Project.Owner == "" {
		return fmt.Errorf("project.owner is required")
	}

	if c.Project.Number == 0 {
		return fmt.Errorf("project.number is required")
	}

	if len(c.Repositories) == 0 {
		return fmt.Errorf("at least one repository is required")
	}

	return nil
}

// ResolveFieldValue maps an alias to its actual GitHub field value.
// If no alias is found, returns the original value unchanged.
func (c *Config) ResolveFieldValue(fieldKey, alias string) string {
	field, ok := c.Fields[fieldKey]
	if !ok {
		return alias
	}

	if actual, ok := field.Values[alias]; ok {
		return actual
	}

	return alias
}

// ValidateFieldValue checks if the given value is a valid alias for the field.
// Returns an error listing available values if the value is not found.
// Returns nil if the field is not configured (allowing pass-through behavior).
func (c *Config) ValidateFieldValue(fieldKey, value string) error {
	field, ok := c.Fields[fieldKey]
	if !ok {
		// Field not configured, allow any value
		return nil
	}

	if len(field.Values) == 0 {
		// No values defined for field, allow any value
		return nil
	}

	// Check if value exists in the field's values map (case-insensitive)
	valueLower := strings.ToLower(value)
	for alias := range field.Values {
		if strings.ToLower(alias) == valueLower {
			return nil
		}
	}

	// Value not found, build error with available values
	var available []string
	for alias := range field.Values {
		available = append(available, alias)
	}

	return fmt.Errorf("invalid %s value %q\nAvailable values: %s", fieldKey, value, strings.Join(available, ", "))
}

// GetFieldName returns the actual GitHub field name for a given key.
// If no mapping exists, returns the original key unchanged.
func (c *Config) GetFieldName(fieldKey string) string {
	field, ok := c.Fields[fieldKey]
	if !ok {
		return fieldKey
	}

	if field.Field != "" {
		return field.Field
	}

	return fieldKey
}

// ApplyEnvOverrides applies environment variable overrides to the config.
// Supported environment variables:
//   - GH_PM_PROJECT_OWNER: overrides project.owner
//   - GH_PM_PROJECT_NUMBER: overrides project.number
func (c *Config) ApplyEnvOverrides() {
	if owner := os.Getenv("GH_PM_PROJECT_OWNER"); owner != "" {
		c.Project.Owner = owner
	}

	if numberStr := os.Getenv("GH_PM_PROJECT_NUMBER"); numberStr != "" {
		if number, err := strconv.Atoi(numberStr); err == nil {
			c.Project.Number = number
		}
	}
}

// Save writes the configuration to the JSON config file.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)

	jsonData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON config: %w", err)
	}
	jsonData = append(jsonData, '\n')

	jsonPath := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// MigrateYAML performs a one-time migration: if .gh-pmu.yml exists alongside
// the JSON config, it deletes the YAML file, updates the version in the JSON
// config, and saves. If no YAML file exists, this is a no-op.
func MigrateYAML(jsonConfigPath string, currentVersion string, w io.Writer) error {
	dir := filepath.Dir(jsonConfigPath)
	yamlPath := filepath.Join(dir, ConfigFileNameYAML)

	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		return nil // No YAML file — nothing to do
	}

	// Delete the legacy YAML config
	if err := os.Remove(yamlPath); err != nil {
		return fmt.Errorf("failed to remove legacy config %s: %w", ConfigFileNameYAML, err)
	}
	fmt.Fprintf(w, "Removed legacy config %s\n", ConfigFileNameYAML)

	// Update version in JSON config
	cfg, err := Load(jsonConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config for version update: %w", err)
	}
	cfg.Version = currentVersion
	if err := cfg.Save(jsonConfigPath); err != nil {
		return fmt.Errorf("failed to save updated config: %w", err)
	}

	return nil
}

// IsIDPF returns true if the config uses IDPF framework validation.
// Returns true for any framework value starting with "IDPF" (case-insensitive),
// including "IDPF", "IDPF-Agile", "idpf", etc.
// IDPF is the default framework when not specified.
func (c *Config) IsIDPF() bool {
	return strings.HasPrefix(strings.ToUpper(c.Framework), "IDPF")
}

// AddFieldMetadata adds or updates field metadata in the config
func (c *Config) AddFieldMetadata(field FieldMetadata) {
	if c.Metadata == nil {
		c.Metadata = &Metadata{}
	}

	// Check if field already exists, update if so
	for i, f := range c.Metadata.Fields {
		if f.Name == field.Name {
			c.Metadata.Fields[i] = field
			return
		}
	}

	// Add new field
	c.Metadata.Fields = append(c.Metadata.Fields, field)
}

// Release contains release management configuration
type Release struct {
	Tracks    map[string]TrackConfig `yaml:"tracks,omitempty" json:"tracks,omitempty"`
	Artifacts *ArtifactConfig        `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Coverage  *CoverageConfig        `yaml:"coverage,omitempty" json:"coverage,omitempty"`
}

// CoverageConfig contains configuration for release coverage gates
type CoverageConfig struct {
	Enabled      *bool    `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Threshold    int      `yaml:"threshold,omitempty" json:"threshold,omitempty"`
	SkipPatterns []string `yaml:"skip_patterns,omitempty" json:"skip_patterns,omitempty"`
}

// ArtifactConfig contains configuration for release artifacts
type ArtifactConfig struct {
	Directory    string `yaml:"directory,omitempty" json:"directory,omitempty"`
	ReleaseNotes bool   `yaml:"release_notes,omitempty" json:"release_notes,omitempty"`
	Changelog    bool   `yaml:"changelog,omitempty" json:"changelog,omitempty"`
}

// TrackConfig contains configuration for a release track
type TrackConfig struct {
	Prefix      string            `yaml:"prefix" json:"prefix"`
	Default     bool              `yaml:"default,omitempty" json:"default,omitempty"`
	Constraints *TrackConstraints `yaml:"constraints,omitempty" json:"constraints,omitempty"`
}

// TrackConstraints contains constraints for a release track
type TrackConstraints struct {
	Version string            `yaml:"version,omitempty" json:"version,omitempty"`
	Labels  *LabelConstraints `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// LabelConstraints contains label requirements for a track
type LabelConstraints struct {
	Required  []string `yaml:"required,omitempty" json:"required,omitempty"`
	Forbidden []string `yaml:"forbidden,omitempty" json:"forbidden,omitempty"`
}

// GetTrackPrefix returns the prefix for a given track name
// Returns "v" for stable track if not configured
func (c *Config) GetTrackPrefix(track string) string {
	if c.Release.Tracks == nil {
		// Default prefixes when not configured
		switch track {
		case "stable", "":
			return "v"
		default:
			return track + "/"
		}
	}

	if cfg, ok := c.Release.Tracks[track]; ok {
		return cfg.Prefix
	}

	// Default for unconfigured tracks
	if track == "stable" || track == "" {
		return "v"
	}
	return track + "/"
}

// GetDefaultTrack returns the default track name
func (c *Config) GetDefaultTrack() string {
	if c.Release.Tracks != nil {
		for name, cfg := range c.Release.Tracks {
			if cfg.Default {
				return name
			}
		}
	}
	return "stable"
}

// GetTrackConstraints returns constraints for a track, or nil if none
func (c *Config) GetTrackConstraints(track string) *TrackConstraints {
	if c.Release.Tracks == nil {
		return nil
	}
	if cfg, ok := c.Release.Tracks[track]; ok {
		return cfg.Constraints
	}
	return nil
}

// FormatReleaseFieldValue formats a version with the track prefix
func (c *Config) FormatReleaseFieldValue(version, track string) string {
	prefix := c.GetTrackPrefix(track)
	return prefix + version
}

// GetArtifactDirectory returns the base artifact directory
func (c *Config) GetArtifactDirectory() string {
	if c.Release.Artifacts != nil && c.Release.Artifacts.Directory != "" {
		return c.Release.Artifacts.Directory
	}
	return "Releases"
}

// GetArtifactPath returns the full artifact path for a release
// For stable: Releases/v1.0.0
// For other tracks: Releases/patch/v1.1.1
func (c *Config) GetArtifactPath(version, track string) string {
	baseDir := c.GetArtifactDirectory()
	if track == "stable" || track == "" {
		return fmt.Sprintf("%s/%s", baseDir, version)
	}
	return fmt.Sprintf("%s/%s/%s", baseDir, track, version)
}

// ShouldGenerateReleaseNotes returns whether release notes should be generated
func (c *Config) ShouldGenerateReleaseNotes() bool {
	if c.Release.Artifacts == nil {
		return true // Default to true
	}
	return c.Release.Artifacts.ReleaseNotes
}

// ShouldGenerateChangelog returns whether changelog should be generated
func (c *Config) ShouldGenerateChangelog() bool {
	if c.Release.Artifacts == nil {
		return true // Default to true
	}
	return c.Release.Artifacts.Changelog
}

// IsCoverageGateEnabled returns whether coverage gate is enabled (default: true)
func (c *Config) IsCoverageGateEnabled() bool {
	if c.Release.Coverage == nil || c.Release.Coverage.Enabled == nil {
		return true // Default to enabled
	}
	return *c.Release.Coverage.Enabled
}

// GetCoverageThreshold returns the minimum patch coverage percentage (default: 80)
func (c *Config) GetCoverageThreshold() int {
	if c.Release.Coverage == nil || c.Release.Coverage.Threshold == 0 {
		return 80 // Default threshold
	}
	return c.Release.Coverage.Threshold
}

// GetCoverageSkipPatterns returns patterns to exclude from coverage analysis
func (c *Config) GetCoverageSkipPatterns() []string {
	if c.Release.Coverage == nil {
		return []string{"*_test.go", "mock_*.go"}
	}
	if len(c.Release.Coverage.SkipPatterns) == 0 {
		return []string{"*_test.go", "mock_*.go"}
	}
	return c.Release.Coverage.SkipPatterns
}

// TempDirName is the name of the temporary directory within the project root
const TempDirName = "tmp"

// GetProjectRoot returns the directory containing .gh-pmu.yml.
// It searches from the current working directory up the directory tree.
func GetProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	configPath, err := FindConfigFile(cwd)
	if err != nil {
		return "", err
	}

	return filepath.Dir(configPath), nil
}

// GetTempDir returns the path to the project's tmp directory and creates it if needed.
// It also ensures tmp/ is in .gitignore.
func GetTempDir() (string, error) {
	projectRoot, err := GetProjectRoot()
	if err != nil {
		return "", err
	}

	tempDir := filepath.Join(projectRoot, TempDirName)

	// Create tmp directory if it doesn't exist
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Ensure tmp/ is in .gitignore
	if err := ensureGitignore(projectRoot); err != nil {
		// Log warning but don't fail - gitignore is nice-to-have
		fmt.Fprintf(os.Stderr, "Warning: could not update .gitignore: %v\n", err)
	}

	return tempDir, nil
}

// CreateTempFile creates a temporary file in the project's tmp directory.
// The pattern follows os.CreateTemp conventions (e.g., "prefix-*.suffix").
// The caller is responsible for closing and removing the file.
func CreateTempFile(pattern string) (*os.File, error) {
	tempDir, err := GetTempDir()
	if err != nil {
		return nil, err
	}

	return os.CreateTemp(tempDir, pattern)
}

// ensureGitignore adds tmp/ to .gitignore if not already present
func ensureGitignore(projectRoot string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	// Read existing content (single read, then close)
	existing, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	// Check if entry already present
	if len(existing) > 0 {
		scanner := bufio.NewScanner(strings.NewReader(string(existing)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == TempDirName || line == TempDirName+"/" {
				return nil // Already present
			}
		}
	}

	// Build content to append
	var content string
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		content = "\n"
	}
	content += TempDirName + "/\n"

	// Write (single handle, then close)
	file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to .gitignore: %w", err)
	}

	return nil
}
