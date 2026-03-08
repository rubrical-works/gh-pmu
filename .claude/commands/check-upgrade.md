---
version: "v0.58.0"
description: Verify hub upgrade integrity for project commands and scripts
argument-hint: ""
---
<!-- MANAGED -->
# /check-upgrade
Verifies hub upgrade integrity for a user project. After updating the hub, confirms extension blocks preserved, custom scripts survived, commands are current, and symlinks healthy.
**Backing script:** `.claude/scripts/shared/check-upgrade.js`
---
## Prerequisites
- Project installed via `install-project-existing.js` or `install-project-new.js`
- Hub updated via `install-hub.js`
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `--hub` | No | Path to hub directory (auto-detected from symlinks if omitted) |
| `--commit` | No | Auto-commit upgraded files if all checks pass |
| `--no-commit` | No | Do not commit |
---
## Workflow
### Step 1: Detect Hub Path
If `--hub` not provided, detect from symlink targets:
```bash
readlink .claude/rules
```
Extract hub root from symlink target. If detection fails: STOP.
### Step 2: Extension Integrity Check
Call `checkExtensionIntegrity(projectDir)`:
1. List all `.claude/commands/*.md` files
2. For extensible commands: extract `USER-EXTENSION-START/END` blocks
3. Check for non-empty content, compare against git state
| Result | Meaning |
|--------|---------|
| PASS | All extension blocks intact |
| WARN | Blocks exist but empty |
| FAIL | Blocks missing or content lost |
### Step 3: Custom Script Check
Call `checkCustomScripts(projectDir)`:
1. List files in `.claude/scripts/` recursively
2. Non-symlink files = user-created — verify they exist
| Result | Meaning |
|--------|---------|
| PASS | All custom scripts present |
| WARN | No custom scripts found |
| FAIL | Custom script(s) missing |
### Step 4: Command Version Drift Check
Call `checkCommandVersionDrift(projectDir, hubDir)`:
1. List all `.claude/commands/*.md` in project
2. Read EXTENSIBLE or MANAGED marker header
3. **Fast path:** If both files have version numbers, compare directly
4. **Default path (diff-based):** Normalize extension block content (strip user customizations), compare project vs hub content
5. For EXTENSIBLE commands, only template content compared — user extension block content stripped before diff
| Result | Meaning |
|--------|---------|
| PASS | All commands match hub version (by version or content diff) |
| FAIL | Command(s) have drifted — content differs from hub |
### Step 4.5: Stale Config Reference Check
Call `checkStaleConfigReferences(projectDir, hubDir)`:
1. Scan all `.claude/commands/*.md` for `.gh-pmu.yml` references
2. If hub available, classify:
   - **Stale:** Project has `.gh-pmu.yml`, hub has `.gh-pmu.json` → needs update
   - **Migrated:** Both have `.gh-pmu.json` → already updated
3. If no hub, flag all `.gh-pmu.yml` references as stale
| Result | Meaning |
|--------|---------|
| PASS | No stale `.gh-pmu.yml` references |
| WARN | Some commands still reference `.gh-pmu.yml` |
### Step 5: Symlink Health Check
Call `checkSymlinkHealth(projectDir)`:
Check: `.claude/rules/`, `.claude/hooks/`, `.claude/scripts/shared/`, `.claude/metadata/`, `.claude/skills/`
For each: verify symlink exists, target valid, target has files.
| Result | Meaning |
|--------|---------|
| PASS | All symlinks valid |
| WARN | Some symlinks missing |
| FAIL | Target invalid or empty |
### Step 6: Produce Report
Use PASS/WARN/FAIL indicators per check:
```
/check-upgrade Report
  ✅ Extension Integrity     — N commands checked, blocks intact
  ✅ Custom Scripts           — M scripts verified
  ✅ Version Drift            — N commands at current hub version
  ✅ Stale Config References  — No .gh-pmu.yml references found
  ✅ Symlink Health           — 5/5 valid
Overall: PASS
```
**Remediation for failures:**
- Extension FAIL: Re-run `install-hub.js` or restore from git
- Custom script FAIL: Restore from git
- Version drift FAIL: Re-run `install-hub.js`
- Stale config refs WARN: Replace `.gh-pmu.yml` with `.gh-pmu.json` or re-run project installer
- Symlink FAIL: Re-run project installer
### Step 7: Optional Commit
Parse the `---JSON---` structured output block from the script. Extract:
- `commitReady` — `true` when all checks passed and `--no-commit` was not set
- `changedFiles` — array of non-symlinked paths from `getCommitableFiles()`
- `hubVersion` — version string from `framework-config.json`
- `commitFlag` / `noCommitFlag` — which flag was passed (if any)
**If `commitReady` is `false`:** Skip commit. Report and end.
**If `commitFlag` is `true`:** The script auto-committed. Report the result and end.
**If neither flag was provided and `commitReady` is `true`:**
Use `AskUserQuestion` to prompt:
```
All checks passed. Commit upgraded files?
Changed files: [changedFiles list from JSON]
```
If user commits (stage only non-symlinked files from `changedFiles`):
```bash
git add .claude/commands/ framework-config.json
git commit -m "chore: upgrade hub to v{hubVersion}"
```
If `hubVersion` is null, use: `git commit -m "chore: upgrade hub"`
**Note:** Symlinked directories (`.claude/rules/`, `.claude/hooks/`, `.claude/scripts/shared/`, `.claude/metadata/`, `.claude/skills/`) are excluded — they point to the hub and are not project-owned.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Hub path not found | "Cannot detect hub path. Use `--hub <path>`." -> STOP |
| No .claude/ directory | "Not an IDPF project." -> STOP |
| Git not initialized | WARN, continue |
| Script require fails | Report error, continue with remaining checks |
---
**End of /check-upgrade Command**
