---
version: "v0.82.0"
description: Transform proposal into Agile PRD
argument-hint: "<issue-number> | extract [<directory>]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /create-prd
Transform a proposal document into an Agile PRD with user stories, acceptance criteria, and epic groupings.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command create-prd`
---
## Prerequisites
**Load from `.claude/metadata/command-boilerplate.json`** -> `prerequisites.common`.
**Fallback:** `gh pmu` installed, `.gh-pmu.json` configured.
**Command-specific:**
- Proposal issue exists with `proposal` label
- Proposal body links to `Proposal/[Name].md`
- Proposal document exists in `Proposal/`
- (Recommended) `CHARTER.md` + `Inception/` artifacts
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
| **Issue-Driven** | `/create-prd 123` | Transform proposal to PRD |
| **Extract** | `/create-prd extract [src/]` | Extract PRD from codebase |
| **Interactive** | `/create-prd` | Prompt for mode selection |
---
## Execution Instructions
**REQUIRED:** Load from `.claude/metadata/command-boilerplate.json` -> `executionInstructions.steps` and `todoRules`.
**Fallback:** Generate TodoWrite todos from phases/steps, include extension point todos, track progress, re-read after compaction.
---
## Workflow (Issue-Driven Mode)
### Phase 1: Fetch Proposal from Issue
**Step 1: Parse issue number**
```bash
issue_num="${1#\#}"
```
**Step 2: Fetch and validate issue**
```bash
gh issue view $issue_num --json labels,body --jq '.labels[].name' | grep -q "proposal"
```
If not proposal: report error with `proposal` label requirement.
**Step 3: Extract proposal document path**
```
Pattern: /Proposal\/[A-Za-z0-9_-]+\.md/
```
If not found: report error with expected format.
**Step 4: Load context files**
| File | Required | Purpose |
|------|----------|---------|
| `<extracted-proposal-path>` | Yes | Source proposal |
| `CHARTER.md` | Recommended | Scope validation |
| `Inception/Scope-Boundaries.md` | Recommended | In/out of scope |
| `Inception/Constraints.md` | Optional | Technical constraints |
| `Inception/Architecture.md` | Optional | System architecture |
**Load Anti-Hallucination Rules:** `{frameworkPath}/Assistant/Anti-Hallucination-Rules-for-PRD-Work.md`

<!-- USER-EXTENSION-START: pre-analysis -->
<!-- USER-EXTENSION-END: pre-analysis -->

### Phase 2: Validate Against Charter
| Finding | Action |
|---------|--------|
| Aligned | Proceed |
| Possibly misaligned | Ask for confirmation |
| Conflicts with out-of-scope | Flag conflict, offer resolution |
**Resolution:** 1. Expand charter 2. Defer to future 3. Proceed (drift) 4. Revise proposal
### Phase 3: Analyze Proposal Gaps
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
Check proposal for `## Path Analysis`.
**If exists:** Extract paths per category:
| Path Category | Informs |
|---------------|---------|
| Exception Paths | Error handling AC |
| Edge/Corner Cases | Boundary-condition AC |
| Negative Test Scenarios | Test plan negative cases |
| Nominal Path | Primary flow validation |
| Alternative Paths | Alternative flow AC |
Parse `###` subsections, extract numbered items, store for Phase 4.5/6.5.
**If missing:** Non-blocking, proceed.
### Phase 3.6: Extract Screen Spec References (if present)
Check proposal for `## Screen Specs` and `## Mockups`.
**If Screen Specs:** Parse file refs, read specs, extract element data for AC in Phase 4.5.
**If Mockups:** Parse refs, note for cross-referencing.
**Consumption only:** Reads refs, does not create. Missing files -> warn and continue.
**If neither:** Non-blocking, proceed.
### Phase 4: Dynamic Question Generation
Generate context-aware questions for missing elements. Rules: reference proposal details, only ask what's missing, allow "skip", present 3-5 at a time.

<!-- USER-EXTENSION-START: pre-transform -->
<!-- USER-EXTENSION-END: pre-transform -->

### Phase 4.5: Story Transformation
**Process:** 1. Identify USER 2. Identify CAPABILITY 3. Identify BENEFIT 4. Transform to story format
**Anti-Pattern Detection:** Flag implementation details, move to Technical Notes.

<!-- USER-EXTENSION-START: post-transform -->
<!-- USER-EXTENSION-END: post-transform -->

