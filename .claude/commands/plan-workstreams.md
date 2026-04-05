---
version: "v0.82.0"
description: Plan concurrent workstreams for parallel epic development
argument-hint: "<epic-numbers> [--streams N] [--dry-run] [--prefix <prefix>] [--cancel]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /plan-workstreams
Plan concurrent workstreams for parallel development across multiple epics. Analyzes module boundaries, computes conflict risk, creates optimized workstream groupings.
**Prerequisites:** `gh pmu` installed, `.gh-pmu.json` configured, 2+ open epic issues.

| Argument | Required | Description |
|----------|----------|-------------|
| `<epic-numbers>` | Yes (plan mode) | Two or more epic issue numbers (e.g., `#100 #101 #102`) |
| `--streams N` | No | Number of concurrent workstreams (default: 2) |
| `--dry-run` | No | Show analysis without creating branches |
| `--prefix <prefix>` | No | Branch prefix (default: `workstream/`) |
| `--cancel` | No | Cancel active workstream plan and clean up |
| `--force` | No | Force cancel even with commits (requires `--cancel`) |

**Notes:** `--cancel` needs no epics. `--force` only with `--cancel`. Plan mode requires 2+ epics. Accepts `#N` or `N`.
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite`
2. **Track Progress:** Mark todos `in_progress` -> `completed`
## Workflow
### Step 1: Parse and Validate Arguments
```bash
node .claude/scripts/shared/plan-workstreams.js [args]
```
Parse JSON output. Validation: 2+ epics for plan mode, `--cancel` needs no epics, `--streams` positive integer, unknown flags error with usage.
### Step 1b: Solo-Mode Detection
1. Call `checkSoloMode('framework-config.json')` from helper
2. If `showAdvisory: true`: display advisory (informational, non-blocking)
3. If `showAdvisory: false`: continue silently
### Step 2: Validate Epic Issues
For each epic:
```bash
gh issue view $EPIC --json number,title,labels,state
```
| Check | Failure |
|-------|---------|
| Not found | "Epic #N not found." -> STOP |
| Not open | "Epic #N is [state], not open." -> STOP |
| Missing label | "Issue #N is not an epic -- missing 'epic' label." -> STOP |
### Step 3: Gather Epic Context
For each validated epic:
1. Fetch sub-issues: `gh pmu sub list $EPIC --json`
2. Read epic description for scope and affected areas
3. Extract module hints via `extractModuleHints()`
4. Build mapping via `buildEpicMapping()`, write to `.tmp-mappings.json`

**No sub-issues:** Uses epic body only for mapping.
**LLM Judgment:** After deterministic extraction, LLM reviews descriptions/AC for additional modules. Merge into mapping.
### Step 4: Codebase Analysis and Conflict Detection
Use `scanDirectories()` and `computeConflictRisk()`:
1. `scanDirectories(repoRoot)` identifies modules and file counts
2. Cross-reference epic mappings against actual directories
3. `buildConflictMatrix(mappings)` computes pairwise overlap

| Risk | Condition |
|------|-----------|
| HIGH | Share primary (non-utility) modules |
| LOW | Share only utility modules (lib/, utils/, shared/, helpers/, common/) |
| NONE | No overlap |

Present analysis to user.
### Step 5: Workstream Grouping
1. `groupWorkstreams(conflictMatrix, epicData, { streams })` -- HIGH-risk pairs in same workstream (union-find), balance by story count
2. Present as table: Workstream, Epics, Story Count, Rationale
3. **User confirmation:** Confirm, adjust, or cancel
4. `validateAdjustment()` -- HIGH-risk pairs cannot be split
5. Write confirmed plan to `.tmp-plan.json` via `buildPlanOutput()`
### Step 6: Execute Plan (if not --dry-run)
1. `generateExecutionCommands(plan)` from helper
2. Execute: `branch-start` creates branch, `assign-epic` assigns via `gh pmu move`
3. `buildWorkstreamsMetadata(plan, sourceBranch, sourceCommit)` -> `.workstreams.json` (fields: `created`, `sourceBranch`, `sourceCommit`, `streams`, `mergeOrder`)
4. Commit `.workstreams.json` to all workstream branches
5. Report branches and assignments
### Step 7: Worktree Setup Guide
1. `formatWorktreeGuide(metadata)` from helper
2. Output: branch names with epics, `git worktree add` commands, merge order recommendation
3. Present guide to user
## Cancel Mode
### Cancel Step 1: Load Metadata and Safety Check
1. `loadWorkstreamsMetadata('.workstreams.json')` -- if not found: "No active workstream plan found." -> STOP
2. For each stream: `checkBranchCommits(branch, sourceCommit)`
3. `buildCancelPlan(metadata, commitChecks)`
4. Display plan with branches, epics, commit status, actions
5. **Safety:** All safe -> proceed; commits without `--force` -> warn, STOP; commits with `--force` -> warn, proceed
### Cancel Step 2: Epic Disposition
| Option | Action |
|--------|--------|
| Return to source | `buildEpicDispositionCommands(metadata, 'return')` |
| Clear to backlog | `buildEpicDispositionCommands(metadata, 'backlog')` |
| Reassign to branch | `buildEpicDispositionCommands(metadata, 'reassign', targetBranch)` |

Execute each generated command.
### Cancel Step 3: Branch Unwinding
1. `buildBranchCleanupCommands(metadata)` -- reverse `mergeOrder`, close trackers, delete remote/local, remove `.workstreams.json`
2. Execute sequentially, handle errors gracefully
3. Commit `.workstreams.json` removal
4. Report deleted branches, reassigned epics, removed metadata
### Cancel Step 4: Worktree Cleanup Reminder
1. `git worktree list --porcelain` -> parse `[{ path, branch }]`
2. Build cancelled branch list from metadata
3. `formatWorktreeCleanupReminder(cancelledBranches, worktreeList)` -- returns `null` if none stale, else `git worktree remove` commands
4. Present reminder (informational, does not auto-remove)
## Error Handling
| Situation | Response |
|-----------|----------|
| No arguments | Show usage with error -> STOP |
| Single epic | "At least 2 epic numbers required." -> STOP |
| Epic not found | "Epic #N not found." -> STOP |
| Epic not open | "Epic #N is [state], not open." -> STOP |
| Missing epic label | "Issue #N is not an epic." -> STOP |
| Unknown flag | "Unknown flag: --X. Usage: ..." -> STOP |
**End of /plan-workstreams Command**
