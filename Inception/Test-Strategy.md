# Test Strategy: gh-pmu

**Last Updated:** 2026-01-04

---

## Testing Philosophy

Multi-layer testing with emphasis on table-driven unit tests and integration tests with mocked API client. UAT tests validate end-to-end workflows.

---

## Test Pyramid

| Level | Coverage | Approach |
|-------|----------|----------|
| Unit | ~75% | Table-driven tests, standard `testing` package |
| Integration | ~60% | Mocked API client, command execution |
| E2E/UAT | Critical paths | Epic-based acceptance scenarios |

---

## Test Types

### Unit Tests

- **Framework:** Go standard `testing` package
- **Location:** `*_test.go` alongside source files
- **Naming:** `TestFunctionName_Scenario`
- **Coverage Target:** 70%+

### Integration Tests

- **Framework:** Go `testing` with build tags
- **Location:** `*_integration_test.go` files
- **Scope:** Command execution with mocked API
- **Environment:** Mock client, test fixtures

### End-to-End Tests

- **Framework:** UAT tests (`uat_*_test.go`)
- **Scope:** Full user workflows (epic → story completion)
- **Environment:** Mocked API, realistic scenarios

---

## Quality Gates

| Gate | Criteria | Enforcement |
|------|----------|-------------|
| Pre-commit | `go test ./...`, `go vet` | Developer discipline |
| PR Merge | All tests pass, coverage maintained | GitHub Actions |
| Release | Integration + UAT pass | Manual verification |

---

## Test Data Strategy

| Type | Approach |
|------|----------|
| Unit test data | Inline fixtures, table-driven |
| Integration test data | `testdata/` directory, mock responses |
| E2E test data | Realistic scenarios with mock API |

---

## Special Testing Considerations

### Performance Testing

Informal benchmarks for startup time and API call latency. Target < 500ms startup.

### Security Testing

Manual review of authentication flow (delegated to gh CLI). No credential storage to test.

### Accessibility Testing

Not applicable (terminal CLI).

---

## Definition of "Tested"

A feature is considered tested when:
- [x] Unit tests cover core logic and edge cases
- [x] Integration tests verify command execution
- [x] Error paths are tested
- [x] All tests pass in CI

---

## Test Files Reference

| Command | Unit Tests | Integration Tests |
|---------|-----------|-------------------|
| list | `list_test.go` | `list_integration_test.go` |
| create | `create_test.go` | `create_integration_test.go` |
| view | `view_test.go` | `view_integration_test.go` |
| edit | `edit_test.go` | `edit_integration_test.go` |
| move | `move_test.go` | `move_integration_test.go` |
| sub | `sub_test.go` | `sub_*_integration_test.go` |
| branch | `branch_test.go` | - |
| label | `label_test.go` | - |
| validation | `validation_test.go` | - |

---

*See also: Charter-Details.md, Constraints.md*
