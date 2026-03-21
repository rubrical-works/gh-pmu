---
version: "v0.67.1"
description: Discover and catalog screen elements from source code (project)
argument-hint: "[screen-name...] [--scope <path>] [--update]"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /catalog-screens

Discovers and catalogs UI screen elements from application source code, producing structured per-screen specification documents. The agent reads source files to identify screens and their elements, then presents findings to the user for confirmation and enrichment with behavioral constraints.

**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command catalog-screens`

---

## Prerequisites

- Project contains UI source code (React, Electron, Vue, vanilla HTML, or React Native)
- `Screen-Specs/` directory at project root (created automatically if missing)

---

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| *(none)* | No | Full discovery — scan all source code for screens |
| `<screen-name>` | No | Catalog a single named screen (e.g., `Login`) |
| `<name1> <name2>` | No | Catalog multiple named screens in one invocation |
| `--scope <path>` | No | Narrow discovery to a specific directory or component subtree |
| `--update` | No | Re-scan source against existing specs; preserve user-enriched fields and report changes |

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

### Step 1: Parse Arguments and Validate

Extract arguments: screen names, `--scope`, `--update`.

**If `--scope` specified:** Validate the path exists relative to project root.
- **Path does not exist:** Report error with path suggestion:
  ```
  Directory not found: {path}
  Did you mean one of these?
    - {similar-path-1}
    - {similar-path-2}
  ```
  → **STOP**

**If `--update` specified:** Verify `Screen-Specs/` directory exists with at least one spec file.
- **No existing specs:** `"No existing screen specs found in Screen-Specs/. Run /catalog-screens first to create initial specs."` → **STOP**

### Step 2: Framework Detection

Scan project source files to identify the UI framework(s) in use.

**Detection strategies:**

| Framework | Detection Signals | Discovery Approach |
|-----------|-------------------|--------------------|
| React / Next.js | `.jsx`/`.tsx` files, React imports, JSX syntax, form libraries (Formik, React Hook Form) | Parse JSX components for props, form elements, event handlers; traverse component hierarchy for nested screens |
| Electron | `BrowserWindow` usage, electron main process, IPC-bound forms | Identify BrowserWindow views, parse IPC channel bindings, extract form elements from renderer process HTML/JSX |
| Vue | `.vue` files, single-file components, `<template>` blocks | Parse `<template>` blocks for form elements, extract `v-model` bindings, `v-if`/`v-show` conditions |
| Vanilla HTML | `.html` files with `<form>`, `<input>`, `<select>` elements | Parse HTML form elements, extract `name`, `type`, `required`, `pattern` attributes directly |
| React Native | React Native imports, `NavigationContainer`, screen components | Identify screen components via navigation structure, extract `TextInput`, `Picker`, `Switch` elements |

**Consistent output:** Regardless of source framework, all screen specs use the same 10-field element table format. The `**Framework:**` metadata field in the output records which detection strategy was used.

**Conditionally rendered elements:** Elements rendered conditionally (`v-if`, ternary JSX, feature flags) must note their rendering conditions in the Notes field of the element table.

**Deeply nested components:** Traverse component trees up to 10+ levels deep. For deeply nested element hierarchies, flatten into a single element table with parent component noted in the Notes field.

**Circular element dependencies:** If element A depends on element B and B depends on A, document both dependencies with a `(circular)` warning in the Dependencies field. Do not fail — report the cycle and continue.

**If no UI framework detected:**
```
No recognized UI framework found in project source.
Suggestions:
  - Check --scope to target a specific directory
  - Verify the project contains UI code
  - For non-standard frameworks, use manual cataloging mode
```
→ **STOP**

**If multiple frameworks detected:** Apply all relevant discovery strategies. Report: `"Detected frameworks: {list}"`

**If source files are unparseable** (binary, minified, unsupported language):
```
Cannot parse source files in {path} — falling back to manual catalog mode.
```
→ Switch to **Manual Catalog Mode** (Step 3b).

### Step 3: Screen Discovery

**Step 3a: Automated Discovery**

Based on the detected framework, scan source code to identify screens and their elements.

**For each screen discovered, extract:**
- Screen name (from component name, route, or file name)
- All interactive elements (inputs, buttons, selects, checkboxes, etc.)
- Element properties discoverable from source (type, label, default values, validation)

**No arguments (full discovery):** Scan all source code within scope. Present all discovered screens to the user:
```
Discovered N screens:
  1. Login — 5 elements (2 inputs, 2 buttons, 1 link)
  2. Dashboard — 12 elements (3 charts, 4 buttons, 5 nav items)
  ...

