---
version: "v0.91.1"
description: Create GitHub epics/stories from PRD (project)
argument-hint: "<issue-number> (e.g., 151)"
copyright: "Rubrical Works (c) 2026"
---

<!-- MANAGED -->
# /create-backlog

Create GitHub epics and stories from an approved PRD with embedded TDD test cases.

## Arguments

`<prd-issue-number>` — PRD tracking issue (`151` or `#151`).

## Execution

Use `TaskCreate` to create tasks from phases below; mark `in_progress` → `completed`. Phases: 1 fetch/validate → 1c PRD review gate → 2 test plan approval gate → 3 parse PRD → 4 load test cases → 5 epics → 6 stories → 7 update PRD status.

## Prerequisites

- PRD tracker with `prd` label, body links to `PRD/PRD-[Name].md`
- Test plan exists at `PRD/[Name]/Test-Plan-[Name].md`
- Test plan approval issue **closed**

## Phase 1: Fetch and Validate PRD Issue

1. Strip `#`: `issue_num="${1#\#}"`
2. Validate PRD label: `gh issue view $issue_num --json labels,body --jq '.labels[].name' | grep -q "prd"`. If missing: `Error: Issue #$issue_num does not have the 'prd' label. Requires a PRD tracker created by /create-prd.`
3. Extract PRD doc path from body via pattern `/PRD\/[A-Za-z0-9_-]+\/PRD-[A-Za-z0-9_-]+\.md/`.

## Phase 1b: (Removed)

Branch assignment is not performed during backlog creation. Issues are created in Backlog status. Use `/assign-branch` afterward.

## Phase 1c: PRD Review Gate

**BLOCKING:** Decomposing an unreviewed PRD risks propagating issues into the epic/story structure.

1. Parse PRD tracker body for checkbox `- \[([ x])\].*PRD reviewed`.
2. Gate:

| State | Action |
|---|---|
| `- [x]` (already-checked) | Proceed normally |
| `- [ ]` or missing | Warn (Step 3) |

3. **Warn** (unchecked):

```
⚠️ PRD has not been reviewed.
Decomposing an unreviewed PRD may propagate unclear scope, missing ACs,
or incomplete stories into the backlog.
PRD Tracker: #$issue_num
```

`AskUserQuestion`:

| Option | Action |
|---|---|
| **Run /review-prd first** (Recommended) | Invoke `/review-prd #$issue_num`, then continue |
| **Continue without review** | Mark checkbox bypassed, proceed |

**Bypass path:**
1. `gh pmu view $issue_num --body-stdout > .tmp-$issue_num.md`
2. Replace `- [ ] PRD reviewed` → `- [x] PRD reviewed (User bypassed PRD review)`
3. `gh pmu edit $issue_num -F .tmp-$issue_num.md && rm .tmp-$issue_num.md`
4. Report: `PRD review bypassed — noted in tracker body`

## Phase 2: Test Plan Approval Gate

**BLOCKING:** Backlog creation blocked until test plan approved.

Find approval issue:
```bash
gh issue list --label "test-plan" --label "approval-required" --state open --json number,title
```

| State | Action |
|---|---|
| **Open** | BLOCK — show message, exit |
| **Closed** | PROCEED |
| **Not found** | WARN — proceed, note missing |

**Block message:**
```
⚠️ Test plan not yet approved.
Test Plan: PRD/{name}/Test-Plan-{name}.md
Approval Issue: #{number} (OPEN)
Review and close the approval issue before creating backlog items.
The test plan ensures all ACs have corresponding test cases.
```

## Phase 3: Parse PRD

Read `PRD/{name}/PRD-{name}.md` and extract Epics, Stories, Acceptance criteria.

| PRD Section | Maps To |
|---|---|
| `## Epics` / `### Epic N:` | Issue with `epic` label |
| User stories under epic | Issues with `story` label |
| Acceptance criteria | Story body checkboxes |
| Priority (P0/P1/P2) | Priority field |

## Phase 4: Load Test Cases

1. Read `PRD/{name}/Test-Plan-{name}.md`
2. Match test cases to stories by title and AC text.
3. Load test config from `Inception/Test-Strategy.md` (framework) and `Inception/Tech-Stack.md` (language):

| Language | Framework | Test Syntax |
|---|---|---|
| TypeScript/JavaScript | Vitest | `test('...', () => { })` |
| TypeScript/JavaScript | Jest | `test('...', () => { })` |
| Python | pytest | `def test_*():` |
| Go | testing | `func Test*(t *testing.T)` |
| Rust | cargo test | `#[test] fn test_*()` |
| Unknown | Unknown | Generic pseudocode |

**Fallback:** If Test-Strategy.md missing, check `{frameworkPath}/IDPF-Agile/Agile-Core.md` for defaults and warn.

## Phase 5: Create Epic Issues

For each epic. Use `gh pmu create` (auto-adds to project board):

