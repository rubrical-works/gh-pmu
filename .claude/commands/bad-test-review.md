---
version: "v0.61.0"
description: Evaluate tests for charter alignment and functional authenticity (project)
argument-hint: "[--full] [--status]"
---
<!-- MANAGED -->

# /bad-test-review
Evaluate every unit and e2e test in the codebase to determine whether the code that causes each test to pass meets the expectations from the `/charter` and project requirements, or whether it merely returns what is required to have that test pass without genuine functional correctness.

## Prerequisites
- `CHARTER.md` exists in project root (run `/charter` first if missing)
- Test files exist in the codebase

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| *(none)* | | Normal incremental run — skip approved+unchanged tests |
| `--full` | No | Ignore manifest, bypass manifest and review all tests |
| `--status` | No | Report manifest statistics without running review |

## Execution Instructions
**REQUIRED:** Before executing this command:
1. **Generate Todo List:** Parse the workflow steps in this spec, then use `TodoWrite` to create todos from the workflow steps so progress is visible and resumable after compaction
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
3. **Post-Compaction:** If resuming after context compaction, re-read this spec and regenerate todos

## Workflow

### Step 1: Parse Arguments
Check for `--full` or `--status` flags.
**If `--status`:** Jump to Step 2b (manifest statistics) then **STOP**.
**If `--full`:** Set `fullMode = true` (skip manifest filtering in Step 4).
**Otherwise:** Normal incremental mode.

### Step 2: Load Manifest
Read `.bad-test-manifest.json` if it exists.
```bash
# Check if manifest exists
ls .bad-test-manifest.json 2>/dev/null
```
**If manifest exists:**
1. Parse the JSON manifest
2. Extract the `charter.contentHash` field
3. Compute current charter hash: read `CHARTER.md` and compute SHA-256 content hash
4. **If charter hash differs** from manifest — set `charterChanged = true`. This triggers full re-evaluation of all tests because the alignment criteria have shifted, even if test files haven't changed.
5. Report: `Charter changed since last review — all tests will be re-evaluated.`
**If manifest does not exist:**
1. Report: `No manifest found. First run — all tests will be reviewed.`
2. Create empty manifest structure in memory

### Step 2b: Manifest Statistics (`--status` only)
**If `--status` flag specified:**
Read `.bad-test-manifest.json` and report manifest statistics:
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
Scan the codebase for all test files matching project conventions:
```
Patterns: tests/, *.test.js, *.test.ts, *.spec.js, *.spec.ts, __tests__/
```
Use Glob to discover all test files. Exclude:
- `node_modules/`
- Build output directories
- Third-party library tests
- Generated test helpers
Report discovery count:
```
Discovered N test files across M directories.
```

### Step 4: Filter by Manifest
**If `--full` mode or `charterChanged`:** Skip filtering — evaluate all discovered tests.
**Otherwise:** For each discovered test file, compute its SHA-256 content hash and check the manifest:
| Condition | Action |
|-----------|--------|
| Test NOT in manifest | New test — queue for evaluation |
| Hash matches, status `approved` | Skip — previously approved, unchanged |
| Hash matches, status `flagged` | Skip — already has open bug issues |
| Hash **differs** from manifest | Re-examine — content changed since last review |
| In manifest but file deleted | Remove from manifest (cleanup) |
Report filter results:
```
Manifest filter:
  New tests: N (queued for review)
  Changed tests: M (queued for re-review)
  Approved (skipped): A
  Flagged (skipped): F
```

### Step 5: Load Charter
Read `CHARTER.md` to extract project goals, conventions, and quality standards for alignment checking.
Key sections to extract:
- Project purpose and goals
- Technology stack and conventions
- Quality standards and non-functional requirements
- Testing expectations

### Step 6: Evaluate Each Test
For each test file queued for evaluation, perform two analyses:

#### 6a: Charter Alignment
Does the test validate behavior that maps to a charter goal, convention, or quality standard?
- Read the test file
- Identify what behavior each test case validates
- Cross-reference against charter goals and conventions
- **Aligned:** Test validates a documented requirement or convention
- **Unaligned:** Test exists but doesn't map to any charter goal (may still be valid — informational only)

#### 6b: Functional Authenticity
Does the implementation behind the passing test genuinely implement the feature, or does it just return hardcoded/minimal values to satisfy the assertion?
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
3. Record the test file, test name, concern type, severity, and evidence

### Step 7: Generate Structured Report
Present findings organized by severity:
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
For each finding or logical group of related findings, create a bug issue to track the fix:
```bash
# Write bug body to temp file, then create via /bug workflow
```
Each bug issue must reference the parent test file, concern type, and evidence:
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
Group related findings (e.g., multiple hollow tests in the same file) into a single bug issue when they share the same root cause.
Report created issues:
```
Created N bug issues:
  #NNN: Hollow test — [description]
  #MMM: Narrow assertions — [description]
```

### Step 9: Update Manifest
Write or update `.bad-test-manifest.json` with results:
1. **Charter hash:** Update to current `CHARTER.md` SHA-256 content hash
2. **Reviewed tests:** For each evaluated test:
   - `contentHash`: Current SHA-256 of test file
   - `status`: `approved` (no findings) or `flagged` (has findings)
   - `reviewedAt`: Today's date
   - `findingCount`: Number of findings
   - `issueRefs`: Bug issue numbers (if flagged)
3. **Deleted tests:** Remove entries for files that no longer exist
4. **Unevaluated tests:** Preserve existing manifest entries for tests that were skipped (approved+unchanged)
```bash
# Write updated manifest
```

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
| Manifest malformed | "Manifest corrupted. Running full review." → continue with --full behavior |
| Test file unreadable | Warn and skip that file, continue with remaining |
| Bug issue creation fails | Warn, include finding in report, continue |
**End of /bad-test-review Command**
