---
version: "v0.58.0"
description: Collaborative path analysis for proposals and enhancements (project)
argument-hint: "#issue"
---

<!-- EXTENSIBLE -->
# /paths
Performs turn-based collaborative scenario path discovery on proposals and enhancements. AI and user work through six scenario categories to systematically identify paths early in the lifecycle.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command paths`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number — accepts `#N` or `N` format |
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction
---
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
### Step 3: Load Proposal Content
**Step 3a:** Parse issue body for proposal file reference: `**File:** Proposal/[Name].md`
**If proposal file found:** Read the file content for analysis.
**If proposal file not found on disk:** Warn and fall back to issue body.
**If no proposal file reference in body:** Use issue body as content source.
**If issue body is empty:**
```
Issue #$ISSUE has no content to analyze. Add a description first.
```
→ **STOP**
### Step 4: Check for Existing Path Analysis
Search the proposal document (or issue body) for existing `## Path Analysis` section.
**If found:** Load as starting point for re-run, parse `###` subsections to pre-populate categories. Inform user.
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
**Step 5a:** AI generates 2-5 contextually relevant candidate scenarios from proposal content.
**Step 5b:** Present candidates via `AskUserQuestion` with `multiSelect: true`. If re-run, include existing paths as pre-populated options.
**Step 5c:** Ask: `"Any paths I missed for ${categoryName}?"` — add user contributions or proceed if none.

<!-- USER-EXTENSION-START: post-category -->
<!-- USER-EXTENSION-END: post-category -->

### Step 6: Consolidate and Confirm
Present consolidated summary with counts per category and total.
Ask user: `"Write this Path Analysis to the proposal document? (yes/no)"`
**If user declines:** `"Path Analysis not written."` → **STOP**
**If no paths confirmed:** `"No paths confirmed. Path Analysis not created."` → **STOP**
### Step 7: Write Path Analysis
**If proposal file exists:** Append or update `## Path Analysis` section.
- If existing section found: replace in place
- If no existing section: append to end (before `## Review Log` if present)
**If no proposal file:** Post as issue comment via `gh issue comment`.
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
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." → STOP |
| Issue is closed | "Analyze anyway? (y/n)" → proceed if confirmed |
| Not proposal or enhancement | "Issue #N is a {type}. /paths supports proposals and enhancements only." → STOP |
| Proposal file not found (when linked) | Fall back to issue body with warning |
| Issue body empty | "Add a description first." → STOP |
| No paths confirmed | "No paths confirmed. Path Analysis not created." → STOP |
| File write fails | "Failed to update proposal file: {error}" → STOP |
| Existing Path Analysis found | Load as starting point for re-run |
---
**End of /paths Command**
