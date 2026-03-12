package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/spf13/cobra"
)

// closeClient defines the interface for API methods used by close functions.
// This allows for easier testing with mock implementations.
type closeClient interface {
	GetProject(owner string, number int) (*api.Project, error)
	GetProjectItemIDForIssue(projectID, owner, repo string, number int) (string, error)
	SetProjectItemField(projectID, itemID, fieldName, value string) error
}

type closeOptions struct {
	reason       string
	comment      string
	updateStatus bool
	repo         string
}

func newCloseCommand() *cobra.Command {
	opts := &closeOptions{}

	cmd := &cobra.Command{
		Use:   "close <issue-number>",
		Short: "Close an issue with reason alias support",
		Long: `Close an issue with support for underscore-style reason aliases.

This command wraps 'gh issue close' and normalizes close reasons to
be consistent with gh-pmu's underscore-based conventions.

Reason aliases:
  not_planned  →  "not planned"
  completed    →  "completed"

Both underscore and space versions are accepted and normalized.

Examples:
  # Close as not planned (using underscore alias)
  gh pmu close 123 --reason not_planned

  # Close as completed with a comment
  gh pmu close 123 --reason completed --comment "Fixed in v1.0"

  # Close and update project status to "done"
  gh pmu close 123 --reason completed --update-status

  # Short flags
  gh pmu close 123 -r not_planned -c "Duplicate of #100"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClose(cmd, args, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.reason, "reason", "r", "", "Reason for closing: not_planned, completed")
	cmd.Flags().StringVarP(&opts.comment, "comment", "c", "", "Leave a closing comment")
	cmd.Flags().BoolVar(&opts.updateStatus, "update-status", false, "Move issue to 'done' status before closing")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository for the issue (owner/repo format)")

	return cmd
}

// normalizeCloseReason converts underscore aliases to the format expected by gh CLI
func normalizeCloseReason(reason string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(reason))

	switch normalized {
	case "not_planned", "not planned", "notplanned":
		return "not planned", nil
	case "completed", "complete", "done":
		return "completed", nil
	case "":
		return "", nil
	default:
		return "", fmt.Errorf("invalid close reason %q: valid values are not_planned, completed", reason)
	}
}

func runClose(cmd *cobra.Command, args []string, opts *closeOptions) error {
	// Parse issue number
	issueNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}

	// Normalize reason if provided
	normalizedReason, err := normalizeCloseReason(opts.reason)
	if err != nil {
		return err
	}

	// If --update-status, update project status to "done" first
	if opts.updateStatus {
		if err := updateStatusToDone(issueNum, opts.repo); err != nil {
			// Warn but continue with close
			fmt.Fprintf(os.Stderr, "Warning: failed to update status: %v\n", err)
		}
	}

	// Build gh issue close command
	ghArgs := []string{"issue", "close", strconv.Itoa(issueNum)}

	if opts.repo != "" {
		// Validate repo format
		parts := strings.Split(opts.repo, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid --repo format: expected owner/repo, got %s", opts.repo)
		}
		ghArgs = append(ghArgs, "-R", opts.repo)
	}

	if normalizedReason != "" {
		ghArgs = append(ghArgs, "--reason", normalizedReason)
	}

	if opts.comment != "" {
		ghArgs = append(ghArgs, "--comment", opts.comment)
	}

	// Execute gh issue close
	ghCmd := exec.Command("gh", ghArgs...)
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr

	return ghCmd.Run()
}

// updateStatusToDone moves the issue to "done" status in the project
func updateStatusToDone(issueNum int, repoOverride string) error {
	// Load configuration
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	cfg, err := config.LoadFromDirectory(cwd)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create API client
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	return updateStatusToDoneWithDeps(issueNum, repoOverride, cfg, client, os.Stdout)
}

// updateStatusToDoneWithDeps is the testable implementation of updateStatusToDone
func updateStatusToDoneWithDeps(issueNum int, repoOverride string, cfg *config.Config, client closeClient, stdout *os.File) error {
	// Determine repository (--repo flag takes precedence over config)
	var owner, repo string
	if repoOverride != "" {
		parts := strings.Split(repoOverride, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid --repo format: expected owner/repo, got %s", repoOverride)
		}
		owner, repo = parts[0], parts[1]
	} else if len(cfg.Repositories) > 0 {
		parts := strings.Split(cfg.Repositories[0], "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid repository format in config: %s", cfg.Repositories[0])
		}
		owner, repo = parts[0], parts[1]
	} else {
		return fmt.Errorf("no repository specified and none configured (use --repo or configure in .gh-pmu.yml)")
	}

	// Get project
	project, err := client.GetProject(cfg.Project.Owner, cfg.Project.Number)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Get the project item ID for this issue (efficient single query)
	itemID, err := client.GetProjectItemIDForIssue(project.ID, owner, repo, issueNum)
	if err != nil {
		return fmt.Errorf("failed to find issue in project: %w", err)
	}

	// Resolve "done" status value
	doneValue := cfg.ResolveFieldValue("status", "done")

	// Update the status
	if err := client.SetProjectItemField(project.ID, itemID, "Status", doneValue); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	fmt.Fprintf(stdout, "✓ Updated issue #%d status to %s\n", issueNum, doneValue)
	return nil
}
