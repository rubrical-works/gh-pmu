---
version: "v0.84.0"
description: Add story to epic with charter compliance (project)
argument-hint: "[epic-number] (e.g., 42 or #42)"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /add-story
Add a new story to an epic with charter compliance validation and automatic test plan updates.
## Arguments
| Argument | Description |
|----------|-------------|
| `[epic-number]` | Parent epic issue number (e.g., `42` or `#42`). Optional - prompts if not specified. |
## Execution Instructions
**REQUIRED:** Before executing:
1. **Create Tasks:** Use `TaskCreate` to create tasks from the phases below (bulk upfront)
2. **Track Progress:** Use `TaskUpdate` to mark tasks `in_progress` -> `completed`
3. **Resume Point:** If interrupted, use `TaskList` to find incomplete tasks
## Phase 1: Select or Create Epic, Gather Story Details
**Step 1: Parse epic number (if provided)**
```bash
epic_num="${1#\#}"
```
**Step 2: If no epic specified, prompt for selection**
```bash
gh issue list --label "epic" --state open --json number,title
```
Display options (always include "Create new epic"). If no epics exist, offer create or cancel.
**ASK USER:** Select an option.
**Step 2a: Create new epic (if selected)**
**ASK USER:** What is the theme or feature area for this epic?
If `CHARTER.md` exists, validate epic theme against charter scope. If concerns, warn and ask to proceed.
```bash
gh pmu create --repo {repository} \
  --title "Epic: {Theme}" \
  --label "epic" \
  --status backlog \
  --assignee @me \
  -F .tmp-epic-body.md
```
**Epic Body Template** (write to `.tmp-epic-body.md`):
```markdown
## Epic: {Theme}
### Vision
{Brief description based on user's theme input}
### Stories
Stories will be linked via `/add-story`.
### Acceptance Criteria
- [ ] All stories completed
- [ ] Integration tested
- [ ] Documentation updated
**Note:** This epic was created via `/add-story`.
Expand with detailed acceptance criteria as scope becomes clearer.
```
Clean up: `rm .tmp-epic-body.md`
**Assign to current branch (if active):**
```bash
gh pmu branch current --json=name 2>/dev/null && \
  gh pmu move {epic_num} --branch current
```
**Step 3: Validate epic exists**
```bash
gh issue view $epic_num --json labels --jq '.labels[].name' | grep -q "epic"
```
If not an epic: `Error: Issue #$epic_num does not have the 'epic' label.`
**Step 4: Gather story details**
**ASK USER:** Please describe the new story:
- What should the user be able to do?
- What is the benefit/value?
- What are the key acceptance criteria?
**Step 5: Transform to story format**
| User Input | Story Field |
|------------|-------------|
| User action description | **I want** clause |
| Benefit/value | **So that** clause |
| Acceptance criteria | Checkbox list |
Infer the user type from context or ask if unclear.
## Phase 2: Charter Compliance Check
**Step 1: Load charter context**
| File | Required | Purpose |
|------|----------|---------|
| `CHARTER.md` | Recommended | Project vision, goals, scope |
| `Inception/Scope-Boundaries.md` | Optional | In/out of scope boundaries |
| `Inception/Constraints.md` | Optional | Technical/business constraints |
If no charter: warn and suggest running `/charter`.
**Step 2: Validate story against charter** -- compare against vision, goals, scope boundaries, constraints.
**Step 3: Report compliance**
- If aligned: report matching goal and scope item
- If concern: show concerning element, constraint, exclusion; **ASK USER** to confirm
## Phase 3: Create Story Issue
**Step 1: Determine priority**
**ASK USER:** What priority? P0 (must have/blocking), P1 (important), P2 (nice to have)
**Step 2: Create story issue**
```bash
gh pmu create --repo {repository} \
  --title "Story: {Story Title}" \
  --label "story" \
  --body "{story_body}" \
  --status backlog \
  --priority {priority} \
  --assignee @me
```
**Story Body Template:**
> **ATOMIC TEMPLATE -- All sections below are REQUIRED.**
> Every story must include ALL sections. Use "N/A" rather than removing.
> This is the **canonical definition** -- other commands reference this template.
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
- [ ] {Criterion 1}
- [ ] {Criterion 2}
- [ ] {Criterion 3}
### Documentation (if applicable)
- [ ] Design decisions documented (update existing or create `Construction/Design-Decisions/YYYY-MM-DD-{topic}.md`)
- [ ] Tech debt logged (update existing or create `Construction/Tech-Debt/YYYY-MM-DD-{topic}.md`)
**Guidelines:** Skip trivial findings. Update existing docs rather than duplicating. For significant tech debt, create an enhancement issue.
### TDD Test Cases
**Note:** Test cases will be added when story work begins.
See test plan for related test cases (if applicable).
### Definition of Done
- [ ] All acceptance criteria met
- [ ] TDD test cases pass
- [ ] Code reviewed
- [ ] No regressions
**Priority:** {P0|P1|P2}
**Parent Epic:** #{epic_num}
```
**Step 3: Link to parent epic**
```bash
gh pmu sub add {epic_num} {story_num} || true
```
## Phase 4: Update Test Plan
**Step 1: Find relevant test plan**
```bash
gh issue view $epic_num --json body --jq '.body' | grep -oE "PRD/[A-Za-z0-9_-]+/PRD-[A-Za-z0-9_-]+\.md"
```
If PRD found, derive: `PRD/{name}/PRD-{name}.md` -> `PRD/{name}/Test-Plan-{name}.md`
If no PRD/Test Plan: note and skip to Phase 5.
**Step 2: Load test configuration** from `Inception/Test-Strategy.md` and `Inception/Tech-Stack.md`. Fallback to `{frameworkPath}/IDPF-Agile/Agile-Core.md` TDD defaults.
**Step 3: Generate test cases** from acceptance criteria (valid input, invalid input, edge case per criterion).
**Step 4: Update test plan document** with story section and test case table.
**Step 5: Commit test plan changes**
```bash
git add PRD/{name}/Test-Plan-{name}.md
git commit -m "docs: add test cases for Story #{story_num}

