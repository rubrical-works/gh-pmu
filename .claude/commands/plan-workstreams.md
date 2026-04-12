---
version: "v0.85.0"
description: Plan concurrent workstreams for parallel epic development
argument-hint: "<epic-numbers> [--streams N] [--dry-run] [--prefix <prefix>] [--cancel]"
copyright: "Rubrical Works (c) 2026"
---

<!-- MANAGED -->
# /plan-workstreams

Plan concurrent workstreams for parallel development across multiple epics. Analyzes codebase for module boundaries, computes conflict risk, and creates optimized workstream groupings.

## Prerequisites

- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- At least 2 open epic issues

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<epic-numbers>` | Yes (plan mode) | Two or more epic issue numbers (e.g., `#100 #101 #102`) |
| `--streams N` | No | Number of concurrent workstreams (default: 2) |
| `--dry-run` | No | Show analysis and grouping without creating branches |
| `--prefix <prefix>` | No | Branch prefix for workstream branches (default: `workstream/`) |
| `--cancel` | No | Cancel active workstream plan and clean up metadata |
| `--force` | No | Force cancel even when commits exist (requires `--cancel`) |

**Notes:**
- `--cancel` does not require epic numbers
- `--force` only used with `--cancel`
- Other modes require 2+ epic numbers
- Epic numbers accept `#N` or `N`

## Execution Instructions

**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Track Progress:** Mark todos `in_progress` → `completed`

## Workflow

### Step 1: Parse and Validate Arguments

```bash
node .claude/scripts/shared/plan-workstreams.js [args]
```

Parse JSON output for validated arguments.

**Validation rules:**
- At least 2 epic numbers required for plan mode
- `--cancel` mode does not require epic numbers
- `--streams` must be a positive integer
- Unknown flags produce clear error messages with usage examples

### Step 1b: Solo-Mode Detection

1. **Detect mode:** Call `checkSoloMode('framework-config.json')` from helper script
2. **If `showAdvisory: true`:** Display advisory before proceeding — informational only, does not block
3. **If `showAdvisory: false`:** Continue silently (team mode or reviewMode unset)

### Step 2: Validate Epic Issues

```bash
gh issue view $EPIC --json number,title,labels,state
```

| Check | Failure Response |
|-------|-----------------|
| Issue not found | "Epic #N not found." → STOP |
| Issue not open | "Epic #N is [state], not open." → STOP |
| Missing epic label | "Issue #N is not an epic — missing 'epic' label." → STOP |

### Step 3: Gather Epic Context

1. Fetch sub-issues: `gh pmu sub list $EPIC --json`
2. Read epic description for scope and affected areas
3. Extract module hints via `extractModuleHints()`
4. Build structured mapping via `buildEpicMapping()`, write to `.tmp-mappings.json`

**Epics with no sub-issues:** Uses epic body only — mapping derived entirely from epic description.

**LLM Judgment:** After extracting deterministic module hints, review each epic's description and sub-issue acceptance criteria to identify additional modules, directories, and files likely modified that regex extraction may miss. Merge LLM-identified modules into the mapping.

### Step 4: Codebase Analysis and Conflict Detection

1. **Directory scanning:** `scanDirectories(repoRoot)` identifies top-level modules and file counts
2. **Module boundary detection:** Cross-reference epic module mappings against actual directories
3. **Conflict risk matrix:** `buildConflictMatrix(mappings)` computes pairwise overlap

**Risk levels:**

| Level | Condition |
|-------|-----------|
| HIGH | Epics share primary (non-utility) modules |
| LOW | Epics share only utility/shared modules (lib/, utils/, shared/, helpers/, common/) |
| NONE | No module overlap between epics |

Present analysis to user.

### Step 5: Workstream Grouping

1. **Greedy partitioning:** `groupWorkstreams(conflictMatrix, epicData, { streams })`
   - HIGH-risk pairs placed in same workstream (via union-find)
   - Remaining epics distributed to balance story counts
2. **Present grouping:** Show plan as formatted table:

   | Workstream | Epics | Story Count | Rationale |
   |------------|-------|-------------|-----------|
   | 1 | #100, #101 | 7 | HIGH-risk pairs co-located |
   | 2 | #102 | 5 | Isolated scope |

