package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/rubrical-works/gh-pmu/internal/ui"
	"github.com/spf13/cobra"
)

// viewClient defines the interface for API methods used by view functions.
// This allows for easier testing with mock implementations.
type viewClient interface {
	GetIssue(owner, repo string, number int) (*api.Issue, error)
	GetIssueWithProjectFields(owner, repo string, number int) (*api.Issue, []api.FieldValue, error)
	GetSubIssues(owner, repo string, number int) ([]api.SubIssue, error)
	GetParentIssue(owner, repo string, number int) (*api.Issue, error)
	GetIssueComments(owner, repo string, number int) ([]api.Comment, error)
	// Batch methods for multi-issue mode
	GetIssuesWithProjectFieldsBatch(owner, repo string, numbers []int) (map[int]*api.Issue, map[int][]api.FieldValue, map[int]error, error)
	GetSubIssuesBatch(owner, repo string, numbers []int) (map[int][]api.SubIssue, error)
	GetParentIssueBatch(owner, repo string, numbers []int) (map[int]*api.Issue, error)
}

// viewIssueRef represents a parsed issue reference for the view command
type viewIssueRef struct {
	owner  string
	repo   string
	number int
}

// viewResult holds all data for a single issue in multi-issue mode
type viewResult struct {
	number      int
	issue       *api.Issue
	fieldValues []api.FieldValue
	subIssues   []api.SubIssue
	parentIssue *api.Issue
	comments    []api.Comment
}

type viewOptions struct {
	jsonFields string // Empty = not set, "all" = all fields, "field1,field2" = specific fields
	jq         string
	template   string // Go template (not supported - returns helpful error)
	web        bool
	comments   bool
	repo       string
	bodyFile   bool
	bodyStdout bool
}

// viewAvailableFields lists all available JSON fields for the view command
var viewAvailableFields = []string{
	"assignees",
	"author",
	"body",
	"branch",
	"comments",
	"fieldValues",
	"labels",
	"milestone",
	"number",
	"parentIssue",
	"priority",
	"state",
	"status",
	"subIssues",
	"subProgress",
	"title",
	"url",
}

