package cmd

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// Command-level scenario tests (#884).
//
// These drive the real command stack — run* -> config -> api.Client -> real
// shurcooL GraphQL plumbing -> redirectTransport -> mock GraphQL server — and
// assert on rendered output. That is the layer the interface-level mocks in
// list_test.go / view_test.go cannot reach: those substitute the client, so
// they never exercise query construction or response decoding.
//
// The wrapper tests in wrapper_test.go configure no responses and assert only
// "not a config error", so they pass against an empty mock. Everything here
// feeds realistic fixtures and asserts on content.
//
// No live GitHub API calls: setupTestEnvironment installs a redirectTransport
// pointing at an httptest server (#884 AC5).

// ---------------------------------------------------------------------------
// Fixture builders
// ---------------------------------------------------------------------------

// gqlData wraps a payload in the GraphQL {"data": ...} envelope.
func gqlData(payload map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"data": payload}
}

// userProjectFixture is the GetUserProject response. setupTestEnvironment
// writes a config with owner "test-org" / number 1, and GetProject tries the
// user projectV2 first, so this short-circuits before getOrgProject.
func userProjectFixture(projectID, title string) map[string]interface{} {
	return gqlData(map[string]interface{}{
		"user": map[string]interface{}{
			"projectV2": map[string]interface{}{
				"id":     projectID,
				"number": 1,
				"title":  title,
				"url":    "https://github.com/orgs/test-org/projects/1",
				"closed": false,
			},
		},
	})
}

// issueNode builds one node of a SearchIssues response. The inline fragment
// (`... on Issue`) flattens into the node object, so these fields sit
// alongside __typename rather than nested under it.
func issueNode(number int, title, state string, labels, assignees []string) map[string]interface{} {
	labelNodes := make([]interface{}, 0, len(labels))
	for _, l := range labels {
		labelNodes = append(labelNodes, map[string]interface{}{"name": l, "color": "ededed"})
	}
	assigneeNodes := make([]interface{}, 0, len(assignees))
	for _, a := range assignees {
		assigneeNodes = append(assigneeNodes, map[string]interface{}{"login": a})
	}

	return map[string]interface{}{
		"__typename": "Issue",
		"id":         "issue-node-id",
		"number":     number,
		"title":      title,
		"body":       "",
		"state":      state,
		"url":        "https://github.com/test-org/test-repo/issues/" + strconv.Itoa(number),
		"repository": map[string]interface{}{"nameWithOwner": "test-org/test-repo"},
		"author":     map[string]interface{}{"login": "author-login"},
		"assignees":  map[string]interface{}{"nodes": assigneeNodes},
		"labels":     map[string]interface{}{"nodes": labelNodes},
		"milestone":  nil,
	}
}

// searchIssuesFixture is the SearchIssues response carrying the given nodes.
func searchIssuesFixture(nodes ...map[string]interface{}) map[string]interface{} {
	asAny := make([]interface{}, 0, len(nodes))
	for _, n := range nodes {
		asAny = append(asAny, n)
	}
	return gqlData(map[string]interface{}{
		"search": map[string]interface{}{
			"nodes":    asAny,
			"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
		},
	})
}

// projectItemNode builds one node of a GetProjectItems response. Both the
// content and fieldValues inline fragments flatten into their node objects.
func projectItemNode(itemID string, number int, title, state, status string) map[string]interface{} {
	fieldValues := []interface{}{}
	if status != "" {
		fieldValues = append(fieldValues, map[string]interface{}{
			"__typename": "ProjectV2ItemFieldSingleSelectValue",
			"name":       status,
			"field":      map[string]interface{}{"name": "Status"},
		})
	}

	return map[string]interface{}{
		"id": itemID,
		"content": map[string]interface{}{
			"__typename": "Issue",
			"id":         "issue-" + strconv.Itoa(number),
			"number":     number,
			"title":      title,
			"body":       "",
			"state":      state,
			"url":        "https://github.com/test-org/test-repo/issues/" + strconv.Itoa(number),
			"repository": map[string]interface{}{"nameWithOwner": "test-org/test-repo"},
			"assignees":  map[string]interface{}{"nodes": []interface{}{}},
			"labels":     map[string]interface{}{"nodes": []interface{}{}},
		},
		"fieldValues": map[string]interface{}{"nodes": fieldValues},
	}
}

