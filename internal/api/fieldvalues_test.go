package api

import "testing"

// Tests for the shared ProjectV2 field-value extraction helpers (#874).
//
// Before extraction, six call sites in queries.go duplicated the __typename
// switch with divergent guards: three typed sites guarded only on a non-empty
// value, one raw site guarded only on a non-empty field name, one guarded on
// both, and one guarded only on the value. The helpers normalize that to a
// single rule — both the field name and the value must be non-empty.

func TestNewFieldValue_SingleSelect(t *testing.T) {
	fv, ok := newFieldValue("ProjectV2ItemFieldSingleSelectValue", "Status", "In progress")
	if !ok {
		t.Fatalf("Expected single-select node to be accepted")
	}
	if fv.Field != "Status" || fv.Value != "In progress" {
		t.Errorf("Expected Status/In progress; got %q/%q", fv.Field, fv.Value)
	}
}

func TestNewFieldValue_Text(t *testing.T) {
	fv, ok := newFieldValue("ProjectV2ItemFieldTextValue", "Branch", "pmu/next-version")
	if !ok {
		t.Fatalf("Expected text node to be accepted")
	}
	if fv.Field != "Branch" || fv.Value != "pmu/next-version" {
		t.Errorf("Expected Branch/pmu/next-version; got %q/%q", fv.Field, fv.Value)
	}
}

func TestNewFieldValue_UnsupportedTypeRejected(t *testing.T) {
	for _, typeName := range []string{
		"ProjectV2ItemFieldNumberValue",
		"ProjectV2ItemFieldDateValue",
		"ProjectV2ItemFieldIterationValue",
		"",
	} {
		if _, ok := newFieldValue(typeName, "Estimate", "5"); ok {
			t.Errorf("Expected %q to be rejected as unsupported", typeName)
		}
	}
}

// Pins the normalized empty-field-name guard. Unifying sites that previously
// disagreed about this check is a semantic decision no prior test constrained,
// so it is asserted directly (#874 AC4).
func TestNewFieldValue_EmptyFieldNameRejected(t *testing.T) {
	if _, ok := newFieldValue("ProjectV2ItemFieldSingleSelectValue", "", "In progress"); ok {
		t.Errorf("Expected empty field name to be rejected for single-select")
	}
	if _, ok := newFieldValue("ProjectV2ItemFieldTextValue", "", "pmu/next-version"); ok {
		t.Errorf("Expected empty field name to be rejected for text")
	}
}

func TestNewFieldValue_EmptyValueRejected(t *testing.T) {
	if _, ok := newFieldValue("ProjectV2ItemFieldSingleSelectValue", "Status", ""); ok {
		t.Errorf("Expected empty value to be rejected for single-select")
	}
	if _, ok := newFieldValue("ProjectV2ItemFieldTextValue", "Branch", ""); ok {
		t.Errorf("Expected empty value to be rejected for text")
	}
}

func TestAppendTypedFieldValues(t *testing.T) {
	nodes := make([]typedFieldValueNode, 4)

	nodes[0].TypeName = "ProjectV2ItemFieldSingleSelectValue"
	nodes[0].ProjectV2ItemFieldSingleSelectValue.Name = "In progress"
	nodes[0].ProjectV2ItemFieldSingleSelectValue.Field.ProjectV2SingleSelectField.Name = "Status"

	nodes[1].TypeName = "ProjectV2ItemFieldTextValue"
	nodes[1].ProjectV2ItemFieldTextValue.Text = "pmu/next-version"
	nodes[1].ProjectV2ItemFieldTextValue.Field.ProjectV2Field.Name = "Branch"

	// Unsupported type — dropped.
	nodes[2].TypeName = "ProjectV2ItemFieldNumberValue"

	// Non-empty value but empty field name — dropped by the normalized guard.
	nodes[3].TypeName = "ProjectV2ItemFieldSingleSelectValue"
	nodes[3].ProjectV2ItemFieldSingleSelectValue.Name = "Orphaned"

	existing := []FieldValue{{Field: "Priority", Value: "P2"}}
	got := appendTypedFieldValues(existing, nodes)

	want := []FieldValue{
		{Field: "Priority", Value: "P2"},
		{Field: "Status", Value: "In progress"},
		{Field: "Branch", Value: "pmu/next-version"},
	}
	assertFieldValues(t, got, want)
}

func TestAppendRawFieldValues(t *testing.T) {
	nodes := []rawFieldValueNode{
		{TypeName: "ProjectV2ItemFieldSingleSelectValue", Name: "Done"},
		{TypeName: "ProjectV2ItemFieldTextValue", Text: "main"},
		{TypeName: "ProjectV2ItemFieldNumberValue", Name: "5"},
		// Field name present but value empty — dropped by the normalized guard.
		{TypeName: "ProjectV2ItemFieldTextValue"},
	}
	nodes[0].Field.Name = "Status"
	nodes[1].Field.Name = "Branch"
	nodes[2].Field.Name = "Estimate"
	nodes[3].Field.Name = "Notes"

	got := appendRawFieldValues(nil, nodes)

	want := []FieldValue{
		{Field: "Status", Value: "Done"},
		{Field: "Branch", Value: "main"},
	}
	assertFieldValues(t, got, want)
}

func TestAppendFieldValues_NilInputLeavesDestinationUntouched(t *testing.T) {
	if got := appendTypedFieldValues(nil, nil); got != nil {
		t.Errorf("Expected nil destination to stay nil; got %v", got)
	}
	if got := appendRawFieldValues(nil, nil); got != nil {
		t.Errorf("Expected nil destination to stay nil; got %v", got)
	}
}

func assertFieldValues(t *testing.T, got, want []FieldValue) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Expected %d field values; got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Index %d: expected %+v; got %+v", i, want[i], got[i])
		}
	}
}
