---
version: "v0.61.0"
description: Review a proposal with tracked history (project)
argument-hint: "#issue [--force]"
---
<!-- EXTENSIBLE -->

# /review-proposal
Reviews a proposal document linked from a GitHub issue, tracking review history with metadata fields and a Review Log table. Evaluates completeness, consistency, feasibility, and quality.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command review-proposal`

## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue body must contain `**File:** Proposal/[Name].md` linking to the proposal

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to the proposal (e.g., `#42` or `42`) |
| `--with` | No | Comma-separated domain extensions (e.g., `--with security,performance`) or `--with all` |
| `--mode` | No | Transient review mode override: `solo`, `team`, or `enterprise`. Does not modify `framework-config.json`. |
| `--force` | No | Force re-review even if issue has `reviewed` label |

## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo item for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction
**Todo Rules:** One todo per numbered step; one todo per active extension; skip commented-out extensions.

## Workflow

### Step 1: Resolve Issue and Proposal File
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
Extract the proposal file path from the issue body:
```
Pattern: **File:** Proposal/[Name].md
```
**If `**File:**` field missing:**
```
Issue #$ISSUE does not link to a proposal file.
Expected `**File:** Proposal/[Name].md` in issue body.
```
→ **STOP**
Read the proposal file at the extracted path.
**If file not found:**
```
Proposal file not found: `{path}`. Check the path in issue #$ISSUE?
```
→ **STOP**
<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 1c: Construction Context Discovery
Search Construction artifact directories for files related to the proposal's domain:
1. Extract keywords from the proposal title and file name (e.g., command name, feature name)
2. Grep `Construction/Design-Decisions/` and `Construction/Tech-Debt/` for those keywords (case-insensitive)
3. Also check for issue number references (`Issue #$ISSUE`) in Construction files
4. For each match, extract: file path, title (first `# ` heading), date (from `**Date:**` field)
**Output as `### Construction Context` section in the review comment (Step 5):**
```
### Construction Context
Design Decisions: N found | Tech Debt: M found

- 📄 `Construction/Design-Decisions/2026-02-08-topic.md` — "Title" (2026-02-08)
```
**No-match path:** If no Construction files match the proposal keywords, report:
```
### Construction Context
No Construction context found for this proposal.
```

### Step 1b: Extension Loading
**If `--with` is specified:**
1. Read `.claude/metadata/review-extensions.json`
2. Parse the `--with` value:
   - `--with all` → load all registered extensions
   - `--with security,performance` → load specified extensions (trim spaces from comma-separated list)
3. For each requested extension ID:
   - Look up the `source` path in the registry
   - Resolve the path relative to `frameworkPath` (from `framework-config.json`)
   - Read the criteria file content
   - Extract the **Proposal Review Questions** section
4. If an extension ID is not found in the registry, warn: `"Unknown extension: {id}. Available: security, accessibility, performance, chaos, contract, qa, seo, privacy"`
5. Store loaded criteria for use in Step 2
**Error Handling (Extension Loading):**
- **Registry not found:** `"Review extensions registry not found. Run hub update or check installation."` → fall back to standard review only
- **Registry malformed:** `"Review extensions registry is malformed. Run hub update or check installation."` → fall back to standard review only
- **Criteria file not found:** `"Warning: Review criteria file not found for '{domain}'. Skipping domain. Update hub to resolve."` → continue with remaining domains
- **All criteria files missing:** `"No review criteria files found. Running standard review only."` → fall back to standard review only
**If `--with` is not specified:** Skip extension loading (standard review only).

