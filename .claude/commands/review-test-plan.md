---
version: "v0.62.1"
description: Review a test plan against its PRD (project)
argument-hint: "#issue [--mode ...] [--force]"
---

<!-- MANAGED -->
# /review-test-plan
Reviews a TDD test plan document linked from a GitHub issue, cross-referencing it against the source PRD for coverage completeness. Delegates setup to `review-preamble.js`, keeping this spec focused on evaluation and model judgment. Document file updates (Reviews metadata, Review Log) are handled inline; issue body updates, comment posting, and label assignment are handled by the calling orchestrator.
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue body must contain `**Test Plan:**` and `**PRD:**` linking to both files
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to the test plan (e.g., `#42` or `42`) |
| `--mode` | No | Transient review mode override: `solo`, `team`, or `enterprise` |
| `--force` | No | Force re-review even if issue has `reviewed` label |
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Create Todo List:** Use `TodoWrite` to create todos from the steps below
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
3. **Post-Compaction:** If resuming after context compaction, re-read this spec and regenerate todos
---
## Workflow
### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/review-preamble.js $ISSUE [--mode mode] [--force]
```
Parse JSON output. If `ok: false`: report `errors[0].message` → **STOP**.
If `earlyExit: true`: report review count and **STOP**.
Extract: `context` (issue data, reviewNumber, `**Test Plan:**` and `**PRD:**` file paths), `criteria`, `warnings`.
Read both the test plan file and PRD file at extracted paths. If either not found → **STOP**.

<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 2: Evaluate Criteria

**Step 2a: Auto-Evaluate Objective Criteria**
Re-read `.claude/metadata/test-plan-review-criteria.json` from disk (not memory). For each criterion, use `autoCheckMethod` to evaluate the test plan and PRD. Emit ✅/⚠️/❌ with evidence. Use `shouldEvaluate(criterionId, ...)` from `review-mode.js` to filter by reviewMode.
**Coverage Analysis (P0):** Execute `coverageAnalysis.procedure` from the criteria file. Map acceptance criteria from PRD to test cases in test plan. Report coverage as structured findings.
**Graceful degradation:** If `test-plan-review-criteria.json` not found or malformed, warn and use inline defaults: AC coverage, Test framework specified, Test levels defined, Story-to-test mapping, Error scenarios present, Boundary conditions tested, Failure modes covered, Integration points mapped, Component interactions verified, Data flow boundaries tested, E2E scenarios cover critical journeys, E2E happy paths and error paths, E2E scenarios map to PRD requirements, Framework consistent with test strategy, Coverage targets realistic, Test coverage proportionate. If criteria array is empty or no criteria found, warn and fall back to inline defaults. Per-criterion validation: skip criteria missing `autoCheckMethod`. All failures non-blocking.

**Step 2b: Ask Subjective Criteria**
Load subjective criteria from `test-plan-review-criteria.json`. Use `AskUserQuestion` with each criterion's `question`, `header`, and `options`. Partial reviews valid. **Solo mode:** skip entirely.
**Coverage gaps are reported as bullet-point concerns** (not tables) for `/resolve-review` parser compatibility.

**Step 2c: Determine Recommendation**
- **Ready for approval** — All ACs have test cases, no blocking concerns
- **Ready with minor gaps** — Small coverage gaps
- **Needs revision** — Significant coverage gaps
- **Needs major rework** — Fundamental coverage issues

### Step 3: Update Test Plan File
**Update `**Reviews:** N` field:** Increment if exists, add `**Reviews:** 1` after metadata fields if not.
**Update Review Log:** Append row to existing `## Review Log` table. If section missing, append at end.
```markdown
| # | Date | Reviewer | Findings Summary |
|---|------|----------|------------------|
| N | YYYY-MM-DD | Claude | [Brief one-line summary] |
```
Each review appends a new row. **Never edit or delete existing rows.**

### Step 4: Write Findings
Write structured findings to `.tmp-$ISSUE-findings.json` for the calling orchestrator. Include recommendation, coverage summary, and all evaluated criteria.

<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

---
## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | Report `errors[0].message` → STOP |
| Test plan file not found | Report path error → STOP |
| PRD file not found | Report path error → STOP |
| Issue closed | Ask user (from preamble context) |
| File write fails | Report error → STOP |
---
**End of /review-test-plan Command**
