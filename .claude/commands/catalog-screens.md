---
version: "v0.70.0"
description: Discover and catalog screen elements from source code (project)
argument-hint: ""
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /catalog-screens
Discovers and catalogs UI screen elements from source code. Fully interactive via `AskUserQuestion`. Element fields defined by shared schema at `.claude/metadata/screen-spec-schema.json`.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command catalog-screens`
---
## Prerequisites
- Project contains UI source code (React, Electron, Vue, vanilla HTML, or React Native)
- Shared screen spec schema: `.claude/metadata/screen-spec-schema.json`
---
## Arguments
Zero arguments. All former arguments (screen names, `--scope`, `--update`) incorporated into interactive question flow.
```
/catalog-screens
```
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction
---
## Workflow

<!-- USER-EXTENSION-START: pre-catalog -->
<!-- USER-EXTENSION-END: pre-catalog -->

### Step 1: Discovery and Interactive Setup
**Step 1a: Load Shared Schema**
Read `.claude/metadata/screen-spec-schema.json` for element field definitions. Do not define fields inline.
**Step 1b: Discover Existing Content**
Before asking questions, scan `Mockups/` and subdirectories:
- List all `Mockups/{Name}/` directories
- Check `Specs/` for existing screen spec files
- Note last-updated dates and element counts
**Step 1c: Interactive Question Flow**
**Q1: What would you like to do?** `AskUserQuestion`:
- "Create new screen specs"
- "Update existing screen specs"
- "Re-scan source for changes"
**Condition:** If no existing specs found, skip Q1 and default to "Create new screen specs".
**Q2: Which mockup set?** `AskUserQuestion`:
- List existing `Mockups/{Name}/` directories
- "Create a new mockup set"
**Q2a** (if new): Free text name → creates `Mockups/{Name}/Specs/`.
**Q3: How should screens be discovered?** `AskUserQuestion`:
- "Scan source code automatically"
- "Scan a specific directory"
- "Enter screen details manually"
**Condition:** Only for "Create new" flow. Update/Re-scan skips to Q6.
**Q3a** (if specific dir): Free text path. Validate existence.
**Q4** (after scan): **Which screens?** `AskUserQuestion` `multiSelect: true`:
- Discovered screens with element counts
- "All of the above"
**Q5** (per screen): **Review elements?** `AskUserQuestion`:
- "Looks good, save as-is"
- "I'd like to add or correct details"
- "Skip this screen"
If "add or correct": prompt per-element for missing schema fields.
**Q6** (Update/Re-scan flow): **Which specs?** `AskUserQuestion` `multiSelect: true`:
- Existing specs with last-updated dates
- "All specs in this set"
### Step 2: Framework Detection
| Framework | Detection Signals | Discovery Approach |
|-----------|-------------------|--------------------|
| React / Next.js | `.jsx`/`.tsx`, React imports, JSX, form libs | Parse JSX for props, elements, handlers; traverse component hierarchy |
| Electron | `BrowserWindow`, electron main, IPC-bound forms | Identify views, parse IPC bindings, extract renderer elements |
| Vue | `.vue` files, `<template>` blocks | Parse templates, extract `v-model`, `v-if`/`v-show` |
| Vanilla HTML | `.html` with `<form>`, `<input>`, `<select>` | Parse elements, extract `name`, `type`, `required`, `pattern` |
| React Native | RN imports, `NavigationContainer` | Identify screens via navigation, extract `TextInput`, `Picker`, `Switch` |
**Consistent output:** All specs use fields from `.claude/metadata/screen-spec-schema.json`.
**Conditionally rendered elements:** Populate `conditionalRender` field.
**Deeply nested components:** Flatten into single table, note parent in `componentRef`.
**Abstraction-layer tracing (CRITICAL):** Follow delegation chains to actual DOM-producing code. Trust implementation over wrapper API. Record in `componentRef` and `libraryComponent`.
**Circular dependencies:** Document with `(circular)` warning. Do not fail.
**No UI framework detected:** Report suggestions → **STOP**
**Multiple frameworks:** Apply all strategies. Report detected list.
**Unparseable source:** Fall back to Manual Catalog Mode (Step 3b).
### Step 3: Screen Discovery
**Step 3a: Automated Discovery**
Extract per screen: name, elements, all discoverable schema fields (core + convention), screen-level `libraries`.
**Delegation chain verification:** Trace to actual rendering code before classifying.
Present screens via Q4.
**Screen not found:** Fuzzy suggestions.
**No screens (CLI/API):** Report → **STOP**
**Step 3b: Manual Catalog Mode (Fallback)**
When automated fails or user selects "Enter screen details manually" in Q3:
1. Ask screen name
2. Prompt per-element for all schema fields (core required, convention optional)
3. Build spec from input

<!-- USER-EXTENSION-START: post-discovery -->
<!-- USER-EXTENSION-END: post-discovery -->

### Step 4: Element Specification and Enrichment
Build per-element spec using fields from `.claude/metadata/screen-spec-schema.json`.
Discovery fills what it can. Present via Q5 for enrichment.
### Step 5: Incremental Update (Q1 "Update"/"Re-scan" flow)
1. Read existing specs from `Mockups/{Name}/Specs/` (Q2 + Q6)
2. Re-scan source
3. Diff: new (append), removed (mark `(source removed)`, preserve user data), changed (preserve user fields), orphaned (fuzzy match, suggest rename), deleted source (preserve, flag)
4. Present changes for confirmation
**Never silently overwrite user-enriched data.**
### Step 6: Write Screen Specs
**Collision protection:** If target exists, `AskUserQuestion`: overwrite / alternative name / skip.
Create `Mockups/{Name}/Specs/` if missing. Write: `Mockups/{Name}/Specs/{Screen-Name}.md`
**Format:**
```markdown
# Screen: {Screen Name}
**Source:** {path}
**Framework:** {framework}
**Route:** {route/URL}
**Last Updated:** {YYYY-MM-DD}
**Elements:** {count}
**Parent Screen:** {parent or "none"}
**Authentication:** {none|required|optional}
**Libraries:** {component: [...], css: [...], form: [...], animation: [...], icon: [...]}
---
## Elements
| Element ID | Type | Label | Default Value | Valid Input | Input Range | Required | Validation Message | Dependencies | Notes |
|------------|------|-------|---------------|-------------|-------------|----------|--------------------|--------------|-------|
### Convention Fields (per element, when discovered)
**{elementId}:**
- dataTestId: `{value}`
- ariaLabel: `{value}`
---
## Related Artifacts
- **Mockup:** (none — run `/mockups` to create)
---
*Cataloged {YYYY-MM-DD} by /catalog-screens*
```

<!-- USER-EXTENSION-START: post-catalog -->
<!-- USER-EXTENSION-END: post-catalog -->

### Step 7: Proposal Writeback (if applicable)
If triggered from proposal context: append `## Screen Specs` with `Mockups/{Name}/Specs/` references.
Invalid path → warn, skip writeback, spec still created.
### Step 8: Report
```
Screen Catalog complete.
  Screens cataloged: N
  Total elements: M
  Output: Mockups/{Name}/Specs/{names...}.md
  Next: Run /mockups to create visual mockups.
```
**STOP.** Do not proceed without user instruction.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| No UI framework | Suggestions → STOP |
| Scan dir not found | Suggestion → re-ask Q3a |
| Screen not found | Fuzzy suggestions → continue |
| CLI/API project | Report → STOP |
| Unparseable source | Manual catalog mode |
| No specs (Update) | Redirect to Create |
| Proposal path invalid | Warn, skip writeback |
| Dirs missing | Create automatically |
| File collision | Ask: overwrite, alternative, skip |
| Schema missing | STOP |
---
**End of /catalog-screens Command**
