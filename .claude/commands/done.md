---
version: "v0.79.0"
description: Complete issues with criteria verification and status transitions (project)
argument-hint: "[#issue... | --all] (optional)"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /done
Complete one or more issues. Moves from `in_review` -> `done` with a STOP boundary. Only handles final transition -- `/work` owns `in_progress` -> `in_review`.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command done`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
- Issue in `in_review` status (use `/work` first)
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | No | Single issue number |
| `#issue #issue...` | | Multiple issue numbers |
| `--all` | | Complete all `in_review` issues on current branch |
| *(none)* | | Queries `in_review` issues for selection |
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
### Step 1: Context Gathering (Preamble Script)
**Single issue:**
```bash
node .claude/scripts/shared/done-preamble.js --issue $ISSUE
```
**Multiple issues:**
```bash
node .claude/scripts/shared/done-preamble.js --issues "$ISSUE1,$ISSUE2"
```
**No arguments (discovery mode):**
```bash
node .claude/scripts/shared/done-preamble.js
```
Parse JSON output and check `ok`:
- **`ok: false`:** Report errors. If `suggestion` present, include it. -> **STOP**
- **Discovery mode:**
  - `query` (no-args): Present `discovery.issues` for selection, re-run with `--issue N`.
  - `all` flag: Confirm, re-run with `--issues`. Deferred push. No issues: STOP.
- **`ok: true` with `diffVerification`:**
  - `requiresConfirmation: true`: Report warnings, ask "Continue?". Yes -> re-run with `--force-move`. No -> STOP.
  - `requiresConfirmation: false`: Proceed.
- **`ok: true` with `gates.movedToDone: true`:**
  - Report status transition, tracker link, next steps guidance.
Report `warnings[]` (non-blocking).
**Multiple issues:** Process each through Step 1, then Steps 2-3 once after last (batch push).
### Step 1a: Epic Detection
Check `context.issue.labels` for `epic` label.
**Not epic:** Skip to Step 2.
**Epic completion flow:**
1. Fetch sub-issues: `gh pmu sub list $ISSUE`
2. Classify by status (done=skip, in_review=queue, in_progress=warn about in_progress sub-issues, backlog/ready=warn)
3. If all sub-issues are already done: skip sub-issue processing, proceed directly to completing the epic
4. Process in_review sub-issues through done workflow, with per-sub-issue reporting. Deferred single batch push.
5. Complete the epic via preamble after all sub-issues processed
6. Report summary: sub-issues completed count + epic completed

<!-- USER-EXTENSION-START: pre-done -->
<!-- USER-EXTENSION-END: pre-done -->

<!-- USER-EXTENSION-START: post-done -->
<!-- USER-EXTENSION-END: post-done -->

### Step 1b: Post Work Summary Comment
After each issue moved to done (only when commits reference issue):
1. Find commits referencing issue
2. No commits: skip
3. Post comment with files changed and commit link via temp file
4. Non-blocking on failure
### Step 2: Push (Batch-Aware)
**Single/last:** `git push`. **Not last:** defer. **Nothing to push:** skip.
### Step 3: Background CI Monitoring (Batch-Aware)
**Only after actual push.**
1. Get SHA
2. Check `context.ci.hasPushWorkflows` -- false: skip
3. Pre-check paths-ignore -- all match: skip
4. Spawn: `node ./.claude/scripts/shared/ci-watch.js --sha $SHA --timeout 300`
**Exit codes:** 0=passed, 1=FAILED, 2=timeout, 3=no run, 4=cancelled.
### Step 4: Cleanup
**MUST DO:** Clear todo list.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | STOP |
| Already closed | Skip |
| Still in_progress | STOP |
| Other status | STOP |
| No issues in review | STOP |
| `gh pmu` fails | STOP |
---
**End of /done Command**
