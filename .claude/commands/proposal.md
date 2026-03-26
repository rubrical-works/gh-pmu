---
version: "v0.73.0"
description: Create a proposal document and tracking issue (project)
argument-hint: "<title>"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /proposal

Creates a proposal document (`Proposal/[Name].md`) and a tracking issue with the `proposal` label. Also triggered by the `idea:` alias.

**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command proposal`

---

## Prerequisites

- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root

---

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<title>` | No | Proposal title (e.g., `Dark Mode Support`) |

If no title provided, prompt the user for one.

**Alias:** `idea:` is treated identically to `proposal:` — same workflow, same output.

---

## Execution Instructions

**REQUIRED:** Before executing this command:

1. **Generate Todo List:** Parse the workflow steps in this spec, then use `TodoWrite` to create todos
2. **Include Extensions:** For each non-empty `USER-EXTENSION` block, add a todo item
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** If resuming after context compaction, re-read this spec and regenerate todos

**Todo Generation Rules:**
- One todo per numbered step
- One todo per active extension point (non-empty `USER-EXTENSION` blocks)
- Skip commented-out extensions
- Use the step name as the todo content

---

## Workflow

### Step 1: Parse Arguments

Extract `<title>` from command arguments.

**If empty:** Ask the user for a proposal title before proceeding.

**If title contains special characters** (backticks, quotes): Escape appropriately for the shell. On Windows, use temp file approach per shell safety rules.

**Name conversion:** Convert title to file name:
- Replace spaces with hyphens
- Use Title-Case for each word
- Example: `dark mode support` → `Dark-Mode-Support`

### Step 2: Check for Existing Proposal

Check if `Proposal/[Name].md` already exists.

**If file exists:**
```
Proposal/[Name].md already exists. Overwrite? (yes/no)
```
- **If yes:** Continue (will overwrite)
- **If no:** STOP

### Step 3: Gather Description (Mode Selection)

Determine the creation mode based on what was provided in the arguments:

| Input | Title | Mode |
|-------|-------|------|
| Bare `/proposal` (no title, no description) | Ask in Step 1 | **Default to Guided** (no mode prompt) |
| Title only `/proposal Dark Mode` | Provided | **Ask Quick/Guided** via `AskUserQuestion` |
| Title + description `/proposal Dark Mode - adds theme switching` | Provided | **Auto-select Quick** (no mode prompt) |

**Detection:** If the arguments contain a descriptive phrase beyond just a title (e.g., a dash-separated explanation, a sentence, or multi-word detail after the title), treat as "title + description". If only a short title phrase (1-4 words, no separator), treat as "title only".

#### Quick Mode

Preserves the current single-prompt behavior:

```
Briefly describe the proposal (problem and proposed solution):
```

**If the user provides a description:** Use it to populate the proposal template.
**If the user declines or says "skip":** Create with placeholder sections.

#### Guided Mode

Walk through each proposal section with targeted prompts:

1. **Problem Statement:** "What problem does this solve?"
2. **Proposed Solution:** "How would you solve it?" (follow-up: "Any specific files/components affected?")
3. **Implementation Criteria:** "What defines 'done'? List the acceptance criteria."
4. **Alternatives Considered:** "What alternatives did you consider and why reject them?" (skippable — user can say "skip")
5. **Impact Assessment:** "Scope, risk level (low/med/high), effort estimate?" (skippable — user can say "skip")
6. **Screen Discovery:** "Any screens affected?" (skippable — user can say "skip" or "no")
   - **If yes:** Offer to run `/catalog-screens` for the affected screens, or link existing `Screen-Specs/` files
   - **If existing specs found:** List them and ask which to reference
   - **If no screens or skipped:** Continue without screen references

**For each prompt:**
- If the user responds: capture the answer for that section
- If the user says "skip": leave section as "To be documented" placeholder
- Populated sections from guided answers replace "To be documented" placeholders in the generated template

#### Title-Only Mode Prompt

When only a title is provided, use `AskUserQuestion` to let the user choose:

```javascript
AskUserQuestion({
  questions: [{
    question: "How would you like to create this proposal?",
    header: "Mode",
    options: [
      { label: "Quick", description: "Single prompt — describe the proposal in one go" },
      { label: "Guided", description: "Step-by-step — prompted for each section individually" }
    ],
    multiSelect: false
  }]
});
```

<!-- USER-EXTENSION-START: pre-create -->
<!-- USER-EXTENSION-END: pre-create -->

### Step 4: Create Proposal Document

Ensure `Proposal/` directory exists (create if missing).

Create `Proposal/[Name].md` with standard template:

```markdown
# Proposal: [Title]

**Status:** Draft
**Created:** [YYYY-MM-DD]
**Author:** AI Assistant
**Tracking Issue:** (will be updated after issue creation)
**Diagrams:** None

---

## Problem Statement

[Problem description or "To be documented"]

## Proposed Solution

[Solution description or "To be documented"]

## Implementation Criteria

- [ ] [Criterion 1]
- [ ] [Criterion 2]

## Alternatives Considered

- [Alternative 1]: [Why not chosen]

## Impact Assessment

- **Scope:** [Files/components affected]
- **Risk:** [Low/Medium/High]
- **Effort:** [Estimate]
```

**Diagrams:** When a diagram path is specified (user provides one during Guided mode or manually), update the `**Diagrams:**` field from "None" to the file path(s). Create `Proposal/Diagrams/` lazily — only when a diagram path is actually specified. Use the naming convention: `Proposal/Diagrams/[Name]-*.drawio.svg` (e.g., `Proposal/Diagrams/Dark-Mode-Support-architecture.drawio.svg`).

### Step 5: Create Tracking Issue

Build the issue body:

```markdown
## Proposal: [Title]

**File:** Proposal/[Name].md

### Summary

[Brief description from Step 3]

### Lifecycle

- [ ] Proposal reviewed
- [ ] Ready for PRD conversion
```

**Critical:** The issue body MUST include `**File:** Proposal/[Name].md` — this is required for `/create-prd` integration.

Create the issue:
```bash
gh pmu create --title "Proposal: {title}" --label proposal --status backlog --priority p2 --assignee @me -F .tmp-body.md
rm .tmp-body.md
```

**Note:** Always use `-F .tmp-body.md` for the body (never inline `--body`).

### Step 6: Update Proposal with Issue Reference

After issue creation, update the proposal document's tracking issue field:

```
**Tracking Issue:** #[issue-number]
```

### Step 7: Report and STOP

```
Created:
  Document: Proposal/[Name].md
  Issue: #$ISSUE_NUM — Proposal: {title}
  Status: Backlog
  Label: proposal

Say "/review-proposal #$ISSUE_NUM" or "/create-prd #$ISSUE_NUM", if ready
```

<!-- USER-EXTENSION-START: post-create -->
<!-- USER-EXTENSION-END: post-create -->

**STOP.** Do not begin work unless the user explicitly says "work", "implement the proposal", or "work issue".

---

## Error Handling

| Situation | Response |
|-----------|----------|
| No title provided | Prompt user for title |
| Empty title after prompt | "A proposal title is required." → STOP |
| Existing file, user declines overwrite | STOP without creating anything |
| `Proposal/` directory missing | Create it silently |
| `gh pmu create` fails | "Failed to create issue: {error}" → STOP |
| Special characters in title | Escape for shell safety |

---

**End of /proposal Command**