```bash
gh pmu create --title "Epic: {Epic Name}" --label "epic" --status backlog \
  --priority {highest_story_priority} -F .tmp-epic-body.md
```

**Priority rule:** Epic = highest priority among child stories (P0+P2 → P0). Default to PRD-level if no per-story priorities.

**Epic body** (`.tmp-epic-body.md`):

```markdown
## Epic: {Epic Name}

**PRD:** PRD/{name}/PRD-{name}.md
**PRD Tracker:** #{prd_issue_number}
**Test Plan:** PRD/{name}/Test-Plan-{name}.md

## Description
{Epic description from PRD}

## Success Criteria
{Success criteria from PRD}

## Stories
Stories will be linked as sub-issues.
```

Cleanup: `rm .tmp-epic-body.md`

## Phase 6: Create Story Issues

For each story:

```bash
gh pmu create --title "Story: {Story Title}" --label "story" --status backlog \
  --priority {prd_priority} -F .tmp-story-body.md
```

**Priority rule:** Story = PRD-specified priority; PRD default if none.

Link to epic: `gh pmu sub add {epic_number} {story_number} || true`. Cleanup: `rm .tmp-story-body.md`.

### Story Body Template

**DEPENDENCY:** Uses **Story Body Template** from `/add-story` Phase 3 — atomic, all sections required. Structural changes go in `/add-story`, not here.

| Template Section | Source |
|---|---|
| **Description** | PRD user story (As a / I want / So that) |
| **Relevant Skills** | `framework-config.json` → `projectSkills` |
| **Acceptance Criteria** | PRD AC checkbox list |
| **Documentation** | Standard checkboxes (atomic) |
| **TDD Test Cases** | ⬇️ EXTENDED (replaces placeholder) |
| **Definition of Done** | ⬇️ EXTENDED (replaces base) |
| **Priority** | PRD priority (P0/P1/P2) |

#### TDD Test Cases Extension

Replace `/add-story` TDD placeholder with skeletons from approved test plan:

```markdown
### TDD Test Cases

**Source:** [Test-Plan-{name}.md](PRD/{name}/Test-Plan-{name}.md#epic-story-section)

Write these tests BEFORE implementation:

#### Unit Tests

```{language}
test('{criterion 1} succeeds with valid input', () => {
  // Arrange / Act / Assert
});

test('{criterion 1} rejects invalid input', () => {
  // Arrange / Act / Assert (expect error)
});
// Additional tests for criteria 2, 3...
```

#### Edge Cases

- [ ] Empty/null input handling
- [ ] Boundary values
- [ ] Error recovery
```

#### Definition of Done Extension

Replace `/add-story` base DoD:

```markdown
### Definition of Done

- [ ] All TDD test cases pass
- [ ] Code coverage ≥ 80%
- [ ] No skipped tests
- [ ] Edge cases handled
- [ ] Acceptance criteria verified
```

## Phase 7: Update PRD Status

1. Update PRD doc: `**Status:** Backlog Created`
2. Prepend banner to PRD tracker body (via `gh pmu edit` + temp file):

```markdown
> **📋 PRD In Progress** — When all stories complete, run `/complete-prd {issue_number}` to verify and close.

## Backlog Summary

✅ Epics: {count}
✅ Stories: {count}
✅ Test cases embedded in each story

---

{original PRD issue body}
```

3. Summary comment:
```bash
gh issue comment $issue_num --body "## Backlog Created

✅ Epics: {count}
✅ Stories: {count}

**Next:** Work stories via \`work #N\`
**When complete:** Run \`/complete-prd $issue_num\`"
```

4. Move PRD: `gh pmu move $issue_num --status in_progress`

PRD remains open until `/complete-prd` verifies all stories Done.

## Output Summary

```
Backlog created from PRD: PRD/{name}/PRD-{name}.md

Epics created:
  • #{N}: Epic: {Name} ({X} stories)
  • #{N}: Epic: {Name} ({Y} stories)

Total: {E} epics, {S} stories

Test cases embedded:
  ✓ {T} test cases pulled from Test-Plan-{name}.md
  ✓ Test skeletons generated ({language} syntax)

PRD status: Backlog Created

Next steps:
1. Assign issues to branch: /assign-branch #epic #story... branch/name
2. Start work: work #{story}
```

## Error Handling

| Situation | Response |
|---|---|
| PRD issue not found | "Issue #N not found. Check the number?" |
| Missing prd label | "Issue #N does not have 'prd' label." |
| PRD path not in body | "Could not find PRD document link in issue body." |
| PRD file not found | "PRD not found at <path>." |
| Test plan not found | "Warning: No test plan found. Stories created without embedded test cases." |
| Test plan not approved | BLOCK with approval instructions |
| No epics in PRD | "PRD contains no epics. Add epics before creating backlog." |

**End of /create-backlog Command**
