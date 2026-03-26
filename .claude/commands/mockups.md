---
version: "v0.72.0"
description: Create text-based or diagrammatic screen mockups (project)
argument-hint: "[#NN]"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /mockups

Creates text-based or diagrammatic screen mockups for UI screens. Fully interactive via `AskUserQuestion`. Optional `#NN` issue reference pre-populates context from a bug, enhancement, proposal, or PRD.

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
/mockups            # Fully interactive, no context
/mockups #42        # Interactive with issue #42 context
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

### Step 1a: Load Context

**If `#NN` provided:**
1. Read the issue body via `gh issue view #NN --json body,title,labels`
2. Extract: issue type (from labels), screen/feature names, existing mockup/spec references

**Always:** Read `.claude/metadata/screen-spec-schema.json`.

### Step 1b: Discover Existing Content

Before asking questions, scan `Mockups/` and subdirectories:
- List all `Mockups/{Name}/` directories
- For each, inventory `Specs/`, `Screens/`, and `AsciiScreens/` contents

### Step 1b-ii: ASCII-Only Detection and Conversion Offer

**Detection:** `AsciiScreens/` has files AND `Screens/` is empty or does not exist.

**If ASCII-only detected:** Use `AskUserQuestion`:
- Question: "This mockup set contains only ASCII mockups. Would you like to convert them to interactive mockups and create specs?"
- **Yes, convert** — Generate `.drawio.svg` from ASCII sources; create specs in `Specs/` if missing. Report: `"Converted {N} ASCII mockups. Created {M} screen specs."` Continue with normal processing.
- **No, continue** — Skip conversion.

**If not ASCII-only:** Skip silently.

### Step 1c: Pipeline Context Detection

**If `#NN` provided**, check for artifacts from other UI design pipeline commands:

**Screen spec detection:**
1. Check issue body/linked proposal for `## Screen Specs` section with file references
2. Scan `Mockups/*/Specs/` for specs matching screens in the issue title/body
3. If found: pre-select in Q4, report: `"Screen specs found: {names} — available as starting point."`

**Path analysis detection:**
1. Check proposal file or issue body for `## Path Analysis` section
2. If found: report category names and path counts
3. `AskUserQuestion`: "Use path analysis to scope mockups" / "Ignore path analysis — scope mockups by screen"
4. If "Use path analysis": generate a mockup per distinct screen state; group by scenario category in README.

**No `#NN` or no artifacts:** Skip silently.

### Step 1d: Interactive Question Flow

**Q1: What would you like to do?** `AskUserQuestion`:
- "Create new mockups"
- "Modify existing mockups"
- "View/browse existing mockup sets"

**Conditions:** With `#NN`: pre-select from issue type (enhancement/proposal → Create, bug → Modify). No existing mockups: skip Q1, default to "Create new mockups".

**Q2: Which mockup set?** `AskUserQuestion`:
- List each existing `Mockups/{Name}/` directory
- "Create a new mockup set"

With `#NN`: pre-suggest name derived from issue title.

**Q2a** (if new set): Free text name via `AskUserQuestion`. With `#NN`: suggest from issue title. Creates `Mockups/{Name}/`.

**Q3: What type of mockups do you need?**

Before presenting options, detect the project's UI framework:
```bash
node .claude/scripts/shared/mockup-detect-framework.js
```
If a supported framework is detected (exit code 0), add "Framework-native components" as the first option.

`AskUserQuestion`:
- "Framework-native components ({framework})" → `Components/` (only shown when framework detected)
- "ASCII/text mockups" → `AsciiScreens/`
- "Interactive UI mockups (drawio.svg)" → `Screens/`
- "Both ASCII + drawio.svg" → both subdirectories

**Framework-native:** Generate layout-only components using the detected framework's extensions (`.tsx`, `.svelte`, `.vue`, `.component.ts`/`.component.html`). Structure and placeholder content only — no business logic, state, or API calls. Written to `Mockups/{Name}/Components/`.

**Fallback decision table (when framework-native is NOT available):**

| Scenario | Detection | Fallback to |
|----------|-----------|-------------|
| No UI framework | No supported package in deps, no charter match | `.drawio.svg` |
| Non-web framework | Only Electron/Tauri/etc. without React/Vue/etc. | `.drawio.svg` |
| Multiple frameworks | >1 supported framework in deps | `.drawio.svg` |
| Framework not chosen | Charter tech stack is TBD/placeholder | `.drawio.svg` |
| Static/content site | Astro/Eleventy/Hugo without component framework | `.drawio.svg` |

When fallback applies, "Framework-native components" is not shown — Q3 presents three options only.

**Q4: How should screen content be sourced?** `AskUserQuestion`:
- "From existing screen specs" (replaces `--from-spec`)
- "From source code discovery"
- "Describe screens manually"
- "From issue #NN description" (only shown when `#NN` provided)

If `Mockups/{Name}/Specs/` has specs, show them as available sources.

**Q4a** (from specs): `AskUserQuestion` with `multiSelect: true` — list available specs in `Mockups/{Name}/Specs/`.

**Q4b** (source discovery): Free text path via `AskUserQuestion`. Defaults to full project scan.

**Q5** (per screen): **Review mockup for {Screen}?** `AskUserQuestion`:
- "Looks good, save it"
- "Make adjustments" → follow-up conversation for changes
- "Skip this screen"

