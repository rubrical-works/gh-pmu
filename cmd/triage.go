package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/spf13/cobra"
)

type triageOptions struct {
	dryRun      bool
	interactive bool
	json        bool
	list        bool
	repo        string
	query       string
	apply       string
}

// triageClient defines the interface for API methods used by triage functions.
// This allows for easier testing with mock implementations.
type triageClient interface {
	GetRepositoryIssues(owner, repo, state string) ([]api.Issue, error)
	SearchRepositoryIssues(owner, repo string, filters api.SearchFilters, limit int) ([]api.Issue, error)
	GetProject(owner string, number int) (*api.Project, error)
	AddIssueToProject(projectID, issueID string) (string, error)
	AddLabelToIssue(owner, repo, issueID, labelName string) error
	SetProjectItemField(projectID, itemID, fieldName, value string) error
}

func newTriageCommand() *cobra.Command {
	opts := &triageOptions{}

	cmd := &cobra.Command{
		Use:   "triage [config-name]",
		Short: "Bulk process issues matching triage rules",
		Long: `Run triage rules to bulk update issues matching certain criteria.

Triage configurations are defined in .gh-pmu.yml under the 'triage' key.
Each triage config has a query to match issues and rules to apply.`,
		Aliases: []string{"tr"},
		Example: `  # List available triage configs
  gh pmu triage --list

  # Preview what a triage rule would do
  gh pmu triage tracked --dry-run

  # Run a triage rule
  gh pmu triage tracked

  # Run interactively (prompt for each issue)
  gh pmu triage tracked --interactive

  # Target a specific repository
  gh pmu triage tracked --repo owner/repo

  # Ad-hoc query without config file
  gh pmu triage --query "is:open -label:triaged" --apply status:backlog

  # Ad-hoc bulk update with multiple fields
  gh pmu triage --query "label:bug" --apply status:in_progress,priority:p1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTriage(cmd, args, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show what would be changed without making changes")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "Prompt before processing each issue")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")
	cmd.Flags().BoolVarP(&opts.list, "list", "l", false, "List available triage configurations")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Target specific repository (owner/repo format)")
	cmd.Flags().StringVarP(&opts.query, "query", "q", "", "Ad-hoc query (e.g., \"is:open -label:triaged\")")
	cmd.Flags().StringVarP(&opts.apply, "apply", "a", "", "Ad-hoc field updates (e.g., \"status:backlog,priority:p1\")")

	return cmd
}

func runTriage(cmd *cobra.Command, args []string, opts *triageOptions) error {
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

	return runTriageWithDeps(cmd, args, opts, cfg, client, os.Stdin)
}

// runTriageWithDeps is the testable implementation of runTriage
func runTriageWithDeps(cmd *cobra.Command, args []string, opts *triageOptions, cfg *config.Config, client triageClient, stdin *os.File) error {
	// List mode
	if opts.list {
		return listTriageConfigs(cmd, cfg, opts.json)
	}

	// Ad-hoc mode with --query flag
	if opts.query != "" {
		return runAdHocTriage(cmd, opts, cfg, client, stdin)
	}

	// Require config name
	if len(args) == 0 {
		return fmt.Errorf("triage config name is required\nUse --list to see available configs, or use --query for ad-hoc triage")
	}

	configName := args[0]
	triageCfg, ok := cfg.Triage[configName]
	if !ok {
		return fmt.Errorf("triage config %q not found\nUse --list to see available configs", configName)
	}

	// Get project
	project, err := client.GetProject(cfg.Project.Owner, cfg.Project.Number)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Search for issues matching the query
	matchingIssues, err := searchIssuesForTriage(client, cfg, triageCfg.Query, opts.repo)
	if err != nil {
		return fmt.Errorf("failed to search issues: %w", err)
	}

	if len(matchingIssues) == 0 {
		if opts.json {
			return outputTriageJSON(cmd, nil, "no-matches", configName)
		}
		cmd.Printf("No issues match the triage query for %q\n", configName)
		return nil
	}

	// Dry run - just show what would be changed
	if opts.dryRun {
		if opts.json {
			return outputTriageJSON(cmd, matchingIssues, "dry-run", configName)
		}
		cmd.Printf("Would process %d issue(s) with triage config %q:\n\n", len(matchingIssues), configName)
		_ = outputTriageTable(cmd, matchingIssues)
		cmd.Println()
		describeTriageActions(cmd, cfg, &triageCfg)
		return nil
	}

	// Process issues
	var processed, skipped, failed int
	reader := bufio.NewReader(stdin)

	for _, issue := range matchingIssues {
		// Interactive mode - prompt for each issue
		if opts.interactive {
			cmd.Printf("\nProcess #%d: %s? [y/n/q] ", issue.Number, issue.Title)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "q" {
				cmd.Println("Aborted.")
				break
			}
			if response != "y" && response != "yes" {
				skipped++
				continue
			}
		}

		// Apply triage rules
		err := applyTriageRules(client, cfg, project, &issue, &triageCfg)
		if err != nil {
			cmd.PrintErrf("Failed to process #%d: %v\n", issue.Number, err)
			failed++
			continue
		}

		processed++
		if !opts.interactive {
			cmd.Printf("Processed #%d: %s\n", issue.Number, issue.Title)
		}
	}

	// Summary
	if opts.json {
		return outputTriageJSON(cmd, matchingIssues, "completed", configName)
	}

	cmd.Printf("\nTriage complete: %d processed", processed)
	if skipped > 0 {
		cmd.Printf(", %d skipped", skipped)
	}
	if failed > 0 {
		cmd.Printf(", %d failed", failed)
	}
	cmd.Println()

	return nil
}

func listTriageConfigs(cmd *cobra.Command, cfg *config.Config, jsonOutput bool) error {
	if len(cfg.Triage) == 0 {
		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(map[string]interface{}{"configs": []interface{}{}})
		}
		cmd.Println("No triage configurations defined in .gh-pmu.yml")
		return nil
	}

	if jsonOutput {
		type triageConfigJSON struct {
			Name        string            `json:"name"`
			Query       string            `json:"query"`
			ApplyLabels []string          `json:"applyLabels,omitempty"`
			ApplyFields map[string]string `json:"applyFields,omitempty"`
		}

		configs := make([]triageConfigJSON, 0, len(cfg.Triage))
		for name, tc := range cfg.Triage {
			configs = append(configs, triageConfigJSON{
				Name:        name,
				Query:       tc.Query,
				ApplyLabels: tc.Apply.Labels,
				ApplyFields: tc.Apply.Fields,
			})
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]interface{}{"configs": configs})
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tQUERY\tACTIONS")

	for name, tc := range cfg.Triage {
		actions := describeActions(&tc)
		query := tc.Query
		if len(query) > 40 {
			query = query[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, query, actions)
	}

	return w.Flush()
}

func describeActions(tc *config.Triage) string {
	var actions []string

	if len(tc.Apply.Labels) > 0 {
		actions = append(actions, fmt.Sprintf("labels: %s", strings.Join(tc.Apply.Labels, ", ")))
	}

	for field, value := range tc.Apply.Fields {
		actions = append(actions, fmt.Sprintf("%s: %s", field, value))
	}

	if len(actions) == 0 {
		if tc.Interactive.Status || tc.Interactive.Estimate {
			return "interactive only"
		}
		return "none"
	}

	return strings.Join(actions, "; ")
}

func describeTriageActions(cmd *cobra.Command, cfg *config.Config, tc *config.Triage) {
	cmd.Println("Actions to apply:")

	if len(tc.Apply.Labels) > 0 {
		cmd.Printf("  • Add labels: %s\n", strings.Join(tc.Apply.Labels, ", "))
	}

	for field, value := range tc.Apply.Fields {
		resolved := cfg.ResolveFieldValue(field, value)
		cmd.Printf("  • Set %s: %s\n", field, resolved)
	}

	if tc.Interactive.Status {
		cmd.Println("  • Prompt for status (interactive)")
	}
	if tc.Interactive.Estimate {
		cmd.Println("  • Prompt for estimate (interactive)")
	}
}

// triageQueryFilters holds parsed query components for triage searches
type triageQueryFilters struct {
	state           string   // "open", "closed", or "all"
	labels          []string // positive label filters (label:X)
	negatedLabels   []string // negative label filters (-label:X)
	canUseSearchAPI bool     // true if all filters can be handled by Search API
}

// parseTriageQuery extracts label and state filters from a triage query string
func parseTriageQuery(query string) triageQueryFilters {
	filters := triageQueryFilters{
		state:           "open",
		labels:          []string{},
		negatedLabels:   []string{},
		canUseSearchAPI: true,
	}

	// Parse state
	if strings.Contains(query, "is:closed") {
		filters.state = "closed"
	} else if strings.Contains(query, "is:all") {
		filters.state = "all"
	}

	// Parse labels using simple tokenization
	// Handle both "label:name" and "-label:name"
	parts := strings.Fields(query)
	for _, part := range parts {
		if strings.HasPrefix(part, "-label:") {
			labelName := strings.TrimPrefix(part, "-label:")
			if labelName != "" {
				filters.negatedLabels = append(filters.negatedLabels, labelName)
			}
		} else if strings.HasPrefix(part, "label:") {
			labelName := strings.TrimPrefix(part, "label:")
			if labelName != "" {
				filters.labels = append(filters.labels, labelName)
			}
		}
		// Note: Unknown query components (not is:open/closed/all, label:, repo:, or state:)
		// are passed through to Search API which may handle them or ignore them
	}

	return filters
}

func searchIssuesForTriage(client triageClient, cfg *config.Config, query string, targetRepo string) ([]api.Issue, error) {
	// Determine which repositories to search
	repos := cfg.Repositories
	if targetRepo != "" {
		// Validate format
		if !strings.Contains(targetRepo, "/") {
			return nil, fmt.Errorf("invalid repository format %q: expected owner/repo", targetRepo)
		}
		repos = []string{targetRepo}
	}

	// Parse query to extract filters
	queryFilters := parseTriageQuery(query)

	var allIssues []api.Issue

	for _, repoFullName := range repos {
		parts := strings.SplitN(repoFullName, "/", 2)
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

		// Determine if we can use Search API optimization
		// Use Search API when we have label filters (positive or negative)
		useSearchAPI := len(queryFilters.labels) > 0 || len(queryFilters.negatedLabels) > 0

		var issues []api.Issue
		var err error

		if useSearchAPI {
			// Build search filters for optimized query
			searchFilters := api.SearchFilters{
				State:  queryFilters.state,
				Labels: queryFilters.labels,
			}

			// Add negated labels to the Search field (Search API supports -label:X syntax)
			var searchParts []string
			for _, label := range queryFilters.negatedLabels {
				searchParts = append(searchParts, fmt.Sprintf("-label:%s", label))
			}
			if len(searchParts) > 0 {
				searchFilters.Search = strings.Join(searchParts, " ")
			}

			issues, err = client.SearchRepositoryIssues(owner, repo, searchFilters, 0)
			if err != nil {
				// Fall back to old method on Search API error
				issues, err = client.GetRepositoryIssues(owner, repo, queryFilters.state)
				if err != nil {
					continue
				}
				// Filter client-side as fallback
				for _, issue := range issues {
					if matchesTriageQuery(issue, query) {
						allIssues = append(allIssues, issue)
					}
				}
				continue
			}

			// Search API already filtered by labels, but still apply any other query filters
			// that might exist (for robustness)
			for _, issue := range issues {
				if matchesTriageQuery(issue, query) {
					allIssues = append(allIssues, issue)
				}
			}
		} else {
			// No label filters - use original method
			issues, err = client.GetRepositoryIssues(owner, repo, queryFilters.state)
			if err != nil {
				continue
			}

			// Filter based on query components
			for _, issue := range issues {
				if matchesTriageQuery(issue, query) {
					allIssues = append(allIssues, issue)
				}
			}
		}
	}

	return allIssues, nil
}

func matchesTriageQuery(issue api.Issue, query string) bool {
	// Basic query matching - supports common GitHub search qualifiers

	// Parse label requirements from query words
	for _, word := range strings.Fields(query) {
		if strings.HasPrefix(word, "-label:") {
			// Negative label: issue must NOT have this label
			labelName := strings.TrimPrefix(word, "-label:")
			for _, label := range issue.Labels {
				if label.Name == labelName {
					return false
				}
			}
		} else if strings.HasPrefix(word, "label:") {
			// Positive label: issue MUST have this label
			labelName := strings.TrimPrefix(word, "label:")
			found := false
			for _, label := range issue.Labels {
				if label.Name == labelName {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check state
	if strings.Contains(query, "is:open") && issue.State != "OPEN" {
		return false
	}
	if strings.Contains(query, "is:closed") && issue.State != "CLOSED" {
		return false
	}

	return true
}

func applyTriageRules(client triageClient, cfg *config.Config, project *api.Project, issue *api.Issue, tc *config.Triage) error {
	// First, ensure issue is in the project
	itemID, err := ensureIssueInProject(client, project.ID, issue.ID)
	if err != nil {
		return fmt.Errorf("failed to add issue to project: %w", err)
	}

	// Apply labels
	if len(tc.Apply.Labels) > 0 {
		for _, label := range tc.Apply.Labels {
			if err := api.WithRetry(func() error {
				return client.AddLabelToIssue(issue.Repository.Owner, issue.Repository.Name, issue.ID, label)
			}, 3); err != nil {
				// Log but don't fail - label might already exist
				continue
			}
		}
	}

	// Apply fields
	for field, value := range tc.Apply.Fields {
		fieldName := cfg.GetFieldName(field)
		resolvedValue := cfg.ResolveFieldValue(field, value)

		if err := api.WithRetry(func() error {
			return client.SetProjectItemField(project.ID, itemID, fieldName, resolvedValue)
		}, 3); err != nil {
			return fmt.Errorf("failed to set %s: %w", field, err)
		}
	}

	return nil
}

func ensureIssueInProject(client triageClient, projectID, issueID string) (string, error) {
	// Try to add - if already exists, this should return the existing item ID
	itemID, err := client.AddIssueToProject(projectID, issueID)
	if err != nil {
		// If error mentions already exists, try to find existing item
		if strings.Contains(err.Error(), "already") {
			// For now, just return the error - we'd need a query to find existing item
			return "", err
		}
		return "", err
	}
	return itemID, nil
}

func outputTriageTable(cmd *cobra.Command, issues []api.Issue) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NUMBER\tTITLE\tSTATE\tLABELS")

	for _, issue := range issues {
		title := issue.Title
		if len(title) > 45 {
			title = title[:42] + "..."
		}

		var labels []string
		for _, l := range issue.Labels {
			labels = append(labels, l.Name)
		}
		labelStr := strings.Join(labels, ", ")
		if labelStr == "" {
			labelStr = "-"
		}

		fmt.Fprintf(w, "#%d\t%s\t%s\t%s\n", issue.Number, title, issue.State, labelStr)
	}

	return w.Flush()
}

type triageJSONOutput struct {
	Status     string            `json:"status"`
	ConfigName string            `json:"configName"`
	Count      int               `json:"count"`
	Issues     []triageJSONIssue `json:"issues"`
}

type triageJSONIssue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state"`
	URL    string   `json:"url"`
	Labels []string `json:"labels"`
}

func outputTriageJSON(cmd *cobra.Command, issues []api.Issue, status, configName string) error {
	output := triageJSONOutput{
		Status:     status,
		ConfigName: configName,
		Count:      len(issues),
		Issues:     make([]triageJSONIssue, 0, len(issues)),
	}

	for _, issue := range issues {
		labels := make([]string, 0, len(issue.Labels))
		for _, l := range issue.Labels {
			labels = append(labels, l.Name)
		}

		output.Issues = append(output.Issues, triageJSONIssue{
			Number: issue.Number,
			Title:  issue.Title,
			State:  issue.State,
			URL:    issue.URL,
			Labels: labels,
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// runAdHocTriage runs a triage operation using --query and --apply flags instead of a config file entry
func runAdHocTriage(cmd *cobra.Command, opts *triageOptions, cfg *config.Config, client triageClient, stdin *os.File) error {
	// Get project
	project, err := client.GetProject(cfg.Project.Owner, cfg.Project.Number)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Search for issues matching the ad-hoc query
	matchingIssues, err := searchIssuesForTriage(client, cfg, opts.query, opts.repo)
	if err != nil {
		return fmt.Errorf("failed to search issues: %w", err)
	}

	if len(matchingIssues) == 0 {
		if opts.json {
			return outputTriageJSON(cmd, nil, "no-matches", "ad-hoc")
		}
		cmd.Println("No issues match the query")
		return nil
	}

	// Parse apply fields
	applyFields := parseTriageApplyFields(opts.apply)

	// Dry run - show what would be changed
	if opts.dryRun {
		if opts.json {
			return outputTriageJSON(cmd, matchingIssues, "dry-run", "ad-hoc")
		}
		cmd.Printf("Would process %d issue(s) with query %q:\n\n", len(matchingIssues), opts.query)
		_ = outputTriageTable(cmd, matchingIssues)
		cmd.Println()
		if len(applyFields) > 0 {
			cmd.Println("Actions to apply:")
			for field, value := range applyFields {
				resolved := cfg.ResolveFieldValue(field, value)
				cmd.Printf("  • Set %s: %s\n", field, resolved)
			}
		}
		return nil
	}

	// Process issues
	var processed, skipped, failed int
	reader := bufio.NewReader(stdin)

	for _, issue := range matchingIssues {
		// Interactive mode - prompt for each issue
		if opts.interactive {
			cmd.Printf("\nProcess #%d: %s? [y/n/q] ", issue.Number, issue.Title)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "q" {
				cmd.Println("Aborted.")
				break
			}
			if response != "y" && response != "yes" {
				skipped++
				continue
			}
		}

		// Apply ad-hoc rules
		err := applyAdHocTriageRules(client, cfg, project, &issue, applyFields)
		if err != nil {
			cmd.PrintErrf("Failed to process #%d: %v\n", issue.Number, err)
			failed++
			continue
		}

		processed++
		if !opts.interactive {
			cmd.Printf("Processed #%d: %s\n", issue.Number, issue.Title)
		}
	}

	// Summary
	if opts.json {
		return outputTriageJSON(cmd, matchingIssues, "completed", "ad-hoc")
	}

	cmd.Printf("\nTriage complete: %d processed", processed)
	if skipped > 0 {
		cmd.Printf(", %d skipped", skipped)
	}
	if failed > 0 {
		cmd.Printf(", %d failed", failed)
	}
	cmd.Println()

	return nil
}

// applyAdHocTriageRules applies fields specified via --apply flag
func applyAdHocTriageRules(client triageClient, cfg *config.Config, project *api.Project, issue *api.Issue, applyFields map[string]string) error {
	// First, ensure issue is in the project
	itemID, err := ensureIssueInProject(client, project.ID, issue.ID)
	if err != nil {
		return fmt.Errorf("failed to add issue to project: %w", err)
	}

	// Apply fields
	for field, value := range applyFields {
		fieldName := cfg.GetFieldName(field)
		resolvedValue := cfg.ResolveFieldValue(field, value)

		if err := api.WithRetry(func() error {
			return client.SetProjectItemField(project.ID, itemID, fieldName, resolvedValue)
		}, 3); err != nil {
			return fmt.Errorf("failed to set %s: %w", field, err)
		}
	}

	return nil
}

// parseTriageApplyFields parses a comma-separated list of key:value pairs
// Example: "status:backlog,priority:p1" -> {"status": "backlog", "priority": "p1"}
func parseTriageApplyFields(s string) map[string]string {
	result := make(map[string]string)
	if s == "" {
		return result
	}

	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" && value != "" {
				result[key] = value
			}
		}
	}
	return result
}
