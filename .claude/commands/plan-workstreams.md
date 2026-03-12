---
version: "v0.61.0"
description: Plan concurrent workstreams for parallel epic development
argument-hint: "<epic-numbers> [--streams N] [--dry-run] [--prefix <prefix>] [--cancel]"
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
| `--force` | No | Force cancel even when commits exist on workstream branches (requires `--cancel`) |
**Notes:**
- `--cancel` does not require epic numbers
- `--force` can only be used with `--cancel`
- All other modes require at least 2 epic numbers
- Epic numbers accept `#N` or `N` format

## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work

## Workflow

### Step 1: Parse and Validate Arguments
Run the helper script to parse arguments:
```bash
node .claude/scripts/shared/plan-workstreams.js [args]
```
Parse the JSON output for validated arguments.
**Validation rules:**
- At least 2 epic numbers required for plan mode
- `--cancel` mode does not require epic numbers
- `--streams` must be a positive integer
- Unknown flags produce clear error messages with usage examples

### Step 1b: Solo-Mode Detection
Check if the user is in solo review mode and display an advisory:
1. **Detect mode:** Call `checkSoloMode('framework-config.json')` from the helper script
2. **If `showAdvisory: true`:** Display the advisory message before proceeding — informational only, does not block the command
3. **If `showAdvisory: false`:** Continue silently (team mode or reviewMode unset)

### Step 2: Validate Epic Issues
For each epic number, validate that the epic issues exist and have the `epic` label:
```bash
gh issue view $EPIC --json number,title,labels,state
```
| Check | Failure Response |
|-------|-----------------|
| Issue not found | "Epic #N not found." → STOP |
| Issue not open | "Epic #N is [state], not open." → STOP |
| Missing epic label | "Issue #N is not an epic — missing 'epic' label." → STOP |

### Step 3: Gather Epic Context
For each validated epic, read the issue body and sub-issues to understand scope:
1. Fetch sub-issues: `gh pmu sub list $EPIC --json`
2. Read epic description for scope and affected areas
3. Extract module hints using `extractModuleHints()` from the helper script
4. Build structured mapping using `buildEpicMapping()` and write to `.tmp-mappings.json`
**Epics with no sub-issues:** Uses epic body only — the mapping is derived entirely from the epic description without sub-issue expansion.
**LLM Judgment:** After extracting deterministic module hints, the LLM reviews each epic's description and sub-issue acceptance criteria to identify additional modules, directories, and files likely to be modified that regex-based extraction may miss. Merge LLM-identified modules into the mapping.

### Step 4: Codebase Analysis and Conflict Detection
Use the helper script's `scanDirectories()` to scan the codebase and `computeConflictRisk()` to detect module overlap:
1. **Directory scanning:** `scanDirectories(repoRoot)` identifies top-level modules and their file counts
2. **Module boundary detection:** Cross-reference epic module mappings against actual directories
3. **Conflict risk matrix:** `buildConflictMatrix(mappings)` computes pairwise overlap between epics
**Risk levels:**
| Level | Condition |
|-------|-----------|
| HIGH | Epics share primary (non-utility) modules |
| LOW | Epics share only utility/shared modules (lib/, utils/, shared/, helpers/, common/) |
| NONE | No module overlap between epics |
Present the analysis to the user.

### Step 5: Workstream Grouping
Using the conflict risk matrix, propose workstream groupings:
1. **Greedy partitioning:** Call `groupWorkstreams(conflictMatrix, epicData, { streams })` from the helper script
   - HIGH-risk pairs are placed in the same workstream (via union-find)
   - Remaining epics distributed to balance story counts across streams
2. **Present grouping:** Show suggested plan as a formatted table:
   | Workstream | Epics | Story Count | Rationale |
   |------------|-------|-------------|-----------|
   | 1 | #100, #101 | 7 | HIGH-risk pairs co-located |
   | 2 | #102 | 5 | Isolated scope |
3. **User confirmation:** Ask user to confirm, adjust (move epics between workstreams), or cancel
4. **Adjustment validation:** If user adjusts, call `validateAdjustment(plan, adjustment, conflictMatrix)` — HIGH-risk pairs cannot be split across workstreams
5. **Write plan:** Confirmed plan written to `.tmp-plan.json` via `buildPlanOutput()`

