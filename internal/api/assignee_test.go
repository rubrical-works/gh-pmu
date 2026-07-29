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
	userIDArgs  []string
}

// newAssigneeResolverMock registers viewerLogin as a known user alongside the
// named ones: the authenticated account is a real account, and the resolver
// looks up its node id after viewer{login} hands back the name.
func newAssigneeResolverMock(viewerLogin string, knownUsers ...string) *assigneeResolverMock {
	known := make(map[string]string, len(knownUsers)+1)
	for i, u := range knownUsers {
		known[u] = "U_kw" + string(rune('A'+i))
	}
	if viewerLogin != "" {
		known[viewerLogin] = "U_kwSELF"
	}
	return &assigneeResolverMock{
		viewerLogin: viewerLogin,
		knownUsers:  known,
		calls:       make(map[string]int),
	}
}

// userIDCalledWith reports whether a GetUserID query was issued for a login.
func (m *assigneeResolverMock) userIDCalledWith(login string) bool {
	for _, a := range m.userIDArgs {
		if a == login {
			return true
		}
	}
	return false
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
		arg := fmt.Sprintf("%v", variables["login"])
		m.userIDArgs = append(m.userIDArgs, arg)
		id, ok := m.knownUsers[arg]
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
	// The sentinel itself must never reach user(login:) — that lookup is what
	// returns NOT_FOUND today. Looking up the login it resolves to is expected:
	// viewer{login} yields no node id, and the assignment paths need one.
	if mock.userIDCalledWith(AssigneeSelf) {
		t.Error("@me was passed to user(login:) as a literal login")
	}
	if !mock.userIDCalledWith("rubrical-worker") {
		t.Error("expected the resolved login to be looked up for its node id")
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

func TestResolveAssignees_AllValidResolveInOrder(t *testing.T) {
	mock := newAssigneeResolverMock("rubrical-worker", "octocat", "hubot")
	client := NewClientWithGraphQL(mock)

	got, err := client.ResolveAssignees([]string{"octocat", AssigneeSelf, "hubot"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"octocat", "rubrical-worker", "hubot"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestResolveAssignees_PartialFailureNamesTheOffenders(t *testing.T) {
	// AC5: partial and total failures share exit 1, so the message is the only
	// thing telling them apart — it has to name which value broke.
	mock := newAssigneeResolverMock("rubrical-worker", "octocat", "hubot")
	client := NewClientWithGraphQL(mock)

	_, err := client.ResolveAssignees([]string{"octocat", "ghost", "hubot"})
	if err == nil {
		t.Fatal("expected an error when one of three assignees is unresolvable")
	}
	if !strings.Contains(err.Error(), "1 of 3") {
		t.Errorf("expected a partial-failure count, got: %v", err)
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("expected the failing login to be named, got: %v", err)
	}
}

func TestResolveAssignees_TotalFailureReportsAll(t *testing.T) {
	// AC5: mirrors subRemoveBatchError — "all N" rather than "N of N", so the
	// message does not imply some succeeded.
	mock := newAssigneeResolverMock("rubrical-worker", "octocat")
	client := NewClientWithGraphQL(mock)

	_, err := client.ResolveAssignees([]string{"ghost", "phantom"})
	if err == nil {
		t.Fatal("expected an error when every assignee is unresolvable")
	}
	if !strings.Contains(err.Error(), "all 2") {
		t.Errorf("expected a total-failure message, got: %v", err)
	}
}

func TestResolveAssignees_EmptyInputIsNotAnError(t *testing.T) {
	// Omitting --assignee is not a failure; it means "leave unassigned".
	mock := newAssigneeResolverMock("rubrical-worker")
	client := NewClientWithGraphQL(mock)

	got, err := client.ResolveAssignees(nil)
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no assignees, got %v", got)
	}
	if len(mock.calls) != 0 {
		t.Errorf("empty input must issue no queries, got %v", mock.calls)
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

	// Two distinct logins get an id lookup each — octocat, and the account @me
	// resolves to. Nine calls collapse to three queries; a fourth would mean the
	// cache is not holding.
	if mock.calls["GetUserID"] != 2 {
		t.Errorf("expected 2 GetUserID calls across 6 resolutions, got %d (args: %v)", mock.calls["GetUserID"], mock.userIDArgs)
	}
	if mock.calls["GetAuthenticatedUser"] != 1 {
		t.Errorf("expected 1 viewer lookup for 3 resolutions of @me, got %d", mock.calls["GetAuthenticatedUser"])
	}
}
