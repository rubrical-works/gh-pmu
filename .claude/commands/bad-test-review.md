---
version: "v0.58.0"
description: Evaluate tests for charter alignment and functional authenticity (project)
argument-hint: "[--full] [--status]"
---
<!-- MANAGED -->
# /bad-test-review
Evaluate every unit and e2e test to determine whether code causing each test to pass meets charter expectations, or merely returns what is required to pass without genuine functional correctness.
---
## Prerequisites
- `CHARTER.md` exists in project root (run `/charter` first if missing)
- Test files exist in the codebase
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| *(none)* | | Normal incremental run — skip approved+unchanged tests |
| `--full` | No | Bypass manifest and review all tests |
| `--status` | No | Report manifest statistics without running review |
---
## Execution Instructions
**REQUIRED:** Before executing this command:
1. **Generate Todo List:** Parse the workflow steps in this spec, then use `TodoWrite` to create todos from the workflow steps so progress is visible and resumable after compaction
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
3. **Post-Compaction:** If resuming after context compaction, re-read this spec and regenerate todos
---
## Workflow
### Step 1: Parse Arguments
Check for `--full` or `--status` flags.
- **If `--status`:** Jump to Step 2b then **STOP**.
- **If `--full`:** Set `fullMode = true` (skip manifest filtering in Step 4).
- **Otherwise:** Normal incremental mode.
### Step 2: Load Manifest
Read `.bad-test-manifest.json` if it exists.
**If manifest exists:**
1. Parse the JSON manifest
2. Extract `charter.contentHash` field
3. Compute current charter hash: read `CHARTER.md` and compute SHA-256
4. **If charter hash differs** — set `charterChanged = true` (alignment criteria shifted, full re-evaluation needed)
5. Report: `Charter changed since last review — all tests will be re-evaluated.`
**If manifest does not exist:**
1. Report: `No manifest found. First run — all tests will be reviewed.`
2. Create empty manifest structure in memory
### Step 2b: Manifest Statistics (`--status` only)
Read `.bad-test-manifest.json` and report:
```
Bad Test Review Manifest:
  Last run: YYYY-MM-DD
  Tests tracked: N total
    Approved: A
    Flagged: F (with open issues)
  Charter hash: sha256:abc123...
  New tests (unreviewed): K
```
Count new tests by scanning test files and comparing against manifest entries.
→ **STOP** after reporting.
### Step 3: Discover Test Files
Scan codebase for all test files:
```
Patterns: tests/, *.test.js, *.test.ts, *.spec.js, *.spec.ts, __tests__/
```
Use Glob to discover all test files. Exclude: `node_modules/`, build output directories, third-party library tests, generated test helpers.
Report: `Discovered N test files across M directories.`
### Step 4: Filter by Manifest
**If `--full` mode or `charterChanged`:** Skip filtering — evaluate all discovered tests.
**Otherwise:** For each discovered test file, compute SHA-256 content hash and check manifest:
| Condition | Action |
|-----------|--------|
| Test NOT in manifest | New test — queue for evaluation |
| Hash matches, status `approved` | Skip — previously approved, unchanged |
| Hash matches, status `flagged` | Skip — already has open bug issues |
| Hash **differs** from manifest | Re-examine — content changed since last review |
| In manifest but file deleted | Remove from manifest (cleanup) |
Report:
```
Manifest filter:
  New tests: N (queued for review)
  Changed tests: M (queued for re-review)
  Approved (skipped): A
  Flagged (skipped): F
```
### Step 5: Load Charter
Read `CHARTER.md` to extract project goals, conventions, and quality standards.
Key sections: project purpose/goals, technology stack/conventions, quality standards/NFRs, testing expectations.
### Step 6: Evaluate Each Test
For each queued test file, perform two analyses:
#### 6a: Charter Alignment
Does the test validate behavior mapping to a charter goal, convention, or quality standard?
- Read the test file and identify what behavior each test case validates
- Cross-reference against charter goals and conventions
- **Aligned:** Validates a documented requirement or convention
- **Unaligned:** Doesn't map to any charter goal (may still be valid — informational only)
#### 6b: Functional Authenticity
Does the implementation genuinely implement the feature, or just return hardcoded/minimal values to satisfy assertions?
**Detection heuristics — flag suspicious patterns:**
| Heuristic | Description | Severity |
|-----------|-------------|----------|
| **Hardcoded return** | Return value in implementation exactly matches test assertion constant | High |
| **No branching** | Function has no branching logic — always returns same value regardless of input | Medium |
| **Single-input coverage** | Implementation only handles the exact inputs used in tests | Medium |
| **Narrow assertions** | Test uses overly narrow assertions that don't cover realistic scenarios | Low |
| **Mock-only validation** | Mock replaces all meaningful behavior — test validates the mock, not the code | High |
| **Same-commit pattern** | Implementation added in the same commit as the test with no other callers | Low |
**For each suspicious pattern found:**
1. Read the implementation file referenced by the test
2. Analyze whether the implementation genuinely handles the tested behavior
3. Record test file, test name, concern type, severity, and evidence
### Step 7: Generate Structured Report
```
## Bad Test Review Report

**Date:** YYYY-MM-DD
**Tests reviewed:** N
**Findings:** M

### High Severity
| Test File | Test Name | Concern | Evidence |
|-----------|-----------|---------|----------|
| tests/foo.test.js | "returns correct value" | Hardcoded return | `getValue()` returns `42`, test asserts `42` |

### Medium Severity
...

### Low Severity
...

### Summary
- High: N findings
- Medium: M findings
- Low: K findings
- Clean: C tests (no concerns)
```
### Step 8: Create Bug Issues
For each finding or logical group of related findings, create a bug issue. Each must reference the parent test file, concern type, and evidence:
```markdown
## Bug: Hollow Test — [test name]

**Test File:** `tests/path/to/file.test.js`
**Concern Type:** [Hardcoded return | No branching | etc.]
**Severity:** [High | Medium | Low]

**Evidence:**
[Description of what was found]

**Test Code:**
```js
[relevant test snippet]
```

**Implementation Code:**
```js
[relevant implementation snippet]
```

**Recommendation:**
[What should change to make this test meaningful]
```
Group related findings (same file, same root cause) into a single bug issue.
Report: `Created N bug issues: #NNN: Hollow test — [description]`
### Step 9: Update Manifest
Write or update `.bad-test-manifest.json`:
1. **Charter hash:** Update to current `CHARTER.md` SHA-256
2. **Reviewed tests:** For each evaluated test: `contentHash`, `status` (`approved`|`flagged`), `reviewedAt`, `findingCount`, `issueRefs`
3. **Deleted tests:** Remove entries for files that no longer exist
4. **Unevaluated tests:** Preserve existing manifest entries for skipped tests
### Step 10: Final Summary
```
Bad Test Review Complete.

Tests reviewed: N (M new, K re-examined)
Tests skipped: A (approved, unchanged)
Findings: F total (H high, M medium, L low)
Bug issues created: B
Manifest updated: .bad-test-manifest.json

Next run will skip N approved+unchanged tests.
```
→ **STOP.**
---
## Error Handling
| Situation | Response |
|-----------|----------|
| CHARTER.md not found | "No charter found. Run `/charter` first." → STOP |
| No test files found | "No test files found matching project conventions." → STOP |
| Manifest malformed | "Manifest corrupted. Running full review." → continue with --full behavior |
| Test file unreadable | Warn and skip that file, continue with remaining |
| Bug issue creation fails | Warn, include finding in report, continue |
---
**End of /bad-test-review Command**
