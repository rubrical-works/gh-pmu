---
version: "v0.62.0"
description: Collaborative path analysis for proposals and enhancements (project)
argument-hint: "#issue"
---
<!-- EXTENSIBLE -->

# /paths
Performs turn-based collaborative scenario path discovery on proposals and enhancements. The AI assistant and user work together through six scenario categories to systematically identify paths early in the lifecycle.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command paths`

## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number — accepts `#N` or `N` format (e.g., `#42` or `42`) |
| `--from-code [path]` | No | Path to source directory for code-based path discovery. When present, delegates to `code-path-discovery` skill instead of loading a proposal document. |

## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction

## Workflow
<!-- USER-EXTENSION-START: pre-paths -->
<!-- USER-EXTENSION-END: pre-paths -->

### Step 1: Fetch Issue
```bash
gh issue view $ISSUE --json number,title,body,state,labels
```
**If not found:** `"Issue #$ISSUE not found."` → **STOP**
**If closed:** `"Issue #$ISSUE is closed. Analyze anyway? (y/n)"` — proceed only if user confirms.

### Step 2: Validate Issue Type
Check issue labels for `proposal` or `enhancement`.
**If neither label present:**
```
Issue #$ISSUE is a {type}. /paths supports proposals and enhancements only.
```
→ **STOP**

### Step 3: Load Content
Check if `--from-code` flag is present. Route to Step 3a or Step 3b accordingly.

#### Step 3a: Load Proposal Content (default)
When `--from-code` is **not** present, load from proposal document.
Parse issue body for proposal file reference:
```
Pattern: **File:** Proposal/[Name].md
```
**If proposal file found:** Read the file content for analysis.
**If proposal file not found on disk:** Warn: `"Proposal file not found at {path}. Falling back to issue body."` — use issue body as fallback.
**If no proposal file reference in body:** Use issue body as content source.
**If issue body is empty (no content to analyze):**
```
Issue #$ISSUE has no content to analyze. Add a description first.
```
→ **STOP**

#### Step 3b: Load Code Paths (--from-code)
When `--from-code` is present, validate the path and delegate to the `code-path-discovery` skill.
**Step 3b-i: Validate path exists**
Check that the provided path exists on disk.
**If path not found:** `"Path not found: \`{path}\`. Check the path and try again."` → **STOP**
**Step 3b-ii: Check for source files**
Scan the path for supported files (`.ts`, `.tsx`, `.js`, `.jsx`).
**If no source files found:** `"No source files found in \`{path}\`. Supported: \`.ts\`, \`.tsx\`, \`.js\`, \`.jsx\`."` → **STOP**
**Step 3b-iii: Broad scope warning**
If the path contains more than 50 source files (e.g., repo root): warn `"Broad scope may produce noisy results. Consider scoping to a feature directory."` — proceed.
**Step 3b-iv: Invoke skill**
Invoke the `code-path-discovery` skill with parameters:
- `path` — the directory path provided with `--from-code`
- `issueTitle` — the issue title from Step 1
- `issueBody` — the issue body from Step 1 (optional context)
The skill scans source files and returns candidate scenarios as `{ shortLabel, description }` objects per category, which feed directly into Step 5's turn-based discovery loop.
**Step 3b-v: Check for zero candidates**
If the skill returns empty arrays for all 6 categories: prompt `"No behavioral paths detected in \`{path}\`. Proceed with manual discovery? (y/n)"` — if yes, continue to Step 5 with empty candidates; if no, **STOP**.

### Step 4: Check for Existing Path Analysis
Search the proposal document (or issue body) for an existing `## Path Analysis` section.
**If found:**
- Load existing paths as starting point for re-run
- Parse each `###` subsection to pre-populate categories
- Inform user: `"Existing Path Analysis found — loading as starting point. You can add, remove, or modify paths."`
**If not found:** Start with empty path sets for all categories.

