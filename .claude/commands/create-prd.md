---
version: "v0.67.1"
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
**Load shared prerequisites from `.claude/metadata/command-boilerplate.json`** -> `prerequisites.common`.
**Graceful degradation:** If not found, defaults: `gh pmu` installed, `.gh-pmu.json` configured.
**Command-specific:**
- Proposal issue with `proposal` label
- Issue body links to `Proposal/[Name].md`
- Proposal document exists
- (Recommended) Charter: `CHARTER.md` + `Inception/` artifacts
---
## Arguments
| Argument | Description |
|----------|-------------|
| `<issue-number>` | Proposal issue number (e.g., `123` or `#123`) |
| `extract` | Extract PRD from existing codebase |
| `extract <directory>` | Extract from specific directory |
---
## Modes
| Mode | Invocation | Description |
|------|------------|-------------|
| **Issue-Driven** | `/create-prd 123` | Transform proposal to PRD |
| **Extract** | `/create-prd extract [dir]` | Extract PRD from codebase |
| **Interactive** | `/create-prd` | Prompt for mode selection |
---
## Execution Instructions
**REQUIRED:** Load from `.claude/metadata/command-boilerplate.json` -> `executionInstructions`.
**Graceful degradation:** If not found, defaults: generate TodoWrite todos, include extension todos, track progress, re-read after compaction.
---
## Workflow (Issue-Driven Mode)
### Phase 1: Fetch Proposal from Issue
**Step 1:** Parse issue number (strip leading `#`).
**Step 2:** Validate `proposal` label:
```bash
gh issue view $issue_num --json labels,body --jq '.labels[].name' | grep -q "proposal"
```
**Step 3:** Extract path: `Pattern: /Proposal\/[A-Za-z0-9_-]+\.md/`
**Step 4:** Load context files:
| File | Required | Purpose |
|------|----------|---------|
| Proposal path | Yes | Source |
| `CHARTER.md` | Recommended | Scope validation |
| `Inception/Scope-Boundaries.md` | Recommended | In/out of scope |
| `Inception/Constraints.md` | Optional | Constraints |
| `Inception/Architecture.md` | Optional | Architecture |
**Load Anti-Hallucination Rules:** `{frameworkPath}/Assistant/Anti-Hallucination-Rules-for-PRD-Work.md`

<!-- USER-EXTENSION-START: pre-analysis -->
<!-- USER-EXTENSION-END: pre-analysis -->

### Phase 2: Validate Against Charter
| Finding | Action |
|---------|--------|
| Aligned | Proceed |
| Possibly misaligned | Ask confirmation |
| Conflicts with out-of-scope | Flag, offer resolution |
**Resolutions:** Expand charter, defer, proceed anyway, revise proposal.
### Phase 3: Analyze Proposal Gaps
| Element | Detection | Gap Action |
|---------|-----------|------------|
| Problem statement | "Problem:", "Issue:" | Ask if missing |
| Proposed solution | "Solution:", "Approach:" | Ask if missing |
| User stories | "As a...", "User can..." | Generate questions |
| Acceptance criteria | "- [ ]", "Done when" | Generate questions |
| Priority | "P0-P3", "High/Medium/Low" | Ask if missing |

<!-- USER-EXTENSION-START: post-analysis -->
<!-- USER-EXTENSION-END: post-analysis -->

### Phase 3.5: Extract Path Analysis (if present)
Check for `## Path Analysis`. If present, extract per category (Exception, Edge, Corner, Negative, Nominal, Alternative). Missing: non-blocking.
### Phase 3.6: Extract Screen Spec References (if present)
Check for `## Screen Specs` and `## Mockups`. Read referenced specs for element data. Consumption only. Missing: non-blocking.
### Phase 4: Dynamic Question Generation
Context-aware questions for missing elements. Reference proposal details, only ask what's missing, allow skip, 3-5 at a time.

<!-- USER-EXTENSION-START: pre-transform -->
<!-- USER-EXTENSION-END: pre-transform -->

