# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.0] - 2026-03-09

### Changed
- Rebranded from "Project Management Unified" to "Praxis Management Utility" — PMU acronym unchanged (#698)
- Renamed organization from rubrical-studios to rubrical-works (#736)
- Made `.gh-pmu.json` the primary config file with `.gh-pmu.yml` as fallback (#737)
- Updated `gh pmu init` display strings to reference `.gh-pmu.json` (#735)
- Updated LICENSE copyright to rubrical-works 2026 (#698)

### Optimized
- `gh pmu list --state all` uses dual search calls (open + closed) instead of full project scan (#740)
- `gh pmu filter` eliminates dead-code `GetProjectItems` fallback; errors when no repos configured (#741)
- `gh pmu board --state all` uses dual search calls instead of full project scan (#742)
- `gh pmu move --recursive` resolves sub-issues via targeted API instead of full project scan (#743)
- `gh pmu branch current` optimized with active label lookup and `--json` support; removed `--refresh` (#738)
- `gh pmu branch close` optimized to use tracker sub-issues instead of full project scan (#739)

## [1.1.0] - 2026-03-03

### Added
- Multi-issue support for `gh pmu view` (#725)
  - Accepts variadic issue numbers: `gh pmu view 42 43 44`
  - `--json` returns a JSON array for multiple issues, single object for one (backward compatible)
  - `--jq` works with multi-issue JSON array output
  - Non-JSON output shows issues sequentially with separator
  - Invalid issue numbers report errors without blocking valid issues
  - `--body-stdout`, `--body-file`, and `--web` restricted to single issue
- Batch GraphQL API methods for O(1) multi-issue fetching (#725)
  - `GetIssuesWithProjectFieldsBatch` — fetch N issues + project fields in one query
  - `GetParentIssueBatch` — fetch parent issues for N issues in one query

### Changed
- Renamed "Rubrical Studios" to "Rubrical Systems" in `--version` output and embedded terms (#726)

## [1.0.5] - 2026-03-02

### Fixed
- `gh pmu move --backlog` now sets status to Backlog in addition to clearing the branch field (#721)
  - Previously only cleared branch association; status remained unchanged
  - Explicit `--status` flag takes precedence when combined with `--backlog`

## [1.0.4] - 2026-03-01

### Changed
- Removed "Board" suffix from project title during `gh pmu init` (#718)
  - `runInitNonInteractive()` and `autoCreateProject()` now use the repository name directly as the project title
  - Extracted `deriveProjectTitle()` helper for testability

## [1.0.3] - 2026-03-01

### Changed
- Renamed `--project` flag to `--source-project` in `gh pmu init --non-interactive` (#714)
  - Non-interactive mode now creates a new project by copying from the source project template
  - Links repository to the newly created project automatically
  - Config file records the new project number (not the source)
- Fixed `CopyProjectV2` GraphQL mutation to use named input type (#714)

## [1.0.2] - 2026-02-27

### Added
- 403/429 HTTP rate limit detection with automatic retry and exponential backoff (#710)
  - Interface-based duck typing for go-gh's `HTTPError` (avoids package name collision)
  - `IsRateLimited()` detects HTTP 429, 403 with rate-limit messaging, and sentinel errors
  - `GetRetryAfter()` extracts Retry-After header value from errors
  - `WithRetry()`/`WithRetryDelays()` with configurable exponential backoff (1s, 2s, 4s, 8s)
  - Non-rate-limit 403 errors (permission denied) are not retried
- Copyright line in `--version` output (#712)
  - `gh pmu --version` now shows `Rubrical Systems (c) 2026` below version string
- Design decision documentation for interface-based HTTP error detection (#710)

### Fixed
- Acceptance section preserved during `gh pmu init` unless major/minor version change (#709)
  - Previously, acceptance was cleared on every init regardless of version change type
  - Now uses `RequiresReAcceptance()` to only clear on major/minor bumps

### Changed
- Updated IDPF framework to v0.54.0

## [1.0.1] - 2026-02-24

### Added
- `.gh-pmu.json` companion config file with dual-file sync (#706)
  - `config.Save()` writes both `.gh-pmu.yml` and `.gh-pmu.json` on every save
  - `gh pmu init` creates both files
  - `FindConfigFile()` falls back to `.gh-pmu.json` if YAML is missing
  - YAML remains authoritative when both files exist
  - All Config structs carry `json` tags alongside `yaml` tags

### Changed
- Acceptance gate now displays full terms text and `--yes` hint on stderr (#705)
- Accept command Long description references "Praxis Management Utility" (#704)
- Terms text updated: title includes "GitHub Praxis Management Utility" (#704)
- Added Rubrical Systems copyright to terms text (#704)
- Updated IDPF framework to v0.49.1

## [1.0.0] - 2026-02-23

### Added
- Source-level version constant in `internal/version/version.go` (#700)
  - `/prepare-release` updates the constant during release workflow
  - `gh pmu --version` reads from source constant with ldflags fallback
- Version field in `.gh-pmu.yml` config (#700)
  - `gh pmu init` writes the current version to config
  - Enables upgrade detection (compare config version vs installed version)
  - Backward compatible: existing configs without version load without error
- `pending` label added to standard labels collection (#701)
  - Color: `#D93F0B` (orange)
  - Indicates at least one `/review-issue` pass was made but not yet resolved

### Changed
- Updated IDPF framework to v0.46.2

## [0.16.0] - 2026-02-20

### Added
- Terms and conditions acceptance gate (#694)
  - `gh pmu accept` command with `--yes` flag for non-interactive acceptance
  - All commands (except `init`, `accept`, `--help`, `--version`) require acceptance
  - Acceptance stored in `.gh-pmu.yml` with user, date, and version fields
  - Re-acceptance triggered on major/minor version bumps (not patches)
  - Terms text embedded in binary via `go:embed`
  - Shared acceptance: one user's acceptance covers all repository collaborators
  - Dev builds skip the acceptance gate

### Changed
- Updated IDPF framework commands and config to v0.46.1

## [0.15.2] - 2026-02-16

### Fixed
- Replace hollow tests in `mutations_test.go` with behavioral validations (#692)
  - `TestCreateIssueInput_OptionalFields` replaced with label ID pass-through test
  - `TestAddIssueToProject_Success` now validates returned item ID
  - `TestCreateIssue_Success` now validates all mutation response fields
  - `TestDeleteProjectField_EmptyFieldID` converted to input validation test
- Remove dead `GetIssuesByRelease`/`GetIssuesByPatch` functions (#692)
  - Zero callers, ignored version parameters, removed from interface and mock
- Rewrite hollow `client_test.go` tests with behavioral assertions (#685)
- Remove 17 compiler-tautology struct tests from `mutations_test.go` (#686)
- Extract and test batch sub-issue parsing logic (#686)
- Rewrite hollow `BodyStdinSetsBodyFile` test to invoke function (#687)
- Replace `extractCommentID` no-op with GraphQL `databaseId` (#687)

### Added
- Integration tests for edit labels and move batch/branch/backlog (#690)

### Changed
- Updated IDPF framework commands and config to v0.44.0

## [0.15.1] - 2026-02-12

### Added
- Auto-manage `assigned` label on branch field changes (#682)
  - `gh pmu move --branch` adds `assigned` label to issues
  - `gh pmu move --backlog` removes `assigned` label from open issues
  - `gh pmu branch close` removes `assigned` label from open incomplete issues
  - `gh pmu branch remove` removes `assigned` label from open issues
  - Label auto-created if missing (uses standard label from `defaults.yml`)
  - Bulk move applies label to all specified issues

### Changed
- Updated IDPF framework commands and config to v0.42.2

## [0.15.0] - 2026-02-12

### Added
- `tech-debt` standard label added to `gh pmu init` (#671)
- `active` standard label for branch tracker identification (#677)
- `reviewed` standard label for issues passing review workflow (#679)
- `assigned` standard label for issues assigned to branches

### Fixed
- Windows CLI length limit for GraphQL queries (#673)
  - Refactored all `-f query=` call sites to use `--input -` with stdin
  - Matches existing `executeBatchMutation()` pattern
- Removed Claude reminder injection from issue bodies (#676)
  - `prependClaudeReminder()` function and all call sites removed
  - Workflow rules enforced via `.claude/rules/` instead

### Changed
- Migrated to IDPF hub-based installation (v0.42.0 → v0.42.1)

## [0.14.3] - 2026-01-29

### Fixed
- `gh pmu intake --apply` now works without a value (#667)
  - Flag accepts optional value using Cobra's `NoOptDefVal`
  - When used without value, applies config defaults as documented
- `gh pmu view --json=status` now returns actual project status (#668)
  - Added `status`, `priority`, `branch` as direct JSON fields
  - Previously returned `null` even when issue had valid project status
  - Fields extract values from `fieldValues` map for convenience

## [0.14.2] - 2026-01-27

### Fixed
- GraphQL union type error in `deleteProjectV2Field` mutation return selection (#661)
  - Changed return struct to use `ClientMutationID` instead of `ProjectV2Field.ID`
  - Avoids "Selections can't be made directly on unions" error during `gh pmu init`

### Changed
- Updated IDPF framework to v0.33.3

## [0.14.1] - 2026-01-27

### Added
- `--body-file` / `-F` flag for `gh pmu sub create` command (#660)
  - Provides consistency with `gh pmu create` and `gh issue create`
  - Supports reading body from file or stdin with `-`

### Fixed
- GraphQL variable type missing in `deleteProjectV2Field` mutation (#661)
  - The shurcooL/graphql library requires named types to generate variable declarations
  - Added `DeleteProjectV2FieldInput` struct following existing patterns

### Changed
- Updated IDPF framework to v0.33.2

## [0.14.0] - 2026-01-26

### Added
- Auto-create project from IDPF Kanban template during `gh pmu init` (#656)
  - When no projects exist and IDPF framework selected, offers to create from template #30
  - Automatically links repository and runs intake to populate board
- New API methods: `CopyProjectFromTemplate`, `GetOwnerID`, `LinkProjectToRepository`

### Changed
- **BREAKING:** Removed all microsprint support (#654)
  - `gh pmu microsprint` command removed
  - `--microsprint` flag removed from `move` and `create` commands
  - Microsprint field automatically removed during `gh pmu init`
- Fixed root command `Use` field from `gh-pmu` to `gh pmu` for correct help text (#654)

### Fixed
- E2E tests updated for IDPF branch and body validation requirements

## [0.13.11] - 2026-01-24

### Added
- `--json` and `--jq` flags for `view` command with project field support (#647)
- `--json-fields` flag to list available JSON fields for view command (#647)
- `--template` flag for `view` command with helpful error directing to `gh issue view` (#650)
- Confirmation prompt when `--force` bypasses checkbox validation on move command (#648)
- IDPF workflow warning after `--force` bypass: "WARNING: Workflow rules may have been violated" (#648)

### Changed
- `--yes` flag on move command now also skips `--force` confirmation prompt (#648)

## [0.13.10] - 2026-01-24

### Changed
- Microsprint commands are now deprecated and will be removed in a future release (#645)
  - All `gh pmu microsprint` subcommands display deprecation message and exit
  - No operations are performed

### Fixed
- Added test coverage for Claude workflow reminder in IDPF projects (#644)

## [0.13.9] - 2026-01-24

### Added
- Claude workflow reminder automatically prepended to issue bodies in IDPF projects (#638)

## [0.13.8] - 2026-01-24

### Changed
- Error message for unchecked checkboxes now includes Claude workflow reminder (#635)

## [0.13.7] - 2026-01-23

### Added
- `--repo` flag for `board` command to view boards from different repositories (#632)
- Kanban-style project focus documented in CLI help and README (#631)
- IDPF-Praxis framework integration note in documentation (#631)

## [0.13.6] - 2026-01-22

### Added
- Four new standard labels for workflow states (#628)
  - `emergency` - P0 emergency issue requiring immediate attention
  - `approval-required` - Requires explicit approval before proceeding
  - `blocked` - Issue is blocked by external dependency
  - `scope-creep` - Issue has grown beyond original scope

## [0.13.5] - 2026-01-22

### Added
- Three new standard labels: `security-required`, `legal-required`, `docs-required` (#626)
- `--state` flag for `board` command to filter by issue state (open/closed/all) (#620)
- E2E test coverage for release→branch migration (#625)

### Changed
- Label validation now enforces standard labels only (#626)
  - Non-standard labels cause errors with helpful message listing available options
  - Standard labels auto-created on demand with correct color and description
- Optimized `board` command using Search API for better performance (#620)
- Optimized `triage` command using Search API for label filtering (#621)
- Optimized `branch current` and `branch close` with two-phase query (#623)
- Refactored `intake` command to use SearchRepositoryIssues for consistency (#622)

### Fixed
- Complete release→branch terminology migration (#614)
- Empty else branch lint warning in triage command

## [0.13.4] - 2026-01-21

### Added
- Non-interactive mode for `gh pmu init` command (#609)
  - `--non-interactive` flag disables UI and requires all flags
  - `--project` flag for project number
  - `--repo` flag for repository (owner/repo format)
  - `--owner` flag for project owner (optional, inferred from repo)
  - `--framework` flag for framework type (defaults to IDPF)
  - `--yes` flag to auto-confirm prompts
- `test-plan` label added to standard labels (#611)

### Fixed
- E2E tests using invalid `--json` flag without field arguments

## [0.13.3] - 2026-01-21

### Added
- `qa-required` label to standard labels (#607)
- `--json` field selection for `view` and `list` commands (#599)
  - `--json` alone lists available fields
  - `--json=field1,field2` returns only specified fields
  - `--jq` flag for filtering JSON output

### Removed
- Microsprint cache in favor of direct tracker issue lookup (#602)
  - Removed `--refresh` flag from `microsprint list`
  - Simplified configuration by removing cache section
- Deprecated `gh pmu release` command (#509)
  - Use `gh pmu branch` instead
  - Removed `--release` flag from `move` command

### Fixed
- IDPF link in README.md now points to correct repository

## [0.13.2] - 2026-01-20

### Changed
- Renamed project text field from "Release" to "Branch" in defaults (#603)
- Updated all command help text to reference "Branch" field instead of "Release"
- Added backward compatibility for existing projects with "Release" field
  - Commands check for "Branch" field first, fall back to "Release"
  - No migration required for existing projects

## [0.13.1] - 2026-01-19

### Added
- Auto-create default labels when assigning non-existent labels (#596)
  - Labels defined in `defaults.yml` are automatically created if missing from repository
  - Warns user when label doesn't exist and isn't in defaults
- Added `prd`, `bug`, and `enhancement` labels to default label set (#595)
- `GetLabel()` method in defaults package for label lookup

### Changed
- Epic label color changed to lighter green (#3fb950) for better visibility

## [0.13.0] - 2026-01-19

### Added
- State filtering for `GetProjectItems` and `GetProjectItemsForBoard` - filter by issue state (OPEN/CLOSED) (#560)
- Auto-create custom fields and labels during `init` command (#587)
- E2E test infrastructure with command integration tests (#582)
- Label verification tests for branch and microsprint trackers

### Fixed
- Cursor-based pagination for `GetRepositoryIssues` to handle large result sets (#557)
- Cursor-based pagination for `GetIssuesByLabel` functions (#558)
- Pagination fallback in `GetSubIssuesBatch` for epics with >100 sub-issues (#559)
- Microsprint add retry for GitHub API eventual consistency
- Use Release field name instead of Branch during init

### Changed
- Extract `buildBatchMutationRequest` for improved testability (#554)
- Update framework to v0.26.3

## [0.12.1] - 2026-01-16

### Fixed
- Batch mutation HTTP 400 error when using `gh pmu move --branch` or `--status` - GraphQL request body now uses valid JSON format (#551)

## [0.12.0] - 2026-01-15

### Added
- Batch field mutations for `move` command - reduces API calls from O(N) to O(N/50) (#543)
- Batch sub-issue queries for recursive operations - fetches sub-issues per level instead of per-issue (#542)
- `GetProjectItemsByIssues` API method for targeted issue queries (#541)
- `GetSubIssuesBatch` API method for batch sub-issue fetching (#542)
- `BatchUpdateProjectItemFields` API method for batch field updates (#543)
- Repository-scoped filtering for `branch current`, `branch close`, `filter`, and `intake` commands (#545, #546, #547, #548)
- Benchmark tests for performance-critical code paths

### Changed
- `filter` command now uses targeted queries when issue URLs are available in stdin (#545)
- `intake` command uses repository filter for single-repo configurations (#548)
- Updated IDPF framework to v0.25.0

### Fixed
- Hook error message now references `--branch` instead of deprecated `--release`
- Removed unused `skippedIssues` variable in move command

## [0.11.1] - 2026-01-12

### Added
- `--state` filter flag for `list` command to filter by issue state (open/closed) (#522)
- `--remove-label` flag for `edit` command to remove labels from issues (#519)
- `--dry-run` flag for `branch close` and `microsprint close` commands (#527)
- DATE field type support in `SetProjectItemField` for date-based project fields (#518)
- Progress indicator for recursive move operations showing per-issue status (#520)
- Pagination support for `GetSubIssues` query to handle 50+ sub-issues (#521)
- Rate limiting protection with exponential backoff for bulk operations (#525)
- Status field value validation before setting, with helpful error messages (#523)

### Fixed
- `setNumberField` now parses value parameter instead of always setting 0 (#513)
- `--label` flag in `edit` command now actually adds labels (#528)
- `--interactive` flag removed from `create` command (was unimplemented stub) (#514)
- `branch close` now reports correct count of moved issues (#515)
- `GetSubIssueCounts` failures now warn and include items instead of silent exclusion (#516)
- Consolidated duplicate `openInBrowser` implementations into `internal/ui` (#517)
- Help text updated from "release" to "branch" terminology throughout (#512)
- Error messages updated to reference "branch" instead of "release" (#511)

### Changed
- `IsIDPF()` now uses case-insensitive prefix matching for framework detection (#524)
- Coverage threshold lowered from 80% to 70%

## [0.11.0] - 2026-01-08

### Changed
- Rename `gh pmu release` command to `gh pmu branch` (#508)
  - All subcommands remain the same: `start`, `current`, `add`, `remove`, `close`, `list`
  - Tracker issues now use `branch` label instead of `release`
- Change tracker label from `release` to `branch` (#505)
  - Migration support: `gh pmu init` automatically migrates legacy labels
- Remove unused `release.active` config section (#504)
  - Active releases now tracked via `branch` label on tracker issues
  - Simplifies config file and reduces maintenance overhead

### Documentation
- Update `docs/commands.md` to reflect `branch` command rename

## [0.10.1] - 2026-01-05

### Added
- `gh pmu comment` subcommand for adding comments to issues (#491)
  - `--body` / `-b` flag for inline comment text
  - `--body-file` / `-F` flag to read comment from file
  - `--body-stdin` flag to read comment from stdin
  - `--repo` / `-R` flag for cross-repository comments
- `--body-stdin` flag for `gh pmu create` command (#491)
  - Reads body content from stdin
  - Mutually exclusive with `--body` and `--body-file`
- `--repo` / `-R` flag for `gh pmu edit` command (#492)
  - Enables editing issues in repositories other than the configured default

### Changed
- `gh pmu move` help text now clearly indicates multiple issue support (#494)
  - Usage changed from `<issue-number>...` to `<issue-number> [issue-number...]`
  - Short description updated to "Update project fields for multiple issues at once"

### Fixed
- Document root cause of GitHub Project automation moving issues back to "In Progress" (#498)
  - "Pull request linked to issue" workflow triggers on `Fixes #XXX` cross-references
  - Recommended solution: Disable the automation or use `Refs #XXX` in PR bodies

### Documentation
- Migrate USER-EXTENSION content into prepare-release.md (#499)
  - Added Documentation Review checklist to post-analysis
  - Added Handle Incomplete Issues and Lint Gate to pre-validation
  - Added Coverage Configuration example to post-validation
  - Added Important Rules (6 guardrails) to pre-tag
  - Added Cleanup Assets and Post-Release Reminder to post-tag
- Add `edit` and `comment` commands to docs/commands.md
- Add `--body-stdin` flag documentation for `create` command

## [0.10.0] - 2026-01-03

### Added
- `--body-stdout` flag for `gh pmu view` command (#487)
  - Outputs raw issue body (markdown) directly to stdout
  - Enables AI assistants to capture body without temp file
- `--body-stdin` flag for `gh pmu edit` command (#488)
  - Reads body content from stdin (raw markdown)
  - Mutually exclusive with `--body` and `--body-file`
  - Supports piped input: `echo "$BODY" | gh pmu edit 123 --body-stdin`

### Changed
- Streamlined body editing workflow reduces tool calls from 5+ to 2-3 (#486)
- Update internal framework to v0.20.1 with charter support

## [0.9.7] - 2025-12-30

### Added
- `gh pmu edit` command with body-file support (#480)
  - `--body-file` / `-F` flag to read body from file
  - `--title` / `-t` flag to update issue title
  - `--body` / `-b` flag for inline body text
  - `--label` / `-l` flag to add labels (repeatable)
  - Completes body-file round-trip workflow: view → edit file → update
- Lint gate in `/prepare-release` workflow (#478)
  - Runs `golangci-lint` before tagging to prevent failed releases
  - New script: `.claude/scripts/prepare-release/lint.js`

### Changed
- `gh pmu release close` now defaults to current release (#479)
  - No argument required when exactly one release is active
  - Shows error with list when multiple releases exist
- Update framework to v0.17.1

## [0.9.6] - 2025-12-29

### Added
- `--body-file` / `-b` flag to `view` command (#474)
  - Writes issue body to `tmp/issue-{number}.md` for easy editing
  - Creates `tmp/` directory automatically if it doesn't exist
  - Outputs file path to stdout for scripting workflows

### Changed
- Migrate internal framework to v0.17.0

## [0.9.5] - 2025-12-28

### Changed
- Reorganize `.claude/scripts/` into purpose-based subdirectories (#469)
  - `close-release/` - Release monitoring scripts
  - `framework/` - Commit analysis, version recommendation, CI waiting
  - `open-release/` - Config verification
  - `prepare-release/` - Coverage analysis
  - `shared/` - Common utilities (lib, sprint scripts, etc.)
- Bump internal framework version to 0.16.0

### Added
- `/prepare-beta` command for beta release workflow
- `.claude/extensions/` directory for custom extension points

## [0.9.4] - 2025-12-26

### Fixed
- `gh pmu release close` no longer generates artifacts (#466)
  - Artifact generation moved to `/prepare-release` Phase 2
  - Prevents overwriting detailed release notes with auto-generated versions
  - Removed REQ-020 tests (AC-020-1 through AC-020-4)

## [0.9.3] - 2025-12-26

### Added
- Sprint and release management commands for Claude Code integration
  - `/assign-release`, `/close-release`, `/open-release`, `/switch-release`
  - `/plan-sprint`, `/end-sprint`, `/sprint-status`, `/sprint-retro`, `/transfer-issue`
- Beta deployment workflow proposal (#458)
  - Side-by-side `gh pmu` + `gh pmub` installation design
  - Prerelease versioning scheme (`v{current}+beta.N`)

### Changed
- `/prepare-release` command restored with full automation (#459)
  - Merged old automation scripts into new 4-phase structure
  - Added `--skip-coverage` flag support
  - Restored: config verification, commit analysis, CI waiting, release monitoring
  - Preserved: lifecycle diagram, `/close-release` integration

## [0.9.2] - 2025-12-23

### Added
- `/prepare-release` workflow now updates GitHub release notes from CHANGELOG (#439)
  - Parses CHANGELOG.md and updates corresponding GitHub release body
  - Cleans up old release assets, keeping only 3 most recent tagged releases
  - New scripts: `update-release-notes.js`, `cleanup-release-assets.js`

### Performance
- `gh pmu move` now caches project fields before bulk updates (#451)
  - Eliminates N+1 API calls when updating multiple fields per issue
  - 68-80% reduction in API calls for typical multi-field updates

### Fixed
- `release current` now shows accurate issue count (#449)
  - Uses project items with field filtering instead of label-based query
  - Fixes discrepancy between `release current` and `list --release current`


## [0.9.1] - 2025-12-23

### Fixed
- `gh pmu init` now detects all status and priority field values from project metadata (#442)
  - Previously hardcoded values missed custom options like "Parking Lot"
  - Now dynamically generates field mappings from actual project fields
  - Handles emoji prefixes (e.g., "🅿️ Parking Lot" → `parking_lot` alias)

## [0.9.0] - 2025-12-23

### Added
- Local caching for `release list` and `microsprint list` commands (#434)
  - Cached data stored in `.gh-pmu.yml` under `cache:` section
  - Cache automatically updated on `start`, `close`, `reopen` commands
  - Use `--refresh` flag to force API fetch and update cache
  - ~6x performance improvement (~118ms cached vs ~780ms uncached)
- Coverage gate for `/prepare-release` workflow
  - Analyzes patch coverage against changed lines since last tag
  - Configurable threshold in `.gh-pmu.yml` under `release.coverage.threshold`

### Fixed
- Test isolation to prevent `.gh-pmu.yml` corruption (#436)
  - All tests that call `cfg.Save()` now use isolated temp directories
  - Added canary test to detect future isolation failures

## [0.8.6] - 2025-12-22

### Fixed
- GraphQL mutation "Missing type definition for variable" error for close/reopen/update operations (#425)
  - shurcooL graphql library requires named struct types for input variables
  - Added `CloseIssueInput`, `ReopenIssueInput`, `UpdateIssueInput` named types
- Temp file utilities now use project root directory instead of OS temp (#427)
  - `config.CreateTempFile()` creates files in `tmp/` alongside `.gh-pmu.yml`
  - Directory automatically added to `.gitignore`

### Changed
- Release close now skips issues in "Parking Lot" status (#426)
  - Parking Lot issues are listed separately in output but not moved to backlog
  - Prevents accidentally moving parked work when closing releases

## [0.8.5] - 2025-12-22

### Fixed
- GraphQL type mismatch in `GetRepositoryIssues` - now uses `IssueState` enum instead of `String` (#422)
- Checkbox validation now ignores checkboxes inside code blocks (#419)
  - Fenced code blocks (``` or ~~~) and indented code blocks are excluded
  - Example checkboxes in documentation no longer trigger validation failures
- Data race in mock handler prevented with mutex (#421)

### Changed
- Test coverage improvements with wrapper tests and TESTING.md documentation (#414, #417)

## [0.8.4] - 2025-12-22

### Added
- Microsprint tracker issues now include descriptive body explaining purpose and commands (#407)
  - Similar to release tracker bodies added in v0.8.2

### Fixed
- License attribution moved to NOTICE file per Apache 2.0 requirements (#402, #403)
- Codecov integration improvements for accurate diff coverage (#393, #394)

### Changed
- Improved test coverage for cmd package with mock client implementations (#395-#401, #405, #406)

## [0.8.3] - 2025-12-21

### Performance
- `list --has-sub-issues` optimized from N+1 API calls to single batch query per repository (#389)
- `close --update-status` optimized with direct issue-to-project-item lookup (3 calls → 2 calls) (#390)

### Added
- README badges: CI, Codecov, Go Report Card, Release, License (#388)

### Changed
- Coverage reporting now via Codecov badge instead of committed reports (#388)
- Removed `coverage` job from CI workflow (#388)
- Deleted `coverage/` directory from repository (#388)

## [0.8.2] - 2025-12-21

### Added
- Release tracker issues now include descriptive body with linked issues (#386)

### Performance
- `gh pmu view` command optimized from ~4.5s to ~0.8s (6x faster) (#386)

## [0.8.1] - 2025-12-21

### Fixed
- `release list` now finds releases with any branch name format (#376)
  - Previously only matched titles starting with "Release: v"
  - Now matches all "Release: " prefixed titles (e.g., "Release: release/v1.0.0")
- Release tests no longer corrupt repository `.gh-pmu.yml` file (#378)
  - Tests now use isolated temp directories for config operations
- Documentation incorrectly claimed `duplicate` close reason exists (#379)
  - Removed non-existent `duplicate`/`dupe` aliases from docs
  - Added note explaining GitHub API only supports `completed` and `not_planned`

### Changed
- CI pipeline now verifies config file integrity after tests
  - Detects test corruption patterns (testowner, testrepo values)
  - Fails build if tests modify `.gh-pmu.yml`

## [0.8.0] - 2025-12-21

### Added
- `gh pmu validation` command group for status transition validation
  - `validation rules` - Display configured transition rules
  - `validation check <from> <to>` - Test if a transition is allowed
  - `validation enable/disable` - Toggle validation enforcement
- Status transition validation integrated into `move`, `create`, `release`, and `microsprint` commands
  - Configurable via `validation` section in `.gh-pmu.yml`
  - Default rules enforce logical workflow progressions
  - `--force` flag available to bypass validation when needed
- `GetAllowedTransitions` API query for retrieving valid status transitions
- Extended configuration schema with `validation` section for custom transition rules
- CodeQL security scanning workflow for automated vulnerability detection

### Fixed
- Config file no longer overwritten by field tests
- Add bounds checking for int to int32 conversion in GraphQL queries
- Remove unused code and fix empty branch lint errors

## [0.7.5] - 2025-12-19

### Added
- `--release` flag to `move` command - set Release field directly (#349)
- `--microsprint` flag to `move` command with `--sprint` alias (#349)
- `--backlog` flag to `move` command - clear Release and Microsprint fields (#349)

### Changed
- **BREAKING**: `release start` now requires `--branch` flag instead of `--version` (#346, #348)
  - Migration: `--version 2.0.0` → `--branch release/v2.0.0`
  - Branch name used literally for tracker title, Release field, and artifact directory
  - Supports any branch convention: `release/v2.0.0`, `patch/v1.9.1`, `hotfix-auth-bypass`

### Removed
- **BREAKING**: Removed entire `patch` command group (~1750 lines) (#346, #348)
  - Migration: `gh pmu patch start --version 1.9.1` → `gh pmu release start --branch patch/v1.9.1`
  - Migration: `gh pmu patch add 42` → `gh pmu release add 42`
  - Migration: `gh pmu patch close --tag` → `gh pmu release close --tag`

### Fixed
- Sub-issues no longer inherit parent labels by default (#359)

## [0.7.4] - 2025-12-19

### Changed
- Remove `--name` and `--track` flags from `release start` command (#345)
  - Simplifies release command interface
  - Interactive mode still available for version selection

### Fixed
- Improve git command error messages (#347)
  - `GitAdd`, `GitTag`, `GitCommit` now capture and return stderr on failure
  - Users see actual git error messages instead of generic "exit status 1"

## [0.7.3] - 2025-12-15

### Added
- Multiple issue support for `move` command (#343)
  - Accept multiple issue numbers: `gh pmu move 42 43 44 --status done`
  - Sequential processing to avoid API rate limits
  - Graceful error handling - individual failures don't stop remaining issues
  - Summary output for bulk operations

### Fixed
- Add retry logic for transient HTTP errors in release scripts


## [0.7.2] - 2025-12-15

### Fixed
- Correct GraphQL field name for `CreateProjectV2Field` mutation

## [0.7.1] - 2025-12-15

### Added
- Additional `gh issue create` options in `gh pmu create` command (#325)
  - `--body-file` / `-F` - Read body text from file (use "-" for stdin)
  - `--editor` / `-e` - Open editor to compose body
  - `--template` / `-T` - Use issue template from `.github/ISSUE_TEMPLATE/`
  - `--web` / `-w` - Open browser after creating issue
- Mutual exclusivity validation for body input options

## [0.7.0] - 2025-12-15

### Added
- **Release track configuration** - Configure track prefix for release artifacts (#321)
  - `release.tracks[].prefix` in `.gh-pmu.yml` for artifact directory naming
  - Supports both versioned (`v1.2.0/`) and prefixed (`main-v1.2.0/`) directory structures
- **Configurable artifact generation** - Control what artifacts are generated during release close (#323)
  - `release.artifacts.directory` - Custom base directory (default: "Releases")
  - `release.artifacts.release_notes` - Toggle release-notes.md generation
  - `release.artifacts.changelog` - Toggle changelog.md generation
- **Patch version validation** - Validate patch versions and enforce label constraints (#322)
  - Error if issue has `breaking-change` label (incompatible with patch releases)
  - Warning if issue lacks `bug`/`fix`/`hotfix` label
  - Validates version is a valid patch increment from latest git tag
- **Init command enhancements** - Improved project initialization workflow
  - Auto-create `Release` and `Microsprint` project fields if missing
  - Auto-create `release` and `microsprint` repository labels if missing
  - Sync active releases from existing tracker issues to `release.active[]`
- **Interactive version selection** - Prompt for version during `release start`
  - Shows latest git tag for reference
  - Validates semver format before proceeding

### Fixed
- Remove unused `name` field from `patchStartOptions` struct
- Fix ineffectual assignment to `fields` in init command

## [0.6.0] - 2025-12-14

### Added
- `gh pmu microsprint` command group for AI-assisted development workflows
  - `microsprint start` - Start a new microsprint with auto-generated naming (YYYY-MM-DD-a pattern)
  - `microsprint add <issue>` - Add issue to current microsprint via Text field
  - `microsprint remove <issue>` - Remove issue from microsprint
  - `microsprint current` - Show current active microsprint details
  - `microsprint close` - Close microsprint with artifact generation (review.md, retro.md)
  - `microsprint list` - List microsprint history (open and closed)
  - `microsprint resolve` - Resolve multiple active microsprint conflicts
- Interactive retrospective prompts during `microsprint close`
  - Prompts for: What Went Well, What Could Be Improved, Action Items
  - `--skip-retro` flag to generate empty template without prompts
  - `--commit` flag to auto-commit generated artifacts
- Team-wide microsprint model with multi-user support
  - Join/Work without/Cancel prompts when another user has active microsprint
  - Confirmation prompt before closing another user's microsprint
- `gh pmu release` command group for version-based deployment workflows (IDPF-Structured)
  - `release start --version` - Start a new release with semver validation
  - `release add/remove <issue>` - Manage issues in release
  - `release current` - Show current release details
  - `release close` - Close release with artifact generation
  - `release list` - List release history
- `gh pmu patch` command group for hotfix deployment workflows (IDPF-LTS)
  - Same subcommand structure as release command
- Integration with existing `move` and `create` commands via `--microsprint` flag
- Artifact generation in `Microsprints/{name}/` directory

### Fixed
- Dynamic date handling in microsprint tests for reliable CI across timezones

### Documentation
- Added upgrade instructions to README

## [0.5.3] - 2025-12-12

### Added
- `--repo` / `-R` flag to `view` command for explicit repository specification
- `--repo` / `-R` flag to `close`, `split`, and `sub remove` commands
  - Completes `--repo` flag support across all commands
  - Enables consistent cross-repository workflows

### Documentation
- Restructured README with concise format and linked How-To guides
- New documentation files: configuration.md, commands.md, sub-issues.md, batch-operations.md, development.md
- Added CONTRIBUTING.md for contributor guidelines
- Added "Unique Capabilities" section highlighting gh-pmu specific flags

## [0.5.2] - 2025-12-12

### Added
- `--repo` / `-R` flag to `list` command for filtering by repository (#268)
  - Provides consistency with `move --repo` and `sub list --repo`
  - Flag takes precedence over config file setting
- `filter` subcommand for piping and filtering issue data (#267)
  - Reads JSON from stdin (piped from `gh issue list --json ...`)
  - Filters by project field values (`--status`, `--priority`)
  - Additional filters: `--assignee`, `--label`
  - Output as table (default) or JSON (`--json`)
  - Example: `gh issue list -R owner/repo --json number,title | gh pmu filter --status ready`

## [0.5.1] - 2025-12-11

### Added
- `--repo` / `-R` flag to `sub list` command for explicit repository specification (#266)
  - Follows same pattern as `move --repo` and `sub add --repo`
  - Enables querying sub-issues from any repository without changing config

## [0.5.0] - 2025-12-11

### Added
- `gh pmu field` command - Manage custom project fields
  - `field create <name>` - Create project fields (text, number, date, single_select)
  - `field list` - List all fields in the project
  - Supports single-select fields with custom options via `--option` flag
  - Auto-updates `.gh-pmu.yml` metadata after field creation
- `--repo` / `-R` flag to `move` command for explicit repository specification (#265)
  - Matches behavior of `sub add` and other commands
  - Enables cross-repo workflows and CI/CD scripting

## [0.4.5] - 2025-12-10

### Fixed
- Help text no longer claims unavailable "project templating" feature (#261)
  - Updated description to: "gh-pmu streamlines GitHub project workflows with unified issue tracking and sub-issue hierarchy."

## [0.4.4] - 2025-12-10

### Added
- `--browser` flag to open history in web browser (#259)
  - Generates styled HTML with GitHub dark theme
  - Clickable commit hashes and issue references
  - Works on Windows, macOS, and Linux
- `--files` flag to show affected files in directory history (#260)
  - Compact view now shows file count: `(3 files)` suffix
  - Full file list displayed with `--files` flag

## [0.4.3] - 2025-12-10

### Added
- Commit body and GitHub comments in detailed single-file history view (#258)
  - Shows full commit message body (multi-line description after subject)
  - Fetches and displays GitHub commit comments via API
  - Common trailers (Co-Authored-By, Signed-off-by) filtered from display

## [0.4.2] - 2025-12-10

### Added
- Config file search now walks up directory tree (#257)
  - Run `gh pmu` commands from any subdirectory of a repo with `.gh-pmu.yml`
  - Similar behavior to how git finds `.git` directories

## [0.4.1] - 2025-12-10

### Added
- Detailed single-file history view with expanded commit information (#256)
  - Shows full commit messages, relative timestamps ("3 days ago"), and line change stats (+/-)
  - Automatically activates for single file paths
  - `--compact` flag to use original compact format for single files

## [0.4.0] - 2025-12-10

### Added
- `gh pmu history` command - Show git commit history for files/directories with issue references (#255)
  - Parse issue/PR references from commit messages (#123, fixes #456)
  - Infer change type from commit prefixes (Fix:, Add:, feat:, etc.)
  - Lipgloss styled terminal output with color-coded change types
  - JSON output format (`--json`)
  - Markdown file output (`--output`) to History/ directory
  - Safety protections: 25 file limit, repo root guard (`--force` to override)
  - Filter options: `--since`, `--limit`

### Dependencies
- Added `github.com/charmbracelet/lipgloss` v1.1.0 for styled terminal output

## [0.3.1] - 2025-12-09

### Changed
- Repository moved from `scooter-indie/gh-pmu` to `rubrical-works/gh-pmu` (#254)
- Installation command updated: `gh extension install rubrical-works/gh-pmu`
- Go module path updated to `github.com/rubrical-works/gh-pmu`

### Note for Existing Users
Existing installations will continue to work via GitHub redirect. For a clean setup, reinstall:
```bash
gh extension remove gh-pmu
gh extension install rubrical-works/gh-pmu
```

## [0.3.0] - 2025-12-08

### Added
- `gh pmu board` command - Interactive terminal-based project board view (#250)
- `gh pmu close` command with reason aliases (completed, not_planned, duplicate, reopened) (#249)
- Integration testing infrastructure with `internal/testutil` package
- Comprehensive integration tests for all commands (list, view, create, move, intake, triage, split, sub add/create/list/remove, init)
- GitHub Actions workflow for integration tests

### Fixed
- `os.Chmod` error handling for errcheck linter compliance
- CI workflow improvements for gh extension installation and authentication

### Changed
- Disabled integration tests workflow (pending test environment fixes)

### Documentation
- Added shell construct limitations guide to `gh-workflow.md` (#252)
- Added integration testing guide for test fixtures

## [0.2.13] - 2025-12-04

### Added
- 31 command enhancements (#94-124)

### Fixed
- Use `t.Fatal` for nil checks to satisfy staticcheck

## [0.2.12] - 2025-12-04

### Fixed
- Staticcheck lint error in triage tests (SA5011) - use `t.Fatal` for nil checks

### Notes
- v0.2.11 release failed to build binaries due to lint error; this release includes all v0.2.11 changes plus the fix

## [0.2.11] - 2025-12-04

### Added
- `--repo` / `-R` flag to `triage` command for targeting specific repositories (#91)

### Fixed
- `GetProjectItems` now uses cursor-based pagination to fetch all items (#90)
  - Previously limited to first 100 items, causing "issue not in project" errors for large projects

### Documentation
- Clarified distinction between labels and project fields in `gh-workflow.md`
- Added pagination integration test scenario to backlog (IT-2.4)

## [0.2.10] - 2025-12-04

### Added
- Comprehensive tests for `cmd/intake.go` output functions (100% coverage)
- Comprehensive tests for `cmd/split.go` output functions (100% coverage)
- Comprehensive tests for `cmd/init.go` helper functions (75-100% coverage)
- Comprehensive tests for `internal/ui/ui.go` spinner methods (96.9% coverage)
- Comprehensive tests for `cmd/view.go` output functions
- Comprehensive tests for `cmd/create.go` runCreate function
- Comprehensive tests for `cmd/move.go` core logic
- Comprehensive tests for `cmd/sub.go` output functions
- Comprehensive tests for `internal/api/mutations.go`
- Comprehensive tests for `cmd/triage.go` command
- Integration testing proposal (`Proposal/PROPOSAL-Automated-Testing.md`)
- Integration testing backlog with 23 stories, 90 story points

### Changed
- Test coverage increased from ~15% to 63.6%
- Fixed golangci-lint errcheck warnings for `os.Chdir` in deferred calls
- Renamed test fixtures from `.gh-pm.yml` to `.gh-pmu.yml`

### Coverage (v0.2.10)
| Package | Coverage |
|---------|----------|
| `internal/api` | 96.6% |
| `internal/config` | 97.0% |
| `internal/ui` | 96.9% |
| `cmd` | 51.2% |
| **Total** | **63.6%** |

## [0.2.9] - 2025-12-03

### Added
- Comprehensive test coverage for triage command

### Changed
- Format all Go files with `gofmt -s`

## [0.2.8] - 2025-12-03

### Added
- Generate Markdown coverage report instead of HTML (`coverage/README.md`)

### Changed
- Remove HTML coverage report in favor of Markdown

## [0.2.7] - 2025-12-03

### Added
- Coverage report generation on releases

### Fixed
- Use `-short` flag in coverage tests to skip auth-dependent tests
- Calculate box width using visible text length (strip ANSI codes)
- Use rune count for visible width calculation in box formatting

## [0.2.6] - 2025-12-03

### Fixed
- Use binary format in goreleaser for gh extension compatibility
- Add Windows support to goreleaser config

## [0.2.5] - 2025-12-03

### Changed
- Consolidate CI workflows with sequential execution

### Fixed
- Format ui.go and remove unused noColor field in Spinner

## [0.2.4] - 2025-12-03

### Added
- Enhanced init UX with project discovery and styled output (Story 1.13)
  - Auto-detect repository from git remote
  - Query GitHub API for associated projects
  - Present numbered list for project selection
  - Styled output with spinners, boxes, and color coding

### Fixed
- Correct gh-sub-issue attribution to yahsan2
- Format cmd/init.go

## [0.2.3] - 2025-12-03

### Changed
- Switch to goreleaser for releases

## [0.2.2] - 2025-12-03

### Fixed
- Simplify release workflow - use default build

## [0.2.1] - 2025-12-03

### Fixed
- init command now creates complete config file (#71)
- Use t.Fatal for nil flag checks (SA5011)
- Address golangci-lint errors
- Format code and downgrade deps for Go 1.22/1.23 compatibility
- Set go.mod to 1.22 for CI compatibility

## [0.2.0] - 2025-12-03

### Added
- Mirror gh-pm CI/CD workflows (#69)
- Release workflow for gh extension install

### Changed
- Rename to gh-pmu for shorter command

### Fixed
- Skip auth-dependent tests in CI
- Use binary format for gh extension install compatibility

## [0.1.0] - 2025-12-03

### Added - Initial Release
- **Project scaffolding**
  - Go module with Cobra CLI framework
  - Makefile with build, test, lint targets
  - GitHub Actions for CI/CD

- **Configuration package** (`internal/config`)
  - Load `.gh-pmu.yml` with Viper
  - Validate required fields
  - Support field aliases
  - Cache project metadata from GitHub API

- **GitHub API client** (`internal/api`)
  - GraphQL client using go-gh library
  - Support for `sub_issues` and `issue_types` feature headers
  - Queries: GetProject, GetProjectFields, GetProjectItems, GetIssue, GetSubIssues, GetParentIssue, GetRepositoryIssues
  - Mutations: CreateIssue, AddIssueToProject, SetProjectItemField, AddSubIssue, RemoveSubIssue

- **Project Management Commands**
  - `gh pmu init` - Interactive project configuration setup
  - `gh pmu list` - List issues with project field values
  - `gh pmu view` - View issue with all project fields and sub-issues
  - `gh pmu create` - Create issue with project fields pre-populated
  - `gh pmu move` - Update issue project fields

- **Sub-Issue Commands**
  - `gh pmu sub add` - Link existing issue as sub-issue
  - `gh pmu sub create` - Create new sub-issue under parent
  - `gh pmu sub list` - List sub-issues with completion count
  - `gh pmu sub remove` - Unlink sub-issue from parent

- **Batch Operations**
  - `gh pmu intake` - Find and add untracked issues to project
  - `gh pmu triage` - Bulk update issues based on config rules
  - `gh pmu split` - Create sub-issues from checklist or arguments

- **Enhanced Integration**
  - Cross-repository sub-issues
  - Sub-issue progress tracking (progress bar in view)
  - Recursive operations (`--recursive` flag for move)

### Attribution
- Based on [gh-pm](https://github.com/yahsan2/gh-pm) and [gh-sub-issue](https://github.com/yahsan2/gh-sub-issue) by [@yahsan2](https://github.com/yahsan2)

---

## Development History

### Pre-release (2025-11-30 to 2025-12-02)

- 2025-11-30: Initial repository setup, README.md
- 2025-12-01: Add proposal document
- 2025-12-02: Add Agile PRD, product backlog, code integration inventory
- 2025-12-02: Complete Sprint 1 (Foundation), Sprint 2 (Core Commands), Sprint 3 (Batch Operations)
- 2025-12-03: Complete Sprint 5 (Enhanced Integration)
- 2025-12-03: Remove Epic 2 & 4 (GitHub API limitations - no view creation API)

---

## Sprint Summary

| Sprint | Focus | Status |
|--------|-------|--------|
| Sprint 1 | Foundation, init, list | Complete |
| Sprint 2 | view, create, move, sub-issues | Complete |
| Sprint 3 | intake, triage, split | Complete |
| Sprint 4 | Epic 2 (Templates) | Removed - API limitations |
| Sprint 5 | Enhanced Integration | Complete |
| Sprint 6 | Test Coverage | Complete (63.6%) |

---

## API Limitations Discovered

### GitHub Projects V2 API
- **No `createProjectV2View` mutation** - Views cannot be created programmatically
- **Status field reserved** - New projects have a default Status field that cannot be replaced
- **Workflows not accessible** - Project automation workflows cannot be read or written via API

These limitations led to removing Epic 2 (Project Templates) from scope.

[Unreleased]: https://github.com/rubrical-works/gh-pmu/compare/v0.15.1...HEAD
[0.15.1]: https://github.com/rubrical-works/gh-pmu/compare/v0.15.0...v0.15.1
[0.8.3]: https://github.com/rubrical-works/gh-pmu/compare/v0.8.2...v0.8.3
[0.8.2]: https://github.com/rubrical-works/gh-pmu/compare/v0.8.1...v0.8.2
[0.8.1]: https://github.com/rubrical-works/gh-pmu/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/rubrical-works/gh-pmu/compare/v0.7.5...v0.8.0
[0.7.5]: https://github.com/rubrical-works/gh-pmu/compare/v0.7.4...v0.7.5
[0.7.4]: https://github.com/rubrical-works/gh-pmu/compare/v0.7.3...v0.7.4
[0.7.3]: https://github.com/rubrical-works/gh-pmu/compare/v0.7.2...v0.7.3
[0.7.2]: https://github.com/rubrical-works/gh-pmu/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/rubrical-works/gh-pmu/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/rubrical-works/gh-pmu/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/rubrical-works/gh-pmu/compare/v0.5.3...v0.6.0
[0.5.3]: https://github.com/rubrical-works/gh-pmu/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/rubrical-works/gh-pmu/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/rubrical-works/gh-pmu/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/rubrical-works/gh-pmu/compare/v0.4.5...v0.5.0
[0.4.5]: https://github.com/rubrical-works/gh-pmu/compare/v0.4.4...v0.4.5
[0.4.4]: https://github.com/rubrical-works/gh-pmu/compare/v0.4.3...v0.4.4
[0.4.3]: https://github.com/rubrical-works/gh-pmu/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/rubrical-works/gh-pmu/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/rubrical-works/gh-pmu/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/rubrical-works/gh-pmu/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/rubrical-works/gh-pmu/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.13...v0.3.0
[0.2.13]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.12...v0.2.13
[0.2.12]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.11...v0.2.12
[0.2.11]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.10...v0.2.11
[0.2.10]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.9...v0.2.10
[0.2.9]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.8...v0.2.9
[0.2.8]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.7...v0.2.8
[0.2.7]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.6...v0.2.7
[0.2.6]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.5...v0.2.6
[0.2.5]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/rubrical-works/gh-pmu/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/rubrical-works/gh-pmu/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/rubrical-works/gh-pmu/releases/tag/v0.1.0
