---
version: "v0.81.1"
description: Verify and close PRD tracker (project)
argument-hint: "<issue-number> (e.g., 151)"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /complete-prd
Verify all epics and stories derived from a PRD are complete, then close the PRD tracker issue.
## Arguments
| Argument | Description |
|----------|-------------|
| `<prd-issue-number>` | PRD tracker issue number (e.g., `151` or `#151`) |
## Prerequisites
- PRD tracker issue exists with `prd` label
- PRD was created via `/create-backlog` (has linked epics)
## Workflow
### Step 1: Validate PRD Issue
```bash
issue_num="${1#\#}"
gh issue view $issue_num --json labels -q '.labels[].name' | grep -q "prd"
```
If not a PRD issue: `Error: Issue #$issue_num does not have the 'prd' label.`
### Step 2: Find Linked Epics
```bash
gh issue list --label "epic" --state all --json number,title,body,state
```
Filter to epics whose body contains `**PRD Tracker:** #{issue_num}`.
If no epics found: warn and suggest manual verification.
### Step 3: Check Epic Completion
For each linked epic:
```bash
gh issue view #{epic} --json state -q '.state'
```
Collect total/closed/open epic counts.
### Step 4: Check Story Completion
For each linked epic, get sub-issues:
```bash
gh pmu sub list #{epic} --json
```
Collect total/closed/open story counts.
### Step 5: Report Status
**If all complete:** Report summary, then close:
```bash
gh pmu move $issue_num --status done
gh issue comment $issue_num --body "## PRD Complete
All epics and stories completed.
**Final Summary:** Epics: {closed_epics}, Stories: {closed_stories}
PRD closed by \`/complete-prd\` on {date}."
```
**If incomplete:** Report open epics and stories with counts. **Do NOT close.**
### Step 6: Move Proposal to Implemented (After Closure)
**Only runs when Step 5 successfully closes the PRD tracker.**
**Step 6a: Find the source proposal**
Extract from PRD tracker body: `**Source Proposal:** #NNN`
If found, read proposal issue to get file path: `**File:** Proposal/[Name].md`
**Step 6b: Move proposal file**
```bash
mkdir -p Proposal/Implemented
git ls-files --error-unmatch Proposal/{Name}.md 2>/dev/null
git add Proposal/{Name}.md
git mv Proposal/{Name}.md Proposal/Implemented/{Name}.md
```
**Step 6c: Handle edge cases**
| Situation | Response |
|-----------|----------|
| Already in `Proposal/Implemented/` | Skip (non-blocking) |
| File not found | Warn and continue (non-blocking) |
| No `**Source Proposal:**` in body | Warn and continue (non-blocking) |
| Proposal issue closed | Use closed issue body (works with `gh issue view`) |
| `git mv` fails | Warn and continue (non-blocking) |
**Step 6d: Include in commit**
```bash
git add Proposal/Implemented/{Name}.md
git commit -m "Refs #$issue_num -- move proposal to Implemented after PRD completion"
```
If no proposal moved, skip commit.
## Verification Logic
```
PRD Complete = (ALL epics CLOSED) AND (ALL stories CLOSED)
```
| Epics | Stories | Result |
|-------|---------|--------|
| All closed | All closed | Close PRD |
| Any open | Any state | Report incomplete |
| All closed | Any open | Report incomplete |
## Error Handling
| Situation | Response |
|-----------|----------|
| PRD issue not found | "Issue #N not found. Check the issue number?" |
| Issue missing prd label | "Issue #N does not have 'prd' label." |
| No linked epics found | Warning + suggest manual verification |
| PRD already closed | "PRD #{N} is already closed." |
## Related Commands
| Command | Purpose |
|---------|---------|
| `/create-backlog` | Creates PRD tracker with linked epics/stories |
| `/add-story` | Adds stories (updates PRD tracker) |
| `/split-story` | Splits stories (updates PRD tracker) |
**End of /complete-prd Command**
