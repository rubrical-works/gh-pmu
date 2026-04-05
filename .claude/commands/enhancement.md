---
version: "v0.82.0"
description: Create an enhancement issue with standard template (project)
argument-hint: "<title>"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /enhancement
Creates a properly labeled enhancement issue with standard template on the project board.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command enhancement`
**Prerequisites:** `gh pmu` installed, `.gh-pmu.json` configured.

| Argument | Required | Description |
|----------|----------|-------------|
| `<title>` | No | Enhancement title (e.g., `add dark mode support`) |

If no title provided, prompt the user.
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` -> `completed`
4. **Post-Compaction:** Re-read spec and regenerate todos
## Workflow
### Step 1: Parse Arguments
Extract `<title>` from arguments.
**If empty:** Ask user for enhancement title.
**If special characters** (backticks, quotes): Escape for shell. On Windows, use temp file approach.
### Step 2: Gather Description
Extract `<body>` from arguments. **IF** insufficient detail, **THEN** ask:
```
Describe the enhancement (what it does, why it's useful):
```
**If description provided:** Use as issue body.
**If user declines or "skip":** Create with minimal body.

<!-- USER-EXTENSION-START: pre-create -->
<!-- USER-EXTENSION-END: pre-create -->

### Step 3: Create Issue
Build issue body with standard template:
```markdown
## Enhancement

**Description:**
{user description or "To be documented"}

**Motivation:**
{infer from description, or "To be documented"}

**Proposed Solution:**
{infer from description, or "To be documented"}

**Scope:**
- **In scope:** {infer from description, or "To be documented"}
- **Out of scope:** {infer from description, or "To be documented"}

**Deployment Impact:** {dev-only | deployed (list affected areas) | unknown}

**Acceptance Criteria:**
- [ ] {infer from description, or "To be documented"}
```
Populate from user's description where possible. Use "To be documented" only for sections without enough input.
```bash
gh pmu create --title "[Enhancement]: {title}" --label enhancement --status backlog --priority p2 --assignee @me -F .tmp-body.md
rm .tmp-body.md
```
Always use `-F .tmp-body.md` (never inline `--body`).
### Step 4: Report and STOP
```
Created: Issue #$ISSUE_NUM -- [Enhancement]: {title}
Status: Backlog
Label: enhancement

Say "/review-issue #$ISSUE_NUM" then "/assign-branch #$ISSUE_NUM" then "work #$ISSUE_NUM" to start working on this enhancement.
```

<!-- USER-EXTENSION-START: post-create -->
<!-- USER-EXTENSION-END: post-create -->

**STOP.** Do not begin work unless user explicitly says "work", "fix that", or "implement that".
## Error Handling
| Situation | Response |
|-----------|----------|
| No title provided | Prompt user for title |
| Empty title after prompt | "An enhancement title is required." -> STOP |
| `gh pmu create` fails | "Failed to create issue: {error}" -> STOP |
| Special characters in title | Escape for shell safety |
**End of /enhancement Command**