### Step 2: Perform Review
Evaluate the proposal using a two-phase approach: **auto-evaluate objective criteria** by reading the proposal file, then **ask the user only about subjective criteria** via `AskUserQuestion`.
**Step 2a: Load reviewMode**
Parse `--mode` from arguments if provided. Invalid values produce a clear error with valid options.
```javascript
const { getReviewMode } = require('./.claude/scripts/shared/lib/review-mode.js');
// modeOverride is the --mode argument value (null if not provided)
const mode = getReviewMode(process.cwd(), modeOverride);
```
**Hint:** Display the active mode and how to override it:
- Without `--mode`: `Reviewing in {mode} mode (override with --mode solo|team|enterprise)`
- With `--mode`: `Reviewing in {mode} mode (--mode override)`
<!-- USER-EXTENSION-START: criteria-customize -->
<!-- USER-EXTENSION-END: criteria-customize -->
**Step 2b: Auto-Evaluate Objective Criteria**
Read the proposal file and auto-evaluate structural/factual criteria. Do NOT ask the user about these.
| Criterion | Auto-Check Method |
|-----------|-------------------|
| Required sections present | Check for: Problem Statement, Proposed Solution, Acceptance Criteria, Out of Scope sections |
| Status field present | Check for `**Status:**` metadata field |
| Cross-references valid | Verify any file paths mentioned in the proposal exist on disk |
| Acceptance criteria defined | Check for `- [ ]` checkbox items or numbered criteria list |
| Prerequisites documented | Check for prerequisites/dependencies section if applicable |
| No internal contradictions | Verify solution addresses the stated problem; out-of-scope items not duplicated in solution |
| Solution detail sufficient for implementation | Check for named files, APIs, data structures, or implementation steps (not just high-level prose) |
| Alternatives considered | Check for alternatives/tradeoffs section documenting at least one rejected approach |
| Impact assessment present | Check for impact/risk/effort section covering scope, risk, and estimated effort |
| Implementation criteria match solution | Cross-reference AC items against the proposed solution — each AC should map to a solution element |
| Edge cases and error handling addressed | Check for error handling, edge case, or failure mode sections |
| Proposal self-contained | Check if external references (URLs, other repos, tools) are explained inline without requiring context |
| Writing clear and unambiguous | Evaluate prose for vague language ("should work", "might need", "probably"), undefined terms, or ambiguous scope |
| Technical feasibility | Assess whether the proposed solution is technically achievable. Check for: technical complexity and risk factors (novel tech, unproven approaches, scaling concerns), dependency availability and compatibility (external libraries, APIs, services), scope clarity and implementation feasibility (can the solution be built as described?), resource/effort proportionality (is the effort reasonable for the value?). Present any feasibility concerns with evidence — e.g., "Feasibility concern: proposal references 3 external APIs without fallback strategy" or "Risk: approach requires real-time processing at scale but no performance analysis provided" |
| Test coverage proportionate | When scope is non-trivial (multiple files, refactoring, new modules), check that the proposal mentions a testing strategy or includes test-related acceptance criteria. For simple single-file changes, testing mention is preferred but not required. Report scope assessment vs testing mentions with evidence. |
| Diagrams verified | If `**Diagrams:**` field lists file paths (not "None"), verify each referenced file exists on disk. Report missing files as ❌. If field is "None" or absent, skip this check (not an error). |
| Path Analysis present | Check for `## Path Analysis` section in proposal document. If missing, suggest: "Consider running `/paths #N` to identify scenario paths before PRD conversion." This is non-blocking — the review continues regardless of whether Path Analysis exists. If present, no suggestion displayed. |
| Screen coverage (UI proposals) | If the proposal mentions screens, UI, forms, or user interface elements, check for `## Screen Specs` or `## Mockups` sections. If screens are mentioned but no specs/mockups are referenced, flag: "UI-related proposal lacks screen specs or mockups — consider running `/catalog-screens` and `/mockups`." If the proposal is CLI-only, API-only, or has no UI references, skip this check entirely (do not false-flag non-UI proposals). |
**Present auto-evaluation results:**
```
Auto-evaluated (objective criteria):
  ✅ Required sections present — Problem Statement, Proposed Solution, AC, Out of Scope all found
  ✅ Status field present — "Draft"
  ✅ Cross-references valid — all 3 file paths verified
  ❌ Acceptance criteria not checkboxed — criteria listed as prose, not testable `- [ ]` items
  ⚠️ Prerequisites section missing — no dependencies documented
```
**Step 2c: Ask Subjective Criteria**
Ask the user only about criteria requiring human judgment.
**Scope Context Display:** Before asking the scope question, extract the scope section from the proposal file to present inline context:
1. Search the proposal file for scope-related sections: `## Scope`, `## In-Scope`, `## Out of Scope`, `**In-Scope:**`, `**Out-of-Scope:**`, or similar headings
2. Extract the scope section content
3. If the scope section is lengthy (>10 lines), display a summary rather than the full text
4. **If no scope section is found:** Skip the scope preview display and proceed normally — do not error
Display the scope preview before `AskUserQuestion`:
```
Scope preview:
  In-Scope: [extracted in-scope items]
  Out-of-Scope: [extracted out-of-scope items]
```
Then ask the scope question with the reviewer having full context:
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
**Conditional follow-up:** If user selects ⚠️ or ❌ for any subjective criterion, ask conversationally for specifics.
**Partial reviews are valid** — the user may stop the review at any point. If they decline to answer a subjective question, record it as "⊘ Skipped" and continue.
**Step 2d: Extension Criteria** (if `--with` specified)
For each loaded extension domain, evaluate the proposal against the extension's Proposal Review Questions. Auto-evaluate objective extension criteria; ask the user about subjective ones.
**Step 2e: Collect All Findings**
Merge auto-evaluated and user-evaluated findings into structured categories: **Strengths**, **Concerns**, **Recommendations**.
Determine a recommendation:
- **Ready for implementation** — No blocking concerns
- **Ready with minor revisions** — Small issues that don't block
- **Needs revision** — Significant concerns that should be addressed first
- **Needs major rework** — Fundamental issues with problem statement or approach
**If extensions were loaded (Step 1b):**
For each loaded extension, present findings as a separate section:
```markdown
### Security Review (IDPF-Security)
- [Finding 1]
- [Finding 2]
```
Extension findings can **escalate** the overall recommendation but cannot downgrade it. If any extension raises a blocking concern, the recommendation must reflect it.
**Applicability Filtering:** Omit extension domain sections that produce no applicable findings. Only domains with actual findings appear in the output and in the `**Extensions Applied:**` header. If no domains produce findings, fall back to standard-only review with a warning: `"No domain extensions produced findings. Showing standard review only."` At least one domain section must appear when `--with` is used; otherwise the fallback applies.

