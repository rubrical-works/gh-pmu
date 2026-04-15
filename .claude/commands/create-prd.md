---
version: "v0.87.0"
description: Transform proposal into Agile PRD
argument-hint: "<issue-number> | extract [<directory>]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /create-prd
Transform a proposal document into an Agile PRD with user stories, acceptance criteria, and epic groupings.
**Extension Points:** `.claude/metadata/extension-points.json` or `/extensions list --command create-prd`

## Prerequisites
Load shared from `.claude/metadata/command-boilerplate.json` -> `prerequisites.common`. **Graceful degradation when boilerplate not found:** use defaults — `gh pmu` installed, `.gh-pmu.json` configured.
**Command-specific:** proposal issue with `proposal` label; body links `Proposal/[Name].md`; document exists in `Proposal/`; (Recommended) `CHARTER.md` + `Inception/`.

## Arguments
| Argument | Description |
|----------|-------------|
| `<issue-number>` | Proposal issue (`123` or `#123`) |
| `extract` | Extract PRD from codebase (requires `/charter`) |
| `extract <directory>` | Extract from specific directory |

## Modes
| Mode | Invocation | Description |
|------|------------|-------------|
| **Issue-Driven** | `/create-prd 123` | Transform proposal to PRD |
| **Extract** | `/create-prd extract [src/]` | Extract PRD from codebase |
| **Interactive** | `/create-prd` | Prompt for mode selection |

## Execution Instructions
**REQUIRED:** Load from `.claude/metadata/command-boilerplate.json` -> `executionInstructions.steps` and `executionInstructions.todoRules`. Defaults if missing: generate TaskCreate tasks from phases/steps, include extension point todos, track progress, re-read spec after compaction.

## Workflow (Issue-Driven Mode)

### Phase 1: Fetch Proposal from Issue
**Step 1: Parse issue number**
```bash
issue_num="${1#\#}"
```
**Step 2: Fetch and validate**
```bash
gh issue view $issue_num --json labels,body --jq '.labels[].name' | grep -q "proposal"
```
If not a proposal issue, error: `Issue #$issue_num does not have the 'proposal' label.` Tell user to create with `proposal: <description>`.

**Step 3: Extract proposal document path** — pattern `/Proposal\/[A-Za-z0-9_-]+\.md/`. If not found, error: `Could not find proposal document link in issue #$issue_num. Expected: File: Proposal/[Name].md`.

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
| Finding | Action |
|---------|--------|
| Aligned | Proceed |
| Possibly misaligned | Ask for confirmation |
| Conflicts with out-of-scope | Flag, offer resolution |

**Resolution Options:** 1) Expand charter scope; 2) Defer to future release; 3) Proceed anyway (creates drift); 4) Revise proposal.

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
Check proposal for `## Path Analysis`. If present, extract per category to inform PRD:
| Path Category | Informs |
|---------------|---------|
| Exception Paths | Error handling acceptance criteria |
| Edge Cases | Boundary-condition acceptance criteria |
| Corner Cases | Boundary-condition acceptance criteria |
| Negative Test Scenarios | Test plan negative test cases |
| Nominal Path | Primary user story flow validation |
| Alternative Paths | Alternative flow acceptance criteria |

**Extraction:** parse each `###` subsection under `## Path Analysis`; extract numbered items as scenario descriptions; store by category for Phase 4.5 and 6.5. If section missing, proceed (non-blocking).

### Phase 3.6: Extract Screen Spec References (if present)
Check proposal for `## Screen Specs` and `## Mockups`.
- **`## Screen Specs`:** parse file references (`Screen-Specs/{Screen-Name}.md`); read each to extract element specs (field names, types, validation rules, defaults) for Phase 4.5.
- **`## Mockups`:** parse references (`Mockups/{Screen-Name}-mockup.md`); note availability for cross-referencing.

**Consumption only:** `/create-prd` reads references — does not discover/create screen specs. If files missing, warn and continue. If neither section present, proceed (non-blocking).

### Phase 4: Dynamic Question Generation
Generate context-aware questions for missing elements. Rules: reference specific proposal details; only ask what's truly missing; allow "skip"/"not sure"; present 3-5 at a time.

<!-- USER-EXTENSION-START: pre-transform -->
<!-- USER-EXTENSION-END: pre-transform -->

### Phase 4.5: Story Transformation
Identify USER (who benefits), CAPABILITY (what can they do), BENEFIT (why it matters); transform to story format.
**Anti-Pattern Detection:** flag implementation details (file ops, internal changes, code-level details); move to Technical Notes.

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
| `solo` | Prompt: consolidate into single epic? |
| `team` | No prompt -- standard multi-epic grouping |
| `enterprise` | No prompt -- standard multi-epic grouping |

