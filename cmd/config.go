package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/rubrical-works/gh-pmu/internal/integrity"
	"github.com/spf13/cobra"
)

type configVerifyOptions struct {
	dir    string
	remote bool
}

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management commands",
	}

	cmd.AddCommand(newConfigVerifyCommand())

	return cmd
}

func newConfigVerifyCommand() *cobra.Command {
	opts := &configVerifyOptions{}

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify config integrity against git HEAD",
		Long: `Check .gh-pmu.json for unauthorized or accidental modifications.

Compares the local config against the last committed version (git HEAD)
and reports any differences. Optionally compares against origin/main.

In strict mode (configIntegrity: "strict" in .gh-pmu.json), returns
a non-zero exit code when drift is detected.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigVerify(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.dir, "dir", "", "Directory to search for config (default: current directory)")
	cmd.Flags().BoolVar(&opts.remote, "remote", false, "Also compare against origin/main")

	return cmd
}

func runConfigVerify(cmd *cobra.Command, opts *configVerifyOptions) error {
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
		return fmt.Errorf("no config file found: %w", err)
	}

	configDir := filepath.Dir(configPath)
	configName := filepath.Base(configPath)
	out := cmd.OutOrStdout()

	// Read local config
	localContent, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read local config: %w", err)
	}

	// Read committed config via git show
	committedContent, err := gitShowFile(configDir, "HEAD:"+configName)
	if err != nil {
		fmt.Fprintf(out, "Warning: could not read committed config: %v\n", err)
		committedContent = nil
	}

	// Compare local vs HEAD
	result, err := integrity.CompareContent(localContent, committedContent)
	if err != nil {
		return fmt.Errorf("comparison failed: %w", err)
	}

	// Checksum comparison — surface I/O errors to stderr rather than
	// silently discarding them. LoadChecksum already returns ("", nil) when
	// no checksum file is present, so this only fires for real failures.
	currentChecksum, err := integrity.ComputeChecksum(configPath)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not compute checksum: %v\n", err)
	}
	storedChecksum, err := integrity.LoadChecksum(configDir)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not read stored checksum: %v\n", err)
	}

	fmt.Fprintf(out, "Config: %s\n", configPath)
	fmt.Fprintf(out, "SHA-256: %s\n", currentChecksum)

	if storedChecksum != "" {
		if currentChecksum == storedChecksum {
			fmt.Fprintf(out, "Checksum: matches stored value\n")
		} else {
			fmt.Fprintf(out, "Checksum: MISMATCH (stored: %s)\n", storedChecksum)
		}
	}

	if !result.Drifted {
		fmt.Fprintf(out, "\nNo drift detected — local config matches HEAD.\n")
	} else {
		fmt.Fprintf(out, "\nDrift detected — local config differs from HEAD:\n")
		fmt.Fprintf(out, "  Changed:\n")
		for _, change := range result.Changes {
			fmt.Fprintf(out, "    • %s\n", change)
		}
		if len(result.Unchanged) > 0 {
			fmt.Fprintf(out, "  Unchanged:\n")
			for _, section := range result.Unchanged {
				fmt.Fprintf(out, "    - %s\n", section)
			}
		}
	}

	// Critical field check against HEAD
	var hasCriticalDrift bool
	if committedContent != nil {
		criticalChanges := compareCriticalFields(localContent, committedContent)
		if len(criticalChanges) > 0 {
			hasCriticalDrift = true
			writeCriticalAlert(cmd.ErrOrStderr(), criticalChanges)
		}
	}

	// Remote comparison
	if opts.remote {
		remoteContent, err := gitShowFile(configDir, "origin/main:"+configName)
		if err != nil {
			fmt.Fprintf(out, "\nRemote: could not read origin/main config: %v\n", err)
		} else {
			remoteResult, err := integrity.CompareContent(localContent, remoteContent)
			if err == nil {
				if !remoteResult.Drifted {
					fmt.Fprintf(out, "\nRemote: local config matches origin/main.\n")
				} else {
					fmt.Fprintf(out, "\nRemote: local config differs from origin/main:\n")
					fmt.Fprintf(out, "  Changed:\n")
					for _, change := range remoteResult.Changes {
						fmt.Fprintf(out, "    • %s\n", change)
					}
					if len(remoteResult.Unchanged) > 0 {
						fmt.Fprintf(out, "  Unchanged:\n")
						for _, section := range remoteResult.Unchanged {
							fmt.Fprintf(out, "    - %s\n", section)
						}
					}
				}

				// Critical field check against remote
				remoteCritical := compareCriticalFields(localContent, remoteContent)
				if len(remoteCritical) > 0 {
					hasCriticalDrift = true
					writeCriticalAlert(cmd.ErrOrStderr(), remoteCritical)
				}
			}
		}
	}

	// Strict mode check
	if (result.Drifted || hasCriticalDrift) && isStrictMode(localContent) {
		return fmt.Errorf("config integrity check failed (strict mode) — resolve drift before continuing")
	}

	return nil
}

// criticalFieldChange represents a change to a critical config field.
type criticalFieldChange struct {
	Field    string
	OldValue string
	NewValue string
}

// compareCriticalFields compares identity fields between local and reference configs.
// Returns nil if no critical fields changed.
func compareCriticalFields(local, reference []byte) []criticalFieldChange {
	type configShape struct {
		Project struct {
			Owner  string `json:"owner"`
			Number int    `json:"number"`
		} `json:"project"`
		Repositories []string `json:"repositories"`
	}

	var localCfg, refCfg configShape
	if err := json.Unmarshal(local, &localCfg); err != nil {
		return nil
	}
	if err := json.Unmarshal(reference, &refCfg); err != nil {
		return nil
	}

	var changes []criticalFieldChange

	if localCfg.Project.Owner != refCfg.Project.Owner {
		changes = append(changes, criticalFieldChange{
			Field:    "project.owner",
			OldValue: refCfg.Project.Owner,
			NewValue: localCfg.Project.Owner,
		})
	}
	if localCfg.Project.Number != refCfg.Project.Number {
		changes = append(changes, criticalFieldChange{
			Field:    "project.number",
			OldValue: fmt.Sprintf("%d", refCfg.Project.Number),
			NewValue: fmt.Sprintf("%d", localCfg.Project.Number),
		})
	}

	localRepo := ""
	if len(localCfg.Repositories) > 0 {
		localRepo = localCfg.Repositories[0]
	}
	refRepo := ""
	if len(refCfg.Repositories) > 0 {
		refRepo = refCfg.Repositories[0]
	}
	if localRepo != refRepo {
		changes = append(changes, criticalFieldChange{
			Field:    "repositories[0]",
			OldValue: refRepo,
			NewValue: localRepo,
		})
	}

	return changes
}

// writeCriticalAlert writes a boxed warning to the given writer for critical field changes.
func writeCriticalAlert(w io.Writer, changes []criticalFieldChange) {
	const width = 63
	border := strings.Repeat("─", width)

	fmt.Fprintf(w, "\n┌─%s─┐\n", border)
	fmt.Fprintf(w, "│  ⚠ CRITICAL CONFIG CHANGE DETECTED%s│\n", strings.Repeat(" ", width-35))
	fmt.Fprintf(w, "├─%s─┤\n", border)
	fmt.Fprintf(w, "│%s│\n", strings.Repeat(" ", width+2))
	fmt.Fprintf(w, "│  The following identity fields have changed from HEAD:%s│\n", strings.Repeat(" ", width-55))
	fmt.Fprintf(w, "│%s│\n", strings.Repeat(" ", width+2))

	for _, c := range changes {
		line := fmt.Sprintf("    %s:  %s  →  %s", c.Field, c.OldValue, c.NewValue)
		padding := width + 2 - len([]rune(line))
		if padding < 1 {
			padding = 1
		}
		fmt.Fprintf(w, "│%s%s│\n", line, strings.Repeat(" ", padding))
	}

	fmt.Fprintf(w, "│%s│\n", strings.Repeat(" ", width+2))
	fmt.Fprintf(w, "│  All gh pmu commands will now target the NEW values.%s│\n", strings.Repeat(" ", width-53))
	fmt.Fprintf(w, "│  If this is unintentional, restore with:%s│\n", strings.Repeat(" ", width-41))
	fmt.Fprintf(w, "│    git checkout -- .gh-pmu.json%s│\n", strings.Repeat(" ", width-31))
	fmt.Fprintf(w, "│%s│\n", strings.Repeat(" ", width+2))
	fmt.Fprintf(w, "└─%s─┘\n", border)
}

// gitShowFile runs git show to read a file from a given ref.
func gitShowFile(dir, ref string) ([]byte, error) {
	cmd := exec.Command("git", "show", ref)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s: %w", ref, err)
	}
	return out, nil
}

// isStrictMode checks if the config has configIntegrity set to "strict".
func isStrictMode(content []byte) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal(content, &raw); err != nil {
		return false
	}
	val, ok := raw["configIntegrity"]
	if !ok {
		return false
	}
	s, ok := val.(string)
	return ok && strings.EqualFold(s, "strict")
}
