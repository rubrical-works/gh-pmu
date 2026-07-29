package cmd

import "fmt"

// fakeAssigneeResolver satisfies assigneeResolver for command-level tests: @me
// becomes a fixed login, anything else passes through, and logins listed in
// unknown are rejected so the abort paths can be exercised.
//
// Embedded by value in the command mocks — the mocks are always used through a
// pointer, so these pointer-receiver methods promote.
type fakeAssigneeResolver struct {
	viewerLogin string
	unknown     map[string]bool
	resolved    []string
}

const defaultFakeViewerLogin = "mock-viewer"

func (f *fakeAssigneeResolver) ResolveAssignee(value string) (string, error) {
	f.resolved = append(f.resolved, value)

	if value == "@me" {
		if f.viewerLogin != "" {
			return f.viewerLogin, nil
		}
		return defaultFakeViewerLogin, nil
	}
	if f.unknown[value] {
		return "", fmt.Errorf("assignee %q could not be resolved", value)
	}
	return value, nil
}

func (f *fakeAssigneeResolver) ResolveAssignees(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		login, err := f.ResolveAssignee(v)
		if err != nil {
			return nil, err
		}
		out = append(out, login)
	}
	return out, nil
}
