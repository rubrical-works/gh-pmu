---
version: "v0.58.0"
description: Transform proposal into Agile PRD
argument-hint: "<issue-number> | extract [<directory>]"
---
<!-- EXTENSIBLE -->
# /create-prd
Transform a proposal document into an Agile PRD with user stories, acceptance criteria, and epic groupings.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command create-prd`
---
## Prerequisites
- Proposal issue exists with `proposal` label
- Proposal issue body contains link to `Proposal/[Name].md`
- Proposal document exists in `Proposal/` directory
- (Recommended) Charter exists: `CHARTER.md` + `Inception/` artifacts
---
## Arguments
| Argument | Description |
|----------|-------------|
| `<issue-number>` | Proposal issue number (e.g., `123` or `#123`) |
| `extract` | Extract PRD from existing codebase (requires `/charter` first) |
| `extract <directory>` | Extract from specific directory |
---
## Modes
| Mode | Invocation | Description |
|------|------------|-------------|
| **Issue-Driven** | `/create-prd 123` or `/create-prd #123` | Transform proposal to PRD |
| **Extract** | `/create-prd extract` or `/create-prd extract src/` | Extract PRD from codebase |
| **Interactive** | `/create-prd` | Prompt for mode selection |
---
## Execution Instructions
**REQUIRED:** Before executing this command:
1. **Generate Todo List:** Parse the phases and extension points in this spec, then use `TodoWrite` to create todos
2. **Include Extensions:** For each non-empty `USER-EXTENSION` block, add a todo item
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** If resuming after context compaction, re-read this spec and regenerate todos
**Todo Generation Rules:**
- One todo per numbered phase/step
- One todo per active extension point (non-empty `USER-EXTENSION` blocks)
- Skip commented-out extensions
- Use the phase/step name as the todo content
---
## Workflow (Issue-Driven Mode)
### Phase 1: Fetch Proposal from Issue
**Step 1: Parse issue number**
```bash
# Strip leading # if present
issue_num="${1#\#}"
```
**Step 2: Fetch and validate issue**
```bash
gh issue view $issue_num --json labels,body --jq '.labels[].name' | grep -q "proposal"
```
**If not a proposal issue:**
```
Error: Issue #$issue_num does not have the 'proposal' label.
This command requires a proposal issue. Create one with:
  proposal: <description>
```
**Step 3: Extract proposal document path**
```
Pattern: /Proposal\/[A-Za-z0-9_-]+\.md/
```
**If proposal path not found:**
```
Error: Could not find proposal document link in issue #$issue_num.
Expected format: File: Proposal/[Name].md
```
**Step 4: Load context files**
| File | Required | Purpose |
|------|----------|---------|
| `<extracted-proposal-path>` | Yes | Source proposal |
| `CHARTER.md` | Recommended | Project scope validation |
| `Inception/Scope-Boundaries.md` | Recommended | In/out of scope |
| `Inception/Constraints.md` | Optional | Technical constraints |
| `Inception/Architecture.md` | Optional | System architecture |
**Load Anti-Hallucination Rules:**
| Context | Rules Path |
|---------|------------|
| All projects | `{frameworkPath}/Assistant/Anti-Hallucination-Rules-for-PRD-Work.md` |

<!-- USER-EXTENSION-START: pre-analysis -->
<!-- USER-EXTENSION-END: pre-analysis -->

### Phase 2: Validate Against Charter
Compare proposal against charter scope:
| Finding | Action |
|---------|--------|
| Aligned | Proceed |
| Possibly misaligned | Ask for confirmation |
| Conflicts with out-of-scope | Flag conflict, offer resolution |
**Resolution Options:**
1. Expand charter scope
2. Defer to future release
3. Proceed anyway (creates drift)
4. Revise proposal
### Phase 3: Analyze Proposal Gaps
Parse proposal to identify present/missing elements:
| Element | Detection Patterns | Gap Action |
|---------|-------------------|------------|
| Problem statement | "Problem:", "Issue:", first paragraph | Ask if missing |
| Proposed solution | "Solution:", "Approach:" | Ask if missing |
| User stories | "As a...", "User can..." | Generate questions |
| Acceptance criteria | "- [ ]", "Done when" | Generate questions |
| Priority | "P0-P3", "High/Medium/Low" | Ask if missing |