### Step 6: Execute Plan (if not --dry-run)
If confirmed (and not `--dry-run`):
1. **Generate commands:** Call `generateExecutionCommands(plan)` from the helper script
2. **Execute sequentially:** For each command in order:
   - `branch-start`: Create workstream branch via `gh pmu branch start --name <branch>`
   - `assign-epic`: Assign epic via `gh pmu move <epic> --branch <branch>`
3. **Write metadata:** Call `buildWorkstreamsMetadata(plan, sourceBranch, sourceCommit)` and write to `.workstreams.json`
   - Fields: `created`, `sourceBranch`, `sourceCommit`, `streams` (per-branch epic list and status), `mergeOrder`
4. **Commit and push:** Commit `.workstreams.json` to all workstream branches
5. **Report:** List created branches and epic assignments

### Step 7: Worktree Setup Guide
After execution, present a git worktree setup guide:
1. **Generate guide:** Call `formatWorktreeGuide(metadata)` from the helper script
2. **Output includes:**
   - Branch names and assigned epics with titles
   - Copy-pasteable `git worktree add` commands for each workstream
   - Merge order recommendation (which workstream to merge first based on conflict risk)
3. **Present** the guide text to the user

## Cancel Mode
When `--cancel` is specified:

### Cancel Step 1: Load Metadata and Safety Check
1. **Load metadata:** Call `loadWorkstreamsMetadata('.workstreams.json')` from the helper script
   - If not found: "No active workstream plan found." → STOP
2. **Check branch commits:** For each stream, call `checkBranchCommits(branch, sourceCommit)`
3. **Build cancel plan:** Call `buildCancelPlan(metadata, commitChecks)`
4. **Display cancellation plan** showing branches, assigned epics, commit status, and actions
5. **Safety gate:**
   - All branches safe (no commits): proceed to Cancel Step 2
   - Commits found without `--force`: warn with commit counts, require `--force` to proceed → STOP
   - Commits found with `--force`: warn and proceed

### Cancel Step 2: Epic Disposition
Prompt user for disposition of orphaned epics:
| Option | Action |
|--------|--------|
| Return to source | `buildEpicDispositionCommands(metadata, 'return')` — moves epics back to `sourceBranch` |
| Clear to backlog | `buildEpicDispositionCommands(metadata, 'backlog')` — clears branch/status via `--backlog` |
| Reassign to branch | `buildEpicDispositionCommands(metadata, 'reassign', targetBranch)` — moves to specified branch |
Execute each command from the generated list.

### Cancel Step 3: Branch Unwinding
1. **Generate cleanup commands:** Call `buildBranchCleanupCommands(metadata)` from the helper script
   - Processes branches in reverse `mergeOrder` (last → first)
   - For each branch: close tracker as "not planned", delete remote, delete local
   - Final command: remove `.workstreams.json`
2. **Execute sequentially:** Run each command, handle errors gracefully (branch already deleted, etc.)
3. **Commit:** Commit `.workstreams.json` removal
4. **Report:** List deleted branches, reassigned epics, removed metadata

### Cancel Step 4: Worktree Cleanup Reminder
After branch deletion, check for stale worktrees referencing cancelled branches:
1. **List worktrees:** Run `git worktree list --porcelain` and parse the output into `[{ path, branch }]`
2. **Build cancelled branch list** from the metadata streams
3. **Format reminder:** Call `formatWorktreeCleanupReminder(cancelledBranches, worktreeList)` from the helper script
   - Returns `null` if no stale worktrees found → skip reminder
   - Returns formatted text with `git worktree remove [path]` commands for each stale worktree
   - Paths with spaces are quoted for safe copy-paste
4. **Present reminder** to the user — informational only, does not auto-remove worktrees

## Error Handling
| Situation | Response |
|-----------|----------|
| No arguments provided | Show usage with error message → STOP |
| Single epic number | "At least 2 epic numbers required." → STOP |
| Epic not found | "Epic #N not found." → STOP |
| Epic not open | "Epic #N is [state], not open." → STOP |
| Missing epic label | "Issue #N is not an epic." → STOP |
| Unknown flag | "Unknown flag: --X. Usage: ..." → STOP |
**End of /plan-workstreams Command**
