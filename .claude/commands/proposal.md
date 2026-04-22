---
version: "v0.90.0"
description: Create a proposal document and tracking issue (project)
argument-hint: "<title>"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /proposal

Creates a proposal document (`Proposal/[Name].md`) and a tracking issue with the `proposal` label. Also triggered by the `idea:` alias.

**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command proposal`

## Prerequisites

- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<title>` | No | Proposal title (e.g., `Dark Mode Support`) |

If no title provided, prompt the user. **Alias:** `idea:` is identical to `proposal:`.

## Execution Instructions

**REQUIRED:** Before executing:

1. Use `TaskCreate` for one task per step below. No routing → bulk create upfront (see rule `07-task-creation-timing.md`).
2. Include one task per active (non-empty) `USER-EXTENSION` block.
3. Mark tasks `in_progress` → `completed` via `TaskUpdate`.
4. **Post-Compaction:** Re-read spec, call `TaskList`, resume from first incomplete task.

## Workflow

### Step 1: Parse Arguments

Extract `<title>` from arguments. **If empty:** ask for title. **If special characters** (backticks, quotes): escape for shell; on Windows use temp file approach.

**Name conversion:** Replace spaces with hyphens, Title-Case each word. Example: `dark mode support` → `Dark-Mode-Support`.

### Step 2: Check for Existing Proposal

If `Proposal/[Name].md` exists, ask `Proposal/[Name].md already exists. Overwrite? (yes/no)`. No → STOP.

### Step 3: Gather Description (Mode Selection)

| Input | Title | Mode |
|-------|-------|------|
| Bare `/proposal` (no title, no description) | Ask in Step 1 | **Default to Guided** (no mode prompt) |
| Title only `/proposal Dark Mode` | Provided | **Ask Quick/Guided** via `AskUserQuestion` |
| Title + description `/proposal Dark Mode - adds theme switching` | Provided | **Auto-select Quick** (no mode prompt) |

**Detection:** Descriptive phrase beyond title (dash-separated, sentence, multi-word detail) → "title + description". Short title (1-4 words, no separator) → "title only".

#### Quick Mode

Single prompt: `Briefly describe the proposal (problem and proposed solution):`

If user provides description: populate template. If declines/skip: placeholder sections.

#### Guided Mode

Walk through sections:

1. **Problem Statement:** "What problem does this solve?"
2. **Proposed Solution:** "How would you solve it?" (follow-up: "Any specific files/components affected?")
3. **Implementation Criteria:** "What defines 'done'? List the acceptance criteria."
4. **Alternatives Considered:** "What alternatives did you consider and why reject them?" (skippable)
5. **Impact Assessment:** "Scope, risk level (low/med/high), effort estimate?" (skippable)
6. **Screen Discovery:** "Any screens affected?" (skippable)
   - Yes → offer `/catalog-screens` or link existing `Screen-Specs/`
   - Existing specs found → list and ask which to reference
   - No or skip → continue without screen references

**For each prompt:** capture answer, or on "skip" leave "To be documented" placeholder. Populated sections replace placeholders.

#### Title-Only Mode Prompt

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

Ensure `Proposal/` directory exists. Create `Proposal/[Name].md`:

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

**Diagrams:** When a diagram path is specified, update `**Diagrams:**` from "None" to the path(s). Create `Proposal/Diagrams/` lazily. Naming: `Proposal/Diagrams/[Name]-*.drawio.svg`.

### Step 5: Create Tracking Issue

Build issue body:

```markdown
## Proposal: [Title]

**File:** Proposal/[Name].md

### Summary

[Brief description from Step 3]

### Lifecycle

- [ ] Proposal reviewed
- [ ] Ready for PRD conversion
```

**Critical:** Body MUST include `**File:** Proposal/[Name].md` — required for `/create-prd` integration.

```bash
gh pmu create --title "Proposal: {title}" --label proposal --status backlog --priority p2 --assignee @me -F .tmp-body.md
rm .tmp-body.md
```

**Note:** Always use `-F .tmp-body.md` (never inline `--body`).

### Step 6: Update Proposal with Issue Reference

Update tracking issue field: `**Tracking Issue:** #[issue-number]`

### Step 6a: Commit Proposal

**Guard:** Only commit if changes exist:
```bash
git diff --name-only -- "Proposal/"
git diff --cached --name-only -- "Proposal/"
```
If no changes, skip silently.

**If changes exist:**
```bash
git add "Proposal/[Name].md"
```

Commit message:
```
docs: add proposal — [Title] (Refs #$ISSUE_NUM)
```

For modifications: `docs: update proposal — [Title] (Refs #$ISSUE_NUM)`

**Note:** Use `Refs #` (not `Fixes #`) per workflow rules — proposal issue stays open.

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

**STOP.** Do not begin work unless user explicitly says "work", "implement the proposal", or "work issue".

## Error Handling

| Situation | Response |
|-----------|----------|
| No title provided | Prompt user for title |
| Empty title after prompt | "A proposal title is required." → STOP |
| Existing file, user declines overwrite | STOP without creating anything |
| `Proposal/` directory missing | Create it silently |
| `gh pmu create` fails | "Failed to create issue: {error}" → STOP |
| Special characters in title | Escape for shell safety |

**End of /proposal Command**
