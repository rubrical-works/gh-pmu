package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/rubrical-works/gh-pmu/internal/defaults"
	"github.com/rubrical-works/gh-pmu/internal/ui"
	"github.com/spf13/cobra"
)

// labelClient defines the interface for API methods used by label functions.
type labelClient interface {
	LabelExists(owner, repo, labelName string) (bool, error)
	CreateLabel(owner, repo, name, color, description string) error
	UpdateLabel(owner, repo, labelName, newName, newColor, newDescription string) error
	DeleteLabel(owner, repo, labelName string) error
	ListLabels(owner, repo string) ([]api.RepoLabel, error)
	GetLabel(owner, repo, labelName string) (*api.RepoLabel, error)
}

func newLabelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "Manage repository labels",
		Long: `Manage repository labels without running a full init.

Use subcommands to sync standard labels, list all labels, or create/update/delete individual labels.`,
	}

	cmd.AddCommand(newLabelSyncCommand())
	cmd.AddCommand(newLabelListCommand())
	cmd.AddCommand(newLabelAddCommand())
	cmd.AddCommand(newLabelUpdateCommand())
	cmd.AddCommand(newLabelDeleteCommand())

	return cmd
}

// --- sync ---

type labelSyncOptions struct {
	dryRun bool
	update bool
	repo   string
}

func newLabelSyncCommand() *cobra.Command {
	opts := &labelSyncOptions{}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync standard labels from defaults",
		Long: `Ensure all standard labels from defaults.yml exist in the repository.

By default, only creates missing labels. Use --update to also update
color and description of existing labels to match defaults.

Examples:
  gh pmu label sync
  gh pmu label sync --dry-run
  gh pmu label sync --update`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelSync(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Preview changes without applying")
	cmd.Flags().BoolVar(&opts.update, "update", false, "Update color/description of existing labels")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository (owner/repo format)")

	return cmd
}

func runLabelSync(cmd *cobra.Command, opts *labelSyncOptions) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}
	return runLabelSyncWithDeps(cmd, opts, client)
}

func runLabelSyncWithDeps(cmd *cobra.Command, opts *labelSyncOptions, client labelClient) error {
	owner, repo, err := resolveRepo(opts.repo)
	if err != nil {
		return err
	}

	defs, err := defaults.Load()
	if err != nil {
		return fmt.Errorf("failed to load defaults: %w", err)
	}

	u := ui.New(cmd.OutOrStdout())
	return SyncLabels(cmd.OutOrStdout(), u, client, owner, repo, defs.Labels, opts.dryRun, opts.update)
}

// SyncLabels synchronizes standard labels to a repository.
// This is the shared logic used by both 'label sync' and 'init'.
func SyncLabels(out io.Writer, u *ui.UI, client labelClient, owner, repo string, labels []defaults.LabelDef, dryRun, update bool) error {
	u.Info("Checking repository labels...")

	var created, updated, skipped int

	for _, labelDef := range labels {
		if update {
			// In update mode, check current state and update if different
			existing, err := client.GetLabel(owner, repo, labelDef.Name)
			if err != nil {
				u.Warning(fmt.Sprintf("Could not check %s label: %v", labelDef.Name, err))
				skipped++
				continue
			}

			if existing == nil {
				// Label doesn't exist — create it
				if dryRun {
					fmt.Fprintf(out, "  would create: %s (#%s)\n", labelDef.Name, labelDef.Color)
					created++
					continue
				}
				spinner := ui.NewSpinner(out, fmt.Sprintf("Creating %s label...", labelDef.Name))
				spinner.Start()
				err := client.CreateLabel(owner, repo, labelDef.Name, labelDef.Color, labelDef.Description)
				spinner.Stop()
				if err != nil {
					u.Warning(fmt.Sprintf("Could not create %s label: %v", labelDef.Name, err))
					skipped++
				} else {
					u.Success(fmt.Sprintf("Created %s label", labelDef.Name))
					created++
				}
				continue
			}

			// Label exists — check if it needs updating
			colorDiffers := !strings.EqualFold(existing.Color, labelDef.Color)
			descDiffers := existing.Description != labelDef.Description
			if colorDiffers || descDiffers {
				if dryRun {
					changes := []string{}
					if colorDiffers {
						changes = append(changes, fmt.Sprintf("color #%s → #%s", existing.Color, labelDef.Color))
					}
					if descDiffers {
						changes = append(changes, "description")
					}
					fmt.Fprintf(out, "  would update: %s (%s)\n", labelDef.Name, strings.Join(changes, ", "))
					updated++
					continue
				}
				spinner := ui.NewSpinner(out, fmt.Sprintf("Updating %s label...", labelDef.Name))
				spinner.Start()
				err := client.UpdateLabel(owner, repo, labelDef.Name, labelDef.Name, labelDef.Color, labelDef.Description)
				spinner.Stop()
				if err != nil {
					u.Warning(fmt.Sprintf("Could not update %s label: %v", labelDef.Name, err))
					skipped++
				} else {
					u.Success(fmt.Sprintf("Updated %s label", labelDef.Name))
					updated++
				}
			} else {
				u.Success(fmt.Sprintf("%s label up to date", labelDef.Name))
			}
		} else {
			// Default mode: only create missing labels
			exists, err := client.LabelExists(owner, repo, labelDef.Name)
			if err != nil {
				u.Warning(fmt.Sprintf("Could not check %s label: %v", labelDef.Name, err))
				skipped++
				continue
			}
			if exists {
				u.Success(fmt.Sprintf("%s label exists", labelDef.Name))
				continue
			}

			if dryRun {
				fmt.Fprintf(out, "  would create: %s (#%s)\n", labelDef.Name, labelDef.Color)
				created++
				continue
			}

			spinner := ui.NewSpinner(out, fmt.Sprintf("Creating %s label...", labelDef.Name))
			spinner.Start()
			err = client.CreateLabel(owner, repo, labelDef.Name, labelDef.Color, labelDef.Description)
			spinner.Stop()
			if err != nil {
				u.Warning(fmt.Sprintf("Could not create %s label: %v", labelDef.Name, err))
				skipped++
			} else {
				u.Success(fmt.Sprintf("Created %s label", labelDef.Name))
				created++
			}
		}
	}

	if dryRun {
		fmt.Fprintf(out, "\nDry run: %d to create, %d to update\n", created, updated)
	} else {
		fmt.Fprintf(out, "\nSync complete: %d created, %d updated, %d skipped\n", created, updated, skipped)
	}

	return nil
}

