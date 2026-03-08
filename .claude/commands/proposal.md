---
version: "v0.58.0"
description: Create a proposal document and tracking issue (project)
argument-hint: "<title>"
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
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo item for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction
**Todo Rules:** One todo per numbered step; one todo per active extension; skip commented-out extensions.
---
## Workflow
### Step 1: Parse Arguments
Extract `<title>` from command arguments.
**If empty:** Ask the user for a proposal title.
**If title contains special characters** (backticks, quotes): Escape for shell. On Windows, use temp file approach.
**Name conversion:** Convert title to file name: replace spaces with hyphens, Title-Case each word. Example: `dark mode support` → `Dark-Mode-Support`
### Step 2: Check for Existing Proposal
Check if `Proposal/[Name].md` already exists.
**If file exists:** Ask `Proposal/[Name].md already exists. Overwrite? (yes/no)`. If no → STOP.
### Step 3: Gather Description (Mode Selection)
Determine creation mode from arguments:
| Input | Title | Mode |
|-------|-------|------|
| Bare `/proposal` (no title, no description) | Ask in Step 1 | **Default to Guided** (no mode prompt) |
| Title only `/proposal Dark Mode` | Provided | **Ask Quick/Guided** via `AskUserQuestion` |
| Title + description `/proposal Dark Mode - adds theme switching` | Provided | **Auto-select Quick** (no mode prompt) |
**Detection:** Descriptive phrase beyond title (dash-separated, sentence, multi-word detail) = "title + description". Short title (1-4 words, no separator) = "title only".
#### Quick Mode
Preserves single-prompt behavior:
```
Briefly describe the proposal (problem and proposed solution):
```
**If user provides description:** Populate template. **If "skip":** Use "To be documented" placeholders.
#### Guided Mode
Walk through each section:
1. **Problem Statement:** "What problem does this solve?"
2. **Proposed Solution:** "How would you solve it?" (follow-up: "Any specific files/components affected?")
3. **Implementation Criteria:** "What defines 'done'? List the acceptance criteria."
4. **Alternatives Considered:** "What alternatives did you consider and why reject them?" (skippable)
5. **Impact Assessment:** "Scope, risk level (low/med/high), effort estimate?" (skippable)
Populated answers replace "To be documented" placeholders in the template.
#### Title-Only Mode Prompt
Use `AskUserQuestion`:
```javascript
AskUserQuestion({
  questions: [{
    question: "How would you like to create this proposal?",
    header: "Mode",
    options: [
      { label: "Quick", description: "Single prompt — describe in one go" },
      { label: "Guided", description: "Step-by-step — prompted for each section" }
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
**Diagrams:** When a diagram path is specified, update `**Diagrams:**` from "None" to the path(s). Create `Proposal/Diagrams/` lazily (only when needed). Naming convention: `Proposal/Diagrams/[Name]-*.drawio.svg`.
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
**Critical:** The issue body MUST include `**File:** Proposal/[Name].md` — required for `/create-prd` integration.
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