#### Solo-Mode Epic Preference
Check `reviewMode` from `framework-config.json`:
```javascript
const { getReviewMode } = require('./.claude/scripts/shared/lib/review-mode.js');
const mode = getReviewMode(process.cwd(), null);
```
| Mode | Behavior |
|------|----------|
| `solo` | Prompt: consolidate into single epic? |
| `team`/`enterprise` | Standard multi-epic grouping |
**When `solo`:** `AskUserQuestion`:
```javascript
AskUserQuestion({
  questions: [{
    question: "Solo mode detected. Group all stories under a single epic for simplicity?",
    header: "Epic structure",
    options: [
      { label: "Single epic (Recommended)", description: "Consolidate under one epic" },
      { label: "Keep multiple epics", description: "Standard multi-epic grouping" }
    ],
    multiSelect: false
  }]
});
```
If confirmed: 1 epic, stories become 1.1, 1.2, etc. If declined: standard grouping.
### Phase 5: Priority Validation
| Priority | Required Distribution |
|----------|----------------------|
| P0 (Must Have) | <=40% |
| P1 (Should Have) | 30-40% |
| P2 (Could Have) | >=20% |
Skip for <6 stories.

<!-- USER-EXTENSION-START: pre-diagram -->
<!-- USER-EXTENSION-END: pre-diagram -->

### Phase 5.5a: Diagram Style Selection
```javascript
AskUserQuestion({
  questions: [{
    question: "Which diagram style should this PRD use?",
    header: "Diagram style",
    options: [
      { label: "drawio (Rich SVG)", description: ".drawio.svg files in Diagrams/" },
      { label: "ASCII (Text-based UML)", description: "Inline in PRD markdown" }
    ],
    multiSelect: false
  }]
});
```
### Phase 5.5b: Diagram Generation
| Diagram Type | Default | When Appropriate |
|--------------|---------|------------------|
| Use Case | ON | User-facing features |
| Activity | ON | Multi-step workflows |
| Sequence | OFF | API/service interactions |
| Class | OFF | Data models |
| Component | OFF | System architecture |
| State | OFF | State machines |
#### drawio style:
Load `{frameworkPath}/Skills/drawio-generation/SKILL.md`. Generate `.drawio.svg` in:
```
PRD/{PRD-Name}/Diagrams/{Epic-Name}/{type}-{description}.drawio.svg
```
#### ASCII style:
Inline in PRD. Wrap in ` ```text ``` `, use box-drawing characters, ensure monospace alignment, place under `### Diagrams` per epic.
| Type | Key Elements |
|------|-------------|
| Use Case | Actors, ellipses, system boundary |
| Activity | Start/end, actions, diamonds, arrows |
| Sequence | Participants, lifelines, arrows, activation |
| Class | Boxes with compartments |
| Component | Boxes with stereotype, interfaces |
| State | States, transitions, start/end markers |
No `Diagrams/` directory for ASCII.

<!-- USER-EXTENSION-START: diagram-generator -->
<!-- USER-EXTENSION-END: diagram-generator -->

<!-- USER-EXTENSION-START: post-diagram -->
<!-- USER-EXTENSION-END: post-diagram -->

<!-- USER-EXTENSION-START: pre-generation -->
<!-- USER-EXTENSION-END: pre-generation -->

### Phase 6: Generate PRD
```
PRD/
└── {PRD-Name}/
    ├── PRD-{PRD-Name}.md
    └── Diagrams/                          <- drawio only
        └── {Epic-Name}/
            └── {type}-{description}.drawio.svg
```
ASCII embeds inline (no Diagrams/ dir). Existing flat PRDs grandfathered.
Create at `PRD/{name}/PRD-{name}.md`.
**Load template:** `{frameworkPath}/Templates/artifacts/prd-template.md`. Populate placeholders.
**Fallback:** Warn, use standard structure (Overview, Epics, Stories, Diagrams, Technical Notes, Out of Scope, Dependencies, Open Questions).

<!-- USER-EXTENSION-START: post-generation -->
<!-- USER-EXTENSION-END: post-generation -->

<!-- USER-EXTENSION-START: quality-checklist -->
<!-- USER-EXTENSION-END: quality-checklist -->

### Phase 6.5: Generate TDD Test Plan
**Step 1: Load test config**
| Source | Data |
|-------|------|
| `Inception/Test-Strategy.md` | Framework, coverage, TDD philosophy |
| `Inception/Tech-Stack.md` | Language for test syntax |
**Fallback:** 1. Check `{frameworkPath}/IDPF-Agile/Agile-Core.md` TDD section 2. Warn 3. Defaults: 80% unit, "TBD" framework
**Step 2:** Generate `PRD/{name}/Test-Plan-{name}.md`
**Load template:** `{frameworkPath}/Templates/artifacts/test-plan-template.md`. Populate placeholders.
**Fallback:** Warn, use standard structure.
**Derivation:** 1. Parse AC from PRD 2. Generate 2-3 test cases per criterion 3. Identify integration points 4. Extract E2E scenarios
### Phase 6.6: Create Test Plan Approval Issue
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