Refs #{epic_num}"
```
## Phase 5: Update PRD Tracker (if applicable)
**Step 1: Check for PRD Tracker reference in epic**
```bash
gh issue view $epic_num --json body --jq '.body' | grep -oE "\*\*PRD Tracker:\*\* #[0-9]+"
```
If no PRD Tracker: skip to Phase 6.
**Step 2: Find PRD document file from tracker**
```bash
gh issue view $prd_num --json body --jq '.body' | grep -oE "PRD/[^/]+/PRD-[^.]+\.md"
```
Store as `$prd_file`. If not found, warn and continue.
**Step 3: Update PRD tracker issue body**
```bash
gh pmu view $prd_num --body-stdout > .tmp-prd-tracker.md
```
**Update all 4 count locations:**
| Location | Pattern | Update |
|----------|---------|--------|
| Backlog Summary story count | `Stories: N` | Increment N |
| Epic table row | `\| {Epic Title} \| #{epic_num} \| {stories} \|` | Append `, #{story_num}` |
| Epics section story count | `\| {Epic Title} \| N stories \| {priority} \|` | Increment N |
| Total line | `**Total:** N user stories` | Increment N |
**For NEW epic:** Add new rows to tables, increment epic count.
```bash
gh pmu edit $prd_num -F .tmp-prd-tracker.md
rm .tmp-prd-tracker.md
```
**Step 4: Update PRD document file (if found)** -- determine next story number under epic, append story section.
**Step 5: Add comment to PRD tracker**
```bash
gh issue comment $prd_num --body "Story #{story_num}: {Story Title}
Epic: #{epic_num}, Priority: {priority}
PRD tracker and document updated via \`/add-story\`"
```
**Step 6: Commit PRD document changes**
```bash
git add "{prd_file}"
git commit -m "docs: add Story {Epic}.{N} to PRD

Story #{story_num}: {Story Title}
Epic: #{epic_num}

Refs #{prd_num}"
```
## Phase 6: Skill Suggestions (Optional)
**Step 1: Check opt-out** -- if `framework-config.json` has `"skillSuggestions": false`, skip to Report.
**Step 2: Run keyword matching**
```bash
node .claude/scripts/shared/lib/skill-keyword-matcher.js \
  --content-file .tmp-skill-content.txt \
  --installed "{comma-separated projectSkills from framework-config.json}"
rm .tmp-skill-content.txt
```
Parse JSON output. Already-installed skills excluded automatically.
**Step 3: If matches found**
**ASK USER:** Install suggested skills? (y/n)
**Step 4: Install selected skills**
```bash
node .claude/scripts/shared/install-skill.js {skill-names...}
```
## Phase 7: Report Completion
```
Story created: #{story_num}
Story: {Title}
Epic: #{epic_num} - {Epic Title}
Priority: {P0|P1|P2}
Charter compliance: Aligned (or Proceeded with warning)
Test plan: {Updated|Not applicable}
PRD tracker: {Updated #{prd_num}|Not PRD-derived}
PRD document: {Updated {prd_file}|Not found|Not PRD-derived}
Skills suggested: {count} (installed: {installed_count})
Next steps:
1. Work the story: work #{story_num}
2. View epic progress: gh pmu sub list #{epic_num}
```
## Error Handling
| Situation | Response |
|-----------|----------|
| Epic not found | "Issue #N not found. Check the issue number?" |
| Issue not an epic | "Issue #N does not have 'epic' label." |
| No epics, user cancels | "Story creation cancelled." |
| Epic creation fails | Report error, do not create orphan story |
| Epic theme out of scope | Warn user, allow override with confirmation |
| No charter, user declines | "Story creation cancelled." |
| Charter concern, user declines | "Story creation cancelled due to scope concerns." |
| Test plan not found | Proceed without test plan update (note in output) |
**End of /add-story Command**