### Step 3: Update Proposal Metadata
Read the current proposal file content.
**Update `**Reviews:** N` field:**
- If `**Reviews:**` field exists: increment the number (e.g., `**Reviews:** 1` → `**Reviews:** 2`)
- If `**Reviews:**` field does not exist: add `**Reviews:** 1` after the existing metadata fields (after `Status`, `Created`, `Author`, `Issue`/`Tracking Issue` lines, before the `---` separator)

### Step 4: Update Review Log
**If `## Review Log` section exists:** Append a new row to the existing table.
**If `## Review Log` section does not exist:**
- If `**End of Proposal**` marker exists: insert the Review Log section before it
- If no `**End of Proposal**` marker: append the Review Log section at the very end of the file
**Review Log format:**
```markdown
---

## Review Log

| # | Date | Reviewer | Findings Summary |
|---|------|----------|------------------|
| 1 | YYYY-MM-DD | Claude | [Brief one-line summary of findings] |
```
Each review appends a new row. **Never edit or delete existing rows** — the log is append-only.
Write the updated proposal file.
**If file write fails:** `"Failed to update proposal file: {error}"` → **STOP**

### Step 5: Post Issue Comment
Post a structured review comment to the GitHub issue:
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
**Backwards compatibility:** The `### Findings` section header and emoji markers (✅ ⚠️ ❌) remain unchanged for `/resolve-review` parser compatibility. The `#### Auto-Evaluated` and `#### User-Evaluated` subsections are additive.
```bash
gh issue comment $ISSUE -F .tmp-review-comment.md
rm .tmp-review-comment.md
```
**If comment post fails:** Warn and continue (non-blocking — the proposal file is already updated).

### Step 5.5: Assign Review Outcome Label (Conditional)
If recommendation starts with "Ready for" (i.e., "Ready for implementation"):
```bash
gh issue edit $ISSUE --add-label=reviewed --remove-label=pending
```
If recommendation does NOT start with "Ready for" (i.e., "Needs minor revision", "Needs revision", or "Needs major rework"):
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
**If `--with` is not specified**, append a discoverability tip after the summary:
```
Tip: Use --with security,performance to add domain-specific review criteria.
Available: security, accessibility, performance, chaos, contract, qa, seo, privacy (or --with all)
```

## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." → STOP |
| Issue missing `**File:**` field | "Issue #N does not link to a proposal file." → STOP |
| Proposal file not found | "Proposal file not found: `{path}`." → STOP |
| Issue closed | "Issue #N is closed. Review anyway? (y/n)" → ask user |
| File write fails | "Failed to update proposal file: {error}" → STOP |
| Comment post fails | Warn, continue (file already updated) |
| No metadata section in file | Create metadata field before first `---` separator |
**End of /review-proposal Command**
