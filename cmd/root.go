package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/rubrical-works/gh-pmu/internal/defaults"
	pkgversion "github.com/rubrical-works/gh-pmu/internal/version"
	"github.com/spf13/cobra"
)

// version is set by ldflags during goreleaser builds.
// When empty (default), falls back to the source constant in internal/version.
var version = ""

func getVersion() string {
	if version != "" {
		return version
	}
	return pkgversion.Version
}

// exemptCommands are commands that do not require terms acceptance.
var exemptCommands = map[string]bool{
	"init":   true,
	"accept": true,
	"help":   true,
}

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gh pmu",
		Short: "GitHub Praxis Management Utility",
		Long: `gh pmu — GitHub Praxis Management Utility.

A GitHub CLI extension for project workflows, sub-issue hierarchies,
and batch operations. Designed for Kanban-style GitHub Projects with
status-based columns (Backlog, In Progress, In Review, Done).

Works seamlessly with the IDPF-Praxis framework for structured
development workflows, or standalone without any framework.

This extension combines and replaces:
  - gh-pm (https://github.com/yahsan2/gh-pm) - Project management
  - gh-sub-issue (https://github.com/yahsan2/gh-sub-issue) - Sub-issue hierarchy

Use 'gh pmu <command> --help' for more information about a command.`,
		Version: getVersion(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			runYAMLMigration(cmd)
			if err := checkAcceptance(cmd); err != nil {
				return err
			}
			return runDailyIntegrityCheck(cmd)
		},
	}

	cmd.SetVersionTemplate("{{.Use}} version {{.Version}}\nRubrical Works (c) 2026\n")

	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newViewCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newEditCommand())
	cmd.AddCommand(newCommentCommand())
	cmd.AddCommand(newMoveCommand())
	cmd.AddCommand(newCloseCommand())
	cmd.AddCommand(newBoardCommand())
	cmd.AddCommand(newSubCommand())
	cmd.AddCommand(newFieldCommand())
	cmd.AddCommand(newIntakeCommand())
	cmd.AddCommand(newTriageCommand())
	cmd.AddCommand(newSplitCommand())
	cmd.AddCommand(newHistoryCommand())
	cmd.AddCommand(newFilterCommand())
	cmd.AddCommand(newBranchCommand())
	cmd.AddCommand(newAcceptCommand())
	cmd.AddCommand(newLabelCommand())
	cmd.AddCommand(newConfigCommand())

	return cmd
}

func Execute() error {
	return NewRootCommand().Execute()
}

// runYAMLMigration performs a one-time migration to remove the legacy .gh-pmu.yml
// file when .gh-pmu.json exists, and updates the config version.
func runYAMLMigration(cmd *cobra.Command) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	jsonPath, err := config.FindConfigFile(cwd)
	if err != nil {
		return
	}

	// Only migrate if we found a JSON config (not a YAML fallback)
	if filepath.Ext(jsonPath) != ".json" {
		return
	}

	if err := config.MigrateYAML(jsonPath, getVersion(), cmd.ErrOrStderr()); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: YAML migration failed: %v\n", err)
	}
}

// checkAcceptance verifies terms have been accepted before running commands.
func checkAcceptance(cmd *cobra.Command) error {
	// Dev/source builds skip acceptance gate — only ldflags-injected builds enforce it
	if version == "" {
		return nil
	}

	// Check if this is an exempt command
	name := cmd.Name()
	if exemptCommands[name] {
		return nil
	}

	// --help flag on any command is always allowed
	if help, _ := cmd.Flags().GetBool("help"); help {
		return nil
	}

	// Try to load config — if no config exists, skip gate (init not run yet)
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	cfg, err := config.LoadFromDirectory(cwd)
	if err != nil {
		// No config file — not initialized yet, skip acceptance check
		return nil
	}

	// Check acceptance state
	if cfg.Acceptance == nil || !cfg.Acceptance.Accepted {
		printTermsAndHint(cmd)
		return fmt.Errorf("terms not accepted — run 'gh pmu accept' first")
	}

	// Check version — re-acceptance needed on major/minor bump
	if config.RequiresReAcceptance(cfg.Acceptance.Version, getVersion()) {
		printTermsAndHint(cmd)
		return fmt.Errorf("terms acceptance outdated (accepted v%s, current v%s) — run 'gh pmu accept' to re-accept",
			cfg.Acceptance.Version, getVersion())
	}

	return nil
}

// runDailyIntegrityCheck performs the throttled config integrity check.
// Runs silently if no config exists or if already checked today.
func runDailyIntegrityCheck(cmd *cobra.Command) error {
	// Skip for exempt commands
	name := cmd.Name()
	if exemptCommands[name] || name == "config" {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	configPath, err := config.FindConfigFile(cwd)
	if err != nil {
		return nil // No config — skip
	}

	return runIntegrityCheck(configPath, cmd.ErrOrStderr())
}

// printTermsAndHint writes the full terms text and acceptance hints to stderr.
func printTermsAndHint(cmd *cobra.Command) {
	w := cmd.ErrOrStderr()
	fmt.Fprintln(w)
	fmt.Fprintln(w, defaults.Terms())
	fmt.Fprintln(w, "Run 'gh pmu accept --yes' for non-interactive acceptance.")
	fmt.Fprintln(w, "Acceptance persists in .gh-pmu.yml (one-time per repo).")
	fmt.Fprintln(w)
}
