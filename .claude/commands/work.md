---
version: "v0.80.0"
description: Start working on issues with validation and auto-TODO (project)
argument-hint: "#issue [#issue...] [--assign] [--nonstop] [--wait] | all in <status>"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /work
Start working on one or more issues. Validates existence, branch assignment, and type, then moves to `in_progress`, extracts auto-TODO, and dispatches to framework methodology.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command work`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
- Issue assigned to a branch (use `/assign-branch` or `--assign`)
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes (one of) | Single issue number |
| `#issue #issue...` | | Multiple numbers |
| `all in <status>` | | All issues in given status |
| `--assign` | No | Auto-assign to current branch |
| `--nonstop` | No | Epic/branch: skip per-sub-issue STOP |
| `--wait` | No | Wait for pending CI to pass before starting work |
---
## Execution Instructions
**REQUIRED:** This is a routed command -- use two-phase task creation:
1. **Phase 1 -- Preamble task only:** Create a single task for the preamble/setup step using `TaskCreate`. Do NOT create tasks for subsequent workflow steps yet.
2. **Phase 2 -- Bulk create after routing:** After the preamble confirms the workflow path (no redirect, no early exit), bulk-create tasks for all remaining workflow steps using `TaskCreate`.
3. **On redirect or early exit:** Mark the preamble task as completed and stop. Do NOT create tasks for the original command's remaining steps.
4. **Include Extensions:** For each non-empty `USER-EXTENSION` block, add a task in Phase 2.
5. **Track Progress:** Mark tasks `in_progress` -> `completed` as you work.
6. **Post-Compaction:** Re-read this spec. Resume from the first incomplete task -- no re-routing needed.
---
## Workflow
### Step 0: Clear Todo List
If not epic or branch tracker, clear todos.

<!-- USER-EXTENSION-START: pre-work -->
<!-- USER-EXTENSION-END: pre-work -->

### Step 1: Context Gathering (Preamble Script)
Run `node .claude/scripts/shared/work-preamble.js` with `--issue N`, `--issues "N,N,N"`, or `--status <status>`. Append `--assign` for auto-assign.
Parse JSON: `ok: false` -> report errors, STOP. Extract `context`, `gates`, `autoTodo`, `warnings`.
**--assign errors:** `ALREADY_ASSIGNED`, `WORKSTREAM_CONFLICT`.

<!-- USER-EXTENSION-START: post-work-start -->
<!-- USER-EXTENSION-END: post-work-start -->

### Step 1a: CI Wait (--wait flag)
**Trigger:** `context.wait` is `true`.
```bash
node .claude/scripts/shared/wait-for-ci.js --branch $(git branch --show-current) --timeout 300
```
- Exit 0 (pass): continue
- Exit 1 (fail): STOP
- Exit 2 (timeout): STOP
- Exit 3 (no runs): continue silently
If `context.wait` not set, skip.
### Step 1b: Epic Complexity Assessment
**Trigger:** `context.type` is `"epic"` and `--nonstop` set.
Run `node .claude/scripts/shared/epic-complexity.js $ISSUE`. `"functional"` -> `strictTDD = true`. Signals: `.claude/metadata/epic-complexity-signals.json`.
### Step 2: Framework Methodology Dispatch
Load core file from `framework-config.json`. Missing: warn, continue.
### Step 2a: Load TDD Checklist
Read `.claude/skills/tdd-process/tdd-checklist.json`. If valid JSON with `red`, `green`, `refactor` objects (each with `required[]` and `gate` string) and `failure-recovery` (with `triggers[]` and `steps[]`):
- Set `tddChecklist` = loaded JSON. Each phase may have `deepReference: { skill, when }` for gate-failure skill loading.
If load fails: warn "TDD checklist not found -- using inline TDD behavior", set `tddChecklist = null`.
### Step 3: Work the Issue
Per AC: mark in_progress, TDD cycle (RED->GREEN->REFACTOR), run tests, mark completed, commit (`Refs #$ISSUE`).
**GATE: Do NOT start next AC until commit made.**
**Sub-Agent Review Gate:** After Agent tool, `git diff --name-only`. Read changed files, verify match. Mandatory when `strictTDD`. Not satisfied by summaries/tests alone.
**TDD Execution (checklist-driven):** When `tddChecklist` loaded: execute `required[]` items per phase, enforce `gate` before proceeding. On failure-recovery trigger, execute recovery steps. On gate failure, check `deepReference` -- load `.claude/skills/{skill-name}/SKILL.md` for guidance, retry. Missing skill: warn, continue.
**TDD Fallback (inline):** When `tddChecklist` null: RED (failing test), GREEN (minimal pass), REFACTOR (analyze, report decision).
If no auto-TODO: single unit. Post-compaction: resume from first incomplete AC.
### Step 3b: Documentation Judgment
Re-read `.claude/scripts/shared/lib/doc-templates.json` from disk. Create if warranted.

