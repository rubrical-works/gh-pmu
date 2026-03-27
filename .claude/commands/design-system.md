---
version: "v0.74.0"
description: Produce DTCG-compliant design tokens with pluggable adapter architecture (project)
argument-hint: "[--discover | --export <adapter> | --theme <name>]"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /design-system
Produces a DTCG-compliant design token file (`Design-System/idpf-design.tokens.json`) with pluggable adapter architecture for extensible discovery and export. Integrates with `/mockups` and `/catalog-screens`.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command design-system`
---
## Prerequisites
- Node.js built-in modules only (no npm dependencies)
- DTCG schema library: `.claude/scripts/shared/lib/dtcg-schema.js`
- DTCG token library: `.claude/scripts/shared/lib/dtcg-tokens.js`
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| *(none)* | No | Init mode — guided walkthrough to create design tokens |
| `--discover` | No | Discovery mode — extract tokens from existing code via adapters |
| `--export <adapter>` | No | Export tokens to framework-specific format |
| `--export all` | No | Export to all detected framework formats |
| `--theme <name>` | No | Generate or apply a theme override file |
```
/design-system              # Init mode (no existing tokens) or show status
/design-system --discover   # Discover tokens from existing project code
/design-system --export css-vars   # Export tokens as CSS custom properties
/design-system --export all        # Export to all detected formats
/design-system --theme dark        # Generate dark theme override
```
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TaskCreate` to create tasks
2. **Track Progress:** Mark each task `in_progress` → `completed` as work proceeds
---
## Workflow
### Step 1: Check for Existing Token File
Check if `Design-System/idpf-design.tokens.json` exists:
- **Exists:** Load and validate against schema. Report status.
- **Does not exist:** Proceed to init mode (Step 2).
### Step 2: Mode Selection
Based on arguments and state:
- **No arguments + no tokens:** Init mode (Step 3)
- **`--discover`:** Discovery mode (Step 4)
- **`--export`:** Export mode (Step 5)
- **`--theme`:** Theme mode (Step 6)
- **No arguments + tokens exist:** Show token summary and available actions
### Step 3: Init Mode (Interactive Walkthrough)
Guided walkthrough modeled after `/charter`:
1. **Color palette:** primary, secondary, accent, neutral scale, semantic colors
2. **Typography:** font families, size scale, weight scale
3. **Spacing:** base unit and scale
4. **Component patterns:** border radii, shadows, transitions
Use `AskUserQuestion` for each category. User can skip optional categories (shadows, transitions).
Output: valid DTCG `idpf-design.tokens.json` + `idpf-design.schema.json`.

<!-- USER-EXTENSION-START: discovery-adapters -->
<!-- USER-EXTENSION-END: discovery-adapters -->

### Step 4: Discovery Mode
Load adapters from `Design-System/adapters/discovery/`:
1. Run each adapter's `detect(projectRoot)` function
2. For adapters returning `true`, run `extract(projectRoot)`
3. Merge discovered tokens
4. Present to user for confirmation/refinement
5. Write confirmed tokens to `idpf-design.tokens.json`
If no adapters detect anything, offer fallback to init mode.

<!-- USER-EXTENSION-START: export-adapters -->
<!-- USER-EXTENSION-END: export-adapters -->

### Step 5: Export Mode
Load adapters from `Design-System/adapters/export/`:
1. Read current `idpf-design.tokens.json`
2. Run specified adapter's `translate(dtcgTokens, options)`
3. Write output files
4. Report generated files
### Step 6: Theme Mode
1. Generate theme file at `Design-System/themes/<name>.tokens.json`
2. Theme overrides base tokens with same `$type` constraints
3. Validate type consistency with base tokens
4. Check for circular alias references
### Step 7: Validate and Write
1. Scaffold `Design-System/` directory structure
2. Generate schema via `dtcg-schema.generateSchema()`
3. Validate tokens via `dtcg-schema.validateTokens()`
4. Write files via `dtcg-tokens.writeTokenFiles()`
5. Report: files created, token count, validation status
---
## Directory Structure
```
Design-System/
  ├── idpf-design.tokens.json      (main DTCG token file)
  ├── idpf-design.schema.json      (generated JSON Schema)
  ├── themes/                       (theme override files)
  ├── adapters/
  │   ├── discovery/                (discovery adapters)
  │   └── export/                   (export adapters)
```
---
## Error Handling
| Error | Response |
|-------|----------|
| Invalid token file (parse error) | Report specific JSON parse error, offer to regenerate |
| Schema validation failure | Report errors with paths, offer to fix |
| Adapter crash | Warn about broken adapter, continue with other adapters |
| Unknown DTCG type | Report error with valid type list |
| Circular alias | Report cycle path, block write |
---
**End of /design-system Command**
