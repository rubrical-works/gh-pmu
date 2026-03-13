---
version: "v0.62.1"
allowed-tools: Bash, AskUserQuestion
description: "Manage project skills: list, install, remove, info (project)"
argument-hint: "[list|install|remove|info] [name] [--verbose]"
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
**Note:** The script is a library. Parse command and execute accordingly:
### Direct Invocation Flow
1. **Parse arguments** via `parseCommand(args)`
2. **Route by mode:**
   - `list` -> Call `listSkills(projectDir, { verbose })`, format with category grouping
   - `install` -> Call `installSkill(projectDir, skillName)`, report result
   - `remove` -> Call `removeSkill(projectDir, skillName)`, report result
   - `info` -> Call `skillInfo(projectDir, skillName)`, format and display
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
