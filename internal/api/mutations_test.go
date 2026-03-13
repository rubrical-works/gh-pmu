package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// ============================================================================
// Mock GraphQL Client for Testing
// ============================================================================

// mockGraphQLClient implements GraphQLClient interface for testing
type mockGraphQLClient struct {
	queryFunc  func(name string, query interface{}, variables map[string]interface{}) error
	mutateFunc func(name string, mutation interface{}, variables map[string]interface{}) error
}

func (m *mockGraphQLClient) Query(name string, query interface{}, variables map[string]interface{}) error {
	if m.queryFunc != nil {
		return m.queryFunc(name, query, variables)
	}
	return nil
}

func (m *mockGraphQLClient) Mutate(name string, mutation interface{}, variables map[string]interface{}) error {
	if m.mutateFunc != nil {
		return m.mutateFunc(name, mutation, variables)
	}
	return nil
}

// createMockWithField creates a mock that returns a project with a specific field type
func createMockWithField(fieldName, fieldType string, options []FieldOption) *mockGraphQLClient {
	return &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetProjectFields" {
				// Use reflection to populate the query response
				v := reflect.ValueOf(query).Elem()
				node := v.FieldByName("Node")
				projectV2 := node.FieldByName("ProjectV2")
				fields := projectV2.FieldByName("Fields")
				nodes := fields.FieldByName("Nodes")

				// Create a new slice with one element
				nodeType := nodes.Type().Elem()
				newNodes := reflect.MakeSlice(nodes.Type(), 1, 1)
				newNode := reflect.New(nodeType).Elem()

				// Set the typename
				if fieldType == "SINGLE_SELECT" {
					newNode.FieldByName("TypeName").SetString("ProjectV2SingleSelectField")
					singleSelect := newNode.FieldByName("ProjectV2SingleSelectField")
					singleSelect.FieldByName("ID").SetString("field-123")
					singleSelect.FieldByName("Name").SetString(fieldName)
					singleSelect.FieldByName("DataType").SetString(fieldType)

					// Set options
					if len(options) > 0 {
						optionsField := singleSelect.FieldByName("Options")
						optType := optionsField.Type().Elem()
						optSlice := reflect.MakeSlice(optionsField.Type(), len(options), len(options))
						for i, opt := range options {
							optVal := reflect.New(optType).Elem()
							optVal.FieldByName("ID").SetString(opt.ID)
							optVal.FieldByName("Name").SetString(opt.Name)
							optSlice.Index(i).Set(optVal)
						}
						optionsField.Set(optSlice)
					}
				} else {
					newNode.FieldByName("TypeName").SetString("ProjectV2Field")
					field := newNode.FieldByName("ProjectV2Field")
					field.FieldByName("ID").SetString("field-123")
					field.FieldByName("Name").SetString(fieldName)
					field.FieldByName("DataType").SetString(fieldType)
				}

				newNodes.Index(0).Set(newNode)
				nodes.Set(newNodes)
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return nil
		},
	}
}

func TestAddLabelToIssue_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				// Return a label ID
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				if repo.IsValid() {
					label := repo.FieldByName("Label")
					if label.IsValid() {
						idField := label.FieldByName("ID")
						if idField.IsValid() && idField.CanSet() {
							idField.SetString("label-123")
						}
					}
				}
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name == "AddLabelsToLabelable" {
				// Mutation succeeds
				return nil
			}
			return errors.New("unexpected mutation")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.AddLabelToIssue("owner", "repo", "issue-id", "bug")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestAddLabelToIssue_NonStandardLabel(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				// Return empty label ID - label not found
				return nil
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.AddLabelToIssue("owner", "repo", "issue-id", "nonexistent")

	if err == nil {
		t.Fatal("Expected error for non-standard label")
	}
	if !strings.Contains(err.Error(), "is not a standard label") {
		t.Errorf("Expected 'is not a standard label' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Available standard labels") {
		t.Errorf("Expected error to list available labels, got: %v", err)
	}
}

func TestAddLabelToIssue_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				// Return a label ID
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				if repo.IsValid() {
					label := repo.FieldByName("Label")
					if label.IsValid() {
						idField := label.FieldByName("ID")
						if idField.IsValid() && idField.CanSet() {
							idField.SetString("label-123")
						}
					}
				}
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name == "AddLabelsToLabelable" {
				return errors.New("permission denied")
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.AddLabelToIssue("owner", "repo", "issue-id", "bug")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to add label") {
		t.Errorf("Expected 'failed to add label' error, got: %v", err)
	}
}

func TestRemoveLabelFromIssue_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				// Return a label ID
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				if repo.IsValid() {
					label := repo.FieldByName("Label")
					if label.IsValid() {
						idField := label.FieldByName("ID")
						if idField.IsValid() && idField.CanSet() {
							idField.SetString("label-123")
						}
					}
				}
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name == "RemoveLabelsFromLabelable" {
				// Mutation succeeds
				return nil
			}
			return errors.New("unexpected mutation")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.RemoveLabelFromIssue("owner", "repo", "issue-id", "bug")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestRemoveLabelFromIssue_LabelNotFound(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				// Return empty label ID - label not found
				return nil
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.RemoveLabelFromIssue("owner", "repo", "issue-id", "nonexistent")

	if err == nil {
		t.Fatal("Expected error when label not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestRemoveLabelFromIssue_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				// Return a label ID
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				if repo.IsValid() {
					label := repo.FieldByName("Label")
					if label.IsValid() {
						idField := label.FieldByName("ID")
						if idField.IsValid() && idField.CanSet() {
							idField.SetString("label-123")
						}
					}
				}
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name == "RemoveLabelsFromLabelable" {
				return errors.New("permission denied")
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.RemoveLabelFromIssue("owner", "repo", "issue-id", "bug")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to remove label") {
		t.Errorf("Expected 'failed to remove label' error, got: %v", err)
	}
}

// ============================================================================
// SetProjectItemField Tests with Mocking
// ============================================================================

func TestSetProjectItemField_FieldNotFound(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			// Return empty fields - no matching field will be found
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "NonExistentField", "value")

	if err == nil {
		t.Fatal("Expected error when field not found")
	}
	if !strings.Contains(err.Error(), "field \"NonExistentField\" not found") {
		t.Errorf("Expected 'field not found' error, got: %v", err)
	}
}

func TestSetProjectItemField_GetFieldsError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("network error")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "Status", "Done")

	if err == nil {
		t.Fatal("Expected error when GetProjectFields fails")
	}
	if !strings.Contains(err.Error(), "failed to get project fields") {
		t.Errorf("Expected 'failed to get project fields' error, got: %v", err)
	}
}

func TestSetProjectItemField_SingleSelectField_Success(t *testing.T) {
	options := []FieldOption{
		{ID: "opt-1", Name: "Todo"},
		{ID: "opt-2", Name: "In Progress"},
		{ID: "opt-3", Name: "Done"},
	}
	mock := createMockWithField("Status", "SINGLE_SELECT", options)

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "Status", "Done")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestSetProjectItemField_SingleSelectField_OptionNotFound(t *testing.T) {
	options := []FieldOption{
		{ID: "opt-1", Name: "Todo"},
		{ID: "opt-2", Name: "Done"},
	}
	mock := createMockWithField("Status", "SINGLE_SELECT", options)

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "Status", "Invalid Option")

	if err == nil {
		t.Fatal("Expected error when option not found")
	}
	if !strings.Contains(err.Error(), "option \"Invalid Option\" not found") {
		t.Errorf("Expected 'option not found' error, got: %v", err)
	}
}

func TestSetProjectItemField_TextField_Success(t *testing.T) {
	mock := createMockWithField("Notes", "TEXT", nil)

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "Notes", "Some notes")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestSetProjectItemField_NumberField_Success(t *testing.T) {
	mock := createMockWithField("Points", "NUMBER", nil)

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "Points", "5")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestSetProjectItemField_NumberField_InvalidValue(t *testing.T) {
	mock := createMockWithField("Points", "NUMBER", nil)

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "Points", "not-a-number")

	if err == nil {
		t.Fatal("Expected error when number value is invalid")
	}
	if !strings.Contains(err.Error(), "invalid number value") {
		t.Errorf("Expected 'invalid number value' error, got: %v", err)
	}
}

