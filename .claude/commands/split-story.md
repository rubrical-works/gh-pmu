---
version: "v0.80.0"
description: Split story into smaller stories (project)
argument-hint: "<story-number> (e.g., 123)"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /split-story
Split a story into smaller stories while maintaining charter compliance and test plan integrity.
## Arguments
| Argument | Description |
|----------|-------------|
| `<story-number>` | Story issue number to split (e.g., `123` or `#123`) |
## Execution Instructions
**REQUIRED:** Use `TodoWrite` to create todos from phases below. Track progress. Resume from incomplete todos if interrupted.
## Phase 1: Fetch and Validate Original Story
**Step 1:** Parse story number — accept `123` or `#123`: `story_num="${1#\#}"`
**Step 2:** Fetch and validate:
```bash
gh issue view $story_num --json labels,body,title --jq '.labels[].name' | grep -q "story"
```
If not a story: error and stop.
**Step 3:** Extract story details:
```bash
gh pmu view $story_num --body-stdout > .tmp-story.md
```
Parse: Title, Description (As a.../I want.../So that...), Acceptance criteria, Priority, Parent epic reference.
**Step 4:** Find parent epic:
```bash
gh pmu sub list --child $story_num --json parent
```
Or parse `Parent Epic: #N` from story body.
## Phase 2: Determine Split Criteria
**ASK USER:** How should this story be split?
| Pattern | Description |
|---------|-------------|
| By acceptance criteria | Each criterion becomes a story |
| By user workflow | Split by distinct user actions |
| By technical component | Split by system area (frontend/backend/API) |
| By priority | Separate must-have from nice-to-have |
| Custom | User defines the split |
For each new story gather: Title, which AC it covers, Priority (inherit or override). **Minimum:** 2 new stories required.
## Phase 3: Charter Compliance Check
**Step 1:** Load `CHARTER.md` (recommended), `Inception/Scope-Boundaries.md` (optional), `Inception/Constraints.md` (optional). If no charter: warn and skip.
**Step 2:** Validate each new story for scope creep, constraint violations, goal alignment.
**Step 3:** Report compliance. If scope concern found, show details and **ASK USER** to confirm.
## Phase 4: Create New Stories
For each new story:
**Step 1:** Create story issue:
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
Load skill: `read {frameworkPath}/Skills/{skill-name}/SKILL.md`
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
**Step 2:** Link to parent epic:
```bash
gh pmu sub add {epic_num} {new_story_num} || true
```
**Step 3:** Collect all new story numbers for reporting.
## Phase 5: Update Original Story
**Step 1:** Fetch and update original story body:
```bash
gh pmu view $story_num --body-stdout > .tmp-original.md
```
Add split notice with new story links, reason, and date.
**Step 2:** Save:
```bash
gh pmu edit $story_num -F .tmp-original.md
rm .tmp-original.md
```
**Step 3:** Close original:
```bash
gh issue close $story_num --comment "Split into: #{new_1}, #{new_2}, ...
Work the split stories instead."
```
## Phase 6: Update Test Plan
**Step 1:** Find test plan via epic's PRD reference:
```bash
gh issue view $epic_num --json body --jq '.body' | grep -oE "PRD/[A-Za-z0-9_-]+/PRD-[A-Za-z0-9_-]+\.md"
```
Derive: `PRD/{name}/PRD-{name}.md` -> `PRD/{name}/Test-Plan-{name}.md`. If not found, skip to Phase 7.
**Step 2:** Find original story section in test plan.
**Step 3:** Replace with split story sections, each with AC and test cases.
**Step 4:** Commit:
```bash
git add PRD/{name}/Test-Plan-{name}.md
git commit -m "docs: split test cases for Story #{original_num}

Split into: #{new_1}, #{new_2}
Refs #{epic_num}"
```
## Phase 7: Update PRD Tracker (if applicable)
Check epic for `**PRD Tracker:** #N`. If found, add comment documenting the split. If no tracker, skip.
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
**End of /split-story Command**
