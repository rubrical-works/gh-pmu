---
version: "v0.67.0"
description: Add story to epic with charter compliance (project)
argument-hint: "[epic-number] (e.g., 42 or #42)"
copyright: "Rubrical Works (c) 2026"
---

<!-- MANAGED -->
# /add-story

Add a new story to an epic with charter compliance validation and automatic test plan updates.

---

## Arguments

| Argument | Description |
|----------|-------------|
| `[epic-number]` | Parent epic issue number (e.g., `42` or `#42`). Optional - prompts if not specified. |

---

## Execution Instructions

**REQUIRED:** Before executing this command:

1. **Create Todo List:** Use `TodoWrite` to create todos from the steps below
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
3. **Resume Point:** If interrupted, todos show where to continue

**Example todo structure:**
```
- [ ] Phase 1: Select or create epic, gather story details
- [ ] Phase 2: Charter compliance check
- [ ] Phase 3: Create story issue
- [ ] Phase 4: Update test plan
- [ ] Phase 5: Update PRD tracker (if applicable)
- [ ] Phase 6: Skill suggestions (optional)
- [ ] Phase 7: Report completion
```

---

## Phase 1: Select or Create Epic, Gather Story Details

**Step 1: Parse epic number (if provided)**

Accept `42` or `#42` format:
```bash
# Strip leading # if present
epic_num="${1#\#}"
```

**Step 2: If no epic specified, prompt for selection**

```bash
gh issue list --label "epic" --state open --json number,title
```

**Display options (always include "Create new epic"):**
```
Which epic should this story be added to?

1. Epic #42: User Authentication (3 stories)
2. Epic #45: Dashboard Improvements (2 stories)
3. [Create new epic]
```

**If no epics exist:**
```
No open epics found.

1. [Create new epic]
2. Cancel

Would you like to create a new epic for this story?
```

**ASK USER:** Select an option.

**Step 2a: Create new epic (if selected)**

If user selects "Create new epic":

**ASK USER:** What is the theme or feature area for this epic?
```
Example: "User Authentication", "Report Generation", "Performance Improvements"
```

**Charter compliance check for epic theme:**

If `CHARTER.md` exists, validate epic theme against charter scope:
- Check theme aligns with project vision
- Check theme is within scope boundaries
- If concerns found, warn user and ask to proceed

**Create epic issue:**

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
# Check if there's an active branch
gh pmu branch current --json=name 2>/dev/null && \
  gh pmu move {epic_num} --branch current
```

Report:
```
✅ Created Epic #{epic_num}: {Theme}
   Assigned to branch: {branch_name} (or "Not assigned - no active branch")
```

**Step 3: Validate epic exists**

```bash
gh issue view $epic_num --json labels --jq '.labels[].name' | grep -q "epic"
```

**If not an epic:**
```
Error: Issue #$epic_num does not have the 'epic' label.
This command requires an epic issue as the parent.
```

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

---

## Phase 2: Charter Compliance Check

**Step 1: Load charter context**

| File | Required | Purpose |
|------|----------|---------|
| `CHARTER.md` | Recommended | Project vision, goals, scope |
| `Inception/Scope-Boundaries.md` | Optional | In/out of scope boundaries |
| `Inception/Constraints.md` | Optional | Technical/business constraints |

**If no charter exists:**
```
⚠️ No CHARTER.md found. Skipping compliance check.
Consider running /charter to establish project scope.
```

**Step 2: Validate story against charter**

Compare story description against:
- Vision alignment
- Goal relevance
- Scope boundaries (in-scope vs out-of-scope)
- Constraint compliance

**Step 3: Report compliance**

**If aligned:**
```
✅ Story aligns with charter scope
   - Matches goal: "{relevant goal}"
   - Within scope: "{relevant in-scope item}"
```

**If potential concern:**
```
⚠️ Potential scope concern:
   - Story mentions: "{concerning element}"
   - Charter constraint: "{relevant constraint}"
   - Out-of-scope item: "{relevant exclusion}"

Proceed anyway? (yes/no)
```

**ASK USER:** Confirm to proceed if concerns found.

---

## Phase 3: Create Story Issue

**Step 1: Determine priority**

**ASK USER:** What priority should this story have?

| Priority | Description |
|----------|-------------|
| P0 | Must have - blocking other work |
| P1 | Should have - important but not blocking |
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
> Every story must include ALL sections. No section may be omitted.
> If a section is not applicable, include it with "N/A" rather than removing it.
> Callers (including `/create-backlog`) must apply the complete template.
> This is the **canonical definition** — other commands reference this template, not duplicate it.

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

---

## Phase 4: Update Test Plan

**Step 1: Find relevant test plan**

Check epic for PRD reference:
```bash
gh issue view $epic_num --json body --jq '.body' | grep -oE "PRD/[A-Za-z0-9_-]+/PRD-[A-Za-z0-9_-]+\.md"
```

**If PRD found, derive test plan path:**
```
PRD/{name}/PRD-{name}.md → PRD/{name}/Test-Plan-{name}.md
```

**If no PRD/Test Plan found:**
```
ℹ️ No test plan found for this epic.
   Test cases will be created when story work begins.
