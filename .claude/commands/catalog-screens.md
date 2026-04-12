---
version: "v0.85.0"
description: Discover and catalog screen elements from source code (project)
argument-hint: "[#NN]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /catalog-screens
Discovers and catalogs UI screen elements from source code, producing structured per-screen spec documents. Fully interactive — uses `AskUserQuestion`. All element fields defined by `.claude/metadata/screen-spec-schema.json`.

**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command catalog-screens`
## Prerequisites
- Project contains UI source code (React, Electron, Vue, vanilla HTML, React Native)
- Shared schema: `.claude/metadata/screen-spec-schema.json`
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#NN` | No | Issue number (enhancement, bug, proposal). Reads body for context, enables writeback. |

```
/catalog-screens            # Interactive, no issue context
/catalog-screens #42        # With issue #42 context (enables writeback)
```
Former arguments (screen names, `--scope`, `--update`) are incorporated into the interactive flow.
## Execution Instructions
**REQUIRED:**
1. **Generate Todo List:** `TodoWrite` from workflow steps
2. **Include Extensions:** Add todo per non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark `in_progress` → `completed`
4. **Post-Compaction:** Re-read spec and regenerate todos
## Workflow

<!-- USER-EXTENSION-START: pre-catalog -->
<!-- USER-EXTENSION-END: pre-catalog -->

### Step 1: Discovery and Interactive Setup
**Step 1a: Load Schema and Issue Context**
Read `.claude/metadata/screen-spec-schema.json` — source of truth for element fields, do not define inline.

**If `#NN` provided:**
1. Read issue body:
```bash
gh issue view #NN --json body,title,labels
```
2. Extract: issue type (from labels), screen/feature names from title/body
3. Hold context for Q2 name suggestion and Step 7 writeback

**Step 1b: Discover Existing Content**
Scan `Mockups/` and subdirectories:
- List all `Mockups/{Name}/` directories
- For each, check `Specs/` for existing files
- Note last-updated dates and element counts

**Step 1c: Interactive Question Flow**
#### Mode Selection (Q1–Q3)
**Q1: What would you like to do?**
- "Create new screen specs"
- "Update existing screen specs"
- "Re-scan source for changes"

**Condition:** If no existing specs found, skip Q1 and default to "Create new".

**Q2: Which mockup set should these specs belong to?**
- List each existing `Mockups/{Name}/` directory
- "Create a new mockup set"

**Q2a** (if "Create new mockup set"): Ask for name. Used to create `Mockups/{Name}/Specs/`.

**Q3: How should screens be discovered?**
- "Scan source code automatically"
- "Scan a specific directory"
- "Enter screen details manually"

**Condition:** Only for "Create new" flow. For Update/Re-scan, skip to Q6.

**Q3a** (if "Scan a specific directory"): Ask for path. Validate existence.
- **Path does not exist:**
  ```
  Directory not found: {path}
  Did you mean one of these?
    - {similar-path-1}
    - {similar-path-2}
  ```
  → Re-ask Q3a
#### Screen Selection and Review (Q4–Q6)
**Q4** (after automated scan): **Which screens should be cataloged?**
`AskUserQuestion` with `multiSelect: true`:
- List discovered screens with element counts (e.g., "Login — 5 elements (2 inputs, 2 buttons, 1 link)")
- "All of the above"

**Q5** (per screen, after extraction): **Review elements for {Screen}?**
- "Looks good, save as-is"
- "I'd like to add or correct details"
- "Skip this screen"

If "add or correct": prompt per-element for missing fields from schema.

**Q6** (Update/Re-scan flow): **Which specs should be updated?**
`multiSelect: true`:
- List existing specs with last-updated dates
- "All specs in this set"
### Step 2: Framework Detection
Scan project source to identify UI framework(s).

| Framework | Detection Signals | Discovery Approach |
|-----------|-------------------|--------------------|
| React / Next.js | `.jsx`/`.tsx`, React imports, JSX, form libraries (Formik, React Hook Form) | Parse JSX for props, form elements, event handlers; traverse component hierarchy |
| Electron | `BrowserWindow`, main process, IPC-bound forms | Identify BrowserWindow views, parse IPC channel bindings, extract form elements from renderer HTML/JSX |
| Vue | `.vue` files, single-file components, `<template>` blocks | Parse `<template>` for form elements, `v-model` bindings, `v-if`/`v-show` |
| Vanilla HTML | `.html` with `<form>`, `<input>`, `<select>` | Parse HTML form elements, extract `name`, `type`, `required`, `pattern` attributes |
| React Native | RN imports, `NavigationContainer`, screen components | Identify via navigation structure, extract `TextInput`, `Picker`, `Switch` |
| Mockup files | `.md`/`.txt` with ASCII art (box-drawing, `[ ]` inputs), `.html` with Tailwind/CSS + form elements | Parse ASCII art for labeled elements (text inputs `[___]`, buttons `[ Label ]`, checkboxes `[x]`, dropdowns); parse HTML for `<input>`, `<button>`, `<select>`, `<form>` |

