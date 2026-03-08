---
version: "v0.58.0"
description: Review a PRD with tracked history (project)
argument-hint: "#issue [--force]"
---

<!-- EXTENSIBLE -->
# /review-prd
Reviews a PRD document linked from a GitHub issue, tracking review history with metadata fields and a Review Log table. Evaluates requirements completeness, user stories, acceptance criteria, NFRs, and test plan alignment.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command review-prd`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue body must reference the PRD file path
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to the PRD (e.g., `#42` or `42`) |
| `--with` | No | Comma-separated domain extensions (e.g., `--with security,performance`) or `--with all` |
| `--mode` | No | Transient review mode override: `solo`, `team`, or `enterprise`. Does not modify `framework-config.json`. |
| `--force` | No | Force re-review even if issue has `reviewed` label |
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo item for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` -> `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction
**Todo Rules:** One todo per numbered step; one todo per active extension; skip commented-out extensions.
---
## Workflow
### Step 1: Resolve Issue and PRD File
```bash
gh issue view $ISSUE --json number,title,body,state,labels
```
**If not found:** `"Issue #$ISSUE not found."` -> **STOP**
**If closed:** `"Issue #$ISSUE is closed. Review anyway? (y/n)"` -- proceed only if user confirms.
**Early-exit gate:** If the issue has the `reviewed` label and `--force` is NOT present, skip the full review:
```
"Issue #$ISSUE already reviewed (Review #N). Use --force to re-review."
```
Extract the review count from the `**Reviews:** N` field in the issue body (if present). -> **STOP**
**If `--force` is present:** Bypass the early-exit gate and proceed with full review.
Extract PRD file path from issue body. Look for:
- `**File:** PRD/[Name]/PRD-[Name].md`
- `**PRD:** PRD/[Name]/PRD-[Name].md`
- Direct path reference to a `PRD/` file
**If no PRD file reference found:** -> **STOP**
Read the PRD file. **If file not found:** -> **STOP**
### Step 2: Locate Test Plan
Check for `PRD/{Name}/Test-Plan-{Name}.md` in same directory.
**If exists:** Read for inclusion in review.
**If not found:** Warning, continue with PRD-only review (non-blocking).
**If exists but empty:** Note in findings.

<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 2b: Extension Loading
**If `--with` is specified:**
1. Read `.claude/metadata/review-extensions.json`
2. Parse `--with` value: `--with all` loads all; `--with security,performance` loads specified (trim spaces)
3. For each extension ID: look up `source` path, read criteria file, extract **PRD Review Questions** section
4. Unknown extension: warn with available list
5. Store loaded criteria for Step 3
**Error Handling:** Registry not found/malformed -> fall back to standard review. Criteria file not found -> skip domain. All missing -> standard review only.
**If `--with` is not specified:** Skip extension loading.
### Step 3: Perform Review
Evaluate using two-phase approach: **auto-evaluate objective criteria**, then **ask user about subjective criteria** via `AskUserQuestion`.
**Step 3a: Load reviewMode**
Parse `--mode` from arguments if provided. Invalid values produce clear error.
```javascript
const { getReviewMode } = require('./.claude/scripts/shared/lib/review-mode.js');
// modeOverride is the --mode argument value (null if not provided)
const mode = getReviewMode(process.cwd(), modeOverride);
```
**Hint:** Display mode and override instructions.

<!-- USER-EXTENSION-START: criteria-customize -->
<!-- USER-EXTENSION-END: criteria-customize -->

