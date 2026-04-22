---
version: "v0.90.0"
description: Split story into smaller stories (project)
argument-hint: "<story-number> (e.g., 123)"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /split-story
Split a story into smaller stories while maintaining charter compliance and test plan integrity.
---
## Arguments
| Argument | Description |
|----------|-------------|
| `<story-number>` | Story to split (e.g., `123` or `#123`) |
---
## Execution
**REQUIRED:**
1. `TaskCreate` from phases below
2. Mark `in_progress` → `completed`
3. Interrupted → tasks show resume point

**Example:**
```
- [ ] Phase 1: Fetch and validate
- [ ] Phase 2: Determine split criteria
- [ ] Phase 3: Charter compliance check
- [ ] Phase 4: Create new stories
- [ ] Phase 5: Update original
- [ ] Phase 6: Update test plan
- [ ] Phase 7: Report completion
```
---
## Phase 1: Fetch and Validate
**Step 1: Parse**
```bash
story_num="${1#\#}"
```
**Step 2: Fetch + validate**
```bash
gh issue view $story_num --json labels,body,title --jq '.labels[].name' | grep -q "story"
```
**Not a story:**
```
Error: Issue #$story_num does not have the 'story' label.
This command requires a story issue to split.
```
**Step 3: Extract details**
```bash
gh pmu view $story_num --body-stdout > .tmp-story.md
```
Parse: title, description (As/I want/So that), AC (checkboxes), priority, parent epic.
**Step 4: Parent epic**
```bash
gh pmu sub list --child $story_num --json parent
```
Or parse body for `Parent Epic: #N`.
---
## Phase 2: Split Criteria
**ASK USER:** How should this story be split?
| Pattern | Description |
|---------|-------------|
| By acceptance criteria | Each criterion → story |
| By user workflow | Distinct user actions |
| By technical component | Frontend/backend/API |
| By priority | Must-have vs nice-to-have |
| Custom | User defines |

**Per new story:** title, criteria covered, priority (inherit or override).
**Minimum 2 new stories required.**
---
## Phase 3: Charter Compliance
**Step 1: Load context**
| File | Required | Purpose |
|------|----------|---------|
| `CHARTER.md` | Recommended | Vision, goals, scope |
| `Inception/Scope-Boundaries.md` | Optional | In/out of scope |
| `Inception/Constraints.md` | Optional | Constraints |

**No charter:**
```
⚠️ No CHARTER.md found. Skipping compliance check.
```
**Step 2: Validate**
Check for scope creep, constraint violations, goal alignment.
**Step 3: Report**
**Aligned:**
```
✅ Split stories align with charter scope
   - All stories within project scope
   - No constraint violations detected
```
**Scope concern:**
```
⚠️ Potential scope concern in split:
   - New story "{title}" mentions: "{concerning element}"
   - This wasn't in the original story
   - Charter constraint: "{relevant constraint}"

Proceed anyway? (yes/no)
```
**ASK USER:** Confirm if concerns found.
---
## Phase 4: Create New Stories
**Step 1: Create**
```bash
gh pmu create --repo {repository} \
  --title "Story: {New Story Title}" \
  --label "story" \
  --body "{story_body}" \
  --status backlog \
  --priority {priority} \
  --assignee @me
```
**Body template:**
```markdown
## Story: {Title}

### Description
As a {user type}, I want {capability} so that {benefit}.

### Relevant Skills
<!-- Read framework-config.json projectSkills; lookup descriptions from .claude/metadata/skill-registry.json -->
**If configured:** list `{skill} - {description}`. Load: `read {frameworkPath}/Skills/{skill}/SKILL.md`
**If none:** "No project skills configured. Run `/charter` to set up."

### Acceptance Criteria
- [ ] {Assigned criterion 1}
- [ ] {Assigned criterion 2}

### Documentation (if applicable)
- [ ] Design decisions documented (`Construction/Design-Decisions/YYYY-MM-DD-{topic}.md`)
- [ ] Tech debt logged (`Construction/Tech-Debt/YYYY-MM-DD-{topic}.md`)

**Guidelines:** Skip trivial. Update existing. Significant debt → enhancement issue.

### Origin
Split from #{original_story_num}: {Original Story Title}

### TDD Test Cases
Inherited from original (see test plan).

### Definition of Done
- [ ] All AC met
- [ ] TDD tests pass
- [ ] Code reviewed
- [ ] No regressions

**Priority:** {P0|P1|P2}
**Parent Epic:** #{epic_num}
```
**Step 2: Link to epic**
```bash
gh pmu sub add {epic_num} {new_story_num} || true
```
**Step 3:** Collect new numbers for reporting.
---
## Phase 5: Update Original
**Step 1: Update body**
```bash
gh pmu view $story_num --body-stdout > .tmp-original.md
```
Append split notice:
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
**Step 2: Save**
```bash
gh pmu edit $story_num -F .tmp-original.md
rm .tmp-original.md
```
**Step 3: Close**
```bash
gh issue close $story_num --comment "Split into: #{new_1}, #{new_2}, ...

Work the split stories instead."
```
---
## Phase 6: Update Test Plan
**Step 1: Find**
```bash
gh issue view $epic_num --json body --jq '.body' | grep -oE "PRD/[A-Za-z0-9_-]+/PRD-[A-Za-z0-9_-]+\.md"
```
**PRD found:** `PRD/{name}/PRD-{name}.md` → `PRD/{name}/Test-Plan-{name}.md`
**No test plan:**
```
ℹ️ No test plan found for this epic.
   Test cases will be managed when story work begins.
```
Skip to Phase 7.
**Step 2: Redistribute**
Find `### Story: {Original Title} (#{original_num})`.
**Step 3: Update structure**
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
**Step 4: Commit**
```bash
git add PRD/{name}/Test-Plan-{name}.md
git commit -m "docs: split test cases for Story #{original_num}

Split into: #{new_1}, #{new_2}
Refs #{epic_num}"
```
---
## Phase 7: Update PRD Tracker (if applicable)
**Step 1: Check**
```bash
gh issue view $epic_num --json body --jq '.body' | grep -oE "\*\*PRD Tracker:\*\* #[0-9]+"
```
**Found:** extract PRD number, comment:
```bash
gh issue comment $prd_num --body "✂️ **Story Split**

Original: #{original_num} - {Original Title}
Split into:
- #{new_story_1}: {Title 1}
- #{new_story_2}: {Title 2}

Epic: #{epic_num}

Split via \`/split-story\`"
```
**Not found:** skip (not PRD-derived).
---
## Phase 8: Report
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
| Not a story | "Issue #N does not have 'story' label." |
| No parent epic | "Could not find parent epic. Link manually after split." |
| <2 stories in split | "Split requires at least 2 new stories." |
| Charter concern declined | "Story split cancelled due to scope concerns." |
| No test plan | Proceed without update (note in output) |
| Already closed | "Story #N is already closed. Cannot split closed stories." |
---
**End of /split-story Command**