// projectItemsFixture is the GetProjectItems response. The `... on ProjectV2`
// inline fragment flattens, so items sits directly under node.
func projectItemsFixture(nodes ...map[string]interface{}) map[string]interface{} {
	asAny := make([]interface{}, 0, len(nodes))
	for _, n := range nodes {
		asAny = append(asAny, n)
	}
	return gqlData(map[string]interface{}{
		"node": map[string]interface{}{
			"items": map[string]interface{}{
				"nodes":    asAny,
				"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
			},
		},
	})
}

// batchIssueLookupFixture is the response to the aliased batch document that
// move issues to resolve issues and their project items:
//
//	query { r0: repository(...) { i0_0: issue(number: 42) { ... } } }
//
// It carries no operation name, so scenarios must register it with
// respondToQueryContaining.
func batchIssueLookupFixture(number int, title, state, projectID, itemID, status string) map[string]interface{} {
	fieldValues := []interface{}{}
	if status != "" {
		fieldValues = append(fieldValues, map[string]interface{}{
			"__typename": "ProjectV2ItemFieldSingleSelectValue",
			"name":       status,
			"field":      map[string]interface{}{"name": "Status"},
		})
	}

	return gqlData(map[string]interface{}{
		"r0": map[string]interface{}{
			"i0_0": map[string]interface{}{
				"id":         "issue-" + strconv.Itoa(number),
				"number":     number,
				"title":      title,
				"body":       "",
				"state":      state,
				"url":        "https://github.com/test-org/test-repo/issues/" + strconv.Itoa(number),
				"repository": map[string]interface{}{"nameWithOwner": "test-org/test-repo"},
				"assignees":  map[string]interface{}{"nodes": []interface{}{}},
				"labels":     map[string]interface{}{"nodes": []interface{}{}},
				"projectItems": map[string]interface{}{"nodes": []interface{}{
					map[string]interface{}{
						"id":          itemID,
						"project":     map[string]interface{}{"id": projectID},
						"fieldValues": map[string]interface{}{"nodes": fieldValues},
					},
				}},
			},
		},
	})
}

