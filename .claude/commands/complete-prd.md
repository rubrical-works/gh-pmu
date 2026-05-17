---
version: "v0.92.0"
description: Verify and close PRD tracker (project)
argument-hint: "<issue-number> (e.g., 151)"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /complete-prd
Verify all epics/stories from a PRD are complete, then close the PRD tracker.
---
## Arguments
| Argument | Description |
|----------|-------------|
| `<prd-issue-number>` | PRD tracker (e.g., `151` or `#151`) |
---
## Usage
```bash
/complete-prd 151
/complete-prd #151
```
---
## Prerequisites
- PRD tracker exists with `prd` label
- Created via `/create-backlog` (has linked epics)
---
## Workflow
### Step 1: Validate PRD Issue
```bash
issue_num="${1#\#}"
gh issue view $issue_num --json labels -q '.labels[].name' | grep -q "prd"
```
**If not PRD:**
```
Error: Issue #$issue_num does not have the 'prd' label.
This command requires a PRD tracker issue.
```
### Step 2: Find Linked Epics
```bash
gh issue list --label "epic" --state all --json number,title,body,state
```
Filter epics whose body contains `**PRD Tracker:** #{issue_num}`.

**None found:**
```
Warning: No epics found linked to PRD #{issue_num}.
Check that epics have "PRD Tracker: #{issue_num}" in their body.
```
### Step 3: Check Epic Completion
```bash
gh issue view #{epic} --json state -q '.state'
```
**Collect:** total, closed, open list.
### Step 4: Check Story Completion
```bash
gh pmu sub list #{epic} --json
```
**Collect:** total, closed, open list.
### Step 5: Report Status
**All complete:**
```
✅ PRD #{issue_num} Verification Complete

## Summary

Epics: {closed_epics}/{total_epics} complete
Stories: {closed_stories}/{total_stories} complete

All work items are complete. Closing PRD tracker.
```
Close:
```bash
gh pmu move $issue_num --status done
gh issue comment $issue_num --body "## PRD Complete ✅

All epics and stories have been completed.

**Final Summary:**
- Epics: {closed_epics}
- Stories: {closed_stories}

PRD closed by \`/complete-prd\` on {date}."
```
**Incomplete:**
```
⚠️ PRD #{issue_num} Not Ready for Closure

## Incomplete Work

### Open Epics ({open_epic_count})
- #{epic1}: {title}
- #{epic2}: {title}

### Open Stories ({open_story_count})
- #{story1}: {title} (Epic: #{epic})
- #{story2}: {title} (Epic: #{epic})

## Summary

Epics: {closed_epics}/{total_epics} complete
Stories: {closed_stories}/{total_stories} complete

Complete the above items before running /complete-prd again.
```
**Do NOT close** if incomplete.
### Step 6: Move Proposal to Implemented (After Closure)
**Only when Step 5 successfully closes PRD.**

**6a: Find source proposal** — extract from PRD body: `**Source Proposal:** #NNN`. Read proposal issue:
```bash
gh issue view $proposal_issue --json body --jq '.body'
```
Extract file path: `**File:** Proposal/[Name].md`.

**6b: Move proposal file**
```bash
mkdir -p Proposal/Implemented
git ls-files --error-unmatch Proposal/{Name}.md 2>/dev/null
git add Proposal/{Name}.md
git mv Proposal/{Name}.md Proposal/Implemented/{Name}.md
```
**6c: Edge cases**
| Situation | Response |
|-----------|----------|
| Already in `Proposal/Implemented/` | Skip (by `/create-prd` Phase 7). Non-blocking. |
| File not found at path | Warn: `"Proposal file not found: {path}. Skipping proposal move."` Continue. |
| No `**Source Proposal:**` in PRD body | Warn: `"No source proposal reference found in PRD tracker."` Continue. |
| Proposal issue not found/closed | Use body from closed issue (`gh issue view` works on closed). |
| `git mv` fails | Warn and continue — non-blocking. |

**6d: Commit if moved**
```bash
git add Proposal/Implemented/{Name}.md
git commit -m "Refs #$issue_num — move proposal to Implemented after PRD completion"
```
If nothing moved, skip commit.
---
## Verification Logic
```
PRD Complete = (ALL epics CLOSED) AND (ALL stories CLOSED)
```
| Epics | Stories | Result |
|-------|---------|--------|
| All closed | All closed | ✅ Close PRD |
| Any open | Any | ❌ Incomplete |
| All closed | Any open | ❌ Incomplete |
---
## Error Handling
| Situation | Response |
|-----------|----------|
| PRD issue not found | "Issue #N not found. Check the issue number?" |
| Missing prd label | "Issue #N does not have 'prd' label." |
| No linked epics | Warning + suggest manual verification |
| PRD already closed | "PRD #{N} is already closed." |
---
## Related
| Command | Purpose |
|---------|---------|
| `/create-backlog` | Creates PRD tracker with epics/stories |
| `/add-story` | Adds stories (updates PRD tracker) |
| `/split-story` | Splits stories (updates PRD tracker) |
---
**End of /complete-prd Command**