func TestSetProjectItemField_NumberField_FloatValue(t *testing.T) {
	mock := createMockWithField("Points", "NUMBER", nil)

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "Points", "3.14")

	if err != nil {
		t.Fatalf("Unexpected error for float value: %v", err)
	}
}

func TestSetProjectItemField_UnsupportedFieldType(t *testing.T) {
	mock := createMockWithField("CustomField", "ITERATION", nil)

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "CustomField", "some-value")

	if err == nil {
		t.Fatal("Expected error for unsupported field type")
	}
	if !strings.Contains(err.Error(), "unsupported field type") {
		t.Errorf("Expected 'unsupported field type' error, got: %v", err)
	}
}

func TestSetProjectItemField_DateField_Success(t *testing.T) {
	mock := createMockWithField("DueDate", "DATE", nil)

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "DueDate", "2024-01-15")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestSetProjectItemField_DateField_InvalidFormat(t *testing.T) {
	mock := createMockWithField("DueDate", "DATE", nil)

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "DueDate", "15/01/2024")

	if err == nil {
		t.Fatal("Expected error when date format is invalid")
	}
	if !strings.Contains(err.Error(), "invalid date format") {
		t.Errorf("Expected 'invalid date format' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "expected YYYY-MM-DD") {
		t.Errorf("Expected format hint in error, got: %v", err)
	}
}

func TestSetProjectItemField_DateField_ClearWithEmpty(t *testing.T) {
	mock := createMockWithField("DueDate", "DATE", nil)
	clearCalled := false
	mock.mutateFunc = func(name string, mutation interface{}, variables map[string]interface{}) error {
		if name == "ClearProjectV2ItemFieldValue" {
			clearCalled = true
		}
		return nil
	}

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "DueDate", "")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !clearCalled {
		t.Error("Expected ClearProjectV2ItemFieldValue mutation to be called")
	}
}

func TestSetProjectItemField_MutationError(t *testing.T) {
	mock := createMockWithField("Notes", "TEXT", nil)
	mock.mutateFunc = func(name string, mutation interface{}, variables map[string]interface{}) error {
		return errors.New("mutation failed")
	}

	client := NewClientWithGraphQL(mock)
	err := client.SetProjectItemField("proj-id", "item-id", "Notes", "Some notes")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to set") {
		t.Errorf("Expected 'failed to set' error, got: %v", err)
	}
}

// ============================================================================
// AddIssueToProject Tests with Mocking
// ============================================================================

func TestAddIssueToProject_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "AddProjectV2ItemById" {
				t.Errorf("Expected mutation name 'AddProjectV2ItemById', got '%s'", name)
			}
			// Populate the response via reflection
			v := reflect.ValueOf(mutation).Elem()
			addItem := v.FieldByName("AddProjectV2ItemById")
			item := addItem.FieldByName("Item")
			item.FieldByName("ID").SetString("PVTI_test-item-123")
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	itemID, err := client.AddIssueToProject("proj-id", "issue-id")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if itemID != "PVTI_test-item-123" {
		t.Errorf("Expected item ID 'PVTI_test-item-123', got '%s'", itemID)
	}
}

func TestAddIssueToProject_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("mutation failed")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.AddIssueToProject("proj-id", "issue-id")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to add issue to project") {
		t.Errorf("Expected 'failed to add issue to project' error, got: %v", err)
	}
}

// ============================================================================
// AddSubIssue Tests with Mocking
// ============================================================================

func TestAddSubIssue_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "AddSubIssue" {
				t.Errorf("Expected mutation name 'AddSubIssue', got '%s'", name)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.AddSubIssue("parent-id", "child-id")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestAddSubIssue_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("mutation failed")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.AddSubIssue("parent-id", "child-id")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to add sub-issue") {
		t.Errorf("Expected 'failed to add sub-issue' error, got: %v", err)
	}
}

// ============================================================================
// RemoveSubIssue Tests with Mocking
// ============================================================================

func TestRemoveSubIssue_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "RemoveSubIssue" {
				t.Errorf("Expected mutation name 'RemoveSubIssue', got '%s'", name)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.RemoveSubIssue("parent-id", "child-id")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestRemoveSubIssue_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("mutation failed")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.RemoveSubIssue("parent-id", "child-id")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to remove sub-issue") {
		t.Errorf("Expected 'failed to remove sub-issue' error, got: %v", err)
	}
}

// ============================================================================
// CreateIssue Tests with Mocking
// ============================================================================

func TestCreateIssue_GetRepositoryIDError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("repo not found")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.CreateIssue("owner", "repo", "title", "body", nil)

	if err == nil {
		t.Fatal("Expected error when getRepositoryID fails")
	}
	if !strings.Contains(err.Error(), "failed to get repository ID") {
		t.Errorf("Expected 'failed to get repository ID' error, got: %v", err)
	}
}

func TestCreateIssue_MutationError(t *testing.T) {
	callCount := 0
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			// First call is getRepositoryID - succeed
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			callCount++
			return errors.New("create issue failed")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.CreateIssue("owner", "repo", "title", "body", nil)

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to create issue") {
		t.Errorf("Expected 'failed to create issue' error, got: %v", err)
	}
}

func TestCreateIssue_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				repo.FieldByName("ID").SetString("repo-id-456")
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "CreateIssue" {
				t.Errorf("Expected mutation name 'CreateIssue', got '%s'", name)
			}
			// Populate mutation response via reflection
			v := reflect.ValueOf(mutation).Elem()
			ci := v.FieldByName("CreateIssue")
			issue := ci.FieldByName("Issue")
			issue.FieldByName("ID").SetString("I_test-issue-789")
			issue.FieldByName("Number").SetInt(99)
			issue.FieldByName("Title").SetString("title")
			issue.FieldByName("Body").SetString("body")
			issue.FieldByName("State").SetString("OPEN")
			issue.FieldByName("URL").SetString("https://github.com/owner/repo/issues/99")
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	issue, err := client.CreateIssue("owner", "repo", "title", "body", nil)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("Expected issue to be returned")
	}
	// Validate fields from mutation response (not just input pass-through)
	if issue.ID != "I_test-issue-789" {
		t.Errorf("Expected ID 'I_test-issue-789', got '%s'", issue.ID)
	}
	if issue.Number != 99 {
		t.Errorf("Expected Number 99, got %d", issue.Number)
	}
	if issue.Title != "title" {
		t.Errorf("Expected Title 'title', got '%s'", issue.Title)
	}
	if issue.URL != "https://github.com/owner/repo/issues/99" {
		t.Errorf("Expected URL 'https://github.com/owner/repo/issues/99', got '%s'", issue.URL)
	}
	if issue.Repository.Owner != "owner" {
		t.Errorf("Expected owner 'owner', got '%s'", issue.Repository.Owner)
	}
	if issue.Repository.Name != "repo" {
		t.Errorf("Expected repo 'repo', got '%s'", issue.Repository.Name)
	}
}

func TestCreateIssue_WithLabels_ErrorsForNonStandardLabels(t *testing.T) {
	// This test verifies that CreateIssue fails for non-standard labels.
	// Note: Label batch queries use exec.Command("gh api graphql") which bypasses the mock,
	// so we only verify the overall behavior, not the query count.
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			// getRepositoryID succeeds, label lookups may not be called via mock
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	// "custom-label" and "unknown" are not in defaults.yml, so they should cause an error
	_, err := client.CreateIssue("owner", "repo", "title", "body", []string{"custom-label", "unknown"})

	if err == nil {
		t.Fatal("Expected error for non-standard labels")
	}
	if !strings.Contains(err.Error(), "is not a standard label") {
		t.Errorf("Expected 'is not a standard label' error, got: %v", err)
	}
}