func newViewCommand() *cobra.Command {
	opts := &viewOptions{}

	cmd := &cobra.Command{
		Use:   "view <issue-number> [issue-number...]",
		Short: "View one or more issues with project metadata",
		Long: `View one or more issues with project field values.

Displays issue details including title, body, state, labels, assignees,
and all project-specific fields like Status and Priority.

When viewing multiple issues, results are shown sequentially with separators.
JSON output (--json) returns an array for multiple issues, a single object for one.

Also shows sub-issues if any exist, and parent issue if this is a sub-issue.

Note: --body-file, --body-stdout, and --web are only supported for single issues.`,
		Args: func(cmd *cobra.Command, args []string) error {
			// --json-fields doesn't require an issue number
			if listFields, _ := cmd.Flags().GetBool("json-fields"); listFields {
				return nil
			}
			if len(args) < 1 {
				return fmt.Errorf("requires at least 1 arg(s), received %d", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runView(cmd, args, opts)
		},
	}

	cmd.Flags().StringVar(&opts.jsonFields, "json", "", "Output JSON with specified fields (comma-separated). Use --json-fields to list available fields")
	cmd.Flags().Bool("json-fields", false, "List available JSON fields")
	cmd.Flags().StringVarP(&opts.jq, "jq", "q", "", "Filter JSON output using a jq expression")
	cmd.Flags().StringVarP(&opts.template, "template", "t", "", "Format output using a Go template (not supported; see error for alternatives)")
	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open issue in browser")
	cmd.Flags().BoolVarP(&opts.comments, "comments", "c", false, "Show issue comments")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository for the issue (owner/repo format)")
	cmd.Flags().BoolVarP(&opts.bodyFile, "body-file", "b", false, "Write issue body to tmp/issue-{number}.md")
	cmd.Flags().BoolVar(&opts.bodyStdout, "body-stdout", false, "Output issue body to stdout (raw markdown)")

	return cmd
}

func runView(cmd *cobra.Command, args []string, opts *viewOptions) error {
	// Handle --json-fields: list available fields (no issue number needed)
	if listFields, _ := cmd.Flags().GetBool("json-fields"); listFields {
		return listAvailableFields(cmd, viewAvailableFields)
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

	// Determine default repository (--repo flag takes precedence over config)
	defaultOwner, defaultRepo := "", ""
	if opts.repo != "" {
		parts := strings.Split(opts.repo, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid --repo format: expected owner/repo, got %s", opts.repo)
		}
		defaultOwner, defaultRepo = parts[0], parts[1]
	} else if len(cfg.Repositories) > 0 {
		parts := strings.Split(cfg.Repositories[0], "/")
		if len(parts) == 2 {
			defaultOwner, defaultRepo = parts[0], parts[1]
		}
	}

	// Parse all issue references
	var refs []viewIssueRef
	var parseErrors []string

	for _, arg := range args {
		owner, repo, number, err := parseIssueReference(arg)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("%s: %v", arg, err))
			continue
		}
		if owner == "" || repo == "" {
			if defaultOwner == "" || defaultRepo == "" {
				return fmt.Errorf("no repository specified and none configured (use --repo or configure in .gh-pmu.yml)")
			}
			owner = defaultOwner
			repo = defaultRepo
		}
		refs = append(refs, viewIssueRef{owner, repo, number})
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	// Single-issue path (backward compatible, unchanged behavior)
	if len(refs) == 1 && len(parseErrors) == 0 {
		return runViewWithDeps(cmd, opts, client, refs[0].owner, refs[0].repo, refs[0].number)
	}

	// Multi-issue path
	return runViewMulti(cmd, opts, client, refs, parseErrors)
}

// runViewMulti handles viewing multiple issues with batch API calls
func runViewMulti(cmd *cobra.Command, opts *viewOptions, client viewClient, refs []viewIssueRef, parseErrors []string) error {
	// Validate single-issue-only flags
	if opts.bodyFile {
		return fmt.Errorf("--body-file is only supported for single issue")
	}
	if opts.bodyStdout {
		return fmt.Errorf("--body-stdout is only supported for single issue")
	}
	if opts.web {
		return fmt.Errorf("--web is only supported for single issue")
	}
	if opts.template != "" {
		return fmt.Errorf("--template is not supported")
	}
	if opts.jq != "" && opts.jsonFields == "" {
		return fmt.Errorf("--jq requires --json")
	}
	if opts.jsonFields != "" && opts.comments {
		return fmt.Errorf("cannot use --json with --comments; use --json comments instead")
	}

	// Report parse errors to stderr
	for _, pe := range parseErrors {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", pe)
	}

	if len(refs) == 0 {
		return fmt.Errorf("no valid issue numbers provided")
	}

	// Group by repo (all issues typically in same repo)
	type repoKey struct{ owner, repo string }
	grouped := make(map[repoKey][]int)
	refOrder := make([]int, 0, len(refs))
	for _, ref := range refs {
		key := repoKey{ref.owner, ref.repo}
		grouped[key] = append(grouped[key], ref.number)
		refOrder = append(refOrder, ref.number)
	}

	// Fetch all data with batch calls
	allIssues := make(map[int]*api.Issue)
	allFieldValues := make(map[int][]api.FieldValue)
	allSubIssues := make(map[int][]api.SubIssue)
	allParentIssues := make(map[int]*api.Issue)
	allErrors := make(map[int]error)

	for key, numbers := range grouped {
		// Call 1: Issues + project fields
		issues, fieldValues, issueErrors, err := client.GetIssuesWithProjectFieldsBatch(key.owner, key.repo, numbers)
		if err != nil {
			return fmt.Errorf("failed to fetch issues: %w", err)
		}
		for num, issue := range issues {
			allIssues[num] = issue
		}
		for num, fvs := range fieldValues {
			allFieldValues[num] = fvs
		}
		for num, e := range issueErrors {
			allErrors[num] = e
		}

		// Collect successfully fetched issue numbers for sub-issue/parent batch
		var validNumbers []int
		for _, num := range numbers {
			if _, ok := allIssues[num]; ok {
				validNumbers = append(validNumbers, num)
			}
		}

		// Calls 2-3: Sub-issues + parent issues (parallel)
		var subIssuesMap map[int][]api.SubIssue
		var parentIssuesMap map[int]*api.Issue
		var wg sync.WaitGroup

		wg.Add(2)
		go func() {
			defer wg.Done()
			subIssuesMap, _ = client.GetSubIssuesBatch(key.owner, key.repo, validNumbers)
		}()
		go func() {
			defer wg.Done()
			parentIssuesMap, _ = client.GetParentIssueBatch(key.owner, key.repo, validNumbers)
		}()
		wg.Wait()

		for num, subs := range subIssuesMap {
			allSubIssues[num] = subs
		}
		for num, parent := range parentIssuesMap {
			allParentIssues[num] = parent
		}
	}

	// Fetch comments per-issue if requested (sequential, opt-in)
	allComments := make(map[int][]api.Comment)
	if opts.comments {
		for _, ref := range refs {
			if _, ok := allIssues[ref.number]; ok {
				comments, _ := client.GetIssueComments(ref.owner, ref.repo, ref.number)
				allComments[ref.number] = comments
			}
		}
	}

	// Build results in argument order
	var results []viewResult
	for _, num := range refOrder {
		if e, ok := allErrors[num]; ok && e != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get issue #%d: %v\n", num, e)
			continue
		}
		issue, ok := allIssues[num]
		if !ok {
			fmt.Fprintf(os.Stderr, "Warning: issue #%d not found\n", num)
			continue
		}
		results = append(results, viewResult{
			number:      num,
			issue:       issue,
			fieldValues: allFieldValues[num],
			subIssues:   allSubIssues[num],
			parentIssue: allParentIssues[num],
			comments:    allComments[num],
		})
	}

	if len(results) == 0 {
		return fmt.Errorf("all issues failed to load")
	}

	// Output
	if opts.jsonFields != "" {
		return outputViewMultiJSON(cmd, opts, results)
	}
	return outputViewMultiTable(cmd, results)
}

// outputViewMultiJSON outputs multiple issues as a JSON array
func outputViewMultiJSON(cmd *cobra.Command, opts *viewOptions, results []viewResult) error {
	requestedFields := parseJSONFields(opts.jsonFields, viewAvailableFields)

	var jsonArray []map[string]interface{}
	for _, r := range results {
		fullOutput := buildViewJSONOutput(r.issue, r.fieldValues, r.subIssues, r.parentIssue, r.comments)
		filteredOutput := filterViewJSONFields(fullOutput, requestedFields)
		jsonArray = append(jsonArray, filteredOutput)
	}

	jsonBytes, err := json.MarshalIndent(jsonArray, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	if opts.jq != "" {
		return applyJQFilter(jsonBytes, opts.jq)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintln(w, string(jsonBytes))
	return nil
}

// outputViewMultiTable outputs multiple issues sequentially with separators
func outputViewMultiTable(cmd *cobra.Command, results []viewResult) error {
	w := cmd.OutOrStdout()
	for i, r := range results {
		if i > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, strings.Repeat("\u2550", 60))
			fmt.Fprintln(w)
		}
		if err := outputViewTable(cmd, r.issue, r.fieldValues, r.subIssues, r.parentIssue, r.comments); err != nil {
			return err
		}
	}
	return nil
}

// runViewWithDeps is the testable implementation of runView
func runViewWithDeps(cmd *cobra.Command, opts *viewOptions, client viewClient, owner, repo string, number int) error {
	// Validate flag mutual exclusivity
	if opts.jsonFields != "" {
		if opts.bodyFile {
			return fmt.Errorf("cannot use --json with --body-file")
		}
		if opts.bodyStdout {
			return fmt.Errorf("cannot use --json with --body-stdout")
		}
		if opts.web {
			return fmt.Errorf("cannot use --json with --web")
		}
		if opts.comments {
			return fmt.Errorf("cannot use --json with --comments; use --json comments instead")
		}
	}

	// Validate --jq requires --json
	if opts.jq != "" && opts.jsonFields == "" {
		return fmt.Errorf("--jq requires --json")
	}

	// --template is not supported; provide helpful error with alternatives
	if opts.template != "" {
		return fmt.Errorf("--template is not supported; use 'gh issue view %d --template' for standard fields, or use --jq for project fields", number)
	}

	// For --web flag, only need basic issue info
	if opts.web {
		issue, err := client.GetIssue(owner, repo, number)
		if err != nil {
			return fmt.Errorf("failed to get issue: %w", err)
		}
		return ui.OpenInBrowser(issue.URL)
	}

	// Fetch issue with project field values in a single query (optimized)
	issue, fieldValues, err := client.GetIssueWithProjectFields(owner, repo, number)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	// Handle --body-file flag: write body to tmp/issue-{number}.md
	if opts.bodyFile {
		return writeBodyToFile(cmd.OutOrStdout(), issue.Number, issue.Body)
	}

	// Handle --body-stdout flag: output body directly to stdout
	if opts.bodyStdout {
		fmt.Fprint(cmd.OutOrStdout(), issue.Body)
		return nil
	}

	// Fetch sub-issues and parent issue in parallel
	var subIssues []api.SubIssue
	var parentIssue *api.Issue
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		subIssues, _ = client.GetSubIssues(owner, repo, number)
	}()
	go func() {
		defer wg.Done()
		parentIssue, _ = client.GetParentIssue(owner, repo, number)
	}()
	wg.Wait()

	// Fetch comments if requested (not parallelized - only when needed)
	var comments []api.Comment
	if opts.comments {
		comments, _ = client.GetIssueComments(owner, repo, number)
	}

	// Output
	if opts.jsonFields != "" {
		return outputViewJSON(cmd, opts, issue, fieldValues, subIssues, parentIssue, comments)
	}

	return outputViewTable(cmd, issue, fieldValues, subIssues, parentIssue, comments)
}

