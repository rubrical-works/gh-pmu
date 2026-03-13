---
version: "v0.62.0"
description: Review stories for direction change (project)
argument-hint: "[epic-number|prd-name]"
---
<!-- MANAGED -->

# /pivot
Review and triage stories when project direction changes. Helps manage scope realignment by reviewing each story and deciding whether to keep, archive, or close it.

## Arguments
| Argument | Description |
|----------|-------------|
| `[epic-number]` | Epic issue number to pivot (e.g., `42` or `#42`) |
| `[prd-name]` | PRD name to pivot all related stories |

## Usage
```bash
/pivot 42           # Pivot stories under Epic #42
/pivot #42          # Same, with hash prefix
/pivot Auth-System  # Pivot all stories from Auth-System PRD
/pivot              # Interactive - prompts for selection
```

## Workflow

### Phase 1: Identify Scope
**Step 1: Determine pivot target**
If argument provided:
- Number → Treat as epic issue number
- Text → Treat as PRD name
If no argument:
**ASK USER:** What would you like to pivot?
1. An epic (enter issue number)
2. A PRD's stories (enter PRD name)
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
Record response for documentation.

### Phase 3: List Stories
**Step 1: Gather stories**
For epic:
```bash
gh pmu sub list $epic_num --json number,title,state
```
For PRD:
```bash
gh issue list --label "story" --search "PRD:$prd_name" --json number,title,state
```
**Step 2: Display stories**
```
Stories to review:

| # | Issue | Title | Status |
|---|-------|-------|--------|
| 1 | #101 | User login | In Progress |
| 2 | #102 | Password reset | Backlog |
| 3 | #103 | OAuth integration | Backlog |
```

### Phase 4: Review Each Story
For each open story, present options:
```
Story #101: User login
Status: In Progress

Options:
1. Keep - Story aligns with new direction
2. Archive - May be relevant later (parking lot)
3. Close - No longer needed (close as not planned)
4. Skip - Decide later

Your choice: ___
```
**Record decision for each story.**

### Phase 5: Apply Actions
**Step 1: Process decisions**
| Decision | Action |
|----------|--------|
| Keep | No change, note in summary |
| Archive | `gh pmu move #{num} --status parking_lot` |
| Close | `gh issue close #{num} --reason "not planned" --comment "Closed during pivot: {reason}"` |
| Skip | No change, include in "pending review" |
**Step 2: Document pivot on parent**
Add comment to epic/PRD issue:
```bash
gh issue comment $parent --body "## Pivot: {date}

**Reason:** {pivot_reason}

### Story Decisions

| Story | Decision |
|-------|----------|
| #101 | Keep |
| #102 | Archive |
| #103 | Close |

{count} kept, {count} archived, {count} closed"
```

### Phase 6: Report Summary
```
Pivot complete: Epic #{num} / PRD {name}

Reason: {pivot_reason}

Stories reviewed: {total}
  ✅ Kept: {count}
  📦 Archived: {count}
  ❌ Closed: {count}
  ⏸️ Pending: {count}

Documentation added to #{parent}

Next steps:
1. Review kept stories for priority adjustment
2. Continue work: work #{next_story}
```

## When to Use
Use `/pivot` when:
- Project requirements significantly change
- Market/business direction shifts
- Scope needs realignment
- Technical constraints require rethinking
**Not for:**
- Minor priority changes (use `gh pmu move --priority`)
- Single story updates (edit directly)
- Adding new stories (use `/add-story`)

## Error Handling
| Situation | Response |
|-----------|----------|
| Epic not found | "Issue #N not found or not an epic." |
| PRD not found | "No PRD found matching '{name}'." |
| No open stories | "No open stories found to review." |
| User cancels | "Pivot cancelled. No changes made." |
**End of /pivot Command**
