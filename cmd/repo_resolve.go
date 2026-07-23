package cmd

import (
	"fmt"
	"strings"

	"github.com/rubrical-works/gh-pmu/internal/config"
)

// resolveRepoDefaults resolves the default owner/repo for a command: the --repo
// flag when provided, otherwise the first configured repository. Both sources are
// validated as "owner/repo" with non-empty components, so malformed values like
// "owner/" or "/repo" are rejected uniformly here instead of being deferred to
// the API layer with a less helpful error.
//
// This replaces the ~40-line "parse --repo or cfg.Repositories[0]" block that was
// copy-pasted across the sub, view, move, split, list, and label commands (with
// validation drift between copies).
func resolveRepoDefaults(cfg *config.Config, repoFlag string) (owner, repo string, err error) {
	if repoFlag != "" {
		return splitOwnerRepo(repoFlag, "--repo")
	}
	if cfg == nil || len(cfg.Repositories) == 0 {
		return "", "", fmt.Errorf("no repository specified and none configured (use --repo owner/repo or add one to .gh-pmu.json)")
	}
	return splitOwnerRepo(cfg.Repositories[0], "configured repository")
}

// splitOwnerRepo parses an "owner/repo" string and rejects empty components.
func splitOwnerRepo(s, source string) (string, string, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid %s format: expected owner/repo, got %q", source, s)
	}
	return parts[0], parts[1], nil
}

// applyRepoDefaults fills empty owner/repo components on a parsed issue reference
// from the resolved defaults. A reference that already carries both an owner and
// a repo is returned unchanged.
func applyRepoDefaults(refOwner, refRepo, defaultOwner, defaultRepo string) (string, string) {
	if refOwner == "" || refRepo == "" {
		return defaultOwner, defaultRepo
	}
	return refOwner, refRepo
}
