---
version: "v0.61.0"
description: Review a test plan against its PRD (project)
argument-hint: "#issue [--force]"
---
<!-- MANAGED -->

# /review-test-plan
Reviews a TDD test plan document linked from a GitHub issue, cross-referencing it against the source PRD for coverage completeness. Tracks review history with metadata fields and a Review Log table.
Unlike `/review-issue`, this command reads two linked documents (the test plan and its source PRD) and performs systematic cross-referencing of acceptance criteria against test cases.

## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue body must contain `**Test Plan:**` linking to the test plan file
- Issue body must contain `**PRD:**` linking to the source PRD file

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to the test plan (e.g., `#42` or `42`) |
| `--mode` | No | Transient review mode override: `solo`, `team`, or `enterprise`. Does not modify `framework-config.json`. |
| `--force` | No | Force re-review even if issue has `reviewed` label |

## Execution Instructions
**REQUIRED:** Before executing:
1. **Create Todo List:** Use `TodoWrite` to create todos from the steps below
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
3. **Post-Compaction:** If resuming after context compaction, re-read this spec and regenerate todos

## Workflow

### Step 1: Resolve Issue and Documents
Look up the issue:
```bash
gh issue view $ISSUE --json number,title,body,state,labels
```
**If not found:** `"Issue #$ISSUE not found."` → **STOP**
**If closed:** `"Issue #$ISSUE is closed. Review anyway? (y/n)"` — proceed only if user confirms.
**Early-exit gate:** If the issue has the `reviewed` label and `--force` is NOT present, skip the full review:
```
"Issue #$ISSUE already reviewed (Review #N). Use --force to re-review."
```
Extract the review count from the `**Reviews:** N` field in the issue body (if present). → **STOP**
**If `--force` is present:** Bypass the early-exit gate and proceed with full review.
Extract document paths from the issue body:
```
Pattern: **Test Plan:** PRD/[Name]/Test-Plan-[Name].md
Pattern: **PRD:** PRD/[Name]/PRD-[Name].md
```
**If `**Test Plan:**` field missing:**
```
Issue #$ISSUE does not link to a test plan file.
Expected `**Test Plan:** PRD/[Name]/Test-Plan-[Name].md` in issue body.
```
→ **STOP**
**If `**PRD:**` field missing:**
```
Issue #$ISSUE does not link to a PRD file.
Expected `**PRD:** PRD/[Name]/PRD-[Name].md` in issue body.
```
→ **STOP**
Read both documents.
**If test plan file not found:**
```
Test plan file not found: `{path}`. Check the path in issue #$ISSUE?
```
→ **STOP**
**If PRD file not found:**
```
PRD file not found: `{path}`. Check the path in issue #$ISSUE?
```
→ **STOP**

### Step 2: Perform Review
Evaluate the test plan using a two-phase approach: **auto-evaluate objective criteria** by reading the test plan and PRD files, then **ask the user only about subjective criteria** via `AskUserQuestion`. Filter criteria by `reviewMode` from `framework-config.json` (or `--mode` override).
**Step 2a: Load reviewMode**
Parse `--mode` from arguments if provided. Invalid values produce a clear error with valid options.
```javascript
const { getReviewMode, shouldEvaluate } = require('./.claude/scripts/shared/lib/review-mode.js');
// modeOverride is the --mode argument value (null if not provided)
const mode = getReviewMode(process.cwd(), modeOverride);
```
**Hint:** Display the active mode and how to override it:
- Without `--mode`: `Reviewing in {mode} mode (override with --mode solo|team|enterprise)`
- With `--mode`: `Reviewing in {mode} mode (--mode override)`
Use `shouldEvaluate(criterionId, process.cwd(), modeOverride)` to filter criteria before evaluation. Criteria not applicable to the current mode should be skipped.
**Step 2b: Auto-Evaluate Objective Criteria**
Systematically analyze the test plan and PRD without user input:
**Coverage Analysis (P0):**
1. **Extract all acceptance criteria** from every user story in the PRD (all `- [ ]` items)
2. **For each acceptance criterion**, verify a corresponding test case exists in the test plan
3. **Report coverage status** for each story:
   - Full coverage: all ACs have test cases
   - Partial coverage: some ACs missing test cases
   - No coverage: story has no test cases at all
