---
version: "v0.91.0"
description: Review stories for direction change (project)
argument-hint: "[epic-number|prd-name]"
copyright: "Rubrical Works (c) 2026"
---

<!-- MANAGED -->
# /pivot

Review and triage stories when project direction changes: keep, archive, or close each story.

## Arguments

| Argument | Description |
|----------|-------------|
| `[epic-number]` | Epic issue number (e.g., `42` or `#42`) |
| `[prd-name]` | PRD name to pivot all related stories |

## Usage

```bash
/pivot 42           # Pivot stories under Epic #42
/pivot Auth-System  # Pivot all stories from Auth-System PRD
/pivot              # Interactive - prompts for selection
```

## Workflow

### Phase 1: Identify Scope

**Step 1: Determine pivot target**

If argument provided: number → epic; text → PRD name.

If no argument, **ASK USER:** What would you like to pivot? (1) epic number, (2) PRD name.

**Step 2: Validate target**

Epic:
```bash
gh issue view $epic_num --json labels --jq '.labels[].name' | grep -q "epic"
```

PRD:
```bash
ls PRD/*/$1* 2>/dev/null || ls PRD/$1* 2>/dev/null
```

### Phase 2: Document Pivot Reason

**ASK USER:** What is the new direction or reason for the pivot? Record for documentation.

### Phase 3: List Stories

Epic:
```bash
gh pmu sub list $epic_num --json number,title,state
```

PRD:
```bash
gh issue list --label "story" --search "PRD:$prd_name" --json number,title,state
```

Display as table (#, Issue, Title, Status).

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
```

Record decision for each story.

### Phase 5: Apply Actions

| Decision | Action |
|----------|--------|
| Keep | No change, note in summary |
| Archive | `gh pmu move #{num} --status parking_lot` |
| Close | `gh issue close #{num} --reason "not planned" --comment "Closed during pivot: {reason}"` |
| Skip | No change, include in "pending review" |

**Document pivot on parent:**

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
  Kept: {count}
  Archived: {count}
  Closed: {count}
  Pending: {count}
Documentation added to #{parent}
Next steps:
1. Review kept stories for priority adjustment
2. Continue work: work #{next_story}
```

## When to Use

Use `/pivot` when project requirements change significantly, direction shifts, scope needs realignment, or technical constraints require rethinking.

**Not for:** minor priority changes (`gh pmu move --priority`), single story updates, adding stories (`/add-story`).

## Error Handling

| Situation | Response |
|-----------|----------|
| Epic not found | "Issue #N not found or not an epic." |
| PRD not found | "No PRD found matching '{name}'." |
| No open stories | "No open stories found to review." |
| User cancels | "Pivot cancelled. No changes made." |

**End of /pivot Command**