**Q6** (Modify flow): **Which mockups?** `AskUserQuestion` with `multiSelect: true` — list existing files in selected set. Then ask what changes are needed via conversation.

**Without `#NN`:** All questions start fresh with no pre-populated answers.

**Per-screen progress tracking:** After Q4 resolves the screen list, create one todo per screen. Post-compaction: re-read spec, check `Mockups/{Name}/` for partially created files, resume from first unwritten screen.

### Step 2: Generate Mockup

**ASCII/text mockup** (written to `Mockups/{Name}/AsciiScreens/{Screen-Name}-mockup.md`):

```markdown
# Mockup: {Screen Name}

**Screen Spec:** Mockups/{Name}/Specs/{Screen-Name}.md
**Created:** {YYYY-MM-DD}

---

## Layout

{ASCII/Unicode box drawing of the screen layout}

## Element Placement Notes

| Element | Position | Size/Span | Notes |
|---------|----------|-----------|-------|

---

*Mockup created {YYYY-MM-DD} by /mockups*
```

**Diagram-based mockup** (written to `Mockups/{Name}/Screens/{Screen-Name}-mockup.drawio.svg`):
Use `.drawio.svg` format with editable `mxGraphModel` structure per the `drawio-generation` skill.

### Step 3: Collision Protection and Write

**Before writing each file,** check if the target exists.
- **Exists:** `AskUserQuestion`: "Overwrite existing mockup" / "Save with alternative name" / "Skip this mockup"
- **Does not exist:** Write directly.

Ensure all directories exist (create if missing): `Mockups/{Name}/`, `Mockups/{Name}/AsciiScreens/`, `Mockups/{Name}/Screens/`, `Mockups/{Name}/Specs/`.

### Step 4: Cross-Reference Updates

**Update screen spec (if exists in `Mockups/{Name}/Specs/`):**
1. Read `Mockups/{Name}/Specs/{Screen-Name}.md`
2. Update `## Related Artifacts` section:
   ```markdown
   ## Related Artifacts

   - **Mockup:** `Mockups/{Name}/AsciiScreens/{Screen-Name}-mockup.md`
   - **Mockup:** `Mockups/{Name}/Screens/{Screen-Name}-mockup.drawio.svg`
   ```
3. Write the updated spec

**Mockup references its spec:** The mockup's `**Screen Spec:**` field points to the screen spec file.

<!-- USER-EXTENSION-START: post-mockup -->
<!-- USER-EXTENSION-END: post-mockup -->

### Step 5: README.md Auto-Generation

Auto-generate (or update) `Mockups/{Name}/README.md`:

```markdown
# Mockup Set: {Name}

**Last Updated:** {YYYY-MM-DD}

## Contents

### Specs
- `Specs/{Screen-Name-1}.md`

### Screens (Interactive)
- `Screens/{Screen-Name-1}-mockup.drawio.svg`

### ASCII Screens
- `AsciiScreens/{Screen-Name-1}-mockup.md`

---

*Auto-generated by /mockups*
```

List all files found in each subdirectory. Omit empty sections.

### Step 6: Issue Writeback (if applicable)

If `#NN` provided, write mockup references back to the source:

**Proposal:** Append or update `## Mockups` section in the proposal document:
```markdown
## Mockups

- `Mockups/{Name}/AsciiScreens/{Screen-Name-1}-mockup.md`
- `Mockups/{Name}/Screens/{Screen-Name-1}-mockup.drawio.svg`
```
Invalid/deleted path → warn, skip writeback, mockup still created.

**Enhancement or Bug:** Update issue body via `gh pmu view #NN --body-stdout` / `gh pmu edit #NN -F`. Append or update `## Mockups` section; replace contents if section exists.

**No `#NN`:** Skip writeback.

### Step 7: Report

```
Mockup complete.
  Mockup set: Mockups/{Name}/
  Screens: {names}
  Output: {list of created/modified files}
  README: Mockups/{Name}/README.md (updated)
  Cross-references: {updated | no spec exists}

  Related: /catalog-screens to create or update screen specs.
```

### Step 8: Commit Offer and STOP

If any files were created or modified, offer to stage and commit.

`AskUserQuestion`: "Stage and commit mockup changes?" — **Yes** / **No**

**If Yes:**
```bash
git add Mockups/{Name}/
git commit -m "Refs #NN -- Add/update mockups for {Name}"
```
- With `#NN`: use `Refs #NN`. Always `Refs` — mockup creation does not close issues.
- Without issue context: descriptive message without issue reference.

**If No:** Skip — do not stage or commit.

**STOP.** Do not proceed without user instruction.

---

## Error Handling

| Situation | Response |
|-----------|----------|
| No argument and no existing mockups | Skip Q1, default to "Create new mockups" |
| `#NN` issue not found | "Issue #NN not found" → continue without issue context |
| Source discovery fails (no spec, no source) | Suggest "Describe screens manually" or "Run /catalog-screens first" |
| `Mockups/` missing | Create directory structure automatically |
| File collision on write | Ask user: overwrite, alternative name, or skip |
| Spec cross-reference update fails | Warn, continue (mockup still created) |
| Proposal writeback path invalid | Warn, skip writeback, mockup still created |
| Schema file missing | "Shared schema not found at .claude/metadata/screen-spec-schema.json" → STOP |

---

**End of /mockups Command**