// ViewJSONOutput represents the JSON output for view command
type ViewJSONOutput struct {
	Number      int               `json:"number"`
	Title       string            `json:"title"`
	State       string            `json:"state"`
	Body        string            `json:"body"`
	URL         string            `json:"url"`
	Author      string            `json:"author"`
	Assignees   []string          `json:"assignees"`
	Labels      []string          `json:"labels"`
	Milestone   string            `json:"milestone,omitempty"`
	FieldValues map[string]string `json:"fieldValues"`
	SubIssues   []SubIssueJSON    `json:"subIssues,omitempty"`
	SubProgress *SubProgressJSON  `json:"subProgress,omitempty"`
	ParentIssue *ParentIssueJSON  `json:"parentIssue,omitempty"`
	Comments    []CommentJSON     `json:"comments,omitempty"`
}

// CommentJSON represents a comment in JSON output
type CommentJSON struct {
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

// SubProgressJSON represents sub-issue progress in JSON output
type SubProgressJSON struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	Percentage int `json:"percentage"`
}

// SubIssueJSON represents a sub-issue in JSON output
type SubIssueJSON struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

// ParentIssueJSON represents the parent issue in JSON output
type ParentIssueJSON struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

func outputViewJSON(cmd *cobra.Command, opts *viewOptions, issue *api.Issue, fieldValues []api.FieldValue, subIssues []api.SubIssue, parentIssue *api.Issue, comments []api.Comment) error {
	// Build full output first
	fullOutput := buildViewJSONOutput(issue, fieldValues, subIssues, parentIssue, comments)

	// Parse requested fields
	requestedFields := parseJSONFields(opts.jsonFields, viewAvailableFields)

	// Filter output to only requested fields
	filteredOutput := filterViewJSONFields(fullOutput, requestedFields)

	// Encode to JSON
	jsonBytes, err := json.MarshalIndent(filteredOutput, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	// Apply jq filter if specified
	if opts.jq != "" {
		return applyJQFilter(jsonBytes, opts.jq)
	}

	// Output JSON
	fmt.Fprintln(cmd.OutOrStdout(), string(jsonBytes))
	return nil
}

// buildViewJSONOutput constructs the full JSON output structure
func buildViewJSONOutput(issue *api.Issue, fieldValues []api.FieldValue, subIssues []api.SubIssue, parentIssue *api.Issue, comments []api.Comment) ViewJSONOutput {
	output := ViewJSONOutput{
		Number:      issue.Number,
		Title:       issue.Title,
		State:       issue.State,
		Body:        issue.Body,
		URL:         issue.URL,
		Author:      issue.Author.Login,
		Assignees:   make([]string, 0),
		Labels:      make([]string, 0),
		FieldValues: make(map[string]string),
	}

	for _, a := range issue.Assignees {
		output.Assignees = append(output.Assignees, a.Login)
	}

	for _, l := range issue.Labels {
		output.Labels = append(output.Labels, l.Name)
	}

	if issue.Milestone != nil {
		output.Milestone = issue.Milestone.Title
	}

	for _, fv := range fieldValues {
		output.FieldValues[fv.Field] = fv.Value
	}

	if len(subIssues) > 0 {
		output.SubIssues = make([]SubIssueJSON, 0, len(subIssues))
		closedCount := 0
		for _, sub := range subIssues {
			output.SubIssues = append(output.SubIssues, SubIssueJSON{
				Number: sub.Number,
				Title:  sub.Title,
				State:  sub.State,
				URL:    sub.URL,
			})
			if sub.State == "CLOSED" {
				closedCount++
			}
		}

		// Add progress info
		total := len(subIssues)
		percentage := 0
		if total > 0 {
			percentage = (closedCount * 100) / total
		}
		output.SubProgress = &SubProgressJSON{
			Total:      total,
			Completed:  closedCount,
			Percentage: percentage,
		}
	}

	if parentIssue != nil {
		output.ParentIssue = &ParentIssueJSON{
			Number: parentIssue.Number,
			Title:  parentIssue.Title,
			URL:    parentIssue.URL,
		}
	}

	if len(comments) > 0 {
		output.Comments = make([]CommentJSON, 0, len(comments))
		for _, c := range comments {
			output.Comments = append(output.Comments, CommentJSON{
				Author:    c.Author,
				Body:      c.Body,
				CreatedAt: c.CreatedAt,
			})
		}
	}

	return output
}

// filterViewJSONFields returns a map with only the requested fields
func filterViewJSONFields(output ViewJSONOutput, fields []string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, field := range fields {
		switch field {
		case "number":
			result["number"] = output.Number
		case "title":
			result["title"] = output.Title
		case "state":
			result["state"] = output.State
		case "body":
			result["body"] = output.Body
		case "url":
			result["url"] = output.URL
		case "author":
			result["author"] = output.Author
		case "assignees":
			result["assignees"] = output.Assignees
		case "labels":
			result["labels"] = output.Labels
		case "milestone":
			if output.Milestone != "" {
				result["milestone"] = output.Milestone
			}
		case "fieldValues":
			result["fieldValues"] = output.FieldValues
		case "status":
			// Extract status from fieldValues (project field)
			if val, ok := output.FieldValues["Status"]; ok && val != "" {
				result["status"] = val
			} else {
				result["status"] = nil
			}
		case "priority":
			// Extract priority from fieldValues (project field)
			if val, ok := output.FieldValues["Priority"]; ok && val != "" {
				result["priority"] = val
			} else {
				result["priority"] = nil
			}
		case "branch":
			// Extract branch from fieldValues (project field)
			if val, ok := output.FieldValues["Branch"]; ok && val != "" {
				result["branch"] = val
			} else {
				result["branch"] = nil
			}
		case "subIssues":
			if output.SubIssues != nil {
				result["subIssues"] = output.SubIssues
			}
		case "subProgress":
			if output.SubProgress != nil {
				result["subProgress"] = output.SubProgress
			}
		case "parentIssue":
			if output.ParentIssue != nil {
				result["parentIssue"] = output.ParentIssue
			}
		case "comments":
			if output.Comments != nil {
				result["comments"] = output.Comments
			}
		}
	}

	return result
}

func outputViewTable(cmd *cobra.Command, issue *api.Issue, fieldValues []api.FieldValue, subIssues []api.SubIssue, parentIssue *api.Issue, comments []api.Comment) error {
	w := cmd.OutOrStdout()

	// Title and state
	fmt.Fprintf(w, "%s #%d\n", issue.Title, issue.Number)
	fmt.Fprintf(w, "State: %s\n", issue.State)
	fmt.Fprintf(w, "URL: %s\n", issue.URL)
	fmt.Fprintln(w)

	// Author
	fmt.Fprintf(w, "Author: @%s\n", issue.Author.Login)

	// Assignees
	if len(issue.Assignees) > 0 {
		var assignees []string
		for _, a := range issue.Assignees {
			assignees = append(assignees, "@"+a.Login)
		}
		fmt.Fprintf(w, "Assignees: %s\n", strings.Join(assignees, ", "))
	}

	// Labels
	if len(issue.Labels) > 0 {
		var labels []string
		for _, l := range issue.Labels {
			labels = append(labels, l.Name)
		}
		fmt.Fprintf(w, "Labels: %s\n", strings.Join(labels, ", "))
	}

	// Milestone
	if issue.Milestone != nil {
		fmt.Fprintf(w, "Milestone: %s\n", issue.Milestone.Title)
	}

	// Project field values
	if len(fieldValues) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Project Fields:")
		for _, fv := range fieldValues {
			fmt.Fprintf(w, "  %s: %s\n", fv.Field, fv.Value)
		}
	}

	// Parent issue
	if parentIssue != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Parent Issue: #%d - %s\n", parentIssue.Number, parentIssue.Title)
	}

	// Sub-issues with progress bar
	if len(subIssues) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Sub-Issues:")
		closedCount := 0
		for _, sub := range subIssues {
			state := "[ ]"
			if sub.State == "CLOSED" {
				state = "[x]"
				closedCount++
			}
			// Show repo info if cross-repo
			if sub.Repository.Owner != "" && sub.Repository.Name != "" {
				parentRepo := issue.Repository.Owner + "/" + issue.Repository.Name
				subRepo := sub.Repository.Owner + "/" + sub.Repository.Name
				if subRepo != parentRepo {
					fmt.Fprintf(w, "  %s %s#%d - %s\n", state, subRepo, sub.Number, sub.Title)
					continue
				}
			}
			fmt.Fprintf(w, "  %s #%d - %s\n", state, sub.Number, sub.Title)
		}

		// Progress bar and percentage
		total := len(subIssues)
		percentage := 0
		if total > 0 {
			percentage = (closedCount * 100) / total
		}
		progressBar := renderProgressBar(closedCount, total, 20)
		fmt.Fprintf(w, "\n%s %d of %d sub-issues complete (%d%%)\n", progressBar, closedCount, total, percentage)
	}

	// Body
	if issue.Body != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "---")
		fmt.Fprintln(w, issue.Body)
	}

	// Comments
	if len(comments) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Comments (%d):\n", len(comments))
		for _, c := range comments {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "@%s commented on %s:\n", c.Author, c.CreatedAt)
			fmt.Fprintln(w, c.Body)
		}
	}

	return nil
}

