package api

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	graphql "github.com/cli/shurcooL-graphql"

	"github.com/rubrical-works/gh-pmu/internal/defaults"
)

// CreateIssue creates a new issue in a repository
func (c *Client) CreateIssue(owner, repo, title, body string, labels []string) (*Issue, error) {

	// First, get the repository ID
	repoID, err := c.GetRepositoryID(owner, repo)
	if err != nil {
		return nil, err
	}

	// Resolve label names to IDs (batch lookup + auto-create standard labels)
	labelIDs, err := c.resolveLabelIDs(owner, repo, labels)
	if err != nil {
		return nil, err
	}

	var mutation struct {
		CreateIssue struct {
			Issue struct {
				ID     string
				Number int
				Title  string
				Body   string
				State  string
				URL    string `graphql:"url"`
			}
		} `graphql:"createIssue(input: $input)"`
	}

	input := CreateIssueInput{
		RepositoryID: graphql.ID(repoID),
		Title:        graphql.String(title),
	}
	if body != "" {
		input.Body = graphql.String(body)
	}
	if len(labelIDs) > 0 {
		input.LabelIDs = &labelIDs
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err = c.gql.Mutate("CreateIssue", &mutation, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	return &Issue{
		ID:     mutation.CreateIssue.Issue.ID,
		Number: mutation.CreateIssue.Issue.Number,
		Title:  mutation.CreateIssue.Issue.Title,
		Body:   mutation.CreateIssue.Issue.Body,
		State:  mutation.CreateIssue.Issue.State,
		URL:    mutation.CreateIssue.Issue.URL,
		Repository: Repository{
			Owner: owner,
			Name:  repo,
		},
	}, nil
}

// CreateIssueInput represents the input for creating an issue
type CreateIssueInput struct {
	RepositoryID graphql.ID     `json:"repositoryId"`
	Title        graphql.String `json:"title"`
	Body         graphql.String `json:"body,omitempty"`
	LabelIDs     *[]graphql.ID  `json:"labelIds,omitempty"`
	AssigneeIDs  *[]graphql.ID  `json:"assigneeIds,omitempty"`
	MilestoneID  *graphql.ID    `json:"milestoneId,omitempty"`
}

// CloseIssueInput represents the input for closing an issue
type CloseIssueInput struct {
	IssueID graphql.ID `json:"issueId"`
}

// ReopenIssueInput represents the input for reopening an issue
type ReopenIssueInput struct {
	IssueID graphql.ID `json:"issueId"`
}

// UpdateIssueInput represents the input for updating an issue
type UpdateIssueInput struct {
	ID    graphql.ID     `json:"id"`
	Body  graphql.String `json:"body,omitempty"`
	Title graphql.String `json:"title,omitempty"`
}

// AddIssueToProject adds an issue to a GitHub Project V2
func (c *Client) AddIssueToProject(projectID, issueID string) (string, error) {

	var mutation struct {
		AddProjectV2ItemById struct {
			Item struct {
				ID string
			}
		} `graphql:"addProjectV2ItemById(input: $input)"`
	}

	input := AddProjectV2ItemByIdInput{
		ProjectID: graphql.ID(projectID),
		ContentID: graphql.ID(issueID),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("AddProjectV2ItemById", &mutation, variables)
	if err != nil {
		return "", fmt.Errorf("failed to add issue to project: %w", err)
	}

	return mutation.AddProjectV2ItemById.Item.ID, nil
}

// AddProjectV2ItemByIdInput represents the input for adding an item to a project
type AddProjectV2ItemByIdInput struct {
	ProjectID graphql.ID `json:"projectId"`
	ContentID graphql.ID `json:"contentId"`
}

// SetProjectItemField sets a field value on a project item.
// This method fetches project fields on each call. For bulk operations,
// use SetProjectItemFieldWithFields with pre-fetched fields for better performance.
func (c *Client) SetProjectItemField(projectID, itemID, fieldName, value string) error {

	// Get the field ID and option ID for single select fields
	fields, err := c.GetProjectFields(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project fields: %w", err)
	}

	return c.SetProjectItemFieldWithFields(projectID, itemID, fieldName, value, fields)
}

// SetProjectItemFieldWithFields sets a field value using pre-fetched project fields.
// Use this method for bulk operations to avoid redundant GetProjectFields API calls.
func (c *Client) SetProjectItemFieldWithFields(projectID, itemID, fieldName, value string, fields []ProjectField) error {

	var field *ProjectField
	for i := range fields {
		if fields[i].Name == fieldName {
			field = &fields[i]
			break
		}
	}

	if field == nil {
		return fmt.Errorf("field %q not found in project", fieldName)
	}

	// Handle different field types
	switch field.DataType {
	case "SINGLE_SELECT":
		return c.setSingleSelectField(projectID, itemID, field, value)
	case "TEXT":
		return c.setTextField(projectID, itemID, field.ID, value)
	case "NUMBER":
		return c.setNumberField(projectID, itemID, field.ID, value)
	case "DATE":
		return c.setDateField(projectID, itemID, field.ID, value)
	default:
		return fmt.Errorf("unsupported field type: %s", field.DataType)
	}
}

func (c *Client) setSingleSelectField(projectID, itemID string, field *ProjectField, value string) error {
	// Find the option ID for the value
	var optionID string
	for _, opt := range field.Options {
		if opt.Name == value {
			optionID = opt.ID
			break
		}
	}

	if optionID == "" {
		return fmt.Errorf("option %q not found for field %q", value, field.Name)
	}

	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ClientMutationID string `graphql:"clientMutationId"`
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := UpdateProjectV2ItemFieldValueInput{
		ProjectID: graphql.ID(projectID),
		ItemID:    graphql.ID(itemID),
		FieldID:   graphql.ID(field.ID),
		Value: ProjectV2FieldValue{
			SingleSelectOptionId: graphql.String(optionID),
		},
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("UpdateProjectV2ItemFieldValue", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to set field value: %w", err)
	}

	return nil
}

func (c *Client) setTextField(projectID, itemID, fieldID, value string) error {
	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ClientMutationID string `graphql:"clientMutationId"`
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := UpdateProjectV2ItemFieldValueInput{
		ProjectID: graphql.ID(projectID),
		ItemID:    graphql.ID(itemID),
		FieldID:   graphql.ID(fieldID),
		Value: ProjectV2FieldValue{
			Text: graphql.String(value),
		},
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("UpdateProjectV2ItemFieldValue", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to set text field value: %w", err)
	}

	return nil
}

func (c *Client) setNumberField(projectID, itemID, fieldID, value string) error {
	numValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid number value %q: %w", value, err)
	}

	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ClientMutationID string `graphql:"clientMutationId"`
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := UpdateProjectV2ItemFieldValueInput{
		ProjectID: graphql.ID(projectID),
		ItemID:    graphql.ID(itemID),
		FieldID:   graphql.ID(fieldID),
		Value: ProjectV2FieldValue{
			Number: graphql.Float(numValue),
		},
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err = c.gql.Mutate("UpdateProjectV2ItemFieldValue", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to set number field value: %w", err)
	}

	return nil
}

func (c *Client) setDateField(projectID, itemID, fieldID, value string) error {
	// Handle empty value to clear the date field
	if value == "" {
		var mutation struct {
			ClearProjectV2ItemFieldValue struct {
				ClientMutationID string `graphql:"clientMutationId"`
			} `graphql:"clearProjectV2ItemFieldValue(input: $input)"`
		}

		input := struct {
			ProjectID graphql.ID `json:"projectId"`
			ItemID    graphql.ID `json:"itemId"`
			FieldID   graphql.ID `json:"fieldId"`
		}{
			ProjectID: graphql.ID(projectID),
			ItemID:    graphql.ID(itemID),
			FieldID:   graphql.ID(fieldID),
		}

		err := c.gql.Mutate("ClearProjectV2ItemFieldValue", &mutation, map[string]interface{}{
			"input": input,
		})
		if err != nil {
			return fmt.Errorf("failed to clear date field: %w", err)
		}
		return nil
	}

	// Validate date format (YYYY-MM-DD)
	_, err := time.Parse("2006-01-02", value)
	if err != nil {
		return fmt.Errorf("invalid date format: expected YYYY-MM-DD, got %q", value)
	}

	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ClientMutationID string `graphql:"clientMutationId"`
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := UpdateProjectV2ItemFieldValueInput{
		ProjectID: graphql.ID(projectID),
		ItemID:    graphql.ID(itemID),
		FieldID:   graphql.ID(fieldID),
		Value: ProjectV2FieldValue{
			Date: graphql.String(value),
		},
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err = c.gql.Mutate("UpdateProjectV2ItemFieldValue", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to set date field value: %w", err)
	}

	return nil
}

// UpdateProjectV2ItemFieldValueInput represents the input for updating a field value
type UpdateProjectV2ItemFieldValueInput struct {
	ProjectID graphql.ID          `json:"projectId"`
	ItemID    graphql.ID          `json:"itemId"`
	FieldID   graphql.ID          `json:"fieldId"`
	Value     ProjectV2FieldValue `json:"value"`
}

// ProjectV2FieldValue represents a field value for a project item
type ProjectV2FieldValue struct {
	Text                 graphql.String `json:"text,omitempty"`
	Number               graphql.Float  `json:"number,omitempty"`
	Date                 graphql.String `json:"date,omitempty"`
	SingleSelectOptionId graphql.String `json:"singleSelectOptionId,omitempty"`
	IterationId          graphql.String `json:"iterationId,omitempty"`
}

// Helper methods

// GetRepositoryID returns the node ID for a repository.
func (c *Client) GetRepositoryID(owner, repo string) (string, error) {
	var query struct {
		Repository struct {
			ID string
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]interface{}{
		"owner": graphql.String(owner),
		"repo":  graphql.String(repo),
	}

	err := c.gql.Query("GetRepositoryID", &query, variables)
	if err != nil {
		return "", fmt.Errorf("failed to get repository ID: %w", err)
	}

	return query.Repository.ID, nil
}

// AddSubIssue links a child issue as a sub-issue of a parent issue
func (c *Client) AddSubIssue(parentIssueID, childIssueID string) error {

	var mutation struct {
		AddSubIssue struct {
			Issue struct {
				ID string
			}
			SubIssue struct {
				ID string
			}
		} `graphql:"addSubIssue(input: $input)"`
	}

	input := AddSubIssueInput{
		IssueID:    graphql.ID(parentIssueID),
		SubIssueID: graphql.ID(childIssueID),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("AddSubIssue", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to add sub-issue: %w", err)
	}

	return nil
}

// AddSubIssueInput represents the input for adding a sub-issue
type AddSubIssueInput struct {
	IssueID    graphql.ID `json:"issueId"`
	SubIssueID graphql.ID `json:"subIssueId"`
}

// RemoveSubIssue removes a child issue from its parent issue
func (c *Client) RemoveSubIssue(parentIssueID, childIssueID string) error {

	var mutation struct {
		RemoveSubIssue struct {
			Issue struct {
				ID string
			}
			SubIssue struct {
				ID string
			}
		} `graphql:"removeSubIssue(input: $input)"`
	}

	input := RemoveSubIssueInput{
		IssueID:    graphql.ID(parentIssueID),
		SubIssueID: graphql.ID(childIssueID),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("RemoveSubIssue", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to remove sub-issue: %w", err)
	}

	return nil
}

// RemoveSubIssueInput represents the input for removing a sub-issue
type RemoveSubIssueInput struct {
	IssueID    graphql.ID `json:"issueId"`
	SubIssueID graphql.ID `json:"subIssueId"`
}

// CreateProjectField creates a new field in a GitHub project.
// Supported field types: TEXT, NUMBER, DATE, SINGLE_SELECT, ITERATION
func (c *Client) CreateProjectField(projectID, name, dataType string, singleSelectOptions []string) (*ProjectField, error) {

	var mutation struct {
		CreateProjectV2Field struct {
			ProjectV2Field struct {
				TypeName       string `graphql:"__typename"`
				ProjectV2Field struct {
					ID   string
					Name string
				} `graphql:"... on ProjectV2Field"`
				ProjectV2SingleSelectField struct {
					ID      string
					Name    string
					Options []struct {
						ID   string
						Name string
					}
				} `graphql:"... on ProjectV2SingleSelectField"`
			} `graphql:"projectV2Field"`
		} `graphql:"createProjectV2Field(input: $input)"`
	}

	input := CreateProjectV2FieldInput{
		ProjectID: graphql.ID(projectID),
		DataType:  graphql.String(dataType),
		Name:      graphql.String(name),
	}

	// Add single select options if provided
	if dataType == "SINGLE_SELECT" && len(singleSelectOptions) > 0 {
		var options []ProjectV2SingleSelectFieldOptionInput
		for _, opt := range singleSelectOptions {
			options = append(options, ProjectV2SingleSelectFieldOptionInput{
				Name: graphql.String(opt),
			})
		}
		input.SingleSelectOptions = &options
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("CreateProjectV2Field", &mutation, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to create project field: %w", err)
	}

	// Build result based on field type
	result := &ProjectField{
		Name:     name,
		DataType: dataType,
	}

	// Extract ID based on the type returned
	if dataType == "SINGLE_SELECT" {
		result.ID = mutation.CreateProjectV2Field.ProjectV2Field.ProjectV2SingleSelectField.ID
		result.Name = mutation.CreateProjectV2Field.ProjectV2Field.ProjectV2SingleSelectField.Name
		for _, opt := range mutation.CreateProjectV2Field.ProjectV2Field.ProjectV2SingleSelectField.Options {
			result.Options = append(result.Options, FieldOption{
				ID:   opt.ID,
				Name: opt.Name,
			})
		}
	} else {
		result.ID = mutation.CreateProjectV2Field.ProjectV2Field.ProjectV2Field.ID
		result.Name = mutation.CreateProjectV2Field.ProjectV2Field.ProjectV2Field.Name
	}

	return result, nil
}

// CreateProjectV2FieldInput represents the input for creating a project field
type CreateProjectV2FieldInput struct {
	ProjectID           graphql.ID                               `json:"projectId"`
	DataType            graphql.String                           `json:"dataType"`
	Name                graphql.String                           `json:"name"`
	SingleSelectOptions *[]ProjectV2SingleSelectFieldOptionInput `json:"singleSelectOptions,omitempty"`
}

// ProjectV2SingleSelectFieldOptionInput represents an option for a single select field
type ProjectV2SingleSelectFieldOptionInput struct {
	Name        graphql.String `json:"name"`
	Color       graphql.String `json:"color,omitempty"`
	Description graphql.String `json:"description,omitempty"`
}

// DeleteProjectV2FieldInput represents the input for deleting a project field
type DeleteProjectV2FieldInput struct {
	FieldID graphql.ID `json:"fieldId"`
}

// DeleteProjectField deletes a field from a GitHub project.
// Note: Built-in fields (Title, Assignees, etc.) cannot be deleted.
func (c *Client) DeleteProjectField(fieldID string) error {
	if fieldID == "" {
		return fmt.Errorf("field ID is required")
	}

	var mutation struct {
		DeleteProjectV2Field struct {
			ClientMutationID string `graphql:"clientMutationId"`
		} `graphql:"deleteProjectV2Field(input: $input)"`
	}

	input := DeleteProjectV2FieldInput{
		FieldID: graphql.ID(fieldID),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("DeleteProjectV2Field", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to delete project field: %w", err)
	}

	return nil
}

// CopyProjectV2Input represents the input for copying a project.
type CopyProjectV2Input struct {
	OwnerId            graphql.ID      `json:"ownerId"`
	ProjectId          graphql.ID      `json:"projectId"`
	Title              graphql.String  `json:"title"`
	IncludeDraftIssues graphql.Boolean `json:"includeDraftIssues"`
}

// CopyProjectFromTemplate creates a new project by copying from a template project.
// ownerID is the node ID of the owner (user or organization)
// sourceProjectID is the node ID of the template project to copy from
// title is the title for the new project
func (c *Client) CopyProjectFromTemplate(ownerID, sourceProjectID, title string) (*Project, error) {

	var mutation struct {
		CopyProjectV2 struct {
			ProjectV2 struct {
				ID     string
				Number int
				Title  string
				URL    string
			}
		} `graphql:"copyProjectV2(input: $input)"`
	}

	input := CopyProjectV2Input{
		OwnerId:            graphql.ID(ownerID),
		ProjectId:          graphql.ID(sourceProjectID),
		Title:              graphql.String(title),
		IncludeDraftIssues: graphql.Boolean(false),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("CopyProjectV2", &mutation, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to copy project: %w", err)
	}

	return &Project{
		ID:     mutation.CopyProjectV2.ProjectV2.ID,
		Number: mutation.CopyProjectV2.ProjectV2.Number,
		Title:  mutation.CopyProjectV2.ProjectV2.Title,
		URL:    mutation.CopyProjectV2.ProjectV2.URL,
	}, nil
}

// GetOwnerID returns the node ID for a user or organization.
func (c *Client) GetOwnerID(owner string) (string, error) {

	// Try as organization first
	var orgQuery struct {
		Organization struct {
			ID string
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login": graphql.String(owner),
	}

	err := c.gql.Query("GetOrganizationID", &orgQuery, variables)
	if err == nil && orgQuery.Organization.ID != "" {
		return orgQuery.Organization.ID, nil
	}

	// Fall back to user
	var userQuery struct {
		User struct {
			ID string
		} `graphql:"user(login: $login)"`
	}

	err = c.gql.Query("GetUserID", &userQuery, variables)
	if err != nil {
		return "", fmt.Errorf("failed to get owner ID for %s: %w", owner, err)
	}

	if userQuery.User.ID == "" {
		return "", fmt.Errorf("owner not found: %s", owner)
	}

	return userQuery.User.ID, nil
}

// LinkProjectToRepository adds a repository to a project's linked repositories.
func (c *Client) LinkProjectToRepository(projectID, repositoryID string) error {

	var mutation struct {
		LinkProjectV2ToRepository struct {
			Repository struct {
				ID string
			}
		} `graphql:"linkProjectV2ToRepository(input: $input)"`
	}

	input := struct {
		ProjectId    graphql.ID `json:"projectId"`
		RepositoryId graphql.ID `json:"repositoryId"`
	}{
		ProjectId:    graphql.ID(projectID),
		RepositoryId: graphql.ID(repositoryID),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("LinkProjectV2ToRepository", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to link repository to project: %w", err)
	}

	return nil
}

// AddLabelToIssue adds a label to an issue.
// If the label doesn't exist in the repository, it will be created automatically.
func (c *Client) AddLabelToIssue(owner, repo, issueID, labelName string) error {

	// Get the label ID, creating the label if it doesn't exist
	labelID, err := c.EnsureLabelExists(owner, repo, labelName)
	if err != nil {
		return err
	}

	// Use addLabelsToLabelable mutation
	var mutation struct {
		AddLabelsToLabelable struct {
			Labelable struct {
				Labels struct {
					TotalCount int
				} `graphql:"labels(first: 0)"`
			}
		} `graphql:"addLabelsToLabelable(input: $input)"`
	}

	type AddLabelsToLabelableInput struct {
		LabelableID graphql.ID   `json:"labelableId"`
		LabelIDs    []graphql.ID `json:"labelIds"`
	}

	input := AddLabelsToLabelableInput{
		LabelableID: graphql.ID(issueID),
		LabelIDs:    []graphql.ID{graphql.ID(labelID)},
	}

	err = c.gql.Mutate("AddLabelsToLabelable", &mutation, map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return fmt.Errorf("failed to add label: %w", err)
	}

	return nil
}

// RemoveLabelFromIssue removes a label from an issue
func (c *Client) RemoveLabelFromIssue(owner, repo, issueID, labelName string) error {

	// Get the label ID first
	labelID, err := c.getLabelID(owner, repo, labelName)
	if err != nil {
		return err
	}

	// Use removeLabelsFromLabelable mutation
	var mutation struct {
		RemoveLabelsFromLabelable struct {
			Labelable struct {
				Labels struct {
					TotalCount int
				} `graphql:"labels(first: 0)"`
			}
		} `graphql:"removeLabelsFromLabelable(input: $input)"`
	}

	type RemoveLabelsFromLabelableInput struct {
		LabelableID graphql.ID   `json:"labelableId"`
		LabelIDs    []graphql.ID `json:"labelIds"`
	}

	input := RemoveLabelsFromLabelableInput{
		LabelableID: graphql.ID(issueID),
		LabelIDs:    []graphql.ID{graphql.ID(labelID)},
	}

	err = c.gql.Mutate("RemoveLabelsFromLabelable", &mutation, map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return fmt.Errorf("failed to remove label: %w", err)
	}

	return nil
}

func (c *Client) getLabelID(owner, repo, labelName string) (string, error) {
	var query struct {
		Repository struct {
			Label struct {
				ID string
			} `graphql:"label(name: $labelName)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]interface{}{
		"owner":     graphql.String(owner),
		"repo":      graphql.String(repo),
		"labelName": graphql.String(labelName),
	}

	err := c.gql.Query("GetLabelID", &query, variables)
	if err != nil {
		return "", fmt.Errorf("failed to get label ID: %w", err)
	}

	if query.Repository.Label.ID == "" {
		return "", fmt.Errorf("label %q not found", labelName)
	}

	return query.Repository.Label.ID, nil
}

// getLabelIDs gets label IDs for multiple labels in a single batch query.
// Returns a map of label name to ID. Labels not found are omitted from the result.
func (c *Client) getLabelIDs(owner, repo string, labelNames []string) (map[string]string, error) {
	if len(labelNames) == 0 {
		return make(map[string]string), nil
	}

	// Build a GraphQL query with aliases for each label
	// Example: query { repository(owner:"o", name:"r") { l0: label(name:"bug") { id } l1: label(name:"help") { id } } }
	var queryParts []string
	for i, name := range labelNames {
		// Escape the label name for GraphQL string literal
		escapedName := strings.ReplaceAll(name, `"`, `\"`)
		queryParts = append(queryParts, fmt.Sprintf(`l%d: label(name: %q) { id }`, i, escapedName))
	}

	query := fmt.Sprintf(`query { repository(owner: %q, name: %q) { %s } }`,
		owner, repo, strings.Join(queryParts, " "))

	// Execute via gh api graphql using stdin to avoid Windows command-line length limits
	requestBody, err := buildGraphQLRequestBody(query)
	if err != nil {
		return nil, fmt.Errorf("failed to build batch label request: %w", err)
	}
	cmd := exec.Command("gh", "api", "graphql", "--input", "-")
	cmd.Stdin = strings.NewReader(requestBody)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to execute batch label query: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute batch label query: %w", err)
	}

	// Parse the response
	var response struct {
		Data struct {
			Repository map[string]struct {
				ID string `json:"id"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse batch label response: %w", err)
	}

	// Build result map
	result := make(map[string]string)
	for i, name := range labelNames {
		alias := fmt.Sprintf("l%d", i)
		if labelData, ok := response.Data.Repository[alias]; ok && labelData.ID != "" {
			result[name] = labelData.ID
		}
	}

	return result, nil
}

// resolveLabelIDs resolves label names to GraphQL IDs, auto-creating standard labels as needed.
// Returns an error if any label is not a recognized standard label.
func (c *Client) resolveLabelIDs(owner, repo string, labels []string) ([]graphql.ID, error) {
	if len(labels) == 0 {
		return nil, nil
	}

	// Load defaults for validation and auto-creation
	defs, loadErr := defaults.Load()
	if loadErr != nil {
		return nil, fmt.Errorf("failed to load defaults: %w", loadErr)
	}

	labelIDMap, err := c.getLabelIDs(owner, repo, labels)
	if err != nil {
		labelIDMap = make(map[string]string)
	}

	var labelIDs []graphql.ID
	for _, labelName := range labels {
		if id, ok := labelIDMap[labelName]; ok {
			labelIDs = append(labelIDs, graphql.ID(id))
			continue
		}

		// Label not found in repo - check if it's a standard label
		labelDef := defs.GetLabel(labelName)
		if labelDef == nil {
			// Not a standard label - error out with helpful message
			return nil, fmt.Errorf("label %q is not a standard label\nAvailable standard labels: %s",
				labelName, strings.Join(defs.GetLabelNames(), ", "))
		}

		// Auto-create from defaults
		fmt.Fprintf(os.Stderr, "Creating standard label %q...\n", labelName)
		if createErr := c.CreateLabel(owner, repo, labelDef.Name, labelDef.Color, labelDef.Description); createErr != nil {
			return nil, fmt.Errorf("failed to create label %q: %w", labelName, createErr)
		}
		// Get the newly created label's ID
		newID, getErr := c.getLabelID(owner, repo, labelName)
		if getErr != nil {
			return nil, fmt.Errorf("failed to get ID for newly created label %q: %w", labelName, getErr)
		}
		labelIDs = append(labelIDs, graphql.ID(newID))
	}

	return labelIDs, nil
}

// EnsureLabelExists checks if a label exists and creates it if not.
// Returns the label ID. Only creates labels that are defined in the standard defaults.
// Returns an error for non-standard labels.
func (c *Client) EnsureLabelExists(owner, repo, labelName string) (string, error) {
	// First try to get the existing label
	labelID, err := c.getLabelID(owner, repo, labelName)
	if err == nil {
		return labelID, nil
	}

	// Label doesn't exist - check if it's a standard label
	defs, loadErr := defaults.Load()
	if loadErr != nil {
		return "", fmt.Errorf("failed to load defaults: %w", loadErr)
	}

	labelDef := defs.GetLabel(labelName)
	if labelDef == nil {
		// Not a standard label - return error with helpful message
		return "", fmt.Errorf("label %q is not a standard label\nAvailable standard labels: %s",
			labelName, strings.Join(defs.GetLabelNames(), ", "))
	}

	// Auto-create the standard label with its defined properties
	fmt.Fprintf(os.Stderr, "Creating standard label %q in %s/%s...\n", labelName, owner, repo)
	createErr := c.CreateLabel(owner, repo, labelDef.Name, labelDef.Color, labelDef.Description)
	if createErr != nil {
		return "", createErr
	}

	// Now get the label ID
	return c.getLabelID(owner, repo, labelName)
}

// getUserID gets a user's ID from their login
func (c *Client) getUserID(login string) (string, error) {
	var query struct {
		User struct {
			ID string
		} `graphql:"user(login: $login)"`
	}

	variables := map[string]interface{}{
		"login": graphql.String(login),
	}

	err := c.gql.Query("GetUserID", &query, variables)
	if err != nil {
		return "", fmt.Errorf("failed to get user ID for %s: %w", login, err)
	}

	if query.User.ID == "" {
		return "", fmt.Errorf("user %q not found", login)
	}

	return query.User.ID, nil
}

// getMilestoneID gets a milestone ID from the repository
func (c *Client) getMilestoneID(owner, repo, milestone string) (string, error) {
	var query struct {
		Repository struct {
			Milestones struct {
				Nodes []struct {
					ID     string
					Title  string
					Number int
				}
			} `graphql:"milestones(first: 100, states: OPEN)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]interface{}{
		"owner": graphql.String(owner),
		"repo":  graphql.String(repo),
	}

	err := c.gql.Query("GetMilestones", &query, variables)
	if err != nil {
		return "", fmt.Errorf("failed to get milestones: %w", err)
	}

	// Try to match by title or number
	for _, m := range query.Repository.Milestones.Nodes {
		if m.Title == milestone || fmt.Sprintf("%d", m.Number) == milestone {
			return m.ID, nil
		}
	}

	return "", fmt.Errorf("milestone %q not found", milestone)
}

// CreateIssueWithOptions creates an issue with extended options
func (c *Client) CreateIssueWithOptions(owner, repo, title, body string, labels, assignees []string, milestone string) (*Issue, error) {

	// First, get the repository ID
	repoID, err := c.GetRepositoryID(owner, repo)
	if err != nil {
		return nil, err
	}

	// Resolve label names to IDs (batch lookup + auto-create standard labels)
	labelIDs, err := c.resolveLabelIDs(owner, repo, labels)
	if err != nil {
		return nil, err
	}

	// Get assignee IDs
	var assigneeIDs []graphql.ID
	if len(assignees) > 0 {
		for _, login := range assignees {
			userID, err := c.getUserID(login)
			if err != nil {
				// Skip users that don't exist
				continue
			}
			assigneeIDs = append(assigneeIDs, graphql.ID(userID))
		}
	}

	// Get milestone ID
	var milestoneID *graphql.ID
	if milestone != "" {
		mID, err := c.getMilestoneID(owner, repo, milestone)
		if err != nil {
			// Non-fatal, just warn
			fmt.Printf("Warning: milestone %q not found\n", milestone)
		} else {
			gqlID := graphql.ID(mID)
			milestoneID = &gqlID
		}
	}

	var mutation struct {
		CreateIssue struct {
			Issue struct {
				ID     string
				Number int
				Title  string
				Body   string
				State  string
				URL    string `graphql:"url"`
			}
		} `graphql:"createIssue(input: $input)"`
	}

	input := CreateIssueInput{
		RepositoryID: graphql.ID(repoID),
		Title:        graphql.String(title),
	}
	if body != "" {
		input.Body = graphql.String(body)
	}
	if len(labelIDs) > 0 {
		input.LabelIDs = &labelIDs
	}
	if len(assigneeIDs) > 0 {
		input.AssigneeIDs = &assigneeIDs
	}
	if milestoneID != nil {
		input.MilestoneID = milestoneID
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err = c.gql.Mutate("CreateIssue", &mutation, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	return &Issue{
		ID:     mutation.CreateIssue.Issue.ID,
		Number: mutation.CreateIssue.Issue.Number,
		Title:  mutation.CreateIssue.Issue.Title,
		Body:   mutation.CreateIssue.Issue.Body,
		State:  mutation.CreateIssue.Issue.State,
		URL:    mutation.CreateIssue.Issue.URL,
		Repository: Repository{
			Owner: owner,
			Name:  repo,
		},
	}, nil
}

// CloseIssue closes an issue by its ID
func (c *Client) CloseIssue(issueID string) error {

	var mutation struct {
		CloseIssue struct {
			Issue struct {
				ID string
			}
		} `graphql:"closeIssue(input: $input)"`
	}

	input := CloseIssueInput{
		IssueID: graphql.ID(issueID),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("CloseIssue", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to close issue: %w", err)
	}

	return nil
}

// ReopenIssue reopens a closed issue
func (c *Client) ReopenIssue(issueID string) error {

	var mutation struct {
		ReopenIssue struct {
			Issue struct {
				ID string
			}
		} `graphql:"reopenIssue(input: $input)"`
	}

	input := ReopenIssueInput{
		IssueID: graphql.ID(issueID),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("ReopenIssue", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to reopen issue: %w", err)
	}

	return nil
}

// UpdateIssueBody updates the body of an issue
func (c *Client) UpdateIssueBody(issueID, body string) error {

	var mutation struct {
		UpdateIssue struct {
			Issue struct {
				ID string
			}
		} `graphql:"updateIssue(input: $input)"`
	}

	input := UpdateIssueInput{
		ID:   graphql.ID(issueID),
		Body: graphql.String(body),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("UpdateIssue", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to update issue body: %w", err)
	}

	return nil
}

// UpdateIssueTitle updates the title of an issue
func (c *Client) UpdateIssueTitle(issueID, title string) error {

	var mutation struct {
		UpdateIssue struct {
			Issue struct {
				ID string
			}
		} `graphql:"updateIssue(input: $input)"`
	}

	input := UpdateIssueInput{
		ID:    graphql.ID(issueID),
		Title: graphql.String(title),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("UpdateIssue", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to update issue title: %w", err)
	}

	return nil
}

// GetIssueByNumber returns an issue by its number (alias for GetIssue)
func (c *Client) GetIssueByNumber(owner, repo string, number int) (*Issue, error) {
	return c.GetIssue(owner, repo, number)
}

// GetProjectItemID returns the project item ID for an issue in a project.
// Paginates through all project items to find the matching issue.
func (c *Client) GetProjectItemID(projectID, issueID string) (string, error) {

	var cursor *graphql.String

	for {
		var query struct {
			Node struct {
				ProjectV2 struct {
					Items struct {
						Nodes []struct {
							ID      string
							Content struct {
								Issue struct {
									ID string
								} `graphql:"... on Issue"`
							}
						}
						PageInfo struct {
							HasNextPage bool
							EndCursor   string
						}
					} `graphql:"items(first: 100, after: $cursor)"`
				} `graphql:"... on ProjectV2"`
			} `graphql:"node(id: $projectId)"`
		}

		variables := map[string]interface{}{
			"projectId": graphql.ID(projectID),
			"cursor":    (*graphql.String)(nil),
		}
		if cursor != nil {
			variables["cursor"] = *cursor
		}

		err := c.gql.Query("GetProjectItems", &query, variables)
		if err != nil {
			return "", fmt.Errorf("failed to get project items: %w", err)
		}

		for _, item := range query.Node.ProjectV2.Items.Nodes {
			if item.Content.Issue.ID == issueID {
				return item.ID, nil
			}
		}

		if !query.Node.ProjectV2.Items.PageInfo.HasNextPage {
			break
		}
		next := graphql.String(query.Node.ProjectV2.Items.PageInfo.EndCursor)
		cursor = &next
	}

	return "", fmt.Errorf("issue not found in project")
}

// GetProjectItemFieldValue returns the value of a field on a project item
func (c *Client) GetProjectItemFieldValue(projectID, itemID, fieldName string) (string, error) {

	var query struct {
		Node struct {
			ProjectV2Item struct {
				FieldValues struct {
					Nodes []struct {
						ProjectV2ItemFieldTextValue struct {
							Text  string
							Field struct {
								Name string
							} `graphql:"field"`
						} `graphql:"... on ProjectV2ItemFieldTextValue"`
						ProjectV2ItemFieldSingleSelectValue struct {
							Name  string
							Field struct {
								Name string
							} `graphql:"field"`
						} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
					}
				} `graphql:"fieldValues(first: 20)"`
			} `graphql:"... on ProjectV2Item"`
		} `graphql:"node(id: $itemId)"`
	}

	variables := map[string]interface{}{
		"itemId": graphql.ID(itemID),
	}

	err := c.gql.Query("GetProjectItemFieldValue", &query, variables)
	if err != nil {
		return "", fmt.Errorf("failed to get field value: %w", err)
	}

	for _, fv := range query.Node.ProjectV2Item.FieldValues.Nodes {
		if fv.ProjectV2ItemFieldTextValue.Field.Name == fieldName {
			return fv.ProjectV2ItemFieldTextValue.Text, nil
		}
		if fv.ProjectV2ItemFieldSingleSelectValue.Field.Name == fieldName {
			return fv.ProjectV2ItemFieldSingleSelectValue.Name, nil
		}
	}

	return "", nil
}

// WriteFile writes content to a file path
func (c *Client) WriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// MkdirAll creates a directory and all parents
func (c *Client) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

// GitAdd stages files to git
func (c *Client) GitAdd(paths ...string) error {
	args := append([]string{"add"}, paths...)
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// GitTag creates an annotated git tag
func (c *Client) GitTag(tag, message string) error {
	cmd := exec.Command("git", "tag", "-a", tag, "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git tag failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// GitCommit creates a git commit with the given message
func (c *Client) GitCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// GitCheckoutNewBranch creates and checks out a new git branch
func (c *Client) GitCheckoutNewBranch(branch string) error {
	cmd := exec.Command("git", "checkout", "-b", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout -b failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// GetAuthenticatedUser returns the login of the currently authenticated user
func (c *Client) GetAuthenticatedUser() (string, error) {

	var query struct {
		Viewer struct {
			Login string
		}
	}

	err := c.gql.Query("GetAuthenticatedUser", &query, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}

	return query.Viewer.Login, nil
}

// LabelExists checks if a label exists in a repository
func (c *Client) LabelExists(owner, repo, labelName string) (bool, error) {
	_, err := c.getLabelID(owner, repo, labelName)
	if err != nil {
		// Label not found is not an error for this function
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateLabel creates a new label in a repository
func (c *Client) CreateLabel(owner, repo, name, color, description string) error {

	// Get repository ID first
	repoID, err := c.GetRepositoryID(owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get repository ID: %w", err)
	}

	var mutation struct {
		CreateLabel struct {
			Label struct {
				ID   string
				Name string
			}
		} `graphql:"createLabel(input: $input)"`
	}

	input := CreateLabelInput{
		RepositoryID: graphql.ID(repoID),
		Name:         graphql.String(name),
		Color:        graphql.String(color),
		Description:  graphql.String(description),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err = c.gql.Mutate("CreateLabel", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to create label: %w", err)
	}

	return nil
}

// CreateLabelInput represents the input for creating a label
type CreateLabelInput struct {
	RepositoryID graphql.ID     `json:"repositoryId"`
	Name         graphql.String `json:"name"`
	Color        graphql.String `json:"color"`
	Description  graphql.String `json:"description,omitempty"`
}

// FieldExists checks if a field exists in a project by name
func (c *Client) FieldExists(projectID, fieldName string) (bool, error) {
	fields, err := c.GetProjectFields(projectID)
	if err != nil {
		return false, err
	}
	for _, f := range fields {
		if f.Name == fieldName {
			return true, nil
		}
	}
	return false, nil
}

// AddIssueComment adds a comment to an issue
func (c *Client) AddIssueComment(issueID, body string) (*Comment, error) {

	var mutation struct {
		AddComment struct {
			CommentEdge struct {
				Node struct {
					ID         string
					DatabaseId int `graphql:"databaseId"`
					Body       string
					CreatedAt  string
					Author     struct {
						Login string
					}
				}
			}
		} `graphql:"addComment(input: $input)"`
	}

	input := AddCommentInput{
		SubjectID: graphql.ID(issueID),
		Body:      graphql.String(body),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := c.gql.Mutate("AddComment", &mutation, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	return &Comment{
		ID:         mutation.AddComment.CommentEdge.Node.ID,
		DatabaseId: mutation.AddComment.CommentEdge.Node.DatabaseId,
		Body:       mutation.AddComment.CommentEdge.Node.Body,
		Author:     mutation.AddComment.CommentEdge.Node.Author.Login,
		CreatedAt:  mutation.AddComment.CommentEdge.Node.CreatedAt,
	}, nil
}

// AddCommentInput represents the input for adding a comment
type AddCommentInput struct {
	SubjectID graphql.ID     `json:"subjectId"`
	Body      graphql.String `json:"body"`
}

// DeleteLabel deletes a label from a repository
func (c *Client) DeleteLabel(owner, repo, labelName string) error {

	// Get the label ID first
	labelID, err := c.getLabelID(owner, repo, labelName)
	if err != nil {
		return fmt.Errorf("failed to get label ID: %w", err)
	}

	var mutation struct {
		DeleteLabel struct {
			ClientMutationID string
		} `graphql:"deleteLabel(input: $input)"`
	}

	input := DeleteLabelInput{
		ID: graphql.ID(labelID),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err = c.gql.Mutate("DeleteLabel", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to delete label: %w", err)
	}

	return nil
}

// DeleteLabelInput represents the input for deleting a label
type DeleteLabelInput struct {
	ID graphql.ID `json:"id"`
}

// UpdateLabel updates a label's properties in a repository
func (c *Client) UpdateLabel(owner, repo, labelName, newName, newColor, newDescription string) error {

	// Get the label ID first
	labelID, err := c.getLabelID(owner, repo, labelName)
	if err != nil {
		return fmt.Errorf("failed to get label ID: %w", err)
	}

	var mutation struct {
		UpdateLabel struct {
			Label struct {
				ID   string
				Name string
			}
		} `graphql:"updateLabel(input: $input)"`
	}

	input := UpdateLabelInput{
		ID:          graphql.ID(labelID),
		Name:        graphql.String(newName),
		Color:       graphql.String(newColor),
		Description: graphql.String(newDescription),
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err = c.gql.Mutate("UpdateLabel", &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to update label: %w", err)
	}

	return nil
}

// UpdateLabelInput represents the input for updating a label
type UpdateLabelInput struct {
	ID          graphql.ID     `json:"id"`
	Name        graphql.String `json:"name,omitempty"`
	Color       graphql.String `json:"color,omitempty"`
	Description graphql.String `json:"description,omitempty"`
}

// FieldUpdate represents a single field update for batch operations
type FieldUpdate struct {
	ItemID    string // Project item ID
	FieldName string // Field name (e.g., "Status", "Priority")
	Value     string // Display value (e.g., "In Progress", "P1")
	// Internal fields set during validation
	fieldID  string
	optionID string // For SINGLE_SELECT fields
	dataType string
}

// BatchUpdateResult represents the result of a single update in a batch
type BatchUpdateResult struct {
	ItemID    string
	FieldName string
	Success   bool
	Error     string
}

// BatchUpdateProjectItemFields executes multiple field updates in a single GraphQL mutation.
// This reduces API calls from O(N) to O(N/batchSize) where batchSize is 50.
// Returns results for each update indicating success or failure.
func (c *Client) BatchUpdateProjectItemFields(projectID string, updates []FieldUpdate, fields []ProjectField) ([]BatchUpdateResult, error) {
	if len(updates) == 0 {
		return []BatchUpdateResult{}, nil
	}

	// Build field lookup map
	fieldMap := make(map[string]*ProjectField)
	for i := range fields {
		fieldMap[fields[i].Name] = &fields[i]
	}

	// Validate and prepare all updates
	var validUpdates []FieldUpdate
	var results []BatchUpdateResult

	for _, update := range updates {
		field, ok := fieldMap[update.FieldName]
		if !ok {
			results = append(results, BatchUpdateResult{
				ItemID:    update.ItemID,
				FieldName: update.FieldName,
				Success:   false,
				Error:     fmt.Sprintf("field %q not found in project", update.FieldName),
			})
			continue
		}

		update.fieldID = field.ID
		update.dataType = field.DataType

		// For SINGLE_SELECT, resolve option ID
		if field.DataType == "SINGLE_SELECT" {
			var optionID string
			for _, opt := range field.Options {
				if opt.Name == update.Value {
					optionID = opt.ID
					break
				}
			}
			if optionID == "" {
				results = append(results, BatchUpdateResult{
					ItemID:    update.ItemID,
					FieldName: update.FieldName,
					Success:   false,
					Error:     fmt.Sprintf("option %q not found for field %q", update.Value, update.FieldName),
				})
				continue
			}
			update.optionID = optionID
		}

		// Validate number fields
		if field.DataType == "NUMBER" {
			if _, err := strconv.ParseFloat(update.Value, 64); err != nil {
				results = append(results, BatchUpdateResult{
					ItemID:    update.ItemID,
					FieldName: update.FieldName,
					Success:   false,
					Error:     fmt.Sprintf("invalid number value: %s", update.Value),
				})
				continue
			}
		}

		// Validate date fields
		if field.DataType == "DATE" && update.Value != "" {
			if _, err := time.Parse("2006-01-02", update.Value); err != nil {
				results = append(results, BatchUpdateResult{
					ItemID:    update.ItemID,
					FieldName: update.FieldName,
					Success:   false,
					Error:     fmt.Sprintf("invalid date format (expected YYYY-MM-DD): %s", update.Value),
				})
				continue
			}
		}

		validUpdates = append(validUpdates, update)
	}

	// Process valid updates in batches
	const batchSize = 50
	for i := 0; i < len(validUpdates); i += batchSize {
		end := i + batchSize
		if end > len(validUpdates) {
			end = len(validUpdates)
		}
		batch := validUpdates[i:end]

		batchResults, err := c.executeBatchMutation(projectID, batch)
		if err != nil {
			// If batch fails entirely, mark all as failed
			for _, update := range batch {
				results = append(results, BatchUpdateResult{
					ItemID:    update.ItemID,
					FieldName: update.FieldName,
					Success:   false,
					Error:     err.Error(),
				})
			}
			continue
		}

		results = append(results, batchResults...)
	}

	return results, nil
}

// buildBatchMutationRequest constructs the GraphQL mutation and JSON request body
// for batch field updates. Returns the mutation query string and marshaled JSON body.
// This function is extracted for testability.
func buildBatchMutationRequest(projectID string, updates []FieldUpdate) (mutation string, requestBody string, err error) {
	if len(updates) == 0 {
		return "", "", nil
	}

	// Build mutation with aliases
	var mutationParts []string
	variablesMap := make(map[string]interface{})

	for i, update := range updates {
		alias := fmt.Sprintf("u%d", i)
		varName := fmt.Sprintf("input%d", i)

		// Build the value object based on field type
		var valueObj map[string]interface{}
		switch update.dataType {
		case "SINGLE_SELECT":
			valueObj = map[string]interface{}{"singleSelectOptionId": update.optionID}
		case "TEXT":
			valueObj = map[string]interface{}{"text": update.Value}
		case "NUMBER":
			// Parse number value - ignore error, default to 0 if invalid
			var numVal float64
			_, _ = fmt.Sscanf(update.Value, "%f", &numVal)
			valueObj = map[string]interface{}{"number": numVal}
		case "DATE":
			valueObj = map[string]interface{}{"date": update.Value}
		default:
			return "", "", fmt.Errorf("unsupported field type: %s", update.dataType)
		}

		mutationParts = append(mutationParts, fmt.Sprintf(
			`%s: updateProjectV2ItemFieldValue(input: $%s) { projectV2Item { id } }`,
			alias, varName))

		// Build input variable using proper Go map (json.Marshal will escape properly)
		variablesMap[varName] = map[string]interface{}{
			"projectId": projectID,
			"itemId":    update.ItemID,
			"fieldId":   update.fieldID,
			"value":     valueObj,
		}
	}

	// Build variable declarations for the mutation signature
	var varDecls []string
	for i := range updates {
		varDecls = append(varDecls, fmt.Sprintf("$input%d: UpdateProjectV2ItemFieldValueInput!", i))
	}

	mutation = fmt.Sprintf(`mutation BatchUpdate(%s) { %s }`,
		strings.Join(varDecls, ", "),
		strings.Join(mutationParts, " "))

	// Build request body using json.Marshal for safe escaping
	requestBodyMap := map[string]interface{}{
		"query":     mutation,
		"variables": variablesMap,
	}
	requestBodyBytes, err := json.Marshal(requestBodyMap)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	return mutation, string(requestBodyBytes), nil
}

// executeBatchMutation executes a batch of field updates in a single GraphQL mutation
func (c *Client) executeBatchMutation(projectID string, updates []FieldUpdate) ([]BatchUpdateResult, error) {
	if len(updates) == 0 {
		return []BatchUpdateResult{}, nil
	}

	// Build request (extracted for testability)
	_, requestBody, err := buildBatchMutationRequest(projectID, updates)
	if err != nil {
		return nil, err
	}

	// Execute via gh api graphql
	cmd := exec.Command("gh", "api", "graphql", "--input", "-")

	cmd.Stdin = strings.NewReader(requestBody)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("batch mutation failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("batch mutation failed: %w", err)
	}

	// Parse response
	var response struct {
		Data   map[string]json.RawMessage `json:"data"`
		Errors []struct {
			Message string   `json:"message"`
			Path    []string `json:"path"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse batch mutation response: %w", err)
	}

	// Build results
	var results []BatchUpdateResult
	for i, update := range updates {
		alias := fmt.Sprintf("u%d", i)
		result := BatchUpdateResult{
			ItemID:    update.ItemID,
			FieldName: update.FieldName,
			Success:   true,
		}

		// Check if this specific update failed
		for _, graphErr := range response.Errors {
			if len(graphErr.Path) > 0 && graphErr.Path[0] == alias {
				result.Success = false
				result.Error = graphErr.Message
				break
			}
		}

		// Check if we got a response for this alias
		if _, ok := response.Data[alias]; !ok && result.Success {
			// No data and no specific error - might have failed silently
			if len(response.Errors) > 0 {
				result.Success = false
				result.Error = response.Errors[0].Message
			}
		}

		results = append(results, result)
	}

	return results, nil
}