**Structural Checks:**
| Criterion | Auto-Check Method |
|-----------|-------------------|
| AC coverage | Cross-reference PRD `- [ ]` items against test plan test cases — report coverage % |
| Test framework specified | Check for framework/tooling section (e.g., Jest, Playwright, pytest) |
| Test levels defined | Check for unit/integration/E2E level categorization |
| Story-to-test mapping | Verify each PRD story has a corresponding test section |
| Error scenarios present | Check for negative/error test cases per story |
| Boundary conditions tested | Check for boundary value tests (empty inputs, max values, off-by-one, null/undefined) |
| Failure modes covered | Check for failure mode tests (network errors, invalid data, timeouts, permission denied) |
| Integration points mapped | Check for integration test cases covering interactions between components, epics, or stories |
| Component interactions verified | Check that integration tests cover data flow between components (not just individual units) |
| Data flow boundaries tested | Check for tests at data transformation points (input parsing, output formatting, cross-system boundaries) |
| E2E scenarios cover critical journeys | Check for end-to-end test cases mapping to the PRD's primary user workflows |
| E2E happy paths and error paths | Verify E2E section includes both success and failure scenarios |
| E2E scenarios map to PRD requirements | Cross-reference E2E test cases against PRD requirements for traceability |
| Framework consistent with test strategy | If `Inception/Test-Strategy.md` exists, verify test plan aligns with its guidance |
| Coverage targets realistic | Compare stated coverage targets to project scope — flag unrealistically high (100%) or low (<60%) targets |
| Test coverage proportionate | Assess whether the test plan's depth is proportionate to the PRD's scope. For each PRD story that adds or modifies code paths, verify the test plan includes tests at the appropriate level (unit for logic, integration for component interaction, E2E for user journeys). Flag stories with complex scope (multiple files, refactoring, new modules) that only have shallow test coverage (e.g., only "existing tests pass" assertions). |
**Present auto-evaluation results:**
```
Auto-evaluated (objective criteria):
  ✅ AC coverage — 35/38 ACs mapped to test cases (92%)
  ✅ Test framework specified — Jest + Playwright
  ✅ Test levels defined — unit (15), integration (8), E2E (4)
  ⚠️ Story-to-test mapping — Story 2.4 has no test section
  ⚠️ Error scenarios — 3/10 stories lack negative test cases
  ✅ Boundary conditions — boundary value tests found for input validation
  ✅ Failure modes — timeout, permission, and invalid data failure tests present
  ⚠️ Integration points — 2/5 integration boundaries lack dedicated test cases
  ✅ E2E scenarios — 4 critical user journeys covered
  ⚠️ E2E error paths — no failure scenario E2E tests found
  ✅ Coverage targets — 85% target aligns with project scope
```
Present the coverage summary to the user before asking subjective questions.
**Step 2c: Ask Subjective Criteria**
Ask the user only about criteria requiring human judgment:
```javascript
AskUserQuestion({
  questions: [
    {
      question: "Are the edge cases and error scenarios thorough enough for the project's risk profile?",
      header: "Edge Cases",
      options: [
        { label: "Thorough ✅", description: "Error scenarios, boundary conditions, and failure modes well covered" },
        { label: "Minor gaps ⚠️", description: "Most error cases covered but some stories missing boundary tests" },
        { label: "Significant gaps ❌", description: "Many stories lack error scenarios or failure mode testing" }
      ],
      multiSelect: false
    },
    {
      question: "Is the overall test strategy appropriate for this project's scope and complexity?",
      header: "Strategy",
      options: [
        { label: "Appropriate ✅", description: "Coverage targets realistic, test level balance makes sense for the scope" },
        { label: "Needs refinement ⚠️", description: "Strategy exists but coverage targets or level balance could improve" },
        { label: "Inappropriate ❌", description: "Strategy misaligned with project scope or missing key considerations" }
      ],
      multiSelect: false
    }
  ]
});
```
**Conditional follow-up:** If user selects ⚠️ or ❌ for any subjective criterion, ask conversationally for specifics.
Collect findings into structured categories: **Strengths**, **Concerns**, **Recommendations**.
**Coverage gaps are reported as bullet-point concerns** (not tables) for `/resolve-review` parser compatibility.
Example concern format:
```
- Story 2.3 AC "Source paths resolve" has no corresponding test case
- Story 1.2 missing error condition tests for failed grep matches
```
Determine a recommendation:
- **Ready for approval** — All ACs have test cases, no blocking concerns
- **Ready with minor gaps** — Small coverage gaps that don't block
- **Needs revision** — Significant coverage gaps that should be addressed
- **Needs major rework** — Fundamental coverage issues or missing test sections

### Step 3: Update Test Plan Metadata
Read the current test plan file content.
**Update `**Reviews:** N` field:**
- If `**Reviews:**` field exists: increment the number (e.g., `**Reviews:** 1` → `**Reviews:** 2`)
- If `**Reviews:**` field does not exist: add `**Reviews:** 1` after the existing metadata fields (after `Source`, `PRD`, `Created`, `Approval Issue` lines, before the first `---` separator)