<!-- USER-EXTENSION-START: post-analysis -->
<!-- USER-EXTENSION-END: post-analysis -->

### Phase 3.5: Extract Path Analysis (if present)
Check the proposal document for a `## Path Analysis` section.
**If `## Path Analysis` section exists:**
Extract paths per category and use them to inform PRD generation:
| Path Category | Informs |
|---------------|---------|
| Exception Paths | Error handling acceptance criteria |
| Edge Cases | Boundary-condition acceptance criteria |
| Corner Cases | Boundary-condition acceptance criteria |
| Negative Test Scenarios | Test plan negative test cases |
| Nominal Path | Primary user story flow validation |
| Alternative Paths | Alternative flow acceptance criteria |
**Extraction process:**
1. Parse each `###` subsection under `## Path Analysis`
2. Extract numbered items as scenario descriptions
3. Store by category for use in Phase 4.5 and Phase 6.5
**If `## Path Analysis` section is missing:** Proceed normally — non-blocking.
### Phase 4: Dynamic Question Generation
Generate context-aware questions for missing elements.
**Question Rules:**
1. Reference specific proposal details
2. Only ask what's truly missing
3. Allow "skip" or "not sure" responses
4. Present 3-5 questions at a time

<!-- USER-EXTENSION-START: pre-transform -->
<!-- USER-EXTENSION-END: pre-transform -->

### Phase 4.5: Story Transformation
Transform proposal requirements into Agile user stories.
**Transformation Process:**
1. Identify USER (who benefits?)
2. Identify CAPABILITY (what can they do?)
3. Identify BENEFIT (why does it matter?)
4. Transform to story format
**Anti-Pattern Detection:** Flag implementation details (file operations, internal changes, code-level details) and move to Technical Notes section.

<!-- USER-EXTENSION-START: post-transform -->
<!-- USER-EXTENSION-END: post-transform -->

#### Solo-Mode Epic Preference
After transforming stories, check `reviewMode` from `framework-config.json`:
```javascript
const { getReviewMode } = require('./.claude/scripts/shared/lib/review-mode.js');
const mode = getReviewMode(process.cwd(), null);
```
| Mode | Behavior |
|------|----------|
| `solo` | Prompt user: consolidate into single epic? |
| `team` | No prompt — standard multi-epic grouping |
| `enterprise` | No prompt — standard multi-epic grouping |
**When `solo` mode detected:**
Use `AskUserQuestion` to offer single-epic consolidation:
```javascript
AskUserQuestion({
  questions: [{
    question: "Solo mode detected. Group all stories under a single epic for simplicity? (Or keep multiple epics for planned workstream use)",
    header: "Epic structure",
    options: [
      { label: "Single epic (Recommended)", description: "Consolidate all stories under one epic — simpler for solo development" },
      { label: "Keep multiple epics", description: "Preserve standard multi-epic grouping (e.g., for concurrent workstreams)" }
    ],
    multiSelect: false
  }]
});
```
- **If confirmed (single epic):** Consolidate all stories into 1 epic. Use descriptive title from proposal name (e.g., "Epic 1: {Feature Name}"). All stories become Story 1.1, 1.2, 1.3, etc.
- **If declined (keep multiple):** Proceed with standard multi-epic grouping.
**When `team` or `enterprise` mode:** Skip this step entirely.
### Phase 5: Priority Validation
Validate priority distribution before generation:
| Priority | Required Distribution |
|----------|----------------------|
| P0 (Must Have) | ≤40% of stories |
| P1 (Should Have) | 30-40% of stories |
| P2 (Could Have) | ≥20% of stories |
**Small PRD Exemption:** Skip validation for PRDs with <6 stories.

<!-- USER-EXTENSION-START: pre-diagram -->
<!-- USER-EXTENSION-END: pre-diagram -->

### Phase 5.5: Diagram Generation
**Load with:** `Skills/drawio-generation/SKILL.md`
**MUST:** Generate UML diagrams as `.drawio.svg` files:
| Diagram Type | Default | When Appropriate |
|--------------|---------|------------------|
| Use Case | ON | User-facing features |
| Activity | ON | Multi-step workflows |
| Sequence | OFF | API/service interactions |
| Class | OFF | Data models, entities |
| Component | OFF | System architecture |
| State | OFF | State machines |