3. **User confirmation:** Confirm, adjust (move epics between workstreams), or cancel
4. **Adjustment validation:** `validateAdjustment(plan, adjustment, conflictMatrix)` — HIGH-risk pairs cannot be split across workstreams
5. **Write plan:** Confirmed plan → `.tmp-plan.json` via `buildPlanOutput()`

### Step 6: Execute Plan (if not --dry-run)

1. **Generate commands:** `generateExecutionCommands(plan)`
2. **Execute sequentially:**
   - `branch-start`: `gh pmu branch start --name <branch>`
   - `assign-epic`: `gh pmu move <epic> --branch <branch>`
3. **Write metadata:** `buildWorkstreamsMetadata(plan, sourceBranch, sourceCommit)` → `.workstreams.json`
   - Fields: `created`, `sourceBranch`, `sourceCommit`, `streams` (per-branch epic list and status), `mergeOrder`
4. **Commit and push:** Commit `.workstreams.json` to all workstream branches
5. **Report:** List created branches and epic assignments

### Step 7: Worktree Setup Guide

1. **Generate:** `formatWorktreeGuide(metadata)`
2. **Output includes:**
   - Branch names and assigned epics with titles
   - Copy-pasteable `git worktree add` commands
   - Merge order recommendation based on conflict risk
3. **Present** guide text to user

## Cancel Mode

### Cancel Step 1: Load Metadata and Safety Check

1. **Load metadata:** `loadWorkstreamsMetadata('.workstreams.json')` — if not found: "No active workstream plan found." → STOP
2. **Check branch commits:** For each stream, `checkBranchCommits(branch, sourceCommit)`
3. **Build cancel plan:** `buildCancelPlan(metadata, commitChecks)`
4. **Display cancellation plan** showing branches, assigned epics, commit status, actions
5. **Safety gate:**
   - All branches safe (no commits): proceed to Step 2
   - Commits found without `--force`: warn with commit counts, require `--force` → STOP
   - Commits with `--force`: warn and proceed

### Cancel Step 2: Epic Disposition

| Option | Action |
|--------|--------|
| Return to source | `buildEpicDispositionCommands(metadata, 'return')` — moves epics back to `sourceBranch` |
| Clear to backlog | `buildEpicDispositionCommands(metadata, 'backlog')` — clears branch/status via `--backlog` |
| Reassign to branch | `buildEpicDispositionCommands(metadata, 'reassign', targetBranch)` — moves to specified branch |

Execute each command from generated list.

### Cancel Step 3: Branch Unwinding

1. **Generate cleanup:** `buildBranchCleanupCommands(metadata)`
   - Processes branches in reverse `mergeOrder` (last → first)
   - Per branch: close tracker as "not planned", delete remote, delete local
   - Final: remove `.workstreams.json`
2. **Execute sequentially:** Handle errors gracefully (branch already deleted, etc.)
3. **Commit:** Commit `.workstreams.json` removal
4. **Report:** Deleted branches, reassigned epics, removed metadata

### Cancel Step 4: Worktree Cleanup Reminder

1. **List worktrees:** `git worktree list --porcelain` parsed to `[{ path, branch }]`
2. **Build cancelled branch list** from metadata streams
3. **Format reminder:** `formatWorktreeCleanupReminder(cancelledBranches, worktreeList)`
   - Returns `null` if no stale worktrees → skip reminder
   - Returns formatted text with `git worktree remove [path]` commands
   - Paths with spaces quoted for safe copy-paste
4. **Present** to user — informational only, does not auto-remove

## Error Handling

| Situation | Response |
|-----------|----------|
| No arguments provided | Show usage → STOP |
| Single epic number | "At least 2 epic numbers required." → STOP |
| Epic not found | "Epic #N not found." → STOP |
| Epic not open | "Epic #N is [state], not open." → STOP |
| Missing epic label | "Issue #N is not an epic." → STOP |
| Unknown flag | "Unknown flag: --X. Usage: ..." → STOP |

**End of /plan-workstreams Command**
