# Testing Guide

Comprehensive testing strategy for gh-pmu.

## Coverage Targets

Measured with `go test -short -cover ./...` (2026-07-14).

| Package | Target | Current | Notes |
|---------|--------|---------|-------|
| `internal/api` | 60%+ | 69.0% | GraphQL mocking is complex |
| `internal/config` | 75%+ | 84.8% | Core configuration parsing |
| `internal/defaults` | 85%+ | 88.9% | Default value resolution |
| `internal/framework` | 85%+ | 88.9% | Framework detection logic |
| `internal/integrity` | 85%+ | 86.7% | Config integrity verification |
| `internal/ui` | 95%+ | 96.9% | UI component rendering |
| `cmd` (wrappers) | 70%+ | 77.0% | Command implementations |

## Adjusted Coverage Accounting (#884)

**Statement coverage overstates verification in this repo. Read this before
treating the `cmd` number as a quality signal.**

### Why the raw number misleads

`go test -cover` measures which statements *executed*, not which were
*verified*. The wrapper tests in `cmd/wrapper_test.go` call each `run*`
function against a mock GraphQL server configured with no responses, then
assert only "this wasn't a config error" — most end with:

```go
if err == nil {
    return          // escape hatch: asserts nothing
}
```

Those calls execute the `run*` body, so every line counts toward coverage
while verifying nothing. Adding the content-asserting scenario tests in
`cmd/scenario_test.go` (#884) moved `cmd` coverage by **0.0 points**
(77.0% → 77.0%) precisely because the lines were already being executed.
The tests got materially stronger; the metric did not move.

**Consequence:** a coverage delta is the wrong acceptance signal for this
package. Prefer asking "does this test fail when the behavior breaks?" —
see the fail-proof discipline under Test Patterns.

### What the `cmd` remainder actually is

The ~23% uncovered in `cmd` is dominated by code that only e2e can reach.
Of the 22 zero-coverage functions, most fall into the exclusions already
listed under [Functions Excluded from Unit Testing](#functions-excluded-from-unit-testing):

| Category | Examples | Reachable by |
|----------|----------|--------------|
| Thin cobra wrappers | `runComment`, `runEdit`, `runFilter`, `runLabelList` | e2e only |
| External process calls | `getCommitStats`, `getCommitBody`, `getCommitFiles` | e2e / manual |
| Terminal UI rendering | `renderHistoryScreen`, `renderDetailedHistoryScreen` | manual |
| Browser handoff | `openHistoryInBrowser` | manual |

A thin wrapper is: load config → build client → delegate to `*WithDeps`.
The delegate carries the logic and is unit-tested directly; the wrapper adds
no branch worth a mock. Excluding that category is what "adjusted accounting"
means here — not lowering the bar, but measuring the code tests can own.

### Where the scenario tests actually pay off

Their value is **cross-package**, which the per-package `cmd` number cannot
show. `cmd` tests exercise real `internal/api` query construction and response
decoding through the genuine shurcooL plumbing:

```bash
go test -short -coverpkg=./... ./cmd/     # 64.4% of ./... from cmd tests alone
```

Interface-level mocks (`list_test.go`, `view_test.go`) substitute the client
entirely, so they never build a query or decode a response. The scenario tests
are the only automated layer that does — which is what makes them the
substitute for the retired live-API suite.

### Reporting rule

When citing coverage, state the command and date. Numbers drift, and stale
figures have caused real misdirection: `backlog/integration-testing-backlog.md`
(2025-12-04) reports `cmd` at 51.2%, which was ~26 points stale by 2026-07-14
and framed #884 around a gap that had largely closed.

## Test Categories

### Unit Tests (`*_test.go`)

Standard Go unit tests for isolated logic:

```bash
go test ./...                    # Run all tests
go test -v ./internal/api/...    # Specific package
go test -short ./...             # Skip long-running tests (CI mode)
```

**Covered:**
- Configuration parsing and validation
- API client methods with mocked GraphQL
- Field value parsing and formatting
- UI component rendering
- Utility functions

### Wrapper Smoke Tests (`cmd/wrapper_test.go`)

Config-loading smoke tests. They run each command against a mock server with
**no responses configured** and assert only that failures aren't config
failures:

```go
func TestRunList_LoadsConfig(t *testing.T) {
    handler := newMockGraphQLHandler()
    _, cleanup := setupTestEnvironment(t, handler)
    defer cleanup()

    err := runList(cmd, opts)
    if err == nil {
        return  // asserts nothing beyond "no config error"
    }
    ...
}
```

**Covered:** config loading, missing-config errors, flag validation, client
creation.

**Not covered:** anything about output, query construction, or decoding. Do not
extend this layer to test behavior — add a scenario test instead.

### Scenario Tests (`cmd/scenario_test.go`)

Command-level tests that drive the **real stack** — `run*` → config →
`api.Client` → real shurcooL GraphQL → `redirectTransport` → mock server — with
realistic fixtures and assertions on rendered output:

```go
func TestScenario_List_RendersIssuesFromProject(t *testing.T) {
    handler := newMockGraphQLHandler()
    handler.respondTo("GetUserProject", userProjectFixture("proj-1", "Test Project"))
    handler.respondTo("SearchIssues", searchIssuesFixture(
        issueNode(101, "First tracked issue", "OPEN", []string{"bug"}, []string{"alice"}),
    ))

    _, cleanup := setupTestEnvironment(t, handler)
    defer cleanup()

    // ... runList, then assert the title appears in the rendered table
}
```

This is the layer that replaces the retired live-API suite: it is the only
automated coverage of query construction and response decoding.

**Fixture matching.** The handler resolves responses deterministically:

1. `respondTo(op, resp)` — exact GraphQL operation name.
2. `respondToQueryContaining(substr, resp)` — ordered substring rules, for
   **anonymous** documents only.
3. `defaultResponse`.

Exact-name matching is not incidental. Several operation names are prefixes of
others — `GetIssue` of `GetIssueComments` / `GetIssueWithProjectFields` /
`GetIssuesByLabel`; `GetProjectItems` of `GetProjectItemsMinimal` /
`GetProjectItemsForBoard` — and the sub-list flow issues both `GetIssue` and
`GetSubIssues`. Substring matching over a map would pick a fixture in Go's
randomized map order and go flaky.

Use `respondToQueryContaining` only where there is no operation name to match:
`getProjectFieldsForIssuesBatch` and `move`'s issue lookup build aliased
documents (`query { r0: repository(...) { i0_0: issue(number: 42) ... } }`).

**Writing a new scenario.** Response shapes must match the real query, and
GraphQL inline fragments (`... on Issue`) flatten into the parent object rather
than nesting. Rather than guessing, print what the client actually sends:

```go
for i, r := range handler.requests {
    t.Logf("REQ %d: op=%s vars=%v\n%s", i, graphQLOperationName(r.Query), r.Variables, r.Query)
}
```

Then build the fixture against the observed document and iterate.

### Vendored Schema Validation (`internal/api/schema_*_test.go`)

Every GraphQL document this repo sends is checked against a vendored copy of
GitHub's published schema. This is the only automated check that catches GitHub
renaming or removing a field: the scenario tests and hand-written mocks encode
*our assumptions* about GitHub's responses, so they keep passing when the real
schema moves out from under a query. Zero API traffic — the capture transport
answers every request itself.

It earns its keep: on its first run it found a live bug (#888).

| Layer | Test |
|-------|------|
| Named operations (40, struct-generated) | `TestNamedOperations_ValidateAgainstVendoredSchema` |
| Anonymous documents (8, `fmt.Sprintf`-built) | `TestRawDocuments_ValidateAgainstVendoredSchema` |
| Coverage completeness | `TestNamedOperations_CoverageIsComplete` |
| Quarantined known-broken queries | `TestNamedOperations_KnownInvalidStillFail` |
| Fail-proof (both paths) | `TestFailProof_*` |

Coverage is enumerated from the production sources rather than a hardcoded list,
so adding an operation without a matching invocation fails the build and names
it. Documents are **captured, not requested**: `constructQuery` is unexported in
shurcooL, so the client is pointed at a recording transport and the document is
read off the wire.

**Refreshing the schema**

```bash
curl -sS -o testdata/graphql/schema.docs.graphql \
  https://docs.github.com/public/fpt/schema.docs.graphql
sha256sum testdata/graphql/schema.docs.graphql
wc -c < testdata/graphql/schema.docs.graphql
```

Update `testdata/graphql/schema-provenance.json` with the new `sha256`, `bytes`
and `retrieved` date, then run `go test -short ./internal/api/`. (Use `-short`:
without it the package also runs `TestNewClient_HasGraphQLClient`, which needs
`gh` auth and is unrelated to the schema.)

Never edit the vendored file by hand — it is a verbatim upstream artifact.
`.gitattributes` marks it `-text` because `core.autocrlf` would otherwise
rewrite its line endings on checkout and break the digest check for everyone but
the person who vendored it.

**Cadence.** Refresh before a release, and whenever a query starts failing
against the real API for no local reason. There is no value in chasing every
upstream deploy: the schema is additive most weeks — it grew 4,147 bytes in a
single day during #886 — and additive changes cannot break us.

**Telling real drift from benign drift**

| Symptom | Meaning | Action |
|---------|---------|--------|
| Digest mismatch, no refresh done | file hand-edited or partially written | restore: `git checkout -- testdata/graphql/` |
| Schema grew, everything still validates | additive upstream change | benign — update the provenance record |
| An operation stops validating **after** a refresh | real drift: GitHub changed a field we use | fix the query — the failure names the field and the type |
| An operation fails but no refresh was done | our query was always wrong | fix the query — this is how #888 surfaced |

The distinction that matters: only a **refresh** can introduce real drift. A
validation failure without one means the query was already broken.

**On failure.** Drift fails the build; it is not a warning. A query referencing
a field GitHub has removed is broken in production, and the point is to learn
that here rather than from a user. If the fix is not immediate, quarantine the
operation in `knownInvalidOperations` against a filed issue — the quarantine is
asserted rather than skipped, so it fails once the bug is fixed and cannot
silently rot into fake coverage.

### Live-API Tests (manual only — never run in CI)

Two suites exercise the real `gh pmu` binary against a live GitHub project.
They make real API calls and **must not run in CI**: doing so caused a GitHub
account lockout on 2025-12-08 (burst GraphQL traffic from shared-IP runners
tripped abuse detection). See `Proposal/Integration-Test-Alternatives` and #876.

| Suite | Tag | Location | Status |
|-------|-----|----------|--------|
| E2E | `e2e` | `test/e2e/` | **Primary** live-API layer — env-gated, retry-aware, self-cleaning |
| Command integration | `integration` | `cmd/*_integration_test.go` | Retained as manual coverage for commands e2e doesn't cover (create, edit, intake, split, triage, fine-grained sub_*) |

Run them **locally** (e.g. before a release):

```bash
go test -tags e2e ./test/e2e/ -v            # requires gh auth; runs against the real project
go test -tags integration ./cmd/ -v         # manual integration suite
```

CI never executes these. It only **compiles** them — with zero API calls —
via the lint job's compile-check so signature drift can't rot them unnoticed
(this is what had broken the now-deleted `internal/api/integration_test.go`):

```bash
go vet -tags "integration e2e" ./...        # type-check only; makes no API calls
```

**The compile-check is the intended CI ceiling, by design — not a gap awaiting
closure.** CI deliberately verifies that the tag-gated suites still *compile*
and stops there, because executing them is what caused the 2025-12-08 account
lockout (#876). Anything that would raise the ceiling — a gated job, a
scheduled run, a "just the read-only tests" carve-out — reopens the failure
mode the ceiling exists to prevent, however narrowly scoped. Automated
verification of these paths comes from the mock-based scenario tests (#884)
instead, and live execution stays manual and pre-release. If the ceiling is
ever to be revisited, that belongs in a proposal amending
`Proposal/PROPOSAL-Integration-Test-Alternatives.md`, not in a workflow edit.

> The old `cmd/uat_epic*_test.go` UAT files and `internal/api/integration_test.go`
> were deleted in #876 (stale/non-compiling). The disabled
> `integration-tests.yml` CI workflow was also removed — do **not** re-enable
> a live-API CI workflow. Mock-based coverage replaced it: the scenario tests
> below (#884) are the automated substitute, and record/replay was evaluated
> and declined (see `Construction/Design-Decisions/`).

### Manual Testing

Functions that require visual verification or user interaction:

```bash
# Visual output
gh pmu board                     # Verify kanban layout
gh pmu history                   # Verify terminal UI
gh pmu list --status in_progress # Verify table formatting
```

## Functions Excluded from Unit Testing

### Interactive CLI Functions

| Function | File | Reason | Alternative |
|----------|------|--------|-------------|
| `runInit` | `cmd/init.go:59` | Uses `bufio.NewReader(os.Stdin)` for prompts | Manual testing, #415 |
| `runFilter` | `cmd/filter.go:80` | Checks `os.Stdin.Stat()` for piped input | Manual testing, #415 |

**Why not mocked:** These functions hardcode `os.Stdin` and would require refactoring to accept `io.Reader` for testability. See issue #415.

### External Process Functions

| Function | File | Reason | Alternative |
|----------|------|--------|-------------|
| `openEditorForBody` | `cmd/create.go:484` | Opens `$EDITOR` or `vim` | Manual testing, #416 |
| `detectRepository` | `cmd/init.go:427` | Runs `git remote get-url` | Manual testing, #416 |
| `GetLatestGitTag` | `internal/api/client.go:151` | Runs `git describe --tags` | Manual testing, #416 |
| `getCommitHistory` | `cmd/history.go` | Multiple `git log` commands | Manual testing, #416 |

**Why not mocked:** These functions call `exec.Command` directly. Refactoring to use a `CommandExecutor` interface would enable testing. See issue #416.

### Terminal UI Rendering

| Function | File | Reason | Alternative |
|----------|------|--------|-------------|
| `renderHistoryScreen` | `cmd/history.go` | Terminal UI with cursor control | Visual verification |
| `outputBoardKanban` | `cmd/board.go` | Kanban board layout | Visual verification |

**Why not tested:** Full-screen/interactive rendering is validated visually.

**Not excluded — `output*` functions.** Table and JSON writers that go through
`cmd.OutOrStdout()` **are** asserted: `cmd.SetOut(&buf)` redirects them and the
content is checked (#875, #878). Do not add new assertion-free tests for them.

The exception is `cmd/triage.go`'s output functions, which write to `os.Stdout`
directly (tabwriter / `json.Encoder`) and so are not capturable through the
cobra writer. **#871** tracks routing them through `cmd.OutOrStdout()`; the
content assertions belong here once it lands.

### Simple Command Wrappers

| Function | File | Reason | Alternative |
|----------|------|--------|-------------|
| `RunE` closures | Various | Inline functions that just call `*WithDeps` | Tested via `*WithDeps` |
| `newXxxCommand` | Multiple | Cobra command setup only | Tested via command execution |

**Why not tested:** These are thin wrappers with no logic. The actual implementation in `*WithDeps` functions is tested.

## Manual Testing Checklist

Before releases, verify these features:

### Initialization
- [ ] `gh pmu init --non-interactive` - Creates config with flags
- [ ] Auto-detects repository from git remote

### Issue Management
- [ ] `gh pmu create --template bug` - Loads issue template
- [ ] `gh pmu move 123 --status done` - Updates and closes

### Board Display
- [ ] `gh pmu board` - Kanban columns render correctly
- [ ] `gh pmu board --status ready` - Filters by status
- [ ] Column widths adapt to terminal size

### History
- [ ] `gh pmu history` - Terminal UI navigates
- [ ] Arrow keys scroll through history
- [ ] `q` exits cleanly

### Branch Workflow
- [ ] `gh pmu branch start --name release/v2.0.0` - Creates branch and tracker
- [ ] `gh pmu branch current` - Shows active branch
- [ ] `gh pmu branch close --tag` - Creates git tag

## Test Patterns

### Fail-Proof Discipline (the signal that replaces coverage)

Because statement coverage counts execution rather than verification (see
[Adjusted Coverage Accounting](#adjusted-coverage-accounting-884)), the
acceptance signal for a test is whether it **fails when the behavior breaks**.

Before trusting a new test, prove it:

1. Break the production logic the test names — one line, the real behavior.
2. Run the test. It must fail, with a message that identifies the break.
3. Restore the production code and re-run.

Worked examples from #878/#884:

| Break applied | Test that caught it |
|---------------|---------------------|
| Drop `cfg.Defaults.Labels` from the create merge | `TestLabelMerging_WithDefaults` |
| Remove the `is:all` branch from `parseTriageQuery` | `TestSearchIssuesForTriage_QueryParsing` |
| Drop the alias lookup in `ResolveFieldValue` | `TestScenario_Move_ResolvesAliasAndSubmitsMutation` |
| Drop `repo:` scoping from the search builder | `TestScenario_List_SearchQueryCarriesRepoAndState` |

That last one matters: without repo scoping, `list` searches all of GitHub. No
coverage delta would have flagged it — the lines ran either way.

**Watch for symmetric fixtures.** A test can execute the right code and still
be blind. `TestViewJSONOutput_WithSubProgress` uses one CLOSED and one OPEN
sub-issue, so inverting `sub.State == "CLOSED"` still yields a count of 1 and
the test passes. It took an all-closed fixture to expose the inversion. When a
test counts or partitions, ensure at least one fixture is asymmetric.

### Wrapper Function Pattern

All command wrappers follow this pattern for testability:

```go
// Wrapper (calls real dependencies)
func runList(cmd *cobra.Command, args []string, opts *listOptions) error {
    cfg, err := config.LoadFromDirectory(".")
    if err != nil {
        return err
    }
    client := api.NewClient()
    return runListWithDeps(cmd, args, opts, cfg, client)
}

// Testable implementation (accepts dependencies)
func runListWithDeps(cmd *cobra.Command, args []string, opts *listOptions,
                     cfg *config.Config, client listClient) error {
    // Actual implementation
}
```

### Mock GraphQL Handler

For testing API calls without real GitHub:

```go
handler := newMockGraphQLHandler()
handler.defaultResponse = map[string]interface{}{
    "data": map[string]interface{}{...},
}
server := httptest.NewServer(handler)
api.SetTestTransport(&redirectTransport{server: server})
```

### Config Not Found Tests

Every wrapper should test missing config:

```go
func TestRunXxx_ConfigNotFound(t *testing.T) {
    tmpDir, _ := os.MkdirTemp("", "gh-pmu-test-*")
    defer os.RemoveAll(tmpDir)
    os.Chdir(tmpDir)

    err := runXxx(cmd, opts)

    if !strings.Contains(err.Error(), "failed to load configuration") {
        t.Errorf("expected config error, got: %v", err)
    }
}
```

## CI Configuration

Tests run automatically on every push and PR:

```yaml
# .github/workflows/ci.yml
- name: Run tests
  run: go test -short -coverprofile=coverage.out ./...
```

The `-short` flag skips tests that require `gh` authentication, enabling tests to run in CI without credentials.

## Related Issues

- #414 - Coverage improvement tracking
- #416 - Refactor external process calls for testability
- #797 - Remove all interactive features (CLI-only mode)

## See Also

- [Development Guide](docs/development.md) - Build and run tests
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
