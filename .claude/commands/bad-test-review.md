---
version: "v0.85.0"
description: Evaluate tests for charter alignment and functional authenticity (project)
argument-hint: "[--full] [--status]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /bad-test-review
Evaluate every unit and e2e test to determine whether the code causing each test to pass meets `/charter` expectations and project requirements, or whether it merely returns what is required to pass without genuine functional correctness.
## Prerequisites
- `CHARTER.md` exists (run `/charter` if missing)
- Test files exist
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| *(none)* | | Normal incremental — skip approved+unchanged tests |
| `--full` | No | Bypass manifest and review all tests |
| `--status` | No | Report manifest statistics without running review |
## Execution Instructions
**REQUIRED:**
1. **Generate Todo List:** Use `TodoWrite` from workflow steps so progress is visible and resumable after compaction
2. **Track Progress:** Mark `in_progress` → `completed`
3. **Post-Compaction:** Re-read this spec and regenerate todos
## Workflow
### Step 1: Parse Arguments
**If `--status`:** Jump to Step 2b, then **STOP**.
**If `--full`:** Set `fullMode = true` (skip manifest filter in Step 4).
**Otherwise:** Normal incremental mode.
### Step 2: Load Manifest
Read `.bad-test-manifest.json` if it exists.

**If manifest exists:**
1. Parse JSON
2. Extract `charter.contentHash`
3. Compute current `CHARTER.md` SHA-256
4. **If charter hash differs** — set `charterChanged = true`. Triggers full re-evaluation because alignment criteria shifted, even if test files didn't change.
5. Report: `Charter changed since last review — all tests will be re-evaluated.`

**If no manifest:**
1. Report: `No manifest found. First run — all tests will be reviewed.`
2. Create empty manifest in memory
### Step 2b: Manifest Statistics (`--status` only)
```
Bad Test Review Manifest:
  Last run: YYYY-MM-DD
  Tests tracked: N total
    Approved: A
    Flagged: F (with open issues)
  Charter hash: sha256:abc123...
  New tests (unreviewed): K
```
Count new tests by scanning files and comparing against manifest.
→ **STOP**.
### Step 3: Discover Test Files
```
Patterns: tests/, *.test.js, *.test.ts, *.spec.js, *.spec.ts, __tests__/
```
Use Glob. Exclude: `node_modules/`, build output, third-party tests, generated helpers.

Report: `Discovered N test files across M directories.`
### Step 4: Filter by Manifest
**If `--full` or `charterChanged`:** Skip filtering — evaluate all.

**Otherwise:** For each discovered file, compute SHA-256 hash and check manifest:

| Condition | Action |
|-----------|--------|
| Not in manifest | New — queue for evaluation |
| Hash matches, status `approved` | Skip |
| Hash matches, status `flagged` | Skip — has open bug issues |
| Hash **differs** | Re-examine — content changed |
| In manifest but file deleted | Remove from manifest |

Report filter results:
```
Manifest filter:
  New tests: N (queued)
  Changed tests: M (queued)
  Approved (skipped): A
  Flagged (skipped): F
```
### Step 5: Load Charter
Read `CHARTER.md` and extract:
- Project purpose and goals
- Technology stack and conventions
- Quality standards and non-functional requirements
- Testing expectations
### Step 6: Evaluate Each Test
#### 6a: Charter Alignment
Does the test validate behavior that maps to a charter goal, convention, or quality standard?
- Read the test file
- Identify what behavior each test case validates
- Cross-reference against charter goals/conventions
- **Aligned:** Validates documented requirement
- **Unaligned:** Doesn't map to charter (may still be valid — informational)
#### 6b: Functional Authenticity
Does the implementation genuinely implement the feature, or return hardcoded/minimal values to satisfy the assertion?

**Detection heuristics — flag suspicious patterns:**

| Heuristic | Description | Severity |
|-----------|-------------|----------|
| **Hardcoded return** | Return value exactly matches test assertion constant | High |
| **No branching** | Function has no branching — always returns same value | Medium |
| **Single-input coverage** | Implementation only handles exact test inputs | Medium |
| **Narrow assertions** | Overly narrow assertions not covering realistic scenarios | Low |
| **Mock-only validation** | Mock replaces all meaningful behavior — test validates the mock | High |
| **Same-commit pattern** | Implementation added in same commit as test with no other callers | Low |

For each suspicious pattern:
1. Read the implementation referenced by the test
2. Analyze whether implementation genuinely handles tested behavior
3. Record: test file, test name, concern type, severity, evidence
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
For each finding or group of related findings, create a bug to track the fix.

Each bug must reference test file, concern type, evidence:
```markdown
## Bug: Hollow Test — [test name]

**Test File:** `tests/path/to/file.test.js`
**Concern Type:** [Hardcoded return | No branching | etc.]
**Severity:** [High | Medium | Low]

**Evidence:**
[What was found]

**Test Code:**
```js
[relevant snippet]
```

**Implementation Code:**
```js
[relevant snippet]
```

**Recommendation:**
[What should change to make this test meaningful]
```
Group related findings (multiple hollow tests in same file) into a single bug when sharing the same root cause.

Report:
```
Created N bug issues:
  #NNN: Hollow test — [description]
  #MMM: Narrow assertions — [description]
```
### Step 8b: Save Report
Write **after** issue creation so issue numbers are available:
```
Construction/Code-Reviews/YYYY-MM-DD-bad-test-report.md
```
Create directory if missing.

**Report format:**
```markdown
# Bad Test Review Report

**Date:** YYYY-MM-DD
**Tests reviewed:** N (M new, K re-examined)
**Tests skipped:** A (approved, unchanged)
**Findings:** F total (H high, M medium, L low)

## High Severity

| Test File | Test Name | Concern | Evidence | Issue |
|-----------|-----------|---------|----------|-------|
| tests/foo.test.js | "returns correct value" | Hardcoded return | `getValue()` returns `42` | #1234 |

## Medium Severity
...

## Low Severity
...

## Clean Tests
C tests passed review with no concerns.

## Issues Created
- #1234 — Hollow test: [description]
- #1235 — Narrow assertions: [description]

## Charter Alignment Notes
- N tests aligned with charter goals
- M tests unaligned (informational — may still be valid)
```
**Issue column:** Show bug issue number (e.g., `#1234`) or `No issue` for informational findings.
### Step 9: Update Manifest
Write/update `.bad-test-manifest.json`:
1. **Charter hash:** Current `CHARTER.md` SHA-256
2. **Reviewed tests:** For each evaluated:
   - `contentHash`: Current file SHA-256
   - `status`: `approved` (no findings) or `flagged` (has findings)
   - `reviewedAt`: Today's date
   - `findingCount`: Number of findings
   - `issueRefs`: Bug issue numbers (if flagged)
3. **Deleted tests:** Remove entries for files that no longer exist
4. **Unevaluated tests:** Preserve existing entries for skipped (approved+unchanged)
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
## Error Handling
| Situation | Response |
|-----------|----------|
| CHARTER.md not found | "No charter found. Run `/charter` first." → STOP |
| No test files found | "No test files found matching project conventions." → STOP |
| Manifest malformed | "Manifest corrupted. Running full review." → continue with --full |
| Test file unreadable | Warn and skip that file, continue |
| Bug issue creation fails | Warn, include in report, continue |

**End of /bad-test-review Command**
