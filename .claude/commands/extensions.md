---
version: "v0.85.0"
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
| `list` | All extension points |
| `list --command X` | Points for specific command |
| `list --status` | Points with X/. filled/empty markers |
| `list --status --command X` | One command's points with markers |
| `view X:Y` | Show content of point Y in command X |
| `edit X:Y` | Edit point Y in command X |
| `validate` | Check all blocks properly formatted |
| `summary` | Per-command filled/empty/total counts |
| `matrix` | Alias for `summary` |
| `recipes` | Common patterns for points |
| `recipes <category>` | Category recipes (ci, coverage, etc.) |
| `recipes --stack <tech>` | Filter by stack (e.g., `--stack node`). Auto-detects if omitted. |
---
## Target Commands
Listed in `.claude/metadata/extension-points.json` (registry).
**Registry:** `.claude/metadata/extension-points.json`
**Command files:** `.claude/commands/*.md` (for `edit`)
### Fallback: Registry Missing
If registry missing/unparseable:
1. **Warn:** `Extension registry not found. Scanning command files directly (slower).`
2. **Fallback:** Scan `.claude/commands/*.md` for EXTENSIBLE + USER-EXTENSION-START/END blocks
---
## Script Delegation
Read-only subcommands (`list`, `view`, `validate`, `summary`, `matrix`, `recipes`, `help`) handled by extensions-cli.js. Run and display stdout directly.
**Script:** `node .claude/scripts/shared/extensions-cli.js`
### Delegated Subcommands
| Subcommand | Script Command |
|------------|---------------|
| `list [--command X] [--status]` | `node .claude/scripts/shared/extensions-cli.js list [--command X] [--status]` |
| `view X:Y` | `node .claude/scripts/shared/extensions-cli.js view X:Y` |
| `validate` | `node .claude/scripts/shared/extensions-cli.js validate` |
| `summary` | `node .claude/scripts/shared/extensions-cli.js summary` |
| `matrix` | `node .claude/scripts/shared/extensions-cli.js matrix` (alias for summary) |
| `recipes [category] [--stack tech]` | `node .claude/scripts/shared/extensions-cli.js recipes [category] [--stack tech]` |
| `help` | `node .claude/scripts/shared/extensions-cli.js help` |
### Dispatch
1. Parse subcommand/args
2. Run script command via Bash
3. Display stdout directly
4. Non-zero exit (1=validation fail, 2=fatal) → report error

**Do NOT** re-interpret or reformat output — display as-is.
---
## Subcommand: edit
**Usage:** `/extensions edit <command>:<point>`
**Example:** `/extensions edit prepare-release:post-validation`
Spec-driven (not delegated) — requires interactive input and file modification.
### Step 1: Locate Block
Read `.claude/commands/{command}.md`. Find `USER-EXTENSION-START:{point}` / `END:{point}`. Not found → error, **STOP**.
### Step 2: Present Current
Show content between markers:
- Empty: `"Extension point '{command}:{point}' is currently empty."`
- Has content: fenced code block
### Step 3: Ask User for Intent (Not Raw Text)
**Do NOT ask for raw replacement text.** Ask User what they want:
- Empty: `"What would you like to add to this extension point?"`
- Has content: `"What changes would you like to make?"`

User describes intent in natural language.
### Step 4: Implement the Edit
Implement the Edit using the **Edit tool** on the command file (preserves formatting).
**Rules:**
- Only modify between START/END markers
- Preserve markers exactly
- Preserve formatting outside the block
- "Add": insert between markers
- "Remove": delete specified, leave markers (may become empty)
- "Replace": replace specified between markers
### Step 4b: Validate (Guardrail)
Validate against `command-spec-audit` skill (Category 4: Extension Points) if installed:
- **Formatting** — matches parent style
- **Conflicts** — not duplicating/contradicting built-ins
- **Size** — flag if >50 lines (recommend script)
- **Marker integrity** — START/END preserved

**Issues found:** Report with severity. Ask `"Fix these issues? (yes/skip)"`. Yes → apply+revalidate. Skip → proceed.
**Skill missing:** Silent skip (non-blocking).
**Passes:** Proceed to Step 5.
### Step 5: Confirm
Report updated block. Show before/after.
**Do NOT rebuild registry.** `hasContent` computed at runtime.
---
## Extension Point Naming
| Pattern | Purpose |
|---------|---------|
| `pre-*` | Before phase |
| `post-*` | After phase |
| `pre-commit` | Generate artifacts before commit |
| `checklist` | Single verification checklist |
| `checklist-before-*` | Pre-action items |
| `checklist-after-*` | Post-action items |
---
## Help
**Usage:** `/extensions --help` or `/extensions`
Delegate to script:
```bash
node .claude/scripts/shared/extensions-cli.js help
```
Display stdout directly.
---
**End of Extensions Command**
