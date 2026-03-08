---
version: "v0.58.0"
description: Split story into smaller stories (project)
argument-hint: "<story-number> (e.g., 123)"
---
<!-- MANAGED -->
# /split-story
Split a story into smaller stories while maintaining charter compliance and test plan integrity.
---
## Arguments
| Argument | Description |
|----------|-------------|
| `<story-number>` | Story issue number to split (e.g., `123` or `#123`) |
---
## Execution Instructions
**REQUIRED:** Use `TodoWrite` to create todos from phases below. Track progress as you work.
---
## Phase 1: Fetch and Validate Original Story
**Step 1: Parse story number** - Accept `123` or `#123` format
**Step 2: Fetch and validate story**
```bash
gh issue view $story_num --json labels,body,title --jq '.labels[].name' | grep -q "story"
```
If not a story: Error with message about missing 'story' label.
**Step 3: Extract story details**
```bash
gh pmu view $story_num --body-stdout > .tmp-story.md
```
Parse: title, description, acceptance criteria, priority, parent epic reference.
**Step 4: Find parent epic**
```bash
gh pmu sub list --child $story_num --json parent
```
Or parse from story body for `Parent Epic: #N`.
---
## Phase 2: Determine Split Criteria
**ASK USER:** How should this story be split?
| Pattern | Description |
|---------|-------------|
| By acceptance criteria | Each criterion becomes a story |
| By user workflow | Split by distinct user actions |
| By technical component | Split by system area (frontend/backend/API) |
| By priority | Separate must-have from nice-to-have |
| Custom | User defines the split |
For each new story: title, which acceptance criteria, priority (inherit or override).
**Minimum:** 2 new stories required.
---
## Phase 3: Charter Compliance Check
**Step 1: Load charter context**
| File | Required | Purpose |
|------|----------|---------|
| `CHARTER.md` | Recommended | Project vision, goals, scope |
| `Inception/Scope-Boundaries.md` | Optional | In/out of scope boundaries |
| `Inception/Constraints.md` | Optional | Technical/business constraints |
If no charter: warn and skip compliance check.
**Step 2:** Validate split stories for scope creep, constraint violations, goal alignment.
**Step 3:** Report compliance or concerns.
**ASK USER:** Confirm to proceed if concerns found.
---
## Phase 4: Create New Stories
For each new story:
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
> **ATOMIC TEMPLATE - All sections REQUIRED.** If N/A, include with "N/A" rather than removing.
```markdown
## Story: {Title}
### Description
As a {user type}, I want {capability} so that {benefit}.
### Relevant Skills
<!-- Read from framework-config.json projectSkills, lookup in .claude/metadata/skill-registry.json -->
Load skill: `read Skills/{skill-name}/SKILL.md`
### Acceptance Criteria
- [ ] {Assigned criterion 1}
- [ ] {Assigned criterion 2}
### Documentation (if applicable)
- [ ] Design decisions documented (`Construction/Design-Decisions/YYYY-MM-DD-{topic}.md`)
- [ ] Tech debt logged (`Construction/Tech-Debt/YYYY-MM-DD-{topic}.md`)
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
---
## Phase 5: Update Original Story
**Step 1:** Export body: `gh pmu view $story_num --body-stdout > .tmp-original.md`
Add split notice section with new story references, reason, and date.
**Step 2:** Save: `gh pmu edit $story_num -F .tmp-original.md && rm .tmp-original.md`
**Step 3: Close original story**
```bash
gh issue close $story_num --comment "Split into: #{new_1}, #{new_2}, ... Work the split stories instead."
```
---
## Phase 6: Update Test Plan
**Step 1:** Find test plan via epic PRD reference.
Derive: `PRD/{name}/PRD-{name}.md` -> `PRD/{name}/Test-Plan-{name}.md`
If no test plan: skip to Phase 7.
**Step 2-3:** Redistribute test cases — replace original story section with split story sections.
**Step 4: Commit**
```bash
git add PRD/{name}/Test-Plan-{name}.md
git commit -m "docs: split test cases for Story #{original_num}

Split into: #{new_1}, #{new_2}
Refs #{epic_num}"
```
---
## Phase 7: Update PRD Tracker (if applicable)
Check epic for PRD Tracker reference. If found, add comment with split details.
If no PRD Tracker: skip.
---
## Phase 8: Report Completion
```
Story split complete: #{original_num} → {count} stories
Original: #{original_num} - {Original Title} (CLOSED)
New stories:
  • #{new_story_1}: {Title 1} (Priority: {P})
  • #{new_story_2}: {Title 2} (Priority: {P})
Epic: #{epic_num}
Charter compliance: {status}
Test plan: {Updated|Not applicable}
PRD tracker: {Updated #{prd_num}|Not PRD-derived}
Next: work #{new_story_1} | gh pmu sub list #{epic_num}
```
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Story not found | "Issue #N not found." |
| Issue not a story | "Issue #N does not have 'story' label." |
| No parent epic found | "Link manually after split." |
| Less than 2 stories | "Split requires at least 2 new stories." |
| Charter concern, user declines | "Story split cancelled." |
| Test plan not found | Proceed without test plan update |
| Already closed | "Cannot split closed stories." |
---
**End of /split-story Command**
