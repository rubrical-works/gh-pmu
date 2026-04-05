---
version: "v0.82.0"
description: Start working on issues with validation and auto-task extraction (project)
argument-hint: "#issue [#issue...] [--assign] [--nonstop] [--wait] | all in <status>"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /work
Start working on one or more issues. Validates existence, branch assignment, and type, then moves to `in_progress`, extracts auto-tasks, and dispatches to framework methodology.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command work`
---
## Prerequisites
- `gh pmu` installed and `.gh-pmu.json` configured
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
| `--wait` | No | Wait for pending CI before starting |
---
## Execution Instructions
**REQUIRED:** Routed command -- two-phase task creation:
1. **Phase 1:** Create preamble task only via `TaskCreate`.
2. **Phase 2:** After preamble confirms path, bulk-create all remaining tasks.
3. **On redirect/early exit:** Complete preamble task, stop.
4. **Extensions:** Add task for each non-empty `USER-EXTENSION` block in Phase 2.
5. **Track:** Mark tasks `in_progress` -> `completed`.
6. **Post-Compaction:** Re-read spec, resume from first incomplete task.
---
## Workflow
### Step 0: Clear Task List
If not epic or branch tracker, clear tasks.

<!-- USER-EXTENSION-START: pre-work -->
<!-- USER-EXTENSION-END: pre-work -->

### Step 1: Context Gathering (Preamble Script)
Run `node .claude/scripts/shared/work-preamble.js` with `--issue N`, `--issues "N,N,N"`, or `--status <status>`. Append `--assign` for auto-assign.
Parse JSON: `ok: false` -> report errors, STOP. Extract `context`, `gates`, `autoTodo`, `warnings`.
**--assign errors:** `ALREADY_ASSIGNED`, `WORKSTREAM_CONFLICT`.
Run `--schema` for envelope field reference.

<!-- USER-EXTENSION-START: post-work-start -->
<!-- USER-EXTENSION-END: post-work-start -->

### Step 1a: CI Wait (--wait flag)
**Trigger:** `context.wait` is `true`.
```bash
node .claude/scripts/shared/wait-for-ci.js --branch $(git branch --show-current) --timeout 300
```
- Exit 0: continue. Exit 1 (fail): STOP. Exit 2 (timeout): STOP. Exit 3 (no runs): continue.
If not set, skip.
### Step 1b: Epic Complexity Assessment
**Trigger:** `context.type` is `"epic"` and `--nonstop` set.
Run `node .claude/scripts/shared/epic-complexity.js $ISSUE`. `"functional"` -> `strictTDD = true`. Signals: `.claude/metadata/epic-complexity-signals.json`.
### Step 2: Framework Methodology Dispatch
Load core file from `framework-config.json`. Missing: warn, continue.
### Step 2a: Load TDD Checklist
Read `.claude/skills/tdd-process/tdd-checklist.json`. Valid JSON with `red`, `green`, `refactor` (each: `required[]`, `gate`) and `failure-recovery` (`triggers[]`, `steps[]`):
- Set `tddChecklist` = loaded JSON. Each phase may have `deepReference: { skill, when }`.
Fail: warn, set `tddChecklist = null`.
### Step 3: Work the Issue
**Pre-Agent Status Gate:** Before spawning Agent for implementation, verify `in_progress`:
```bash
gh pmu view $ISSUE --json=status --jq='.status'
```
Not in progress -> `gh pmu move $ISSUE --status in_progress`. Only for implementation agents (not research/review).
Per AC: mark in_progress, TDD cycle (RED->GREEN->REFACTOR), run tests, mark completed, commit (`Refs #$ISSUE`).
**GATE: Do NOT start next AC until commit made.**
**Sub-Agent Review Gate:** After Agent tool, `git diff --name-only`. Read changed files, verify match. Mandatory when `strictTDD`. Not satisfied by summaries/tests alone.
**TDD (checklist-driven):** Execute `required[]` per phase, enforce `gate`. On failure-recovery trigger, execute recovery steps. On gate failure, check `deepReference` -- load `.claude/skills/{skill-name}/SKILL.md`, retry. Missing skill: warn, continue.
**TDD (inline fallback):** RED (failing test), GREEN (minimal pass), REFACTOR (analyze, report).
No auto-tasks: single unit. Post-compaction: resume from first incomplete AC.
### Step 3b: Documentation Judgment
Re-read `.claude/scripts/shared/lib/doc-templates.json` from disk. Create if warranted.

<!-- USER-EXTENSION-START: post-implementation -->
<!-- USER-EXTENSION-END: post-implementation -->

### Step 4: Verify Acceptance Criteria (with QA Extraction)
**Re-read files before evaluating each AC.** Do NOT evaluate from memory.
Can verify -> `[x]`. Cannot verify -> QA extraction (4a).
Update issue body via `gh pmu view/edit` with temp file.
#### Step 4a: QA Extraction -- Automatic Sub-Issue Creation
Re-read `.claude/scripts/shared/lib/qa-config.json`. Match unverifiable ACs against keywords. For each, **automatically**:
1. `gh pmu sub create --parent $ISSUE --title "QA: [AC description]" --label qa-required -F .tmp-qa-body.md`
2. Annotate parent AC as `[x] AC text -> QA: #NNN`
Silent, automatic. Works in standard and `--nonstop` mode.
#### Step 4b: Force-Move Prohibition
**NEVER** use `--force` to bypass unchecked ACs on issues you implemented. Legitimate: epic parents, external, branch trackers, test-plan approvals.
Detection: if issue has `story`/`enhancement` label AND was moved to `in_progress` this session -> **HALT**, report unchecked count, verify via Step 4.

<!-- USER-EXTENSION-START: post-ac-verification -->
<!-- USER-EXTENSION-END: post-ac-verification -->

#### Step 4c: Log Changed Files to Issue Body
```bash
git log --name-status --grep="Refs #$ISSUE" --pretty=format:"" | sort -u | grep -v "^$"
```
Categorize: `A`=Added, `M`=Modified, `D`=Deleted. Separate test/source via `.claude/scripts/shared/lib/classify-changed-files.js`. Append "Files Changed" section to issue body. Omit empty categories. Skip if no commits.
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
**Autonomous Epic/Branch processing:** Ascending numeric order (or custom Processing Order). Skip done/in_review.
**Default:** Per-sub-issue STOP.
**`--nonstop`:** No STOP between sub-issues. One commit per AC. Commits local. Failure halts with resume instructions.
**`--nonstop` rules:** `Refs #N` commits. No push (deferred to `/done`). Test/AC/QA/pmu failure halts immediately with sub-issue count and resume instructions.
**Post-compaction:** Check `gh pmu sub list`, resume from first incomplete.
**`--nonstop` summary:** Report processed/skipped/failed counts.
#### Step 6a: Post-Nonstop Audit
(1) Commit density warning. (2) AC checkbox audit. (3) Test coverage audit.
**After all sub-issues done:**
- **Epic:** Evaluate ACs, move to in_review, STOP
- **Branch tracker:** Report, STOP. Do NOT suggest "done":
```
All sub-issues on branch {branch-name} are in review or done.
Next: /merge-branch or /prepare-release
```
**Default mode:** Never skip per-sub-issue STOP. **Continuous:** Sub-issues only to `in_review` -- user runs `/done` after review.
---
## Error Handling
**STOP:** Not found, no branch, pmu failure, ALREADY_ASSIGNED, WORKSTREAM_CONFLICT.
**Non-blocking:** PRD tracker missing, framework missing, no ACs, already in_progress.
---
**End of /work Command**