### Step 4: Update Review Log
**If `## Review Log` section exists:** Append a new row to the existing table.
**If `## Review Log` section does not exist:** Append the Review Log section at the very end of the file.
**Review Log format:**
```markdown
---

## Review Log

| # | Date | Reviewer | Findings Summary |
|---|------|----------|------------------|
| 1 | YYYY-MM-DD | Claude | [Brief one-line summary of findings] |
```
Each review appends a new row. **Never edit or delete existing rows** — the log is append-only.
Write the updated test plan file.
**If file write fails:** `"Failed to update test plan file: {error}"` → **STOP**

### Step 5: Post Issue Comment
Post a structured review comment to the GitHub issue:
```markdown
## Test Plan Review #N — YYYY-MM-DD

**Test Plan:** PRD/[Name]/Test-Plan-[Name].md
**PRD:** PRD/[Name]/PRD-[Name].md
**Total Reviews:** N

### Coverage Summary

- Stories with full coverage: X/Y
- Stories with partial coverage: X/Y
- Stories with no coverage: X/Y

### Findings

#### Auto-Evaluated
- ✅ [Criterion] — [evidence]
- ❌ [Criterion] — [what's missing]

#### User-Evaluated
- ✅ [Criterion] — [user assessment]
- ⚠️ [Criterion] — [user concern]

**Strengths:**
- [Strength 1]

**Concerns:**
- [Concern 1]

**Recommendations:**
- [Recommendation 1]

### Recommendation

[Ready for approval | Ready with minor gaps | Needs revision | Needs major rework]
```
**Backwards compatibility:** The `### Findings` section header and emoji markers (✅ ⚠️ ❌) remain unchanged for `/resolve-review` parser compatibility. The `#### Auto-Evaluated` and `#### User-Evaluated` subsections are additive.
```bash
gh issue comment $ISSUE -F .tmp-review-comment.md
rm .tmp-review-comment.md
```
**If comment post fails:** Warn and continue (non-blocking — the test plan file is already updated).

### Step 5.5: Assign Review Outcome Label (Conditional)
If recommendation starts with "Ready for" (i.e., "Ready for approval"):
```bash
gh issue edit $ISSUE --add-label=reviewed --remove-label=pending
```
If recommendation does NOT start with "Ready for" (i.e., "Needs minor revision", "Needs revision", or "Needs major rework"):
```bash
gh issue edit $ISSUE --add-label=pending --remove-label=reviewed
```

### Step 5.6: Approval Gate AC Check-Off (Conditional)
**Only when recommendation is "Ready for approval":**
This step automatically checks off acceptance criteria on the approval issue that passed review, reducing the manual step between review and approval.
1. **Export the approval issue body:**
   ```bash
   gh pmu view $ISSUE --body-stdout > .tmp-$ISSUE.md
   ```
2. **For each `- [ ]` checkbox** in the issue body:
   - If the corresponding criterion **passed** the review (✅ in findings): replace `- [ ]` with `- [x]`
   - If the criterion **failed or was flagged** (❌ or ⚠️ in findings): leave as `- [ ]` (unchecked)
3. **Update the issue body:**
   ```bash
   gh pmu edit $ISSUE -F .tmp-$ISSUE.md && rm .tmp-$ISSUE.md
   ```
4. **Move the approval issue to `in_review`** (enables `/done` closure):
   ```bash
   gh pmu move $ISSUE --status in_review
   ```
5. Report:
   ```
   Approval gate: X/Y criteria checked off. Issue #$ISSUE moved to in_review.
   Run /done #$ISSUE to close the approval gate.
   ```
**If recommendation is NOT "Ready for approval":** Skip this step entirely — no AC check-off, no status transition.

### Step 6: Report Summary
```
Review #N complete for Test Plan: [Title]
  Test Plan: PRD/[Name]/Test-Plan-[Name].md
  PRD: PRD/[Name]/PRD-[Name].md
  Coverage: X/Y stories fully covered
  Recommendation: [recommendation]
  Reviews: N (updated)
  Review Log: [appended | created]
  Issue comment: [posted | failed]
```

## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." → STOP |
| Issue missing `**Test Plan:**` field | "Issue #N does not link to a test plan file." → STOP |
| Issue missing `**PRD:**` field | "Issue #N does not link to a PRD file." → STOP |
| Test plan file not found | "Test plan file not found: `{path}`." → STOP |
| PRD file not found | "PRD file not found: `{path}`." → STOP |
| Issue closed | "Issue #N is closed. Review anyway? (y/n)" → ask user |
| File write fails | "Failed to update test plan file: {error}" → STOP |
| Comment post fails | Warn, continue (file already updated) |
| No metadata section in file | Create metadata field before first `---` separator |
| PRD has no acceptance criteria | Flag as critical concern — cannot verify coverage |
**End of /review-test-plan Command**
