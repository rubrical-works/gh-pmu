---
version: "v0.86.0"
description: Complete issues with criteria verification and status transitions (project)
argument-hint: "[#issue... | --all] (optional)"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /done
Move issues from `in_review` → `done` with a STOP boundary. Final transition only — `/work` owns `in_progress` → `in_review`.
**Extension Points:** `/extensions list --command done`

## Prerequisites
- `gh pmu` installed, `.gh-pmu.json` configured
- Issue in `in_review` (use `/work` first)

## Arguments
| Argument | Description |
|---|---|
| `#issue` | Single issue (`#42` or `42`) |
| `#issue #issue...` | Multiple |
| `--all` | All `in_review` on current branch (with confirmation) |
| *(none)* | Query `in_review` issues for selection |

## Execution Instructions
**REQUIRED:** Routed command — two-phase task creation. Phase 1: one preamble task. Phase 2 (after preamble confirms no redirect/early exit): bulk-create remaining tasks. Redirect/early exit → mark preamble complete, stop. For each non-empty `USER-EXTENSION`, add a Phase 2 task. Track `in_progress` → `completed`. Post-compaction: re-read spec, resume from first incomplete task — no re-routing.
---
## Workflow
### Step 1: Context Gathering (Preamble Script)
Consolidates validation, diff verification, status transition, tracker linking, CI pre-check.

```bash
node .claude/scripts/shared/done-preamble.js --issue $ISSUE          # single
node .claude/scripts/shared/done-preamble.js --issues "$ISSUE1,$ISSUE2"  # multiple
node .claude/scripts/shared/done-preamble.js                         # discovery
```

Parse JSON, check `ok`:
- **`ok: false`:** report `errors[]` (`code`, `message`, optional `suggestion`) → **STOP**
- **Discovery** (`discovery` field): `mode: 'query'` (no-args) — present `discovery.issues` for user selection, re-run with `--issue N`. `mode: 'all'` (`--all`) — present list, ask "Complete all N in_review issues?"; yes → re-run with `--issues` for all numbers (deferred push: single push after last); empty list → "No in_review issues on current branch", STOP.
- **`ok: true` + `diffVerification`:** `requiresConfirmation: true` → report `warnings`, ask "Continue? (yes/no)"; yes → re-run with `--force-move`; no → **STOP**. `requiresConfirmation: false` → already moved to done, proceed.
- **`ok: true` + `gates.movedToDone: true`:** report `Issue #$ISSUE: $TITLE → Done`. `context.trackerLinked: true` → `Linked #$ISSUE to branch tracker #$TRACKER`. `context.nextSteps` present → report `context.nextSteps.guidance` (approval-gate steps, e.g., `/review-prd` before `/create-backlog`).

Report any `warnings[]` (non-blocking).

**Multiple issues:** process each through Step 1 sequentially; execute Steps 2–3 once after the last (batch push). Count total at start, track position.

### Step 1a: Epic Detection (epic completion flow)
After preamble succeeds for a single issue, check `context.issue.labels` for `epic`. Not an epic → skip to Step 2.

**Epic detected:** `gh pmu sub list $ISSUE`, then classify:

| Sub-Issue Status | Action |
|---|---|
| `done` | Skip — already complete |
| `in_review` | Queue for done processing |
| `in_progress` | **Warn:** "Sub-issue #N is still in_progress — complete via /work first" |
| `backlog`/`ready`/other | **Warn:** "Sub-issue #N is in {status} — was never started" |

All `done` → skip processing, proceed to epic. `in_review` exist → process each through standard `/done` (Steps 1–3); per-sub-issue `Sub-issue #N: $TITLE → Done (M/T processed)`; push deferred until after epic. Then run preamble for the epic itself. Final report:
```
Epic #$ISSUE: $TITLE — Done
  Sub-issues completed: N
  Sub-issues already done: M
  Sub-issues warned (not ready): K
  Epic: Done
```
**Push for epics:** all sub-issue + epic transitions are a single batch — push deferred until after epic completes (Step 2).

<!-- USER-EXTENSION-START: pre-done -->
<!-- USER-EXTENSION-END: pre-done -->

<!-- USER-EXTENSION-START: post-done -->
<!-- USER-EXTENSION-END: post-done -->

### Step 1b: Post Work Summary Comment
After each issue moves to done, post a summary comment IF commits referencing the issue exist. `git log --all --oneline --grep="Refs #$ISSUE\|Fixes #$ISSUE\|Closes #$ISSUE"`. No commits → skip (no-op close). Otherwise: get latest SHA + `git diff --name-only $FIRST_COMMIT~1..$LATEST_COMMIT`, construct repo URL from `.gh-pmu.json` `repositories[0]`, post comment via `-F` containing `**Work completed:**` heading, a `Files changed:` bulleted list of backticked paths, and a `Commit: https://github.com/{owner}/{repo}/commit/{sha}` URL line (multiple commits → link latest). **Non-blocking:** comment failure → log warning, continue.

### Step 2: Push (Batch-Aware)
Single issue OR last in batch: `git push` → report `Pushed.` Not last → skip → `"Push deferred (N remaining)"`. **No-commit detection:** `git log @{u}..HEAD --oneline` empty → `"Nothing to push"`, skip to Step 3.

### Step 3: Background CI Monitoring (Batch-Aware)
**Only after push (Step 2 actually pushed).** Deferred/skipped → skip CI monitoring for this issue.

`sha=$(git rev-parse HEAD)`. Check `context.ci.hasPushWorkflows`: `false` → skip, report `"CI skipped (no push-triggered workflows)"`. **Pre-check paths-ignore:** `shouldSkipMonitoring(changedFiles, pathsIgnore)` is synchronous, returns `boolean`. `changedFiles` via `git diff --name-only HEAD~1`; `pathsIgnore` from workflow YAML. All match → skip, `"CI skipped (paths-ignore)"`. Otherwise spawn background (`run_in_background: true`):
```bash
node ./.claude/scripts/shared/ci-watch.js --sha $SHA --timeout 300
```
Report `"CI monitoring started in background."`

**Exit codes:**
| Code | Report |
|---|---|
| 0 | `"CI passed for #$ISSUE (duration)"` |
| 1 | `"CI FAILED. Failed step: \"step-name\". Run: gh run view <id> --log-failed"` |
| 2 | `"CI still running after 5m. Check: gh run list --commit $SHA"` |
| 3 | `"No CI run triggered (paths-ignore likely)"` |
| 4 | `"CI cancelled (superseded by newer push)"` |

Multiple workflows → report per-workflow from `workflows[]`.

### Step 4: Cleanup
**MUST DO:** Clear task list.
---
## Error Handling
| Situation | Response |
|---|---|
| Issue not found | "Issue #N not found." → STOP |
| Issue already closed | "Issue #N is already closed." → skip |
| Issue still in_progress | "Complete work first via /work." → STOP |
| Issue in other status | "Move to in_progress first via /work." → STOP |
| No issues in review | "No issues in review." → STOP |
| `gh pmu` fails | "Failed to update issue: {error}" → STOP |
---
**End of /done Command**
