---
version: "v0.67.1"
description: Reset bug/enhancement/prd/proposal/epic issue to clean slate (project)
argument-hint: "#issue [--dry-run]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /issue-reset
Reset a bug, enhancement, PRD, proposal, or epic issue to a clean slate. For epics, recursively resets all sub-issues.
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number to reset |
| `--dry-run` | No | Preview without executing |
---
## Workflow
### Step 1: Validate Issue Type
```bash
gh pmu view $ISSUE --json=number,title,labels,body,state
```
**Allowed:** `bug`, `enhancement`, `prd`, `proposal`, `epic`
**Rejected:** `story`, `branch`, or other → STOP
### Step 1a: Epic Detection
If `epic` label: fetch sub-issues via `gh pmu sub list $ISSUE`.
### Step 2: Analyze Reset Scope
| Item | Detection |
|------|-----------|
| AC checkboxes | Count `[x]` boxes |
| Reviews counter | Parse `**Reviews:** N` |
| Auto-generated sections | `#### Proposed Solution`, `#### Proposed Fix` |
| Review comments | `## Issue Review`, `## Proposal Review`, `## PRD Review` |
| `reviewed` label | Labels array |
| Test plan files | `Construction/Test-Plans/` referencing `#$ISSUE` |
**Proposal:** Also check associated PRD issue and conversion status.
**Epic:** Also count sub-issues and their reset scope.
### Step 3: Dry-Run or Confirmation
`--dry-run`: Report what would be reset → STOP.
Otherwise: `AskUserQuestion` — "Proceed with reset?" (include sub-issue count for epics). Decline → STOP.
### Step 4: Execute Reset
**4a:** Reset body — uncheck `[x]`→`[ ]`, reset `**Reviews:**` to 0, remove auto-generated sections.
**4b:** `gh pmu move $ISSUE --status backlog`
**4c:** Delete review comments matching patterns via `gh api -X DELETE`.
**4d:** `gh issue edit $ISSUE --remove-label reviewed` (skip if absent).
**4e:** `git rm` test plan files referencing issue.
**4f:** Proposal PRD cleanup — delete or warn if converted to backlog.
**4g:** Epic recursive reset — apply 4a–4e to each sub-issue, then to epic itself.
### Step 5: Commit Changes
If files deleted: `git commit -m "Refs #$ISSUE — reset issue artifacts"`
### Step 6: Post Audit Comment
```bash
gh issue comment $ISSUE --body "Issue reset by /issue-reset on $DATE. Removed: $N review comments, $M test plan files. Status: backlog."
```
### Step 7: Report Completion
Report actions taken (checkboxes, reviews, comments, labels, files, status, sub-issues if epic). **STOP.**
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | STOP |
| Invalid type | STOP |
| User declines | STOP |
| No review comments | Skip, report 0 |
| No test plans | Skip, report 0 |
| git rm fails | Continue |
| PRD converted | Warn, ask disposition |
| Epic no sub-issues | Reset epic only |
---
**End of /issue-reset Command**
