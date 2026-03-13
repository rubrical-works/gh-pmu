---
version: "v0.62.1"
description: Start working on issues with validation and auto-TODO (project)
argument-hint: "#issue [#issue...] [--assign] [--nonstop] | all in <status>"
---
<!-- EXTENSIBLE -->

# /work
Start working on one or more issues. Validates issue existence, branch assignment, and issue type, then moves to `in_progress`, extracts auto-TODO, and dispatches to framework methodology.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command work`

## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue assigned to a branch (use `/assign-branch` first, or pass `--assign` to auto-assign)

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes (one of) | Single issue number (e.g., `#42` or `42`) |
| `#issue #issue...` | | Multiple issue numbers (e.g., `#42 #43 #44`) |
| `all in <status>` | | All issues in given status (e.g., `all in backlog`) |
| `--assign` | No | Assign issue(s) to current branch before starting work |
| `--nonstop` | No | Epic/branch tracker: skip per-sub-issue STOP, process all to `in_review` continuously |

## Execution Instructions
Generate todo list from workflow steps (one per step + active extensions). Track `in_progress` → `completed`. Post-compaction: re-read spec and regenerate todos.

## Workflow

### Step 0: Conditional - Clear Todo List
If not working on an epic or branch tracker, clear todo list.
<!-- USER-EXTENSION-START: pre-work -->
<!-- USER-EXTENSION-END: pre-work -->

### Step 1: Context Gathering (Preamble Script)
Run `node .claude/scripts/shared/work-preamble.js` with `--issue N`, `--issues "N,N,N"`, or `--status <status>`. Append `--assign` to auto-assign to current branch.
Parse JSON output: if `ok: false`, report `errors[]` and STOP. If `ok: true`, extract `context`, `gates`, `autoTodo`, `warnings`. Report gate results and warnings.
**--assign errors:** `ALREADY_ASSIGNED` (different branch), `WORKSTREAM_CONFLICT` (use `/assign-branch` to override).
Run `--schema` for envelope field reference.
<!-- USER-EXTENSION-START: post-work-start -->
<!-- USER-EXTENSION-END: post-work-start -->

### Step 2: Framework Methodology Dispatch
Load `{frameworkPath}/{framework}/` core file from `framework-config.json`. If missing, warn and continue.

### Step 3: Work the Issue
For each AC (or batch of closely related ACs):
1. Mark TODO `in_progress`
2. Execute TDD cycle (RED → GREEN → REFACTOR per framework)
3. Run full test suite — all tests must pass
4. Mark TODO `completed`
5. **COMMIT** — `Refs #$ISSUE — <description>`. One commit per AC; closely related ACs may share a commit, but the gate still applies after the batch.
   **GATE: Do NOT start the next AC until this commit is made.**
**RED:** Write failing test. **GREEN:** Minimal implementation to pass. **REFACTOR:** Analyze for duplication, naming, complexity. Report decision (refactor or skip with reason). Keep tests passing.
If no auto-TODO, work as single unit. Post-compaction: re-read spec and resume from first incomplete AC.

### Step 3b: Documentation Judgment
Evaluate whether documentation (design decision or tech debt) is warranted. Re-read `.claude/scripts/shared/lib/doc-templates.json` from disk (not memory) for category criteria, target paths, and naming rules. If warranted, create the document and commit with `Refs #$ISSUE`.
<!-- USER-EXTENSION-START: post-implementation -->
<!-- USER-EXTENSION-END: post-implementation -->

### Step 4: Verify Acceptance Criteria (with QA Extraction)
**IMPORTANT — Ground in file state:** Before evaluating each AC, re-read the actual file content using the Read tool. Do NOT evaluate from memory — re-read to confirm the criterion is met in current code. This prevents batch fatigue hallucination.
For each AC checkbox in the issue body:
- **Can verify** → Mark `[x]`, continue
- **Cannot verify** (manual, external) → Check for QA extraction (Step 4a), then **STOP** and present options, wait for user disposition
After all ACs resolved, export and update issue body:
```bash
gh pmu view $ISSUE --body-stdout > .tmp-$ISSUE.md
# Update checkboxes to [x]
gh pmu edit $ISSUE -F .tmp-$ISSUE.md && rm .tmp-$ISSUE.md
```

#### Step 4a: QA Extraction — Manual Test AC Detection
Re-read `.claude/scripts/shared/lib/qa-config.json` from disk (not memory) for detection keywords and body template. Match unverifiable ACs against keywords (case-insensitive). If matched, present candidates via `AskUserQuestion` (multiSelect, include "Skip all"). For confirmed ACs, create sub-issues with `gh pmu sub create --parent $ISSUE --title "QA: [AC description]" --label qa-required -F .tmp-qa-body.md`. Annotate parent AC as unchecked with `→ QA: #NNN`. Parent stays `in_review` until all QA sub-issues closed.
<!-- USER-EXTENSION-START: post-ac-verification -->
<!-- USER-EXTENSION-END: post-ac-verification -->

### Step 5: Move to in_review
```bash
gh pmu move $ISSUE --status in_review
```

### Step 6: STOP Boundary — Report and Wait
```
Issue #$ISSUE: $TITLE — In Review
Say "done" or run /done #$ISSUE to close this issue.
```
**STOP.** Wait for user to say "done". Do NOT close the issue.
**CRITICAL — Autonomous Epic/Branch Sub-Issue Processing:**
When working an epic or branch tracker (`context.type` is `"epic"` or `"branch"`), process sub-issues autonomously in ascending numeric order (default), or custom **Processing Order:** from epic body. Sub-issues already in `in_review` or `done` are skipped (`context.skipped`).
**Default mode:** Per `02-github-workflow.md` Section 4 — each sub-issue: `in_progress` → Steps 3–4 → `in_review` → **STOP** per sub-issue → user says "done" → next.
**`--nonstop` mode:** Same cycle but **no STOP** between sub-issues. Report `"Sub-issue #N: $TITLE → In Review (M/T processed)"` and continue immediately. Ignored for standard issues.
**`--nonstop` rules:** One commit per AC using `Refs #N`. Commits remain local (no push — deferred to `/done`). Test failure, AC failure, QA extraction, or `gh pmu` error halt execution immediately — report which sub-issue failed, how many completed, and resume instructions.
**`--nonstop` post-compaction recovery:** Re-read this spec, check `gh pmu sub list $ISSUE`, resume from first sub-issue not in `in_review`/`done`.
**`--nonstop` summary report:** After all processed, report `Nonstop Processing Complete` with sub-issues processed/skipped/failed counts.
**After all sub-issues reach `in_review` or `done`:**
- **Epic:** Evaluate epic's own acceptance criteria (Step 4), move epic to `in_review`, **STOP** — wait for "done"
- **Branch tracker:** Report completion, suggest `/merge-branch` or `/prepare-release`. Do not close tracker.
**Default mode:** Never skip per-sub-issue STOP boundary. **Continuous mode:** Sub-issues only moved to `in_review` (not `done`) — user runs `/done` after review.

## Error Handling
**STOP errors:** Issue not found, no branch assignment, `gh pmu` failure, `ALREADY_ASSIGNED` (different branch), `WORKSTREAM_CONFLICT` (use `/assign-branch`).
**Non-blocking:** PRD tracker not found, framework file missing, no acceptance criteria, issue already in_progress.
**End of /work Command**