```
Skip to Phase 5.

**Step 2: Load test configuration**

**From `Inception/Test-Strategy.md`:** Test framework, test organization structure
**From `Inception/Tech-Stack.md`:** Language (for test syntax style)

**Fallback:** If Test-Strategy.md missing, use `{frameworkPath}/IDPF-Agile/Agile-Core.md` TDD defaults and warn user.

**Step 3: Generate test cases from acceptance criteria**

For each acceptance criterion, generate test skeleton using loaded test configuration:

| Criterion | Test Cases |
|-----------|------------|
| {Criterion text} | Valid input test, Invalid input test, Edge case |

**Step 4: Update test plan document**

Add new section to test plan:

```markdown
### Story: {Story Title} (#{story_num})

| Acceptance Criteria | Test Cases |
|--------------------|------------|
| {Criterion 1} | ✓ Test valid input |
|                | ✓ Test invalid input |
|                | ✓ Test edge case |
| {Criterion 2} | ✓ Test valid scenario |
|                | ✓ Test error handling |
```

**Step 5: Commit test plan changes**

```bash
git add PRD/{name}/Test-Plan-{name}.md
git commit -m "docs: add test cases for Story #{story_num}

Refs #{epic_num}"
```

---

## Phase 5: Update PRD Tracker (if applicable)

**Step 1: Check for PRD Tracker reference in epic**

```bash
gh issue view $epic_num --json body --jq '.body' | grep -oE "\*\*PRD Tracker:\*\* #[0-9]+"
```

**If no PRD Tracker:** Skip to Phase 6 (epic is not PRD-derived).

**Step 2: Find PRD document file from tracker**

```bash
# Extract PRD file path from tracker body
gh issue view $prd_num --json body --jq '.body' | grep -oE "PRD/[^/]+/PRD-[^.]+\.md"
```

Store as `$prd_file`. If not found, warn and continue without document update.

**Step 3: Update PRD tracker issue body**

Export tracker body, update counts, then save:

```bash
gh pmu view $prd_num --body-stdout > .tmp-prd-tracker.md
```

**Update all 4 count locations:**

| Location | Pattern | Update |
|----------|---------|--------|
| Backlog Summary story count | `✅ Stories: N` | Increment N |
| Epic table row | `\| {Epic Title} \| #{epic_num} \| {stories} \|` | Append `, #{story_num}` to stories |
| Epics section story count | `\| {Epic Title} \| N stories \| {priority} \|` | Increment N |
| Total line | `**Total:** N user stories` | Increment N |

**For NEW epic** (created via "Create new epic" option):
- Add new row to Backlog Summary table: `| {Epic Title} | #{epic_num} | #{story_num} |`
- Add new row to Epics section: `| {Epic Title} | 1 stories | {priority} |`
- Increment `✅ Epics: N` count

```bash
gh pmu edit $prd_num -F .tmp-prd-tracker.md
rm .tmp-prd-tracker.md
```

**Step 4: Update PRD document file (if found)**

Read the PRD document and add story section under the epic's User Stories heading:

```bash
# Find epic number in PRD (e.g., "Epic 4" for #1143)
# Parse from epic table row mapping or tracker body
```

**Determine next story number:**
- Find highest story number under this epic (e.g., 4.2)
- Increment to get next (e.g., 4.3)

**Append story section** after the last story in this epic:

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

**For NEW epic:** Also add epic section to Epics overview.

**Step 5: Add comment to PRD tracker**

```bash
gh issue comment $prd_num --body "📝 **Story Added**

Story #{story_num}: {Story Title}
Epic: #{epic_num}
Priority: {priority}

PRD tracker and document updated.
Added via \`/add-story\`"
```

**Step 6: Commit PRD document changes**

If PRD document was updated:

```bash
git add "{prd_file}"
git commit -m "docs: add Story {Epic}.{N} to PRD

Story #{story_num}: {Story Title}
Epic: #{epic_num}

Refs #{prd_num}"
```

---

## Phase 6: Skill Suggestions (Optional)

**Purpose:** Suggest relevant skills based on technologies mentioned in the new story.

**Step 1: Check opt-out setting**

Read `framework-config.json`:
```json
{
  "skillSuggestions": false  // If present and false, skip this phase
}
```

**If skillSuggestions is false:** Skip to Report Completion.

**Step 2: Run keyword matching**

Combine story title + acceptance criteria text. Write to a temp file, then run:

```bash
node .claude/scripts/shared/lib/skill-keyword-matcher.js \
  --content-file .tmp-skill-content.txt \
  --installed "{comma-separated projectSkills from framework-config.json}"
rm .tmp-skill-content.txt
```

Parse JSON output: array of `{skill, matchedKeywords}` objects. Already-installed skills are excluded automatically.

**Step 3: If matches found, display and prompt**

```
This story references technologies with available skills:
  • playwright-setup - Playwright test automation setup
  • error-handling-patterns - Error handling strategies

Install suggested skills? (y/n)
```

**ASK USER:** Install suggested skills? (y/n)

**Step 4: Install selected skills**

```bash
node .claude/scripts/shared/install-skill.js {skill-names...}
```

Report result inline:
```
✓ playwright-setup - Installed (5 resources)
```

---

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

---

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

---

**End of /add-story Command**
