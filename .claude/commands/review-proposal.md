---
version: "v0.58.0"
description: Review a proposal with tracked history (project)
argument-hint: "#issue [--force]"
---

<!-- EXTENSIBLE -->
# /review-proposal
Reviews a proposal document linked from a GitHub issue, tracking review history with metadata fields and a Review Log table. Evaluates completeness, consistency, feasibility, and quality.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command review-proposal`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue body must contain `**File:** Proposal/[Name].md` linking to the proposal
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to the proposal (e.g., `#42` or `42`) |
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
### Step 1: Resolve Issue and Proposal File
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
Extract proposal file path from issue body:
```
Pattern: **File:** Proposal/[Name].md
```
**If `**File:**` field missing:** -> **STOP**
Read the proposal file. **If file not found:** -> **STOP**

<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 1c: Construction Context Discovery
Search Construction artifact directories for files related to the proposal:
1. Extract keywords from the proposal title and file name
2. Grep `Construction/Design-Decisions/` and `Construction/Tech-Debt/` for those keywords (case-insensitive)
3. Also check for issue number references (`Issue #$ISSUE`)
4. For each match, extract: file path, title, date
Output as `### Construction Context` section in review comment:
```
### Construction Context
Design Decisions: N found | Tech Debt: M found
- 📄 `Construction/Design-Decisions/YYYY-MM-DD-topic.md` — "Title" (YYYY-MM-DD)
```
**No-match path:** Report `No Construction context found for this proposal.`
### Step 1b: Extension Loading
**If `--with` is specified:**
1. Read `.claude/metadata/review-extensions.json`
2. Parse `--with` value: `--with all` loads all; `--with security,performance` loads specified (trim spaces)
3. For each extension ID: look up `source` path, read criteria file, extract **Proposal Review Questions** section
4. Unknown extension: warn with available list
5. Store loaded criteria for Step 2
**Error Handling:** Registry not found/malformed -> fall back to standard review. Criteria file not found -> skip domain. All missing -> standard review only.
**If `--with` is not specified:** Skip extension loading.
### Step 2: Perform Review
Evaluate using two-phase approach: **auto-evaluate objective criteria**, then **ask user about subjective criteria** via `AskUserQuestion`.
**Step 2a: Load reviewMode**
Parse `--mode` from arguments if provided. Invalid values produce clear error.
```javascript
const { getReviewMode } = require('./.claude/scripts/shared/lib/review-mode.js');
// modeOverride is the --mode argument value (null if not provided)
const mode = getReviewMode(process.cwd(), modeOverride);
```
**Hint:** Display mode and override instructions.

<!-- USER-EXTENSION-START: criteria-customize -->
<!-- USER-EXTENSION-END: criteria-customize -->