// renderProgressBar creates a visual progress bar
// Example: [████████░░░░░░░░░░░░] for 40% complete
func renderProgressBar(completed, total, width int) string {
	if total == 0 {
		return "[" + strings.Repeat("░", width) + "]"
	}

	filled := (completed * width) / total
	if filled > width {
		filled = width
	}

	empty := width - filled
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}

// writeBodyToFile writes the issue body to tmp/issue-{number}.md
// Creates the tmp directory if it doesn't exist
func writeBodyToFile(w io.Writer, number int, body string) error {
	// Create tmp directory if it doesn't exist
	tmpDir := "tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create tmp directory: %w", err)
	}

	// Write body to file
	filename := fmt.Sprintf("issue-%d.md", number)
	filePath := filepath.Join(tmpDir, filename)

	if err := os.WriteFile(filePath, []byte(body), 0644); err != nil {
		return fmt.Errorf("failed to write body file: %w", err)
	}

	fmt.Fprintln(w, filePath)
	return nil
}

// parseIssueNumber parses a string into an issue number
// Accepts formats: "123" or "#123"
func parseIssueNumber(s string) (int, error) {
	// Strip leading # if present
	s = strings.TrimPrefix(s, "#")

	num, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid issue number: %s", s)
	}

	if num <= 0 {
		return 0, fmt.Errorf("issue number must be positive: %d", num)
	}

	return num, nil
}