func TestCreateIssue_WithLabels_AutoCreatesStandardLabel(t *testing.T) {
	// This test verifies that CreateIssue auto-creates standard labels that don't exist.
	queryCallCount := 0
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			queryCallCount++
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				repo.FieldByName("ID").SetString("repo-123")
			}
			if name == "GetLabelID" {
				// First call returns empty (label doesn't exist), second call returns the created label
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				if queryCallCount > 2 {
					// After creation, return the label ID
					label.FieldByName("ID").SetString("label-bug-123")
				}
				// First calls return empty ID (label not found)
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name == "CreateLabel" {
				// Verify the label is being created with correct properties
				input := variables["input"].(CreateLabelInput)
				if string(input.Name) != "bug" {
					t.Errorf("Expected label name 'bug', got '%s'", input.Name)
				}
				// "bug" label should have color "d73a4a" per defaults.yml
				if string(input.Color) != "d73a4a" {
					t.Errorf("Expected bug label color 'd73a4a', got '%s'", input.Color)
				}
			}
			if name == "CreateIssue" {
				// Return a successful issue creation
				v := reflect.ValueOf(mutation).Elem()
				createIssue := v.FieldByName("CreateIssue")
				issue := createIssue.FieldByName("Issue")
				issue.FieldByName("ID").SetString("issue-123")
				issue.FieldByName("Number").SetInt(1)
				issue.FieldByName("Title").SetString("Test Issue")
				issue.FieldByName("URL").SetString("https://github.com/owner/repo/issues/1")
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	// "bug" is a standard label, should be auto-created if missing
	issue, err := client.CreateIssue("owner", "repo", "Test Issue", "body", []string{"bug"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("Expected issue to be created")
	}
}

// ============================================================================
// getLabelID Tests with Mocking
// ============================================================================

func TestGetLabelID_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				// Use reflection to populate the label ID
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				label.FieldByName("ID").SetString("label-123")
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	labelID, err := client.getLabelID("owner", "repo", "bug")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if labelID != "label-123" {
		t.Errorf("Expected label ID 'label-123', got '%s'", labelID)
	}
}

func TestGetLabelID_QueryError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("network error")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.getLabelID("owner", "repo", "bug")

	if err == nil {
		t.Fatal("Expected error when query fails")
	}
	if !strings.Contains(err.Error(), "failed to get label ID") {
		t.Errorf("Expected 'failed to get label ID' error, got: %v", err)
	}
}

func TestGetLabelID_LabelNotFound(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			// Don't populate the label ID - leave it empty
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.getLabelID("owner", "repo", "nonexistent")

	if err == nil {
		t.Fatal("Expected error when label not found")
	}
	if !strings.Contains(err.Error(), "label \"nonexistent\" not found") {
		t.Errorf("Expected 'label not found' error, got: %v", err)
	}
}

// ============================================================================
// CreateIssueInput Optional Fields Tests
// ============================================================================

func TestCreateIssueInput_LabelsIncludedInMutation(t *testing.T) {
	// Verify that when CreateIssue is called with labels, the mutation input
	// includes label IDs (not just struct zero-value checks)
	var capturedInput CreateIssueInput
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				repo.FieldByName("ID").SetString("repo-123")
			}
			if name == "GetLabelID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				label.FieldByName("ID").SetString("label-bug-id")
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name == "CreateIssue" {
				capturedInput = variables["input"].(CreateIssueInput)
				v := reflect.ValueOf(mutation).Elem()
				ci := v.FieldByName("CreateIssue")
				issue := ci.FieldByName("Issue")
				issue.FieldByName("ID").SetString("issue-456")
				issue.FieldByName("Number").SetInt(42)
				issue.FieldByName("Title").SetString("Test Issue")
				issue.FieldByName("URL").SetString("https://github.com/owner/repo/issues/42")
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	issue, err := client.CreateIssue("owner", "repo", "Test Issue", "body", []string{"bug"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("Expected issue to be returned")
	}

	// Verify label IDs were included in the mutation input
	if capturedInput.LabelIDs == nil {
		t.Fatal("Expected LabelIDs to be set in mutation input")
	}
	if len(*capturedInput.LabelIDs) != 1 {
		t.Fatalf("Expected 1 label ID, got %d", len(*capturedInput.LabelIDs))
	}
	labelID := fmt.Sprintf("%v", (*capturedInput.LabelIDs)[0])
	if labelID != "label-bug-id" {
		t.Errorf("Expected label ID 'label-bug-id', got '%s'", labelID)
	}

	// Verify no body is omitted when provided
	if string(capturedInput.Body) != "body" {
		t.Errorf("Expected body 'body', got '%s'", capturedInput.Body)
	}
}

// ============================================================================
// CreateProjectField Tests
// ============================================================================

func TestCreateProjectField_TextFieldSuccess(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "CreateProjectV2Field" {
				t.Errorf("Expected mutation name 'CreateProjectV2Field', got '%s'", name)
			}

			// Verify input
			input, ok := variables["input"].(CreateProjectV2FieldInput)
			if !ok {
				t.Fatal("Expected CreateProjectV2FieldInput in variables")
			}
			if string(input.Name) != "PRD" {
				t.Errorf("Expected field name 'PRD', got '%s'", input.Name)
			}
			if string(input.DataType) != "TEXT" {
				t.Errorf("Expected data type 'TEXT', got '%s'", input.DataType)
			}

			// Populate response using reflection
			v := reflect.ValueOf(mutation).Elem()
			createField := v.FieldByName("CreateProjectV2Field")
			projectV2Field := createField.FieldByName("ProjectV2Field")
			field := projectV2Field.FieldByName("ProjectV2Field")
			field.FieldByName("ID").SetString("PVTF_new123")
			field.FieldByName("Name").SetString("PRD")

			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	field, err := client.CreateProjectField("proj-id", "PRD", "TEXT", nil)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if field == nil {
		t.Fatal("Expected field to be returned")
	}
	if field.Name != "PRD" {
		t.Errorf("Expected field name 'PRD', got '%s'", field.Name)
	}
	if field.DataType != "TEXT" {
		t.Errorf("Expected data type 'TEXT', got '%s'", field.DataType)
	}
}

func TestCreateProjectField_SingleSelectSuccess(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			// Verify input has options
			input, ok := variables["input"].(CreateProjectV2FieldInput)
			if !ok {
				t.Fatal("Expected CreateProjectV2FieldInput in variables")
			}
			if string(input.DataType) != "SINGLE_SELECT" {
				t.Errorf("Expected data type 'SINGLE_SELECT', got '%s'", input.DataType)
			}
			if input.SingleSelectOptions == nil || len(*input.SingleSelectOptions) != 2 {
				t.Error("Expected 2 single select options")
			}

			// Populate response
			v := reflect.ValueOf(mutation).Elem()
			createField := v.FieldByName("CreateProjectV2Field")
			projectV2Field := createField.FieldByName("ProjectV2Field")
			singleSelect := projectV2Field.FieldByName("ProjectV2SingleSelectField")
			singleSelect.FieldByName("ID").SetString("PVTSSF_new123")
			singleSelect.FieldByName("Name").SetString("Environment")

			// Set options
			optionsField := singleSelect.FieldByName("Options")
			optType := optionsField.Type().Elem()
			optSlice := reflect.MakeSlice(optionsField.Type(), 2, 2)

			opt1 := reflect.New(optType).Elem()
			opt1.FieldByName("ID").SetString("opt1")
			opt1.FieldByName("Name").SetString("Dev")
			optSlice.Index(0).Set(opt1)

			opt2 := reflect.New(optType).Elem()
			opt2.FieldByName("ID").SetString("opt2")
			opt2.FieldByName("Name").SetString("Prod")
			optSlice.Index(1).Set(opt2)

			optionsField.Set(optSlice)

			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	field, err := client.CreateProjectField("proj-id", "Environment", "SINGLE_SELECT", []string{"Dev", "Prod"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if field == nil {
		t.Fatal("Expected field to be returned")
	}
	if field.DataType != "SINGLE_SELECT" {
		t.Errorf("Expected data type 'SINGLE_SELECT', got '%s'", field.DataType)
	}
	if len(field.Options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(field.Options))
	}
}

func TestCreateProjectField_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("mutation failed")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.CreateProjectField("proj-id", "TestField", "TEXT", nil)

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to create project field") {
		t.Errorf("Expected 'failed to create project field' error, got: %v", err)
	}
}

// ============================================================================
// Git Command Error Message Tests
// ============================================================================