// projectFieldsFixture is the GetProjectFields response, giving move the
// Status field id and its selectable options.
func projectFieldsFixture() map[string]interface{} {
	return gqlData(map[string]interface{}{
		"node": map[string]interface{}{
			"fields": map[string]interface{}{
				"nodes": []interface{}{
					map[string]interface{}{
						"__typename": "ProjectV2SingleSelectField",
						"id":         "field-status",
						"name":       "Status",
						"dataType":   "SINGLE_SELECT",
						"options": []interface{}{
							map[string]interface{}{"id": "opt-backlog", "name": "Backlog"},
							map[string]interface{}{"id": "opt-inprog", "name": "In Progress"},
							map[string]interface{}{"id": "opt-done", "name": "Done"},
						},
					},
				},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// No-live-traffic guard (#884 AC5)
// ---------------------------------------------------------------------------

// TestScenario_AllTrafficReachesMockServer is the empirical counterpart to the
// structural guarantee that setupTestEnvironment installs a redirectTransport:
// it proves the command's requests actually arrived at the mock rather than
// leaving the process. If the transport hook regressed, the command would talk
// to api.github.com and the handler would record nothing.
func TestScenario_AllTrafficReachesMockServer(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetUserProject", userProjectFixture("proj-1", "Test Project"))
	handler.respondTo("SearchIssues", searchIssuesFixture(
		issueNode(101, "Tracked", "OPEN", nil, nil),
	))

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := runList(cmd, &listOptions{}); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	if len(handler.operationNames()) == 0 {
		t.Fatal("mock server received no requests — traffic escaped the test transport")
	}
	// A successful render proves the response came from the mock: no live
	// endpoint would return this fixture's data for project test-org/1.
	if !strings.Contains(buf.String(), "Tracked") {
		t.Errorf("expected the mock fixture to drive output, got:\n%s", buf.String())
	}
}

// ---------------------------------------------------------------------------
// Harness self-tests
// ---------------------------------------------------------------------------

func TestGraphQLOperationName(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"query GetUserProject($number:Int!$owner:String!){user{id}}", "GetUserProject"},
		{"mutation CreateIssue($input:CreateIssueInput!){createIssue{id}}", "CreateIssue"},
		{"  query   GetIssue($n:Int!){x}", "GetIssue"},
		{"query { n0: node(id: \"abc\") { id } }", ""}, // anonymous batch query
		{"{ viewer { login } }", ""},                   // shorthand
		{"", ""},
	}

	for _, tt := range tests {
		if got := graphQLOperationName(tt.query); got != tt.want {
			t.Errorf("graphQLOperationName(%q) = %q, want %q", tt.query, got, tt.want)
		}
	}
}

// TestMockHandler_ExactOperationMatch pins the reason matching is exact rather
// than substring: GetIssue is a prefix of GetIssueComments, so a substring
// matcher could serve the wrong fixture depending on map iteration order.
func TestMockHandler_ExactOperationMatch(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetIssue", gqlData(map[string]interface{}{"marker": "issue"}))
	handler.respondTo("GetIssueComments", gqlData(map[string]interface{}{"marker": "comments"}))

	got := decodeHandlerResponse(t, handler, "query GetIssueComments($n:Int!){x}")
	if got["marker"] != "comments" {
		t.Errorf("expected GetIssueComments fixture, got %v", got)
	}

	got = decodeHandlerResponse(t, handler, "query GetIssue($n:Int!){x}")
	if got["marker"] != "issue" {
		t.Errorf("expected GetIssue fixture, got %v", got)
	}
}

func TestMockHandler_SubstringMatcherForAnonymousQuery(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetIssue", gqlData(map[string]interface{}{"marker": "named"}))
	handler.respondToQueryContaining("n0: node(id:", gqlData(map[string]interface{}{"marker": "batch"}))

	got := decodeHandlerResponse(t, handler, `query { n0: node(id: "abc") { id } }`)
	if got["marker"] != "batch" {
		t.Errorf("expected anonymous batch fixture, got %v", got)
	}
}

func TestMockHandler_FallsBackToDefaultResponse(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.defaultResponse = gqlData(map[string]interface{}{"marker": "default"})

	got := decodeHandlerResponse(t, handler, "query SomethingUnregistered($n:Int!){x}")
	if got["marker"] != "default" {
		t.Errorf("expected default fixture, got %v", got)
	}
}

// decodeHandlerResponse posts a query at the handler and returns the decoded
// "data" object.
func decodeHandlerResponse(t *testing.T, h *mockGraphQLHandler, query string) map[string]interface{} {
	t.Helper()

	body, err := json.Marshal(graphQLRequest{Query: query})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(body))
	h.ServeHTTP(rec, req)

	var envelope map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to decode handler response: %v (body %s)", err, rec.Body.String())
	}
	data, ok := envelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("response has no data object: %s", rec.Body.String())
	}
	return data
}

// ---------------------------------------------------------------------------
// list scenarios
// ---------------------------------------------------------------------------

// TestScenario_List_RendersIssuesFromProject drives runList through the real
// GraphQL stack and asserts the rendered table contains the issues the server
// returned. The wrapper-test equivalent (TestRunList_LoadsConfig) passes
// against an empty mock and asserts nothing about output.
func TestScenario_List_RendersIssuesFromProject(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetUserProject", userProjectFixture("proj-1", "Test Project"))
	handler.respondTo("SearchIssues", searchIssuesFixture(
		issueNode(101, "First tracked issue", "OPEN", []string{"bug"}, []string{"alice"}),
		issueNode(102, "Second tracked issue", "OPEN", []string{"enhancement"}, nil),
	))

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := runList(cmd, &listOptions{}); err != nil {
		t.Fatalf("runList() error = %v (output %q)", err, buf.String())
	}

	out := buf.String()
	if !strings.Contains(out, "First tracked issue") {
		t.Errorf("expected output to contain first issue title, got:\n%s", out)
	}
	if !strings.Contains(out, "Second tracked issue") {
		t.Errorf("expected output to contain second issue title, got:\n%s", out)
	}
	if strings.Contains(out, "No issues found") {
		t.Errorf("expected rendered issues, got the empty-state message:\n%s", out)
	}

	// The command must have gone through the project lookup before searching.
	ops := handler.operationNames()
	if len(ops) < 2 || ops[0] != "GetUserProject" {
		t.Errorf("expected GetUserProject first, got %v", ops)
	}
}

// TestScenario_List_EmptyProjectRendersEmptyState is the counterpart: with the
// same wiring but no issues, the command reports the empty state rather than
// erroring.
func TestScenario_List_EmptyProjectRendersEmptyState(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetUserProject", userProjectFixture("proj-1", "Test Project"))
	handler.respondTo("SearchIssues", searchIssuesFixture())

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := runList(cmd, &listOptions{}); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	if !strings.Contains(buf.String(), "No issues found") {
		t.Errorf("expected empty-state message, got:\n%s", buf.String())
	}
}

