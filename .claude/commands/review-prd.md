---
version: "v0.61.0"
description: Review a PRD with tracked history (project)
argument-hint: "#issue [--force]"
---
<!-- EXTENSIBLE -->

# /review-prd
Reviews a PRD document linked from a GitHub issue, tracking review history with metadata fields and a Review Log table. Evaluates requirements completeness, user stories, acceptance criteria, NFRs, and test plan alignment.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command review-prd`

## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue body must reference the PRD file path

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to the PRD (e.g., `#42` or `42`) |
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

### Step 1: Resolve Issue and PRD File
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
Extract the PRD file path from the issue body. Look for patterns:
- `**File:** PRD/[Name]/PRD-[Name].md`
- `**PRD:** PRD/[Name]/PRD-[Name].md`
- Direct path reference to a `PRD/` file
**If no PRD file reference found:**
```
Issue #$ISSUE does not link to a PRD file.
Expected a PRD file path (e.g., `PRD/[Name]/PRD-[Name].md`) in the issue body.
```
→ **STOP**
Read the PRD file at the extracted path.
**If file not found:**
```
PRD file not found: `{path}`. Check the path in issue #$ISSUE?
```
→ **STOP**

### Step 2: Locate Test Plan
Check for a test plan file alongside the PRD:
```
PRD/{Name}/Test-Plan-{Name}.md
```
Look for files matching `Test-Plan-*.md` in the same directory as the PRD file.
**If test plan exists:** Read it for inclusion in the review.
**If no test plan found:**
```
Warning: No test plan found alongside PRD. Review will proceed without test plan alignment check.
```
Continue with PRD-only review (non-blocking).
**If test plan exists but is empty:** Note incomplete test plan in review findings.
<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 2b: Extension Loading
**If `--with` is specified:**
1. Read `.claude/metadata/review-extensions.json`
2. Parse the `--with` value:
   - `--with all` → load all registered extensions
   - `--with security,performance` → load specified extensions (trim spaces from comma-separated list)
3. For each requested extension ID:
   - Look up the `source` path in the registry
   - Resolve the path relative to `frameworkPath` (from `framework-config.json`)
   - Read the criteria file content
   - Extract the **PRD Review Questions** section
4. If an extension ID is not found in the registry, warn: `"Unknown extension: {id}. Available: security, accessibility, performance, chaos, contract, qa, seo, privacy"`
5. Store loaded criteria for use in Step 3
**Error Handling (Extension Loading):**
- **Registry not found:** `"Review extensions registry not found. Run hub update or check installation."` → fall back to standard review only
- **Registry malformed:** `"Review extensions registry is malformed. Run hub update or check installation."` → fall back to standard review only
- **Criteria file not found:** `"Warning: Review criteria file not found for '{domain}'. Skipping domain. Update hub to resolve."` → continue with remaining domains
- **All criteria files missing:** `"No review criteria files found. Running standard review only."` → fall back to standard review only
**If `--with` is not specified:** Skip extension loading (standard review only).

