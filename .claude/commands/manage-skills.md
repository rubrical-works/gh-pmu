---
version: "v0.65.0"
allowed-tools: Bash, AskUserQuestion
description: "Manage project skills: list, install, remove, info (project)"
argument-hint: "[list|install|remove|info] [name] [--verbose]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
Unified skill management for discovery, installation, removal, and status.
## Subcommands
| Subcommand | Description |
|------------|-------------|
| `list` | Show installed and available skills |
| `install <name>` | Install a skill by name |
| `remove <name>` | Remove an installed skill |
| `info <name>` | Show skill details |
| *(none)* | Same as `list` |
## Execution
Run the manage-skills script:
```bash
node .claude/scripts/shared/manage-skills.js "$ARGUMENTS"
```
The script has a CLI entry point and outputs JSON. Parse the JSON result and format for the user.
### Routing Reference
The script routes internally by subcommand:
   - `list` -> `listSkills(projectDir, { verbose })` with category grouping
   - `install` -> Resolves `hubDir` from `framework-config.json` → `frameworkPath`. Calls `installSkill(projectDir, hubDir, skillName)`
   - `remove` -> `removeSkill(projectDir, skillName)`
   - `info` -> `skillInfo(projectDir, skillName)`
### List Display Format
Both `/manage-skills` (no args) and `/manage-skills list` produce same output. Use `listSkills()` output (each skill has `name`, `description`, `installed`, `isDefault`, `category`).
```
Project Skills ({installedCount} installed / {availableCount} available)
Installed:
  checkmark skill-name [default] -- Description
  checkmark skill-name -- Description
Available:
  Testing:
      skill-name -- Description
  Architecture:
      skill-name -- Description
```
**Rules:**
- Summary line: installed/available counts
- Installed section: alphabetical; append `[default]` if `isDefault`
- Available section: grouped by `category` (title-cased), alphabetical within each
- Categories sorted alphabetically
### Post-Install Hooks
After `install`, check result for `postInstall` field. If present, report: `Post-install: {postInstall.description}`
### Default Skills Indicator
In `list` output, append `[default]` for skills with `isDefault: true`.
### Default Skill Removal Warning
When `remove` returns `isDefault: true`, display warning from `result.warnings`:
```
Warning: 'tdd-red-phase' is a default skill and will be re-added on next charter refresh.
Removed: tdd-red-phase
```
### Tech Stack Recommendations
After `list`, if skills have `suggests` field: `Recommended for your stack: {skill-name} -- {reason}`
