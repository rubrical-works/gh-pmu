---
version: "v0.77.4"
description: Review stories for direction change (project)
argument-hint: "[epic-number|prd-name]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /pivot
Review and triage stories when project direction changes. Manages scope realignment by deciding to keep, archive, or close each story.

| Argument | Description |
|----------|-------------|
| `[epic-number]` | Epic issue number to pivot (e.g., `42` or `#42`) |
| `[prd-name]` | PRD name to pivot all related stories |

**Usage:** `/pivot 42`, `/pivot #42`, `/pivot Auth-System`, `/pivot` (interactive)
## Workflow
### Phase 1: Identify Scope
**Step 1: Determine pivot target**
- Number argument -> epic issue number
- Text argument -> PRD name
- No argument -> **ASK USER:** Epic number or PRD name?

**Step 2: Validate target**
For epic:
```bash
gh issue view $epic_num --json labels --jq '.labels[].name' | grep -q "epic"
```
For PRD:
```bash
ls PRD/*/$1* 2>/dev/null || ls PRD/$1* 2>/dev/null
```
### Phase 2: Document Pivot Reason
**ASK USER:** What is the new direction or reason for the pivot?
### Phase 3: List Stories
For epic:
```bash
gh pmu sub list $epic_num --json number,title,state
```
For PRD:
```bash
gh issue list --label "story" --search "PRD:$prd_name" --json number,title,state
```
Display as table with #, Issue, Title, Status.
### Phase 4: Review Each Story
For each open story, present:
```
Story #101: User login
Status: In Progress
Options:
1. Keep - Aligns with new direction
2. Archive - May be relevant later (parking lot)
3. Close - No longer needed (not planned)
4. Skip - Decide later
```
Record decision for each story.
### Phase 5: Apply Actions
| Decision | Action |
|----------|--------|
| Keep | No change, note in summary |
| Archive | `gh pmu move #{num} --status parking_lot` |
| Close | `gh issue close #{num} --reason "not planned" --comment "Closed during pivot: {reason}"` |
| Skip | No change, include in "pending review" |

Document pivot on parent -- add comment to epic/PRD issue with date, reason, and story decision table.
### Phase 6: Report Summary
```
Pivot complete: Epic #{num} / PRD {name}
Reason: {pivot_reason}
Stories reviewed: {total}
  Kept: {count}, Archived: {count}, Closed: {count}, Pending: {count}
Documentation added to #{parent}
Next steps:
1. Review kept stories for priority adjustment
2. Continue work: work #{next_story}
```
## When to Use
- Project requirements significantly change
- Market/business direction shifts
- Scope needs realignment
- Technical constraints require rethinking

**Not for:** Minor priority changes (`gh pmu move --priority`), single story updates, adding stories (`/add-story`).
## Error Handling
| Situation | Response |
|-----------|----------|
| Epic not found | "Issue #N not found or not an epic." |
| PRD not found | "No PRD found matching '{name}'." |
| No open stories | "No open stories found to review." |
| User cancels | "Pivot cancelled. No changes made." |
**End of /pivot Command**
