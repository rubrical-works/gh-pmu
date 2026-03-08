---
version: "v0.58.0"
description: Plan concurrent workstreams for parallel epic development
argument-hint: "<epic-numbers> [--streams N] [--dry-run] [--prefix <prefix>] [--cancel]"
---

<!-- MANAGED -->
# /plan-workstreams
Plan concurrent workstreams for parallel development across multiple epics. Analyzes codebase for module boundaries, computes conflict risk, and creates optimized workstream groupings.
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- At least 2 open epic issues
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `<epic-numbers>` | Yes (plan mode) | Two or more epic issue numbers (e.g., `#100 #101 #102`) |
| `--streams N` | No | Number of concurrent workstreams (default: 2) |
| `--dry-run` | No | Show analysis and grouping without creating branches |
| `--prefix <prefix>` | No | Branch prefix for workstream branches (default: `workstream/`) |
| `--cancel` | No | Cancel active workstream plan and clean up metadata |
| `--force` | No | Force cancel even when commits exist on workstream branches (requires `--cancel`) |
**Notes:** `--cancel` does not require epic numbers. `--force` requires `--cancel`. All other modes require 2+ epic numbers. Accepts `#N` or `N` format.
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
---
## Workflow
### Step 1: Parse and Validate Arguments
```bash
node .claude/scripts/shared/plan-workstreams.js [args]
```
Parse JSON output. Validation: 2+ epics for plan mode, `--cancel` needs no epics, `--streams` must be positive integer.
### Step 1b: Solo-Mode Detection
1. Call `checkSoloMode('framework-config.json')` from helper script
2. If `showAdvisory: true`: display advisory (informational, non-blocking)
3. If `showAdvisory: false`: continue silently
### Step 2: Validate Epic Issues
```bash
gh issue view $EPIC --json number,title,labels,state
```
| Check | Failure Response |
|-------|-----------------|
| Issue not found | "Epic #N not found." → STOP |
| Issue not open | "Epic #N is [state], not open." → STOP |
| Missing epic label | "Issue #N is not an epic." → STOP |
### Step 3: Gather Epic Context
1. Fetch sub-issues: `gh pmu sub list $EPIC --json`
2. Read epic description for scope
3. Extract module hints via `extractModuleHints()`
4. Build mapping via `buildEpicMapping()`, write to `.tmp-mappings.json`
**Epics with no sub-issues:** Uses epic body only.
**LLM Judgment:** After deterministic extraction, review descriptions/AC for additional modules regex may miss. Merge into mapping.
### Step 4: Codebase Analysis and Conflict Detection
1. **Directory scanning:** `scanDirectories(repoRoot)`
2. **Module boundary detection:** Cross-reference mappings against actual directories
3. **Conflict risk matrix:** `buildConflictMatrix(mappings)`
| Level | Condition |
|-------|-----------|
| HIGH | Epics share primary (non-utility) modules |
| LOW | Epics share only utility/shared modules |
| NONE | No module overlap |
Present analysis to user.
### Step 5: Workstream Grouping
1. **Greedy partitioning:** `groupWorkstreams(conflictMatrix, epicData, { streams })`
   - HIGH-risk pairs placed in same workstream (union-find)
   - Remaining distributed to balance story counts
2. **Present grouping** as formatted table
3. **User confirmation:** confirm, adjust, or cancel
4. **Adjustment validation:** `validateAdjustment()` — HIGH-risk pairs cannot be split
5. **Write plan:** `buildPlanOutput()` → `.tmp-plan.json`
### Step 6: Execute Plan (if not --dry-run)
1. `generateExecutionCommands(plan)` from helper
2. Execute sequentially: `branch-start` then `assign-epic` for each
3. Write metadata via `buildWorkstreamsMetadata()` → `.workstreams.json`
4. Commit and push `.workstreams.json` to all workstream branches
5. Report created branches and assignments
### Step 7: Worktree Setup Guide
1. `formatWorktreeGuide(metadata)` from helper
2. Output: branch names, `git worktree add` commands, merge order recommendation
3. Present guide to user
---
## Cancel Mode
When `--cancel` is specified:
### Cancel Step 1: Load Metadata and Safety Check
1. `loadWorkstreamsMetadata('.workstreams.json')` — if not found → STOP
2. Check branch commits via `checkBranchCommits(branch, sourceCommit)`
3. `buildCancelPlan(metadata, commitChecks)`
4. Display cancellation plan
5. **Safety gate:** All safe → proceed; commits without `--force` → STOP; commits with `--force` → warn and proceed
### Cancel Step 2: Epic Disposition
| Option | Action |
|--------|--------|
| Return to source | `buildEpicDispositionCommands(metadata, 'return')` |
| Clear to backlog | `buildEpicDispositionCommands(metadata, 'backlog')` |
| Reassign to branch | `buildEpicDispositionCommands(metadata, 'reassign', targetBranch)` |
### Cancel Step 3: Branch Unwinding
1. `buildBranchCleanupCommands(metadata)` — reverse mergeOrder
2. Execute sequentially (close tracker, delete remote, delete local)
3. Commit `.workstreams.json` removal
4. Report
### Cancel Step 4: Worktree Cleanup Reminder
1. `git worktree list --porcelain` → parse into `[{ path, branch }]`
2. Build cancelled branch list from metadata
3. `formatWorktreeCleanupReminder(cancelledBranches, worktreeList)` → null if no stale worktrees
4. Present reminder (informational only)
---
## Error Handling
| Situation | Response |
|-----------|----------|
| No arguments | Show usage → STOP |
| Single epic | "At least 2 epic numbers required." → STOP |
| Epic not found | "Epic #N not found." → STOP |
| Epic not open | "Epic #N is [state], not open." → STOP |
| Missing epic label | "Issue #N is not an epic." → STOP |
| Unknown flag | "Unknown flag: --X. Usage: ..." → STOP |
---
**End of /plan-workstreams Command**
