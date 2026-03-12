package defaults

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	defs, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if defs == nil {
		t.Fatal("Load() returned nil")
	}
}

func TestLoad_HasLabels(t *testing.T) {
	defs, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expectedLabels := []string{"branch", "epic", "story", "proposal", "prd", "bug", "enhancement", "qa-required", "test-plan", "security-required", "legal-required", "docs-required", "emergency", "approval-required", "blocked", "scope-creep", "tech-debt", "active", "reviewed", "pending", "security-finding", "assigned"}

	if len(defs.Labels) != len(expectedLabels) {
		t.Errorf("expected %d labels, got %d", len(expectedLabels), len(defs.Labels))
	}

	labelNames := make(map[string]bool)
	for _, label := range defs.Labels {
		labelNames[label.Name] = true
	}

	for _, expected := range expectedLabels {
		if !labelNames[expected] {
			t.Errorf("expected label %q not found", expected)
		}
	}
}

func TestLoad_LabelProperties(t *testing.T) {
	defs, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, label := range defs.Labels {
		if label.Name == "" {
			t.Error("label has empty name")
		}
		if label.Color == "" {
			t.Errorf("label %q has empty color", label.Name)
		}
		if label.Description == "" {
			t.Errorf("label %q has empty description", label.Name)
		}
	}
}

func TestLoad_HasRequiredFields(t *testing.T) {
	defs, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(defs.Fields.Required) == 0 {
		t.Error("expected at least one required field")
	}

	// Status should be required
	var statusField *FieldDef
	for i := range defs.Fields.Required {
		if defs.Fields.Required[i].Name == "Status" {
			statusField = &defs.Fields.Required[i]
			break
		}
	}

	if statusField == nil {
		t.Fatal("Status field not found in required fields")
	}

	if statusField.Type != "SINGLE_SELECT" {
		t.Errorf("Status field type = %q, want SINGLE_SELECT", statusField.Type)
	}

	expectedOptions := []string{"Backlog", "In progress", "In review", "Done"}
	if len(statusField.Options) != len(expectedOptions) {
		t.Errorf("Status field has %d options, want %d", len(statusField.Options), len(expectedOptions))
	}
}

func TestLoad_HasCreateIfMissingFields(t *testing.T) {
	defs, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expectedFields := map[string]string{
		"Priority": "SINGLE_SELECT",
		"Branch":   "TEXT",
	}

	if len(defs.Fields.CreateIfMissing) != len(expectedFields) {
		t.Errorf("expected %d create_if_missing fields, got %d", len(expectedFields), len(defs.Fields.CreateIfMissing))
	}

	for _, field := range defs.Fields.CreateIfMissing {
		expectedType, ok := expectedFields[field.Name]
		if !ok {
			t.Errorf("unexpected field %q in create_if_missing", field.Name)
			continue
		}
		if field.Type != expectedType {
			t.Errorf("field %q has type %q, want %q", field.Name, field.Type, expectedType)
		}
	}
}

func TestLoad_PriorityFieldHasOptions(t *testing.T) {
	defs, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	var priorityField *FieldDef
	for i := range defs.Fields.CreateIfMissing {
		if defs.Fields.CreateIfMissing[i].Name == "Priority" {
			priorityField = &defs.Fields.CreateIfMissing[i]
			break
		}
	}

	if priorityField == nil {
		t.Fatal("Priority field not found in create_if_missing fields")
	}

	expectedOptions := []string{"P0", "P1", "P2"}
	if len(priorityField.Options) != len(expectedOptions) {
		t.Errorf("Priority field has %d options, want %d", len(priorityField.Options), len(expectedOptions))
	}

	for i, opt := range priorityField.Options {
		if opt != expectedOptions[i] {
			t.Errorf("Priority option[%d] = %q, want %q", i, opt, expectedOptions[i])
		}
	}
}

func TestMustLoad(t *testing.T) {
	// MustLoad should not panic with valid embedded data
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustLoad() panicked: %v", r)
		}
	}()

	defs := MustLoad()
	if defs == nil {
		t.Error("MustLoad() returned nil")
	}
}

