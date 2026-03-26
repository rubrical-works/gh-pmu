---
version: "v0.72.0"
description: Split story into smaller stories (project)
argument-hint: "<story-number> (e.g., 123)"
copyright: "Rubrical Works (c) 2026"
---

<!-- MANAGED -->
# /split-story

Split a story into smaller, more manageable stories while maintaining charter compliance and test plan integrity.

---

## Arguments

| Argument | Description |
|----------|-------------|
| `<story-number>` | Story issue number to split (e.g., `123` or `#123`) |

---

## Execution Instructions

**REQUIRED:** Before executing this command:

1. **Create Todo List:** Use `TodoWrite` to create todos from the steps below
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
3. **Resume Point:** If interrupted, todos show where to continue

**Example todo structure:**
```
- [ ] Phase 1: Fetch and validate original story
- [ ] Phase 2: Determine split criteria
- [ ] Phase 3: Charter compliance check
- [ ] Phase 4: Create new stories
- [ ] Phase 5: Update original story
- [ ] Phase 6: Update test plan
- [ ] Phase 7: Report completion
```

---

## Phase 1: Fetch and Validate Original Story

**Step 1: Parse story number**

Accept `123` or `#123` format:
```bash
# Strip leading # if present
story_num="${1#\#}"
```

**Step 2: Fetch and validate story**

```bash
gh issue view $story_num --json labels,body,title --jq '.labels[].name' | grep -q "story"
```

**If not a story:**
```
Error: Issue #$story_num does not have the 'story' label.
This command requires a story issue to split.
```

**Step 3: Extract story details**

```bash
gh pmu view $story_num --body-stdout > .tmp-story.md
```

Parse from story body:
- Title
- Description (As a... I want... So that...)
- Acceptance criteria (checkbox list)
- Priority
- Parent epic reference

**Step 4: Find parent epic**

```bash
gh pmu sub list --child $story_num --json parent
```

Or parse from story body for `Parent Epic: #N` reference.

---

## Phase 2: Determine Split Criteria

**ASK USER:** How should this story be split?

**Common split patterns:**

| Pattern | Description |
|---------|-------------|
| By acceptance criteria | Each criterion becomes a story |
| By user workflow | Split by distinct user actions |
| By technical component | Split by system area (frontend/backend/API) |
| By priority | Separate must-have from nice-to-have |
| Custom | User defines the split |

**Gather split details:**

For each new story:
1. Title
2. Which acceptance criteria it covers
3. Priority (inherit or override)

**Minimum:** 2 new stories required for a valid split.

---

## Phase 3: Charter Compliance Check

**Step 1: Load charter context**

| File | Required | Purpose |
|------|----------|---------|
| `CHARTER.md` | Recommended | Project vision, goals, scope |
| `Inception/Scope-Boundaries.md` | Optional | In/out of scope boundaries |
| `Inception/Constraints.md` | Optional | Technical/business constraints |

**If no charter exists:**
```
⚠️ No CHARTER.md found. Skipping compliance check.
```

**Step 2: Validate split stories against charter**

Check each new story for:
- Scope creep (split introduces out-of-scope work)
- Constraint violations
- Goal alignment

**Step 3: Report compliance**

**If all aligned:**
```
✅ Split stories align with charter scope
   - All stories within project scope
   - No constraint violations detected
```

**If scope concern:**
```
⚠️ Potential scope concern in split:
   - New story "{title}" mentions: "{concerning element}"
   - This wasn't in the original story
   - Charter constraint: "{relevant constraint}"

Proceed anyway? (yes/no)
```

**ASK USER:** Confirm to proceed if concerns found.

---

## Phase 4: Create New Stories

For each new story from the split:

**Step 1: Create story issue**

```bash
gh pmu create --repo {repository} \
  --title "Story: {New Story Title}" \
  --label "story" \
  --body "{story_body}" \
  --status backlog \
  --priority {priority} \
  --assignee @me
```

**Story Body Template:**

```markdown
## Story: {Title}

### Description

As a {user type}, I want {capability} so that {benefit}.

### Relevant Skills

<!-- Read from framework-config.json projectSkills array -->
<!-- For each skill, lookup description from .claude/metadata/skill-registry.json -->

**If projectSkills configured:**
- {skill-name} - {description from skill-registry.json}
- {skill-name} - {description}

Load skill: `read Skills/{skill-name}/SKILL.md`

**If no projectSkills:**
No project skills configured. Run `/charter` to set up project-specific skills.

### Acceptance Criteria

- [ ] {Assigned criterion 1}
- [ ] {Assigned criterion 2}

### Documentation (if applicable)

- [ ] Design decisions documented (update existing or create `Construction/Design-Decisions/YYYY-MM-DD-{topic}.md`)
- [ ] Tech debt logged (update existing or create `Construction/Tech-Debt/YYYY-MM-DD-{topic}.md`)

**Guidelines:** Skip trivial findings. Update existing docs rather than duplicating. For significant tech debt, create an enhancement issue.

### Origin

Split from #{original_story_num}: {Original Story Title}

### TDD Test Cases

Test cases inherited from original story (see test plan).

### Definition of Done

- [ ] All acceptance criteria met
- [ ] TDD test cases pass
- [ ] Code reviewed
- [ ] No regressions

**Priority:** {P0|P1|P2}
**Parent Epic:** #{epic_num}
```