### Phase 4.5: Story Transformation
Identify USER, CAPABILITY, BENEFIT. Transform to story format.
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
### Phase 5: Priority Validation
| Priority | Distribution |
|----------|-------------|
| P0 | <=40% |
| P1 | 30-40% |
| P2 | >=20% |
**Exemption:** Skip for <6 stories.

<!-- USER-EXTENSION-START: pre-diagram -->
<!-- USER-EXTENSION-END: pre-diagram -->

### Phase 5.5: Diagram Generation
**Load:** `{frameworkPath}/Skills/drawio-generation/SKILL.md`
**MUST:** Generate `.drawio.svg` diagrams:
| Type | Default | When |
|------|---------|------|
| Use Case | ON | User-facing features |
| Activity | ON | Multi-step workflows |
| Sequence | OFF | API interactions |
| Class | OFF | Data models |
| Component | OFF | Architecture |
| State | OFF | State machines |

<!-- USER-EXTENSION-START: diagram-generator -->
<!-- USER-EXTENSION-END: diagram-generator -->

<!-- USER-EXTENSION-START: post-diagram -->
<!-- USER-EXTENSION-END: post-diagram -->

<!-- USER-EXTENSION-START: pre-generation -->
<!-- USER-EXTENSION-END: pre-generation -->

### Phase 6: Generate PRD
Create in `PRD/{PRD-Name}/PRD-{PRD-Name}.md` with `Diagrams/` subdirectory. Existing flat PRDs grandfathered.
**Load template from `{frameworkPath}/Templates/artifacts/prd-template.md`.** Graceful degradation if missing.

<!-- USER-EXTENSION-START: post-generation -->
<!-- USER-EXTENSION-END: post-generation -->

<!-- USER-EXTENSION-START: quality-checklist -->
<!-- USER-EXTENSION-END: quality-checklist -->

### Phase 6.5: Generate TDD Test Plan
Load test config from `Inception/Test-Strategy.md` and `Inception/Tech-Stack.md`. Fallback: framework defaults.
Generate `PRD/{name}/Test-Plan-{name}.md`. Load template from `{frameworkPath}/Templates/artifacts/test-plan-template.md`.
Derivation: parse ACs, generate 2-3 test cases per criterion, identify integration points, extract E2E scenarios.
### Phase 6.6: Create Test Plan Approval Issue
```bash
gh pmu create --label test-plan --label approval-required --assignee @me \
  --title "Approve Test Plan: {Name}" --body "..." --status backlog
```
Update test plan frontmatter with issue number.
### Phase 7: Proposal Lifecycle Completion
**Issue-Driven only.** Move proposal (`git mv`), close proposal issue, create PRD tracking issue, report completion.
---
## Interactive Mode
No arguments: prompt for mode (proposal issue or code extraction).
---
## Workflow (Extract Mode)
1. Check `{frameworkPath}/Skills/codebase-analysis/SKILL.md` exists. If not: STOP.
2. Load skill.
3. Run codebase analysis.
4. Bridge to Phase 6. Present features with confidence levels.
5. Add extraction metadata.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | Error |
| Missing proposal label | Error |
| Path not in body | Error |
| File not found | Error |
| No Inception/ | Limited validation |
| User skips all | Insufficient detail |
| Empty proposal | Needs more detail |
---
## Quality Checklist
- [ ] All stories have ACs
- [ ] Prioritized (P0-P2)
- [ ] Priority distribution valid (or <6)
- [ ] Technical Notes separated
- [ ] Out of scope stated
- [ ] Open questions flagged
- [ ] Create-Backlog compatible
---
## Technical Skills Mapping
Run `node .claude/scripts/shared/prd-skill-matcher.js --prd "PRD/{name}/PRD-{name}.md"`.
**ASK USER** to add detected skills. Update `projectSkills` in `framework-config.json`.
Persist via `persistSuggestions()`.
---
**End of /create-prd Command**
