---
version: "v0.58.0"
description: Create GitHub epics/stories from PRD (project)
argument-hint: "<issue-number> (e.g., 151)"
---
<!-- MANAGED -->
# /create-backlog
Create GitHub epics and stories from an approved PRD with embedded TDD test cases.
---
## Arguments
| Argument | Description |
|----------|-------------|
| `<prd-issue-number>` | PRD tracking issue number (e.g., `151` or `#151`) |
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Create Todo List:** Use `TodoWrite` from steps below
2. **Track Progress:** Mark todos `in_progress` → `completed`
3. **Resume Point:** Todos show where to continue if interrupted
---
## Prerequisites
- PRD tracking issue exists with `prd` label
- PRD issue body contains link to `PRD/PRD-[Name].md`
- Test plan exists: `PRD/[Name]/Test-Plan-[Name].md`
- Test plan approval issue is **closed** (approved)
---
## Phase 1: Fetch and Validate PRD Issue
**Step 1: Parse issue number** — Accept `151` or `#151` format
**Step 2: Fetch and validate issue**
```bash
gh issue view $issue_num --json labels,body --jq '.labels[].name' | grep -q "prd"
```
If not PRD issue: Error with message about missing 'prd' label.
**Step 3: Extract PRD document path**
Pattern: `/PRD\/[A-Za-z0-9_-]+\/PRD-[A-Za-z0-9_-]+\.md/`
---
## Phase 1b: Branch Validation
**BLOCKING:** Backlog creation requires an active branch.
**Step 1:** Check for branch tracker: `gh pmu branch current`
**Step 2:** Enumerate open branches: `gh pmu branch list`
**Step 3: Branch resolution**
| Branches Found | Action |
|----------------|--------|
| **None** | STOP with error — create branch first |
| **Exactly one** | Auto-assign PRD issue to that branch |
| **Multiple** | Prompt user to select |
If exactly one: `gh pmu move $issue_num --branch current`
If multiple: Use `AskUserQuestion` to let user select, then assign.
---
## Phase 1c: PRD Review Gate
**BLOCKING:** Decomposing an unreviewed PRD risks propagating issues into the epic/story structure.
**Step 1:** Parse PRD tracker body for checkbox: `- [([ x])].*PRD reviewed`
**Step 2: Gate decision**
| Checkbox State | Action |
|----------------|--------|
| Checked (`- [x]`) | Proceed normally — no gate |
| Unchecked (`- [ ]`) or not found | Warn and present options |
**Step 3: Warn and present options (unchecked)**
Use `AskUserQuestion` with two options:
| Option | Description |
|--------|-------------|
| **Run /review-prd first** (Recommended) | Invoke `/review-prd #$issue_num`, then continue |
| **Continue without review** | Proceed, mark checkbox as bypassed |
If "Run /review-prd first": Invoke `/review-prd #$issue_num`, then continue to Phase 2.
If "Continue without review": Update checkbox to `- [x] PRD reviewed (User bypassed PRD review)`, report bypass, continue to Phase 2.
---
## Phase 2: Test Plan Approval Gate
**BLOCKING:** Backlog creation blocked until test plan approved.
**Step 1:** Find test plan approval issue:
```bash
gh issue list --label "test-plan" --label "approval-required" --state open --json number,title
```
**Step 2:** Check approval status
| State | Action |
|-------|--------|
| Open | BLOCK — show message and exit |
| Closed | PROCEED |
| Not found | WARN — proceed but note missing test plan |
---
## Phase 3: Parse PRD for Epics and Stories
**Step 1:** Load PRD document — Read `PRD/{name}/PRD-{name}.md`
**Step 2: Structure extraction**
| PRD Section | Maps To |
|-------------|---------|
| `## Epics` / `### Epic N:` | GitHub issue with `epic` label |
| User stories under epic | GitHub issues with `story` label |
| Acceptance criteria | Story body checkboxes |
| Priority (P0/P1/P2) | Priority field |
---
## Phase 4: Load Test Cases from Test Plan
**Step 1:** Read `PRD/{name}/Test-Plan-{name}.md`
**Step 2:** Match test cases to stories by title or acceptance criteria text
**Step 3:** Load test configuration
**From `Inception/Test-Strategy.md`:** Test framework
**From `Inception/Tech-Stack.md`:** Language (for test syntax)
| Language | Framework | Syntax |
|----------|-----------|--------|
| TypeScript/JavaScript | Vitest/Jest | `test('...', () => { })` |
| Python | pytest | `def test_*():` |
| Go | testing | `func Test*(t *testing.T)` |
| Rust | cargo test | `#[test] fn test_*()` |
| Unknown | Unknown | Generic pseudocode |
**Fallback:** If Test-Strategy.md missing, check `IDPF-Agile/Agile-Core.md` for defaults and warn user.
---
## Phase 5: Create Epic Issues
```bash
gh pmu create \
  --title "Epic: {Epic Name}" \
  --label "epic" \
  --status backlog \
  -F .tmp-epic-body.md
```
**Epic body template:** Include PRD link, PRD tracker, test plan, description, success criteria.
Clean up: `rm .tmp-epic-body.md`
---
## Phase 6: Create Story Issues with Test Cases
```bash
gh pmu create \
  --title "Story: {Story Title}" \
  --label "story" \
  --status backlog \
  -F .tmp-story-body.md
```
Clean up: `rm .tmp-story-body.md`
Link to parent: `gh pmu sub add {epic_number} {story_number} || true`
### Story Body Template
**DEPENDENCY:** Uses `/add-story` Phase 3 Story Body Template (atomic — all sections required).
| Template Section | Source |
|-----------------|--------|
| Description | PRD user story |
| Relevant Skills | `framework-config.json` → `projectSkills` |
| Acceptance Criteria | PRD criteria (checkbox list) |
| Documentation | Standard checkboxes |
| TDD Test Cases | From approved test plan |
| Definition of Done | TDD-specific checklist |
| Priority | PRD priority |
#### TDD Test Cases Extension
Replace placeholder with actual test skeletons:
```markdown
### TDD Test Cases
**Source:** [Test-Plan-{name}.md](PRD/{name}/Test-Plan-{name}.md#epic-story-section)
Write these tests BEFORE implementation:
#### Unit Tests
```{language}
test('{criterion} succeeds with valid input', () => {
  // Arrange -> Act -> Assert
});
test('{criterion} rejects invalid input', () => {
  // Arrange -> Act -> Assert: expect error
});
```
#### Edge Cases
- [ ] Empty/null input handling
- [ ] Boundary values
- [ ] Error recovery
```
#### Definition of Done Extension
```markdown
### Definition of Done
- [ ] All TDD test cases pass
- [ ] Code coverage >= 80%
- [ ] No skipped tests
- [ ] Edge cases handled
- [ ] Acceptance criteria verified
```
---
## Phase 7: Update PRD Status
**Step 1:** Change PRD document status to "Backlog Created"
**Step 2:** Prepend banner to PRD tracker issue:
```markdown
> **PRD In Progress** — When all stories complete, run `/complete-prd {issue_number}`
## Backlog Summary
Epics: {count} | Stories: {count} | Test cases embedded
```
Use `gh pmu edit` with temp file to update the body.
**Step 3:** Add summary comment
**Step 4:** `gh pmu move $issue_num --status in_progress`
---
## Phase 8: Skill Suggestions (Optional)
**Step 1:** Check `framework-config.json` for `skillSuggestions: false` (skip if false)
**Step 2:** Run keyword matching:
```bash
node .claude/scripts/shared/lib/skill-keyword-matcher.js \
  --content-file .tmp-skill-content.txt \
  --installed "{comma-separated projectSkills from framework-config.json}"
rm .tmp-skill-content.txt
```
Parse JSON output: array of `{skill, matchedKeywords}` objects. Already-installed skills excluded automatically.
**Step 3:** If matches found, display table
**ASK USER:** Install suggested skills? (y/n/select)
| Response | Action |
|----------|--------|
| `y` | Install all matched skills |
| `n` | Skip |
| `select` | Present numbered list |
**Step 4:** Install via `node .claude/scripts/shared/install-skill.js {skill-names...}`
---
## Output Summary
```
Backlog created from PRD: PRD/{name}/PRD-{name}.md
Epics: {E} | Stories: {S}
Test cases: {T} embedded ({language} syntax)
Skills suggested: {count} (installed: {installed_count})
PRD status: Backlog Created
Next: gh pmu move #{story} --branch current | work #{story}
```
---
## Error Handling
| Situation | Response |
|-----------|----------|
| PRD issue not found | "Issue #N not found" |
| Missing prd label | "Issue #N does not have 'prd' label" |
| PRD path not in body | "Could not find PRD document link" |
| PRD file not found | "PRD not found at <path>" |
| Test plan not found | Warning: Stories created without test cases |
| Test plan not approved | BLOCK with approval instructions |
| No epics in PRD | "PRD contains no epics" |
---
**End of /create-backlog Command**