<!-- USER-EXTENSION-START: diagram-generator -->
<!-- USER-EXTENSION-END: diagram-generator -->

<!-- USER-EXTENSION-START: post-diagram -->
<!-- USER-EXTENSION-END: post-diagram -->

<!-- USER-EXTENSION-START: pre-generation -->
<!-- USER-EXTENSION-END: pre-generation -->

### Phase 6: Generate PRD
Create PRD in directory structure:
```
PRD/
└── {PRD-Name}/
    ├── PRD-{PRD-Name}.md
    └── Diagrams/
        └── {Epic-Name}/
            └── {type}-{description}.drawio.svg
```
**Note:** Existing flat PRDs (`PRD/PRD-{name}.md`) are grandfathered.
Create PRD document at `PRD/{name}/PRD-{name}.md`:
```markdown
# PRD: <Feature Name>

**Status:** Draft
**Created:** <date>
**Source Proposal:** <proposal-path>
**Proposal Issue:** #<issue-number> (closed)

---

## Overview
<From proposal>

---

## Epics
### Epic 1: <Theme>
Stories: 1.1, 1.2, 1.3

---

## User Stories

### Story 1.1: <Title>
**As a** <user type>
**I want** <capability>
**So that** <benefit>

**Acceptance Criteria:**
- [ ] <criterion>

**Priority:** P0 - Must Have

---

## Diagrams

| Epic | Diagram | Description |
|------|---------|-------------|
| Epic 1 | `Diagrams/Epic-1/use-case-desc.drawio.svg` | Actor interactions |

> **Traceability:** Diagram elements cite source (story ID, AC number).

---

## Technical Notes
> Implementation hints, not requirements.
> Do not create issues from this section.

---

## Out of Scope
<Explicit exclusions>

---

## Dependencies
<Cross-references>

---

## Open Questions
<Unresolved items>

---

*Generated by create-prd skill*
*Ready for Create-Backlog*
```

<!-- USER-EXTENSION-START: post-generation -->
<!-- USER-EXTENSION-END: post-generation -->

<!-- USER-EXTENSION-START: quality-checklist -->
<!-- USER-EXTENSION-END: quality-checklist -->

