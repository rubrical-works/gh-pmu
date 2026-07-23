package api

// Shared extraction of ProjectV2 field values (#874).
//
// Six call sites in queries.go previously duplicated the __typename switch that
// turns ProjectV2ItemFieldValue union members into []FieldValue — three against
// the typed shurcooL-graphql selection and three against raw JSON. Their guards
// had drifted apart: some required a non-empty value, one required a non-empty
// field name, one required both, and one required neither consistently. That
// drift is the same class of divergence behind #856.
//
// These helpers collapse the switch to one place and normalize the guard: a
// node contributes a FieldValue only when it is a supported union member and
// both its field name and its value are non-empty. Dropping empty field names
// matters because the ProjectV2 field resolver can return rows whose name is
// null (see ErrFieldResolverUnavailable) — those would otherwise surface as
// FieldValue entries with an empty Field, silently unmatched by every caller
// that looks a field up by name.

const (
	typeSingleSelectValue = "ProjectV2ItemFieldSingleSelectValue"
	typeTextValue         = "ProjectV2ItemFieldTextValue"
)

// typedFieldValueNode mirrors the ProjectV2ItemFieldValue union selection used
// by the typed project-item queries. It is a named type so the three typed call
// sites share one definition rather than three identical anonymous structs.
type typedFieldValueNode struct {
	TypeName                            string `graphql:"__typename"`
	ProjectV2ItemFieldSingleSelectValue struct {
		Name  string
		Field struct {
			ProjectV2SingleSelectField struct {
				Name string
			} `graphql:"... on ProjectV2SingleSelectField"`
		}
	} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
	ProjectV2ItemFieldTextValue struct {
		Text  string
		Field struct {
			ProjectV2Field struct {
				Name string
			} `graphql:"... on ProjectV2Field"`
		}
	} `graphql:"... on ProjectV2ItemFieldTextValue"`
}

// rawFieldValueNode mirrors the same union as decoded from raw JSON, where the
// aliased selection flattens every member's name/text onto one object.
type rawFieldValueNode struct {
	TypeName string `json:"__typename"`
	Name     string `json:"name"`
	Text     string `json:"text"`
	Field    struct {
		Name string `json:"name"`
	} `json:"field"`
}

// newFieldValue normalizes one union member into a FieldValue. ok is false when
// typeName is not a supported member, or when either fieldName or value is
// empty.
func newFieldValue(typeName, fieldName, value string) (FieldValue, bool) {
	switch typeName {
	case typeSingleSelectValue, typeTextValue:
	default:
		return FieldValue{}, false
	}
	if fieldName == "" || value == "" {
		return FieldValue{}, false
	}
	return FieldValue{Field: fieldName, Value: value}, true
}

// appendTypedFieldValues appends every extractable field value in nodes to dst.
func appendTypedFieldValues(dst []FieldValue, nodes []typedFieldValueNode) []FieldValue {
	for _, n := range nodes {
		var fieldName, value string
		switch n.TypeName {
		case typeSingleSelectValue:
			fieldName = n.ProjectV2ItemFieldSingleSelectValue.Field.ProjectV2SingleSelectField.Name
			value = n.ProjectV2ItemFieldSingleSelectValue.Name
		case typeTextValue:
			fieldName = n.ProjectV2ItemFieldTextValue.Field.ProjectV2Field.Name
			value = n.ProjectV2ItemFieldTextValue.Text
		}
		if fv, ok := newFieldValue(n.TypeName, fieldName, value); ok {
			dst = append(dst, fv)
		}
	}
	return dst
}

// appendRawFieldValues appends every extractable field value in nodes to dst.
func appendRawFieldValues(dst []FieldValue, nodes []rawFieldValueNode) []FieldValue {
	for _, n := range nodes {
		var value string
		switch n.TypeName {
		case typeSingleSelectValue:
			value = n.Name
		case typeTextValue:
			value = n.Text
		}
		if fv, ok := newFieldValue(n.TypeName, n.Field.Name, value); ok {
			dst = append(dst, fv)
		}
	}
	return dst
}