### Step 5: Turn-Based Discovery
Walk through 6 categories in order:
1. **Nominal Path** — Expected successful flow (happy path)
2. **Alternative Paths** — Valid but non-primary flows
3. **Exception Paths** — Error conditions and system responses
4. **Edge Cases** — Boundary conditions with unusual but valid inputs
5. **Corner Cases** — Combinations of edge cases or rare intersections
6. **Negative Test Scenarios** — Intentionally invalid inputs/states
**For each category:**
**Step 5a: AI generates candidate scenarios**
Read the proposal/enhancement content and generate 2–5 contextually relevant candidate scenarios for the current category. Scenarios should be specific to the proposal content, not generic.
**Step 5b: User validates via AskUserQuestion**
Present candidates using `AskUserQuestion` with `multiSelect: true`:
```javascript
AskUserQuestion({
  questions: [{
    question: `${categoryName}: Select the paths that apply (deselect any that don't):`,
    header: categoryName,
    options: candidateScenarios.map(s => ({
      label: s.shortLabel,
      description: s.description
    })),
    multiSelect: true
  }]
});
```
If existing paths are loaded (re-run), include them as pre-populated options.
**Step 5c: User contributes missing paths**
After each category, ask: `"Any paths I missed for ${categoryName}?"` via conversational prompt.
- If user provides additional paths: add to confirmed list
- If user says "no" / "none" / skips: proceed to next category
<!-- USER-EXTENSION-START: post-category -->
<!-- USER-EXTENSION-END: post-category -->

### Step 6: Consolidate and Confirm
After all 6 categories are complete, present the consolidated path analysis:
```
Path Analysis Summary:
  Nominal Path: N paths
  Alternative Paths: N paths
  Exception Paths: N paths
  Edge Cases: N paths
  Corner Cases: N paths
  Negative Test Scenarios: N paths
  Total: N paths
```
Ask user: `"Write this Path Analysis to the proposal document? (yes/no)"`
**If user declines:** `"Path Analysis not written."` → **STOP**
**If no paths were confirmed across all categories:**
```
No paths confirmed. Path Analysis not created.
```
→ **STOP**

### Step 7: Write Path Analysis
**If proposal file exists:** Append or update `## Path Analysis` section in the proposal document.
- If existing `## Path Analysis` section found: replace it in place (do not create duplicate)
- If no existing section: append to end of document (before `## Review Log` if present)
**If no proposal file:** Post Path Analysis as an issue comment via `gh issue comment`.
**Path Analysis section format:**
```markdown
## Path Analysis
### Nominal Path
1. [Scenario description]
### Alternative Paths
1. [Scenario description]
### Exception Paths
1. [Scenario description]
### Edge Cases
1. [Scenario description]
### Corner Cases
1. [Scenario description]
### Negative Test Scenarios
1. [Scenario description]
---
*Path analysis performed [YYYY-MM-DD] — collaborative discovery between AI and user.*
```
**If file write fails:** `"Failed to update proposal file: {error}"` → **STOP**
<!-- USER-EXTENSION-START: post-paths -->
<!-- USER-EXTENSION-END: post-paths -->

### Step 8: Report
```
Path Analysis complete for Issue #$ISSUE: $TITLE
  Nominal Path: N paths
  Alternative Paths: N paths
  Exception Paths: N paths
  Edge Cases: N paths
  Corner Cases: N paths
  Negative Test Scenarios: N paths
  Total: N paths
  Written to: [file path or "issue comment"]
```
**STOP.** Do not proceed without user instruction.

## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." → STOP |
| Issue is closed | "Issue #N is closed. Analyze anyway? (y/n)" → proceed if confirmed |
| Issue is not proposal or enhancement | "Issue #N is a {type}. /paths supports proposals and enhancements only." → STOP |
| Proposal file not found (when linked) | Fall back to issue body with warning |
| Issue body empty (no content to analyze) | "Issue #N has no content to analyze. Add a description first." → STOP |
| No paths confirmed across all categories | "No paths confirmed. Path Analysis not created." → STOP |
| File write fails | "Failed to update proposal file: {error}" → STOP |
| Existing Path Analysis section found | Load as starting point for re-run (do not overwrite without user interaction) |
| `--from-code` path not found | "Path not found: `{path}`. Check the path and try again." → STOP |
| `--from-code` path has no source files | "No source files found in `{path}`. Supported: `.ts`, `.tsx`, `.js`, `.jsx`." → STOP |
| `--from-code` produces zero candidates | "No behavioral paths detected in `{path}`. Proceed with manual discovery? (y/n)" — if yes, continue with empty candidates; if no, STOP |
| `--from-code` path is too broad | Warn: "Broad scope may produce noisy results. Consider scoping to a feature directory." — proceed |
**End of /paths Command**