// TestScenario_List_SearchQueryCarriesRepoAndState asserts the query string the
// command sends upstream — the part interface-level mocks skip entirely.
func TestScenario_List_SearchQueryCarriesRepoAndState(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetUserProject", userProjectFixture("proj-1", "Test Project"))
	handler.respondTo("SearchIssues", searchIssuesFixture(
		issueNode(101, "Tracked", "OPEN", nil, nil),
	))

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := runList(cmd, &listOptions{}); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	reqs := handler.requestsFor("SearchIssues")
	if len(reqs) == 0 {
		t.Fatal("expected a SearchIssues request")
	}
	q, _ := reqs[0].Variables["query"].(string)
	if !strings.Contains(q, "repo:test-org/test-repo") {
		t.Errorf("expected search query scoped to the configured repo, got %q", q)
	}
	if !strings.Contains(q, "is:issue") {
		t.Errorf("expected search query restricted to issues, got %q", q)
	}
}

// ---------------------------------------------------------------------------
// view scenarios
// ---------------------------------------------------------------------------

// issueWithProjectFieldsFixture is the GetIssueWithProjectFields response.
// status/priority arrive as project single-select field values.
func issueWithProjectFieldsFixture(number int, title, body, state, status string) map[string]interface{} {
	fieldValues := []interface{}{
		map[string]interface{}{
			"__typename": "ProjectV2ItemFieldSingleSelectValue",
			"name":       status,
			"field":      map[string]interface{}{"name": "Status"},
		},
	}

	return gqlData(map[string]interface{}{
		"repository": map[string]interface{}{
			"issue": map[string]interface{}{
				"id":        "issue-node-id",
				"number":    number,
				"title":     title,
				"body":      body,
				"state":     state,
				"url":       "https://github.com/test-org/test-repo/issues/" + strconv.Itoa(number),
				"author":    map[string]interface{}{"login": "author-login"},
				"assignees": map[string]interface{}{"nodes": []interface{}{map[string]interface{}{"login": "alice"}}},
				"labels": map[string]interface{}{"nodes": []interface{}{
					map[string]interface{}{"name": "bug", "color": "d73a4a"},
				}},
				"milestone": nil,
				"projectItems": map[string]interface{}{"nodes": []interface{}{
					map[string]interface{}{"fieldValues": map[string]interface{}{"nodes": fieldValues}},
				}},
			},
		},
	})
}

// noParentFixture / noSubIssuesFixture are the "issue stands alone" responses
// that view issues after the main lookup.
func noParentFixture() map[string]interface{} {
	return gqlData(map[string]interface{}{
		"repository": map[string]interface{}{
			"issue": map[string]interface{}{"parent": nil},
		},
	})
}

func subIssuesFixture(nodes ...map[string]interface{}) map[string]interface{} {
	asAny := make([]interface{}, 0, len(nodes))
	for _, n := range nodes {
		asAny = append(asAny, n)
	}
	return gqlData(map[string]interface{}{
		"repository": map[string]interface{}{
			"issue": map[string]interface{}{
				"subIssues": map[string]interface{}{
					"nodes":    asAny,
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
				},
			},
		},
	})
}

func subIssueNode(number int, title, state string) map[string]interface{} {
	return map[string]interface{}{
		"id":     "sub-" + strconv.Itoa(number),
		"number": number,
		"title":  title,
		"state":  state,
		"url":    "https://github.com/test-org/test-repo/issues/" + strconv.Itoa(number),
		"repository": map[string]interface{}{
			"name":  "test-repo",
			"owner": map[string]interface{}{"login": "test-org"},
		},
	}
}

