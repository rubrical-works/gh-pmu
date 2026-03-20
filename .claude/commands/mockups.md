---
version: "v0.67.0"
description: Create text-based or diagrammatic screen mockups (project)
argument-hint: "<screen-name> [screen-name...] [--from-spec]"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /mockups

Creates text-based or diagrammatic screen mockups for UI screens, with cross-references to screen specs. Mockups visualize element layout and placement before implementation.

**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command mockups`

---

## Prerequisites

- `Mockups/` directory at project root (created automatically if missing)
- For `--from-spec`: Existing screen spec in `Screen-Specs/{name}.md`

---

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<screen-name>` | Yes | Create a mockup for a specific screen (e.g., `Login`) |
| `<name1> <name2>` | No | Create mockups for multiple screens in one invocation |
| `--from-spec` | No | Generate mockup from an existing `Screen-Specs/{name}.md` instead of source code |

---

## Execution Instructions

**REQUIRED:** Before executing:

1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction

---

## Workflow

<!-- USER-EXTENSION-START: pre-mockup -->
<!-- USER-EXTENSION-END: pre-mockup -->

### Step 1: Parse Arguments and Resolve Source

Extract screen name(s) and `--from-spec` flag.

**If `--from-spec` specified:**
1. Check for `Screen-Specs/{screen-name}.md`
2. **If found:** Read the spec and use its element table as the mockup source
3. **If not found:**
   ```
   Screen spec not found: Screen-Specs/{screen-name}.md
   Run /catalog-screens {screen-name} first, or omit --from-spec to discover from source.
   ```
   → **STOP** (for that screen; continue with others if multi-screen)

**If `--from-spec` NOT specified:**
1. Check for existing `Screen-Specs/{screen-name}.md` — if found, use it (implicit --from-spec)
2. If no spec exists: discover screen from source code (same discovery as `/catalog-screens` Step 2-3)
3. **If source discovery fails:** Report with suggestions:
   ```
   Screen "{name}" not found in source and no spec exists.
   Run /catalog-screens {name} first, or check the screen name.
   ```
   → **STOP** (for that screen)

### Step 2: Generate Mockup

Based on the screen's elements (from spec or source discovery), create a visual representation.

**Mockup format options:**

| Format | Output File | When to Use |
|--------|-------------|-------------|
| Text-based (ASCII/Unicode) | `Mockups/{Screen-Name}-mockup.md` | Default — simple forms, standard layouts |
| Diagram-based | `Mockups/{Screen-Name}-mockup.drawio.svg` | Complex layouts, multi-panel screens |

**Text-based mockup structure:**

```markdown
# Mockup: {Screen Name}

**Screen Spec:** Screen-Specs/{Screen-Name}.md
**Created:** {YYYY-MM-DD}

---

## Layout

{ASCII/Unicode box drawing of the screen layout}

Example:
+------------------------------------------+
|              Login Screen                 |
+------------------------------------------+
|                                           |
|  Username:  [____________________]        |
|                                           |
|  Password:  [____________________]        |
|                                           |
|  [x] Remember me                         |
|                                           |
|         [ Login ]   [ Cancel ]            |
|                                           |
|  Forgot password?                         |
+------------------------------------------+

## Element Placement Notes

| Element | Position | Size/Span | Notes |
|---------|----------|-----------|-------|
| username | Row 1 | Full width | Auto-focus on load |
| password | Row 2 | Full width | — |
| remember | Row 3 | Checkbox | Default: unchecked |
| login | Row 4, Left | Button | Primary action |
| cancel | Row 4, Right | Button | Secondary action |
| forgot | Row 5 | Link | Below buttons |

---

*Mockup created {YYYY-MM-DD} by /mockups*
```

**Diagram-based mockup:** Use `.drawio.svg` format with editable `mxGraphModel` structure per the `drawio-generation` skill.

### Step 3: Cross-Reference Updates

After creating the mockup:

**Update screen spec (if exists):**
1. Read `Screen-Specs/{Screen-Name}.md`
2. Update the `## Related Artifacts` section:
   ```markdown
   ## Related Artifacts

   - **Mockup:** `Mockups/{Screen-Name}-mockup.md`
   ```
3. Write the updated spec

**Mockup references its spec:**
The mockup's `**Screen Spec:**` field (in the header) points to the screen spec file.

<!-- USER-EXTENSION-START: post-mockup -->
<!-- USER-EXTENSION-END: post-mockup -->

### Step 4: Proposal Writeback (if applicable)

If `/mockups` was triggered from a proposal context (proposal file path available):

1. Read the proposal document
2. Append or update `## Mockups` section with file references:
   ```markdown
   ## Mockups

   - `Mockups/{Screen-Name-1}-mockup.md`
   - `Mockups/{Screen-Name-2}-mockup.md`
   ```
3. If proposal file path is invalid or deleted → warn, skip writeback, mockup still created

### Step 5: Write Output

Ensure `Mockups/` directory exists (create if missing).

Write the mockup file to `Mockups/{Screen-Name}-mockup.md` (or `.drawio.svg`).

### Step 6: Report

```
Mockup complete.
  Screens: {names}
  Output: Mockups/{names}-mockup.md
  Cross-references: {updated | no spec exists}

  Related: /catalog-screens {name} to create or update screen specs.
```

**STOP.** Do not proceed without user instruction.

---

## Error Handling

| Situation | Response |
|-----------|----------|
| Screen name not provided | "Provide a screen name: `/mockups Login`" → STOP |
| `--from-spec` with missing spec | "Screen spec not found. Run /catalog-screens first." → STOP |
| Source discovery fails (no spec, no source) | "Screen not found in source and no spec exists." → STOP |
| `Mockups/` missing | Create directory automatically |
| Spec cross-reference update fails | Warn, continue (mockup still created) |
| Proposal writeback path invalid | Warn, skip writeback, mockup still created |

---

**End of /mockups Command**