**Consistent output:** Regardless of framework, all specs use fields from `.claude/metadata/screen-spec-schema.json`. The `framework` metadata field records detection strategy (`mockup-ascii` or `mockup-html` for mockup sources).

**Conditionally rendered elements:** Elements rendered via `v-if`, ternary JSX, feature flags MUST populate the `conditionalRender` field.

**Deeply nested components:** Traverse up to 10+ levels. For deep hierarchies, flatten into one element table with parent component noted in `componentRef`.

**Abstraction-layer tracing (CRITICAL):** When a component delegates rendering to another layer (e.g., Svelte bridge calling vanilla JS `addEditOps()`), follow delegation to actual DOM-producing code. Classify elements based on rendering implementation, not wrapper API. A bridge exposing 3 methods may render a single `<select>` with 12+ options — the dropdown is the element, not the 3 methods. When wrapper and implementation disagree, trust the implementation. Record delegation source in `componentRef`, library wrapper in `libraryComponent`.

**Circular element dependencies:** If A depends on B and B depends on A, document both with `(circular)` warning in `dependencies`. Report and continue — do not fail.
#### Framework Edge Cases
**If no UI framework detected:**
Before stopping, check for mockup files:
```bash
find $SCAN_DIR -maxdepth 2 \( -name "*.md" -o -name "*.txt" \) -exec grep -l '[+|┌┐└┘├┤─│\[___\]]' {} \;
find $SCAN_DIR -maxdepth 2 -name "*.html" -exec grep -l 'tailwindcss\|<form\|<input\|<button\|<select' {} \;
```
**If mockup files found:** `AskUserQuestion`:
- "Yes, extract from mockups"
- "No, enter manually"
- "Cancel"

**If no mockup files:**
```
No recognized UI framework found.
Suggestions:
  - Re-run and choose "Scan a specific directory"
  - Verify the project contains UI code
  - Choose "Enter screen details manually"
```
→ **STOP**

**If multiple frameworks detected:** Apply all strategies. Report: `"Detected frameworks: {list}"`

**If source unparseable** (binary, minified, unsupported):
```
Cannot parse source files in {path} — falling back to manual catalog mode.
```
→ Switch to **Manual Catalog Mode** (Step 3b).
### Step 3: Screen Discovery
**Step 3a: Automated Discovery**
For each screen discovered, extract:
- Screen name (from component name, route, or file name)
- All interactive elements (inputs, buttons, selects, checkboxes)
- Element properties from schema:
  - Core fields: type, label, default values, validation, dependencies
  - Convention fields: `dataTestId`, `ariaLabel`, `ariaRole`, `cssSelector`, `placeholder`, `autocomplete`, `eventHandlers`, `stateBinding`, `conditionalRender`, `componentRef`, `libraryComponent`
- Screen-level: `libraries` (component/CSS/form/animation/icon libraries detected)

**Delegation chain verification:** When a component delegates (bridge pattern, imperative rendering, vanilla JS from framework components), trace to actual rendering code before classifying. Wrapper API shape (e.g., 3 methods) may not match rendered UI (e.g., 1 dropdown with 12 options). Read the delegated method to determine:
- DOM elements actually created (`<select>`, `<input>`, `<button>`)
- How many distinct interactive elements rendered
- Whether elements are conditional

Present discovered screens via Q4.

**Screen not found in source:**
```
Screen "{name}" not found.
Did you mean one of these?
  - Foobar (src/views/Foobar.tsx)
  - FooBarSettings (src/views/FooBarSettings.vue)
```

**CLI-only or API-only project:**
```
No screens found in project source.
This appears to be a CLI-only or API-only project.
Re-run and choose "Scan a specific directory" if UI code is in a specific directory.
```
→ **STOP**

**Step 3b: Manual Catalog Mode (Fallback)**
When automated discovery fails or user chose manual:
1. Ask user to name the screen
2. Prompt per-element for schema fields (core required, convention optional)
3. Build spec from user input

Use `AskUserQuestion` per element.

<!-- USER-EXTENSION-START: post-discovery -->
<!-- USER-EXTENSION-END: post-discovery -->

### Step 4: Element Specification and Enrichment
Build per-element spec using fields from `.claude/metadata/screen-spec-schema.json`.

