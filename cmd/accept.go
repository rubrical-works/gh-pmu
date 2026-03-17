package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/rubrical-works/gh-pmu/internal/defaults"
	"github.com/rubrical-works/gh-pmu/internal/integrity"
	"github.com/spf13/cobra"
)

type acceptOptions struct {
	yes bool
	dir string
}

func newAcceptCommand() *cobra.Command {
	opts := &acceptOptions{}

	cmd := &cobra.Command{
		Use:   "accept",
		Short: "Accept terms and conditions",
		Long: `Accept the Praxis Management Utility (gh-pmu) terms and conditions for this repository.

Terms must be accepted before using gh-pmu commands. Acceptance is
recorded in .gh-pmu.yml and covers all repository collaborators.

Use --yes to accept non-interactively (e.g., from Claude Code).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAccept(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Accept terms without interactive prompt")
	cmd.Flags().StringVar(&opts.dir, "dir", "", "Directory to search for config (default: current directory)")

	return cmd
}

func runAccept(cmd *cobra.Command, opts *acceptOptions) error {
	dir := opts.dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	configPath, err := config.FindConfigFile(dir)
	if err != nil {
		return fmt.Errorf("no .gh-pmu.yml found: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	out := cmd.OutOrStdout()

	// Display terms
	fmt.Fprintln(out)
	fmt.Fprintln(out, defaults.Terms())

	// Shared acceptance notice
	fmt.Fprintln(out, "NOTE: By accepting, you accept on behalf of all users and")
	fmt.Fprintln(out, "collaborators of this repository's gh-pmu configuration.")
	fmt.Fprintln(out)

	if !opts.yes {
		// Interactive prompt
		fmt.Fprintf(out, "Do you accept these terms? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			return fmt.Errorf("terms declined — gh-pmu commands are unavailable until terms are accepted")
		}
	}

	// Record acceptance
	user := detectGitUser()
	cfg.Acceptance = &config.Acceptance{
		Accepted: true,
		User:     user,
		Date:     time.Now().Format("2006-01-02"),
		Version:  getVersion(),
	}

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save acceptance: %w", err)
	}

	// Update checksum after saving config
	if err := integrity.UpdateChecksumForConfig(configPath); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not update checksum: %v\n", err)
	}

	fmt.Fprintf(out, "Terms accepted by %s (v%s).\n", user, getVersion())
	return nil
}

// detectGitUser returns the git user name, or "unknown" if not configured.
func detectGitUser() string {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return "unknown"
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "unknown"
	}
	return name
}
