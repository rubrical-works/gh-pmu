---
version: "v0.77.3"
description: Create a proposal document and tracking issue (project)
argument-hint: "<title>"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /proposal
Creates a proposal document (`Proposal/[Name].md`) and tracking issue with `proposal` label. Also triggered by `idea:` alias.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command proposal`
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `<title>` | No | Proposal title (e.g., `Dark Mode Support`) |
If no title provided, prompt user. **Alias:** `idea:` = `proposal:`.
## Execution Instructions
**REQUIRED:** Parse workflow steps, use `TodoWrite` to create todos. Include non-empty `USER-EXTENSION` blocks. Track progress. Re-read spec after compaction.
## Workflow
### Step 1: Parse Arguments
Extract `<title>`. **If empty:** Ask user. **If special characters:** Escape for shell (Windows: temp file).
**Name conversion:** Replace spaces with hyphens, Title-Case. Example: `dark mode support` -> `Dark-Mode-Support`
### Step 2: Check for Existing Proposal
If `Proposal/[Name].md` exists: ask overwrite? If no, STOP.
### Step 3: Gather Description (Mode Selection)
| Input | Title | Mode |
|-------|-------|------|
| Bare `/proposal` | Ask in Step 1 | **Default Guided** |
| Title only `/proposal Dark Mode` | Provided | **Ask Quick/Guided** |
| Title + description `/proposal Dark Mode - adds theme switching` | Provided | **Auto Quick** |
**Detection:** Descriptive phrase beyond title = "title + description". Short phrase (1-4 words) = "title only".
#### Quick Mode
Prompt: `Briefly describe the proposal (problem and proposed solution):` Use response to populate template. If "skip": use placeholders.
#### Guided Mode
1. **Problem Statement:** "What problem does this solve?"
2. **Proposed Solution:** "How would you solve it?" (follow-up: files/components?)
3. **Implementation Criteria:** "What defines 'done'?"
4. **Alternatives Considered:** (skippable)
5. **Impact Assessment:** scope, risk, effort (skippable)
6. **Screen Discovery:** screens affected? (skippable) — offer `/catalog-screens` or link `Screen-Specs/`
Each prompt: capture answer or "To be documented" if skipped.
#### Title-Only Mode Prompt
Use `AskUserQuestion` with options: "Quick" or "Guided".

<!-- USER-EXTENSION-START: pre-create -->
<!-- USER-EXTENSION-END: pre-create -->

### Step 4: Create Proposal Document
Ensure `Proposal/` exists. Create `Proposal/[Name].md`:
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
**Diagrams:** When path specified, update `**Diagrams:**` field. Create `Proposal/Diagrams/` lazily. Convention: `Proposal/Diagrams/[Name]-*.drawio.svg`.
### Step 5: Create Tracking Issue
Build issue body with `**File:** Proposal/[Name].md` (required for `/create-prd`) and lifecycle checklist.
```bash
gh pmu create --title "Proposal: {title}" --label proposal --status backlog --priority p2 --assignee @me -F .tmp-body.md
rm .tmp-body.md
```
Always use `-F .tmp-body.md` (never inline `--body`).
### Step 6: Update Proposal with Issue Reference
Update `**Tracking Issue:** #[issue-number]` in proposal document.
### Step 6a: Commit Proposal
**Guard:** Only commit if changes in `Proposal/`:
```bash
git diff --name-only -- "Proposal/"
git diff --cached --name-only -- "Proposal/"
```
If no changes, skip. If changes:
```bash
git add "Proposal/[Name].md"
```
Commit: `docs: add proposal — [Title] (Refs #$ISSUE_NUM)` (or `update proposal` for modifications). Use `Refs #` not `Fixes #`.
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
| Empty title after prompt | "A proposal title is required." -> STOP |
| Existing file, user declines overwrite | STOP without creating anything |
| `Proposal/` directory missing | Create it silently |
| `gh pmu create` fails | "Failed to create issue: {error}" -> STOP |
| Special characters in title | Escape for shell safety |
**End of /proposal Command**
