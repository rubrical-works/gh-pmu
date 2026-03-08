---
version: "v0.58.0"
description: Review a test plan against its PRD (project)
argument-hint: "#issue [--force]"
---

<!-- MANAGED -->
# /review-test-plan
Reviews a TDD test plan document linked from a GitHub issue, cross-referencing against the source PRD for coverage completeness. Tracks review history with metadata fields and a Review Log table.
Unlike `/review-issue`, reads two linked documents (test plan and PRD) and performs systematic AC-to-test-case cross-referencing.
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue body must contain `**Test Plan:**` linking to the test plan file
- Issue body must contain `**PRD:**` linking to the source PRD file
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to the test plan (e.g., `#42` or `42`) |
| `--mode` | No | Transient review mode override: `solo`, `team`, or `enterprise`. Does not modify `framework-config.json`. |
| `--force` | No | Force re-review even if issue has `reviewed` label |
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Create Todo List:** Use `TodoWrite` from steps below
2. **Track Progress:** Mark todos `in_progress` -> `completed` as you work
3. **Post-Compaction:** Re-read spec and regenerate todos
---
## Workflow
### Step 1: Resolve Issue and Documents
```bash
gh issue view $ISSUE --json number,title,body,state,labels
```
**If not found:** -> **STOP**
**If closed:** Ask user to confirm.
**Early-exit gate:** If the issue has the `reviewed` label and `--force` is NOT present, skip the full review:
```
"Issue #$ISSUE already reviewed (Review #N). Use --force to re-review."
```
Extract the review count from the `**Reviews:** N` field in the issue body (if present). -> **STOP**
**If `--force` is present:** Bypass the early-exit gate and proceed with full review.
Extract document paths from issue body:
```
Pattern: **Test Plan:** PRD/[Name]/Test-Plan-[Name].md
Pattern: **PRD:** PRD/[Name]/PRD-[Name].md
```
**If `**Test Plan:**` or `**PRD:**` field missing:** -> **STOP**
Read both documents. **If either file not found:** -> **STOP**
### Step 2: Perform Review
Two-phase approach: **auto-evaluate objective criteria**, then **ask user about subjective criteria**. Filter by `reviewMode` (or `--mode` override).
**Step 2a: Load reviewMode**
Parse `--mode` from arguments if provided. Invalid values produce clear error.
```javascript
const { getReviewMode, shouldEvaluate } = require('./.claude/scripts/shared/lib/review-mode.js');
// modeOverride is the --mode argument value (null if not provided)
const mode = getReviewMode(process.cwd(), modeOverride);
```
**Hint:** Display mode and override instructions.
Use `shouldEvaluate(criterionId, process.cwd(), modeOverride)` to filter criteria.
**Step 2b: Auto-Evaluate Objective Criteria**
**Coverage Analysis (P0):**
1. Extract all `- [ ]` acceptance criteria from every PRD user story
2. For each AC, verify corresponding test case exists in test plan
3. Report coverage: full, partial, or none per story
**Structural Checks:**
| Criterion | Auto-Check Method |
|-----------|-------------------|
| AC coverage | Cross-reference PRD `- [ ]` items against test cases -- report coverage % |
| Test framework specified | Check for framework/tooling section |
| Test levels defined | Check for unit/integration/E2E categorization |
| Story-to-test mapping | Verify each PRD story has corresponding test section |
| Error scenarios present | Check for negative/error test cases per story |
| Boundary conditions tested | Check for boundary value tests (empty, max, off-by-one, null) |
| Failure modes covered | Check for failure tests (network errors, invalid data, timeouts, permission denied) |
| Integration points mapped | Check for integration tests covering component interactions |
| Component interactions verified | Check data flow between components (not just units) |
| Data flow boundaries tested | Check tests at data transformation points |
| E2E scenarios cover critical journeys | Check E2E test cases map to PRD user workflows |
| E2E happy paths and error paths | Verify E2E includes both success and failure scenarios |
| E2E scenarios map to PRD requirements | Cross-reference E2E against PRD for traceability |
| Framework consistent with test strategy | If `Inception/Test-Strategy.md` exists, verify alignment |
| Coverage targets realistic | Flag unrealistically high (100%) or low (<60%) targets |
| Test coverage proportionate | Verify depth proportionate to PRD scope. Flag complex stories with shallow coverage. |
Present coverage summary before asking subjective questions.
**Step 2c: Ask Subjective Criteria**
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
**Conditional follow-up:** If user selects ⚠️ or ❌, ask conversationally for specifics.
Collect into: **Strengths**, **Concerns**, **Recommendations**.
**Coverage gaps as bullet-point concerns** (not tables) for `/resolve-review` compatibility.
Determine recommendation:
- **Ready for approval** -- All ACs have test cases, no blocking concerns
- **Ready with minor gaps** -- Small coverage gaps
- **Needs revision** -- Significant coverage gaps
- **Needs major rework** -- Fundamental coverage issues
### Step 3: Update Test Plan Metadata
**Update `**Reviews:** N` field:** Increment if exists, add `**Reviews:** 1` after metadata fields before first `---`.
### Step 4: Update Review Log
**If `## Review Log` exists:** Append new row.
**If not:** Append at end of file.
```markdown
---

## Review Log

| # | Date | Reviewer | Findings Summary |
|---|------|----------|------------------|
| 1 | YYYY-MM-DD | Claude | [Brief one-line summary of findings] |
```
Append-only -- **never edit or delete existing rows**.
Write updated file. **If file write fails:** -> **STOP**
### Step 5: Post Issue Comment
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
**Backwards compatibility:** `### Findings` header and emoji markers unchanged for `/resolve-review` parser. `#### Auto-Evaluated` and `#### User-Evaluated` are additive.
```bash
gh issue comment $ISSUE -F .tmp-review-comment.md
rm .tmp-review-comment.md
```
**If comment post fails:** Warn and continue (non-blocking).
### Step 5.5: Assign Review Outcome Label (Conditional)
If recommendation starts with "Ready for":
```bash
gh issue edit $ISSUE --add-label=reviewed --remove-label=pending
```
If NOT "Ready for" (Needs minor revision, Needs revision, Needs major rework):
```bash
gh issue edit $ISSUE --add-label=pending --remove-label=reviewed
```
### Step 5.6: Approval Gate AC Check-Off (Conditional)
**Only when recommendation is "Ready for approval":**
Automatically checks off acceptance criteria on the approval issue that passed review.
1. Export: `gh pmu view $ISSUE --body-stdout > .tmp-$ISSUE.md`
2. For each `- [ ]` checkbox: if criterion **passed** (✅): replace with `- [x]`. If **failed or flagged** (❌ or ⚠️): leave unchecked.
3. Update: `gh pmu edit $ISSUE -F .tmp-$ISSUE.md && rm .tmp-$ISSUE.md`
4. Move to `in_review`: `gh pmu move $ISSUE --status in_review`
5. Report: `Approval gate: X/Y criteria checked off. Issue #$ISSUE moved to in_review. Run /done #$ISSUE to close.`
**If not "Ready for approval":** Skip entirely -- no AC check-off, no status transition.
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
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." -> STOP |
| Issue missing `**Test Plan:**` field | "Issue #N does not link to a test plan file." -> STOP |
| Issue missing `**PRD:**` field | "Issue #N does not link to a PRD file." -> STOP |
| Test plan file not found | "Test plan file not found: `{path}`." -> STOP |
| PRD file not found | "PRD file not found: `{path}`." -> STOP |
| Issue closed | "Issue #N is closed. Review anyway? (y/n)" -> ask user |
| File write fails | "Failed to update test plan file: {error}" -> STOP |
| Comment post fails | Warn, continue (file already updated) |
| No metadata section | Create metadata field before first `---` separator |
| PRD has no acceptance criteria | Flag as critical concern |
---
**End of /review-test-plan Command**
