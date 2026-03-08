---
version: "v0.58.0"
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
| *(none)* | Interactive mode — select action |
## Execution
```bash
node .claude/scripts/shared/manage-skills.js "$ARGUMENTS"
```
The script is a library. Parse the command and execute:
### Direct Invocation Flow
1. **Parse arguments** via `parseCommand(args)` from the script
2. **Route by mode:**
   - `list` → `listSkills(projectDir, { verbose })`, format and display
   - `install` → `installSkill(projectDir, skillName)`, report result
   - `remove` → `removeSkill(projectDir, skillName)`, report result
   - `info` → `skillInfo(projectDir, skillName)`, format and display
   - `interactive` → `interactiveData(projectDir)`, use `AskUserQuestion` to pick action
### Interactive Mode
When no subcommand:
1. Call `interactiveData(projectDir)` for installed and available skills
2. Present grouped list via `AskUserQuestion` (installed with remove option, available with install option, info for any)
3. Execute selected action
### Post-Install Hooks
After `install`, check result for `postInstall` field. If present, report:
```
Post-install: {postInstall.description}
```
### Default Skills Indicator
When formatting `list` output, check each skill's `isDefault` field. If `true`, append `[default]` tag:
```
  ✓ tdd-red-phase [default] — Write failing tests for specific behavior
  ✓ electron-development — Patterns for Electron app development
    api-versioning — API versioning strategies
```
### Default Skill Removal Warning
When `remove` returns `isDefault: true`, display the warning from `result.warnings`:
```
⚠ 'tdd-red-phase' is a default skill and will be re-added on next charter refresh.
Removed: tdd-red-phase
```
### Tech Stack Recommendations
After `list`, if skills have `suggests` field:
```
Recommended for your stack: {skill-name} — {reason}
```
