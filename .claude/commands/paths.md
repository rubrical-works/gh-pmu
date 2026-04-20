---
version: "v0.89.0"
description: Collaborative path analysis for proposals and enhancements (project)
argument-hint: "#issue"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /paths
Turn-based collaborative scenario path discovery on proposals/enhancements. AI and user work through scenario categories to identify paths early.
**Extension Points:** `.claude/metadata/extension-points.json` or `/extensions list --command paths`
## Prerequisites
- `gh pmu` installed
- `.gh-pmu.json` in repo root
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | `#N` or `N` (e.g., `#42`) |
| `--quick` | No | First 3 categories only (Nominal, Alternative, Exception) |
| `--dry-run` | No | Non-interactive summary, no prompts/file changes |
| `--categories IDs` | No | CSV IDs for selective re-run. Valid: `nominal`, `alternative`, `exception`, `edge`, `corner`, `negative` |
| `--from-code [path]` | No | Delegate to `code-path-discovery` skill |
## Execution
**REQUIRED:** `TaskCreate` tasks from steps + one per non-empty `USER-EXTENSION` block; mark `in_progress`→`completed`; post-compaction re-read and regenerate.
## Workflow

<!-- USER-EXTENSION-START: pre-paths -->
<!-- USER-EXTENSION-END: pre-paths -->

### Step 1: Setup (Preamble)
```bash
node ./.claude/scripts/shared/paths-preamble.js $ISSUE [--quick] [--dry-run] [--categories IDs] [--from-code path]
```
Parse JSON. `ok: false` → report `errors[0].message` → **STOP**.
Extract `context`: issue, config (categories+descriptions), flags, proposalFile, partial/resumeFrom.
**`context.issue.type === 'enhancement'`:** Display `context.config.fromCodeHint`.
### Step 2: Load Content
**`--from-code`:** Validate path → Step 2b.
**Else:** Read `context.proposalFile`. Not found → fall back to issue body.
**Issue body empty:** `"Issue #$ISSUE has no content to analyze."` → **STOP**
#### Step 2a: Load Screen Specs (supplementary)
1. Check proposal/issue body for `## Screen Specs` with file refs (e.g., `Mockups/{Name}/Specs/{Screen}.md`)
2. None: scan `Mockups/*/Specs/` for specs whose names match issue title/body terms
3. **Found:** Read each, extract element data — types, validation rules (`validationMessage`, `inputRange`), `required`, `dependencies`, `conditionalRender`, `defaultValue`
4. Hold for Step 4. Element data feeds discovery:
   - **Edge:** boundary values from `inputRange`
   - **Exception:** required field violations, validation failures
   - **Corner Cases:** dependency + conditional rendering combinations
   - **Negative Tests:** invalid inputs from type/validation constraints
5. Report: `"Screen specs loaded: {N} screens, {M} elements — will inform path candidate generation."`

**None found:** Skip silently.
#### Step 2b: Code Paths (--from-code)
Validate path, scan `.ts/.tsx/.js/.jsx`, warn if >50 files. Invoke skill with `path`/`issueTitle`/`issueBody`. Zero candidates → `AskUserQuestion`: "Yes, manual discovery" / "No, stop".
### Step 3: Check Existing / Partial
Search `## Path Analysis`. Use `context.partial`.
**Partial marker:** Load completed categories, resume at `context.resumeFrom`.
**Full found:** Load as starting point for re-run.
**Not found:** Empty path sets.
### Step 4: Turn-Based Discovery (or Dry-Run)
**`--dry-run`:** Generate candidates for all active categories, display single grouped summary (show existing alongside new). No prompts/changes → skip to Step 7. **STOP.**

**For each category in `context.config.categories`:**

