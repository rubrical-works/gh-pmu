package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/spf13/cobra"
)

// filterClient defines the interface for API methods used by filter functions.
// This allows for easier testing with mock implementations.
type filterClient interface {
	GetProject(owner string, number int) (*api.Project, error)
	GetProjectItems(projectID string, filter *api.ProjectItemsFilter) ([]api.ProjectItem, error)
	GetProjectItemsByIssues(projectID string, refs []api.IssueRef) ([]api.ProjectItem, error)
}

type filterOptions struct {
	status   string
	priority string
	assignee string
	label    string
	json     bool
}

// FilterInput represents the expected JSON input format from gh issue list
type FilterInput struct {
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	State     string  `json:"state"`
	URL       string  `json:"url"`
	Body      string  `json:"body"`
	Labels    []Label `json:"labels"`
	Assignees []User  `json:"assignees"`
}

// Label represents a label in the input JSON
type Label struct {
	Name string `json:"name"`
}

// User represents a user/assignee in the input JSON
type User struct {
	Login string `json:"login"`
}

func newFilterCommand() *cobra.Command {
	opts := &filterOptions{}

	cmd := &cobra.Command{
		Use:   "filter",
		Short: "Filter piped issue JSON by project field values",
		Long: `Filter JSON input from 'gh issue list' by project field values.

This command reads issue JSON from stdin and filters based on project board
field values like status and priority. Issues are looked up in the configured
project to determine their field values.

Example:
  gh issue list --repo owner/repo --json number,title,state | gh pmu filter --status ready
  gh issue list -R owner/repo --json number,title --limit 100 | gh pmu filter --status in_progress --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFilter(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.status, "status", "s", "", "Filter by status (e.g., backlog, ready, in_progress)")
	cmd.Flags().StringVarP(&opts.priority, "priority", "p", "", "Filter by priority (e.g., p0, p1, p2)")
	cmd.Flags().StringVarP(&opts.assignee, "assignee", "a", "", "Filter by assignee login")
	cmd.Flags().StringVarP(&opts.label, "label", "l", "", "Filter by label name")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format (default is table)")

	return cmd
}

// hasPipedInput checks whether the given file has piped (non-terminal) input.
// Returns false if file is nil or Stat fails, to gracefully handle environments
// where os.Stdin.Stat() returns an error (e.g., certain Windows terminals).
func hasPipedInput(f *os.File) bool {
	if f == nil {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func runFilter(cmd *cobra.Command, opts *filterOptions) error {
	// Check if stdin has data
	if !hasPipedInput(os.Stdin) {
		return fmt.Errorf("no input provided - pipe issue JSON from 'gh issue list --json ...'")
	}

	// Load configuration
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	cfg, err := config.LoadFromDirectory(cwd)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nRun 'gh pmu init' to create a configuration file", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	return runFilterWithDeps(cmd, opts, cfg, client, os.Stdin)
}

// runFilterWithDeps is the testable implementation of runFilter
func runFilterWithDeps(cmd *cobra.Command, opts *filterOptions, cfg *config.Config, client filterClient, stdin *os.File) error {
	// Get project
	project, err := client.GetProject(cfg.Project.Owner, cfg.Project.Number)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Read and parse JSON input from stdin
	var issues []FilterInput
	scanner := bufio.NewScanner(stdin)
	var inputBuilder strings.Builder
	for scanner.Scan() {
		inputBuilder.WriteString(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	inputStr := strings.TrimSpace(inputBuilder.String())
	if inputStr == "" {
		return fmt.Errorf("empty input - pipe issue JSON from 'gh issue list --json ...'")
	}

	if err := json.Unmarshal([]byte(inputStr), &issues); err != nil {
		return fmt.Errorf("failed to parse JSON input: %w\nExpected JSON array from 'gh issue list --json ...'", err)
	}

	// Build IssueRef list for targeted query (optimization)
	// Parse repo info from URLs in the input
	var items []api.ProjectItem
	refs := buildIssueRefsFromInput(issues, cfg)

	if len(refs) == 0 {
		return fmt.Errorf("no repositories configured — cannot build issue refs for targeted query\nConfigure repositories in .gh-pmu.json or provide issues with URL fields")
	}

	// Use targeted query - fetch only the specific issues we need
	items, err = client.GetProjectItemsByIssues(project.ID, refs)
	if err != nil {
		// Fall back to full fetch if targeted query fails
		items, err = client.GetProjectItems(project.ID, nil)
		if err != nil {
			return fmt.Errorf("failed to get project items: %w", err)
		}
	}

	// Build a map of issue number -> project item for quick lookup
	itemsByNumber := make(map[int]api.ProjectItem)
	for _, item := range items {
		if item.Issue != nil {
			itemsByNumber[item.Issue.Number] = item
		}
	}

	// Filter issues based on project field values
	var filtered []FilterInput
	for _, issue := range issues {
		item, exists := itemsByNumber[issue.Number]
		if !exists {
			// Issue not in project, skip
			continue
		}

		// Apply status filter
		if opts.status != "" {
			targetStatus := cfg.ResolveFieldValue("status", opts.status)
			if !hasFieldValue(item, "Status", targetStatus) {
				continue
			}
		}

		// Apply priority filter
		if opts.priority != "" {
			targetPriority := cfg.ResolveFieldValue("priority", opts.priority)
			if !hasFieldValue(item, "Priority", targetPriority) {
				continue
			}
		}

		// Apply assignee filter (from input JSON, not project)
		if opts.assignee != "" {
			if !hasAssignee(issue, opts.assignee) {
				continue
			}
		}

		// Apply label filter (from input JSON, not project)
		if opts.label != "" {
			if !hasLabel(issue, opts.label) {
				continue
			}
		}

		filtered = append(filtered, issue)
	}

	// Output results
	if opts.json {
		return outputFilterJSON(cmd, filtered)
	}
	return outputFilterTable(cmd, filtered)
}

// hasFieldValue checks if a project item has a specific field value
func hasFieldValue(item api.ProjectItem, fieldName, value string) bool {
	for _, fv := range item.FieldValues {
		if strings.EqualFold(fv.Field, fieldName) && strings.EqualFold(fv.Value, value) {
			return true
		}
	}
	return false
}

// hasAssignee checks if an issue has a specific assignee
func hasAssignee(issue FilterInput, assignee string) bool {
	for _, a := range issue.Assignees {
		if strings.EqualFold(a.Login, assignee) {
			return true
		}
	}
	return false
}

// hasLabel checks if an issue has a specific label
func hasLabel(issue FilterInput, label string) bool {
	for _, l := range issue.Labels {
		if strings.EqualFold(l.Name, label) {
			return true
		}
	}
	return false
}

// outputFilterJSON outputs filtered issues as JSON
func outputFilterJSON(cmd *cobra.Command, issues []FilterInput) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(issues)
}

// outputFilterTable outputs filtered issues as a table
func outputFilterTable(cmd *cobra.Command, issues []FilterInput) error {
	if len(issues) == 0 {
		cmd.Println("No matching issues found")
		return nil
	}

	// Simple table output
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "NUMBER\tTITLE\tSTATE")
	for _, issue := range issues {
		title := issue.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		fmt.Fprintf(w, "#%d\t%s\t%s\n", issue.Number, title, issue.State)
	}
	return nil
}

// buildIssueRefsFromInput builds IssueRef slice from input issues.
// Attempts to parse repo from URL fields, falls back to config repositories.
func buildIssueRefsFromInput(issues []FilterInput, cfg *config.Config) []api.IssueRef {
	var refs []api.IssueRef

	for _, issue := range issues {
		if issue.Number == 0 {
			continue
		}

		// Try to parse repo from URL (format: https://github.com/owner/repo/issues/123)
		owner, repo := parseRepoFromURL(issue.URL)

		// Fall back to first configured repository if URL parsing fails
		if owner == "" || repo == "" {
			if len(cfg.Repositories) > 0 {
				parts := strings.SplitN(cfg.Repositories[0], "/", 2)
				if len(parts) == 2 {
					owner, repo = parts[0], parts[1]
				}
			}
		}

		if owner != "" && repo != "" {
			refs = append(refs, api.IssueRef{
				Owner:  owner,
				Repo:   repo,
				Number: issue.Number,
			})
		}
	}

	return refs
}

// parseRepoFromURL extracts owner and repo from a GitHub issue URL
// URL format: https://github.com/owner/repo/issues/123
func parseRepoFromURL(url string) (owner, repo string) {
	if url == "" {
		return "", ""
	}

	// Remove protocol prefix
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Expected format: github.com/owner/repo/issues/123
	parts := strings.Split(url, "/")
	if len(parts) >= 4 && parts[0] == "github.com" {
		return parts[1], parts[2]
	}

	return "", ""
}
