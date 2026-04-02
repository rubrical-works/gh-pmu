---
version: "v0.77.4"
description: Create a branch with tracker issue (project)
argument-hint: "<branch-name> (e.g., release/v0.16.0, my-feature, bugfix-123)"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /create-branch
Creates a new branch and associated tracker issue for any branch type.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command create-branch`
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `$1` | Yes | Branch name (any valid git branch name) |
## Execution Guidance
Use 2-3 coarse tasks: "Validate and create branch", "Configure branch", "Report and commit". Validate branch name inline. Chain commands with `&&` or run in parallel where noted.
## Workflow
### Step 1: Validate and Check Working Directory
Validate branch name — must be valid git branch name (no spaces, no special chars git rejects). If invalid/empty, error and stop.
```bash
git status --porcelain
```
**If changes exist:** Report "Uncommitted changes detected. Carried to new branch." Save output for Step 4. Continue (do NOT block).

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
Creates git branch and tracker issue with `branch` label. Extract tracker number from output. Write tracker body with workflow instructions and update:
```bash
gh pmu edit [TRACKER_NUMBER] -F .tmp-body.md && rm .tmp-body.md
```
Tracker body includes: assign issues (`/assign-branch`), work all (`/work #tracker`), work single (`/work #N`), when ready (`/merge-branch` or `/prepare-release`).
### Step 3: Configure Branch (parallelizable)
```bash
git checkout "$BRANCH" && git push -u origin "$BRANCH"
```
**In parallel** (after checkout/push):
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
**If uncommitted changes in Step 1:** Report carried-over files.
**Conditional Commit Prompt:** If any changes exist:
**ASK USER:** "Stage and commit all changes to new branch? (y/n)"
- **Yes:** Auto-generate message: `chore: committed {basenames} during create-branch`. Run `git add -A && git commit -m "<msg>"`. Do NOT prompt for message.
- **No:** Continue without modifying working tree
**Always end with:**
```
Next steps:
1. Assign issues: /assign-branch #N #N ...
2. Work issues: work #N
3. When ready: /prepare-release
```
**End of Create Branch**