func TestGitAdd_ErrorMessageIncludesGitOutput(t *testing.T) {
	client, cErr := NewClient()
	if cErr != nil {
		t.Skipf("Skipping - requires auth: %v", cErr)
	}

	// Try to add a non-existent file - this will fail
	err := client.GitAdd("/nonexistent/path/that/does/not/exist.txt")

	if err == nil {
		t.Fatal("Expected error when adding non-existent file")
	}

	// Verify error message includes "git add failed:" prefix
	if !strings.Contains(err.Error(), "git add failed:") {
		t.Errorf("Expected error to contain 'git add failed:', got: %v", err)
	}

	// Verify error message includes git's actual output
	// Different git versions may say "pathspec", "Invalid path", or "No such file"
	errMsg := err.Error()
	hasGitOutput := strings.Contains(errMsg, "pathspec") ||
		strings.Contains(errMsg, "Invalid path") ||
		strings.Contains(errMsg, "No such file") ||
		strings.Contains(errMsg, "fatal:")
	if !hasGitOutput {
		t.Errorf("Expected error to contain git's error output, got: %v", err)
	}
}

func TestGitTag_ErrorMessageIncludesGitOutput(t *testing.T) {
	client, cErr := NewClient()
	if cErr != nil {
		t.Skipf("Skipping - requires auth: %v", cErr)
	}

	// Try to create a tag with invalid characters - this will fail
	// Using a tag name with spaces which is invalid
	err := client.GitTag("invalid tag name", "test message")

	if err == nil {
		t.Fatal("Expected error when creating tag with invalid name")
	}

	// Verify error message includes "git tag failed:" prefix
	if !strings.Contains(err.Error(), "git tag failed:") {
		t.Errorf("Expected error to contain 'git tag failed:', got: %v", err)
	}
}

func TestGitCommit_ErrorMessageIncludesGitOutput(t *testing.T) {
	client, cErr := NewClient()
	if cErr != nil {
		t.Skipf("Skipping - requires auth: %v", cErr)
	}

	// Try to commit with nothing staged - this will fail in most cases
	// Note: This test assumes we're not in a state where a commit would succeed
	err := client.GitCommit("test commit message")

	// If there's nothing to commit, git will return an error
	// We just verify that IF there's an error, it has the right format
	if err != nil {
		if !strings.Contains(err.Error(), "git commit failed:") {
			t.Errorf("Expected error to contain 'git commit failed:', got: %v", err)
		}
	}
}

// ============================================================================
// CloseIssue, ReopenIssue, UpdateIssueBody Tests
// ============================================================================

func TestCloseIssue_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "CloseIssue" {
				t.Errorf("Expected mutation name 'CloseIssue', got '%s'", name)
			}

			// Verify input type is CloseIssueInput (not anonymous struct)
			input, ok := variables["input"].(CloseIssueInput)
			if !ok {
				t.Fatal("Expected CloseIssueInput type in variables, got anonymous struct")
			}
			if input.IssueID.(string) != "issue-123" {
				t.Errorf("Expected IssueID 'issue-123', got '%v'", input.IssueID)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.CloseIssue("issue-123")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestCloseIssue_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("mutation failed")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.CloseIssue("issue-id")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to close issue") {
		t.Errorf("Expected 'failed to close issue' error, got: %v", err)
	}
}

func TestReopenIssue_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "ReopenIssue" {
				t.Errorf("Expected mutation name 'ReopenIssue', got '%s'", name)
			}

			// Verify input type is ReopenIssueInput (not anonymous struct)
			input, ok := variables["input"].(ReopenIssueInput)
			if !ok {
				t.Fatal("Expected ReopenIssueInput type in variables, got anonymous struct")
			}
			if input.IssueID.(string) != "issue-456" {
				t.Errorf("Expected IssueID 'issue-456', got '%v'", input.IssueID)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.ReopenIssue("issue-456")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestReopenIssue_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("mutation failed")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.ReopenIssue("issue-id")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to reopen issue") {
		t.Errorf("Expected 'failed to reopen issue' error, got: %v", err)
	}
}

func TestUpdateIssueBody_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "UpdateIssue" {
				t.Errorf("Expected mutation name 'UpdateIssue', got '%s'", name)
			}

			// Verify input type is UpdateIssueInput (not anonymous struct)
			input, ok := variables["input"].(UpdateIssueInput)
			if !ok {
				t.Fatal("Expected UpdateIssueInput type in variables, got anonymous struct")
			}
			if input.ID.(string) != "issue-789" {
				t.Errorf("Expected ID 'issue-789', got '%v'", input.ID)
			}
			if string(input.Body) != "updated body content" {
				t.Errorf("Expected Body 'updated body content', got '%s'", input.Body)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.UpdateIssueBody("issue-789", "updated body content")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestUpdateIssueBody_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("mutation failed")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.UpdateIssueBody("issue-id", "body")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to update issue body") {
		t.Errorf("Expected 'failed to update issue body' error, got: %v", err)
	}
}

// ============================================================================
// UpdateIssueTitle Tests
// ============================================================================

func TestUpdateIssueTitle_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "UpdateIssue" {
				t.Errorf("Expected mutation name 'UpdateIssue', got '%s'", name)
			}

			// Verify input type is UpdateIssueInput
			input, ok := variables["input"].(UpdateIssueInput)
			if !ok {
				t.Fatal("Expected UpdateIssueInput type in variables")
			}
			if input.ID.(string) != "issue-456" {
				t.Errorf("Expected ID 'issue-456', got '%v'", input.ID)
			}
			if string(input.Title) != "updated title" {
				t.Errorf("Expected Title 'updated title', got '%s'", input.Title)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.UpdateIssueTitle("issue-456", "updated title")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestUpdateIssueTitle_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("mutation failed")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.UpdateIssueTitle("issue-id", "title")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to update issue title") {
		t.Errorf("Expected 'failed to update issue title' error, got: %v", err)
	}
}

// ============================================================================
// SetProjectItemFieldWithFields Tests
// ============================================================================

func TestSetProjectItemFieldWithFields_FieldNotFound(t *testing.T) {
	mock := &mockGraphQLClient{}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{ID: "field-123", Name: "Status", DataType: "SINGLE_SELECT"},
	}

	err := client.SetProjectItemFieldWithFields("proj-id", "item-id", "Priority", "P1", fields)
	if err == nil {
		t.Fatal("Expected error when field not found")
	}
	if !strings.Contains(err.Error(), "field \"Priority\" not found") {
		t.Errorf("Expected 'field not found' error, got: %v", err)
	}
}

func TestSetProjectItemFieldWithFields_SingleSelectField_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return nil
		},
	}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{
			ID:       "field-123",
			Name:     "Status",
			DataType: "SINGLE_SELECT",
			Options: []FieldOption{
				{ID: "opt-1", Name: "Todo"},
				{ID: "opt-2", Name: "Done"},
			},
		},
	}

	err := client.SetProjectItemFieldWithFields("proj-id", "item-id", "Status", "Done", fields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestSetProjectItemFieldWithFields_SingleSelectField_OptionNotFound(t *testing.T) {
	mock := &mockGraphQLClient{}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{
			ID:       "field-123",
			Name:     "Status",
			DataType: "SINGLE_SELECT",
			Options: []FieldOption{
				{ID: "opt-1", Name: "Todo"},
				{ID: "opt-2", Name: "Done"},
			},
		},
	}

	err := client.SetProjectItemFieldWithFields("proj-id", "item-id", "Status", "Invalid", fields)
	if err == nil {
		t.Fatal("Expected error when option not found")
	}
	if !strings.Contains(err.Error(), "option \"Invalid\" not found") {
		t.Errorf("Expected 'option not found' error, got: %v", err)
	}
}

func TestSetProjectItemFieldWithFields_TextField_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return nil
		},
	}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{ID: "field-123", Name: "Notes", DataType: "TEXT"},
	}

	err := client.SetProjectItemFieldWithFields("proj-id", "item-id", "Notes", "Some notes", fields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestSetProjectItemFieldWithFields_NumberField_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return nil
		},
	}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{ID: "field-123", Name: "Points", DataType: "NUMBER"},
	}

	err := client.SetProjectItemFieldWithFields("proj-id", "item-id", "Points", "5", fields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestSetProjectItemFieldWithFields_NumberField_InvalidValue(t *testing.T) {
	mock := &mockGraphQLClient{}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{ID: "field-123", Name: "Points", DataType: "NUMBER"},
	}

	err := client.SetProjectItemFieldWithFields("proj-id", "item-id", "Points", "invalid", fields)
	if err == nil {
		t.Fatal("Expected error when number value is invalid")
	}
	if !strings.Contains(err.Error(), "invalid number value") {
		t.Errorf("Expected 'invalid number value' error, got: %v", err)
	}
}