### Step 3: Perform Review
Evaluate the PRD using a two-phase approach: **auto-evaluate objective criteria** by reading the PRD and test plan files, then **ask the user only about subjective criteria** via `AskUserQuestion`.
**Step 3a: Load reviewMode**
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
**Step 3b: Auto-Evaluate Objective Criteria**
Read the PRD file (and test plan if present) and auto-evaluate structural/factual criteria. Do NOT ask the user about these.
| Criterion | Auto-Check Method |
|-----------|-------------------|
| Required sections present | Check for: Summary, Problem Statement, Proposed Solution, User Stories, Acceptance Criteria, Out of Scope, NFRs |
| Scope boundaries defined | Check for explicit in-scope / out-of-scope sections with specific boundary statements |
| Success criteria measurable | Check for quantitative language, verifiable outcomes, or measurable thresholds in requirements |
| Story format compliance | Verify each user story follows "As a ... I want ... So that ..." pattern (or close variant) |
| All stories have ACs | Check each story/epic has `- [ ]` checkbox items |
| Story numbering consistent | Verify epic/story numbering is sequential and cross-referenced correctly |
| Story priorities assigned | Check if stories have priority indicators (P0/P1/P2) or priority ordering |
| Stories sized appropriately | Check AC count and scope complexity per story — flag stories with >8 ACs or overly broad scope |
| AC cover happy paths and error cases | For each story, verify ACs include both success scenarios and error/failure handling |
| Edge cases identified | Check for edge case mentions in AC or body (boundary conditions, empty inputs, concurrent access, etc.) |
| Cross-references valid | Verify file paths, issue numbers, and proposal references exist |
| Performance requirements specified | Check NFR section for performance targets (response time, throughput, resource limits) |
| Security considerations documented | Check NFR section for security requirements (authentication, authorization, data handling, input validation) |
| Scalability or availability requirements | Check NFR section for scalability limits, availability targets, or capacity planning |
| Test plan presence | Check if test plan file exists alongside PRD (from Step 2) |
| AC coverage (if test plan) | Cross-reference each `- [ ]` AC against test plan test cases — report coverage % |
| Integration test scenarios (if test plan) | Check test plan for integration test cases covering component interactions |
| E2E test scenarios (if test plan) | Check test plan for end-to-end scenarios covering critical user journeys from the PRD |
| Test coverage approach documented | Check test plan for coverage targets and testing strategy section |
| Test coverage proportionate | For each user story, verify that ACs include test-related requirements proportionate to the story's scope. Stories that add or modify code paths should have ACs requiring unit tests, integration tests, or E2E tests as appropriate. Flag stories with complex scope (multiple files, refactoring, new modules) whose ACs lack any test requirement. |
| Screen spec alignment (UI PRDs) | If the PRD references screen specs (via `## Screen Specs` in the source proposal or direct `Screen-Specs/` references), verify that PRD acceptance criteria for UI features align with the referenced screen specs. Flag any PRD requirement that introduces UI elements not present in the proposal's screen specs. If no screen spec references exist, skip this check (non-blocking). |
**Present auto-evaluation results:**
```
Auto-evaluated (objective criteria):
  ✅ Required sections present — all 7 sections found
  ✅ Story format compliance — 10/10 stories follow "As a..." format
  ✅ All stories have ACs — 38 total ACs across 10 stories
  ⚠️ Story numbering — Epic 2, Story 2.4 referenced but not defined
  ✅ Cross-references valid — proposal #1187, 3 file paths verified
  ✅ Test plan present — Test-Plan-DrawIO-Generation-Skill.md found
  ⚠️ AC coverage — 35/38 ACs mapped to test cases (92% coverage)
```
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
**Note:** NFR adequacy is now auto-evaluated (performance, security, scalability checked individually). The subjective question shifted from "are NFRs adequate?" to "is the decomposition right?" — a judgment that genuinely requires human context about the project.
**Conditional follow-up:** If user selects ⚠️ or ❌ for any subjective criterion, ask conversationally for specifics.
**Step 3d: Extension Criteria** (if `--with` specified)
For each loaded extension domain, evaluate the PRD against the extension's PRD Review Questions. Auto-evaluate objective extension criteria; ask the user about subjective ones.
**Step 3e: Collect All Findings**
Merge auto-evaluated and user-evaluated findings into structured categories: **Strengths**, **Concerns**, **Recommendations**.
Determine a recommendation:
- **Ready for backlog creation** — No blocking concerns
- **Ready with minor revisions** — Small issues that don't block
- **Needs revision** — Significant concerns that should be addressed first
- **Needs major rework** — Fundamental issues with requirements or scope
**If extensions were loaded (Step 2b):**
For each loaded extension, present findings as a separate section:
```markdown
### Security Review (IDPF-Security)
- [Finding 1]
- [Finding 2]
```
Extension findings can **escalate** the overall recommendation but cannot downgrade it.
**Applicability Filtering:** Omit extension domain sections that produce no applicable findings. Only domains with actual findings appear in the output and in the `**Extensions Applied:**` header. If no domains produce findings, fall back to standard-only review with a warning: `"No domain extensions produced findings. Showing standard review only."` At least one domain section must appear when `--with` is used; otherwise the fallback applies.

