---
version: "v0.85.0"
description: Create a bug issue with standard template (project)
argument-hint: "<title>"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /bug
Creates a properly labeled bug issue with a standard template and adds it to the project board.

**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command bug`
## Prerequisites
- `gh pmu` installed
- `.gh-pmu.json` in repo root
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `<title>` | No | Bug title (e.g., `assign-branch fails on Windows paths`) |

If no title, prompt.
## Execution Instructions
**REQUIRED:**
1. **Generate Todo List:** `TodoWrite` from workflow steps
2. **Include Extensions:** Add a todo per non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark `in_progress` → `completed`
4. **Post-Compaction:** Re-read spec and regenerate todos

**Todo Rules:**
- One todo per numbered step
- One todo per active extension point
- Skip commented-out extensions
- Use step name as todo content
## Workflow
### Step 1: Parse Arguments
Extract `<title>`. **If empty:** ask the user for one.

**If title contains special characters** (backticks, quotes): escape for shell. On Windows, use temp file approach per shell safety rules.
### Step 2: Gather Description
Extract `<body>` from command arguments.

**IF** insufficient detail in arguments, **THEN** ask the user:
```
Describe the bug (steps to reproduce, expected vs actual behavior):
```
**If description provided:** Use as issue body.
**If declined or "skip":** Create with minimal body.
### Step 2b: Detect Version
Priority:
1. `package.json` → `version` (Node.js)
2. Latest git tag (`git describe --tags --abbrev=0`)
3. If none, prompt: `"Which version was this bug found in?"`

**If detected**, confirm via `AskUserQuestion`:
- Question: `"Detected version: {detected-version}. Is this correct?"`
- Options: `"Yes, use {detected-version}"` (default), `"No, let me specify"`
- If "No", ask conversationally

**If override provided**, use it.

<!-- USER-EXTENSION-START: pre-create -->
<!-- USER-EXTENSION-END: pre-create -->

### Step 3: Create Issue
Build the issue body with standard bug template:
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
- [ ] {infer from description — e.g., "Bug no longer reproduces following the Steps to Reproduce", or "To be documented"}

**Proposed Fix:**
{infer from description if enough context, or "To be documented"}
```
Populate from description where possible. Use "To be documented" placeholders only when lacking input.

Create:
```bash
gh pmu create --title "[Bug]: {title}" --label bug --status backlog --priority p1 --assignee @me -F .tmp-body.md
rm .tmp-body.md
```
**Note:** Always use `-F .tmp-body.md` (never inline `--body`).
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
| No title provided | Prompt user |
| Empty title after prompt | "A bug title is required." → STOP |
| `gh pmu create` fails | "Failed to create issue: {error}" → STOP |
| Special characters in title | Escape for shell safety |

**End of /bug Command**