**When `solo`:** Use `AskUserQuestion` with options "Single epic (Recommended) — consolidate all stories under one epic for solo development" vs "Keep multiple epics — preserve standard multi-epic grouping for concurrent workstreams". Header "Epic structure", `multiSelect: false`.
- **If confirmed:** consolidate into 1 epic with descriptive title from proposal name (e.g., "Epic 1: {Feature Name}"); stories become 1.1, 1.2, 1.3, etc.
- **If declined:** standard multi-epic grouping.
**`team`/`enterprise`:** skip entirely.

### Phase 5: Priority Validation
| Priority | Required Distribution |
|----------|----------------------|
| P0 (Must Have) | <=40% of stories |
| P1 (Should Have) | 30-40% of stories |
| P2 (Could Have) | >=20% of stories |

**Small PRD Exemption:** skip validation for PRDs with <6 stories.

<!-- USER-EXTENSION-START: pre-diagram -->
<!-- USER-EXTENSION-END: pre-diagram -->

### Phase 5.5a: Diagram Style Selection
Use `AskUserQuestion` ("Which diagram style should this PRD use?", header "Diagram style", `multiSelect: false`) with options:
- **drawio (Rich SVG):** generate `.drawio.svg` files — editable in draw.io, rich visuals, in `Diagrams/`.
- **ASCII (Text-based UML):** generate text-based UML diagrams inline in PRD markdown — renders everywhere, clean diffs, no external tooling.

Store selected style for Phase 5.5b.

### Phase 5.5b: Diagram Generation
**Diagram type selection** (regardless of style):
| Diagram Type | Default | When Appropriate |
|--------------|---------|------------------|
| Use Case | ON | User-facing features |
| Activity | ON | Multi-step workflows |
| Sequence | OFF | API/service interactions |
| Class | OFF | Data models, entities |
| Component | OFF | System architecture |
| State | OFF | State machines |

**If drawio:** Load `{frameworkPath}/Skills/drawio-generation/SKILL.md`. **MUST** generate UML diagrams as `.drawio.svg` files at `PRD/{PRD-Name}/Diagrams/{Epic-Name}/{type}-{description}.drawio.svg`.

**If ASCII:** Generate text-based UML inline using box-drawing characters.
- Wrap each in code fence (` ```text ... ``` `) for monospace
- Use box-drawing characters — no `+`, `-`, `|` substitutes
- Ensure monospace alignment (one column per character)
- Place under `### Diagrams` section per epic

**ASCII type templates:**
| Type | Key Elements |
|------|-------------|
| Use Case | Actors (stick figure text), ellipses (rounded boxes), system boundary |
| Activity | Start/end nodes, action boxes, decision diamonds, arrows |
| Sequence | Participant boxes, lifelines, arrows, activation bars |
| Class | Class boxes with compartments (name, attributes, methods) |
| Component | Component boxes with stereotype, interfaces |
| State | State boxes, transitions with labels, start/end markers |

No `Diagrams/` directory for ASCII — diagrams live in PRD markdown.

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
    └── Diagrams/                          ← drawio style only
        └── {Epic-Name}/
            └── {type}-{description}.drawio.svg
```
ASCII style: no `Diagrams/` subdirectory (diagrams inline). Existing flat PRDs (`PRD/PRD-{name}.md`) grandfathered.

Create `PRD/{name}/PRD-{name}.md`. Load template from `{frameworkPath}/Templates/artifacts/prd-template.md` and populate placeholders. If template missing, warn `"PRD template file missing, using inline fallback."` and use sections: Overview, Epics, User Stories, Diagrams, Technical Notes, Out of Scope, Dependencies, Open Questions.

<!-- USER-EXTENSION-START: post-generation -->
<!-- USER-EXTENSION-END: post-generation -->

<!-- USER-EXTENSION-START: quality-checklist -->
<!-- USER-EXTENSION-END: quality-checklist -->

### Phase 6.5: Generate TDD Test Plan
**Step 1: Load test configuration**
| Source File | Data to Extract |
|-------------|-----------------|
| `Inception/Test-Strategy.md` | Test framework, coverage targets, TDD philosophy |
| `Inception/Tech-Stack.md` | Language (for test syntax) |

**Fallback (if Test-Strategy.md missing):** check `{frameworkPath}/IDPF-Agile/Agile-Core.md` TDD Cycle section; warn `"No Test-Strategy.md found. Using framework defaults. Run /charter to customize."`; defaults: 80% unit coverage, "TBD" for framework.

**Step 2: Generate** `PRD/{name}/Test-Plan-{name}.md`. Load template from `{frameworkPath}/Templates/artifacts/test-plan-template.md`. If missing, warn `"Test plan template file missing, using inline fallback."` and use sections: Source, Test Strategy Overview, Epic Test Coverage, Integration Test Points, E2E Scenarios, Coverage Targets, Approval Checklist.

**Derivation Rules:** parse each story's acceptance criteria; for each criterion generate 2-3 test cases (valid, invalid, edge); identify cross-story/cross-epic integration points; extract E2E scenarios from user journeys.

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

**⚠️ Create-Backlog is blocked until this issue is closed.**" \
  --status backlog
```
Update test plan frontmatter with the issue number after creation.

