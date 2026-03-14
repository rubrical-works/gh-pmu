package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/spf13/cobra"
)

// splitClient defines the interface for API methods used by split functions.
// This allows for easier testing with mock implementations.
type splitClient interface {
	GetIssue(owner, repo string, number int) (*api.Issue, error)
	CreateIssue(owner, repo, title, body string, labels []string) (*api.Issue, error)
	AddSubIssue(parentID, issueID string) error
}

type splitOptions struct {
	from   string
	dryRun bool
	json   bool
	repo   string
}

func newSplitCommand() *cobra.Command {
	opts := &splitOptions{}

	cmd := &cobra.Command{
		Use:   "split <issue> [tasks...]",
		Short: "Split an issue into sub-issues",
		Long: `Split an issue into multiple sub-issues from a checklist or arguments.

The checklist can come from:
- The issue body (--from=body)
- An external file (--from=path/to/file.md)
- Command line arguments (gh pmu split 123 "Task 1" "Task 2")

Only unchecked items (- [ ]) are converted to sub-issues.
Completed items (- [x]) are skipped.`,
		Example: `  # Split from issue body checklist
  gh pmu split 123 --from=body

  # Split from external file
  gh pmu split 123 --from=tasks.md

  # Split from command line arguments
  gh pmu split 123 "Implement feature A" "Implement feature B" "Write tests"

  # Preview without creating
  gh pmu split 123 --from=body --dry-run`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSplit(cmd, args, opts)
		},
	}

	cmd.Flags().StringVar(&opts.from, "from", "", "Source for tasks: 'body' (issue body) or file path")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show what would be created without making changes")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository for the issue (owner/repo format)")

	return cmd
}

func runSplit(cmd *cobra.Command, args []string, opts *splitOptions) error {
	// Parse issue number
	issueNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid issue number: %s", args[0])
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

	// Determine repository (--repo flag takes precedence over config)
	var owner, repo string
	if opts.repo != "" {
		parts := strings.Split(opts.repo, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid --repo format: expected owner/repo, got %s", opts.repo)
		}
		owner, repo = parts[0], parts[1]
	} else if len(cfg.Repositories) > 0 {
		parts := strings.SplitN(cfg.Repositories[0], "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid repository format: %s", cfg.Repositories[0])
		}
		owner, repo = parts[0], parts[1]
	} else {
		return fmt.Errorf("no repository specified and none configured (use --repo or configure in .gh-pmu.yml)")
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	return runSplitWithDeps(cmd, args, opts, client, owner, repo, issueNum)
}

// runSplitWithDeps is the testable implementation of runSplit
func runSplitWithDeps(cmd *cobra.Command, args []string, opts *splitOptions, client splitClient, owner, repo string, issueNum int) error {
	// Get the parent issue
	parentIssue, err := client.GetIssue(owner, repo, issueNum)
	if err != nil {
		return fmt.Errorf("failed to get issue #%d: %w", issueNum, err)
	}

	// Determine tasks to create
	var tasks []string

	if opts.from != "" {
		if opts.from == "body" {
			// Parse from issue body
			tasks = parseChecklist(parentIssue.Body)
		} else {
			// Parse from file
			content, err := os.ReadFile(opts.from)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", opts.from, err)
			}
			tasks = parseChecklist(string(content))
		}
	} else if len(args) > 1 {
		// Tasks from command line arguments
		tasks = args[1:]
	} else {
		return fmt.Errorf("no tasks specified\nUse --from=body, --from=<file>, or provide tasks as arguments")
	}

	if len(tasks) == 0 {
		if opts.json {
			return outputSplitJSON(cmd, parentIssue, nil, "no-tasks")
		}
		cmd.Println("No tasks found to create as sub-issues")
		return nil
	}

	// Dry run - just show what would be created
	if opts.dryRun {
		if opts.json {
			return outputSplitJSON(cmd, parentIssue, tasks, "dry-run")
		}
		cmd.Printf("Would create %d sub-issue(s) under #%d: %s\n\n", len(tasks), parentIssue.Number, parentIssue.Title)
		for i, task := range tasks {
			cmd.Printf("  %d. %s\n", i+1, task)
		}
		return nil
	}

	// Create sub-issues
	var created []api.Issue
	var failed []string

	for _, task := range tasks {
		// Create the issue
		newIssue, err := client.CreateIssue(owner, repo, task, "", nil)
		if err != nil {
			cmd.PrintErrf("Failed to create sub-issue %q: %v\n", task, err)
			failed = append(failed, task)
			continue
		}

		// Link as sub-issue
		err = client.AddSubIssue(parentIssue.ID, newIssue.ID)
		if err != nil {
			cmd.PrintErrf("Created #%d but failed to link as sub-issue: %v\n", newIssue.Number, err)
			// Still count as created since issue exists
		}

		created = append(created, *newIssue)
		cmd.Printf("Created sub-issue #%d: %s\n", newIssue.Number, newIssue.Title)
	}

	// Summary
	if opts.json {
		return outputSplitJSONCreated(cmd, parentIssue, created, failed)
	}

	cmd.Printf("\nSplit complete: %d sub-issue(s) created under #%d", len(created), parentIssue.Number)
	if len(failed) > 0 {
		cmd.Printf(" (%d failed)", len(failed))
	}
	cmd.Println()

	return nil
}

// parseChecklist extracts unchecked checklist items from markdown text
func parseChecklist(text string) []string {
	var tasks []string

	// Match unchecked checklist items: - [ ] Task text
	// Regex: starts with - [ ] followed by whitespace and task text
	re := regexp.MustCompile(`(?m)^[\s]*-\s*\[\s*\]\s*(.+)$`)

	matches := re.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			task := strings.TrimSpace(match[1])
			if task != "" {
				tasks = append(tasks, task)
			}
		}
	}

	return tasks
}

func outputSplitJSON(cmd *cobra.Command, parent *api.Issue, tasks []string, status string) error {
	output := map[string]interface{}{
		"status": status,
		"parent": map[string]interface{}{
			"number": parent.Number,
			"title":  parent.Title,
			"url":    parent.URL,
		},
		"taskCount": len(tasks),
		"tasks":     tasks,
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputSplitJSONCreated(cmd *cobra.Command, parent *api.Issue, created []api.Issue, failed []string) error {
	createdJSON := make([]map[string]interface{}, 0, len(created))
	for _, issue := range created {
		createdJSON = append(createdJSON, map[string]interface{}{
			"number": issue.Number,
			"title":  issue.Title,
			"url":    issue.URL,
		})
	}

	output := map[string]interface{}{
		"status": "completed",
		"parent": map[string]interface{}{
			"number": parent.Number,
			"title":  parent.Title,
			"url":    parent.URL,
		},
		"createdCount": len(created),
		"failedCount":  len(failed),
		"created":      createdJSON,
		"failed":       failed,
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
