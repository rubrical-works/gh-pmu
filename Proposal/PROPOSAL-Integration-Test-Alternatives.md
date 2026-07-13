# Proposal: Integration Test Alternatives

**Version:** 1.0
**Date:** 2025-12-08
**Author:** API-Integration-Specialist
**Status:** Adopted (2026-07-13, via #876)
**Predecessor:** PROPOSAL-Automated-Testing.md (v1.3)

---

## Decision (adopted 2026-07-13, #876)

This proposal is **adopted**. The resolution combines the three alternatives:

1. **Local-only live-API testing** — the `test/e2e` suite (`-tags e2e`) is the
   primary live-API layer, run manually before releases; the sound
   `cmd/*_integration_test.go` suite (`-tags integration`) is retained as
   manual coverage. Neither runs in CI (account-lockout risk).
2. **Lockout-safe CI** — CI compile-checks tag-gated tests via
   `go vet -tags "integration e2e" ./...` (zero API calls) so they can't rot.
3. **Improved mock-based coverage** — tracked as a follow-up in #884
   (wrapper-harness scenarios, VCR record/replay, vendored-schema validation).

The broken/stale remnants (`internal/api/integration_test.go`,
`cmd/uat_epic*_test.go`) and the disabled `integration-tests.yml` workflow were
deleted in #876.

---

## Executive Summary

### Problem Statement

The integration testing strategy defined in PROPOSAL-Automated-Testing.md resulted in **GitHub account lockout** when executed via CI. Running automated tests against the live GitHub API triggers rate limiting and abuse detection mechanisms that can lock out user accounts.

**Key Finding:** Integration tests that make real GitHub API calls cannot safely run in CI environments.

### Current State

| Metric | Value | Notes |
|--------|-------|-------|
| Unit test coverage | 58.4% | Safe, no API calls |
| Integration tests exist | 12 files | Cannot run in CI |
| CI workflow | Disabled | Caused account lockout |
| Target coverage | 80% | Unreachable with current approach |

### Proposed Alternatives

This proposal presents three alternative approaches to achieve test coverage goals without risking account lockout:

1. **Mock-based unit tests** for `run*` functions
2. **Local-only integration testing** with manual execution
3. **Adjusted coverage targets** based on testable code only

---

## Background

### What Happened

On 2025-12-08, running the integration tests workflow (`integration-tests.yml`) via GitHub Actions resulted in account lockout. The workflow made multiple GitHub API calls (GraphQL queries and mutations) in rapid succession, triggering GitHub's abuse detection.

### Root Cause Analysis

| Factor | Impact |
|--------|--------|
| Multiple API calls per test | High request volume |
| Parallel test execution | Burst traffic pattern |
| CI environment IP reputation | Shared infrastructure |
| No rate limit handling | Exceeded limits quickly |

### Why CI Integration Tests Are Problematic

1. **Rate Limits**: GitHub API has strict rate limits (5,000 requests/hour for authenticated users)
2. **Abuse Detection**: Automated patterns trigger security systems
3. **Shared IPs**: CI runners share IP addresses, compounding rate limit issues
4. **Token Exposure Risk**: Storing PATs in CI secrets is a security concern
5. **Test Isolation**: Hard to prevent tests from affecting real project data

---

## Alternative 1: Mock-Based Unit Tests

### Overview

Replace integration tests with unit tests that mock the GitHub API client. This approach tests the command logic without making real API calls.

### Implementation Strategy

```go
// Example: Testing runList with mocked client
func TestRunList_WithMockedClient(t *testing.T) {
    // Create mock client
    mockClient := &MockAPIClient{
        GetProjectItemsFunc: func(projectID string, opts ...QueryOption) ([]ProjectItem, error) {
            return []ProjectItem{
                {ID: "item1", Title: "Test Issue 1", Status: "Backlog"},
                {ID: "item2", Title: "Test Issue 2", Status: "In Progress"},
            }, nil
        },
    }

    // Inject mock into command
    cmd := newListCommand()
    cmd.SetClient(mockClient)

    // Execute and verify
    output := captureOutput(func() {
        cmd.Execute()
    })

    assert.Contains(t, output, "Test Issue 1")
    assert.Contains(t, output, "Test Issue 2")
}
```

### Required Changes

1. **Refactor commands** to accept an API client interface
2. **Create mock implementations** of the API client
3. **Add dependency injection** to command constructors
4. **Write unit tests** for all `run*` functions using mocks

### Pros and Cons

| Pros | Cons |
|------|------|
| Safe to run in CI | Doesn't test real API behavior |
| Fast execution | Requires significant refactoring |
| No rate limit concerns | Mocks may drift from real API |
| High coverage achievable | More test maintenance |

### Effort Estimate

| Task | Story Points |
|------|--------------|
| Define API client interface | 3 |
| Create mock implementations | 5 |
| Refactor commands for DI | 8 |
| Write mock-based unit tests | 13 |
| **Total** | **29** |

---

## Alternative 2: Local-Only Integration Testing

### Overview

Keep existing integration tests but document them as **local-only**. Developers run them manually against their own test fixtures before significant releases.

### Implementation Strategy

1. **Remove CI workflow entirely** (already done)
2. **Document local setup** in TESTING.md
3. **Create personal test fixtures** guide
4. **Add rate limit handling** to tests
5. **Implement test throttling** (delays between API calls)

### Local Test Execution

```bash
# Developer runs locally with their own token
export TEST_PROJECT_OWNER="my-username"
export TEST_PROJECT_NUMBER="99"
export TEST_REPO_OWNER="my-username"
export TEST_REPO_NAME="my-test-repo"

# Run with throttling
go test -v -tags=integration -parallel=1 -timeout=30m ./...
```

### Safeguards to Add

```go
// Add to testutil package
func ThrottleAPICall() {
    time.Sleep(500 * time.Millisecond) // Prevent rate limiting
}

func RequireManualConfirmation(t *testing.T) {
    if os.Getenv("INTEGRATION_TEST_CONFIRMED") != "yes" {
        t.Skip("Set INTEGRATION_TEST_CONFIRMED=yes to run integration tests")
    }
}
```

### Pros and Cons

| Pros | Cons |
|------|------|
| Tests real API behavior | No automated regression testing |
| Minimal code changes | Relies on developer discipline |
| Catches real integration bugs | May be skipped before releases |
| Already implemented | Each developer needs test fixtures |

### Effort Estimate

| Task | Story Points |
|------|--------------|
| Update TESTING.md documentation | 2 |
| Add throttling to existing tests | 3 |
| Create personal fixture setup guide | 2 |
| Add manual confirmation safeguard | 1 |
| **Total** | **8** |

---

## Alternative 3: Adjusted Coverage Targets

### Overview

Accept that certain code paths cannot be automatically tested and adjust coverage targets accordingly. Focus unit test coverage on testable code.

### Revised Coverage Model

| Category | Code | Target | Rationale |
|----------|------|--------|-----------|
| **Testable** | Business logic, parsing, validation | 90% | Can be unit tested |
| **API Boundary** | `run*` function API calls | 0% (excluded) | Requires real API |
| **Entry Points** | `main()`, `Execute()` | 0% (excluded) | Not unit testable |

### Implementation

1. **Identify API boundary code** in each `run*` function
2. **Extract testable logic** into separate functions
3. **Write thorough unit tests** for extracted logic
4. **Document exclusions** in coverage reports

### Example Refactoring

```go
// BEFORE: runList has untestable API calls mixed with testable logic
func runList(cmd *cobra.Command, args []string) error {
    // Testable: flag parsing, validation
    status, _ := cmd.Flags().GetString("status")

    // Untestable: API call
    items, err := client.GetProjectItems(projectID)

    // Testable: filtering, formatting
    filtered := filterItems(items, status)
    output := formatOutput(filtered)

    fmt.Print(output)
    return nil
}

// AFTER: Extract testable logic
func runList(cmd *cobra.Command, args []string) error {
    opts, err := parseListOptions(cmd)  // Testable
    if err != nil {
        return err
    }

    items, err := client.GetProjectItems(projectID)  // Untestable boundary
    if err != nil {
        return err
    }

    output := formatListOutput(items, opts)  // Testable
    fmt.Print(output)
    return nil
}

// Unit testable functions
func parseListOptions(cmd *cobra.Command) (*ListOptions, error) { ... }
func formatListOutput(items []ProjectItem, opts *ListOptions) string { ... }
```

### Revised Targets

| Package | Current | Revised Target | Notes |
|---------|---------|----------------|-------|
| `cmd/` | 49.8% | 70% | Excluding API boundary |
| `internal/api/` | 80.4% | 85% | Focus on error handling |
| `internal/config/` | 97.0% | 95% | Maintain |
| `internal/ui/` | 96.9% | 95% | Maintain |
| **Total** | **58.4%** | **70%** | Realistic target |

### Pros and Cons

| Pros | Cons |
|------|------|
| Achievable targets | Lower overall coverage |
| Focuses effort on valuable tests | May miss integration bugs |
| No refactoring required | Requires buy-in on exclusions |
| Clear documentation of gaps | Some code paths never tested |

### Effort Estimate

| Task | Story Points |
|------|--------------|
| Extract testable logic from run functions | 8 |
| Write unit tests for extracted functions | 13 |
| Document coverage exclusions | 2 |
| Update CI coverage thresholds | 1 |
| **Total** | **24** |

---

## Recommendation

### Proposed Approach: Hybrid (Alternatives 2 + 3)

Combine **local-only integration testing** with **adjusted coverage targets**:

1. **Keep integration tests** for local/manual execution
2. **Extract testable logic** from `run*` functions
3. **Set realistic CI coverage target** of 70%
4. **Document the testing strategy** clearly

### Implementation Phases

#### Phase 1: Immediate (2 SP)
- [x] Disable CI integration tests workflow
- [ ] Update TESTING.md with local-only guidance
- [ ] Add throttling safeguards to existing tests

#### Phase 2: Short-term (8 SP)
- [ ] Extract testable logic from `run*` functions
- [ ] Write unit tests for extracted logic
- [ ] Update coverage thresholds in CI

#### Phase 3: Ongoing
- [ ] Run integration tests locally before major releases
- [ ] Maintain documentation
- [ ] Consider Alternative 1 (mocking) if coverage gaps become problematic

### Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Unit test coverage | 58.4% | 70% |
| CI stability | N/A | 100% pass rate |
| Integration test coverage | Manual | Run before releases |
| Account lockouts | 1 | 0 |

---

## Open Questions

| # | Question | Impact | Resolution |
|---|----------|--------|------------|
| 1 | Should we invest in full mocking (Alt 1)? | High effort, high coverage | Defer unless 70% insufficient |
| 2 | How often should local integration tests run? | Test confidence vs effort | Before major releases |
| 3 | Should we create a pre-release checklist? | Process documentation | Yes, include in TESTING.md |

---

## References

- PROPOSAL-Automated-Testing.md v1.3
- Epic #88: Integration Testing
- TESTING.md (current integration test documentation)

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-12-08 | API-Integration-Specialist | Initial proposal following CI lockout incident |
