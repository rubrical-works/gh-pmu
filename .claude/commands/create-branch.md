---
version: "v0.74.0"
description: Create a branch with tracker issue (project)
argument-hint: "<branch-name> (e.g., release/v0.16.0, my-feature, bugfix-123)"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /create-branch
Creates a new branch and associated tracker issue for any branch type.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command create-branch`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `$1` | Yes | Branch name (any valid git branch name) |
---
## Execution Guidance
Use 2-3 coarse tasks: "Validate and create branch", "Configure branch", "Report and commit". Validate branch name inline (no dedicated tool call). Chain independent shell commands with `&&` or run in parallel where noted.
---
## Workflow
### Step 1: Validate and Check Working Directory
Validate branch name inline — must be valid git branch name (no spaces, no special characters git rejects). If invalid or empty, report error and stop.
```bash
git status --porcelain
```
**If changes exist:**
1. Report: "Uncommitted changes detected. These will be carried to the new branch."
2. Save output for Step 4 reporting
3. Continue with branch creation (do NOT block)

<!-- USER-EXTENSION-START: pre-create -->
### Verify Config
```bash
git status --porcelain .gh-pmu.json
```
**If `success` is false, STOP.**
<!-- USER-EXTENSION-END: pre-create -->

### Step 2: Create Branch and Populate Tracker
```bash
gh pmu branch start --name "$BRANCH"
```
Creates git branch `$BRANCH` and tracker issue with `branch` label. Extract tracker number from output.
Then write tracker body and update in one call:
```markdown
## Branch: $BRANCH

Tracker issue for branch `$BRANCH`.

### Workflow

- **Assign issues:** `/assign-branch #N #N ...`
- **Work all issues:** `/work #[tracker-number]` (processes sub-issues sequentially)
- **Work single issue:** `/work #N`
- **When ready:** `/merge-branch` or `/prepare-release`

### Sub-Issues

Issues assigned to this branch appear as sub-issues below.
```
```bash
gh pmu edit [TRACKER_NUMBER] -F .tmp-body.md && rm .tmp-body.md
```
### Step 3: Configure Branch (parallelizable)
Switch to branch, push to remote, set labels, and auto-assign tracker. Chain with `&&` or run in parallel tool calls.
```bash
git checkout "$BRANCH" && git push -u origin "$BRANCH"
```
**In parallel** (after checkout/push completes):
```bash
node .claude/scripts/shared/lib/active-label.js ensure [TRACKER_NUMBER]
```
```bash
gh pmu move [TRACKER_NUMBER] --branch "$BRANCH" && gh issue edit [TRACKER_NUMBER] --add-label assigned
```

<!-- USER-EXTENSION-START: post-create -->
<!-- USER-EXTENSION-END: post-create -->

### Step 4: Report Completion
```
Branch created.

Branch: $BRANCH
Tracker: #[tracker-issue-number]
```
**If uncommitted changes detected in Step 1:**
Report carried-over files from saved `git status --porcelain` output.
**Conditional Commit Prompt:**
If any changes exist (staged, unstaged, or untracked):
**ASK USER:** "Stage and commit all changes to new branch? (y/n)"
- **Yes:** Auto-generate commit message from changed files using format: `chore: committed {file summaries} during create-branch` (e.g., `chore: committed skill-keywords.json during create-branch`). Use basenames, not full paths. Run `git add -A && git commit -m "<auto-message>"`, report success. Do NOT prompt for a commit message.
- **No:** Continue without modifying working tree
**Always end with:**
```
Next steps:
1. Assign issues: /assign-branch #N #N ...
2. Work issues: work #N
3. When ready: /prepare-release
```
---
**End of Create Branch**