**Step 2b: Auto-Evaluate Objective Criteria**
Read proposal file and auto-evaluate. Do NOT ask the user.
| Criterion | Auto-Check Method |
|-----------|-------------------|
| Required sections present | Check for: Problem Statement, Proposed Solution, Acceptance Criteria, Out of Scope |
| Status field present | Check for `**Status:**` metadata field |
| Cross-references valid | Verify file paths mentioned exist on disk |
| Acceptance criteria defined | Check for `- [ ]` items or numbered criteria |
| Prerequisites documented | Check for prerequisites/dependencies section |
| No internal contradictions | Verify solution addresses problem; out-of-scope not duplicated in solution |
| Solution detail sufficient | Check for named files, APIs, data structures, implementation steps (not just prose) |
| Alternatives considered | Check for alternatives/tradeoffs section with at least one rejected approach |
| Impact assessment present | Check for impact/risk/effort section |
| Implementation criteria match solution | Cross-reference ACs against solution -- each AC should map to a solution element |
| Edge cases and error handling | Check for error handling, edge case, or failure mode sections |
| Proposal self-contained | Check external references explained inline |
| Writing clear and unambiguous | Check for vague language ("should work", "might need", "probably"), undefined terms |
| Technical feasibility | Assess technical achievability: complexity/risk factors, dependency availability, scope clarity, effort proportionality. Present concerns with evidence. |
| Test coverage proportionate | For non-trivial scope, check for testing strategy or test-related ACs. Simple single-file changes: preferred but not required. Report scope vs testing with evidence. |
| Diagrams verified | If `**Diagrams:**` lists file paths (not "None"), verify each file exists on disk. Missing files → ❌. "None" or absent → skip. |
**Step 2c: Ask Subjective Criteria**
**Scope Context Display:** Before asking the scope question, extract the scope section from the proposal:
1. Search for scope-related sections: `## Scope`, `## In-Scope`, `## Out of Scope`, `**In-Scope:**`, etc.
2. Extract scope content (summarize if >10 lines)
3. If no scope section found: skip preview, proceed normally
Display before `AskUserQuestion`:
```
Scope preview:
  In-Scope: [extracted in-scope items]
  Out-of-Scope: [extracted out-of-scope items]
```
```javascript
AskUserQuestion({
  questions: [
    {
      question: "Is the scope appropriate — neither too broad nor too narrow?",
      header: "Scope",
      options: [
        { label: "Appropriate ✅", description: "Scope is well-bounded, achievable, and addresses the core problem" },
        { label: "Needs adjustment ⚠️", description: "Slightly too broad or missing a key aspect" },
        { label: "Problematic ❌", description: "Scope is too large, too vague, or misses the core problem" }
      ],
      multiSelect: false
    }
  ]
});
```
**Conditional follow-up:** If user selects ⚠️ or ❌, ask conversationally for specifics.
**Partial reviews are valid** -- if user declines, record as "⊘ Skipped" and continue.
**Step 2d: Extension Criteria** (if `--with` specified)
For each loaded extension, evaluate against Proposal Review Questions. Auto-evaluate objective; ask about subjective.
**Step 2e: Collect All Findings**
Merge into: **Strengths**, **Concerns**, **Recommendations**.
Determine recommendation:
- **Ready for implementation** -- No blocking concerns
- **Ready with minor revisions** -- Small issues that don't block
- **Needs revision** -- Significant concerns
- **Needs major rework** -- Fundamental issues
**If extensions loaded:** Present as separate sections. Extensions can **escalate** but not downgrade.
**Applicability Filtering:** Omit domains with no findings. At least one domain section must appear when `--with` used; otherwise fallback.
### Step 3: Update Proposal Metadata
**Update `**Reviews:** N` field:** Increment if exists, add `**Reviews:** 1` after metadata fields (after Status, Created, Author, Issue lines) before `---`.
### Step 4: Update Review Log
**If `## Review Log` exists:** Append new row.
**If not:** Insert before `**End of Proposal**` marker, or append at end.
```markdown
---

## Review Log

| # | Date | Reviewer | Findings Summary |
|---|------|----------|------------------|
| 1 | YYYY-MM-DD | Claude | [Brief one-line summary of findings] |
```
Append-only -- **never edit or delete existing rows**.
Write updated proposal file. **If file write fails:** -> **STOP**
### Step 5: Post Issue Comment
```markdown
## Proposal Review #N — YYYY-MM-DD

**File:** Proposal/[Name].md
**Total Reviews:** N
**Extensions Applied:** {list of applied extensions, or "None"}

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

[Ready for implementation | Ready with minor revisions | Needs revision | Needs major rework]
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

<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

### Step 6: Report Summary
```
Review #N complete for Proposal: [Title]
  File: Proposal/[Name].md
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
| Issue missing `**File:**` field | "Issue #N does not link to a proposal file." -> STOP |
| Proposal file not found | "Proposal file not found: `{path}`." -> STOP |
| Issue closed | "Issue #N is closed. Review anyway? (y/n)" -> ask user |
| File write fails | "Failed to update proposal file: {error}" -> STOP |
| Comment post fails | Warn, continue (file already updated) |
| No metadata section | Create metadata field before first `---` separator |
---
**End of /review-proposal Command**