**Step 3b: Auto-Evaluate Objective Criteria**
Read PRD (and test plan if present) and auto-evaluate. Do NOT ask the user.
| Criterion | Auto-Check Method |
|-----------|-------------------|
| Required sections present | Check for: Summary, Problem Statement, Proposed Solution, User Stories, Acceptance Criteria, Out of Scope, NFRs |
| Scope boundaries defined | Check for explicit in-scope / out-of-scope sections |
| Success criteria measurable | Check for quantitative language, verifiable outcomes |
| Story format compliance | Verify "As a ... I want ... So that ..." pattern |
| All stories have ACs | Check each story/epic has `- [ ]` items |
| Story numbering consistent | Verify sequential numbering and cross-references |
| Story priorities assigned | Check for P0/P1/P2 indicators |
| Stories sized appropriately | Flag stories with >8 ACs or overly broad scope |
| AC cover happy paths and error cases | Verify both success and error/failure handling |
| Edge cases identified | Check for boundary conditions, empty inputs, concurrent access |
| Cross-references valid | Verify file paths, issue numbers, proposal references exist |
| Performance requirements specified | Check NFR section for performance targets |
| Security considerations documented | Check NFR section for security requirements |
| Scalability or availability requirements | Check NFR section for scalability/availability targets |
| Test plan presence | Check if test plan file exists (from Step 2) |
| AC coverage (if test plan) | Cross-reference `- [ ]` ACs against test cases -- report coverage % |
| Integration test scenarios (if test plan) | Check for integration test cases |
| E2E test scenarios (if test plan) | Check for E2E scenarios covering critical journeys |
| Test coverage approach documented | Check for coverage targets and strategy section |
| Test coverage proportionate | Verify test requirements proportionate to story scope. Flag stories with complex scope lacking test requirements. |
**Step 3c: Ask Subjective Criteria**
Ask the user only about criteria requiring human judgment.
**Decomposition context preview:** Before presenting the `AskUserQuestion`, emit a concise summary of the PRD's epic/story structure so the user has context for the Decomposition question:
```
Decomposition summary:
  Epic 1: [Epic title] ([N] stories)
    1.1 [Story title], 1.2 [Story title], ...
  Epic 2: [Epic title] ([N] stories)
    2.1 [Story title], 2.2 [Story title], ...
```
Parse epic and story titles from the PRD content already loaded in Step 3b. Each epic should show its story count and a comma-separated list of story titles/numbers.
```javascript
AskUserQuestion({
  questions: [
    {
      question: "Are the acceptance criteria specific enough to be independently testable? (Auto-checks verified structure; this asks about semantic quality.)",
      header: "AC Quality",
      options: [
        { label: "Testable ✅", description: "ACs are specific, measurable, and independently verifiable" },
        { label: "Mostly testable ⚠️", description: "Most ACs are testable but some are vague or overlapping" },
        { label: "Not testable ❌", description: "ACs are too vague, subjective, or not independently verifiable" }
      ],
      multiSelect: false
    },
    {
      question: "Does the PRD adequately decompose the problem — are the epics/stories the right granularity for this scope?",
      header: "Decomposition",
      options: [
        { label: "Well decomposed ✅", description: "Stories are right-sized, epics group logically, nothing missing" },
        { label: "Minor issues ⚠️", description: "Some stories too large or grouping could improve" },
        { label: "Needs rework ❌", description: "Stories too coarse, missing key functionality, or poorly grouped" }
      ],
      multiSelect: false
    }
  ]
});
```
**Conditional follow-up:** If user selects ⚠️ or ❌, ask conversationally for specifics.
**Step 3d: Extension Criteria** (if `--with` specified)
For each loaded extension, evaluate PRD against PRD Review Questions. Auto-evaluate objective; ask about subjective.
**Step 3e: Collect All Findings**
Merge into: **Strengths**, **Concerns**, **Recommendations**.
Determine recommendation:
- **Ready for backlog creation** -- No blocking concerns
- **Ready with minor revisions** -- Small issues that don't block
- **Needs revision** -- Significant concerns
- **Needs major rework** -- Fundamental issues
**If extensions loaded:** Present as separate sections. Extensions can **escalate** but not downgrade.
**Applicability Filtering:** Omit domains with no findings. At least one domain section must appear when `--with` used; otherwise fallback.
### Step 4: Update PRD Metadata
**Update `**Reviews:** N` field:** Increment if exists, add `**Reviews:** 1` before first `---` if not.
### Step 5: Update Review Log
**If `## Review Log` exists:** Append new row.
**If not:** Insert before `**End of PRD**` marker, or append at end (DD14 fallback).
```markdown
---

## Review Log

| # | Date | Reviewer | Findings Summary |
|---|------|----------|------------------|
| 1 | YYYY-MM-DD | Claude | [Brief one-line summary of findings] |
```
Append-only -- **never edit or delete existing rows**.
Write updated PRD file. **If file write fails:** -> **STOP**
### Step 6: Post Issue Comment
```markdown
## PRD Review #N — YYYY-MM-DD

**File:** PRD/[Name]/PRD-[Name].md
**Total Reviews:** N
**Extensions Applied:** {list of applied extensions, or "None"}
**Test Plan:** [Reviewed | Not found]

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

[Ready for backlog creation | Ready with minor revisions | Needs revision | Needs major rework]
```
**Backwards compatibility:** `### Findings` header and emoji markers unchanged for `/resolve-review` parser. `#### Auto-Evaluated` and `#### User-Evaluated` are additive.
```bash
gh issue comment $ISSUE -F .tmp-review-comment.md
rm .tmp-review-comment.md
```
**If comment post fails:** Warn and continue (non-blocking).
### Step 6.5: Assign Review Outcome Label (Conditional)
If recommendation starts with "Ready for":
```bash
gh issue edit $ISSUE --add-label=reviewed --remove-label=pending
```
If NOT "Ready for" (Needs minor revision, Needs revision, Needs major rework):
```bash
gh issue edit $ISSUE --add-label=pending --remove-label=reviewed
```
### Step 6.6: AC Check-Off (Conditional)
**Only when recommendation is "Ready for backlog creation":**
Automatically checks off acceptance criteria on the PRD issue that passed review.
1. Export: `gh pmu view $ISSUE --body-stdout > .tmp-$ISSUE.md`
2. For each `- [ ]` checkbox: if criterion **passed** (✅): replace with `- [x]`. If **failed or flagged** (❌ or ⚠️): leave unchecked.
3. Update: `gh pmu edit $ISSUE -F .tmp-$ISSUE.md && rm .tmp-$ISSUE.md`
4. Report: `AC check-off: X/Y criteria checked off on issue #$ISSUE.`
**No status transition** -- `/create-backlog` owns the `in_progress` transition.
**If not "Ready for backlog creation":** Skip entirely.

<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

### Step 7: Report Summary
```
Review #N complete for PRD: [Title]
  File: PRD/[Name]/PRD-[Name].md
  Test Plan: [Reviewed | Not found]
  Recommendation: [recommendation]
  Reviews: N (updated)
  Review Log: [appended | created]
  Issue comment: [posted | failed]
```
**If `--with` is not specified**, append:
```
Tip: Use --with security,performance to add domain-specific review criteria.
Available: security, accessibility, performance, chaos, contract, qa, seo, privacy (or --with all)
```
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." -> STOP |
| Issue missing PRD file reference | "Issue #N does not link to a PRD file." -> STOP |
| PRD file not found | "PRD file not found: `{path}`." -> STOP |
| Issue closed | "Issue #N is closed. Review anyway? (y/n)" -> ask user |
| Test plan not found | Warning, continue with PRD-only review |
| Test plan empty | Note in findings, continue |
| File write fails | "Failed to update PRD file: {error}" -> STOP |
| Comment post fails | Warn, continue (file already updated) |
| No metadata section | Create metadata field before first `---` separator |
---
**End of /review-prd Command**
