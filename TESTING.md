# Testing Guide

Comprehensive testing strategy for gh-pmu.

## Coverage Targets

| Package | Target | Current | Notes |
|---------|--------|---------|-------|
| Overall | 68-70% | 68.5% | Coverage target for CLI commands |
| `internal/api` | 60%+ | 59.4% | GraphQL mocking is complex |
| `internal/config` | 75%+ | 74.4% | Core configuration parsing |
| `internal/framework` | 85%+ | 88.9% | Framework detection logic |
| `internal/ui` | 95%+ | 96.9% | UI component rendering |
| `cmd` (wrappers) | 70%+ | 68.8% | Command implementations |

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

### Integration Tests (`cmd/wrapper_test.go`)

Tests that exercise command execution with mocked HTTP transport:

```go
// Pattern: Setup test environment with mock server
func TestRunList_LoadsConfig(t *testing.T) {
    handler := newMockGraphQLHandler()
    _, cleanup := setupTestEnvironment(t, handler)
    defer cleanup()

    // Execute command
    err := runList(cmd, []string{}, opts)

    // Assert config loading worked (API errors expected)
}
```

**Covered:**
- Config loading for all command wrappers
- Error handling when config is missing
- Flag parsing and validation
- API client creation

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
| `runMicrosprintResolveWithDeps` | `cmd/microsprint.go` | Uses `fmt.Scanln` for confirmation | Manual testing, #415 |

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
| Various `output*` functions | Multiple | Table/JSON formatting | Visual verification |

**Why not tested:** Output formatting is validated visually. The logic extracting data is tested; only the presentation layer is excluded.

### Simple Command Wrappers

| Function | File | Reason | Alternative |
|----------|------|--------|-------------|
| `RunE` closures | `cmd/release.go` | Inline functions that just call `*WithDeps` | Tested via `*WithDeps` |
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

### Microsprint/Release
- [ ] `gh pmu microsprint start` - Creates tracker issue
- [ ] `gh pmu microsprint resolve` - Handles conflicts
- [ ] `gh pmu release close --tag` - Creates git tag

## Test Patterns

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

- #414 - Coverage improvement from 63.2% to 68.5%
- #416 - Refactor external process calls for testability
- #797 - Remove all interactive features (CLI-only mode)

## See Also

- [Development Guide](docs/development.md) - Build and run tests
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
