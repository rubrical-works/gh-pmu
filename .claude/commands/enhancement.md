---
version: "v0.65.0"
description: Create an enhancement issue with standard template (project)
argument-hint: "<title>"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /enhancement

Creates a properly labeled enhancement issue with a standard template and adds it to the project board.

**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command enhancement`

---

## Prerequisites

- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root

---

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<title>` | No | Enhancement title (e.g., `add dark mode support`) |

If no title provided, prompt the user for one.

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

**If empty:** Ask the user for an enhancement title before proceeding.

**If title contains special characters** (backticks, quotes): Escape appropriately for the shell. On Windows, use temp file approach per shell safety rules.

### Step 2: Gather Description

Extract `<body>` from command arguments.

**IF** there is insufficient detail provided in the arguments to create the issue, **THEN**:

```
Describe the enhancement (what it does, why it's useful):
```

**If the user provides a description:** Use it as the issue body.
**If the user declines or says "skip":** Create with a minimal body.

<!-- USER-EXTENSION-START: pre-create -->
<!-- USER-EXTENSION-END: pre-create -->

### Step 3: Create Issue

Build the issue body with a standard enhancement template:

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

**Acceptance Criteria:**
- [ ] {infer from description, or "To be documented"}
```

Populate sections from the user's description where possible. Use "To be documented" placeholders only for sections without enough input.

Create the issue:
```bash
gh pmu create --title "[Enhancement]: {title}" --label enhancement --status backlog --priority p2 --assignee @me -F .tmp-body.md
rm .tmp-body.md
```

**Note:** Always use `-F .tmp-body.md` for the body (never inline `--body`).

### Step 4: Report and STOP

```
Created: Issue #$ISSUE_NUM — [Enhancement]: {title}
Status: Backlog
Label: enhancement

Say "/review-issue #$ISSUE_NUM" then "/assign-branch #$ISSUE_NUM" then "work #$ISSUE_NUM" to start working on this enhancement.
```

<!-- USER-EXTENSION-START: post-create -->
<!-- USER-EXTENSION-END: post-create -->

**STOP.** Do not begin work unless the user explicitly says "work", "fix that", or "implement that".

---

## Error Handling

| Situation | Response |
|-----------|----------|
| No title provided | Prompt user for title |
| Empty title after prompt | "An enhancement title is required." → STOP |
| `gh pmu create` fails | "Failed to create issue: {error}" → STOP |
| Special characters in title | Escape for shell safety |

---

**End of /enhancement Command**