func TestGetLabel_Found(t *testing.T) {
	defs := MustLoad()

	// Test each known label
	knownLabels := []string{"branch", "epic", "story", "proposal", "prd", "bug", "enhancement", "qa-required", "test-plan", "security-required", "legal-required", "docs-required", "emergency", "approval-required", "blocked", "scope-creep"}
	for _, name := range knownLabels {
		label := defs.GetLabel(name)
		if label == nil {
			t.Errorf("GetLabel(%q) returned nil, expected label", name)
			continue
		}
		if label.Name != name {
			t.Errorf("GetLabel(%q).Name = %q, want %q", name, label.Name, name)
		}
		if label.Color == "" {
			t.Errorf("GetLabel(%q).Color is empty", name)
		}
		if label.Description == "" {
			t.Errorf("GetLabel(%q).Description is empty", name)
		}
	}
}

func TestGetLabel_NotFound(t *testing.T) {
	defs := MustLoad()

	// Test non-existent labels
	nonExistent := []string{"nonexistent", "foo", "bar", "unknown-label"}
	for _, name := range nonExistent {
		label := defs.GetLabel(name)
		if label != nil {
			t.Errorf("GetLabel(%q) = %v, want nil", name, label)
		}
	}
}

func TestGetLabelNames(t *testing.T) {
	defs := MustLoad()

	names := defs.GetLabelNames()

	// Should have same count as Labels
	if len(names) != len(defs.Labels) {
		t.Errorf("GetLabelNames() returned %d names, want %d", len(names), len(defs.Labels))
	}

	// All standard labels should be in the list
	expectedLabels := []string{"branch", "epic", "story", "proposal", "prd", "bug", "enhancement", "qa-required", "test-plan", "security-required", "legal-required", "docs-required", "emergency", "approval-required", "blocked", "scope-creep", "tech-debt", "active", "reviewed", "pending", "security-finding", "assigned"}
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	for _, expected := range expectedLabels {
		if !nameSet[expected] {
			t.Errorf("GetLabelNames() missing expected label %q", expected)
		}
	}
}

func TestTerms_ReturnsNonEmptyText(t *testing.T) {
	// ACT
	text := Terms()

	// ASSERT: Text is not empty and contains key sections
	if text == "" {
		t.Fatal("Terms() returned empty string")
	}

	if len(text) < 100 {
		t.Errorf("Terms() text seems too short: %d chars", len(text))
	}
}

func TestTerms_ContainsRequiredSections(t *testing.T) {
	text := Terms()

	requiredPhrases := []string{
		"Terms and Conditions",
		"What this tool does",
		"Your responsibility",
		"No warranty",
		"Liability",
		"Shared acceptance",
		"as-is",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(text, phrase) {
			t.Errorf("Terms() missing required phrase: %q", phrase)
		}
	}
}

func TestTerms_ContainsPraxisName(t *testing.T) {
	text := Terms()

	if !strings.Contains(text, "Praxis Management Utility") {
		t.Error("Terms() should reference 'Praxis Management Utility'")
	}
}

func TestTerms_ContainsCopyright(t *testing.T) {
	text := Terms()

	if !strings.Contains(text, "Rubrical Works") {
		t.Error("Terms() should contain Rubrical Works copyright")
	}
}

func TestIsStandardLabel(t *testing.T) {
	defs := MustLoad()

	// Standard labels should return true
	standardLabels := []string{"branch", "bug", "enhancement", "security-required", "legal-required", "docs-required"}
	for _, name := range standardLabels {
		if !defs.IsStandardLabel(name) {
			t.Errorf("IsStandardLabel(%q) = false, want true", name)
		}
	}

	// Non-standard labels should return false
	nonStandardLabels := []string{"foo", "bar", "custom-label", "my-label"}
	for _, name := range nonStandardLabels {
		if defs.IsStandardLabel(name) {
			t.Errorf("IsStandardLabel(%q) = true, want false", name)
		}
	}
}
