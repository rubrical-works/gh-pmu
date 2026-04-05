---
version: "v0.81.1"
description: Create a bug issue with standard template (project)
argument-hint: "<title>"
copyright: "Rubrical Works (c) 2026"
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
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** For each non-empty `USER-EXTENSION` block, add a todo item
3. **Track Progress:** Mark todos `in_progress` -> `completed`
4. **Post-Compaction:** Re-read this spec and regenerate todos
## Workflow
### Step 1: Parse Arguments
Extract `<title>` from arguments. If empty, ask user. Escape special characters for shell.
### Step 2: Gather Description
Extract `<body>` from arguments. If insufficient detail:
```
Describe the bug (steps to reproduce, expected vs actual behavior):
```
If user declines or says "skip": create with minimal body.
### Step 2b: Detect Version
Auto-detect priority: `package.json` version -> latest git tag -> prompt user.
If detected, confirm via `AskUserQuestion`: `"Detected version: {version}. Is this correct?"`
Options: `"Yes, use {version}"` (default), `"No, let me specify"`

<!-- USER-EXTENSION-START: pre-create -->
<!-- USER-EXTENSION-END: pre-create -->

### Step 3: Create Issue
Build issue body with standard bug template:
```markdown
## Bug Report
**Description:**
{user description or "To be documented"}
**Version:**
{detected or user-provided version}
**Steps to Reproduce:**
1. ...
**Expected Behavior:**
...
**Actual Behavior:**
...
**Scope:**
- **In scope:** {infer from description, or "To be documented"}
- **Out of scope:** {infer from description, or "To be documented"}
**Acceptance Criteria:**
- [ ] {infer from description, or "To be documented"}
**Proposed Fix:**
{infer from description if enough context, or "To be documented"}
```
Populate from user's description where possible. Use "To be documented" for sections without enough input.
```bash
gh pmu create --title "[Bug]: {title}" --label bug --status backlog --priority p1 --assignee @me -F .tmp-body.md
rm .tmp-body.md
```
Always use `-F .tmp-body.md` for the body (never inline `--body`).
### Step 4: Report and STOP
```
Created: Issue #$ISSUE_NUM -- [Bug]: {title}
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
| Empty title after prompt | "A bug title is required." -> STOP |
| `gh pmu create` fails | "Failed to create issue: {error}" -> STOP |
| Special characters in title | Escape for shell safety |
**End of /bug Command**
