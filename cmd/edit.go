package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/spf13/cobra"
)

// editClient defines the interface for API methods used by edit functions.
// This allows for easier testing with mock implementations.
type editClient interface {
	GetIssueByNumber(owner, repo string, number int) (*api.Issue, error)
	UpdateIssueBody(issueID, body string) error
	UpdateIssueTitle(issueID, title string) error
	AddLabelToIssue(owner, repo, issueID, labelName string) error
	RemoveLabelFromIssue(owner, repo, issueID, labelName string) error
}

type editOptions struct {
	issueNumber  int
	title        string
	body         string
	bodyFile     string
	bodyStdin    bool
	addLabels    []string
	removeLabels []string
	repo         string
}

func newEditCommand() *cobra.Command {
	opts := &editOptions{}

	cmd := &cobra.Command{
		Use:   "edit <issue-number>",
		Short: "Edit an issue",
		Long: `Edit an existing issue's title, body, or labels.

This command provides body-file support to complete the round-trip workflow:
  1. Export body: gh pmu view 123 --body-file
  2. Edit the file
  3. Update issue: gh pmu edit 123 --body-file tmp/issue-123.md

Examples:
  gh pmu edit 123 --body-file tmp/issue-123.md
  gh pmu edit 123 --title "New title"
  gh pmu edit 123 --label bug --label urgent
  gh pmu edit 123 --remove-label old-label
  gh pmu edit 123 --body-file issue.md --title "Updated" --label fix
  gh pmu edit 123 --body "Updated body" --repo owner/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var issueNum int
			if _, err := fmt.Sscanf(args[0], "%d", &issueNum); err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}
			opts.issueNumber = issueNum

			return runEdit(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "New issue title")
	cmd.Flags().StringVarP(&opts.body, "body", "b", "", "New issue body")
	cmd.Flags().StringVarP(&opts.bodyFile, "body-file", "F", "", "Read body text from file (use \"-\" to read from standard input)")
	cmd.Flags().BoolVar(&opts.bodyStdin, "body-stdin", false, "Read body from stdin (raw markdown)")
	cmd.Flags().StringArrayVarP(&opts.addLabels, "label", "l", nil, "Add labels (can be specified multiple times)")
	cmd.Flags().StringArrayVar(&opts.removeLabels, "remove-label", nil, "Remove labels (can be specified multiple times)")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository for the issue (owner/repo format)")

	return cmd
}

func runEdit(cmd *cobra.Command, opts *editOptions) error {
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

	// Determine repository
	var owner, repo string
	if opts.repo != "" {
		// Use --repo flag
		repoParts := strings.Split(opts.repo, "/")
		if len(repoParts) != 2 {
			return fmt.Errorf("invalid --repo format: expected owner/repo, got %s", opts.repo)
		}
		owner, repo = repoParts[0], repoParts[1]
	} else {
		// Use config
		if len(cfg.Repositories) == 0 {
			return fmt.Errorf("no repository configured")
		}
		repoParts := strings.Split(cfg.Repositories[0], "/")
		if len(repoParts) != 2 {
			return fmt.Errorf("invalid repository format in config: %s", cfg.Repositories[0])
		}
		owner, repo = repoParts[0], repoParts[1]
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	return runEditWithDeps(cmd, opts, cfg, client, owner, repo)
}

// runEditWithDeps is the testable implementation of runEdit
func runEditWithDeps(cmd *cobra.Command, opts *editOptions, cfg *config.Config, client editClient, owner, repo string) error {
	// Validate that at least one edit option is provided
	if opts.title == "" && opts.body == "" && opts.bodyFile == "" && !opts.bodyStdin && len(opts.addLabels) == 0 && len(opts.removeLabels) == 0 {
		return fmt.Errorf("at least one of --title, --body, --body-file, --body-stdin, --label, or --remove-label is required")
	}

	// Cannot use --body and --body-file together
	if opts.body != "" && opts.bodyFile != "" {
		return fmt.Errorf("cannot use --body and --body-file together")
	}

	// Cannot use --body-stdin with --body or --body-file
	if opts.bodyStdin && (opts.body != "" || opts.bodyFile != "") {
		return fmt.Errorf("cannot use --body-stdin with --body or --body-file")
	}

	// Convert --body-stdin to internal --body-file="-" representation
	if opts.bodyStdin {
		opts.bodyFile = "-"
	}

	// Get the issue to edit
	issue, err := client.GetIssueByNumber(owner, repo, opts.issueNumber)
	if err != nil {
		return fmt.Errorf("failed to get issue #%d: %w", opts.issueNumber, err)
	}

	var updates []string

	// Update title if provided
	if opts.title != "" {
		if err := client.UpdateIssueTitle(issue.ID, opts.title); err != nil {
			return fmt.Errorf("failed to update title: %w", err)
		}
		updates = append(updates, "title")
	}

	// Update body if provided
	body := opts.body
	if opts.bodyFile != "" {
		content, err := readBodyFile(opts.bodyFile)
		if err != nil {
			return fmt.Errorf("failed to read body file: %w", err)
		}
		body = content
	}
	if body != "" {
		if err := client.UpdateIssueBody(issue.ID, body); err != nil {
			return fmt.Errorf("failed to update body: %w", err)
		}
		updates = append(updates, "body")
	}

	// Add labels if provided
	if len(opts.addLabels) > 0 {
		for _, label := range opts.addLabels {
			if err := client.AddLabelToIssue(owner, repo, issue.ID, label); err != nil {
				return fmt.Errorf("failed to add label '%s': %w", label, err)
			}
		}
		updates = append(updates, fmt.Sprintf("%d label(s) added", len(opts.addLabels)))
	}

	// Remove labels if provided
	if len(opts.removeLabels) > 0 {
		for _, label := range opts.removeLabels {
			if err := client.RemoveLabelFromIssue(owner, repo, issue.ID, label); err != nil {
				return fmt.Errorf("failed to remove label '%s': %w", label, err)
			}
		}
		updates = append(updates, fmt.Sprintf("%d label(s) removed", len(opts.removeLabels)))
	}

	// Output confirmation
	fmt.Fprintf(cmd.OutOrStdout(), "Updated issue #%d: %s\n", opts.issueNumber, strings.Join(updates, ", "))
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", issue.URL)

	return nil
}