func TestSetProjectItemFieldWithFields_UnsupportedFieldType(t *testing.T) {
	mock := &mockGraphQLClient{}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{ID: "field-123", Name: "Sprint", DataType: "ITERATION"},
	}

	err := client.SetProjectItemFieldWithFields("proj-id", "item-id", "Sprint", "Sprint 1", fields)
	if err == nil {
		t.Fatal("Expected error for unsupported field type")
	}
	if !strings.Contains(err.Error(), "unsupported field type") {
		t.Errorf("Expected 'unsupported field type' error, got: %v", err)
	}
}

func TestSetProjectItemFieldWithFields_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("mutation failed")
		},
	}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{ID: "field-123", Name: "Notes", DataType: "TEXT"},
	}

	err := client.SetProjectItemFieldWithFields("proj-id", "item-id", "Notes", "Some notes", fields)
	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to set") {
		t.Errorf("Expected 'failed to set' error, got: %v", err)
	}
}

// ============================================================================
// BatchUpdateProjectItemFields Tests
// ============================================================================

func TestBatchUpdateProjectItemFields_EmptyUpdates(t *testing.T) {
	mock := &mockGraphQLClient{}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{ID: "field-123", Name: "Status", DataType: "SINGLE_SELECT"},
	}

	results, err := client.BatchUpdateProjectItemFields("proj-id", []FieldUpdate{}, fields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestBatchUpdateProjectItemFields_FieldNotFound(t *testing.T) {
	mock := &mockGraphQLClient{}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{ID: "field-123", Name: "Status", DataType: "SINGLE_SELECT"},
	}

	updates := []FieldUpdate{
		{ItemID: "item-1", FieldName: "NonExistent", Value: "value"},
	}

	results, err := client.BatchUpdateProjectItemFields("proj-id", updates, fields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("Expected result to be failure")
	}
	if !strings.Contains(results[0].Error, "not found") {
		t.Errorf("Expected 'not found' in error, got: %v", results[0].Error)
	}
}

func TestBatchUpdateProjectItemFields_SingleSelectOptionNotFound(t *testing.T) {
	mock := &mockGraphQLClient{}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{
			ID:       "field-123",
			Name:     "Status",
			DataType: "SINGLE_SELECT",
			Options: []FieldOption{
				{ID: "opt-1", Name: "Todo"},
				{ID: "opt-2", Name: "Done"},
			},
		},
	}

	updates := []FieldUpdate{
		{ItemID: "item-1", FieldName: "Status", Value: "InvalidOption"},
	}

	results, err := client.BatchUpdateProjectItemFields("proj-id", updates, fields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("Expected result to be failure")
	}
	if !strings.Contains(results[0].Error, "not found") {
		t.Errorf("Expected 'not found' in error, got: %v", results[0].Error)
	}
}

func TestBatchUpdateProjectItemFields_NumberFieldInvalidValue(t *testing.T) {
	mock := &mockGraphQLClient{}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{ID: "field-123", Name: "Points", DataType: "NUMBER"},
	}

	updates := []FieldUpdate{
		{ItemID: "item-1", FieldName: "Points", Value: "not-a-number"},
	}

	results, err := client.BatchUpdateProjectItemFields("proj-id", updates, fields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("Expected result to be failure")
	}
	if !strings.Contains(results[0].Error, "invalid number value") {
		t.Errorf("Expected 'invalid number value' in error, got: %v", results[0].Error)
	}
}

func TestBatchUpdateProjectItemFields_UnsupportedFieldType(t *testing.T) {
	mock := &mockGraphQLClient{}
	client := NewClientWithGraphQL(mock)

	fields := []ProjectField{
		{ID: "field-123", Name: "Sprint", DataType: "ITERATION"},
	}

	updates := []FieldUpdate{
		{ItemID: "item-1", FieldName: "Sprint", Value: "Sprint 1"},
	}

	// Unsupported field types pass validation but fail in executeBatchMutation
	// Since executeBatchMutation uses exec.Command, it will fail with error
	results, err := client.BatchUpdateProjectItemFields("proj-id", updates, fields)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// The result should contain an error from the failed batch mutation
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	// Note: The actual error depends on executeBatchMutation behavior
	// which will return "unsupported field type" error
	if results[0].Success {
		t.Error("Expected result to be failure for unsupported field type")
	}
}

// ============================================================================
// buildBatchMutationRequest Tests
// ============================================================================

func TestBuildBatchMutationRequest_EmptyUpdates(t *testing.T) {
	mutation, body, err := buildBatchMutationRequest("proj-id", []FieldUpdate{})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if mutation != "" {
		t.Errorf("Expected empty mutation, got '%s'", mutation)
	}
	if body != "" {
		t.Errorf("Expected empty body, got '%s'", body)
	}
}

func TestBuildBatchMutationRequest_TextField(t *testing.T) {
	updates := []FieldUpdate{{
		ItemID:    "item-1",
		FieldName: "Release",
		Value:     "v1.0.0",
		fieldID:   "field-1",
		dataType:  "TEXT",
	}}

	mutation, body, err := buildBatchMutationRequest("proj-id", updates)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify mutation structure
	if !strings.Contains(mutation, "mutation BatchUpdate") {
		t.Error("Mutation should contain 'mutation BatchUpdate'")
	}
	if !strings.Contains(mutation, "updateProjectV2ItemFieldValue") {
		t.Error("Mutation should contain 'updateProjectV2ItemFieldValue'")
	}
	if !strings.Contains(mutation, "$input0: UpdateProjectV2ItemFieldValueInput!") {
		t.Error("Mutation should contain variable declaration")
	}

	// Verify JSON body
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("Failed to parse body as JSON: %v", err)
	}

	variables := parsed["variables"].(map[string]interface{})
	input0 := variables["input0"].(map[string]interface{})
	value := input0["value"].(map[string]interface{})

	if value["text"] != "v1.0.0" {
		t.Errorf("Expected text value 'v1.0.0', got '%v'", value["text"])
	}
	if input0["projectId"] != "proj-id" {
		t.Errorf("Expected projectId 'proj-id', got '%v'", input0["projectId"])
	}
	if input0["itemId"] != "item-1" {
		t.Errorf("Expected itemId 'item-1', got '%v'", input0["itemId"])
	}
	if input0["fieldId"] != "field-1" {
		t.Errorf("Expected fieldId 'field-1', got '%v'", input0["fieldId"])
	}
}

func TestBuildBatchMutationRequest_SingleSelectField(t *testing.T) {
	updates := []FieldUpdate{{
		ItemID:    "item-1",
		FieldName: "Status",
		Value:     "In Progress",
		fieldID:   "field-1",
		optionID:  "option-123",
		dataType:  "SINGLE_SELECT",
	}}

	_, body, err := buildBatchMutationRequest("proj-id", updates)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("Failed to parse body as JSON: %v", err)
	}

	variables := parsed["variables"].(map[string]interface{})
	input0 := variables["input0"].(map[string]interface{})
	value := input0["value"].(map[string]interface{})

	if value["singleSelectOptionId"] != "option-123" {
		t.Errorf("Expected singleSelectOptionId 'option-123', got '%v'", value["singleSelectOptionId"])
	}
}

func TestBuildBatchMutationRequest_NumberField(t *testing.T) {
	updates := []FieldUpdate{{
		ItemID:    "item-1",
		FieldName: "Points",
		Value:     "42.5",
		fieldID:   "field-1",
		dataType:  "NUMBER",
	}}

	_, body, err := buildBatchMutationRequest("proj-id", updates)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("Failed to parse body as JSON: %v", err)
	}

	variables := parsed["variables"].(map[string]interface{})
	input0 := variables["input0"].(map[string]interface{})
	value := input0["value"].(map[string]interface{})

	if value["number"] != 42.5 {
		t.Errorf("Expected number 42.5, got '%v'", value["number"])
	}
}