<!-- USER-EXTENSION-START: post-implementation -->
<!-- USER-EXTENSION-END: post-implementation -->

### Step 4: Verify Acceptance Criteria (with QA Extraction)
**Re-read files before evaluating each AC.** Do NOT evaluate from memory.
Can verify -> `[x]`. Cannot verify -> QA extraction (4a).
Update issue body via `gh pmu view/edit` with temp file.
#### Step 4a: QA Extraction -- Automatic Sub-Issue Creation
Re-read `.claude/scripts/shared/lib/qa-config.json`. Match unverifiable ACs against keywords. For each match, **automatically** (no user confirmation):
1. `gh pmu sub create --parent $ISSUE --title "QA: [AC description]" --label qa-required -F .tmp-qa-body.md`
2. Sub-issue body: AC text, parent reference, QA context
3. Annotate parent AC as `[x] AC text -> QA: #NNN`
Silent, automatic flow. Works in standard and `--nonstop` mode.
#### Step 4b: Force-Move Prohibition
**NEVER** use `--force` to bypass unchecked ACs on issues you implemented. Legitimate: epic parents, external, branch trackers, test-plan approvals.

<!-- USER-EXTENSION-START: post-ac-verification -->
<!-- USER-EXTENSION-END: post-ac-verification -->

#### Step 4c: Log Changed Files to Issue Body
Compute files changed during this issue's work:
```bash
git log --name-status --grep="Refs #$ISSUE" --pretty=format:"" | sort -u | grep -v "^$"
```
Categorize: `A` = Added, `M` = Modified, `D` = Deleted. Separate test files from source files using `.claude/scripts/shared/lib/classify-changed-files.js`. Append "Files Changed" section to issue body (non-destructively). Omit empty categories/sub-categories. Skip if no commits found.
### Step 5: Move to in_review
```bash
gh pmu move $ISSUE --status in_review
```
### Step 6: STOP Boundary
```
Issue #$ISSUE: $TITLE -- In Review
Say "done" or run /done #$ISSUE to close.
```
**STOP.** Do NOT close.
**CRITICAL -- Autonomous Epic/Branch processing:** Ascending numeric order (or custom Processing Order). Skip done/in_review.
**Default:** Per-sub-issue STOP.
**`--nonstop`:** No STOP between sub-issues. One commit per AC. Commits local. Failure halts with resume instructions.
**Post-compaction:** Check `gh pmu sub list`, resume from first incomplete.
#### Step 6a: Post-Nonstop Audit
(1) Commit density warning. (2) AC checkbox audit. (3) Test coverage audit.
**After all sub-issues done:**
- **Epic:** Evaluate ACs, move to in_review, STOP
- **Branch tracker:** Report, STOP. Do NOT suggest "done":
```
All sub-issues on branch {branch-name} are in review or done.
Next: /merge-branch or /prepare-release
```
---
## Error Handling
**STOP:** Not found, no branch, pmu failure, ALREADY_ASSIGNED, WORKSTREAM_CONFLICT.
**Non-blocking:** PRD tracker missing, framework missing, no ACs, already in_progress.
---
**End of /work Command**