// parseIssueReference parses an issue reference string
// Accepts formats: "123", "#123", "owner/repo#123", or full GitHub issue URL
// Returns owner, repo, number (owner/repo may be empty if not specified)
func parseIssueReference(s string) (owner, repo string, number int, err error) {
	// Check for GitHub URL format
	// Formats: https://github.com/owner/repo/issues/123
	//          https://github.com/owner/repo/issues/123#issuecomment-...
	if strings.HasPrefix(s, "https://github.com/") || strings.HasPrefix(s, "http://github.com/") {
		owner, repo, number, err = parseIssueURL(s)
		if err != nil {
			return "", "", 0, err
		}
		return owner, repo, number, nil
	}

	// Check for owner/repo#number format
	if idx := strings.Index(s, "#"); idx > 0 {
		// Has # with something before it - could be owner/repo#number
		repoRef := s[:idx]
		numStr := s[idx+1:]

		if slashIdx := strings.Index(repoRef, "/"); slashIdx > 0 {
			owner = repoRef[:slashIdx]
			repo = repoRef[slashIdx+1:]

			number, err = parseIssueNumber(numStr)
			if err != nil {
				return "", "", 0, err
			}
			return owner, repo, number, nil
		}
	}

	// Try parsing as simple number or #number
	number, err = parseIssueNumber(s)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid issue reference: %s", s)
	}

	return "", "", number, nil
}

