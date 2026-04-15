---
version: "v0.87.0"
description: Merge branch to main with gated checks (project)
argument-hint: "[--skip-gates] [--dry-run]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /merge-branch
Merge current branch to main with gated validation. For merges without version tags (features, refactoring). For versioned releases, use `/prepare-release`.
**Extension Points:** `.claude/metadata/extension-points.json` or `/extensions list --command merge-branch`
---
## Arguments
| Argument | Description |
|----------|-------------|
| `--skip-gates` | Emergency bypass (caution) |
| `--dry-run` | Preview only |
---
## Execution
**REQUIRED:**
1. Parse phases+extensions → `TaskCreate`
2. Task per active `USER-EXTENSION` block
3. Mark `in_progress` → `completed`
4. **Post-Compaction:** re-read, regenerate tasks

**Rules:** One task per numbered phase/step; one per active extension; skip commented-out; phase/step name as content.
---
## Pre-Checks
### Verify Feature Branch
```bash
BRANCH=$(git branch --show-current)
```
Must NOT be `main`. Typical: `feature/*`, `fix/*`, `idpf/*`, `patch/*`, `release/*`.
### Check Tracker
```bash
gh pmu branch current --json tracker
```
If present, closed at end.
---

<!-- USER-EXTENSION-START: pre-gate -->
<!-- Setup: prepare environment before gate checks -->
<!-- USER-EXTENSION-END: pre-gate -->

## Phase 1: Gates
**If `--skip-gates`, skip to Phase 2.**
### Default Gates (Framework-Provided)
Always run (cannot disable):
#### Gate 1.1: No Uncommitted Changes
```bash
git status --porcelain
```
**FAIL if output non-empty.**
#### Gate 1.2: Tests Pass
```bash
npm test 2>/dev/null || echo "No test script configured"
```
**FAIL if tests fail.** Skip if no script.

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

### Summary
- ✅ Passed
- ❌ Failed (with details)

**Any failure → STOP.**

<!-- USER-EXTENSION-START: post-gate -->
<!-- Post-gate: actions after all gates pass -->
<!-- USER-EXTENSION-END: post-gate -->

---
## Phase 2: Create and Merge PR
### 2.1: Push
```bash
git push origin $(git branch --show-current)
```
### 2.2: Create PR
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

### 2.3: Wait for Approval
**ASK USER:** Review and approve the PR.
```bash
gh pr view --json reviewDecision
```
#### Gate 2.4: PR Approved
**FAIL if not approved** (unless `--skip-gates`).
### 2.5: Merge
```bash
gh pr merge --merge
git checkout main
git pull origin main
```

<!-- USER-EXTENSION-START: post-merge -->
<!-- Post-merge: actions after PR is merged -->
<!-- USER-EXTENSION-END: post-merge -->

### 2.6: Workstream Detection (Post-Merge)
After merge, check workstream plan:
1. **Read from disk:** `loadWorkstreamsMetadata('.workstreams.json')`. Not found → skip.
2. **Check:** `postMergeWorkstreamCheck(metadata, mergedBranch)`. `isWorkstream: false` → skip.
3. **Update:** write `updatedMetadata` to `.workstreams.json` (status `"merged"`)
4. **Commit:** `git add .workstreams.json && git commit -m "Update workstream metadata: $BRANCH merged"`
5. **Sibling warning:** `activeSiblings` non-empty → `formatSiblingWarning(activeSiblings, sharedModules)`, display
6. **All merged:** `allMerged: true` → "All workstreams merged. Consider removing `.workstreams.json`."
---
## Phase 3: Cleanup
### 3.1: Close Tracker (if exists)
```bash
node .claude/scripts/shared/lib/active-label.js remove [TRACKER_NUMBER]
gh issue close [TRACKER_NUMBER] --comment "Branch merged to main"
```
### 3.2: Close Branch in Project
```bash
gh pmu branch close 2>/dev/null || echo "No branch to close"
```
### 3.3: Delete Branch
```bash
git push origin --delete $BRANCH
git branch -d $BRANCH
```

<!-- USER-EXTENSION-START: post-close -->
<!-- Post-close: notifications, announcements -->
<!-- USER-EXTENSION-END: post-close -->

---
## Completion
- ✅ All gates passed
- ✅ PR created and merged
- ✅ Tracker closed (if applicable)
- ✅ Branch deleted
---
## Comparison: /merge-branch vs /prepare-release
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

**Use `/merge-branch`:** Feature, fix, non-versioned work.
**Use `/prepare-release`:** Versioned releases with CHANGELOG + tags.
---
**End of Merge Branch**
