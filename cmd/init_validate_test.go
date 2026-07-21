package cmd

import (
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/defaults"
)

// Tests for the shared IDPF required-field validation extracted from
// runInitExistingProject and runInitPostCreate (#874). The two init paths
// returned byte-identical error strings; they differed only in that init.go
// printed a longer stderr hint while init_atomic.go routed through rollback.
// The hint is therefore returned separately from the error.

func requiredStatusField() []defaults.FieldDef {
	return []defaults.FieldDef{{
		Name:    "Status",
		Type:    "SINGLE_SELECT",
		Options: []string{"Backlog", "Done"},
	}}
}

func projectStatusField(options ...string) []api.ProjectField {
	opts := make([]api.FieldOption, 0, len(options))
	for i, name := range options {
		opts = append(opts, api.FieldOption{ID: string(rune('a' + i)), Name: name})
	}
	return []api.ProjectField{{ID: "f1", Name: "Status", DataType: "SINGLE_SELECT", Options: opts}}
}

func TestValidateRequiredFields_AllPresent(t *testing.T) {
	hint, err := validateRequiredFields(projectStatusField("Backlog", "Done"), requiredStatusField())
	if err != nil {
		t.Fatalf("Expected validation to pass; got %v", err)
	}
	if hint != "" {
		t.Errorf("Expected no hint on success; got %q", hint)
	}
}

func TestValidateRequiredFields_MissingFieldCarriesHint(t *testing.T) {
	hint, err := validateRequiredFields(nil, requiredStatusField())
	if err == nil {
		t.Fatal("Expected an error when the required field is absent")
	}
	// Byte-identical to what both init paths returned before extraction.
	if err.Error() != `required field "Status" not found in project` {
		t.Errorf("Unexpected error string: %q", err.Error())
	}
	if !strings.Contains(hint, "create it in the project settings before connecting") {
		t.Errorf("Expected the operator hint for a missing field; got %q", hint)
	}
}

func TestValidateRequiredFields_TypeMismatchHasNoHint(t *testing.T) {
	fields := []api.ProjectField{{ID: "f1", Name: "Status", DataType: "TEXT"}}

	hint, err := validateRequiredFields(fields, requiredStatusField())
	if err == nil {
		t.Fatal("Expected an error when the field type differs")
	}
	if err.Error() != `field "Status" has type TEXT, expected SINGLE_SELECT` {
		t.Errorf("Unexpected error string: %q", err.Error())
	}
	if hint != "" {
		t.Errorf("Expected no hint for a type mismatch; got %q", hint)
	}
}

func TestValidateRequiredFields_MissingOptionHasNoHint(t *testing.T) {
	hint, err := validateRequiredFields(projectStatusField("Backlog"), requiredStatusField())
	if err == nil {
		t.Fatal("Expected an error when a required option is absent")
	}
	if err.Error() != `field "Status" missing required option "Done"` {
		t.Errorf("Unexpected error string: %q", err.Error())
	}
	if hint != "" {
		t.Errorf("Expected no hint for a missing option; got %q", hint)
	}
}

// Options are only checked for SINGLE_SELECT definitions that declare them.
func TestValidateRequiredFields_NonSingleSelectSkipsOptionCheck(t *testing.T) {
	required := []defaults.FieldDef{{Name: "Branch", Type: "TEXT", Options: []string{"ignored"}}}
	fields := []api.ProjectField{{ID: "f1", Name: "Branch", DataType: "TEXT"}}

	if _, err := validateRequiredFields(fields, required); err != nil {
		t.Errorf("Expected TEXT field to pass without an option check; got %v", err)
	}
}

func TestValidateRequiredFields_NoRequirementsPasses(t *testing.T) {
	if _, err := validateRequiredFields(nil, nil); err != nil {
		t.Errorf("Expected no requirements to pass; got %v", err)
	}
}
