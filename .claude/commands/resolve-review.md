---
version: "v0.86.0"
description: Resolve review findings for an issue (project)
argument-hint: "#issue"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /resolve-review
Parse the latest review findings and resolve each one. Delegates parsing/classification to `resolve-preamble.js`. Works with findings from `/review-issue`, `/review-proposal`, `/review-prd`, `/review-test-plan`.
---
## Prerequisites
- `gh pmu` installed
- `.gh-pmu.json` configured
- Issue has ≥1 review comment
---
## Arguments
| Argument | Description |
|----------|-------------|
| `#issue` | Issue number (e.g., `#42`) |
---
## Execution
**REQUIRED — routed command, two-phase task creation:**
1. **Phase 1 — Preamble task only:** `TaskCreate` single preamble/setup task. Do NOT create subsequent tasks yet.
2. **Phase 2 — Bulk after routing:** After preamble confirms path (no redirect, no early exit), bulk-create remaining workflow tasks.
3. **Redirect or early exit:** Mark preamble done, stop. Do NOT create remaining tasks.
4. **Include Extensions:** Active `USER-EXTENSION` block → Phase 2 task
5. Mark `in_progress` → `completed`
6. **Post-Compaction:** Re-read, resume from first incomplete — no re-routing.
---
## Workflow
### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/resolve-preamble.js $ISSUE
```
Parse JSON. `ok: false` → report `errors[0].message` → **STOP**.
`earlyExit: true` (recommendation "Ready for") → "Already ready — no action needed." → **STOP**.
Extract: `context` (reviewType, reviewNumber, recommendation), `findings` (autoFixable, needsUserInput, passed).
Report: `"Resolving N findings from {reviewType} Review #M..."` with auto-fixable/user-input counts.
### Step 2: Pass 1 — Auto-Fix
Iterate `findings.autoFixable`. Apply and report:
- **Priority not set:** `gh pmu move $ISSUE --priority p2`
- **Missing labels:** `gh issue edit $ISSUE --add-label {label}` — inferred
- **Body-modifying** (missing AC, repro, format): show preview and confirm — body is harder to undo
```
Auto-resolved:
  ✓ Priority set to P2 (default)
  ✓ Added label: enhancement
  ✓ Added AC section skeleton (confirmed)
```
### Step 3: Pass 2 — User Input
Iterate `findings.needsUserInput`. Use `AskUserQuestion`:
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
- **Accept:** apply, `"✓ {change applied}"`
- **Alternative:** ask conversationally, apply
- **Skip:** `"⊘ Skipped: {finding}"`

**Title rewording:** Propose new title from content, present "Accept", "Edit", "Skip".
### Step 4: Re-Review
After all resolved, invoke re-review via Skill tool with `--force`:
```
Skill("review-issue", "#$ISSUE --force")
```
`/review-issue` handles full cycle (preamble → evaluate → finalize), including `reviewed`/`pending` label swap.
Report:
```
/resolve-review #$ISSUE complete.
  Findings resolved: N
  Re-review: [recommendation from re-review]
```
If user declined all: `"No changes made. Review findings remain unresolved."` → **STOP**
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | Report error → STOP |
| No review comment | Preamble errors → STOP |
| Already ready | "Already ready — no action needed." → STOP |
| `gh pmu` fails | Report error → STOP |
| User declines all | "No changes made." → STOP |
| Re-review finds new issues | Report — user can re-run |
---
**End of /resolve-review Command**
