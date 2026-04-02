---
version: "v0.79.0"
description: Merge branch to main with gated checks (project)
argument-hint: "[--skip-gates] [--dry-run]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /merge-branch
Merge current branch to main with gated validation. For non-versioned merges (features, refactoring). For versioned releases with tagging, use `/prepare-release`.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command merge-branch`

| Argument | Description |
|----------|-------------|
| `--skip-gates` | Emergency bypass - skip all gates |
| `--dry-run` | Preview actions without executing |

## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse phases and extension points, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` -> `completed`
4. **Post-Compaction:** Re-read spec and regenerate todos
## Pre-Checks
**Verify on feature branch:**
```bash
BRANCH=$(git branch --show-current)
```
Must NOT be on `main`. Typical: `feature/*`, `fix/*`, `idpf/*`, `patch/*`, `release/*`.
**Check for tracker:**
```bash
gh pmu branch current --json tracker
```
If tracker exists, it will be closed at the end.

<!-- USER-EXTENSION-START: pre-gate -->
<!-- Setup: prepare environment before gate checks -->
<!-- USER-EXTENSION-END: pre-gate -->

## Phase 1: Gate Checks
**If `--skip-gates` passed, skip to Phase 2.**
#### Gate 1.1: No Uncommitted Changes
```bash
git status --porcelain
```
**FAIL if output not empty.**
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

**Gate Summary:** Report pass/fail with details. **If any gate fails, STOP.**

<!-- USER-EXTENSION-START: post-gate -->
<!-- Post-gate: actions after all gates pass -->
<!-- USER-EXTENSION-END: post-gate -->

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
**FAIL if PR not approved** (unless `--skip-gates`).
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
After merging, check if branch is part of a workstream:
1. **Read metadata from disk:** Call `loadWorkstreamsMetadata('.workstreams.json')` -- if not found: skip
2. **Check workstream:** Call `postMergeWorkstreamCheck(metadata, mergedBranch)` -- if `isWorkstream: false`: skip
3. **Update metadata:** Write `updatedMetadata` to `.workstreams.json` (status -> `"merged"`)
4. **Commit:** `git add .workstreams.json && git commit -m "Update workstream metadata: $BRANCH merged"`
5. **Sibling warning:** If `activeSiblings` non-empty, call `formatSiblingWarning(activeSiblings, sharedModules)` and display
6. **All merged:** If `allMerged: true`: "All workstreams merged. Consider removing `.workstreams.json`."
## Phase 3: Cleanup
### Step 3.1: Close Tracker Issue (if exists)
```bash
node .claude/scripts/shared/lib/active-label.js remove [TRACKER_NUMBER]
gh issue close [TRACKER_NUMBER] --comment "Branch merged to main"
```
### Step 3.2: Close Branch in Project
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

## Completion
Branch merge complete: all gates passed, PR created and merged, tracker closed (if applicable), branch deleted.
## /merge-branch vs /prepare-release
| Feature | /merge-branch | /prepare-release |
|---------|---------------|------------------|
| Version bump | No | Yes |
| CHANGELOG update | No | Yes |
| Git tag | No | Yes |
| GitHub Release | No | Yes |
| Gates | Yes | Yes (via validation) |
| PR to main | Yes | Yes |
| Close tracker | Yes | Yes |
| Delete branch | Yes | Yes |

**Use `/merge-branch` for:** Feature/fix branches, non-versioned work.
**Use `/prepare-release` for:** Versioned releases with CHANGELOG and tags.
**End of Merge Branch**
