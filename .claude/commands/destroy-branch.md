---
version: "v0.87.0"
description: Safely delete branch with confirmation (project)
argument-hint: "[branch-name] [--force]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /destroy-branch
Safely abandon and delete a branch. Destructive — requires explicit confirmation.
**Extension Points:** `.claude/metadata/extension-points.json` or `/extensions list --command destroy-branch`
---
## Arguments
| Argument | Description |
|----------|-------------|
| `[branch-name]` | Branch to destroy (defaults to current) |
| `--force` | Skip confirmation (dangerous) |
---
## Pre-Checks
### Identify Target
```bash
BRANCH=${1:-$(git branch --show-current)}
```
### Cannot Destroy Main
```bash
if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
  echo "ERROR: Cannot destroy main/master branch"
  exit 1
fi
```
### Check Exists
```bash
git rev-parse --verify "$BRANCH" 2>/dev/null
```
**FAIL if branch does not exist.**
---

<!-- USER-EXTENSION-START: pre-destroy -->
<!-- Pre-destruction validation: check for unmerged commits, etc. -->
<!-- USER-EXTENSION-END: pre-destroy -->

## Phase 1: Confirmation
**⚠️ DESTRUCTIVE OPERATION**

Will permanently delete:
- Local: `$BRANCH`
- Remote: `origin/$BRANCH`
- Artifacts: `Releases/[prefix]/[identifier]/`
- Tracker issue (closed "not planned")
### Step 1.1: Show What Will Be Destroyed
```bash
git log main..$BRANCH --oneline 2>/dev/null || echo "No unmerged commits"
ls -la Releases/*/$BRANCH/ 2>/dev/null || echo "No release artifacts found"
```
### Step 1.2: Require Explicit Confirmation
**If `--force` NOT passed:**
**ASK USER:** Type the full branch name to confirm destruction.
Must type exactly: `$BRANCH`
**If mismatch, ABORT.**

<!-- USER-EXTENSION-START: post-confirm -->
<!-- Post-confirmation: actions after user confirms but before deletion -->
<!-- USER-EXTENSION-END: post-confirm -->

---
## Phase 2: Close Tracker
### 2.1: Find Tracker
```bash
gh pmu branch current --json tracker 2>/dev/null
```
### 2.1.5: Remove Active Label
If tracker found:
```bash
node .claude/scripts/shared/lib/active-label.js remove [TRACKER_NUMBER]
```
### 2.2: Close as Not Planned
```bash
gh issue close [TRACKER_NUMBER] \
  --reason "not planned" \
  --comment "Branch destroyed via /destroy-branch. Work abandoned."
```
### 2.3: Close Branch in Project
```bash
gh pmu branch close 2>/dev/null || echo "No branch to close"
```
---
## Phase 3: Delete Artifacts
### 3.1: Identify Directory
- `release/vX.Y.Z` → `Releases/release/vX.Y.Z/`
- `patch/vX.Y.Z` → `Releases/patch/vX.Y.Z/`
- `feature/name` → `Releases/feature/name/` (if exists)
### 3.2: Delete
```bash
ARTIFACT_DIR="Releases/${BRANCH_PREFIX}/${BRANCH_ID}"
if [ -d "$ARTIFACT_DIR" ]; then
  rm -rf "$ARTIFACT_DIR"
  git add -A
  git commit -m "chore: remove artifacts for destroyed branch $BRANCH"
fi
```
---
## Phase 4: Delete Branch
### 4.1: Switch to Main (if on target)
```bash
if [ "$(git branch --show-current)" = "$BRANCH" ]; then
  git checkout main
  git pull origin main
fi
```
### 4.2: Delete Remote
```bash
git push origin --delete "$BRANCH" 2>/dev/null || echo "Remote branch not found"
```
### 4.3: Delete Local
```bash
git branch -D "$BRANCH"
```
`-D` (force) since user confirmed abandoning unmerged work.

<!-- USER-EXTENSION-START: post-destroy -->
<!-- Post-destruction: notifications, audit logging -->
<!-- USER-EXTENSION-END: post-destroy -->

---
## Completion
Branch destroyed:
- ✅ User confirmed
- ✅ Tracker closed (not planned)
- ✅ Artifacts deleted
- ✅ Remote deleted
- ✅ Local deleted

**This cannot be undone.** Unpushed commits are lost.
---
## Recovery
1. **Pushed before deletion:** Check teammate clones
2. **Local only:** `git reflog` within ~30 days
3. **Artifacts:** Check backups or git history
---
**End of Destroy Branch**