### Step 4: Update PRD Metadata
Read the current PRD file content.
**Update `**Reviews:** N` field:**
- If `**Reviews:**` field exists: increment the number (e.g., `**Reviews:** 1` → `**Reviews:** 2`)
- If `**Reviews:**` field does not exist: add `**Reviews:** 1` after the existing metadata fields, before the first `---` separator

### Step 5: Update Review Log
**If `## Review Log` section exists:** Append a new row to the existing table.
**If `## Review Log` section does not exist:**
- If `**End of PRD**` marker exists: insert the Review Log section before it
- If no `**End of PRD**` marker (DD14 fallback): append the Review Log section at the very end of the file
**Review Log format:**
```markdown
---

## Review Log

| # | Date | Reviewer | Findings Summary |
|---|------|----------|------------------|
| 1 | YYYY-MM-DD | Claude | [Brief one-line summary of findings] |
```
Each review appends a new row. **Never edit or delete existing rows** — the log is append-only.
Write the updated PRD file.
**If file write fails:** `"Failed to update PRD file: {error}"` → **STOP**

### Step 6: Post Issue Comment
Post a structured review comment to the GitHub issue:
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
**Backwards compatibility:** The `### Findings` section header and emoji markers (✅ ⚠️ ❌) remain unchanged for `/resolve-review` parser compatibility. The `#### Auto-Evaluated` and `#### User-Evaluated` subsections are additive.
```bash
gh issue comment $ISSUE -F .tmp-review-comment.md
rm .tmp-review-comment.md
```
**If comment post fails:** Warn and continue (non-blocking — the PRD file is already updated).

### Step 6.5: Assign Review Outcome Label (Conditional)
If recommendation starts with "Ready for" (i.e., "Ready for backlog creation"):
```bash
gh issue edit $ISSUE --add-label=reviewed --remove-label=pending
```
If recommendation does NOT start with "Ready for" (i.e., "Needs minor revision", "Needs revision", or "Needs major rework"):
```bash
gh issue edit $ISSUE --add-label=pending --remove-label=reviewed
```

### Step 6.6: AC Check-Off (Conditional)
**Only when recommendation is "Ready for backlog creation":**
This step automatically checks off acceptance criteria on the PRD issue that passed review, reducing the manual step between review and `/create-backlog`.
1. **Export the PRD issue body:**
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
4. Report:
   ```
   AC check-off: X/Y criteria checked off on issue #$ISSUE.
   ```
**No status transition** — `/create-backlog` owns the `in_progress` transition.
**If recommendation is NOT "Ready for backlog creation":** Skip this step entirely — no AC check-off.
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
**If `--with` is not specified**, append a discoverability tip after the summary:
```
Tip: Use --with security,performance to add domain-specific review criteria.
Available: security, accessibility, performance, chaos, contract, qa, seo, privacy (or --with all)
```

## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." → STOP |
| Issue missing PRD file reference | "Issue #N does not link to a PRD file." → STOP |
| PRD file not found | "PRD file not found: `{path}`." → STOP |
| Issue closed | "Issue #N is closed. Review anyway? (y/n)" → ask user |
| Test plan not found | Warning, continue with PRD-only review |
| Test plan empty | Note in findings, continue |
| File write fails | "Failed to update PRD file: {error}" → STOP |
| Comment post fails | Warn, continue (file already updated) |
| No metadata section in file | Create metadata field before first `---` separator |
**End of /review-prd Command**
