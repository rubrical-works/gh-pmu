---
version: "v0.65.0"
description: Safely delete branch with confirmation (project)
argument-hint: "[branch-name] [--force]"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /destroy-branch

Safely abandon and delete a branch. This is a destructive operation that requires explicit confirmation.

**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command destroy-branch`

---

## Arguments

| Argument | Description |
|----------|-------------|
| `[branch-name]` | Branch to destroy (optional, defaults to current) |
| `--force` | Skip confirmation (dangerous) |

---

## Pre-Checks

### Identify Target Branch

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

### Check Branch Exists

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

This will permanently delete:
- Local branch: `$BRANCH`
- Remote branch: `origin/$BRANCH`
- Release artifacts: `Releases/[prefix]/[identifier]/`
- Tracker issue (closed as "not planned")

### Step 1.1: Show What Will Be Destroyed

```bash
# Show unmerged commits (if any)
git log main..$BRANCH --oneline 2>/dev/null || echo "No unmerged commits"

# Show related artifacts
ls -la Releases/*/$BRANCH/ 2>/dev/null || echo "No release artifacts found"
```

### Step 1.2: Require Explicit Confirmation

**If `--force` is NOT passed:**

**ASK USER:** Type the full branch name to confirm destruction.

The user must type exactly: `$BRANCH`

**If input does not match, ABORT.**

<!-- USER-EXTENSION-START: post-confirm -->
<!-- Post-confirmation: actions after user confirms but before deletion -->
<!-- USER-EXTENSION-END: post-confirm -->

---

## Phase 2: Close Tracker

### Step 2.1: Find Tracker Issue

```bash
gh pmu branch current --json tracker 2>/dev/null
```

### Step 2.1.5: Remove Active Label

If a tracker issue was found:

```bash
node .claude/scripts/shared/lib/active-label.js remove [TRACKER_NUMBER]
```

### Step 2.2: Close as Not Planned

If a tracker issue exists:

```bash
gh issue close [TRACKER_NUMBER] \
  --reason "not planned" \
  --comment "Branch destroyed via /destroy-branch. Work abandoned."
```

### Step 2.3: Close Branch in Project

```bash
gh pmu branch close 2>/dev/null || echo "No branch to close"
```

---

## Phase 3: Delete Artifacts

### Step 3.1: Identify Artifact Directory

Parse branch name to find artifacts:
- `release/vX.Y.Z` → `Releases/release/vX.Y.Z/`
- `patch/vX.Y.Z` → `Releases/patch/vX.Y.Z/`
- `feature/name` → `Releases/feature/name/` (if exists)

### Step 3.2: Delete Artifact Directory

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

### Step 4.1: Switch to Main (if on target branch)

```bash
if [ "$(git branch --show-current)" = "$BRANCH" ]; then
  git checkout main
  git pull origin main
fi
```

### Step 4.2: Delete Remote Branch

```bash
git push origin --delete "$BRANCH" 2>/dev/null || echo "Remote branch not found"
```

### Step 4.3: Delete Local Branch

```bash
git branch -D "$BRANCH"
```

Note: Using `-D` (force delete) since we've confirmed the user wants to abandon unmerged work.

<!-- USER-EXTENSION-START: post-destroy -->
<!-- Post-destruction: notifications, audit logging -->
<!-- USER-EXTENSION-END: post-destroy -->

---

## Completion

Branch destroyed:
- ✅ User confirmed destruction
- ✅ Tracker issue closed (not planned)
- ✅ Artifacts deleted
- ✅ Remote branch deleted
- ✅ Local branch deleted

**This action cannot be undone.** If commits were not pushed elsewhere, they are lost.

---

## Recovery

If you need to recover a destroyed branch:

1. **If pushed to remote before deletion:** Check if any team member has the branch
2. **If only local:** Use `git reflog` within ~30 days to find the commit
3. **Artifacts:** Check backups or git history for artifact files

---

**End of Destroy Branch**
