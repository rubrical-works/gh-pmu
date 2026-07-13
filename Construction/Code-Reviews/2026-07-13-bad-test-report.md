# Bad Test Review Report

**Date:** 2026-07-13
**Mode:** `--full` (manifest bypassed)
**Tests reviewed:** 67 files (all; 9 new since last run 2026-03-12, 58 re-examined)
**Tests skipped:** 0
**Findings:** 73 total (0 high, 28 medium, 45 low)
**Bug issues created:** 8 (#875–#882)

The suite is fundamentally healthy: no hardcoded-return or true mock-only hollowness — mocks sit at the transport boundary and real logic executes; integration/e2e tests run the real binary against live GitHub state with independent read-back in the strong files. Findings cluster into six recurring patterns rather than isolated bad tests.

## High Severity

None.

## Medium Severity

| Test File | Test Name | Concern | Evidence (abridged) | Issue |
|-----------|-----------|---------|---------------------|-------|
| cmd/create_test.go | TestLabelMerging_* (4) | Tautological — simulates merge inline | Real merge (create.go:256) never invoked; mock never captures labels | #878 |
| cmd/filter_test.go | TestRunFilterWithDeps_FilterBy* (5) | Assertion-free on stale premise | Output capturable via OutOrStdout; only err==nil checked | #875 |
| cmd/intake_test.go | TestOutputIntakeTable (truncation) | Assertion-free on stale premise | Buffer set then ignored; truncation never verified | #875 |
| cmd/intake_test.go | TestOutputIntakeJSON (status/count) | Named properties never checked | status/count never read back | #875 |
| cmd/list_test.go | TestOutputTable_TitleTruncation et al. | Assertion-free on stale premise | outputTable/outputJSON write to OutOrStdout; SetOut already called | #875 |
| cmd/split_test.go | TestSplitJSONOutput_Structure | Self-referential round-trip | outputSplitJSON never invoked; JSON shape verified nowhere | #878 |
| cmd/sub_test.go | TestSubCreateOptions_Defaults | Certifies no-op flags | --inherit-assignees/--inherit-milestone never read in production (#867) | #880 |
| cmd/triage_test.go | TestSearchIssuesForTriage_QueryParsing | Self-referential — reimplements logic | Asserts against its own inline copy of state detection | #878 |
| cmd/uat_epic1_test.go | TestUAT_SplitIssue_Workflow | Stale — nonexistent interface | split --task flag doesn't exist; asserted string never emitted | #876 |
| cmd/uat_epic1_test.go | 4 tests with bare `view --json` | Stale — flag parse error | --json is StringVar without NoOptDefVal; exit 0 impossible | #876 |
| cmd/uat_epic3_test.go | TestUAT_ProgressTracking_WithSubIssues | Stale — asserts field never emitted | 'totalCount' doesn't exist in view JSON schema | #876 |
| cmd/uat_epic3_test.go | _NoSubIssues / _MultiLevelTreeWorkflow | Stale — bare --json | Same parse-error defect | #876 |
| (suite) .github/workflows/integration-tests.yml.disabled | all cmd/*_integration_test.go | Never runs in CI | Only -tags=integration workflow is disabled | #876 |
| cmd/list_integration_test.go | TestRunList_Integration_FilterBy* (7) | Positive-only filter checks | No exclusion assertions; comments acknowledge the gap | #879 |
| cmd/triage_integration_test.go | TestRunTriage_Integration_NamedConfigDryRun | Cannot-fail conditional | `if ExitCode==0 {assert}` — any failure passes | #882 |
| cmd/triage_integration_test.go | TestRunTriage_Integration_JSONOutput | JSON never parsed | Only `Stdout != ""`; --apply mutation unverified | #882 |
| cmd/sub_create_integration_test.go | _NoInheritLabels | Negative behavior unverified | Asserts only "Created sub-issue" | #880 |
| cmd/sub_create_integration_test.go | _InheritAssignees | Vacuously tested | Parent has no assignees; only exit 0 asserted | #880 |
| cmd/init_integration_test.go | _ProjectValidation | Validation never exercised | No invalid project supplied; run error ignored | #879 |
| internal/api/integration_test.go | entire file | Does not compile under its tag | NewClient() assignment mismatch; confirmed via go vet | #876 |
| internal/api/mutations_test.go | TestGitCommit_ErrorMessageIncludesGitOutput | Side effect + vacuous path | Can create a REAL commit with staged changes; success auto-passes | #881 |
| internal/ui/ui_test.go | TestSpinner_StartStop | Misses known restart panic | Start→Stop→Start (the panicking path) untested | #877 |
| internal/ui/ui_test.go | TestSpinner_UpdateMessage | Concurrency never exercised | Sequential calls; -race cannot trip ui.go:323 race | #877 |
| test/e2e/board_test.go | TestBoardWithFilter | No effect verification | No contrast issue; ignored --status passes | #879 |
| test/e2e/branch_test.go | TestBranchLifecycle | Membership never verified | move asserts exit 0 only; soft close check | #879 |
| test/e2e/filter_test.go | TestFilterByPriority | No exclusion assertions | Ignored --priority passes both subtests | #879 |
| test/e2e/filter_test.go | TestFilterCombined | AND semantics unproven | Single issue, inclusion-only | #879 |
| test/e2e/init_test.go | TestInitNonInteractiveWithOwner | --owner effect unverified | Config content never read; owner equals default | #879 |

## Low Severity

45 findings, all documented with evidence in the review working files; grouped summary (Issue column shows the grouped bug where the low was folded in, else "No issue"):

| Test File | Tests / Pattern | Concern | Issue |
|-----------|-----------------|---------|-------|
| cmd/create_test.go | TestCreateOptions_* | Zero-value tautology | #878 |
| cmd/filter_test.go | TestOutputFilterTable_WithIssues/_TitleTruncation | Assertion-free (stale premise) | #875 |
| cmd/edit_test.go | _UsesRepoFromOptions/_CrossRepoEditing | Repo override unobservable in mock | No issue |
| cmd/close_test.go | _WithRepoOverride | Override not captured by mock | No issue |
| cmd/board_test.go | _WithStatusFilter | Presence-only assertion | No issue |
| cmd/field_test.go | TestRunFieldCreate_* | Creation args never verified in mock | No issue |
| cmd/history_test.go | TestCommitInfo_JSONMarshalling, TestCommitComment_Fields | Tautological | #878 |
| cmd/history_test.go | TestOutputHistoryJSON_*, TestValidateHistorySafety_NormalPath | Narrow assertions (os.Stdout genuinely uncapturable here; cwd-dependent) | No issue |
| cmd/init_test.go | TestValidateProject_* | Dead-code test (validateProject has no callers) | No issue |
| cmd/list_test.go | TestRunListWithDeps_With*Filter (11) | Wiring unverified (pure filters are table-tested) | #875 |
| cmd/list_test.go | _FunctionSignatureCheck; _EmptyItems/_NilIssueItems | Compile-time-only; auth-gated for pure logic | No issue |
| cmd/split_test.go | TestOutputSplitJSON*; TestSplitOptions | err==nil only; zero-value tautology | #875 / #878 |
| cmd/sub_test.go | TestOutputSubListJSON_*/Table_* (8) | Dead-code tests (legacy output funcs) | No issue |
| cmd/triage_test.go | TestListTriageConfigs/TestOutputTriage*; TestTriageOptions | err==nil only; zero-value tautology | #875 / #878 |
| cmd/validation_test.go | TestCountCheckedBoxes/TestCountCodeBlockCheckboxes | Dead-code tests | No issue |
| cmd/view_test.go | TestOutputView* (~16); TestViewJSONOutput_* round-trips | Assertion-free (stale premise); self-referential | #875 / #878 |
| cmd/wrapper_test.go | TestRun*_LoadsConfig (~12); _WithUpdateStatus_ConfigNotFound | Near-unfalsifiable; zero assertions | No issue |
| cmd/init_integration_test.go | _FieldMetadataFetching/_ConfigFileCreation | Failure converted to skip | #879 |
| cmd/intake_integration_test.go | _NoUntracked; _DryRun | Smoke test; dry-run non-mutation unverified | No issue |
| cmd/triage_integration_test.go | dry-run tests; _NoQueryOrConfig | Non-mutation unverified; accepts every outcome | #882 |
| cmd/sub_create_integration_test.go | _InheritLabels | Verifies CLI's own output, not GitHub state | #880 |
| internal/api/client_test.go | _ReturnsErrorOnAuthFailure; TestNewClient_ReturnsClient | Vacuous/environment-conditional; tautological | No issue |
| internal/api/mutations_test.go | _WithAssignees/_WithMilestone | Resolved IDs never verified in mutation input | No issue |
| internal/api/queries_test.go | *_EmptyInput (3) | Auth-gated tests for pure early-return logic | No issue |
| internal/config/config_test.go | TestValidate_Missing* | Comments claim message checks not performed | No issue |
| internal/ui/ui_test.go | "clears line on stop"; TestUI_SummaryBox | Assertion satisfied by every frame; Contains-only | #877 |
| internal/version/version_test.go | TestVersion_* | Constant pin with weak semver check | #878 |
| test/e2e/board_test.go | TestBoardRendering | Column placement unchecked | #879 |
| test/e2e/cleanup_test.go | TestCleanupE2EIssues | Env-gated utility, never runs, no assertions | No issue |
| test/e2e/filter_test.go | TestFilterByBranch ("current") | Inclusion-only for keyword path | #879 |
| test/e2e/init_test.go | _NonInteractiveMode; _ExistingConfigNoYes | Silent skip on missing field; YAML path reuses JSON filename (setup bug) | #879 |
| test/e2e/view_test.go | TestViewJSONWithProjectFields | Non-empty check where exact value was set up | No issue |
| test/e2e/workflow_test.go | TestSubIssueWorkflow; TestForceYesWorkflowWarning | Conditional assertion; self-reported status | No issue |

## Clean Tests

30 files passed review with no concerns: cmd/accept, acceptance_gate, branch, comment, config, integrity_check, label, move, root (unit); cmd/create, edit, move, split, sub_add, sub_list, sub_remove, view (integration); internal/api/errors, fields_fallback, retry, retry_client, retry_integration, validate; internal/config/acceptance, defaults/embed, framework/detect, integrity/integrity; test/e2e/config, e2e, helpers.

Standout suites: `cmd/move_test.go` (call counts, depth limits, --force semantics), `cmd/edit_integration_test.go` and `cmd/move_integration_test.go` (independent read-back incl. negative checks), `internal/api/fields_fallback_test.go` (#853 fallback engagement/non-engagement), `test/e2e/workflow_test.go`.

## Issues Created

- #875 — Hollow tests: ~35 assertion-free output tests across 6 cmd test files (stale os.Stdout comments)
- #876 — Dormant integration/UAT infrastructure: disabled CI workflow, non-compiling integration_test.go, stale UAT invocations
- #877 — Spinner tests skip the known panic/race paths (companion to #870)
- #878 — Tautological/self-referential tests never exercising production code
- #879 — Positive-only filter assertions in e2e/integration tests
- #880 — Inherit-flag tests certify no-op flags (companion to #867)
- #881 — TestGitCommit side effect: can create a real commit, asserts nothing on success
- #882 — Hollow triage integration tests

## Charter Alignment Notes

- 66 of 67 test files align with charter-scoped features (field management, sub-issues, batch ops, branch tracking, board, labels, config integrity, status transitions, E2E infrastructure).
- 1 unaligned (informational): `cmd/history_test.go` — the history command (git-log reporting/HTML export) does not map to any charter-listed v1.4.x feature; its helper tests are otherwise genuine.

---
*Generated by /bad-test-review --full on 2026-07-13. Evaluation: 7 parallel review batches reading tests plus exercised implementations.*
