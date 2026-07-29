package api

import (
	"fmt"
	"strings"

	graphql "github.com/cli/shurcooL-graphql"
)

// AssigneeSelf is the sentinel callers pass to mean "whoever is authenticated",
// matching the gh CLI's own convention. It is not a login: GitHub's
// user(login:) resolver returns NOT_FOUND for it.
const AssigneeSelf = "@me"

// resolvedAssignee is one memoized resolution: the login a value resolves to and
// that account's node id.
type resolvedAssignee struct {
	login string
	id    string
}

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
	login, _, err := c.resolveAssignee(value)
	return login, err
}

// resolveAssignee resolves a value to both its login and its account node id.
//
// The id is carried alongside the login because the assignment paths need it for
// createIssue, and fetching it separately would mean a second lookup for a value
// the validation step already resolved. @me costs two queries the first time —
// viewer{login} does not return an id — but the resolved login is cached under
// its own key as well, so `--assignee @me --assignee <that same login>` still
// totals two, not three.
func (c *Client) resolveAssignee(value string) (login, id string, err error) {
	if cached, ok := c.assigneeCache[value]; ok {
		return cached.login, cached.id, nil
	}

	if value == AssigneeSelf {
		self, err := c.GetAuthenticatedUser()
		if err != nil {
			return "", "", fmt.Errorf("cannot resolve %s: %w", AssigneeSelf, err)
		}
		if self == "" {
			return "", "", fmt.Errorf("cannot resolve %s: authenticated user has no login", AssigneeSelf)
		}
		// Recurses once: self is a real login, so this takes the branch below.
		resolvedLogin, resolvedID, err := c.resolveAssignee(self)
		if err != nil {
			return "", "", fmt.Errorf("cannot resolve %s: %w", AssigneeSelf, err)
		}
		c.cacheAssignee(value, resolvedLogin, resolvedID)
		return resolvedLogin, resolvedID, nil
	}

	userID, err := c.getUserID(value)
	if err != nil {
		return "", "", fmt.Errorf("assignee %q could not be resolved: %w", value, err)
	}
	c.cacheAssignee(value, value, userID)
	return value, userID, nil
}

func (c *Client) cacheAssignee(key, login, id string) {
	if c.assigneeCache == nil {
		c.assigneeCache = make(map[string]resolvedAssignee)
	}
	c.assigneeCache[key] = resolvedAssignee{login: login, id: id}
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

// resolveAssigneeIDs resolves a --assignee set to the account node ids the
// createIssue mutation expects. Same all-or-nothing contract as
// ResolveAssignees: the ids are only returned if every value resolved.
func (c *Client) resolveAssigneeIDs(values []string) ([]graphql.ID, error) {
	if len(values) == 0 {
		return nil, nil
	}

	ids := make([]graphql.ID, 0, len(values))
	var failed []string
	for _, v := range values {
		_, id, err := c.resolveAssignee(v)
		if err != nil {
			failed = append(failed, v)
			continue
		}
		ids = append(ids, graphql.ID(id))
	}

	if err := assigneeBatchError(len(failed), len(values), failed); err != nil {
		return nil, err
	}
	return ids, nil
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