// parseIssueURL parses a GitHub issue URL and extracts owner, repo, and number
// Supports formats:
//   - https://github.com/owner/repo/issues/123
//   - https://github.com/owner/repo/issues/123#issuecomment-...
func parseIssueURL(url string) (owner, repo string, number int, err error) {
	// Remove protocol prefix
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Remove github.com prefix
	if !strings.HasPrefix(url, "github.com/") {
		return "", "", 0, fmt.Errorf("invalid GitHub URL: not a github.com URL")
	}
	url = strings.TrimPrefix(url, "github.com/")

	// Split path parts: owner/repo/issues/number[#anchor]
	parts := strings.Split(url, "/")
	if len(parts) < 4 {
		return "", "", 0, fmt.Errorf("invalid GitHub issue URL format")
	}

	owner = parts[0]
	repo = parts[1]

	if parts[2] != "issues" {
		return "", "", 0, fmt.Errorf("URL is not an issue URL (expected /issues/)")
	}

	// Parse issue number (may have anchor suffix like #issuecomment-123)
	numStr := parts[3]
	if anchorIdx := strings.Index(numStr, "#"); anchorIdx > 0 {
		numStr = numStr[:anchorIdx]
	}

	number, err = parseIssueNumber(numStr)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid issue number in URL: %s", numStr)
	}

	return owner, repo, number, nil
}

// listAvailableFields outputs the list of available JSON fields
func listAvailableFields(cmd *cobra.Command, fields []string) error {
	sorted := make([]string, len(fields))
	copy(sorted, fields)
	sort.Strings(sorted)

	fmt.Fprintln(cmd.OutOrStdout(), "Specify one or more comma-separated fields for `--json`:")
	for _, field := range sorted {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", field)
	}
	return nil
}

// parseJSONFields parses comma-separated field names and returns a slice
// If input is empty or contains only whitespace, returns all available fields
func parseJSONFields(input string, available []string) []string {
	input = strings.TrimSpace(input)
	if input == "" || input == "_list_" {
		return available
	}

	parts := strings.Split(input, ",")
	var result []string
	for _, p := range parts {
		field := strings.TrimSpace(p)
		if field != "" {
			result = append(result, field)
		}
	}

	if len(result) == 0 {
		return available
	}
	return result
}

// applyJQFilter applies a jq expression to JSON input and outputs the result
func applyJQFilter(jsonBytes []byte, jqExpr string) error {
	cmd := exec.Command("jq", jqExpr)
	cmd.Stdin = strings.NewReader(string(jsonBytes))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
