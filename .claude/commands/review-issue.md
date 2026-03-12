---
version: "v0.61.0"
description: Review issues with type-specific criteria (project)
argument-hint: "#issue [#issue...] [--force]"
---
<!-- EXTENSIBLE -->

# /review-issue
Reviews one or more GitHub issues with type-specific criteria based on labels (bug, enhancement, story, epic). Tracks review count in issue body and posts structured findings as comment. No linked file — operates directly on the issue.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command review-issue`

## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | One or more issue numbers (e.g., `#42` or `42 43 44`) |
| `--with` | No | Comma-separated domain extensions (e.g., `--with security,performance`) or `--with all` |
| `--mode` | No | Transient review mode override: `solo`, `team`, or `enterprise`. Does not modify `framework-config.json`. |
| `--force` | No | Force re-review even if issue has `reviewed` label |
Accepts multiple issue numbers: `/review-issue #42 #43 #44` — reviews each sequentially.

## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction

## Workflow
**For multiple issues:** Process each issue sequentially through Steps 1–6.

### Step 1: Fetch Issue
```bash
gh issue view $ISSUE --json number,title,body,state,labels
```
**If not found:** `"Issue #$ISSUE not found."` → **STOP** (skip to next if batch)
**If closed:** `"Issue #$ISSUE is closed. Review anyway? (y/n)"` — proceed only if user confirms.
**Early-exit gate:** If the issue has the `reviewed` label and `--force` is NOT present, skip the full review:
```
"Issue #$ISSUE already reviewed (Review #N). Use --force to re-review."
```
Extract the review count from the `**Reviews:** N` field in the issue body (if present). → **STOP**
**If `--force` is present:** Bypass the early-exit gate and proceed with full review.

### Step 2: Determine Issue Type
Use the shared `getIssueType()` utility from `.claude/scripts/shared/lib/issue-type.js` to determine the issue type from the labels already fetched in Step 1 (no additional API call needed):
```javascript
const { getIssueType } = require('./.claude/scripts/shared/lib/issue-type.js');
const { type, redirect } = getIssueType(issueData); // issueData from Step 1
```
| Label Found | Review Type |
|-------------|-------------|
| `bug` | Bug review criteria |
| `enhancement` | Enhancement review criteria |
| `story` | Story review criteria |
| `epic` | Epic review criteria |
| `test-plan` | **Redirect** → `/review-test-plan #$ISSUE` |
| `proposal` | **Redirect** → `/review-proposal #$ISSUE` |
| `prd` | **Redirect** → `/review-prd #$ISSUE` |
| None recognized | Generic review criteria |
**If redirect label detected** (`test-plan`, `proposal`, `prd`):
```
Issue #$ISSUE has label '$LABEL' — redirecting to the specialized review command.
```
Invoke the appropriate command using the Skill tool. → **STOP** (the specialized command handles the review)
**If multiple recognized labels:** Use first recognized label as primary type. Redirect labels take precedence over review labels.
**If no recognized labels:**
```
Issue #$ISSUE has no recognized label (bug, enhancement, story, epic). Using generic review criteria.
```
Continue with generic criteria (non-blocking).
<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 2b: Extension Loading
**If `--with` is specified:**
1. Read `.claude/metadata/review-extensions.json`
2. Parse `--with`: `all` loads all extensions, comma-separated loads specific ones
3. For each ID: resolve `source` path relative to `frameworkPath` (from `framework-config.json`), read criteria file, extract **Issue Review Questions**
4. Unknown IDs produce warning with available list
**Error handling:** See `.claude/scripts/shared/lib/load-review-extensions.js` for error messages and fallback behavior. All errors fall back to standard review only (non-blocking):
- Unknown extension ID: warn with available list, skipping unknown
- Criteria file not found for domain: skip domain, warn, continue with remaining
- Registry not found or malformed: standard review only
**If `--with` is not specified:** Skip extension loading.

