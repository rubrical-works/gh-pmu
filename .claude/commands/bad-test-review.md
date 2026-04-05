---
version: "v0.81.1"
description: Evaluate tests for charter alignment and functional authenticity (project)
argument-hint: "[--full] [--status]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /bad-test-review
Evaluate every test to determine whether passing code meets charter expectations or merely returns values to satisfy assertions.
## Prerequisites
- `CHARTER.md` exists (run `/charter` first if missing)
- Test files exist in the codebase
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| *(none)* | | Normal incremental run -- skip approved+unchanged tests |
| `--full` | No | Bypass manifest, review all tests |
| `--status` | No | Report manifest statistics only |
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Track Progress:** Mark todos `in_progress` -> `completed`
3. **Post-Compaction:** Re-read this spec and regenerate todos
### Step 1: Parse Arguments
Check for `--full` or `--status` flags.
- `--status`: Jump to Step 2b then **STOP**
- `--full`: Set `fullMode = true` (skip manifest filtering)
- Otherwise: Normal incremental mode
### Step 2: Load Manifest
Read `.bad-test-manifest.json` if it exists.
**If exists:**
1. Parse JSON, extract `charter.contentHash`
2. Compute current charter SHA-256 hash
3. If charter hash differs: set `charterChanged = true`, report re-evaluation trigger
**If not exists:** Report first run, create empty manifest in memory.
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
-> **STOP** after reporting.
### Step 3: Discover Test Files
Scan for test files: `tests/, *.test.js, *.test.ts, *.spec.js, *.spec.ts, __tests__/`
Exclude: `node_modules/`, build output, third-party tests, generated helpers.
Report discovery count.
### Step 4: Filter by Manifest
**If `--full` or `charterChanged`:** Evaluate all tests.
**Otherwise:**
| Condition | Action |
|-----------|--------|
| Test NOT in manifest | Queue for evaluation |
| Hash matches, `approved` | Skip |
| Hash matches, `flagged` | Skip |
| Hash **differs** | Re-examine |
| In manifest but deleted | Remove from manifest |
Report filter results (new, changed, approved skipped, flagged skipped).
### Step 5: Load Charter
Read `CHARTER.md` — extract goals, conventions, quality standards, testing expectations.
### Step 6: Evaluate Each Test
For each queued test file:
**6a: Charter Alignment** — does the test validate behavior mapping to charter goals/conventions?
- **Aligned:** Validates documented requirement
- **Unaligned:** Doesn't map to charter goal (informational)
**6b: Functional Authenticity** — does the implementation genuinely implement the feature?
**Detection heuristics:**
| Heuristic | Description | Severity |
|-----------|-------------|----------|
| **Hardcoded return** | Return value matches test assertion constant | High |
| **No branching** | Always returns same value regardless of input | Medium |
| **Single-input coverage** | Only handles exact test inputs | Medium |
| **Narrow assertions** | Overly narrow, doesn't cover realistic scenarios | Low |
| **Mock-only validation** | Mock replaces all meaningful behavior | High |
| **Same-commit pattern** | Implementation added with test, no other callers | Low |
For each suspicious pattern: read implementation, analyze, record file/name/concern/severity/evidence.
### Step 7: Generate Structured Report
Present findings by severity with table format: Test File, Test Name, Concern, Evidence.
Include summary counts (High/Medium/Low/Clean).
### Step 8: Create Bug Issues
For each finding or logical group, create a bug issue with:
- Test file, concern type, severity, evidence
- Relevant test and implementation code snippets
- Recommendation for fix
Group related findings sharing the same root cause into single issues.
### Step 8b: Save Report
Write to `Construction/Code-Reviews/YYYY-MM-DD-bad-test-report.md` (create directory if needed).
Include: date, counts, severity tables with issue numbers, clean tests count, issues created list, charter alignment notes.
### Step 9: Update Manifest
Write/update `.bad-test-manifest.json`:
1. **Charter hash:** Current SHA-256
2. **Reviewed tests:** `contentHash`, `status` (approved/flagged), `reviewedAt`, `findingCount`, `issueRefs`
3. **Deleted tests:** Remove entries
4. **Unevaluated tests:** Preserve existing entries
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
-> **STOP.**
## Error Handling
| Situation | Response |
|-----------|----------|
| CHARTER.md not found | "No charter found. Run `/charter` first." -> STOP |
| No test files found | "No test files found matching project conventions." -> STOP |
| Manifest malformed | "Manifest corrupted. Running full review." -> continue as --full |
| Test file unreadable | Warn and skip, continue with remaining |
| Bug issue creation fails | Warn, include finding in report, continue |
**End of /bad-test-review Command**
