---
version: "v0.88.0"
description: Produce DTCG-compliant design tokens with pluggable adapter architecture (project)
argument-hint: "[--discover | --export <adapter> | --theme <name>]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /design-system
Produce DTCG-compliant design token file (`Design-System/idpf-design.tokens.json`) with pluggable adapter architecture. Integrates with `/mockups` and `/catalog-screens`.
**Extension Points:** `.claude/metadata/extension-points.json` or `/extensions list --command design-system`
---
## Prerequisites
- Node.js built-ins only (no npm deps)
- DTCG schema: `.claude/scripts/shared/lib/dtcg-schema.js`
- DTCG tokens: `.claude/scripts/shared/lib/dtcg-tokens.js`
---
## Arguments
| Argument | Description |
|----------|-------------|
| *(none)* | Init mode or show status |
| `--discover` | Extract tokens from existing code via adapters |
| `--export <adapter>` | Export tokens to framework-specific format |
| `--export all` | Export to all detected formats |
| `--theme <name>` | Generate/apply theme override |
| `--from-screenshot <path>` | AC31 — extract token candidates (color, typography, spacing, gradient) from a visual reference. Path validated via `validateScreenshotFile` from `.claude/scripts/shared/lib/screenshot-input.js`. Multimodal Read produces candidates → `applyCandidateSelection` from `.claude/scripts/shared/lib/screenshot-token-candidates.js` applies user accept/reject/partial decision before any write (AC32). |
| `--diff` | AC33 — 4-category drift report (additions/removals/mismatches/broken aliases) via each adapter's `diff(projectRoot, currentTokens)` (AC38), aggregated by `buildDriftReport` + rendered by `formatDriftReport` from `.claude/scripts/shared/lib/token-drift-report.js`. No writes. |
| `--discover --diff` | AC34 — combine discovery + diff, then selectively apply via `applySelectedDiff` (AC39). User decline → no writes. |

```
/design-system              # Init or status
/design-system --discover   # Discover from code
/design-system --export css-vars   # CSS custom properties
/design-system --export all        # All detected formats
/design-system --theme dark        # Dark theme override
/design-system --from-screenshot ./design/home.png   # Bootstrap tokens from image
```
---
## Execution
**REQUIRED:**
1. Parse workflow steps → `TaskCreate`
2. Mark tasks `in_progress` → `completed`
---
## Workflow
### Step 1: Check Existing Token File
Invoke `detectMode('Design-System/idpf-design.tokens.json')` from `.claude/scripts/shared/lib/token-update-mode.js` (AC35):
- **`'init'`** (missing): proceed to init (Step 2)
- **`'update'`** (exists + parses): update walkthrough — list groups, selective edit, `mergeUpdate(original, edits)` preserves untouched groups byte-for-byte
- **`'unreadable'`** (parse fails): report parse error, offer `backupAndOffer(path)` to back up to `*.bak.{ts}` and reinit (Exception #7)
### Step 2: Mode Selection
- **No args + no tokens:** Init (Step 3)
- **`--discover`:** Step 4
- **`--export`:** Step 5
- **`--theme`:** Step 6
- **No args + tokens exist:** Show summary + actions
### Step 3: Init Mode (Interactive)
Guided walkthrough (modeled on `/charter`):
1. **Color palette:** primary, secondary, accent, neutral scale, semantic
2. **Typography:** families, size scale, weight scale
3. **Spacing:** base unit + scale
4. **Component patterns:** radii, shadows, transitions

Use `AskUserQuestion` per category. Optional skip: shadows, transitions.
Output: valid DTCG `idpf-design.tokens.json` + `idpf-design.schema.json`.

<!-- USER-EXTENSION-START: discovery-adapters -->
<!-- USER-EXTENSION-END: discovery-adapters -->

### Step 4: Discovery Mode
Load adapters from `Design-System/adapters/discovery/`:
1. Run each `detect(projectRoot)`
2. For `true`, run `extract(projectRoot)`
3. Merge discovered tokens
4. Present for confirmation/refinement
5. Write to `idpf-design.tokens.json`

No adapters detect → offer fallback to init.

<!-- USER-EXTENSION-START: export-adapters -->
<!-- USER-EXTENSION-END: export-adapters -->

### Step 5: Export Mode
Load `Design-System/adapters/export/`:
1. Read `idpf-design.tokens.json`
2. Run adapter `translate(dtcgTokens, options)`
3. Write outputs
4. Report generated files
### Step 6: Theme Mode
1. Generate `Design-System/themes/<name>.tokens.json`
2. Theme overrides base tokens with same `$type`
3. Validate type consistency
4. Check circular alias references
### Step 7: Validate and Write
1. Scaffold `Design-System/` structure
2. `dtcg-schema.generateSchema()`
3. `dtcg-schema.validateTokens()`
4. `dtcg-tokens.writeTokenFiles()`
5. Report: files created, token count, validation status
### Step 8: Token Change Propagation (AC36, AC37)
After write in update mode:
1. Compute `changedKeys` = paths whose `$value` differs between original and merged token tree.
2. `loadCatalog('Mockups/screen-catalog.json')` (from `.claude/scripts/shared/lib/screen-catalog.js`).
3. `findAffectedMockups(catalog, changedKeys)` from `.claude/scripts/shared/lib/token-propagation.js` — returns screens whose `tokenDependencies` (AC37) overlap.
4. Empty → "No affected mockups."
5. Non-empty → `AskUserQuestion` yes/no/select → `applyPropagationDecision` → regenerate via `/mockups` for each chosen screen (mockups remain stale on `no`).
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
| Invalid token file (parse) | Specific JSON error, offer regenerate |
| Schema validation failure | Errors with paths, offer fix |
| Adapter crash | Warn, continue with others |
| Unknown DTCG type | Error with valid type list |
| Circular alias | Report cycle path, block write |
---
**End of /design-system Command**