### Step 3: Perform Review
Evaluate the issue using a two-phase approach: **auto-evaluate objective criteria** by reading the issue, then **ask the user only about subjective criteria** via `AskUserQuestion`. Filter criteria by `reviewMode` from `framework-config.json` (or `--mode` override).
**Step 3a: Load reviewMode Configuration**
Parse `--mode` from arguments if provided. Invalid values produce a clear error with valid options.
```javascript
const { getReviewMode, shouldEvaluate, isObjective } = require('./.claude/scripts/shared/lib/review-mode.js');
// modeOverride is the --mode argument value (null if not provided)
const mode = getReviewMode(process.cwd(), modeOverride); // 'solo', 'team', or 'enterprise'
```
**Hint:** Display the active mode and how to override it:
- Without `--mode`: `Reviewing in {mode} mode (override with --mode solo|team|enterprise)`
- With `--mode`: `Reviewing in {mode} mode (--mode override)`
<!-- USER-EXTENSION-START: criteria-customize -->
<!-- USER-EXTENSION-END: criteria-customize -->
**Step 3b: Auto-Evaluate Objective Criteria**
For each **objective** criterion applicable to the current reviewMode, evaluate autonomously by reading the issue content. Do NOT ask the user about these — the assistant performs the analysis.
**Common Objective Criteria:** Evaluate each criterion from `.claude/metadata/review-mode-criteria.json` where `type: "objective"` and `shouldEvaluate()` returns true for the current reviewMode. Use the `autoCheck` field for evaluation guidance.
**Type-Specific Objective Checks:** Re-read `.claude/metadata/review-criteria.json` from disk (not memory) for the detected issue type. Each entry has `name` and `autoCheck` fields describing what to check. For epic type, the `sub-issue-review` criterion requires recursive review of each sub-issue through Steps 3b-3c, including 3b-ii (auto-generate proposed solution/fix) with per-sub-issue body updates.
**Present auto-evaluation results immediately:**
```
Auto-evaluated (objective criteria):
  ✅ Title clear and descriptive — "[actual title text]"
  ✅ Labels correct — enhancement
  ❌ Acceptance criteria missing — no `- [ ]` items found in body
  ⚠️ Priority not set
  ✅ Bug: Repro steps present — numbered list found
  ❌ Bug: Expected/actual behavior missing — no "Expected"/"Actual" sections
```
Emit ✅ for pass, ⚠️ for partial/uncertain, ❌ for missing/fail. Include brief evidence for each finding.
**Screen Catalog Validation (UI issues):** If the issue's acceptance criteria reference UI elements (screens, forms, inputs, buttons, validation), check for `Screen-Specs/` directory at project root. If it exists, cross-reference AC element mentions against screen spec element tables — flag missing validation rules or unspecified defaults. If `Screen-Specs/` does not exist, report gracefully: "No screen catalog found (`Screen-Specs/` missing) — consider running `/catalog-screens` to establish element specifications." Skip this check entirely for non-UI issues (CLI tools, API endpoints, backend logic).
**Step 3b-ii: Auto-Generate Proposed Solution/Fix (Enhancement, Bug, and Story)**
**Trigger:** (Enhancement or Story type AND `proposed-solution` check is ❌/⚠️) OR (Bug type AND `proposed-fix-described` check is ❌/⚠️). Does NOT apply to epic types.
**Placeholder detection:** Treat as missing if section body is under 20 characters or matches: "To be documented", "TBD", "...", or empty section.
**When triggered:**
1. Read the issue Description, Motivation (enhancement/story) or Repro/Expected/Actual (bug) sections
2. Use Glob/Grep/Read tools to identify relevant codebase files based on keywords from the issue
3. Generate a structured section:
   - **Approach:** 1-2 sentence summary of the proposed solution or fix
   - **Files to modify:** table with file path, change description, rationale
   - **Implementation steps:** numbered list of concrete steps
   - **Testing considerations:** regression tests (bugs) or feature tests (enhancements/stories)
4. Present in review output under findings:
   - Enhancement/Story: `#### Proposed Solution (Auto-Generated)`
   - Bug: `#### Proposed Fix (Auto-Generated)`
5. Append to issue body automatically during Step 4 (body metadata update) — add or replace the Proposed Solution/Fix section alongside the `**Reviews:** N` update. The review comment captures generated content as audit trail.
**When NOT triggered:** Issue already has a substantive solution/fix section (>20 chars, no placeholder patterns). Continue normally — do not overwrite existing content.
**Step 3b-iii: Construction Context Discovery (Epic Only)**
**Trigger:** Issue type is `epic`. Skipped for all other types.
After evaluating sub-issues, scan Construction artifact directories for files linked to this epic's sub-issues:
1. Collect sub-issue numbers from `gh pmu sub list $ISSUE`
2. Grep `Construction/Design-Decisions/` and `Construction/Tech-Debt/` for `Issue #N` patterns matching any sub-issue number
3. For each match, extract: file path, title (first `# ` heading), date (from `**Date:**` field), linked issue number
**Output as `### Construction Context` section in the review comment:**
```
### Construction Context
Design Decisions: N found | Tech Debt: M found

- 📄 `Construction/Design-Decisions/2026-02-08-topic.md` — "Title" (Issue #42, 2026-02-08)
- 📄 `Construction/Tech-Debt/2026-02-10-topic.md` — "Title" (Issue #55, 2026-02-10)
```
**No-match path:** If no Construction files reference any sub-issue numbers, report:
```
### Construction Context
No Construction context found for this epic's sub-issues.
```
**Step 3c: Ask Subjective Criteria**
For **subjective** criteria applicable to the current reviewMode, use `AskUserQuestion`. Re-read `.claude/metadata/review-mode-criteria.json` from disk (not memory) where `type: "subjective"` — each entry has `question`, `header`, and `options` fields for constructing the question.
**Solo mode:** No subjective criteria — skip the question step entirely.
**Team/enterprise mode — description preview:** When `description-sufficient` is included, emit a 3-5 line description summary **before** `AskUserQuestion` so the reviewer has immediate context.
If no subjective criteria apply (all filtered by reviewMode), skip the question step entirely.
**Step 3d: Extension Criteria** (if `--with` specified)
For each loaded extension domain, evaluate the issue against the extension's Issue Review Questions. Auto-evaluate any objective extension criteria; ask the user about subjective ones.
**Step 3e: Collect All Findings**
Merge auto-evaluated and user-evaluated findings using check marks:
- `✅` — Criterion met
- `⚠️` — Needs attention
- `❌` — Missing or incorrect
Determine a recommendation:
- **Ready for work** — No blocking concerns
- **Needs minor revision** — Small issues, can start with awareness
- **Needs revision** — Should be addressed before starting work
- **Needs major rework** — Fundamental issues with issue definition
**If extensions were loaded (Step 2b):**
For each loaded extension, present findings as a separate section:
```markdown
### Security Review (IDPF-Security)
- [Finding 1]
- [Finding 2]
```
Extension findings can **escalate** the overall recommendation but cannot downgrade it.
**Applicability Filtering:** Omit domains with no findings from output and `**Extensions Applied:**` header. At least one domain section must appear when `--with` is used; if no domains produce findings, fall back to standard review with warning.

