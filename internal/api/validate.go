package api

import (
	"fmt"
	"regexp"
)

// Input validators for values interpolated into raw GraphQL queries.
// These reject any input that could break out of a quoted GraphQL string or
// that would be escaped differently by Go's %q verb vs the GraphQL spec.
//
// See https://docs.github.com/en/graphql for server-side constraints:
//   - owner: GitHub usernames are up to 39 chars, [a-zA-Z0-9] or hyphens (no
//     leading/trailing hyphen).
//   - repo:  up to 100 chars, [a-zA-Z0-9._-].
//   - label: up to 50 chars; GitHub permits most printable characters but we
//     additionally reject quotes, backslashes and control characters so the
//     result is safe to embed in a Go-literal-quoted GraphQL string.
//   - nodeID: opaque GraphQL node identifiers ([A-Za-z0-9_=-]).
var (
	validOwner   = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)
	validRepo    = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
	validNodeID  = regexp.MustCompile(`^[A-Za-z0-9_=-]+$`)
	labelBadRune = regexp.MustCompile(`["\\]`)
)

// validateOwner validates a GitHub owner (user or organization) name.
func validateOwner(owner string) error {
	if owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if !validOwner.MatchString(owner) {
		return fmt.Errorf("owner contains invalid characters: %q", owner)
	}
	return nil
}

// validateRepo validates a GitHub repository name.
func validateRepo(repo string) error {
	if repo == "" {
		return fmt.Errorf("repo cannot be empty")
	}
	if !validRepo.MatchString(repo) {
		return fmt.Errorf("repo contains invalid characters: %q", repo)
	}
	return nil
}

// validateLabelName validates a GitHub label name for safe interpolation into
// a quoted GraphQL string literal.
func validateLabelName(name string) error {
	if name == "" {
		return fmt.Errorf("label name cannot be empty")
	}
	if len(name) > 50 {
		return fmt.Errorf("label name exceeds 50 characters: %q", name)
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("label name contains control character: %q", name)
		}
	}
	if labelBadRune.MatchString(name) {
		return fmt.Errorf("label name contains unsafe character (\" or \\): %q", name)
	}
	return nil
}

// validateNodeID validates an opaque GraphQL node ID.
func validateNodeID(id string) error {
	if id == "" {
		return fmt.Errorf("node ID cannot be empty")
	}
	if !validNodeID.MatchString(id) {
		return fmt.Errorf("node ID contains invalid characters: %q", id)
	}
	return nil
}

// validateOwnerRepo validates both halves of an owner/repo pair.
func validateOwnerRepo(owner, repo string) error {
	if err := validateOwner(owner); err != nil {
		return err
	}
	return validateRepo(repo)
}

// validateLabelNames validates a slice of label names, returning the first
// error encountered with the offending label reported via its index in the
// message so callers can locate it.
func validateLabelNames(names []string) error {
	for i, n := range names {
		if err := validateLabelName(n); err != nil {
			return fmt.Errorf("labels[%d]: %w", i, err)
		}
	}
	return nil
}