func TestBuildBatchMutationRequest_DateField(t *testing.T) {
	updates := []FieldUpdate{{
		ItemID:    "item-1",
		FieldName: "DueDate",
		Value:     "2024-01-15",
		fieldID:   "field-1",
		dataType:  "DATE",
	}}

	_, body, err := buildBatchMutationRequest("proj-id", updates)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("Failed to parse body as JSON: %v", err)
	}

	variables := parsed["variables"].(map[string]interface{})
	input0 := variables["input0"].(map[string]interface{})
	value := input0["value"].(map[string]interface{})

	if value["date"] != "2024-01-15" {
		t.Errorf("Expected date '2024-01-15', got '%v'", value["date"])
	}
}

func TestBuildBatchMutationRequest_MultipleBatch(t *testing.T) {
	updates := []FieldUpdate{
		{
			ItemID:    "item-1",
			FieldName: "Status",
			Value:     "Done",
			fieldID:   "field-status",
			optionID:  "opt-done",
			dataType:  "SINGLE_SELECT",
		},
		{
			ItemID:    "item-1",
			FieldName: "Release",
			Value:     "v2.0.0",
			fieldID:   "field-release",
			dataType:  "TEXT",
		},
		{
			ItemID:    "item-2",
			FieldName: "Priority",
			Value:     "P0",
			fieldID:   "field-priority",
			optionID:  "opt-p0",
			dataType:  "SINGLE_SELECT",
		},
	}

	mutation, body, err := buildBatchMutationRequest("proj-id", updates)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify mutation has all variable declarations
	if !strings.Contains(mutation, "$input0") {
		t.Error("Mutation should contain $input0")
	}
	if !strings.Contains(mutation, "$input1") {
		t.Error("Mutation should contain $input1")
	}
	if !strings.Contains(mutation, "$input2") {
		t.Error("Mutation should contain $input2")
	}

	// Verify mutation has all aliases
	if !strings.Contains(mutation, "u0:") {
		t.Error("Mutation should contain alias u0")
	}
	if !strings.Contains(mutation, "u1:") {
		t.Error("Mutation should contain alias u1")
	}
	if !strings.Contains(mutation, "u2:") {
		t.Error("Mutation should contain alias u2")
	}

	// Verify JSON body has all inputs
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("Failed to parse body as JSON: %v", err)
	}

	variables := parsed["variables"].(map[string]interface{})
	if _, ok := variables["input0"]; !ok {
		t.Error("Variables should contain input0")
	}
	if _, ok := variables["input1"]; !ok {
		t.Error("Variables should contain input1")
	}
	if _, ok := variables["input2"]; !ok {
		t.Error("Variables should contain input2")
	}
}

func TestBuildBatchMutationRequest_SpecialCharacters(t *testing.T) {
	updates := []FieldUpdate{{
		ItemID:    "item-1",
		FieldName: "Release",
		Value:     `Test with "quotes" and\nnewlines and special chars: <>&`,
		fieldID:   "field-1",
		dataType:  "TEXT",
	}}

	_, body, err := buildBatchMutationRequest("proj-id", updates)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the JSON is valid (json.Marshal escapes properly)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("Failed to parse body as JSON - special characters not properly escaped: %v", err)
	}

	// Verify the value is preserved correctly
	variables := parsed["variables"].(map[string]interface{})
	input0 := variables["input0"].(map[string]interface{})
	value := input0["value"].(map[string]interface{})

	expected := `Test with "quotes" and\nnewlines and special chars: <>&`
	if value["text"] != expected {
		t.Errorf("Expected text '%s', got '%v'", expected, value["text"])
	}
}

func TestBuildBatchMutationRequest_UnsupportedFieldType(t *testing.T) {
	updates := []FieldUpdate{{
		ItemID:    "item-1",
		FieldName: "Unknown",
		Value:     "value",
		fieldID:   "field-1",
		dataType:  "UNKNOWN_TYPE",
	}}

	_, _, err := buildBatchMutationRequest("proj-id", updates)

	if err == nil {
		t.Fatal("Expected error for unsupported field type")
	}
	if !strings.Contains(err.Error(), "unsupported field type") {
		t.Errorf("Error should mention unsupported field type, got: %v", err)
	}
}

// ============================================================================
// CreateLabel Tests
// ============================================================================

func TestCreateLabel_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetRepositoryID" {
				// Return a repository ID
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				if repo.IsValid() {
					idField := repo.FieldByName("ID")
					if idField.IsValid() && idField.CanSet() {
						idField.SetString("repo-123")
					}
				}
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "CreateLabel" {
				t.Errorf("Expected mutation name 'CreateLabel', got '%s'", name)
			}

			// Verify input
			input, ok := variables["input"].(CreateLabelInput)
			if !ok {
				t.Fatal("Expected CreateLabelInput in variables")
			}
			if string(input.Name) != "bug" {
				t.Errorf("Expected label name 'bug', got '%s'", input.Name)
			}
			if string(input.Color) != "d73a4a" {
				t.Errorf("Expected color 'd73a4a', got '%s'", input.Color)
			}
			if string(input.Description) != "Something isn't working" {
				t.Errorf("Expected description, got '%s'", input.Description)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.CreateLabel("owner", "repo", "bug", "d73a4a", "Something isn't working")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestCreateLabel_GetRepositoryIDError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("repo not found")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.CreateLabel("owner", "repo", "bug", "d73a4a", "")

	if err == nil {
		t.Fatal("Expected error when getRepositoryID fails")
	}
	if !strings.Contains(err.Error(), "failed to get repository ID") {
		t.Errorf("Expected 'failed to get repository ID' error, got: %v", err)
	}
}

func TestCreateLabel_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				if repo.IsValid() {
					idField := repo.FieldByName("ID")
					if idField.IsValid() && idField.CanSet() {
						idField.SetString("repo-123")
					}
				}
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("label already exists")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.CreateLabel("owner", "repo", "bug", "d73a4a", "")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to create label") {
		t.Errorf("Expected 'failed to create label' error, got: %v", err)
	}
}

// ============================================================================
// LabelExists Tests
// ============================================================================

func TestLabelExists_LabelFound(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				label.FieldByName("ID").SetString("label-123")
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	exists, err := client.LabelExists("owner", "repo", "bug")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !exists {
		t.Error("Expected LabelExists to return true")
	}
}

func TestLabelExists_LabelNotFound(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			// Don't populate the label ID - label not found
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	exists, err := client.LabelExists("owner", "repo", "nonexistent")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if exists {
		t.Error("Expected LabelExists to return false for nonexistent label")
	}
}

func TestLabelExists_QueryError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("network error")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.LabelExists("owner", "repo", "bug")

	if err == nil {
		t.Fatal("Expected error when query fails")
	}
}

// ============================================================================
// EnsureLabelExists Tests
// ============================================================================

func TestEnsureLabelExists_LabelAlreadyExists(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				label.FieldByName("ID").SetString("label-123")
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	labelID, err := client.EnsureLabelExists("owner", "repo", "bug")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if labelID != "label-123" {
		t.Errorf("Expected label ID 'label-123', got '%s'", labelID)
	}
}

func TestEnsureLabelExists_CreatesStandardLabel(t *testing.T) {
	queryCallCount := 0
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			queryCallCount++
			if name == "GetLabelID" {
				if queryCallCount == 1 {
					// First call - label doesn't exist
					return nil
				}
				// Second call - label was created
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				label.FieldByName("ID").SetString("new-label-123")
			}
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				repo.FieldByName("ID").SetString("repo-123")
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name == "CreateLabel" {
				// Verify the color matches the standard "bug" label definition
				input := variables["input"].(CreateLabelInput)
				if string(input.Color) != "d73a4a" {
					t.Errorf("Expected bug label color 'd73a4a', got '%s'", input.Color)
				}
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	// Use "bug" which is a standard label defined in defaults.yml
	labelID, err := client.EnsureLabelExists("owner", "repo", "bug")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if labelID != "new-label-123" {
		t.Errorf("Expected label ID 'new-label-123', got '%s'", labelID)
	}
}

func TestEnsureLabelExists_ErrorsForNonStandardLabel(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				// Label doesn't exist
				return nil
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.EnsureLabelExists("owner", "repo", "custom-label")

	if err == nil {
		t.Fatal("Expected error for non-standard label")
	}
	if !strings.Contains(err.Error(), "is not a standard label") {
		t.Errorf("Expected 'is not a standard label' error, got: %v", err)
	}
}

func TestEnsureLabelExists_CreateLabelFails(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				repo.FieldByName("ID").SetString("repo-123")
			}
			// Don't populate label ID - label not found
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("permission denied")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.EnsureLabelExists("owner", "repo", "bug")

	if err == nil {
		t.Fatal("Expected error when CreateLabel fails")
	}
}

