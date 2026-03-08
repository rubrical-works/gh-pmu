---
version: "v0.58.0"
description: Merge branch to main with gated checks (project)
argument-hint: "[--skip-gates] [--dry-run]"
---

<!-- EXTENSIBLE -->
# /merge-branch
Merge current branch to main with gated validation. For non-versioned merges (features, fixes). For versioned releases with tags, use `/prepare-release`.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command merge-branch`
---
## Arguments
| Argument | Description |
|----------|-------------|
| `--skip-gates` | Emergency bypass - skip all gates |
| `--dry-run` | Preview actions without executing |
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse phases and extension points, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo item for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction
**Todo Rules:** One todo per numbered phase/step; one todo per active extension; skip commented-out extensions.
---
## Pre-Checks
### Verify on Feature Branch
```bash
BRANCH=$(git branch --show-current)
```
Must NOT be on `main`.
### Check for Tracker Issue
```bash
gh pmu branch current --json tracker
```
If a tracker issue exists, it will be closed at the end.
---

<!-- USER-EXTENSION-START: pre-gate -->
<!-- Setup: prepare environment before gate checks -->
<!-- USER-EXTENSION-END: pre-gate -->

## Phase 1: Gate Checks
**If `--skip-gates` is passed, skip to Phase 2.**
### Default Gates (Framework-Provided)
#### Gate 1.1: No Uncommitted Changes
```bash
git status --porcelain
```
**FAIL if output is not empty.** All changes must be committed.
#### Gate 1.2: Tests Pass
```bash
npm test 2>/dev/null || echo "No test script configured"
```
**FAIL if tests fail.** Skip if no test script.

<!-- USER-EXTENSION-START: gates -->
#### Gate 1.3: E2E Tests Pass

**If `--skip-e2e` was passed, skip this gate.**

```bash
node .claude/scripts/e2e/run-e2e-gate.js
```

The script outputs JSON: `{"success": true/false, "testsRun": N, "testsPassed": N, "duration": N}`

**FAIL if `success` is false.**

E2E tests validate complete workflows against the test project.
<!-- USER-EXTENSION-END: gates -->

### Gate Summary
Report: Gate passed / Gate failed (with details)
**If any gate fails, STOP and report.**

<!-- USER-EXTENSION-START: post-gate -->
<!-- Post-gate: actions after all gates pass -->
<!-- USER-EXTENSION-END: post-gate -->

---
## Phase 2: Create and Merge PR
### Step 2.1: Push Branch
```bash
git push origin $(git branch --show-current)
```
### Step 2.2: Create PR
```bash
gh pr create --base main --head $(git branch --show-current) \
  --title "Merge: $(git branch --show-current)"
```

<!-- USER-EXTENSION-START: post-pr-create -->

### Wait for CI
```bash
node .claude/scripts/framework/wait-for-ci.js
```
<!-- USER-EXTENSION-END: post-pr-create -->

### Step 2.3: Wait for PR Approval
**ASK USER:** Review and approve the PR.
```bash
gh pr view --json reviewDecision
```
#### Gate 2.4: PR Approved
**FAIL if PR is not approved** (unless `--skip-gates`).
### Step 2.5: Merge PR
```bash
gh pr merge --merge
git checkout main
git pull origin main
```

<!-- USER-EXTENSION-START: post-merge -->
<!-- Post-merge: actions after PR is merged -->
<!-- USER-EXTENSION-END: post-merge -->

### Step 2.6: Workstream Detection (Post-Merge)
After merging to main, check if the merged branch is part of a workstream plan:
1. `loadWorkstreamsMetadata('.workstreams.json')` — if not found: skip
2. `postMergeWorkstreamCheck(metadata, mergedBranch)` — if not workstream: skip
3. Write `updatedMetadata` back (status → `"merged"`)
4. Commit: `git add .workstreams.json && git commit -m "Update workstream metadata: $BRANCH merged"`
5. If `activeSiblings` non-empty: `formatSiblingWarning()` and display
6. If `allMerged: true`: display cleanup note
---
## Phase 3: Cleanup
### Step 3.1: Close Tracker Issue (if exists)
Remove the active label before closing:
```bash
node .claude/scripts/shared/lib/active-label.js remove [TRACKER_NUMBER]
gh issue close [TRACKER_NUMBER] --comment "Branch merged to main"
```
### Step 3.2: Close Branch in Project (if exists)
```bash
gh pmu branch close 2>/dev/null || echo "No branch to close"
```
### Step 3.3: Delete Branch
```bash
git push origin --delete $BRANCH
git branch -d $BRANCH
```

<!-- USER-EXTENSION-START: post-close -->
<!-- Post-close: notifications, announcements -->
<!-- USER-EXTENSION-END: post-close -->

---
## Completion
Branch merge complete:
- All gates passed
- PR created and merged
- Tracker issue closed (if applicable)
- Branch deleted
---
## /merge-branch vs /prepare-release
| Feature | /merge-branch | /prepare-release |
|---------|---------------|------------------|
| Version bump | No | Yes |
| CHANGELOG update | No | Yes |
| Git tag | No | Yes |
| GitHub Release | No | Yes |
| Gates | Yes | Yes (via validation) |
| PR to main | Yes | Yes |
**Use `/merge-branch` for:** Feature branches, fix branches, non-versioned work.
**Use `/prepare-release` for:** Versioned releases with CHANGELOG and tags.
---
**End of Merge Branch**
