---
version: "v0.58.0"
description: Tag beta from feature branch (no merge to main)
argument-hint: "[--skip-coverage] [--dry-run] [--help]"
---

<!-- EXTENSIBLE -->
# /prepare-beta
Tag a beta release from feature branch without merging to main.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command prepare-beta`
---
## Arguments
| Argument | Description |
|----------|-------------|
| `--skip-coverage` | Skip coverage gate |
| `--dry-run` | Preview without changes |
| `--help` | Show extension points |
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse phases and extension points, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo item for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` -> `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction
**Todo Rules:** One todo per numbered phase/step; one todo per active extension; skip commented-out extensions.
---
## Pre-Checks
### Verify NOT on Main
```bash
BRANCH=$(git branch --show-current)
if [ "$BRANCH" = "main" ]; then
  echo "Error: Cannot create beta from main."
  exit 1
fi
```
---

<!-- USER-EXTENSION-START: pre-phase-1 -->
<!-- USER-EXTENSION-END: pre-phase-1 -->

## Phase 1: Analysis
### Step 1.1: Analyze Changes
```bash
git log $(git describe --tags --abbrev=0)..HEAD --oneline
```
### Analyze Commits
```bash
node .claude/scripts/shared/analyze-commits.js
```
Outputs JSON: `lastTag`, `commits`, `summary` (counts by type).
### Recommend Version
```bash
node .claude/scripts/shared/recommend-version.js
```
Recommends beta version (e.g., `v1.0.0-beta.1`).

<!-- USER-EXTENSION-START: post-analysis -->
### Analyze Commits

```bash
node .claude/scripts/shared/analyze-commits.js
```

### Recommend Version

```bash
node .claude/scripts/shared/recommend-version.js
```

Recommend beta version (e.g., `v1.0.0-beta.1`).

### E2E Impact Analysis

```bash
node .claude/scripts/e2e/analyze-e2e-impact.js
```

The script analyzes which E2E tests may be impacted by changes:
- `impactedTests`: Test files that cover changed commands
- `newCommandsWithoutTests`: Commands modified without E2E coverage
- `recommendation`: Suggested test review actions

**If `newCommandsWithoutTests` is non-empty, warn user about missing coverage.**
<!-- USER-EXTENSION-END: post-analysis -->

**ASK USER:** Confirm beta version before proceeding.
---
## Phase 2: Validation

<!-- USER-EXTENSION-START: pre-validation -->
### Lint Gate

```bash
node .claude/scripts/prepare-release/lint.js
```

The script outputs JSON: `{"success": true/false, "message": "..."}`

**If `success` is false, STOP and report the error.**

Runs `golangci-lint run --timeout=5m` to catch lint errors before tagging.
<!-- USER-EXTENSION-END: pre-validation -->

<!-- USER-EXTENSION-START: post-validation -->
### Coverage Gate (Optional for Beta)

**If `--skip-coverage` was passed, skip this section.**

```bash
node .claude/scripts/prepare-release/coverage.js
```

**If `success` is false, STOP and report the error.**

### E2E Gate (Optional for Beta)

**If `--skip-e2e` was passed, skip this section.**

```bash
node .claude/scripts/e2e/run-e2e-gate.js
```

The script outputs JSON: `{"success": true/false, "testsRun": N, "testsPassed": N, "duration": N}`

**If `success` is false, STOP and report the error.**

E2E tests validate complete workflows against the test project.
<!-- USER-EXTENSION-END: post-validation -->

**ASK USER:** Confirm validation passed before proceeding.
---
## Phase 3: Prepare
Update CHANGELOG.md with beta section.

<!-- USER-EXTENSION-START: post-prepare -->
### Wait for CI

```bash
node .claude/scripts/shared/wait-for-ci.js
```

The script polls CI status every 60 seconds (5-minute timeout).

**If CI fails, STOP and report the error.**

<!-- USER-EXTENSION-END: post-prepare -->

<!-- USER-EXTENSION-START: pre-commit -->

<!-- USER-EXTENSION-END: pre-commit -->

---
## Phase 4: Tag (No Merge)
### Step 4.1: Commit Changes
```bash
git add -A
git commit -m "chore: prepare beta $VERSION"
git push origin $(git branch --show-current)
```

<!-- USER-EXTENSION-START: pre-tag -->

### Important Rules

1. **NEVER skip CI verification** - Always wait for green CI
2. **NEVER auto-create tags** - Always get user confirmation
3. **NEVER guess version numbers** - Base on actual commit analysis
4. **ALWAYS show changes before committing** - User must approve
5. **NEVER declare release complete after pushing tag** - Monitor until assets uploaded

### Beta Tag Authorization

The pre-push hook blocks version tags. For beta tags, authorize before pushing:

```bash
echo 'beta-authorized' > .release-authorized
git push origin $VERSION
rm .release-authorized
```
<!-- USER-EXTENSION-END: pre-tag -->

### Step 4.2: Create Beta Tag
**ASK USER:** Confirm ready to tag beta.
```bash
git tag -a $VERSION -m "Beta $VERSION"
git push origin $VERSION
```
**Note:** Beta tags feature branch. No merge to main.
### Step 4.3: Wait for CI Workflow

**Conditional:** Check if CI workflows exist before waiting.

```bash
# Check for .github/workflows/*.yml or *.yaml files
ls .github/workflows/*.yml .github/workflows/*.yaml 2>/dev/null
```

**If no workflow files found:** Skip CI wait with message:
```
No CI workflows detected — skipping CI wait.
```

**If workflow files exist:** Proceed normally:
```bash
node .claude/scripts/shared/wait-for-ci.js
```
**If CI fails, STOP and report.**
### Step 4.4: Update Release Notes
```bash
node .claude/scripts/shared/update-release-notes.js
```

<!-- USER-EXTENSION-START: post-tag -->
### Monitor Beta Build

```bash
node .claude/scripts/close-release/monitor-release.js
```

Monitor beta build and asset upload.

### Post-Release Reminder

**Releasing a beta does NOT close related issues.**

Issues included in this beta still require explicit user approval ("Done") to close.
Do NOT auto-close issues just because a beta shipped.
<!-- USER-EXTENSION-END: post-tag -->

---
## Next Step
Beta is tagged. When ready for full release:
1. Merge feature branch to main
2. Run `/prepare-release` for official release
---
## Summary Checklist
**Core (Before tagging):**
- [ ] Not on main branch
- [ ] Commits analyzed
- [ ] Beta version confirmed
- [ ] Tests passing
- [ ] CHANGELOG updated with beta section

<!-- USER-EXTENSION-START: checklist-before-tag -->
- [ ] Coverage gate passed (or `--skip-coverage`)
<!-- USER-EXTENSION-END: checklist-before-tag -->

**Core (After tagging):**
- [ ] Beta tag pushed
- [ ] CI workflow completed
- [ ] Release notes updated

<!-- USER-EXTENSION-START: checklist-after-tag -->
- [ ] Beta build monitored
<!-- USER-EXTENSION-END: checklist-after-tag -->

---
**End of Prepare Beta**
