---
version: "v0.86.0"
description: Create an enhancement issue with standard template (project)
argument-hint: "<title>"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /enhancement
Create a labeled enhancement issue with standard template and add to project board.
**Extension Points:** `.claude/metadata/extension-points.json` or `/extensions list --command enhancement`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
---
## Arguments
| Argument | Description |
|----------|-------------|
| `<title>` | Enhancement title (e.g., `add dark mode`) |

If not provided, prompt user.
---
## Execution
**REQUIRED before executing:**
1. Use `TaskCreate` for one task per step below. No routing → bulk create upfront (see rule `07-task-creation-timing.md`).
2. Include one task per active (non-empty) `USER-EXTENSION` block.
3. Mark tasks `in_progress` → `completed` via `TaskUpdate`.
4. **Post-Compaction:** re-read spec, call `TaskList`, resume from first incomplete task.
---
## Workflow
### Step 1: Parse Arguments
Extract `<title>`.
**Empty:** Ask user before proceeding.
**Special chars** (backticks, quotes): Escape for shell. On Windows, use temp file per shell safety.
### Step 2: Gather Description
Extract `<body>` from args.
**IF insufficient detail**, THEN:
```
Describe the enhancement (what it does, why it's useful):
```
**Description provided:** use as body. **Declined/"skip":** minimal body.

<!-- USER-EXTENSION-START: pre-create -->
<!-- USER-EXTENSION-END: pre-create -->

### Step 3: Create Issue
Body template:
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
Populate from user input where possible. Use "To be documented" only where insufficient.

Create:
```bash
gh pmu create --title "[Enhancement]: {title}" --label enhancement --status backlog --priority p2 --assignee @me -F .tmp-body.md
rm .tmp-body.md
```
**Note:** Always `-F .tmp-body.md` (never inline `--body`).
### Step 4: Report and STOP
```
Created: Issue #$ISSUE_NUM — [Enhancement]: {title}
Status: Backlog
Label: enhancement

Say "/review-issue #$ISSUE_NUM" then "/assign-branch #$ISSUE_NUM" then "work #$ISSUE_NUM" to start working on this enhancement.
```

<!-- USER-EXTENSION-START: post-create -->
<!-- USER-EXTENSION-END: post-create -->

**STOP.** Do NOT begin work unless user says "work", "fix that", or "implement that".
---
## Error Handling
| Situation | Response |
|-----------|----------|
| No title | Prompt user |
| Empty after prompt | "An enhancement title is required." → STOP |
| `gh pmu create` fails | "Failed to create issue: {error}" → STOP |
| Special chars | Escape for shell safety |
---
**End of /enhancement Command**
