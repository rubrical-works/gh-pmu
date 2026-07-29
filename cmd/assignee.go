package cmd

// assigneeResolver is the slice of the API client that every --assignee-accepting
// command needs: @me becomes the authenticated login, and any other value is
// validated before it is used to assign or to match.
//
// The filter commands need this as much as the assignment ones. list routes
// through the Search API when a repository filter is available, where
// "assignee:@me" is valid syntax GitHub resolves server-side, and falls back to
// a literal comparison otherwise — so the same flag silently meant two different
// things depending on which path ran. Resolving before either consumes the value
// is what makes them agree.
type assigneeResolver interface {
	ResolveAssignee(value string) (string, error)
	ResolveAssignees(values []string) ([]string, error)
}
