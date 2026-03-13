---
version: "v0.62.0"
description: Complete issues with criteria verification and status transitions (project)
argument-hint: "[#issue... | --all] (optional)"
---
<!-- EXTENSIBLE -->

# /done
Complete one or more issues. Moves from `in_review` → `done` with a STOP boundary. Only handles the final transition — `/work` owns `in_progress` → `in_review`.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command done`

## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue in `in_review` status (use `/work` first)

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | No | Single issue number (e.g., `#42` or `42`) |
| `#issue #issue...` | | Multiple issue numbers (e.g., `#42 #43 #44`) |
| `--all` | | Complete all `in_review` issues on current branch (with confirmation) |
| *(none)* | | Queries `in_review` issues for selection |

## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction

## Workflow

### Step 1: Context Gathering (Preamble Script)
Run the preamble script to consolidate validation, diff verification, status transition, tracker linking, and CI pre-check into a single invocation:
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
Parse the JSON output and check `ok`:
- **If `ok: false`:** Report errors from `errors[]` array (each has `code` and `message`). If error has `suggestion`, include it. → **STOP**
- **If discovery mode** (`discovery` field present):
  - `discovery.mode: 'query'` (no-args): Present `discovery.issues` list for user selection. After user selects, re-run with `--issue N`.
  - `discovery.mode: 'all'` (`--all` flag): Present `discovery.issues` list for confirmation — "Complete all N in_review issues?" If user confirms, re-run preamble with `--issues` for all issue numbers. Deferred push applies (single push after last issue). If no issues found, report "No in_review issues on current branch" and STOP.
**If `ok: true` with `diffVerification`:**
- `diffVerification.requiresConfirmation: true` → Report warnings/concerns from `diffVerification.warnings`, ask user "Continue? (yes/no)". If yes, re-run with `--force-move`. If no → **STOP**.
- `diffVerification.requiresConfirmation: false` → Issue already moved to done. Proceed.
**If `ok: true` with `gates.movedToDone: true`:**
- Report: `Issue #$ISSUE: $TITLE → Done`
- If `context.trackerLinked: true`: Report `Linked #$ISSUE to branch tracker #$TRACKER`
- If `context.nextSteps` present: Report `context.nextSteps.guidance` (approval-gate next steps — e.g., `/review-prd` before `/create-backlog`)
**Report any warnings** from `warnings[]` array (non-blocking).
**Multiple issues:** Process each sequentially through Step 1, then execute Steps 2-3 once after the last issue (batch push optimization).
**Batch push detection:** When 2+ issues are in scope (explicit list, discovery selection, or branch tracker batch), determine the total count at the start. Track position as each issue completes Step 1.

### Step 1a: Epic Detection
After the preamble succeeds for a single issue, check whether the issue has the `epic` label (from `context.issue.labels` in the preamble output).
**If not an epic:** Skip to Step 2 (standard single-issue flow — no change).
**If epic label detected:** Enter the epic completion flow:
1. **Fetch sub-issues:**
   ```bash
   gh pmu sub list $ISSUE
   ```
2. **Classify sub-issues by status:**
   | Sub-Issue Status | Action |
   |------------------|--------|
   | `done` | Skip — already completed |
   | `in_review` | Queue for done processing |
   | `in_progress` | **Warn:** "Sub-issue #N is still in_progress — complete work via /work first" |
   | `backlog` / `ready` / other | **Warn:** "Sub-issue #N is in {status} — was never started" |
3. **If all sub-issues are already `done`:** Skip sub-issue processing, proceed directly to completing the epic itself.
4. **If `in_review` sub-issues exist:** Process each through the standard `/done` workflow (Steps 1-3 of the preamble), with per-sub-issue reporting:
   ```
   Sub-issue #N: $TITLE → Done (M/T processed)
   ```
   Push is deferred until after the epic (single push — batch-aware).
5. **Complete the epic:** After all sub-issues are processed, run the preamble for the epic issue itself to move it to done.
6. **Report summary:**
   ```
   Epic #$ISSUE: $TITLE — Done
     Sub-issues completed: N
     Sub-issues already done: M
     Sub-issues warned (not ready): K
     Epic: Done
   ```
**Push behavior for epics:** All sub-issue and epic done transitions are treated as a single batch — push is deferred until after the epic itself is completed (Step 2).
<!-- USER-EXTENSION-START: pre-done -->
<!-- USER-EXTENSION-END: pre-done -->
<!-- USER-EXTENSION-START: post-done -->
<!-- USER-EXTENSION-END: post-done -->

### Step 2: Push (Batch-Aware)
**Single issue or last issue in batch:**
```bash
git push
```
Report: `Pushed.`
**Not last issue in batch:** Skip push. Report: `"Push deferred (N remaining)"`
**No-commit detection:** Before pushing, check if there are unpushed commits:
```bash
git log @{u}..HEAD --oneline
```
If empty (nothing to push): Report `"Nothing to push"` and skip to Step 3.

### Step 3: Background CI Monitoring (Batch-Aware)
**Only execute after push (Step 2 actually pushed).** If push was deferred or skipped, skip CI monitoring for this issue.
After push:
1. Get SHA: `sha=$(git rev-parse HEAD)`
2. **Check `context.ci.hasPushWorkflows`** from preamble output:
   - If `false`, skip CI monitoring and report: `"CI skipped (no push-triggered workflows)"`
   - If `true`, continue to step 3.
3. **Pre-check paths-ignore:** `shouldSkipMonitoring(changedFiles, pathsIgnore)` is synchronous and returns `boolean`. Obtain `changedFiles` via `git diff --name-only HEAD~1` and `pathsIgnore` from workflow YAML. If all files match, skip CI monitoring and report: `"CI skipped (paths-ignore)"`
4. **Spawn background:** Bash with `run_in_background: true`:
   ```bash
   node ./.claude/scripts/shared/ci-watch.js --sha $SHA --timeout 300
   ```
5. Report: `"CI monitoring started in background."`
**Exit codes:**
| Code | Report |
|------|--------|
| 0 | `"CI passed for #$ISSUE (duration)"` |
| 1 | `"CI FAILED. Failed step: \"step-name\". Run: gh run view <id> --log-failed"` |
| 2 | `"CI still running after 5m. Check: gh run list --commit $SHA"` |
| 3 | `"No CI run triggered (paths-ignore likely)"` |
| 4 | `"CI cancelled (superseded by newer push)"` |
**Multiple workflows:** Report per-workflow from `workflows` array.

### Step 4: Cleanup
**MUST DO:** Clear todo list.

## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." → STOP |
| Issue already closed | "Issue #N is already closed." → skip |
| Issue still in_progress | "Complete work first via /work." → STOP |
| Issue in other status | "Move to in_progress first via /work." → STOP |
| No issues in review | "No issues in review." → STOP |
| `gh pmu` fails | "Failed to update issue: {error}" → STOP |
**End of /done Command**
