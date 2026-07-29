package api

import (
	"fmt"
	"strings"
)

// AssigneeSelf is the sentinel callers pass to mean "whoever is authenticated",
// matching the gh CLI's own convention. It is not a login: GitHub's
// user(login:) resolver returns NOT_FOUND for it.
const AssigneeSelf = "@me"

// ResolveAssignee turns one --assignee value into a usable GitHub login.
//
// AssigneeSelf becomes the authenticated account's login — the same value
// `gh api user --jq .login` returns. Any other value passes through unchanged,
// but only once it is confirmed to name a real account. Validating rather than
// trusting matters because createIssue accepts an empty assignee set without
// complaint, so an unresolvable login otherwise yields a silently unassigned
// issue.
//
// Results are memoized per client. A single command can mention the same
// assignee several times — batch creation, or explicit values merged with
// --from-file entries — and each distinct value costs one lookup, not one per
// occurrence. Only successes are cached; a failure aborts the command, so there
// is no second attempt to serve from cache.
func (c *Client) ResolveAssignee(value string) (string, error) {
	if cached, ok := c.assigneeCache[value]; ok {
		return cached, nil
	}

	var resolved string
	if value == AssigneeSelf {
		login, err := c.GetAuthenticatedUser()
		if err != nil {
			return "", fmt.Errorf("cannot resolve %s: %w", AssigneeSelf, err)
		}
		if login == "" {
			return "", fmt.Errorf("cannot resolve %s: authenticated user has no login", AssigneeSelf)
		}
		resolved = login
	} else {
		if _, err := c.getUserID(value); err != nil {
			return "", fmt.Errorf("assignee %q could not be resolved: %w", value, err)
		}
		resolved = value
	}

	if c.assigneeCache == nil {
		c.assigneeCache = make(map[string]string)
	}
	c.assigneeCache[value] = resolved
	return resolved, nil
}

// ResolveAssignees resolves a whole --assignee set, returning the logins in the
// order supplied. Any unresolvable value fails the batch: a command that cannot
// honour every assignee it was given should not proceed with a subset, because
// the result would look successful while quietly differing from what was asked.
//
// Nothing is resolved for an empty set — omitting --assignee means "leave
// unassigned", which is not a failure.
func (c *Client) ResolveAssignees(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	resolved := make([]string, 0, len(values))
	var failed []string
	for _, v := range values {
		login, err := c.ResolveAssignee(v)
		if err != nil {
			failed = append(failed, v)
			continue
		}
		resolved = append(resolved, login)
	}

	if err := assigneeBatchError(len(failed), len(values), failed); err != nil {
		return nil, err
	}
	return resolved, nil
}

// assigneeBatchError phrases a batch resolution failure, mirroring
// subRemoveBatchError: "all N" when nothing resolved, "N of M" otherwise, so the
// message never implies partial success where there was none. Both cases exit 1
// — the wording is what distinguishes them.
func assigneeBatchError(failCount, total int, failed []string) error {
	if failCount == 0 {
		return nil
	}
	if failCount == total {
		return fmt.Errorf("all %d assignees could not be resolved: %s", total, strings.Join(failed, ", "))
	}
	return fmt.Errorf("%d of %d assignees could not be resolved: %s", failCount, total, strings.Join(failed, ", "))
}