**Step 2: Link to parent epic**

```bash
gh pmu sub add {epic_num} {new_story_num} || true
```

**Step 3: Track created stories**

Collect all new story numbers for reporting.

---

## Phase 5: Update Original Story

**Step 1: Update original story body**

```bash
gh pmu view $story_num --body-stdout > .tmp-original.md
```

Add split reference section:

```markdown
---

## Split Notice

This story was split into smaller stories:
- #{new_story_1}: {Title 1}
- #{new_story_2}: {Title 2}

**Reason:** {User's split rationale}
**Date:** {date}

This issue is now closed. Work the split stories instead.
```

**Step 2: Save updated body**

```bash
gh pmu edit $story_num -F .tmp-original.md
rm .tmp-original.md
```

**Step 3: Close original story**

```bash
gh issue close $story_num --comment "Split into: #{new_1}, #{new_2}, ...

Work the split stories instead."
```

---

## Phase 6: Update Test Plan

**Step 1: Find relevant test plan**

Check epic for PRD reference:
```bash
gh issue view $epic_num --json body --jq '.body' | grep -oE "PRD/[A-Za-z0-9_-]+/PRD-[A-Za-z0-9_-]+\.md"
```

**If PRD found, derive test plan path:**
```
PRD/{name}/PRD-{name}.md → PRD/{name}/Test-Plan-{name}.md
```

**If no test plan found:**
```
ℹ️ No test plan found for this epic.
   Test cases will be managed when story work begins.
```
Skip to Phase 7.

**Step 2: Redistribute test cases**

In test plan, find section for original story:
```markdown
### Story: {Original Title} (#{original_num})
```

**Step 3: Update test plan structure**

Replace original story section with split stories:

```markdown
### Story: {Original Title} (#{original_num}) - SPLIT

**Split into:**
- #{new_story_1}: {Title 1}
- #{new_story_2}: {Title 2}

---

### Story: {New Story 1 Title} (#{new_story_1})

| Acceptance Criteria | Test Cases |
|--------------------|------------|
| {Assigned criterion} | ✓ Test valid input |
|                      | ✓ Test invalid input |

---

### Story: {New Story 2 Title} (#{new_story_2})

| Acceptance Criteria | Test Cases |
|--------------------|------------|
| {Assigned criterion} | ✓ Test valid scenario |
|                      | ✓ Test error handling |
```

**Step 4: Commit test plan changes**

```bash
git add PRD/{name}/Test-Plan-{name}.md
git commit -m "docs: split test cases for Story #{original_num}

Split into: #{new_1}, #{new_2}
Refs #{epic_num}"
```

---

## Phase 7: Update PRD Tracker (if applicable)

**Step 1: Check for PRD Tracker reference in epic**

```bash
gh issue view $epic_num --json body --jq '.body' | grep -oE "\*\*PRD Tracker:\*\* #[0-9]+"
```

**If PRD Tracker found:**

Extract PRD issue number and add comment:

```bash
gh issue comment $prd_num --body "✂️ **Story Split**

Original: #{original_num} - {Original Title}
Split into:
- #{new_story_1}: {Title 1}
- #{new_story_2}: {Title 2}

Epic: #{epic_num}

Split via \`/split-story\`"
```

**If no PRD Tracker:**
Skip this step (epic is not PRD-derived).

---

## Phase 8: Report Completion

```
Story split complete: #{original_num} → {count} stories

Original story: #{original_num} - {Original Title} (CLOSED)

New stories created:
  • #{new_story_1}: {Title 1} (Priority: {P})
  • #{new_story_2}: {Title 2} (Priority: {P})

Parent epic: #{epic_num} - {Epic Title}

Charter compliance: ✅ All stories aligned (or ⚠️ Proceeded with warning)

Test plan: {Updated|Not applicable}
PRD tracker: {Updated #{prd_num}|Not PRD-derived}

Next steps:
1. Work a split story: work #{new_story_1}
2. View epic progress: gh pmu sub list #{epic_num}
```

---

## Error Handling

| Situation | Response |
|-----------|----------|
| Story not found | "Issue #N not found. Check the issue number?" |
| Issue not a story | "Issue #N does not have 'story' label." |
| No parent epic found | "Could not find parent epic. Link manually after split." |
| Less than 2 stories in split | "Split requires at least 2 new stories." |
| Charter concern, user declines | "Story split cancelled due to scope concerns." |
| Test plan not found | Proceed without test plan update (note in output) |
| Original story already closed | "Story #N is already closed. Cannot split closed stories." |

---

**End of /split-story Command**