// ============================================================================
// DeleteLabel Tests
// ============================================================================

func TestDeleteLabel_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				label.FieldByName("ID").SetString("label-123")
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "DeleteLabel" {
				t.Errorf("Expected mutation name 'DeleteLabel', got '%s'", name)
			}

			input, ok := variables["input"].(DeleteLabelInput)
			if !ok {
				t.Fatal("Expected DeleteLabelInput in variables")
			}
			if input.ID.(string) != "label-123" {
				t.Errorf("Expected label ID 'label-123', got '%v'", input.ID)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.DeleteLabel("owner", "repo", "bug")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestDeleteLabel_LabelNotFound(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			// Don't populate label ID - label not found
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.DeleteLabel("owner", "repo", "nonexistent")

	if err == nil {
		t.Fatal("Expected error when label not found")
	}
	if !strings.Contains(err.Error(), "failed to get label ID") {
		t.Errorf("Expected 'failed to get label ID' error, got: %v", err)
	}
}

func TestDeleteLabel_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				label.FieldByName("ID").SetString("label-123")
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("permission denied")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.DeleteLabel("owner", "repo", "bug")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to delete label") {
		t.Errorf("Expected 'failed to delete label' error, got: %v", err)
	}
}

// ============================================================================
// UpdateLabel Tests
// ============================================================================

func TestUpdateLabel_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				label.FieldByName("ID").SetString("label-123")
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "UpdateLabel" {
				t.Errorf("Expected mutation name 'UpdateLabel', got '%s'", name)
			}

			input, ok := variables["input"].(UpdateLabelInput)
			if !ok {
				t.Fatal("Expected UpdateLabelInput in variables")
			}
			if input.ID.(string) != "label-123" {
				t.Errorf("Expected label ID 'label-123', got '%v'", input.ID)
			}
			if string(input.Name) != "bug-renamed" {
				t.Errorf("Expected new name 'bug-renamed', got '%s'", input.Name)
			}
			if string(input.Color) != "ff0000" {
				t.Errorf("Expected color 'ff0000', got '%s'", input.Color)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.UpdateLabel("owner", "repo", "bug", "bug-renamed", "ff0000", "New description")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestUpdateLabel_LabelNotFound(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			// Don't populate label ID - label not found
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.UpdateLabel("owner", "repo", "nonexistent", "new-name", "ff0000", "")

	if err == nil {
		t.Fatal("Expected error when label not found")
	}
	if !strings.Contains(err.Error(), "failed to get label ID") {
		t.Errorf("Expected 'failed to get label ID' error, got: %v", err)
	}
}

func TestUpdateLabel_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				label.FieldByName("ID").SetString("label-123")
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("permission denied")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.UpdateLabel("owner", "repo", "bug", "new-name", "ff0000", "")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to update label") {
		t.Errorf("Expected 'failed to update label' error, got: %v", err)
	}
}

// ============================================================================
// getUserID Tests
// ============================================================================

func TestGetUserID_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetUserID" {
				v := reflect.ValueOf(query).Elem()
				user := v.FieldByName("User")
				user.FieldByName("ID").SetString("user-123")
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	userID, err := client.getUserID("testuser")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got '%s'", userID)
	}
}

func TestGetUserID_UserNotFound(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			// Don't populate user ID - user not found
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.getUserID("nonexistent")

	if err == nil {
		t.Fatal("Expected error when user not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestGetUserID_QueryError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("network error")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.getUserID("testuser")

	if err == nil {
		t.Fatal("Expected error when query fails")
	}
	if !strings.Contains(err.Error(), "failed to get user ID") {
		t.Errorf("Expected 'failed to get user ID' error, got: %v", err)
	}
}

// ============================================================================
// getMilestoneID Tests
// ============================================================================

func TestGetMilestoneID_ByTitle(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetMilestones" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				milestones := repo.FieldByName("Milestones")
				nodes := milestones.FieldByName("Nodes")

				// Create slice with one milestone
				nodeType := nodes.Type().Elem()
				newNodes := reflect.MakeSlice(nodes.Type(), 1, 1)
				newNode := reflect.New(nodeType).Elem()
				newNode.FieldByName("ID").SetString("milestone-123")
				newNode.FieldByName("Title").SetString("v1.0.0")
				newNode.FieldByName("Number").SetInt(1)
				newNodes.Index(0).Set(newNode)
				nodes.Set(newNodes)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	milestoneID, err := client.getMilestoneID("owner", "repo", "v1.0.0")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if milestoneID != "milestone-123" {
		t.Errorf("Expected milestone ID 'milestone-123', got '%s'", milestoneID)
	}
}

func TestGetMilestoneID_ByNumber(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetMilestones" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				milestones := repo.FieldByName("Milestones")
				nodes := milestones.FieldByName("Nodes")

				// Create slice with one milestone
				nodeType := nodes.Type().Elem()
				newNodes := reflect.MakeSlice(nodes.Type(), 1, 1)
				newNode := reflect.New(nodeType).Elem()
				newNode.FieldByName("ID").SetString("milestone-456")
				newNode.FieldByName("Title").SetString("Sprint 1")
				newNode.FieldByName("Number").SetInt(5)
				newNodes.Index(0).Set(newNode)
				nodes.Set(newNodes)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	milestoneID, err := client.getMilestoneID("owner", "repo", "5")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if milestoneID != "milestone-456" {
		t.Errorf("Expected milestone ID 'milestone-456', got '%s'", milestoneID)
	}
}

func TestGetMilestoneID_NotFound(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			// Return empty milestones
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.getMilestoneID("owner", "repo", "nonexistent")

	if err == nil {
		t.Fatal("Expected error when milestone not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestGetMilestoneID_QueryError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("network error")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.getMilestoneID("owner", "repo", "v1.0.0")

	if err == nil {
		t.Fatal("Expected error when query fails")
	}
	if !strings.Contains(err.Error(), "failed to get milestones") {
		t.Errorf("Expected 'failed to get milestones' error, got: %v", err)
	}
}

// ============================================================================
// CreateIssueWithOptions Tests
// ============================================================================

func TestCreateIssueWithOptions_GetRepositoryIDError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("repo not found")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.CreateIssueWithOptions("owner", "repo", "title", "body", nil, nil, "")

	if err == nil {
		t.Fatal("Expected error when getRepositoryID fails")
	}
}

func TestCreateIssueWithOptions_BasicSuccess(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				repo.FieldByName("ID").SetString("repo-123")
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "CreateIssue" {
				t.Errorf("Expected mutation name 'CreateIssue', got '%s'", name)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	issue, err := client.CreateIssueWithOptions("owner", "repo", "title", "body", nil, nil, "")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if issue == nil {
		t.Fatal("Expected issue to be returned")
	}
}

func TestCreateIssueWithOptions_WithAssignees(t *testing.T) {
	queryCount := 0
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			queryCount++
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				repo.FieldByName("ID").SetString("repo-123")
			}
			if name == "GetUserID" {
				v := reflect.ValueOf(query).Elem()
				user := v.FieldByName("User")
				user.FieldByName("ID").SetString("user-123")
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.CreateIssueWithOptions("owner", "repo", "title", "body", nil, []string{"testuser"}, "")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestCreateIssueWithOptions_WithMilestone(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				repo.FieldByName("ID").SetString("repo-123")
			}
			if name == "GetMilestones" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				milestones := repo.FieldByName("Milestones")
				nodes := milestones.FieldByName("Nodes")

				nodeType := nodes.Type().Elem()
				newNodes := reflect.MakeSlice(nodes.Type(), 1, 1)
				newNode := reflect.New(nodeType).Elem()
				newNode.FieldByName("ID").SetString("milestone-123")
				newNode.FieldByName("Title").SetString("v1.0.0")
				newNode.FieldByName("Number").SetInt(1)
				newNodes.Index(0).Set(newNode)
				nodes.Set(newNodes)
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.CreateIssueWithOptions("owner", "repo", "title", "body", nil, nil, "v1.0.0")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestCreateIssueWithOptions_MilestoneNotFound(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				repo.FieldByName("ID").SetString("repo-123")
			}
			// Don't populate milestones - not found
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	// Should succeed but print a warning
	_, err := client.CreateIssueWithOptions("owner", "repo", "title", "body", nil, nil, "nonexistent")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestCreateIssueWithOptions_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetRepositoryID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				repo.FieldByName("ID").SetString("repo-123")
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("create issue failed")
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.CreateIssueWithOptions("owner", "repo", "title", "body", nil, nil, "")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to create issue") {
		t.Errorf("Expected 'failed to create issue' error, got: %v", err)
	}
}

// ============================================================================
// DeleteProjectField Tests
// ============================================================================

func TestDeleteProjectField_Success(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name != "DeleteProjectV2Field" {
				t.Errorf("Expected mutation name 'DeleteProjectV2Field', got '%s'", name)
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.DeleteProjectField("FIELD_123")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestDeleteProjectField_MutationError(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			return errors.New("cannot delete built-in field")
		},
	}

	client := NewClientWithGraphQL(mock)
	err := client.DeleteProjectField("FIELD_123")

	if err == nil {
		t.Fatal("Expected error when mutation fails")
	}
	if !strings.Contains(err.Error(), "failed to delete project field") {
		t.Errorf("Expected 'failed to delete project field' error, got: %v", err)
	}
}

func TestDeleteProjectField_InputVariables(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			input, ok := variables["input"].(DeleteProjectV2FieldInput)
			if !ok {
				t.Fatal("Expected input to be DeleteProjectV2FieldInput type")
			}
			if input.FieldID != "FIELD_ABC" {
				t.Errorf("Expected FieldID 'FIELD_ABC', got '%s'", input.FieldID)
			}
			return nil
		},
	}
	client := NewClientWithGraphQL(mock)
	_ = client.DeleteProjectField("FIELD_ABC")
}

func TestDeleteProjectField_EmptyFieldID(t *testing.T) {
	mock := &mockGraphQLClient{
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			t.Error("Mutation should not be called with empty field ID")
			return nil
		},
	}
	client := NewClientWithGraphQL(mock)
	err := client.DeleteProjectField("")
	if err == nil {
		t.Error("Expected error for empty field ID")
	}
	if err != nil && !strings.Contains(err.Error(), "field ID is required") {
		t.Errorf("Expected 'field ID is required' error, got: %v", err)
	}
}