### Phase 6.5: Generate TDD Test Plan
Create test plan artifact from PRD acceptance criteria.
**Step 1: Load test configuration from project files**
| Source File | Data to Extract |
|-------------|-----------------|
| `Inception/Test-Strategy.md` | Test framework, coverage targets, TDD philosophy |
| `Inception/Tech-Stack.md` | Language (for test syntax) |
**Fallback chain (if Test-Strategy.md missing):**
1. Check `IDPF-Agile/Agile-Core.md` TDD Cycle section (framework-level defaults)
2. Warn: "No Test-Strategy.md found. Using framework defaults. Run /charter to customize."
3. Use defaults: 80% unit coverage, "TBD" for framework
**Step 2: Generate test plan**
**Generate:** `PRD/{name}/Test-Plan-{name}.md`
```markdown
# TDD Test Plan: {Feature Name}

## Source
- **PRD:** PRD-{name}.md
- **Created:** {date}
- **Approval Issue:** #{to-be-created}
- **Test Strategy:** Inception/Test-Strategy.md

## Test Strategy Overview

| Level | Scope | Framework |
|-------|-------|-----------|
| Unit | Individual functions/components | {from Test-Strategy.md → Framework → Unit Tests} |
| Integration | Cross-component flows | {from Test-Strategy.md → Framework → Integration} |
| E2E | Critical user journeys | {from Test-Strategy.md → Framework → E2E, or "TBD"} |

## Epic Test Coverage

### Epic 1: {Name}

| Story | Acceptance Criteria | Test Cases |
|-------|--------------------| -----------|
| {Story title} | {AC from PRD} | ✓ Test valid input |
|               |               | ✓ Test invalid input |
|               |               | ✓ Test edge case |

[Repeat for each epic/story]

## Integration Test Points

| Components | Test Scenario | Priority |
|------------|---------------|----------|
| [Component A] ↔ [Component B] | Data flows correctly | P0 |

## E2E Scenarios

| Scenario | User Journey | Expected Outcome |
|----------|--------------|------------------|
| Happy path | User completes primary flow | Success state |
| Error recovery | User encounters error | Graceful handling |

## Coverage Targets

| Type | Target | Rationale |
|------|--------|-----------|
| Unit | {from Test-Strategy.md → Coverage Targets → Unit, default: 80%+} | Core logic |
| Integration | {from Test-Strategy.md → Coverage Targets → Integration, default: Key flows} | Boundary verification |
| E2E | {from Test-Strategy.md → Coverage Targets → E2E, default: Critical paths} | Journey validation |

## Approval Checklist

- [ ] All PRD acceptance criteria have test cases
- [ ] Edge cases and error conditions identified
- [ ] Integration points between epics mapped
- [ ] E2E scenarios cover critical journeys
- [ ] Coverage targets are realistic
```
**Derivation Rules:**
1. Parse each story's acceptance criteria from PRD
2. For each criterion, generate 2-3 test cases (valid, invalid, edge)
3. Identify cross-story/cross-epic integration points
4. Extract E2E scenarios from user journeys in PRD
### Phase 6.6: Create Test Plan Approval Issue
**Create GitHub issue for test plan approval:**
```bash
gh pmu create --label test-plan --label approval-required --assignee @me \
  --title "Approve Test Plan: {Name}" \
  --body "## Test Plan Review

A TDD test plan has been generated for **{Name}**.

**Test Plan:** PRD/{name}/Test-Plan-{name}.md
**PRD:** PRD/{name}/PRD-{name}.md

## Review Checklist

- [ ] Test cases cover all acceptance criteria
- [ ] Edge cases and error scenarios included
- [ ] Integration test points are complete
- [ ] E2E scenarios cover critical paths
- [ ] Coverage targets are appropriate

## Instructions

1. Review the test plan document
2. Check all boxes above when satisfied
3. Comment with any required changes
4. Close this issue to approve

**⚠️ Create-Backlog is blocked until this issue is closed.**" \
  --status backlog
```
**Update test plan with issue number:** After issue creation, update the Test Plan frontmatter with the approval issue number.
### Phase 7: Proposal Lifecycle Completion
**Only for Issue-Driven Mode** - Complete the proposal lifecycle after PRD generation.
**Step 1: Move proposal document**
Before moving, check if the proposal file is tracked by git.
```bash
# Check if file is tracked
git ls-files --error-unmatch Proposal/{Name}.md 2>/dev/null

# If untracked: git add first so git mv can work
git add Proposal/{Name}.md

# Then move
git mv Proposal/{Name}.md Proposal/Implemented/{Name}.md
```
**Logic:**
- If `git ls-files --error-unmatch` succeeds → file is already tracked, skip `git add`
- If it fails → file is untracked, run `git add` before `git mv`
**Step 2: Close proposal issue**
```bash
gh issue close $issue_num --comment "Transformed to PRD: PRD/{name}/PRD-{name}.md"
gh pmu move $issue_num --status done
```
**Step 3: Create PRD tracking issue**
```bash
gh pmu create --label prd --assignee @me \
  --title "PRD: {Name}" \
  --body "## PRD Document

**File:** PRD/{name}/PRD-{name}.md
**Test Plan:** PRD/{name}/Test-Plan-{name}.md
**Source Proposal:** #$issue_num (closed)

## Status

- [ ] PRD reviewed
- [ ] Test plan approved (see #{test_plan_issue})
- [ ] Ready for backlog creation

## Next Step

1. Review and close test plan approval issue: #{test_plan_issue}
2. Run: \`/create-backlog {this-issue-number}\`" \
  --status backlog
```
**Step 4: Report completion**
```
PRD created: PRD/{name}/PRD-{name}.md
Test Plan created: PRD/{name}/Test-Plan-{name}.md

Proposal lifecycle completed:
  ✓ Proposal archived: Proposal/Implemented/{Name}.md
  ✓ Proposal issue closed: #{issue_num}
  ✓ PRD tracking issue created: #{prd_issue_num}
  ✓ Test plan approval issue created: #{test_plan_issue_num}

Diagrams (if generated):
  PRD/{name}/Diagrams/{epic}/use-case-{desc}.drawio.svg
  PRD/{name}/Diagrams/{epic}/activity-{desc}.drawio.svg

⚠️ APPROVAL REQUIRED before Create-Backlog:
  Review and close: #{test_plan_issue_num} (Approve Test Plan)

Next steps:
1. Review the PRD and test plan
2. Approve test plan by closing #{test_plan_issue_num}
3. Run /create-backlog {prd_issue_num} to generate issues
```
---
## Interactive Mode
For `/create-prd` (no arguments):
```
How would you like to create the PRD?

1. From a proposal issue (enter issue number)
2. From existing code (extraction)

> [user selects]
```
**If user selects 1:**
```
Enter the proposal issue number: ___
```
Then proceed with Issue-Driven Mode workflow.
---
## Workflow (Extract Mode)
For `/create-prd extract` or `/create-prd extract <directory>`:
### Step 1: Check Inception/ Artifacts
```bash
test -d Inception
```
**If missing:**
```
No Inception/ artifacts found.
For best results, run /charter first to establish project context.
Options:
1. Run /charter now (recommended)
2. Proceed without charter context
3. Cancel
```
### Step 2: Load Analysis Skill
Load `Skills/codebase-analysis/SKILL.md` for analysis capabilities.
### Step 3: Analyze Codebase
Run analysis on target (entire project or specified directory):
| Analysis | Output |
|----------|--------|
| Tech stack detection | Languages, frameworks, dependencies |
| Architecture inference | Structure, layers, patterns |
| Test parsing | Features from test descriptions |
| NFR detection | NFRs from code patterns |
### Step 4: User Validation
Present extracted features with confidence levels for user selection.
### Step 5: Diagram Generation
Same diagram workflow as promote mode (Phase 5.5).
### Step 6: Generate PRD
Same Phase 6 output format, with additions:
- Confidence levels for each story
- Extraction metadata section
- Evidence citations for each feature
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found. Check the issue number?" |
| Issue missing proposal label | "Issue #N does not have 'proposal' label." |
| Proposal path not in issue body | "Could not find proposal document link in issue body." |
| Proposal file not found | "Proposal not found at <path>. Check the file exists?" |
| No Inception/ artifacts | "No charter context. Proceeding with limited validation." |
| User skips all questions | "Insufficient detail. Add more to proposal first?" |
| Empty proposal | "Proposal needs more detail. Minimum: problem + solution." |
---
## Quality Checklist
Before finalizing PRD:
- [ ] All user stories have acceptance criteria
- [ ] Requirements are prioritized (P0-P2)
- [ ] Priority distribution is valid (or <6 stories)
- [ ] Technical Notes separated from stories
- [ ] Out of scope explicitly stated
- [ ] Open questions flagged
- [ ] PRD is Create-Backlog compatible
---
## Technical Skills Mapping
After PRD generation, check for additional skills based on technical requirements:
### Step 1: Read Config
Use the Read tool to read `framework-config.json` (do NOT use Glob — `.claude/metadata/` is symlinked in user projects and Glob does not follow symlinks). Get existing `projectSkills` array (may be empty or absent).
### Step 2: Run keyword matching
Collect PRD content (tech requirements, user stories, non-functional requirements). Write to a temp file, then run:
```bash
node .claude/scripts/shared/lib/skill-keyword-matcher.js \
  --content-file .tmp-skill-content.txt \
  --installed "{comma-separated projectSkills from framework-config.json}"
rm .tmp-skill-content.txt
```
Parse JSON output: array of `{skill, matchedKeywords}` objects. Already-installed skills are excluded automatically.
### Step 3: Present New Skills to User
If new skills detected:
**ASK USER:**
```
PRD mentions technical requirements that suggest additional skills:

- ci-cd-pipeline-design (CI/CD pipeline mentioned in Non-Functional Requirements)
- api-versioning (API versioning needed for service integration)

Add to project skills? (yes/no/edit)
```
### Step 4: Update framework-config.json
If user confirms:
```javascript
config.projectSkills = [...existingSkills, ...newSkills];
fs.writeFileSync('framework-config.json', JSON.stringify(config, null, 2));
```
Report added skills:
```
Added skills: ci-cd-pipeline-design, api-versioning
Total project skills: 4
```
---
**End of /create-prd Command**
