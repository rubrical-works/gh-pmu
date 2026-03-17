package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rubrical-works/gh-pmu/internal/integrity"
)

// runIntegrityCheck performs the daily config integrity check.
// It compares the local config against the committed version in git HEAD.
// Returns nil on success (even if drift is detected in non-strict mode).
// Returns an error only in strict mode when drift is detected.
// Writes warnings to w when drift is detected.
func runIntegrityCheck(configPath string, w io.Writer) error {
	configDir := filepath.Dir(configPath)
	configName := filepath.Base(configPath)

	// Check throttle — skip if already checked today
	throttled, err := integrity.IsThrottled(configDir)
	if err != nil {
		// Non-fatal: if we can't read throttle state, continue with check
		throttled = false
	}
	if throttled {
		return nil
	}

	// Read local config
	localContent, err := os.ReadFile(configPath)
	if err != nil {
		return nil // Can't read config — skip silently
	}

	// Read committed config via git show
	committedContent, err := gitShowFile(configDir, "HEAD:"+configName)
	if err != nil {
		// No committed version — skip check (might be new repo)
		return nil
	}

	// Compare
	result, err := integrity.CompareContent(localContent, committedContent)
	if err != nil {
		return nil // Comparison failed — skip silently
	}

	// Record that we performed the check today
	_ = integrity.RecordCheck(configDir)

	if !result.Drifted {
		return nil
	}

	// Drift detected — warn user
	fmt.Fprintf(w, "Warning: .gh-pmu config drift detected (local differs from HEAD):\n")
	for _, change := range result.Changes {
		fmt.Fprintf(w, "  • %s\n", change)
	}
	fmt.Fprintf(w, "Run 'gh pmu config verify' for full details.\n\n")

	// Strict mode: block command execution
	if isStrictMode(localContent) {
		return fmt.Errorf("config integrity check failed (strict mode) — resolve drift before continuing")
	}

	return nil
}