### Phase 7: Proposal Lifecycle Completion
**Issue-Driven Mode only.**

**Step 1: Move proposal document** (check git tracking first)
```bash
git ls-files --error-unmatch Proposal/{Name}.md 2>/dev/null
git add Proposal/{Name}.md
git mv Proposal/{Name}.md Proposal/Implemented/{Name}.md
```
If `git ls-files --error-unmatch` succeeds → tracked, skip `git add`. If fails → untracked, run `git add` first.

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
PRD: PRD/{name}/PRD-{name}.md | Test Plan: PRD/{name}/Test-Plan-{name}.md
Proposal archived: Proposal/Implemented/{Name}.md | Proposal issue #{issue_num} closed
PRD tracker issue: #{prd_issue_num} | Test plan approval issue: #{test_plan_issue_num}
Diagrams: PRD/{name}/Diagrams/ (if generated)
⚠️ Approve test plan (#{test_plan_issue_num}) before running /create-backlog
Next: /create-backlog {prd_issue_num}
```

## Interactive Mode
For `/create-prd` (no args), prompt:
```
How would you like to create the PRD?
1. From a proposal issue (enter issue number)
2. From existing code (extraction)
> [user selects]
```
**If 1:** prompt for issue number, proceed with Issue-Driven workflow.

## Workflow (Extract Mode)
For `/create-prd extract [<directory>]`:

**Step 1: Check Prerequisites** — verify `{frameworkPath}/Skills/codebase-analysis/SKILL.md` exists. Check `Inception/` artifacts. If skill missing: `codebase-analysis skill not installed. Install via px-manager or ask the user to install it.` → **STOP**. If `Inception/` missing, warn and offer `/charter` (non-blocking).

**Step 2: Load Skill** — read `{frameworkPath}/Skills/codebase-analysis/SKILL.md`.

**Step 3: Run Codebase Analysis** — delegate to skill (entire project or specified directory). Skill handles tech stack detection, architecture inference, test parsing, NFR detection.

**Step 4: Bridge to Phase 6** — use skill output to generate PRD via Phase 6. Same diagram style selection (5.5a) and generation (5.5b) as Issue-Driven. Present extracted features with confidence levels for user selection before generation.

**Step 5: Add Extraction Metadata** — augment Phase 6 output with confidence levels per story, extraction metadata section, evidence citations per feature.

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

## Quality Checklist
- [ ] All user stories have acceptance criteria
- [ ] Requirements prioritized (P0-P2)
- [ ] Priority distribution valid (or <6 stories)
- [ ] Technical Notes separated from stories
- [ ] Out of scope explicitly stated
- [ ] Open questions flagged
- [ ] PRD is Create-Backlog compatible

## Technical Skills Mapping
After PRD generation, check for additional skills based on technical requirements.

**Step 1: Run skill matcher**
```bash
node .claude/scripts/shared/prd-skill-matcher.js --prd "PRD/{name}/PRD-{name}.md"
```
Parse JSON: `{ matchedSkills, existingSkills, newSkills, registryAvailable }`. Script reads `framework-config.json` for installed skills and `.claude/metadata/skill-keywords.json` for keyword registry.
**Note:** If reading `framework-config.json` manually, use Read tool (do NOT use Glob — `.claude/metadata/` is symlinked in user projects and Glob does not follow symlinks). If script missing/crashes, warn `"Skill matching unavailable, skipping."` and continue (non-blocking).

**Step 2: Present New Skills** — **ASK USER:**
```
PRD mentions technical requirements that suggest additional skills:

- ci-cd-pipeline-design (CI/CD pipeline mentioned in Non-Functional Requirements)
- api-versioning (API versioning needed for service integration)

Add to project skills? (yes/no/edit)
```

**Step 3: Update framework-config.json** — use the `framework-config.js` helper which validates against `.claude/metadata/framework-config.schema.json` before writing (schema-invalid output rejected at write time):
```javascript
const fwconfig = require('./.claude/scripts/shared/lib/framework-config.js');
const config = fwconfig.read(process.cwd());
config.projectSkills = [...new Set([...(config.projectSkills || []), ...newSkills])].sort();
fwconfig.write(process.cwd(), config);
```
If `fwconfig.write` throws a validation error, surface the message and stop — do not retry with raw `fs.writeFileSync`. Report:
```
Added skills: ci-cd-pipeline-design, api-versioning
Total project skills: 4
```

**Step 4: Persist Confirmed Suggestions** — after user confirms (or declines), persist for px-manager discovery:
```javascript
const { persistSuggestions } = require('./.claude/scripts/shared/lib/persist-skill-suggestions');
persistSuggestions('framework-config.json', confirmedSuggestions, '#ISSUE');
```
Writes to `suggestedSkills` in `framework-config.json`. Skills already in `projectSkills` excluded. Declined suggestions not written.

**End of /create-prd Command**