// --- list ---

type labelListOptions struct {
	repo string
}

func newLabelListCommand() *cobra.Command {
	opts := &labelListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all repository labels",
		Long: `List all labels in the repository with a standard/custom indicator.

Labels that match entries in defaults.yml are marked as "standard".
Other labels are marked as "custom".

Examples:
  gh pmu label list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelList(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository (owner/repo format)")

	return cmd
}

func runLabelList(cmd *cobra.Command, opts *labelListOptions) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}
	return runLabelListWithDeps(cmd, opts, client)
}

func runLabelListWithDeps(cmd *cobra.Command, opts *labelListOptions, client labelClient) error {
	owner, repo, err := resolveRepo(opts.repo)
	if err != nil {
		return err
	}

	defs, err := defaults.Load()
	if err != nil {
		return fmt.Errorf("failed to load defaults: %w", err)
	}

	labels, err := client.ListLabels(owner, repo)
	if err != nil {
		return fmt.Errorf("failed to list labels: %w", err)
	}

	out := cmd.OutOrStdout()

	if len(labels) == 0 {
		fmt.Fprintln(out, "No labels found.")
		return nil
	}

	// Find max name length for alignment
	maxName := 0
	for _, l := range labels {
		if len(l.Name) > maxName {
			maxName = len(l.Name)
		}
	}

	for _, l := range labels {
		kind := "custom"
		if defs.IsStandardLabel(l.Name) {
			kind = "standard"
		}
		fmt.Fprintf(out, "  %-*s  #%-6s  %-8s  %s\n", maxName, l.Name, l.Color, kind, l.Description)
	}

	fmt.Fprintf(out, "\n%d labels (%d standard, %d custom)\n",
		len(labels), countStandard(labels, defs), len(labels)-countStandard(labels, defs))

	return nil
}

func countStandard(labels []api.RepoLabel, defs *defaults.Defaults) int {
	count := 0
	for _, l := range labels {
		if defs.IsStandardLabel(l.Name) {
			count++
		}
	}
	return count
}

// --- add ---

type labelAddOptions struct {
	color       string
	description string
	repo        string
}

func newLabelAddCommand() *cobra.Command {
	opts := &labelAddOptions{}

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a label",
		Long: `Create a new label in the repository.

Examples:
  gh pmu label add my-label --color ff0000 --description "My custom label"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelAdd(cmd, args, opts)
		},
	}

	cmd.Flags().StringVar(&opts.color, "color", "", "Label color (hex without #)")
	cmd.Flags().StringVar(&opts.description, "description", "", "Label description")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository (owner/repo format)")
	_ = cmd.MarkFlagRequired("color")

	return cmd
}

