---
version: "v0.79.0"
description: Produce DTCG-compliant design tokens with pluggable adapter architecture (project)
argument-hint: "[--discover | --export <adapter> | --theme <name>]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /design-system
Produces a DTCG-compliant design token file (`Design-System/idpf-design.tokens.json`) with pluggable adapter architecture. Integrates with `/mockups` and `/catalog-screens`.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command design-system`
## Prerequisites
- Node.js built-in modules only (no npm dependencies)
- DTCG schema library: `.claude/scripts/shared/lib/dtcg-schema.js`
- DTCG token library: `.claude/scripts/shared/lib/dtcg-tokens.js`
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| *(none)* | No | Init mode — guided walkthrough |
| `--discover` | No | Extract tokens from existing code via adapters |
| `--export <adapter>` | No | Export to framework-specific format |
| `--export all` | No | Export to all detected formats |
| `--theme <name>` | No | Generate or apply theme override |
## Execution Instructions
**REQUIRED:** Parse workflow steps, use `TaskCreate` to create tasks. Track progress.
## Workflow
### Step 1: Check for Existing Token File
Check `Design-System/idpf-design.tokens.json`:
- **Exists:** Load, validate against schema, report status.
- **Missing:** Proceed to init mode (Step 2).
### Step 2: Mode Selection
- No args + no tokens: Init (Step 3)
- `--discover`: Discovery (Step 4)
- `--export`: Export (Step 5)
- `--theme`: Theme (Step 6)
- No args + tokens exist: Show summary and available actions
### Step 3: Init Mode (Interactive Walkthrough)
Guided walkthrough:
1. **Color palette:** primary, secondary, accent, neutral scale, semantic
2. **Typography:** families, size scale, weight scale
3. **Spacing:** base unit and scale
4. **Component patterns:** border radii, shadows, transitions
Use `AskUserQuestion` per category. User can skip optional categories.
Output: valid DTCG `idpf-design.tokens.json` + `idpf-design.schema.json`.

<!-- USER-EXTENSION-START: discovery-adapters -->
<!-- USER-EXTENSION-END: discovery-adapters -->

### Step 4: Discovery Mode
Load adapters from `Design-System/adapters/discovery/`:
1. Run each adapter's `detect(projectRoot)`
2. If `true`, run `extract(projectRoot)`
3. Merge discovered tokens
4. Present to user for confirmation
5. Write to `idpf-design.tokens.json`
If nothing detected, offer fallback to init mode.

<!-- USER-EXTENSION-START: export-adapters -->
<!-- USER-EXTENSION-END: export-adapters -->

### Step 5: Export Mode
Load adapters from `Design-System/adapters/export/`:
1. Read current `idpf-design.tokens.json`
2. Run adapter's `translate(dtcgTokens, options)`
3. Write output files and report
### Step 6: Theme Mode
1. Generate `Design-System/themes/<name>.tokens.json`
2. Override base tokens with same `$type` constraints
3. Validate type consistency
4. Check for circular alias references
### Step 7: Validate and Write
1. Scaffold `Design-System/` directory
2. Generate schema via `dtcg-schema.generateSchema()`
3. Validate via `dtcg-schema.validateTokens()`
4. Write via `dtcg-tokens.writeTokenFiles()`
5. Report: files created, token count, validation status
## Directory Structure
```
Design-System/
  ├── idpf-design.tokens.json
  ├── idpf-design.schema.json
  ├── themes/
  ├── adapters/
  │   ├── discovery/
  │   └── export/
```
## Error Handling
| Error | Response |
|-------|----------|
| Invalid token file | Report parse error, offer regenerate |
| Schema validation failure | Report errors with paths, offer fix |
| Adapter crash | Warn, continue with other adapters |
| Unknown DTCG type | Report error with valid type list |
| Circular alias | Report cycle path, block write |
**End of /design-system Command**
