---
version: "v0.77.4"
description: Discover and catalog screen elements from source code (project)
argument-hint: "[#NN]"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /catalog-screens

Discovers and catalogs UI screen elements from application source code, producing structured per-screen specification documents. Fully interactive via `AskUserQuestion`. Element fields defined by shared schema at `.claude/metadata/screen-spec-schema.json`.

**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command catalog-screens`

---

## Prerequisites

- Project contains UI source code (React, Electron, Vue, vanilla HTML, or React Native)
- Shared screen spec schema: `.claude/metadata/screen-spec-schema.json`

---

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `#NN` | No | Issue number for context and writeback of screen spec references |

```
/catalog-screens            # Fully interactive, no issue context
/catalog-screens #42        # Interactive with issue #42 context (enables writeback)
```

All former arguments (screen names, `--scope`, `--update`) incorporated into interactive question flow.

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

**Step 1a: Load Shared Schema and Issue Context**

Read `.claude/metadata/screen-spec-schema.json` for element field definitions. Do not define fields inline.

**If `#NN` provided:**
1. `gh issue view #NN --json body,title,labels`
2. Extract: issue type (from labels), screen/feature names from title/body
3. Hold for Q2 name suggestion and Step 7 writeback

**Step 1b: Discover Existing Content**

Before asking questions, scan `Mockups/` and subdirectories:
- List all `Mockups/{Name}/` directories
- Check `Specs/` for existing screen spec files with last-updated dates and element counts

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

**Q3a** (if specific dir): Free text path. Validate existence. Not found: report error with path suggestions → re-ask Q3a.

**Q4** (after scan): **Which screens?** `AskUserQuestion` `multiSelect: true`:
- Discovered screens with element counts (e.g., "Login — 5 elements (2 inputs, 2 buttons, 1 link)")
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
| React / Next.js | `.jsx`/`.tsx`, React imports, form libs (Formik, React Hook Form) | Parse JSX for props, elements, handlers; traverse component hierarchy |
| Electron | `BrowserWindow`, IPC-bound forms | Identify views, parse IPC bindings, extract renderer elements |
| Vue | `.vue` files, `<template>` blocks | Parse templates, extract `v-model`, `v-if`/`v-show` |
| Vanilla HTML | `.html` with `<form>`, `<input>`, `<select>` | Parse elements, extract `name`, `type`, `required`, `pattern` |
| React Native | RN imports, `NavigationContainer` | Identify screens via navigation, extract `TextInput`, `Picker`, `Switch` |
| Mockup files | `.md`/`.txt` with ASCII art, `.html` with Tailwind/form elements | Parse ASCII art for labeled elements; parse HTML for form elements |

**Consistent output:** All specs use fields from `.claude/metadata/screen-spec-schema.json`. `framework` metadata field records detection strategy (`mockup-ascii` or `mockup-html` for mockup sources).

**Conditionally rendered elements:** Populate `conditionalRender` field.

**Deeply nested components:** Flatten into single element table; note parent in `componentRef`.

**Abstraction-layer tracing (CRITICAL):** Follow delegation chains to actual DOM-producing code. Classify based on rendering implementation, not wrapper API. A bridge exposing 3 methods may render a single `<select>` with 12+ options — the dropdown is the element, not the 3 methods. Trust implementation over wrapper when they disagree. Record delegation source in `componentRef`, library wrapper in `libraryComponent`.

**Circular dependencies:** Document with `(circular)` warning in `dependencies` field. Do not fail.

#### Framework Edge Cases

**If no UI framework detected:** Check for mockup files first:
```bash
find $SCAN_DIR -maxdepth 2 \( -name "*.md" -o -name "*.txt" \) -exec grep -l '[+|┌┐└┘├┤─│\[___\]]' {} \;
find $SCAN_DIR -maxdepth 2 -name "*.html" -exec grep -l 'tailwindcss\|<form\|<input\|<button\|<select' {} \;
```
**If mockup files found:** `AskUserQuestion`: "Extract screen specs from mockups?" — Yes / No, enter manually / Cancel.

**If no mockup files found:**
```
No recognized UI framework found in project source.
Suggestions:
  - Re-run and choose "Scan a specific directory" to target a subdirectory
  - Verify the project contains UI code
  - Choose "Enter screen details manually" for non-standard frameworks
```
→ **STOP**

**If multiple frameworks detected:** Apply all strategies. Report: `"Detected frameworks: {list}"`

**If source files are unparseable:** `"Cannot parse source files in {path} — falling back to manual catalog mode."` → Switch to Step 3b.

### Step 3: Screen Discovery

**Step 3a: Automated Discovery**

Extract per screen: name, all interactive elements, all discoverable schema fields (core + convention), screen-level `libraries`.

**Delegation chain verification:** Trace to actual rendering code before classifying. The wrapper API shape may not match the rendered UI — read the delegated implementation to determine DOM elements created, count, and conditionality.