func runLabelAdd(cmd *cobra.Command, args []string, opts *labelAddOptions) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}
	return runLabelAddWithDeps(cmd, args, opts, client)
}

func runLabelAddWithDeps(cmd *cobra.Command, args []string, opts *labelAddOptions, client labelClient) error {
	owner, repo, err := resolveRepo(opts.repo)
	if err != nil {
		return err
	}

	name := args[0]
	color := strings.TrimPrefix(opts.color, "#")

	err = client.CreateLabel(owner, repo, name, color, opts.description)
	if err != nil {
		return fmt.Errorf("failed to create label: %w", err)
	}

	u := ui.New(cmd.OutOrStdout())
	u.Success(fmt.Sprintf("Created label %q (#%s)", name, color))
	return nil
}

// --- update ---

type labelUpdateOptions struct {
	color       string
	description string
	repo        string
}

func newLabelUpdateCommand() *cobra.Command {
	opts := &labelUpdateOptions{}

	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update a label",
		Long: `Update an existing label's color or description.

Examples:
  gh pmu label update my-label --color 00ff00
  gh pmu label update my-label --description "Updated description"
  gh pmu label update my-label --color 00ff00 --description "New desc"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelUpdate(cmd, args, opts)
		},
	}

	cmd.Flags().StringVar(&opts.color, "color", "", "New label color (hex without #)")
	cmd.Flags().StringVar(&opts.description, "description", "", "New label description")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository (owner/repo format)")

	return cmd
}

func runLabelUpdate(cmd *cobra.Command, args []string, opts *labelUpdateOptions) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}
	return runLabelUpdateWithDeps(cmd, args, opts, client)
}

func runLabelUpdateWithDeps(cmd *cobra.Command, args []string, opts *labelUpdateOptions, client labelClient) error {
	owner, repo, err := resolveRepo(opts.repo)
	if err != nil {
		return err
	}

	name := args[0]

	// Get current label to fill in unchanged fields
	existing, err := client.GetLabel(owner, repo, name)
	if err != nil {
		return fmt.Errorf("failed to get label: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("label %q not found", name)
	}

	newColor := existing.Color
	if opts.color != "" {
		newColor = strings.TrimPrefix(opts.color, "#")
	}

	newDesc := existing.Description
	if cmd.Flags().Changed("description") {
		newDesc = opts.description
	}

	err = client.UpdateLabel(owner, repo, name, name, newColor, newDesc)
	if err != nil {
		return fmt.Errorf("failed to update label: %w", err)
	}

	u := ui.New(cmd.OutOrStdout())
	u.Success(fmt.Sprintf("Updated label %q", name))
	return nil
}

// --- delete ---

type labelDeleteOptions struct {
	repo string
}

func newLabelDeleteCommand() *cobra.Command {
	opts := &labelDeleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a label",
		Long: `Delete a label from the repository.

Examples:
  gh pmu label delete my-label`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelDelete(cmd, args, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository (owner/repo format)")

	return cmd
}

func runLabelDelete(cmd *cobra.Command, args []string, opts *labelDeleteOptions) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}
	return runLabelDeleteWithDeps(cmd, args, opts, client)
}

func runLabelDeleteWithDeps(cmd *cobra.Command, args []string, opts *labelDeleteOptions, client labelClient) error {
	owner, repo, err := resolveRepo(opts.repo)
	if err != nil {
		return err
	}

	name := args[0]

	err = client.DeleteLabel(owner, repo, name)
	if err != nil {
		return fmt.Errorf("failed to delete label: %w", err)
	}

	u := ui.New(cmd.OutOrStdout())
	u.Success(fmt.Sprintf("Deleted label %q", name))
	return nil
}

// --- helpers ---

// resolveRepo parses owner/repo from flag or config
func resolveRepo(repoOverride string) (string, string, error) {
	if repoOverride != "" {
		parts := strings.Split(repoOverride, "/")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid --repo format: expected owner/repo, got %s", repoOverride)
		}
		return parts[0], parts[1], nil
	}

	dir := mustGetwd()
	configPath, err := config.FindConfigFile(dir)
	if err != nil {
		return "", "", fmt.Errorf("no repository specified and no config found (use --repo or run gh pmu init)")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Repositories) == 0 {
		return "", "", fmt.Errorf("no repository configured (use --repo or add to .gh-pmu.json)")
	}

	parts := strings.Split(cfg.Repositories[0], "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format in config: %s", cfg.Repositories[0])
	}

	return parts[0], parts[1], nil
}
