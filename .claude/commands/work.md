---
version: "v0.87.0"
description: Start working on issues with validation and auto-task extraction (project)
argument-hint: "#issue [#issue...] [--assign] [--nonstop] [--wait] | all in <status>"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /work
Validates issue, branch assignment, type; moves to `in_progress`; extracts auto-tasks; dispatches to framework methodology.
**Extension Points:** `/extensions list --command work`

## Prerequisites
- `gh pmu` installed, `.gh-pmu.json` configured
- Issue assigned to a branch (`/assign-branch`, or `--assign`)

## Arguments
| Argument | Description |
|---|---|
| `#issue` (req'd) | Single issue (`#42` or `42`) |
| `#issue #issue...` | Multiple issues |
| `all in <status>` | All issues in given status |
| `--assign` | Assign to current branch first |
| `--nonstop` | Epic/branch: skip per-sub-issue STOP |
| `--wait` | Wait for pending CI |

## Execution Instructions
**REQUIRED:** Routed command — two-phase task creation. Task list IS the runtime step machine.

### Phase 1 — Preamble Task Only
Create exactly one task via `TaskCreate`:

| Subject | Rationale |
|---|---|
| `Step 1: Context Gathering (preamble script)` | All downstream gates depend on parsed context. Skipping collapses the compaction-recovery model — routing, auto-task, PRD tracker moves are all downstream of this JSON. |

Do NOT create Phase 2 tasks until routing is confirmed.

### Phase 2 — Enumerated Bulk Creation (after routing confirmed)
After Step 1 completes with no redirect/early exit, **Read** `.claude/scripts/shared/lib/work-task-definitions.json` and create one `TaskCreate` per entry, in order. Each entry supplies `subject`, `condition`, `description`. Conditional entries (non-`always` `condition`) are created OR explicitly skipped with a one-line note — never silently omitted.

**Per-AC subtasks (runtime-only, not in the JSON):** When Step 3 begins and `context.acceptanceCriteria` known, create one subtask per AC under Step 3, named `AC: <criterion text>`, description `Commit required after this AC (commit-per-AC gate).`

### Behavior Rules
1. **Redirect/early exit:** Mark preamble complete, stop. No Phase 2 tasks.
2. **Conditional skip:** Report `Skipped: Step Xa (condition: <condition> not met)`. Silent omission forbidden.
3. **Extensions:** For each non-empty `USER-EXTENSION`, add task `Extension: <block-id>` after the step it extends.
4. **Track:** `in_progress` → `completed`. Never mark complete what you did not execute.
5. **Post-compaction:** Re-read spec, call `TaskList`. First `in_progress` (or first `pending`) is resume point. Task list is authoritative.
6. **Description discipline:** Every `TaskCreate` MUST include rationale. Subject = *what*, description = *why*.
---
## Workflow
### Step 0: Conditional - Clear Task List
If not epic/branch tracker, clear task list.

<!-- USER-EXTENSION-START: pre-work -->
<!-- USER-EXTENSION-END: pre-work -->

### Step 1: Context Gathering (Preamble Script)
`node .claude/scripts/shared/work-preamble.js` with `--issue N`, `--issues "N,N,N"`, or `--status <status>`. `--assign` to auto-assign.

Parse JSON: `ok: false` → report `errors[]`, STOP. `ok: true` → extract `context`, `gates`, `autoTask`, `warnings`, report.

**--assign errors:** `ALREADY_ASSIGNED` (different branch), `WORKSTREAM_CONFLICT` (use `/assign-branch`). `--schema` for envelope reference.

<!-- USER-EXTENSION-START: post-work-start -->
<!-- USER-EXTENSION-END: post-work-start -->

### Step 1a: CI Wait (--wait)
**Trigger:** `context.wait == true`.
```bash
node .claude/scripts/shared/wait-for-ci.js --branch $(git branch --show-current) --timeout 300
```
0=pass, continue. 1=fail → **STOP**. 2=timeout → **STOP**. 3=no runs, continue.

### Step 1b: Epic Complexity Assessment
**Trigger:** `context.type == "epic"` and `--nonstop`. Run `node .claude/scripts/shared/epic-complexity.js $ISSUE`. `classification == "functional"` → `strictTDD = true`. Signals: `.claude/metadata/epic-complexity-signals.json`.

### Step 2: Framework Methodology Dispatch
Load `{frameworkPath}/{framework}/` core file from `framework-config.json`. Missing → warn, continue.

### Step 2a: Load TDD Checklist
Read `.claude/skills/tdd-process/tdd-checklist.json`. Valid (`red`/`green`/`refactor` each with `required[]` + `gate`, plus `failure-recovery` with `triggers[]` + `steps[]`) → `tddChecklist` = JSON; phases may include `deepReference: { skill, when }`. Invalid/missing → warn `"TDD checklist not found or invalid — using inline TDD behavior."`, set `tddChecklist = null`.

### Step 3: Work the Issue
**Pre-Agent Status Verification Gate (compaction-recovery safeguard):** Before spawning any Agent for **implementation work**, verify `in_progress` via `gh pmu view $ISSUE --json=status --jq='.status'`. Not in progress → `gh pmu move $ISSUE --status in_progress` before spawning. Implementation Agents only — NOT research/review/Explore.

For each AC (or batch of related ACs): mark `in_progress` → TDD cycle (RED → GREEN → REFACTOR) → run full test suite (all must pass) → mark `completed` → **COMMIT** `Refs #$ISSUE — <description>`. One commit per AC; related ACs may share, gate still applies after the batch. **GATE: Do NOT start the next AC until commit is made.**

**Sub-Agent Review Gate:** After any Agent returns, `git diff --name-only`. Read each modified file, verify changes match current AC, fix mismatches before committing. Mandatory when `strictTDD`; post-hoc otherwise. Gate is NOT satisfied by trusting agent summaries or passing tests alone — **file content must be read and verified**.

**TDD Execution (`tddChecklist` loaded):** for each phase, execute every item in `{phase}.required`, enforce `{phase}.gate` before proceeding. On `failure-recovery.triggers` match, execute `failure-recovery.steps`. **Deep Reference:** if a phase gate fails first attempt, check `{phase}.deepReference` — Read `.claude/skills/{skill}/SKILL.md`, retry. Missing skill → warn `"Deep reference skill '{name}' not found"`, proceed.

**TDD fallback (`tddChecklist == null`):** RED=failing test, GREEN=minimal pass, REFACTOR=analyze duplication/naming/complexity, report decision, keep tests passing.

No auto-tasks → single unit. Post-compaction: re-read spec, resume from first incomplete AC.

### Step 3b: Documentation Judgment
Evaluate whether design-decision/tech-debt documentation is warranted. Re-read `.claude/scripts/shared/lib/doc-templates.json` from disk for criteria, paths, naming. If warranted, create document and commit `Refs #$ISSUE`.

<!-- USER-EXTENSION-START: post-implementation -->
<!-- USER-EXTENSION-END: post-implementation -->

### Step 4: Verify Acceptance Criteria (with QA Extraction)
**IMPORTANT — Ground in file state:** Re-read each file via Read tool before evaluating its AC. Do NOT evaluate from memory. Prevents batch fatigue hallucination.

Per AC: verifiable → `[x]`; unverifiable → auto-extract via Step 4a, mark `[x]` with annotation. After all ACs resolved, update issue body via `.tmp-$ISSUE.md` flow (`gh pmu view --body-stdout` → edit → `gh pmu edit -F` → `rm`).

#### Step 4a: QA Extraction — Automatic Sub-Issue Creation
Run `node .claude/scripts/shared/qa-extract.js --issue $ISSUE`. Script reads `qa-config.json`, matches unverifiable ACs against keywords, creates labeled QA sub-issues, and returns `{ matched: [{ acText, subIssueNumber, annotation }] }`. Apply each `annotation` to the parent body. Works identically in standard and `--nonstop`.

#### Step 4b: Force-Move Prohibition
**NEVER** use `gh pmu move --force` to bypass unchecked ACs on issues you implemented. Legitimate: epic parents, external closures, branch trackers, test-plan approvals. Detection: `story`/`enhancement` label AND `in_progress` this session → **HALT**, report unchecked count, verify via Step 4 instead.

<!-- USER-EXTENSION-START: post-ac-verification -->
<!-- USER-EXTENSION-END: post-ac-verification -->

#### Step 4c: Log Changed Files to Issue Body
Run `node .claude/scripts/shared/log-changed-files.js --issue $ISSUE`. Prints formatted `### Files Changed` section to stdout (empty if no commits match — caller skips append). Append stdout (when non-empty) to issue body via `gh pmu view --body-stdout` → edit → `gh pmu edit -F`.

### Step 5: Move to in_review
```bash
gh pmu move $ISSUE --status in_review
```

### Step 6: STOP Boundary — Report and Wait
```
Issue #$ISSUE: $TITLE — In Review
Say "done" or run /done #$ISSUE to close this issue.
```
**STOP.** Wait for "done". Do NOT close.

**Post-STOP Cleanup:** Clear the task list after presenting the STOP message. Tasks have served their purpose (progress tracking + compaction recovery). Ensures the next command — `/work`, `/review-issue`, `/assign-branch`, etc. — starts clean. Applies to standard, epic, and branch tracker paths alike. Step 0's conditional clear remains as a safety net for crashed/interrupted sessions.

**CRITICAL — Autonomous Epic/Branch processing:** For `context.type == "epic"`/`"branch"`, process sub-issues in ascending numeric order (default) or custom **Processing Order:** from epic body. Skip sub-issues already in `in_review` or `done`.

**Default mode** (per `02-github-workflow.md` §4): each sub-issue → `in_progress` → Steps 3–4 → `in_review` → **STOP** per sub-issue → user "done" → next.

**`--nonstop` mode:** same cycle, **no STOP** between sub-issues. Report `"Sub-issue #N: $TITLE → In Review (M/T processed)"`, continue. Ignored for standard issues. One commit/AC (`Refs #N`); commits local (push deferred to `/done`). Test/AC/QA/`gh pmu` failure halts immediately — report which sub-issue, count completed, resume instructions. Post-compaction: re-read spec, `gh pmu sub list $ISSUE`, resume from first not in `in_review`/`done`. Final: `Nonstop Processing Complete` (processed/skipped/failed).

#### Step 6a: Post-Nonstop Audit
Run `node .claude/scripts/shared/nonstop-audit.js --issue $ISSUE` after all sub-issues reach `in_review`, before moving the epic. Returns `{ ok, warnings, blocks }`: (1) commit density (warnings, non-blocking), (2) AC checkbox (blocks — require Step 4 on flagged sub-issues). Audit (3) test coverage remains skill-delegated to `tdd-refactor-coverage-audit` (rubrical-works/idpf-skills-dev#168) — not in-script.

**After all sub-issues reach `in_review`/`done`:**
- **Epic:** Evaluate epic acceptance criteria via Step 4, move to `in_review`, **STOP** — wait for "done".
- **Branch tracker:** Report and STOP. Do **NOT** suggest "done" or `/done`:
```
All sub-issues on branch {branch-name} are in review or done.
  Sub-issues processed: N
  Sub-issues skipped: M
Next: /merge-branch or /prepare-release
```
**Default mode:** Never skip per-sub-issue STOP boundary. **Continuous mode:** sub-issues moved only to `in_review`, not `done` — user runs `/done` after review.
---
## Error Handling
**STOP errors:** Issue not found, no branch assignment, `gh pmu` failure, `ALREADY_ASSIGNED`, `WORKSTREAM_CONFLICT`.
**Non-blocking:** PRD tracker not found, framework file missing, no acceptance criteria, issue already `in_progress`.
---
**End of /work Command**
