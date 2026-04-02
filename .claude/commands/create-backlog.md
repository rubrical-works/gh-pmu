---
version: "v0.79.0"
description: Create GitHub epics/stories from PRD (project)
argument-hint: "<issue-number> (e.g., 151)"
copyright: "Rubrical Works (c) 2026"
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
1. **Create Todo List:** Use `TodoWrite` to create todos from the steps below
2. **Track Progress:** Mark todos `in_progress` -> `completed` as you work
3. **Resume Point:** If interrupted, todos show where to continue
---
## Prerequisites
- PRD tracking issue exists with `prd` label
- PRD issue body contains link to `PRD/PRD-[Name].md`
- Test plan exists: `PRD/[Name]/Test-Plan-[Name].md`
- Test plan approval issue is **closed** (approved)
---
## Phase 1: Fetch and Validate PRD Issue
**Step 1:** Parse issue number (accept `151` or `#151`).
**Step 2:** Validate issue has `prd` label:
```bash
gh issue view $issue_num --json labels,body --jq '.labels[].name' | grep -q "prd"
```
If not a PRD issue: error and stop.
**Step 3:** Extract PRD path: `Pattern: /PRD\/[A-Za-z0-9_-]+\/PRD-[A-Za-z0-9_-]+\.md/`
---
## Phase 1b: (Removed)
Branch assignment is not performed during backlog creation. Use `/assign-branch` after.
---
## Phase 1c: PRD Review Gate
**BLOCKING:** Decomposing an unreviewed PRD risks propagating issues.
Parse tracker body for `- [([ x])].*PRD reviewed` checkbox.
| Checkbox State | Action |
|----------------|--------|
| Checked (`[x]`) | Proceed |
| Unchecked/not found | Warn and present options |
Options via `AskUserQuestion`:
- **Run /review-prd first** (Recommended)
- **Continue without review** -- update checkbox as bypassed
---
## Phase 2: Test Plan Approval Gate
**BLOCKING:** Blocked until test plan approved.
```bash
gh issue list --label "test-plan" --label "approval-required" --state open --json number,title
```
| State | Action |
|-------|--------|
| **Open** | BLOCK |
| **Closed** | PROCEED |
| **Not found** | WARN and proceed |
---
## Phase 3: Parse PRD for Epics and Stories
Read `PRD/{name}/PRD-{name}.md` and extract:
- Epics -> `epic` label issues
- Stories -> `story` label issues
- Acceptance criteria -> checkboxes
- Priority -> Priority field
---
## Phase 4: Load Test Cases from Test Plan
Read `PRD/{name}/Test-Plan-{name}.md`. Match test cases to stories by title and AC text.
**Test config from:** `Inception/Test-Strategy.md` (framework), `Inception/Tech-Stack.md` (language).
| Language | Framework | Syntax |
|----------|-----------|--------|
| TypeScript/JavaScript | Vitest/Jest | `test('...', () => { })` |
| Python | pytest | `def test_*():` |
| Go | testing | `func Test*(t *testing.T)` |
| Rust | cargo test | `#[test] fn test_*()` |
| Unknown | Unknown | Pseudocode |
**Fallback:** Check `{frameworkPath}/IDPF-Agile/Agile-Core.md` defaults, warn.
---
## Phase 5: Create Epic Issues
```bash
gh pmu create --title "Epic: {Epic Name}" --label "epic" --status backlog --priority {highest_story_priority} -F .tmp-epic-body.md
```
**Priority rule:** Epic priority = highest priority among child stories. If none specified, use PRD-level default.
Epic body: PRD reference, tracker, test plan, description, success criteria. Clean up temp file.
---
## Phase 6: Create Story Issues with Test Cases
```bash
gh pmu create --title "Story: {Story Title}" --label "story" --status backlog --priority {prd_priority} -F .tmp-story-body.md
```
**Priority rule:** Story priority = PRD-specified priority. Use PRD-level default if none per-story.
Link: `gh pmu sub add {epic_number} {story_number} || true`
### Story Body Template
**DEPENDENCY:** Uses `/add-story` Phase 3 template (atomic).
| Section | Source |
|---------|--------|
| Description | PRD user story |
| Relevant Skills | `framework-config.json` -> `projectSkills` |
| Acceptance Criteria | PRD ACs (checkboxes) |
| Documentation | Standard checkboxes |
| TDD Test Cases | Extended (replaces placeholder) |
| Definition of Done | Extended (replaces base) |
| Priority | PRD priority |
#### TDD Test Cases Extension
Replace placeholder with test skeletons from test plan. Include unit tests and edge cases.
#### Definition of Done Extension
```markdown
- [ ] All TDD test cases pass
- [ ] Code coverage >= 80%
- [ ] No skipped tests
- [ ] Edge cases handled
- [ ] Acceptance criteria verified
```
---
## Phase 7: Update PRD Status
1. Change status to `Backlog Created`
2. Prepend instruction banner to tracker body
3. Add summary comment
4. Move to in_progress: `gh pmu move $issue_num --status in_progress`
PRD remains open until `/complete-prd`.
---
## Phase 8: Skill Suggestions (Optional)
If `skillSuggestions: false` in config, skip.
Run keyword matcher, display matches, **ASK USER** to install (y/n/select).
Install: `node .claude/scripts/shared/install-skill.js {names...}`
Persist: `persistSuggestions('framework-config.json', confirmedSuggestions, '#ISSUE')`
---
## Output Summary
```
Backlog created from PRD: PRD/{name}/PRD-{name}.md
Epics: #{N}: {Name} ({X} stories)
Total: {E} epics, {S} stories
Test cases: {T} from Test-Plan-{name}.md
Next: assign stories, start work
```
---
## Error Handling
| Situation | Response |
|-----------|----------|
| PRD issue not found | Error |
| Missing prd label | Error |
| PRD path not in body | Error |
| PRD file not found | Error |
| Test plan not found | Warning |
| Test plan not approved | BLOCK |
| No epics | Error |
---
**End of /create-backlog Command**