Confirm these screens? Select which to catalog:
```
Use `AskUserQuestion` with `multiSelect: true` to let user confirm/deselect screens.

**Named screen(s):** Search for the named screen(s) in source code.
- **Screen found:** Proceed to element extraction.
- **Screen not found:** Report with fuzzy suggestions:
  ```
  Screen "FooBar" not found in source.
  Did you mean one of these?
    - Foobar (src/views/Foobar.tsx)
    - FooBarSettings (src/views/FooBarSettings.vue)
  ```
  → **STOP** (for that screen; continue with others if multi-screen)

**CLI-only or API-only project (no screens found):**
```
No screens found in project source.
This appears to be a CLI-only or API-only project.
Check --scope if UI code is in a specific directory.
```
→ **STOP**

**Step 3b: Manual Catalog Mode (Fallback)**

When automated discovery fails (unparseable source, unsupported framework):

1. Ask user to name the screen
2. For each element, prompt user for the 10 specification fields
3. Build the spec from user input

Use `AskUserQuestion` for each element to gather field values.

<!-- USER-EXTENSION-START: post-discovery -->
<!-- USER-EXTENSION-END: post-discovery -->

### Step 4: Element Specification

For each confirmed screen, build the per-element specification table.

**Per-element fields (10 required):**

| Field | Description |
|-------|-------------|
| Element ID/Name | Unique identifier for the element within the screen |
| Type | Element type (text input, dropdown, checkbox, button, link, etc.) |
| Label | User-facing label text |
| Default Value | Initial/default value (if any) |
| Valid Input | Description of valid input values or formats |
| Input Range | Minimum/maximum values, length constraints |
| Required | Yes/No — whether the element is required |
| Validation Message | Error message shown on invalid input |
| Dependencies | Other elements this element depends on (e.g., "Visible when Country = US") |
| Notes | Additional behavioral constraints, conditional rendering, etc. |

**Discovery fills what it can from source code.** Present to user for enrichment:
```
Screen: Login
  Element: username (text input)
    - Label: "Username" (from source)
    - Required: Yes (from validation)
    - Default Value: (unknown — please specify or leave blank)
    - Validation Message: (unknown — please specify)
    ...
```

Use `AskUserQuestion` to let user enrich fields the agent couldn't determine.

### Step 5: Incremental Update (`--update` mode)

When `--update` is specified:

1. **Read existing specs** from `Screen-Specs/`
2. **Re-scan source** for current elements
3. **Diff** source-discovered elements against existing spec:
   - **New elements:** Report additions, append to spec
   - **Removed elements:** Mark as `(source removed)` in spec, preserve all user-enriched data, flag for review
   - **Changed elements:** Report changes, preserve user-enriched fields (defaults, validation rules, notes)
   - **Orphaned specs:** Screen renamed in source but spec has old name — detect via fuzzy matching (similar element sets, file path patterns), suggest rename: `"Screen-Specs/OldName.md appears orphaned — source component renamed to NewName. Rename spec? (y/n)"`
   - **Deleted source components:** If a source component is deleted entirely, mark all its elements as `(source removed)` in the spec, preserve all user data, and flag: `"Source component deleted — spec preserved for review"`
4. **Present changes** to user for confirmation before writing

**Conflict resolution:** If user edits spec while `--update` runs, re-read the spec file before writing.

**Never silently overwrite user-enriched data.**

### Step 6: Write Screen Specs

Ensure `Screen-Specs/` directory exists (create if missing).

Write one file per screen: `Screen-Specs/{Screen-Name}.md`

**Screen spec format:**

```markdown
# Screen: {Screen Name}

**Source:** {source file path}
**Framework:** {detected framework}
**Last Updated:** {YYYY-MM-DD}
**Elements:** {count}

---

## Elements

| Element ID/Name | Type | Label | Default Value | Valid Input | Input Range | Required | Validation Message | Dependencies | Notes |
|-----------------|------|-------|---------------|-------------|-------------|----------|--------------------|--------------|-------|
| username | text input | Username | (none) | Non-empty string | 3-50 chars | Yes | "Username is required" | — | — |
| password | password | Password | (none) | Min 8 chars, 1 uppercase | 8-128 chars | Yes | "Password must be at least 8 characters" | — | — |

---

## Related Artifacts

- **Mockup:** (none — run `/mockups {Screen-Name}` to create)

---

*Cataloged {YYYY-MM-DD} by /catalog-screens*
```

<!-- USER-EXTENSION-START: post-catalog -->
<!-- USER-EXTENSION-END: post-catalog -->

### Step 7: Proposal Writeback (if applicable)

If `/catalog-screens` was triggered from a proposal context (proposal file path available):

1. Read the proposal document
2. Append or update `## Screen Specs` section with file references:
   ```markdown
   ## Screen Specs

   - `Screen-Specs/{Screen-Name-1}.md` — {element count} elements
   - `Screen-Specs/{Screen-Name-2}.md` — {element count} elements
   ```
3. If proposal file path is invalid or deleted → warn, skip writeback, screen spec still created

### Step 8: Report

```
Screen Catalog complete.
  Screens cataloged: N
  Total elements: M
  Output: Screen-Specs/{names...}.md

  Next: Run /mockups {screen-name} to create visual mockups.
```

**STOP.** Do not proceed without user instruction.

---

## Error Handling

| Situation | Response |
|-----------|----------|
| No UI framework detected | Report failure with suggestions → STOP |
| `--scope` path not found | Error with path suggestion → STOP |
| Named screen not found | Report with fuzzy suggestions from discovered screens → STOP (for that screen) |
| CLI/API project, no screens | Report no screens found, suggest checking project type → STOP |
| Unparseable source files | Fall back to manual catalog mode |
| `--update` with no existing specs | "Run /catalog-screens first" → STOP |
| Proposal writeback path invalid | Warn, skip writeback, spec still created |
| `Screen-Specs/` missing | Create directory automatically |

---

**End of /catalog-screens Command**
