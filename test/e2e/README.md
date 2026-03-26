# E2E Tests

End-to-end tests for gh-pmu that run against a real GitHub project. These tests build the binary with coverage instrumentation and execute commands against live GitHub APIs.

## Prerequisites

- GitHub CLI authenticated (`gh auth status`)
- `.gh-pmu.json` configured in the repository root
- Network access to GitHub API

## Running

```bash
go test -tags e2e ./test/e2e/ -v -count=1
```

The `e2e` build tag is required — `go test ./...` does **not** include these tests.

## What's Tested

| Suite | Coverage |
|-------|----------|
| Board | Rendering, filtering by priority |
| Branch | Start, current, list, close lifecycle |
| Filter | By status, priority, branch, no-branch, combined |
| Init | Non-interactive modes, flag validation, overwrite |
| View | JSON output with standard and project fields |
| Workflow | Create-to-close, sub-issues, multi-move, labels, force-yes |

## Runtime

Typical run: ~5 minutes. Tests create and delete real GitHub issues; cleanup runs automatically via `t.Cleanup`.

## Adding Tests

- Use the `//go:build e2e` build tag
- Use `createTestIssue` / `createTestBranch` helpers from `helpers_test.go`
- Register cleanup with the helper's built-in `t.Cleanup` to delete test issues
- Prefix test issue titles with `[E2E]` for easy identification
