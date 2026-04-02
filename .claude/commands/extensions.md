---
version: "v0.77.4"
description: Discover, view, and manage extension points in release commands
argument-hint: "list|view|edit|validate|summary|recipes [args]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /extensions
Unified management of extension points across release commands.
## Subcommands
| Subcommand | Description |
|------------|-------------|
| `list` | Show all extension points |
| `list --command X` | Points for specific command |
| `list --status` | Points with X/. filled/empty markers |
| `list --status --command X` | One command with markers |
| `view X:Y` | Show content of point Y in command X |
| `edit X:Y` | Edit point Y in command X |
| `validate` | Check all extension blocks formatting |
| `summary` | Per-command filled/empty/total counts |
| `matrix` | Alias for `summary` |
| `recipes` | Common patterns for extension points |
| `recipes <category>` | Recipes for specific category |
---
## Target Commands
**Registry:** `.claude/metadata/extension-points.json`
**Command files:** `.claude/commands/*.md` (for `edit`)
### Fallback: Registry Missing
1. **Warn:** `Extension registry not found. Scanning command files directly (slower).`
2. **Fallback:** Scan `.claude/commands/*.md` for EXTENSIBLE markers and USER-EXTENSION blocks
---
## Script Delegation
Read-only subcommands delegated to `node .claude/scripts/shared/extensions-cli.js`.
| Subcommand | Script Command |
|------------|---------------|
| `list [--command X] [--status]` | `extensions-cli.js list [--command X] [--status]` |
| `view X:Y` | `extensions-cli.js view X:Y` |
| `validate` | `extensions-cli.js validate` |
| `summary` | `extensions-cli.js summary` |
| `matrix` | `extensions-cli.js matrix` |
| `recipes [category]` | `extensions-cli.js recipes [category]` |
| `help` | `extensions-cli.js help` |
**Dispatch:** Parse subcommand, run via Bash, display stdout as-is (do NOT reformat). Non-zero exit: 1=validation failures, 2=fatal error.
---
## Subcommand: edit
**Usage:** `/extensions edit <command>:<point>`
Spec-driven (not delegated) -- requires interactive input and file modification.
### Step 1: Locate Extension Block
Read `.claude/commands/{command}.md`. Find `USER-EXTENSION-START:{point}` and `USER-EXTENSION-END:{point}`. Not found -> error and **STOP**.
### Step 2: Present Current Content
Empty: `"Extension point '{command}:{point}' is currently empty."`
Has content: display in fenced code block.
### Step 3: Ask User Intent
**Do NOT ask for raw text.** Ask what they want:
- Empty: "What would you like to add?"
- Has content: "What changes would you like to make?"
### Step 4: Implement Edit
Use **Edit tool** on command file.
- Only modify between START/END markers
- Preserve markers exactly
- Preserve surrounding formatting
- Add/remove/replace as appropriate
### Step 4b: Validate (Guardrail)
If `command-spec-audit` skill installed, check: formatting, conflicts, size (>50 lines -> recommend refactoring), marker integrity.
**Issues found:** Report, ask `"Fix these issues? (yes/skip)"`. Fix if yes.
**Not installed:** Skip silently.
### Step 5: Confirm Change
Show before/after summary. **Do NOT rebuild registry** (`hasContent` computed at runtime).
---
## Naming Convention
| Pattern | Purpose |
|---------|---------|
| `pre-*` | Before a phase |
| `post-*` | After a phase |
| `pre-commit` | Artifacts before commit |
| `checklist` | Single verification |
| `checklist-before-*` | Pre-action verification |
| `checklist-after-*` | Post-action verification |
---
## Help
No subcommand or `--help`:
```bash
node .claude/scripts/shared/extensions-cli.js help
```
Display stdout directly.
---
**End of Extensions Command**
