---
version: "v0.81.1"
description: Resolve review findings for an issue (project)
argument-hint: "#issue"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /resolve-review
Parse latest review findings on an issue and systematically resolve each one. Delegates comment parsing and classification to `resolve-preamble.js`. Works with `/review-issue`, `/review-proposal`, `/review-prd`, `/review-test-plan`.
**Prerequisites:** `gh pmu` installed, `.gh-pmu.json` configured, issue has review comment(s).
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number (e.g., `#42` or `42`) |
## Execution Instructions
**REQUIRED:** This is a routed command -- use two-phase task creation:
1. **Phase 1 -- Preamble task only:** Create a single task for the preamble/setup step using `TaskCreate`. Do NOT create tasks for subsequent workflow steps yet.
2. **Phase 2 -- Bulk create after routing:** After the preamble confirms the workflow path (no redirect, no early exit), bulk-create tasks for all remaining workflow steps using `TaskCreate`.
3. **On redirect or early exit:** Mark the preamble task as completed and stop. Do NOT create tasks for the original command's remaining steps.
4. **Track Progress:** Mark tasks `in_progress` -> `completed` as you work.
5. **Post-Compaction:** Re-read this spec. Resume from the first incomplete task -- no re-routing needed.
## Workflow
### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/resolve-preamble.js $ISSUE
```
If `ok: false`: report `errors[0].message` -> **STOP**.
If `earlyExit: true`: "Already ready -- no action needed." -> **STOP**.
Extract: `context` (reviewType, reviewNumber, recommendation), `findings` (autoFixable, needsUserInput, passed).
Report: `"Resolving N findings from {reviewType} Review #M..."` with counts.
### Step 2: Resolve -- Pass 1 (Auto-Fix)
Iterate `findings.autoFixable`. For each:
- **Priority not set:** `gh pmu move $ISSUE --priority p2` -- apply, report
- **Missing labels:** `gh issue edit $ISSUE --add-label {label}` -- add, report
- **Body-modifying fixes** (missing AC, repro steps, format): preview and confirm before applying
```
Auto-resolved:
  check: Priority set to P2 (default)
  check: Added label: enhancement
  check: Added AC section skeleton (confirmed)
```
### Step 3: Resolve -- Pass 2 (User Input)
Iterate `findings.needsUserInput`. For each, use `AskUserQuestion`:
```javascript
AskUserQuestion({
  questions: [{
    question: `Finding: ${finding.criterion}\nDetail: ${finding.detail}`,
    header: "Resolution",
    options: [
      { label: "Accept suggestion", description: "Apply suggested change" },
      { label: "Provide alternative", description: "Specify your own resolution" },
      { label: "Skip", description: "Leave unresolved" }
    ],
    multiSelect: false
  }]
});
```
- **Accept** -> apply, report
- **Alternative** -> ask conversationally, then apply
- **Skip** -> report skipped
**Title rewording:** Propose new title, present via AskUserQuestion with "Accept", "Edit", "Skip".
### Step 4: Re-Review
After resolving, invoke re-review with `--force`:
```
Skill("review-issue", "#$ISSUE --force")
```
`/review-issue` handles full re-review cycle (preamble -> evaluate -> finalize), including label management.
```
/resolve-review #$ISSUE complete.
  Findings resolved: N
  Re-review: [recommendation from re-review]
```
If user declined all fixes: `"No changes made. Review findings remain unresolved."` -> **STOP**
## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | Report error -> STOP |
| No review comment | Preamble returns error -> STOP |
| Already ready | "Already ready -- no action needed." -> STOP |
| `gh pmu` fails | Report error -> STOP |
| User declines all | "No changes made." -> STOP |
| Re-review finds new issues | Report -- user can re-run |
**End of /resolve-review Command**
