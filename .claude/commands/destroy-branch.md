---
version: "v0.80.0"
description: Safely delete branch with confirmation (project)
argument-hint: "[branch-name] [--force]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /destroy-branch
Safely abandon and delete a branch. Destructive — requires explicit confirmation.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command destroy-branch`
| Argument | Description |
|----------|-------------|
| `[branch-name]` | Branch to destroy (defaults to current) |
| `--force` | Skip confirmation |
## Pre-Checks
```bash
BRANCH=${1:-$(git branch --show-current)}
```
Block if main/master. Fail if branch does not exist (`git rev-parse --verify`).

<!-- USER-EXTENSION-START: pre-destroy -->
<!-- Pre-destruction validation: check for unmerged commits, etc. -->
<!-- USER-EXTENSION-END: pre-destroy -->

## Phase 1: Confirmation
**DESTRUCTIVE** — permanently deletes: local `$BRANCH`, remote `origin/$BRANCH`, artifacts `Releases/[prefix]/[id]/`, tracker (closed "not planned").
### Step 1.1: Show What Will Be Destroyed
```bash
git log main..$BRANCH --oneline 2>/dev/null || echo "No unmerged commits"
ls -la Releases/*/$BRANCH/ 2>/dev/null || echo "No release artifacts found"
```
### Step 1.2: Require Explicit Confirmation
**If `--force` NOT passed:**
**ASK USER:** Type full branch name `$BRANCH` to confirm. Mismatch → ABORT.

<!-- USER-EXTENSION-START: post-confirm -->
<!-- Post-confirmation: actions after user confirms but before deletion -->
<!-- USER-EXTENSION-END: post-confirm -->

## Phase 2: Close Tracker
```bash
gh pmu branch current --json tracker 2>/dev/null
```
If tracker found:
```bash
node .claude/scripts/shared/lib/active-label.js remove [TRACKER_NUMBER]
gh issue close [TRACKER_NUMBER] --reason "not planned" --comment "Branch destroyed via /destroy-branch."
gh pmu branch close 2>/dev/null || echo "No branch to close"
```
## Phase 3: Delete Artifacts
Parse branch name: `release/vX.Y.Z` → `Releases/release/vX.Y.Z/`, `patch/` → `Releases/patch/`, `feature/` → `Releases/feature/` (if exists).
```bash
ARTIFACT_DIR="Releases/${BRANCH_PREFIX}/${BRANCH_ID}"
if [ -d "$ARTIFACT_DIR" ]; then
  rm -rf "$ARTIFACT_DIR"
  git add -A
  git commit -m "chore: remove artifacts for destroyed branch $BRANCH"
fi
```
## Phase 4: Delete Branch
```bash
[ "$(git branch --show-current)" = "$BRANCH" ] && git checkout main && git pull origin main
git push origin --delete "$BRANCH" 2>/dev/null || echo "Remote branch not found"
git branch -D "$BRANCH"
```

<!-- USER-EXTENSION-START: post-destroy -->
<!-- Post-destruction: notifications, audit logging -->
<!-- USER-EXTENSION-END: post-destroy -->

## Completion
Branch destroyed: confirmed, tracker closed (not planned), artifacts deleted, remote+local deleted.
**Cannot be undone.** Recovery: `git reflog` (local, ~30 days), team members (remote), backups (artifacts).
**End of Destroy Branch**
