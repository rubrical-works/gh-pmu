package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/spf13/cobra"
)

// commentClient defines the interface for API methods used by comment functions.
// This allows for easier testing with mock implementations.
type commentClient interface {
	GetIssueByNumber(owner, repo string, number int) (*api.Issue, error)
	AddIssueComment(issueID, body string) (*api.Comment, error)
}

type commentOptions struct {
	issueNumber int
	body        string
	bodyFile    string
	bodyStdin   bool
	repo        string
}

func newCommentCommand() *cobra.Command {
	opts := &commentOptions{}

	cmd := &cobra.Command{
		Use:   "comment <issue-number>",
		Short: "Add a comment to an issue",
		Long: `Add a comment to an existing issue.

The comment body can be provided via --body flag, read from a file with -F,
or read from stdin with --body-stdin.

Examples:
  # Add a simple comment
  gh pmu comment 123 --body "This is a comment"

  # Add a comment from a file
  gh pmu comment 123 -F comment.md

  # Add a comment from stdin (useful with heredocs)
  gh pmu comment 123 --body-stdin <<'EOF'
  Multi-line comment
  with markdown support
  EOF

  # Pipe content as a comment
  echo "Comment from pipe" | gh pmu comment 123 --body-stdin

  # Specify repository explicitly
  gh pmu comment 123 --body "Comment" --repo owner/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var issueNum int
			if _, err := fmt.Sscanf(args[0], "%d", &issueNum); err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}
			opts.issueNumber = issueNum

			return runComment(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.body, "body", "b", "", "Comment body text")
	cmd.Flags().StringVarP(&opts.bodyFile, "body-file", "F", "", "Read comment body from file (use \"-\" to read from standard input)")
	cmd.Flags().BoolVar(&opts.bodyStdin, "body-stdin", false, "Read comment body from standard input")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository for the issue (owner/repo format)")

	return cmd
}

func runComment(cmd *cobra.Command, opts *commentOptions) error {
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
		repoParts := strings.Split(opts.repo, "/")
		if len(repoParts) != 2 {
			return fmt.Errorf("invalid --repo format: expected owner/repo, got %s", opts.repo)
		}
		owner, repo = repoParts[0], repoParts[1]
	} else {
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

	return runCommentWithDeps(cmd, opts, client, owner, repo)
}

// runCommentWithDeps is the testable implementation of runComment
func runCommentWithDeps(cmd *cobra.Command, opts *commentOptions, client commentClient, owner, repo string) error {
	// Validate that exactly one body source is provided
	bodySourceCount := 0
	if opts.body != "" {
		bodySourceCount++
	}
	if opts.bodyFile != "" {
		bodySourceCount++
	}
	if opts.bodyStdin {
		bodySourceCount++
	}

	if bodySourceCount == 0 {
		return fmt.Errorf("one of --body, --body-file (-F), or --body-stdin is required")
	}
	if bodySourceCount > 1 {
		return fmt.Errorf("only one of --body, --body-file (-F), or --body-stdin can be used")
	}

	// Get comment body
	body := opts.body

	if opts.bodyStdin {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read body from stdin: %w", err)
		}
		body = string(content)
	}

	if opts.bodyFile != "" {
		content, err := readBodyFile(opts.bodyFile)
		if err != nil {
			return fmt.Errorf("failed to read body file: %w", err)
		}
		body = content
	}

	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	// Get the issue
	issue, err := client.GetIssueByNumber(owner, repo, opts.issueNumber)
	if err != nil {
		return fmt.Errorf("failed to get issue #%d: %w", opts.issueNumber, err)
	}

	// Add the comment
	comment, err := client.AddIssueComment(issue.ID, body)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added comment to issue #%d\n", opts.issueNumber)
	fmt.Fprintf(cmd.OutOrStdout(), "%s#issuecomment-%d\n", issue.URL, comment.DatabaseId)

	return nil
}
