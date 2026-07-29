package api

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// assigneeResolverMock answers the two queries the resolver depends on:
// GetAuthenticatedUser (viewer{login}) and GetUserID (user(login:){id}). It
// counts calls per operation so memoization can be asserted directly rather
// than inferred.
type assigneeResolverMock struct {
	viewerLogin string
	viewerErr   error
	knownUsers  map[string]string
	calls       map[string]int
}

func newAssigneeResolverMock(viewerLogin string, knownUsers ...string) *assigneeResolverMock {
	known := make(map[string]string, len(knownUsers))
	for i, u := range knownUsers {
		known[u] = "U_kw" + string(rune('A'+i))
	}
	return &assigneeResolverMock{
		viewerLogin: viewerLogin,
		knownUsers:  known,
		calls:       make(map[string]int),
	}
}

func (m *assigneeResolverMock) Query(name string, query interface{}, variables map[string]interface{}) error {
	m.calls[name]++

	switch name {
	case "GetAuthenticatedUser":
		if m.viewerErr != nil {
			return m.viewerErr
		}
		reflect.ValueOf(query).Elem().FieldByName("Viewer").FieldByName("Login").SetString(m.viewerLogin)
		return nil
	case "GetUserID":
		id, ok := m.knownUsers[fmt.Sprintf("%v", variables["login"])]
		if !ok {
			// Mirrors GitHub: a missing user comes back as a null node, which
			// getUserID turns into `user %q not found`.
			return nil
		}
		reflect.ValueOf(query).Elem().FieldByName("User").FieldByName("ID").SetString(id)
		return nil
	}
	return nil
}

func (m *assigneeResolverMock) Mutate(name string, mutation interface{}, variables map[string]interface{}) error {
	m.calls[name]++
	return nil
}

func TestResolveAssignee_AtMeResolvesToAuthenticatedLogin(t *testing.T) {
	// AC1: @me is a sentinel, not a login. It must become whatever
	// `gh api user --jq .login` returns — the viewer{login} value.
	mock := newAssigneeResolverMock("rubrical-worker")
	client := NewClientWithGraphQL(mock)

	got, err := client.ResolveAssignee("@me")
	if err != nil {
		t.Fatalf("ResolveAssignee(@me) returned error: %v", err)
	}
	if got != "rubrical-worker" {
		t.Errorf("expected @me to resolve to %q, got %q", "rubrical-worker", got)
	}
	if mock.calls["GetUserID"] != 0 {
		t.Errorf("@me must not be validated as a literal login; GetUserID called %d time(s)", mock.calls["GetUserID"])
	}
}

func TestResolveAssignee_ValidLoginPassesThroughUnchanged(t *testing.T) {
	// AC2: an explicit login is used as given — but only once it is confirmed
	// to be a real account.
	mock := newAssigneeResolverMock("rubrical-worker", "octocat")
	client := NewClientWithGraphQL(mock)

	got, err := client.ResolveAssignee("octocat")
	if err != nil {
		t.Fatalf("ResolveAssignee(octocat) returned error: %v", err)
	}
	if got != "octocat" {
		t.Errorf("expected octocat to pass through unchanged, got %q", got)
	}
	if mock.calls["GetAuthenticatedUser"] != 0 {
		t.Errorf("an explicit login must not trigger a viewer lookup; called %d time(s)", mock.calls["GetAuthenticatedUser"])
	}
}

func TestResolveAssignee_UnknownLoginIsRejected(t *testing.T) {
	// AC3: the failure that previously warned and continued must now be an error
	// the caller cannot ignore.
	mock := newAssigneeResolverMock("rubrical-worker", "octocat")
	client := NewClientWithGraphQL(mock)

	_, err := client.ResolveAssignee("ghost")
	if err == nil {
		t.Fatal("expected an error for an unknown login, got nil")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("error must name the offending login, got: %v", err)
	}
}

func TestResolveAssignee_ViewerLookupFailurePropagates(t *testing.T) {
	// AC4: if @me cannot be resolved there is no safe fallback — the local git
	// identity is a different account entirely.
	mock := newAssigneeResolverMock("")
	mock.viewerErr = errors.New("network unreachable")
	client := NewClientWithGraphQL(mock)

	_, err := client.ResolveAssignee("@me")
	if err == nil {
		t.Fatal("expected an error when the viewer lookup fails, got nil")
	}
	if !strings.Contains(err.Error(), "@me") {
		t.Errorf("error must mention @me so the user knows which value failed, got: %v", err)
	}
}

func TestResolveAssignee_MemoizesPerDistinctValue(t *testing.T) {
	// AC8: a command may pass the same assignee several times (batch create,
	// from-file merge). One lookup per distinct value, not per occurrence.
	mock := newAssigneeResolverMock("rubrical-worker", "octocat")
	client := NewClientWithGraphQL(mock)

	for i := 0; i < 3; i++ {
		if _, err := client.ResolveAssignee("octocat"); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		if _, err := client.ResolveAssignee("@me"); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}

	if mock.calls["GetUserID"] != 1 {
		t.Errorf("expected 1 GetUserID call for 3 resolutions of the same login, got %d", mock.calls["GetUserID"])
	}
	if mock.calls["GetAuthenticatedUser"] != 1 {
		t.Errorf("expected 1 viewer lookup for 3 resolutions of @me, got %d", mock.calls["GetAuthenticatedUser"])
	}
}