**4a: Breadcrumb** (N = 1-based, clamped 1..total)
```
[N/{total}] {category.name} — {X} paths confirmed so far
            {category.description}
            Remaining: {remaining category names}
```
**4b:** AI generates 2–5 candidates specific to proposal. With screen spec data (Step 2a), use element details for precision — boundary values from `inputRange`, required field violations, dependency chains, conditional rendering edge cases.
**4c: User validates via AskUserQuestion**
```javascript
AskUserQuestion({
  questions: [{
    question: `${category.name}: Select the paths that apply:`,
    header: category.name,
    options: [...candidates, { label: "Skip this category", description: "Move on without confirming paths" }],
    multiSelect: true
  }]
});
```
"Skip this category" → record 0, proceed.
Re-run with existing paths: include as pre-populated options.
**4d: User contributes / generates more.** `AskUserQuestion`: "No, continue" / "Yes, add paths" / "Generate more candidates". "Generate more": AI produces 2-3 additional avoiding duplicates. All duplicates → "No new unique candidates."
**4e: Buffer confirmed** for incremental save.

<!-- USER-EXTENSION-START: post-category -->
<!-- USER-EXTENSION-END: post-category -->

**On interruption:** Write partial with `(Partial — N/{total} categories)` marker.
### Step 5: Consolidate and Confirm
Display full list grouped:
```
Path Analysis Summary ({total} paths):
  {category.name} ({count}):
    1. {scenario description}
    2. {scenario description}
  ...
```
`AskUserQuestion`: "Write to document" / "Review again" / "Discard".
**Discard:** `"Path Analysis not written."` → **STOP**
**No paths confirmed:** `"No paths confirmed. Path Analysis not created."` → **STOP**
### Step 6: Write Path Analysis
**Proposal file exists:** Append/update `## Path Analysis` (replace if exists, append before `## Review Log` if new).
**No proposal file:** `gh issue comment`.
**Format:** `## Path Analysis` with `###` per category, numbered items, dated footer.
**`--quick`:** Footer note `(Quick pass — 3/{total} categories)`.
**File write fails:** `"Failed to update proposal file: {error}"` → **STOP**
### Step 6a: Generate ACs (Enhancement Only)
**Trigger:** `context.issue.type === 'enhancement'` AND paths written in Step 6.
```javascript
const { generateACsFromPaths } = require('.claude/scripts/shared/lib/paths-ac-generator.js');
const result = generateACsFromPaths(pathsByCategory, issueType, existingACs);
```
`pathsByCategory` from Step 5; `issueType` from preamble; `existingACs` are current `- [ ]` lines from issue body.

**`result.skipped`:** Continue silently (non-enhancement).

**ACs generated:**
1. Read body: `gh pmu view $ISSUE --body-stdout > .tmp-$ISSUE.md`
2. Append generated ACs after existing Acceptance Criteria section (use `result.markdown`)
3. Update: `gh pmu edit $ISSUE -F .tmp-$ISSUE.md && rm .tmp-$ISSUE.md`
4. Report `result.report` (e.g., "Added 12 acceptance criteria from 23 paths")

**Categorical grouping:** When 6+ paths confirmed, ACs grouped under italic category headings (e.g., *Nominal Flow*, *Error Handling*) per px-manager#782 pattern.
**Deduplication:** Paths fuzzy-matching existing ACs are skipped. Report includes duplicate count.

<!-- USER-EXTENSION-START: post-paths -->
<!-- USER-EXTENSION-END: post-paths -->

### Step 7: Report
```
Path Analysis complete for Issue #$ISSUE: $TITLE
  {category.name}: N paths (per active category)
  Total: N paths
  Written to: [file path or "issue comment"]
```
**STOP.** Do not proceed without user instruction.
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | Preamble error → STOP |
| Issue not proposal/enhancement | Preamble error → STOP |
| Issue closed | Preamble warns, ask to confirm |
| Proposal file not found | Fall back to issue body with warning |
| Issue body empty | "No content to analyze." → STOP |
| No paths confirmed | "No paths confirmed." → STOP |
| File write fails | Report error → STOP |
| `--from-code` path not found | "Path not found." → STOP |
| `--from-code` no source files | "No source files found." → STOP |
| `--from-code` zero candidates | AskUserQuestion: manual discovery or stop |
| `--from-code` broad scope | Warn, proceed |
| Flag conflict | Preamble error → STOP |
| Invalid `--categories` | Preamble error with valid list → STOP |

**End of /paths Command**