Discovery fills what it can from source. Present for enrichment via Q5:
```
Screen: Login
  Element: username (text-input)
    - label: "Username" (from source)
    - required: true (from validation)
    - dataTestId: "username-input" (from source)
    - defaultValue: (unknown — please specify or leave blank)
    - validationMessage: (unknown — please specify)
```
Use `AskUserQuestion` for fields the agent couldn't determine.
### Step 5: Incremental Update (Q1 "Update" or "Re-scan" flow)
1. **Read existing specs** from `Mockups/{Name}/Specs/` (selected via Q2, filtered via Q6)
2. **Re-scan source** for current elements
3. **Diff** source-discovered against existing:
   - **New elements:** Report additions, append to spec
   - **Removed elements:** Mark as `(source removed)`, preserve all user-enriched data, flag for review
   - **Changed elements:** Report changes, preserve user-enriched fields (defaults, validation rules, notes, populated convention fields)
   - **Orphaned specs:** Screen renamed — detect via fuzzy matching, suggest rename via `AskUserQuestion`
   - **Deleted source components:** Mark all elements as `(source removed)`, preserve user data, flag: `"Source component deleted — spec preserved for review"`
4. **Present changes** for confirmation before writing

**Conflict resolution:** If user edits spec while update runs, re-read before writing.

**Never silently overwrite user-enriched data.**
### Step 6: Write Screen Specs
#### Step 6a: Collision Protection
Check if target file exists.
- **Exists:** `AskUserQuestion`:
  - "Overwrite existing spec"
  - "Save with alternative name" (suggest `{Screen-Name}-v2.md`)
  - "Skip this screen"
- **Does not exist:** Write directly.

Ensure `Mockups/{Name}/Specs/` exists (create if missing, including parents).

Write one file per screen: `Mockups/{Name}/Specs/{Screen-Name}.md`
#### Step 6b: Screen Spec Format
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
| username | text-input | Username | (none) | Non-empty string | {"minLength":3,"maxLength":50} | Yes | "Username is required" | — | — |

### Convention Fields (per element, when discovered)

**username:**
- dataTestId: `username-input`
- ariaLabel: `Enter your username`
- cssSelector: `.login-form input[name="username"]`
- stateBinding: `formik.values.username`

---

## Related Artifacts

- **Mockup:** (none — run `/mockups` to create)

---

*Cataloged {YYYY-MM-DD} by /catalog-screens*
```

<!-- USER-EXTENSION-START: post-catalog -->
<!-- USER-EXTENSION-END: post-catalog -->

### Step 7: Issue Writeback (if applicable)
If triggered with `#NN`, write spec references back:

**Proposal:** Read document, append/update `## Screen Specs` section:
```markdown
## Screen Specs

- `Mockups/{Name}/Specs/{Screen-Name-1}.md` — {element count} elements
- `Mockups/{Name}/Specs/{Screen-Name-2}.md` — {element count} elements
```
If proposal path invalid → warn, skip writeback, spec still created.

**Enhancement or Bug:** Update issue body with same format. Replace contents if section already exists.
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
If any files created/modified, offer to commit.

`AskUserQuestion`:
- Question: "Stage and commit catalog changes?"
- Options:
  - **Yes** — Stage and commit all screen spec artifacts
  - **No** — Skip

**If Yes:**
```bash
git add Mockups/{Name}/Specs/
git commit -m "Refs #NN -- Catalog screen elements for {Name}"
```
- If `#NN` available, use `Refs #NN` (cataloging does not close issues)
- If no issue context, descriptive message without issue: `"Catalog screen elements for {Name}"`

**If No:** Skip — do not stage or commit.

**STOP.** Do not proceed without user instruction.
## Error Handling
| Situation | Response |
|-----------|----------|
| No UI framework detected | Report failure with suggestions → STOP |
| Scan directory not found | Error with path suggestion → re-ask Q3a |
| Screen not found in source | Report with fuzzy suggestions → continue |
| CLI/API project, no screens | Report, suggest re-running → STOP |
| Unparseable source files | Fall back to manual catalog mode |
| No existing specs (Update flow) | Skip Q6, report "No specs found" → redirect to Create |
| `#NN` issue not found | "Issue #NN not found" → continue without context |
| Writeback path invalid (proposal) | Warn, skip writeback, spec still created |
| Writeback update fails (enhancement/bug) | Warn, skip writeback, spec still created |
| `Mockups/{Name}/Specs/` missing | Create directory automatically |
| File collision on write | Ask: overwrite, alternative name, skip |
| Schema file missing | "Shared schema not found at .claude/metadata/screen-spec-schema.json" → STOP |

**End of /catalog-screens Command**
