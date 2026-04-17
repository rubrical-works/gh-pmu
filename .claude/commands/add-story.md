---
version: "v0.88.0"
description: Add story to epic with charter compliance (project)
argument-hint: "[epic-number] (e.g., 42 or #42)"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /add-story
Add a story to an epic with charter compliance and test plan updates.
## Arguments
| Argument | Description |
|----------|-------------|
| `[epic-number]` | Parent epic (e.g., `42` or `#42`). Prompts if omitted. |
## Execution Instructions
**REQUIRED:**
1. **Create Tasks:** Use `TaskCreate` to bulk-create tasks from phases below.
2. **Track Progress:** `TaskUpdate` to mark `in_progress` → `completed`.
3. **Resume:** `TaskList` to find incomplete tasks.

Tasks: Phase 1 select/create epic; Phase 2 charter check; Phase 3 create story; Phase 4 update test plan; Phase 5 update PRD tracker; Phase 6 skill suggestions; Phase 7 report.
## Phase 1: Select or Create Epic, Gather Story Details
**Step 1:** Parse epic number — strip leading `#`: `epic_num="${1#\#}"`
**Step 2:** If no epic specified, list and prompt:
```bash
gh issue list --label "epic" --state open --json number,title
```
Display options including "[Create new epic]". If no epics exist, offer create or cancel.
**ASK USER:** Select an option.
**Step 2a: Create new epic (if selected)**
**ASK USER:** Theme/feature area (e.g., "User Authentication").

Charter compliance check: if `CHARTER.md` exists, validate theme against vision and scope; warn and ask to proceed if concerns.

Create epic:
```bash
gh pmu create --repo {repository} \
  --title "Epic: {Theme}" \
  --label "epic" \
  --status backlog \
  --assignee @me \
  -F .tmp-epic-body.md
```
Epic body template (`.tmp-epic-body.md`):
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

**Note:** Created via `/add-story`. Expand with detailed acceptance criteria.
```
Clean up: `rm .tmp-epic-body.md`

Assign to current branch if active:
```bash
gh pmu branch current --json=name 2>/dev/null && \
  gh pmu move {epic_num} --branch current
```
Report: `✅ Created Epic #{epic_num}: {Theme} — Assigned to branch: {branch_name}`
**Step 3: Validate epic**
```bash
gh issue view $epic_num --json labels --jq '.labels[].name' | grep -q "epic"
```
If not an epic: `Error: Issue #$epic_num does not have the 'epic' label.`
**Step 4: Gather story details**
**ASK USER:** Describe the new story — what can the user do, benefit, acceptance criteria.
**Step 5: Transform to story format**
| User Input | Story Field |
|------------|-------------|
| User action description | **I want** clause |
| Benefit/value | **So that** clause |
| Acceptance criteria | Checkbox list |
Infer user type from context or ask.
## Phase 2: Charter Compliance Check
**Step 1: Load charter context**
| File | Required | Purpose |
|------|----------|---------|
| `CHARTER.md` | Recommended | Project vision, goals, scope |
| `Inception/Scope-Boundaries.md` | Optional | In/out of scope boundaries |
| `Inception/Constraints.md` | Optional | Technical/business constraints |

If no charter: `⚠️ No CHARTER.md found. Skipping compliance check. Consider /charter.`
**Step 2: Validate story against:** vision alignment, goal relevance, scope boundaries, constraint compliance.
**Step 3: Report compliance**
- Aligned: `✅ Story aligns with charter scope`
- Concern: `⚠️ Potential scope concern: ... Proceed anyway? (yes/no)`
**ASK USER:** Confirm to proceed if concerns found.
## Phase 3: Create Story Issue
**Step 1: Determine priority**
**ASK USER:** Priority:
| Priority | Description |
|----------|-------------|
| P0 | Must have - blocking |
| P1 | Should have - important |
| P2 | Could have - nice to have |
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
> **⚠️ ATOMIC TEMPLATE — All sections below are REQUIRED.**
> Every story MUST include ALL sections. No section may be omitted.
> If N/A, include with "N/A" rather than removing.
> Callers (including `/create-backlog`) must apply the complete template.
> This is the **canonical definition** — other commands reference, not duplicate.