// TestScenario_View_RendersIssueDetail drives runView through the real stack.
// The wrapper equivalent (TestRunView_LoadsConfig) renders " #0 / State: /
// URL:" against an empty mock and passes.
func TestScenario_View_RendersIssueDetail(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetIssueWithProjectFields",
		issueWithProjectFieldsFixture(42, "Scenario issue title", "Scenario issue body", "OPEN", "In Progress"))
	handler.respondTo("GetParentIssue", noParentFixture())
	handler.respondTo("GetSubIssues", subIssuesFixture())

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newViewCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := runView(cmd, []string{"42"}, &viewOptions{}); err != nil {
		t.Fatalf("runView() error = %v (output %q)", err, buf.String())
	}

	out := buf.String()
	for _, want := range []string{"Scenario issue title", "#42", "Scenario issue body", "OPEN"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected view output to contain %q, got:\n%s", want, out)
		}
	}
	// Project field values must be resolved from projectItems, not dropped.
	if !strings.Contains(out, "In Progress") {
		t.Errorf("expected view output to show the Status field value, got:\n%s", out)
	}
}

// TestScenario_View_JSONOutputDecodes asserts the --json path emits decodable
// JSON carrying the fetched issue.
func TestScenario_View_JSONOutputDecodes(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetIssueWithProjectFields",
		issueWithProjectFieldsFixture(42, "JSON scenario", "body text", "OPEN", "Backlog"))
	handler.respondTo("GetParentIssue", noParentFixture())
	handler.respondTo("GetSubIssues", subIssuesFixture())

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newViewCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := runView(cmd, []string{"42"}, &viewOptions{jsonFields: "number,title,state"}); err != nil {
		t.Fatalf("runView() error = %v (output %q)", err, buf.String())
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("expected decodable JSON, got error %v for output:\n%s", err, buf.String())
	}
	if int(decoded["number"].(float64)) != 42 {
		t.Errorf("expected number 42, got %v", decoded["number"])
	}
	if decoded["title"] != "JSON scenario" {
		t.Errorf("expected title 'JSON scenario', got %v", decoded["title"])
	}
}