### Step 4: Update Issue Body Metadata
Export the current issue body:
```bash
gh pmu view $ISSUE --body-stdout > .tmp-$ISSUE.md
```
**Update `**Reviews:** N` field:**
- If `**Reviews:**` field exists: increment the number
- If `**Reviews:**` field does not exist: add `**Reviews:** 1` at the end of the issue body (after acceptance criteria if present)
Write the updated body:
```bash
gh pmu edit $ISSUE -F .tmp-$ISSUE.md && rm .tmp-$ISSUE.md
```
**If edit fails:** `"Failed to update issue body: {error}"` → **STOP**

### Step 5: Post Issue Comment
Post a structured review comment:
```markdown
## Issue Review #N — YYYY-MM-DD

**Issue:** #$ISSUE — $TITLE
**Type:** [Bug | Enhancement | Story | Epic | Generic]
**Total Reviews:** N
**Extensions Applied:** {list of applied extensions, or "None"}

### Findings

#### Auto-Evaluated
- ✅ [Criterion] — [evidence]
- ❌ [Criterion] — [what's missing]

#### User-Evaluated
- ✅ [Criterion] — [user assessment]
- ⚠️ [Criterion] — [user concern]

### Recommendation
**[Ready for work | Needs minor revision | Needs revision | Needs major rework]** — [Brief explanation]
```
**Backwards compatibility:** The `### Findings` section header and emoji markers (✅ ⚠️ ❌) remain unchanged for `/resolve-review` parser compatibility. The `#### Auto-Evaluated` and `#### User-Evaluated` subsections are additive.
```bash
gh issue comment $ISSUE -F .tmp-review-comment.md
rm .tmp-review-comment.md
```
**If comment post fails:** Warn and continue (non-blocking — body metadata already updated).

### Step 5.5: Assign Review Outcome Label (Conditional)
If recommendation starts with "Ready for" (i.e., "Ready for work"):
```bash
gh issue edit $ISSUE --add-label=reviewed --remove-label=pending
```
**Epic sub-issue label propagation:** If the issue type is `epic`, also apply the `reviewed` label to all sub-issues:
1. Retrieve sub-issues: `gh pmu sub list $ISSUE`
2. For each sub-issue, apply: `gh issue edit $SUB_ISSUE --add-label=reviewed --remove-label=pending`
3. Track the count of sub-issues labeled for Step 6 reporting
If the issue is not an epic, only the issue itself is labeled (no sub-issue handling).
If recommendation does NOT start with "Ready for" (i.e., "Needs minor revision", "Needs revision", or "Needs major rework"):
```bash
gh issue edit $ISSUE --add-label=pending --remove-label=reviewed
```
<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

### Step 6: Report Summary
```
Review #N complete for Issue #$ISSUE: $TITLE
  Type: [Bug | Enhancement | Story | Epic | Generic]
  Recommendation: [recommendation]
  Reviews: N (updated)
  Issue comment: [posted | failed]
```
**For epic issues with "Ready for" recommendation:** Include sub-issue reviewed label count:
```
  Sub-issues labeled 'reviewed': M
```
**For multiple issues:** Report summary for each, then batch summary at end.
**If `--with` is not specified**, append a discoverability tip after the summary:
```
Tip: Use --with security,performance to add domain-specific review criteria.
Available: security, accessibility, performance, chaos, contract, qa, seo, privacy (or --with all)
```

## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." → STOP (skip to next if batch) |
| Issue closed | "Issue #N is closed. Review anyway? (y/n)" → ask user |
| Unknown issue type | "Using generic review criteria." → warn, continue |
| Issue body is empty | Flag as critical finding in review |
| `gh pmu edit` fails | "Failed to update issue body: {error}" → STOP |
| Comment post fails | Warn, continue (body already updated) |
| Multiple recognized labels | Use first recognized label as primary type |
**End of /review-issue Command**