```markdown
## Story: {Title}
### Description
As a {user type}, I want {capability} so that {benefit}.
### Relevant Skills
<!-- Read from framework-config.json projectSkills array -->
<!-- For each, lookup from .claude/metadata/skill-registry.json -->

**If projectSkills configured:**
- {skill-name} - {description}

Load skill: `read {frameworkPath}/Skills/{skill-name}/SKILL.md`

**If no projectSkills:** Run `/charter` to set up project-specific skills.
### Acceptance Criteria
- [ ] {Criterion 1}
- [ ] {Criterion 2}
- [ ] {Criterion 3}
### Documentation (if applicable)
- [ ] Design decisions documented (update existing or create `Construction/Design-Decisions/YYYY-MM-DD-{topic}.md`)
- [ ] Tech debt logged (update existing or create `Construction/Tech-Debt/YYYY-MM-DD-{topic}.md`)

**Guidelines:** Skip trivial findings. Update existing docs rather than duplicating. For significant tech debt, create an enhancement issue.
### TDD Test Cases
**Note:** Test cases added when story work begins. See test plan for related cases.
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
**Step 1: Find test plan** — check epic for PRD ref:
```bash
gh issue view $epic_num --json body --jq '.body' | grep -oE "PRD/[A-Za-z0-9_-]+/PRD-[A-Za-z0-9_-]+\.md"
```
Derive: `PRD/{name}/PRD-{name}.md → PRD/{name}/Test-Plan-{name}.md`

If no test plan found: `ℹ️ No test plan. Test cases created when work begins.` → skip to Phase 5.
**Step 2: Load test config** — from `Inception/Test-Strategy.md` (framework, organization) and `Inception/Tech-Stack.md` (language). Fallback to `{frameworkPath}/IDPF-Agile/Agile-Core.md` TDD defaults with warning.
**Step 3: Generate test cases from acceptance criteria**
| Criterion | Test Cases |
|-----------|------------|
| {Criterion text} | Valid input, Invalid input, Edge case |
**Step 4: Update test plan**
```markdown
### Story: {Story Title} (#{story_num})

| Acceptance Criteria | Test Cases |
|--------------------|------------|
| {Criterion 1} | ✓ Test valid input |
|                | ✓ Test invalid input |
|                | ✓ Test edge case |
```
**Step 5: Commit**
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
If none: skip to Phase 6.
**Step 2: Find PRD file**
```bash
gh issue view $prd_num --json body --jq '.body' | grep -oE "PRD/[^/]+/PRD-[^.]+\.md"
```
Store as `$prd_file`. Warn and continue without document update if missing.
**Step 3: Update PRD tracker body**
```bash
gh pmu view $prd_num --body-stdout > .tmp-prd-tracker.md
```
Update all 4 count locations:
| Location | Pattern | Update |
|----------|---------|--------|
| Backlog Summary story count | `✅ Stories: N` | Increment N |
| Epic table row | `\| {Epic Title} \| #{epic_num} \| {stories} \|` | Append `, #{story_num}` |
| Epics section story count | `\| {Epic Title} \| N stories \| {priority} \|` | Increment N |
| Total line | `**Total:** N user stories` | Increment N |

For NEW epic:
- Add new row to Backlog Summary: `| {Epic Title} | #{epic_num} | #{story_num} |`
- Add new row to Epics section: `| {Epic Title} | 1 stories | {priority} |`
- Increment `✅ Epics: N`

```bash
gh pmu edit $prd_num -F .tmp-prd-tracker.md
rm .tmp-prd-tracker.md
```
**Step 4: Update PRD document file (if found)**
Determine next story number (highest under epic + 1, e.g., 4.2 → 4.3). Append after last story in this epic:
```markdown
#### Story {Epic}.{N}: {Story Title}
**As a** {user type}
**I want** {capability}
**So that** {benefit}

**Acceptance Criteria:**
- [ ] {Criterion 1}
- [ ] {Criterion 2}
- [ ] {Criterion 3}

**Priority:** {P0|P1|P2}
**Issue:** #{story_num}

---
```
For NEW epic: also add epic section to Epics overview.
**Step 5: Comment on PRD tracker**
```bash
gh issue comment $prd_num --body "📝 **Story Added**

Story #{story_num}: {Story Title}
Epic: #{epic_num}
Priority: {priority}

PRD tracker and document updated.
Added via \`/add-story\`"
```
**Step 6: Commit PRD document changes** (if updated)
```bash
git add "{prd_file}"
git commit -m "docs: add Story {Epic}.{N} to PRD

Story #{story_num}: {Story Title}
Epic: #{epic_num}

Refs #{prd_num}"
```
## Phase 6: Skill Suggestions (Optional)
Suggest relevant skills based on technologies in the new story.
**Step 1:** Read `framework-config.json` — if `skillSuggestions: false`, skip phase.
**Step 2: Keyword match** — combine story title + acceptance criteria text, write to temp file:
```bash
node .claude/scripts/shared/lib/skill-keyword-matcher.js \
  --content-file .tmp-skill-content.txt \
  --installed "{comma-separated projectSkills}"
rm .tmp-skill-content.txt
```
Parse JSON output: `{skill, matchedKeywords}`. Already-installed excluded.
**Step 3: If matches, prompt**
```
This story references technologies with available skills:
  • playwright-setup - Playwright test automation setup

Install suggested skills? (y/n)
```
**ASK USER:** Install?
**Step 4: Install**
```bash
node .claude/scripts/shared/install-skill.js {skill-names...}
```
Report inline: `✓ playwright-setup - Installed (5 resources)`
## Phase 7: Report Completion
```
Story created: #{story_num}

Story: {Title}
Epic: #{epic_num} - {Epic Title}
Priority: {P0|P1|P2}

Charter compliance: ✅ Aligned (or ⚠️ Proceeded with warning)

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
| Epic theme out of scope | Warn, allow override with confirmation |
| No charter, user declines | "Story creation cancelled." |
| Charter concern, user declines | "Story creation cancelled due to scope concerns." |
| Test plan not found | Proceed without test plan update |

**End of /add-story Command**