Present screens via Q4.

**Screen not found:** Report with fuzzy suggestions → continue with others.

**No screens (CLI/API project):**
```
No screens found in project source.
This appears to be a CLI-only or API-only project.
Re-run and choose "Scan a specific directory" if UI code is in a specific directory.
```
→ **STOP**

**Step 3b: Manual Catalog Mode (Fallback)**

When automated fails or user selects "Enter screen details manually" in Q3:
1. Ask screen name
2. Prompt per-element for all schema fields (core required, convention optional)
3. Build spec from input

<!-- USER-EXTENSION-START: post-discovery -->
<!-- USER-EXTENSION-END: post-discovery -->

### Step 4: Element Specification and Enrichment

Build per-element spec using fields from `.claude/metadata/screen-spec-schema.json`. Discovery fills what it can from source. Present via Q5 for user enrichment of unknown fields.

### Step 5: Incremental Update (Q1 "Update"/"Re-scan" flow)

1. Read existing specs from `Mockups/{Name}/Specs/` (selected via Q2, filtered via Q6)
2. Re-scan source for current elements
3. Diff against existing spec:
   - **New elements:** Append to spec
   - **Removed elements:** Mark `(source removed)`, preserve user-enriched data, flag for review
   - **Changed elements:** Report changes, preserve user-enriched fields
   - **Orphaned specs:** Detect renamed source via fuzzy matching, suggest rename via `AskUserQuestion`
   - **Deleted source components:** Mark all elements `(source removed)`, preserve user data, flag: `"Source component deleted — spec preserved for review"`
4. Present changes for confirmation before writing

**Never silently overwrite user-enriched data.**

### Step 6: Write Screen Specs

**Collision protection:** If target exists, `AskUserQuestion`: overwrite / alternative name (e.g., `{Screen-Name}-v2.md`) / skip.

Create `Mockups/{Name}/Specs/` if missing. Write: `Mockups/{Name}/Specs/{Screen-Name}.md`

**Format:**
```markdown
# Screen: {Screen Name}

**Source:** {source file path}
**Framework:** {detected framework}
**Route:** {route/URL pattern, if applicable}
**Last Updated:** {YYYY-MM-DD}
**Elements:** {count}
**Parent Screen:** {parent screen name, or "none"}
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
- cssSelector: `{value}`
- stateBinding: `{value}`

---

## Related Artifacts

- **Mockup:** (none — run `/mockups` to create)

---

*Cataloged {YYYY-MM-DD} by /catalog-screens*
```

<!-- USER-EXTENSION-START: post-catalog -->
<!-- USER-EXTENSION-END: post-catalog -->

### Step 7: Issue Writeback (if applicable)

If `#NN` provided, write screen spec references back to the source.

**Proposal:** Append/update `## Screen Specs` section in proposal document with file references and element counts. Invalid path → warn, skip writeback, spec still created.

**Enhancement or Bug:** Update issue body. Append/update `## Screen Specs` section (replace if exists):
```bash
gh pmu view #NN --body-stdout > .tmp-#NN.md
# Update ## Screen Specs section
gh pmu edit #NN -F .tmp-#NN.md && rm .tmp-#NN.md
```

**No `#NN`:** Skip writeback.

### Step 8: Report

```
Screen Catalog complete.
  Screens cataloged: N
  Total elements: M
  Output: Mockups/{Name}/Specs/{names...}.md
  Next: Run /mockups to create visual mockups.
```

### Step 9: Commit Offer

If any files were created or modified, offer to stage and commit.

`AskUserQuestion`: "Stage and commit catalog changes?" — **Yes** / **No**

**If Yes:**
```bash
git add Mockups/{Name}/Specs/
git commit -m "Refs #NN -- Catalog screen elements for {Name}"
```
- With `#NN`: use `Refs #NN` (cataloging does not close issues)
- Without: descriptive message without issue reference

**If No:** Skip — do not stage or commit.

**STOP.** Do not proceed without user instruction.

---

## Error Handling

| Situation | Response |
|-----------|----------|
| No UI framework detected | Report failure with suggestions → STOP |
| Scan directory not found | Error with path suggestion → re-ask Q3a |
| Screen not found in source | Fuzzy suggestions → continue with others |
| CLI/API project, no screens | Report → STOP |
| Unparseable source files | Manual catalog mode |
| No existing specs (Update flow) | Skip Q6, redirect to Create flow |
| `#NN` issue not found | Continue without issue context |
| Writeback path invalid (proposal) | Warn, skip writeback, spec still created |
| Writeback update fails (enhancement/bug) | Warn, skip writeback, spec still created |
| `Mockups/{Name}/Specs/` missing | Create directory automatically |
| File collision on write | Ask: overwrite, alternative name, skip |
| Schema file missing | "Shared schema not found at .claude/metadata/screen-spec-schema.json" → STOP |

---

**End of /catalog-screens Command**
