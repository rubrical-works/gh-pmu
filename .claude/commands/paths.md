---
version: "v0.84.0"
description: Collaborative path analysis for proposals and enhancements (project)
argument-hint: "#issue"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /paths
Turn-based collaborative scenario path discovery on proposals and enhancements.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command paths`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number (`#N` or `N`) |
| `--quick` | No | First 3 categories only |
| `--dry-run` | No | Non-interactive summary |
| `--categories IDs` | No | Comma-separated: `nominal`, `alternative`, `exception`, `edge`, `corner`, `negative` |
| `--from-code [path]` | No | Delegate to `code-path-discovery` skill |
---
## Execution Instructions
**REQUIRED:** Generate TodoWrite todos. Include extensions. Track progress. Post-compaction: re-read spec.
---
## Workflow

<!-- USER-EXTENSION-START: pre-paths -->
<!-- USER-EXTENSION-END: pre-paths -->

### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/paths-preamble.js $ISSUE [--quick] [--dry-run] [--categories IDs] [--from-code path]
```
If `ok: false`: report error -> **STOP**.
Extract `context`: issue data, config (categories), flags, proposalFile, partial/resumeFrom.
If enhancement: display `context.config.fromCodeHint`.
### Step 2: Load Content
**`--from-code`:** Validate path, delegate to skill (Step 2b).
**Otherwise:** Read `context.proposalFile`. Fall back to issue body. Empty: STOP.
#### Step 2a: Load Screen Specs (Supplementary Context)
After loading primary content, check for associated screen specs:
1. Check proposal/issue body for `## Screen Specs` section with file references
2. If none: scan `Mockups/*/Specs/` for specs matching issue title/body terms
3. If found: read each spec, extract element data (types, validation, required, dependencies, conditionalRender, defaults)
4. Use as supplementary context for Step 4:
   - **Edge Cases:** boundary values from `inputRange`
   - **Exception Paths:** required field violations, validation failures
   - **Corner Cases:** element dependencies, conditional rendering combos
   - **Negative Test Scenarios:** invalid inputs from type/validation constraints
5. Report: `"Screen specs loaded: {N} screens, {M} elements -- will inform path candidate generation."`
No specs found: skip silently.
#### Step 2b: Code Paths
Validate path, scan source files, warn >50. Invoke skill. Zero candidates: AskUserQuestion manual/stop.
### Step 3: Check Existing / Partial Analysis
Search for `## Path Analysis`. Use `context.partial`.
Partial: resume. Full: load as starting point. Not found: empty.
### Step 4: Turn-Based Discovery (or Dry-Run)
**Dry-run:** Generate all candidates, display grouped summary -> Step 7 -> STOP.
**Per category:**
**4a:** Progress breadcrumb. **4b:** AI generates 2-5 candidates (use screen spec data when available for precise scenarios). **4c:** User validates (AskUserQuestion, multiSelect, "Skip" option). **4d:** User adds or generates more. **4e:** Buffer confirmed.

<!-- USER-EXTENSION-START: post-category -->
<!-- USER-EXTENSION-END: post-category -->

**Interruption:** Write partial with marker.
### Step 5: Consolidate and Confirm
Display grouped list. AskUserQuestion: write/review/discard. Discard or no paths: STOP.
### Step 6: Write Path Analysis
File exists: append/update `## Path Analysis`. No file: issue comment. Quick: note in footer. Write failure: STOP.
#### Step 6a: Generate Acceptance Criteria (Enhancement Only)
**Trigger:** `context.issue.type === 'enhancement'` AND paths were written in Step 6.
```javascript
const { generateACsFromPaths } = require('.claude/scripts/shared/lib/paths-ac-generator.js');
const result = generateACsFromPaths(pathsByCategory, issueType, existingACs);
```
- If `result.skipped`: continue silently.
- If ACs generated:
  1. Read issue body: `gh pmu view $ISSUE --body-stdout > .tmp-$ISSUE.md`
  2. Append generated ACs after existing Acceptance Criteria section (`result.markdown`)
  3. Update: `gh pmu edit $ISSUE -F .tmp-$ISSUE.md && rm .tmp-$ISSUE.md`
  4. Report: `result.report`
- **Categorical grouping:** 6+ paths -> ACs grouped under italic category headings
- **Deduplication:** Fuzzy-match existing ACs skipped. Report includes duplicate count.

<!-- USER-EXTENSION-START: post-paths -->
<!-- USER-EXTENSION-END: post-paths -->

### Step 7: Report
```
Path Analysis complete for Issue #$ISSUE: $TITLE
  {category}: N paths
  Total: N paths
  Written to: [path or "issue comment"]
```
**STOP.**
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | STOP |
| Not proposal/enhancement | STOP |
| Issue closed | Warn, ask |
| Proposal not found | Fall back to body |
| Body empty | STOP |
| No paths | STOP |
| File write fails | STOP |
| `--from-code` path/files issues | STOP |
| Flag conflict / invalid categories | STOP |
---
**End of /paths Command**
