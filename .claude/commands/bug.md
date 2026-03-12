---
version: "v0.61.0"
description: Create a bug issue with standard template (project)
argument-hint: "<title>"
---
<!-- EXTENSIBLE -->

# /bug
Creates a properly labeled bug issue with a standard template and adds it to the project board.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command bug`

## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `<title>` | No | Bug title (e.g., `assign-branch fails on Windows paths`) |
If no title provided, prompt the user for one.

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

## Workflow

### Step 1: Parse Arguments
Extract `<title>` from command arguments.
**If empty:** Ask the user for a bug title before proceeding.
**If title contains special characters** (backticks, quotes): Escape appropriately for the shell. On Windows, use temp file approach per shell safety rules.

### Step 2: Gather Description
Extract `<body>` from command arguments.
**IF** there is insufficient detail provided in the arguments to create the issue, **THEN**:
Ask the user to describe the bug:
```
Describe the bug (steps to reproduce, expected vs actual behavior):
```
**If the user provides a description:** Use it as the issue body.
**If the user declines or says "skip":** Create with a minimal body.
<!-- USER-EXTENSION-START: pre-create -->
<!-- USER-EXTENSION-END: pre-create -->

### Step 3: Create Issue
Build the issue body with a standard bug template:
```markdown
## Bug Report

**Description:**
{user description or "To be documented"}

**Steps to Reproduce:**
1. ...

**Expected Behavior:**
...

**Actual Behavior:**
...
```
Create the issue:
```bash
gh pmu create --title "[Bug]: {title}" --label bug --status backlog --priority p1 --assignee @me -F .tmp-body.md
rm .tmp-body.md
```
**Note:** Always use `-F .tmp-body.md` for the body (never inline `--body`).

### Step 4: Report and STOP
```
Created: Issue #$ISSUE_NUM — [Bug]: {title}
Status: Backlog
Label: bug

Say "/review-issue #$ISSUE_NUM" then "/assign-branch #$ISSUE_NUM" then "work #$ISSUE_NUM" to start working on this bug.
```
<!-- USER-EXTENSION-START: post-create -->
<!-- USER-EXTENSION-END: post-create -->
**STOP.** Do not begin work unless the user explicitly says "work", "fix that", or "implement that".

## Error Handling
| Situation | Response |
|-----------|----------|
| No title provided | Prompt user for title |
| Empty title after prompt | "A bug title is required." → STOP |
| `gh pmu create` fails | "Failed to create issue: {error}" → STOP |
| Special characters in title | Escape for shell safety |
**End of /bug Command**
