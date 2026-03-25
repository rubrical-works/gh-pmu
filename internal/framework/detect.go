package framework

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ghPmuConfig represents the relevant parts of .gh-pmu.json for framework detection
type ghPmuConfig struct {
	Framework string `json:"framework"`
}

// frameworkConfig represents the relevant parts of framework-config.json
type frameworkConfig struct {
	ProjectType struct {
		ProcessFramework string `json:"processFramework"`
	} `json:"projectType"`
}

// DetectFramework detects the framework from config files in the given directory.
// It checks in order:
//  1. .gh-pmu.json - framework field
//  2. framework-config.json - projectType.processFramework field
//
// Returns empty string if no framework is configured (no restriction applied).
func DetectFramework(dir string) (string, error) {
	// Check .gh-pmu.json first (takes precedence)
	framework, err := detectFromGhPmuJson(dir)
	if err == nil && framework != "" {
		return framework, nil
	}

	// Fall back to framework-config.json
	framework, err = detectFromFrameworkConfigJson(dir)
	if err == nil && framework != "" {
		return framework, nil
	}

	// No framework configured - return empty (no restriction)
	return "", nil
}

// detectFromGhPmuJson reads the framework field from .gh-pmu.json
func detectFromGhPmuJson(dir string) (string, error) {
	path := filepath.Join(dir, ".gh-pmu.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var cfg ghPmuConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}

	return cfg.Framework, nil
}

// detectFromFrameworkConfigJson reads the processFramework field from framework-config.json
func detectFromFrameworkConfigJson(dir string) (string, error) {
	path := filepath.Join(dir, "framework-config.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var cfg frameworkConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}

	return cfg.ProjectType.ProcessFramework, nil
}

// CommandRestrictionError represents an error when a command is not applicable for a framework
type CommandRestrictionError struct {
	Framework  string
	Command    string
	Suggestion string
}

func (e *CommandRestrictionError) Error() string {
	return fmt.Sprintf("%s commands not applicable for %s. Use `gh pmu %s`", e.Command, e.Framework, e.Suggestion)
}

// frameworkCommandMap defines which commands are allowed for each framework
// and what suggestion to provide when a command is blocked
var frameworkCommandMap = map[string]struct {
	allowed     []string
	suggestions map[string]string
}{
	"IDPF-Agile": {
		allowed: []string{"branch"},
		suggestions: map[string]string{
			"release": "branch start --name ...",
			"patch":   "branch start --name ...",
		},
	},
	"IDPF-Structured": {
		allowed: []string{"release"},
		suggestions: map[string]string{
			"patch": "release start --version X.Y.Z",
		},
	},
	"IDPF-LTS": {
		allowed: []string{"patch"},
		suggestions: map[string]string{
			"release": "patch start --version X.Y.Z",
		},
	},
}

// ValidateCommand checks if a command is applicable for the given framework.
// Returns nil if the command is allowed, or a CommandRestrictionError with
// a helpful suggestion if the command is not applicable.
func ValidateCommand(framework, command string) error {
	// If no framework detected, all commands are allowed
	if framework == "" {
		return nil
	}

	config, ok := frameworkCommandMap[framework]
	if !ok {
		// Unknown framework - allow all commands
		return nil
	}

	// Check if command is in the allowed list
	for _, allowed := range config.allowed {
		if allowed == command {
			return nil
		}
	}

	// Command is not allowed - return error with suggestion
	suggestion, ok := config.suggestions[command]
	if !ok {
		suggestion = "an appropriate command for " + framework
	}

	return &CommandRestrictionError{
		Framework:  framework,
		Command:    command,
		Suggestion: suggestion,
	}
}
