---
version: "v0.91.0"
description: Create a bug issue with standard template (project)
argument-hint: "<title>"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /bug
Create a labeled bug issue with standard template and add to project board.
**Extension Points:** `.claude/metadata/extension-points.json` or `/extensions list --command bug`
## Prerequisites
- `gh pmu` installed, `.gh-pmu.json` configured
## Arguments
| Argument | Description |
|----------|-------------|
| `<title>` | Bug title (e.g., `assign-branch fails on Windows paths`) |

If not provided, prompt user.
## Execution
**REQUIRED before executing:**
1. Use `TaskCreate` for one task per step below. No routing → bulk create upfront (see rule `07-task-creation-timing.md`).
2. Include one task per active (non-empty) `USER-EXTENSION` block.
3. Mark tasks `in_progress` → `completed` via `TaskUpdate`.
4. **Post-Compaction:** re-read spec, call `TaskList`, resume from first incomplete task.
## Workflow
### Step 1: Parse Arguments
Extract `<title>`.
**Empty:** Ask user before proceeding.
**Special chars** (backticks, quotes): Escape for shell. On Windows, use temp file per shell safety.
### Step 2: Gather Description
Extract `<body>` from args.
**IF insufficient detail**, THEN:
```
Describe the bug (steps to reproduce, expected vs actual behavior):
```
**Description provided:** use as body. **Declined/"skip":** minimal body.
### Step 2b: Detect Version
Priority: `package.json` → `version` | git tag (`git describe --tags --abbrev=0`) | prompt user.
**If detected**, confirm via `AskUserQuestion` with "Yes, use {version}" (default) / "No, let me specify".
**Override provided:** use it.

<!-- USER-EXTENSION-START: pre-create -->
<!-- USER-EXTENSION-END: pre-create -->

### Step 3: Create Issue
Body template:
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

**Deployment Impact:** {dev-only | deployed (list affected areas) | unknown}

**Acceptance Criteria:**
- [ ] {infer from description, or "To be documented"}

**Proposed Fix:**
{infer from description if enough context, or "To be documented"}
```
Populate from user input where possible. Use "To be documented" only where insufficient.

Create:
```bash
gh pmu create --title "[Bug]: {title}" --label bug --status backlog --priority p1 --assignee @me -F .tmp-body.md
rm .tmp-body.md
```
**Note:** Always `-F .tmp-body.md` (never inline `--body`).
### Step 4: Report and STOP
```
Created: Issue #$ISSUE_NUM — [Bug]: {title}
Status: Backlog
Label: bug

Say "/review-issue #$ISSUE_NUM" then "/assign-branch #$ISSUE_NUM" then "work #$ISSUE_NUM" to start working on this bug.
```

<!-- USER-EXTENSION-START: post-create -->
<!-- USER-EXTENSION-END: post-create -->

**STOP.** Do NOT begin work unless user says "work", "fix that", or "implement that".
## Error Handling
| Situation | Response |
|-----------|----------|
| No title | Prompt user |
| Empty after prompt | "A bug title is required." → STOP |
| `gh pmu create` fails | "Failed to create issue: {error}" → STOP |
| Special chars | Escape for shell safety |

**End of /bug Command**