**Warning: Create-Backlog is blocked until this issue is closed.**" \
  --status backlog
```
Update test plan frontmatter with issue number after creation.
### Phase 7: Proposal Lifecycle Completion
**Issue-Driven Mode only.**
**Step 1: Move proposal**
```bash
git ls-files --error-unmatch Proposal/{Name}.md 2>/dev/null
git add Proposal/{Name}.md
git mv Proposal/{Name}.md Proposal/Implemented/{Name}.md
```
If tracked: skip `git add`. If untracked: `git add` then `git mv`.
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
**Step 4: Report**
```
PRD: PRD/{name}/PRD-{name}.md | Test Plan: PRD/{name}/Test-Plan-{name}.md
Proposal archived: Proposal/Implemented/{Name}.md | Proposal issue #{issue_num} closed
PRD tracker: #{prd_issue_num} | Test plan approval: #{test_plan_issue_num}
Diagrams: PRD/{name}/Diagrams/ (if generated)
Warning: Approve test plan before /create-backlog
Next: /create-backlog {prd_issue_num}
```
---
## Interactive Mode
For `/create-prd` (no args): prompt mode selection (proposal issue or extraction).
If issue-driven: prompt for issue number, proceed.
---
## Workflow (Extract Mode)
### Step 1: Check Prerequisites
Verify `{frameworkPath}/Skills/codebase-analysis/SKILL.md` exists. Check `Inception/`.
If skill not found -> **STOP**
If `Inception/` missing: warn, offer `/charter` (non-blocking).
### Step 2: Load Skill
Read `{frameworkPath}/Skills/codebase-analysis/SKILL.md`.
### Step 3: Run Codebase Analysis
Delegate to skill for tech stack detection, architecture inference, test parsing, NFR detection.
### Step 4: Bridge to Phase 6
Use analysis output for PRD via Phase 6. Same diagram workflow. Present features with confidence levels.
### Step 5: Add Extraction Metadata
Add confidence levels, extraction metadata section, evidence citations.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." |
| Missing proposal label | "Issue #N does not have 'proposal' label." |
| No proposal path in body | "Could not find proposal document link." |
| Proposal file missing | "Proposal not found at <path>." |
| No Inception/ | "Proceeding with limited validation." |
| User skips all questions | "Insufficient detail. Add more to proposal?" |
| Empty proposal | "Minimum: problem + solution." |
---
## Quality Checklist
- [ ] All stories have acceptance criteria
- [ ] Requirements prioritized (P0-P2)
- [ ] Priority distribution valid (or <6 stories)
- [ ] Technical Notes separated from stories
- [ ] Out of scope stated
- [ ] Open questions flagged
- [ ] Create-Backlog compatible
---
## Technical Skills Mapping
### Step 1: Run skill matcher
```bash
node .claude/scripts/shared/prd-skill-matcher.js --prd "PRD/{name}/PRD-{name}.md"
```
Returns: `{ matchedSkills, existingSkills, newSkills, registryAvailable }`. Uses `framework-config.json` and `.claude/metadata/skill-keywords.json` automatically.
**Note:** Read `framework-config.json` with Read tool (not Glob -- symlinked).
**Fallback:** Script missing/crashes -> warn, continue.
### Step 2: Present New Skills
**ASK USER:**
```
PRD mentions technical requirements that suggest additional skills:

- ci-cd-pipeline-design (CI/CD pipeline mentioned in Non-Functional Requirements)
- api-versioning (API versioning needed for service integration)

Add to project skills? (yes/no/edit)
```
### Step 3: Update framework-config.json
```javascript
config.projectSkills = [...existingSkills, ...newSkills];
fs.writeFileSync('framework-config.json', JSON.stringify(config, null, 2));
```
### Step 4: Persist Confirmed Suggestions
```javascript
const { persistSuggestions } = require('./.claude/scripts/shared/lib/persist-skill-suggestions');
persistSuggestions('framework-config.json', confirmedSuggestions, '#ISSUE');
```
Writes to `suggestedSkills`. Excludes already-installed. Declined not written.
---
**End of /create-prd Command**
