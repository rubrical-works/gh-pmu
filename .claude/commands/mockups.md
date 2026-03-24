---
version: "v0.70.0"
description: Create text-based or diagrammatic screen mockups (project)
argument-hint: "[#NN]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /mockups
Creates text-based or diagrammatic screen mockups. Fully interactive via `AskUserQuestion`. Optional `#NN` issue reference pre-populates context.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command mockups`
---
## Prerequisites
- Shared screen spec schema: `.claude/metadata/screen-spec-schema.json`
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#NN` | No | Issue number (bug/enhancement/proposal/PRD). Pre-populates interactive flow. |
```
/mockups            # Fully interactive
/mockups #42        # With issue context
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

<!-- USER-EXTENSION-START: pre-mockup -->
<!-- USER-EXTENSION-END: pre-mockup -->

### Step 1: Discovery and Interactive Setup
**Step 1a: Load Context**
**If `#NN` provided:** Read issue via `gh issue view #NN --json body,title,labels`. Extract type, screen names, existing references.
**Always:** Read `.claude/metadata/screen-spec-schema.json`.
**Step 1b: Discover Existing Content**
Before asking questions, scan `Mockups/` and subdirectories:
- List all `Mockups/{Name}/` directories
- Inventory `Specs/`, `Screens/`, `AsciiScreens/` contents
**Step 1c: Interactive Question Flow**
**Q1: What would you like to do?** `AskUserQuestion`:
- "Create new mockups"
- "Modify existing mockups"
- "View/browse existing mockup sets"
**Conditions:** With `#NN`: pre-select from issue type (enhancement→Create, bug→Modify). No existing mockups: skip Q1, default to "Create new mockups".
**Q2: Which mockup set?** `AskUserQuestion`:
- List existing `Mockups/{Name}/` directories
- "Create a new mockup set"
With `#NN`: pre-suggest name from issue title.
**Q2a** (if new): Free text name. With `#NN`: suggest from title.
**Q3: What type of mockups?** `AskUserQuestion`:
- "ASCII/text mockups" → `AsciiScreens/`
- "Interactive UI mockups (drawio.svg)" → `Screens/`
- "Both"
**Q4: How should content be sourced?** `AskUserQuestion`:
- "From existing screen specs"
- "From source code discovery"
- "Describe screens manually"
- "From issue #NN description" (only with `#NN`)
If `Mockups/{Name}/Specs/` has specs, show as sources.
**Q4a** (from specs): `multiSelect: true` list of available specs.
**Q4b** (source discovery): Free text path, defaults to full project.
**Q5** (per screen): **Review mockup?** `AskUserQuestion`:
- "Looks good, save it"
- "Make adjustments"
- "Skip this screen"
**Q6** (Modify flow): **Which mockups?** `multiSelect: true` list of existing files. Then ask changes via conversation.
**Without `#NN`:** All questions start fresh, no pre-populated answers.
### Step 2: Generate Mockup
**ASCII/text** → `Mockups/{Name}/AsciiScreens/{Screen-Name}-mockup.md`:
```markdown
# Mockup: {Screen Name}
**Screen Spec:** Mockups/{Name}/Specs/{Screen-Name}.md
**Created:** {YYYY-MM-DD}
---
## Layout
{ASCII/Unicode box drawing}
## Element Placement Notes
| Element | Position | Size/Span | Notes |
|---------|----------|-----------|-------|
---
*Mockup created {YYYY-MM-DD} by /mockups*
```
**Diagram** → `Mockups/{Name}/Screens/{Screen-Name}-mockup.drawio.svg`: Use `drawio-generation` skill.
### Step 3: Collision Protection and Write
**If target exists:** `AskUserQuestion`: overwrite / alternative name / skip.
**If not:** Write directly. Create all dirs if missing.
### Step 4: Cross-Reference Updates
Update screen spec `## Related Artifacts` with mockup paths. Mockup header references its spec.

<!-- USER-EXTENSION-START: post-mockup -->
<!-- USER-EXTENSION-END: post-mockup -->

### Step 5: README.md Auto-Generation
Auto-generate `Mockups/{Name}/README.md` listing all files in Specs/, Screens/, AsciiScreens/. Omit empty sections. Updated after each mockup creation/modification.
### Step 6: Proposal Writeback (if applicable)
If from proposal context or `#NN` is proposal: append `## Mockups` section with file references. Invalid path → warn, skip, mockup still created.
### Step 7: Report
```
Mockup complete.
  Mockup set: Mockups/{Name}/
  Screens: {names}
  Output: {files}
  README: Mockups/{Name}/README.md (updated)
  Cross-references: {updated | no spec}
  Related: /catalog-screens to create or update screen specs.
```
**STOP.** Do not proceed without user instruction.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| No arg, no mockups | Skip Q1, default Create |
| `#NN` not found | Continue without context |
| Source discovery fails | Suggest manual or /catalog-screens |
| Dirs missing | Create automatically |
| File collision | Ask: overwrite, alternative, skip |
| Spec update fails | Warn, continue |
| Proposal path invalid | Warn, skip writeback |
| Schema missing | STOP |
---
**End of /mockups Command**