// ============================================================================
// resolveLabelIDs Tests
// ============================================================================

func TestResolveLabelIDs_FoundLabels(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetLabelID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				label.FieldByName("ID").SetString("label-123")
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	ids, err := client.resolveLabelIDs("owner", "repo", []string{"bug"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("Expected 1 label ID, got %d", len(ids))
	}
}

func TestResolveLabelIDs_AutoCreatesStandardLabel(t *testing.T) {
	queryCallCount := 0
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			queryCallCount++
			if name == "GetLabelID" {
				v := reflect.ValueOf(query).Elem()
				repo := v.FieldByName("Repository")
				label := repo.FieldByName("Label")
				if queryCallCount > 1 {
					// After creation, return the label ID
					label.FieldByName("ID").SetString("label-bug-123")
				}
				// First call returns empty ID (label not found)
			}
			return nil
		},
		mutateFunc: func(name string, mutation interface{}, variables map[string]interface{}) error {
			if name == "CreateLabel" {
				input := variables["input"].(CreateLabelInput)
				if string(input.Name) != "bug" {
					t.Errorf("Expected label name 'bug', got '%s'", input.Name)
				}
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	ids, err := client.resolveLabelIDs("owner", "repo", []string{"bug"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("Expected 1 label ID, got %d", len(ids))
	}
}

func TestResolveLabelIDs_ErrorsForNonStandardLabel(t *testing.T) {
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	_, err := client.resolveLabelIDs("owner", "repo", []string{"custom-nonexistent"})

	if err == nil {
		t.Fatal("Expected error for non-standard label")
	}
	if !strings.Contains(err.Error(), "is not a standard label") {
		t.Errorf("Expected 'is not a standard label' error, got: %v", err)
	}
}

// ============================================================================
// GetProjectItemID Pagination Tests
// ============================================================================

func TestGetProjectItemID_Pagination_FindsItemOnPage2(t *testing.T) {
	callCount := 0
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetProjectItems" {
				callCount++
				v := reflect.ValueOf(query).Elem()
				node := v.FieldByName("Node")
				projectV2 := node.FieldByName("ProjectV2")
				items := projectV2.FieldByName("Items")
				nodes := items.FieldByName("Nodes")
				pageInfoField := items.FieldByName("PageInfo")

				nodeType := nodes.Type().Elem()

				if callCount == 1 {
					// Page 1: item with different issue ID, hasNextPage=true
					newNodes := reflect.MakeSlice(nodes.Type(), 1, 1)
					n1 := reflect.New(nodeType).Elem()
					n1.FieldByName("ID").SetString("item-1")
					content1 := n1.FieldByName("Content")
					issue1 := content1.FieldByName("Issue")
					issue1.FieldByName("ID").SetString("issue-AAA")
					newNodes.Index(0).Set(n1)
					nodes.Set(newNodes)

					pageInfoField.FieldByName("HasNextPage").SetBool(true)
					pageInfoField.FieldByName("EndCursor").SetString("cursor-1")
				} else if callCount == 2 {
					// Page 2: target item, hasNextPage=false
					newNodes := reflect.MakeSlice(nodes.Type(), 1, 1)
					n2 := reflect.New(nodeType).Elem()
					n2.FieldByName("ID").SetString("item-2")
					content2 := n2.FieldByName("Content")
					issue2 := content2.FieldByName("Issue")
					issue2.FieldByName("ID").SetString("issue-BBB")
					newNodes.Index(0).Set(n2)
					nodes.Set(newNodes)

					pageInfoField.FieldByName("HasNextPage").SetBool(false)
					pageInfoField.FieldByName("EndCursor").SetString("")
				}
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	itemID, err := client.GetProjectItemID("proj-123", "issue-BBB")

	if err != nil {
		t.Fatalf("Expected to find item on page 2, got error: %v", err)
	}
	if itemID != "item-2" {
		t.Errorf("Expected item ID 'item-2', got '%s'", itemID)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 API calls (pagination), got %d", callCount)
	}
}

func TestGetProjectItemID_FindsItemOnPage1(t *testing.T) {
	callCount := 0
	mock := &mockGraphQLClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			if name == "GetProjectItems" {
				callCount++
				v := reflect.ValueOf(query).Elem()
				node := v.FieldByName("Node")
				projectV2 := node.FieldByName("ProjectV2")
				items := projectV2.FieldByName("Items")
				nodes := items.FieldByName("Nodes")
				pageInfoField := items.FieldByName("PageInfo")

				nodeType := nodes.Type().Elem()

				newNodes := reflect.MakeSlice(nodes.Type(), 1, 1)
				n1 := reflect.New(nodeType).Elem()
				n1.FieldByName("ID").SetString("item-1")
				content1 := n1.FieldByName("Content")
				issue1 := content1.FieldByName("Issue")
				issue1.FieldByName("ID").SetString("issue-AAA")
				newNodes.Index(0).Set(n1)
				nodes.Set(newNodes)

				pageInfoField.FieldByName("HasNextPage").SetBool(false)
				pageInfoField.FieldByName("EndCursor").SetString("")
			}
			return nil
		},
	}

	client := NewClientWithGraphQL(mock)
	itemID, err := client.GetProjectItemID("proj-123", "issue-AAA")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if itemID != "item-1" {
		t.Errorf("Expected item ID 'item-1', got '%s'", itemID)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 API call, got %d", callCount)
	}
}