// TestScenario_View_RendersSubIssueProgress covers the parent path: sub-issues
// present, progress derived from their state.
func TestScenario_View_RendersSubIssueProgress(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetIssueWithProjectFields",
		issueWithProjectFieldsFixture(10, "Parent epic", "epic body", "OPEN", "In Progress"))
	handler.respondTo("GetParentIssue", noParentFixture())
	handler.respondTo("GetSubIssues", subIssuesFixture(
		subIssueNode(11, "Done child", "CLOSED"),
		subIssueNode(12, "Open child", "OPEN"),
	))

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newViewCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := runView(cmd, []string{"10"}, &viewOptions{}); err != nil {
		t.Fatalf("runView() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Done child") || !strings.Contains(out, "Open child") {
		t.Errorf("expected both sub-issues listed, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// sub lifecycle scenarios
// ---------------------------------------------------------------------------

func issueFixture(number int, title, state string) map[string]interface{} {
	return gqlData(map[string]interface{}{
		"repository": map[string]interface{}{
			"issue": map[string]interface{}{
				"id":        "issue-" + strconv.Itoa(number),
				"number":    number,
				"title":     title,
				"body":      "",
				"state":     state,
				"url":       "https://github.com/test-org/test-repo/issues/" + strconv.Itoa(number),
				"author":    map[string]interface{}{"login": "author-login"},
				"assignees": map[string]interface{}{"nodes": []interface{}{}},
				"labels":    map[string]interface{}{"nodes": []interface{}{}},
				"milestone": nil,
			},
		},
	})
}

// TestScenario_SubList_RendersChildren drives runSubList end to end. Note this
// relies on exact operation matching: the flow issues GetIssue *and*
// GetSubIssues, and GetIssue is a prefix of other operation names.
func TestScenario_SubList_RendersChildren(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetIssue", issueFixture(10, "Parent epic", "OPEN"))
	handler.respondTo("GetSubIssues", subIssuesFixture(
		subIssueNode(11, "First child", "CLOSED"),
		subIssueNode(12, "Second child", "OPEN"),
	))

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newSubListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &subListOptions{state: "all", relation: "children"}
	if err := runSubList(cmd, []string{"10"}, opts); err != nil {
		t.Fatalf("runSubList() error = %v (output %q)", err, buf.String())
	}

	out := buf.String()
	if !strings.Contains(out, "Parent epic") {
		t.Errorf("expected parent title in output, got:\n%s", out)
	}
	if !strings.Contains(out, "First child") || !strings.Contains(out, "Second child") {
		t.Errorf("expected both children listed, got:\n%s", out)
	}
	if strings.Contains(out, "No sub-issues found") {
		t.Errorf("expected children, got the empty-state message:\n%s", out)
	}

	// Exact matching must have routed both operations to distinct fixtures.
	ops := handler.operationNames()
	if len(ops) < 2 || ops[0] != "GetIssue" || ops[1] != "GetSubIssues" {
		t.Errorf("expected GetIssue then GetSubIssues, got %v", ops)
	}
}

// ---------------------------------------------------------------------------
// move scenarios (mutation path)
// ---------------------------------------------------------------------------

// TestScenario_Move_ResolvesAliasAndSubmitsMutation is the highest-value
// scenario here: it drives the whole write chain — config alias "in_progress"
// -> field value "In Progress" -> single-select option id "opt-inprog" -> the
// BatchUpdate mutation payload. Interface-level mocks stop at the client
// boundary and never verify the option id actually submitted.
func TestScenario_Move_ResolvesAliasAndSubmitsMutation(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetUserProject", userProjectFixture("proj-1", "Test Project"))
	handler.respondToQueryContaining("i0_0: issue(number:",
		batchIssueLookupFixture(42, "Movable issue", "OPEN", "proj-1", "item-1", "Backlog"))
	handler.respondTo("GetProjectFields", projectFieldsFixture())

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newMoveCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	if err := runMove(cmd, []string{"42"}, &moveOptions{status: "in_progress"}); err != nil {
		t.Fatalf("runMove() error = %v (output %q)", err, buf.String())
	}

	out := buf.String()
	if !strings.Contains(out, "Updated issue #42") {
		t.Errorf("expected confirmation for issue #42, got:\n%s", out)
	}
	if !strings.Contains(out, "Status -> In Progress") {
		t.Errorf("expected the resolved status in output, got:\n%s", out)
	}

	// The mutation must carry the resolved option id, not the raw alias.
	reqs := handler.requestsFor("BatchUpdate")
	if len(reqs) != 1 {
		t.Fatalf("expected exactly one BatchUpdate mutation, got %d (ops: %v)",
			len(reqs), handler.operationNames())
	}
	input, ok := reqs[0].Variables["input0"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected input0 object in mutation variables, got %v", reqs[0].Variables)
	}
	if input["fieldId"] != "field-status" {
		t.Errorf("expected fieldId 'field-status', got %v", input["fieldId"])
	}
	if input["itemId"] != "item-1" {
		t.Errorf("expected itemId 'item-1', got %v", input["itemId"])
	}
	if input["projectId"] != "proj-1" {
		t.Errorf("expected projectId 'proj-1', got %v", input["projectId"])
	}
	value, ok := input["value"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected value object, got %v", input["value"])
	}
	if value["singleSelectOptionId"] != "opt-inprog" {
		t.Errorf("expected the 'in_progress' alias to resolve to option id 'opt-inprog', got %v",
			value["singleSelectOptionId"])
	}
}

// TestScenario_Move_UnknownIssueReportsError covers the failure path: the batch
// lookup finds nothing, so no mutation should be attempted.
func TestScenario_Move_UnknownIssueReportsError(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetUserProject", userProjectFixture("proj-1", "Test Project"))
	handler.respondToQueryContaining("i0_0: issue(number:", gqlData(map[string]interface{}{
		"r0": map[string]interface{}{"i0_0": nil},
	}))
	handler.respondTo("GetProjectFields", projectFieldsFixture())

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newMoveCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := runMove(cmd, []string{"999"}, &moveOptions{status: "in_progress"})
	if err == nil {
		t.Fatalf("expected an error for an unresolvable issue, got nil (output %q)", buf.String())
	}

	if got := handler.requestsFor("BatchUpdate"); len(got) != 0 {
		t.Errorf("expected no mutation when the issue cannot be resolved, got %d", len(got))
	}
}

func TestScenario_SubList_NoChildrenRendersEmptyState(t *testing.T) {
	handler := newMockGraphQLHandler()
	handler.respondTo("GetIssue", issueFixture(10, "Childless issue", "OPEN"))
	handler.respondTo("GetSubIssues", subIssuesFixture())

	_, cleanup := setupTestEnvironment(t, handler)
	defer cleanup()

	cmd := newSubListCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	opts := &subListOptions{state: "all", relation: "children"}
	if err := runSubList(cmd, []string{"10"}, opts); err != nil {
		t.Fatalf("runSubList() error = %v", err)
	}

	if !strings.Contains(buf.String(), "No sub-issues found") {
		t.Errorf("expected empty-state message, got:\n%s", buf.String())
	}
}
