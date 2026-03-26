---
version: "v0.72.0"
description: Resolve review findings for an issue (project)
argument-hint: "#issue"
copyright: "Rubrical Works (c) 2026"
---

<!-- MANAGED -->
# /resolve-review
Parse the latest review findings on an issue and systematically resolve each one. Delegates comment parsing and finding classification to `resolve-preamble.js`, keeping this spec focused on resolution judgment. Works with findings from `/review-issue`, `/review-proposal`, `/review-prd`, and `/review-test-plan`.
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue has at least one review comment
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number (e.g., `#42` or `42`) |
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Create Todo List:** Use `TodoWrite` to create todos from the steps below
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
3. **Resume Point:** If interrupted, todos show where to continue
---
## Workflow
### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/resolve-preamble.js $ISSUE
```
Parse JSON output. If `ok: false`: report `errors[0].message` → **STOP**.
If `earlyExit: true` (recommendation starts with "Ready for"): report "Already ready — no action needed." → **STOP**.
Extract: `context` (reviewType, reviewNumber, recommendation), `findings` (autoFixable, needsUserInput, passed).
Report summary: `"Resolving N findings from {reviewType} Review #M..."` showing auto-fixable and user-input counts.

### Step 2: Resolve — Pass 1 (Auto-Fix)
Iterate `findings.autoFixable`. For each finding, apply the fix immediately and report:
- **Priority not set:** `gh pmu move $ISSUE --priority p2` — apply default, report
- **Missing labels:** `gh issue edit $ISSUE --add-label {label}` — add inferred label, report
- **Body-modifying fixes** (missing AC section, missing repro steps, format issues): show preview and confirm before applying — body changes are harder to undo
```
Auto-resolved:
  ✓ Priority set to P2 (default)
  ✓ Added label: enhancement
  ✓ Added AC section skeleton (confirmed)
```

### Step 3: Resolve — Pass 2 (User Input)
Iterate `findings.needsUserInput`. For each finding, use `AskUserQuestion`:
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
- **Accept suggestion** → apply, report `"✓ {change applied}"`
- **Provide alternative** → ask conversationally, then apply
- **Skip** → report `"⊘ Skipped: {finding}"`

**For title rewording:** Propose new title based on issue content, present via AskUserQuestion with "Accept", "Edit", "Skip" options.

### Step 4: Re-Review
After all findings resolved, invoke re-review using the Skill tool with `--force`:
```
Skill("review-issue", "#$ISSUE --force")
```
The thin orchestrator `/review-issue` handles the full re-review cycle (preamble → evaluate → finalize), including label management (`reviewed`/`pending` swap).
Report final status:
```
/resolve-review #$ISSUE complete.
  Findings resolved: N
  Re-review: [recommendation from re-review]
```
If user declined all fixes: `"No changes made. Review findings remain unresolved."` → **STOP**
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | Report error → STOP |
| No review comment found | Preamble returns error → STOP |
| Already ready | "Already ready — no action needed." → STOP |
| `gh pmu` command fails | Report error → STOP |
| User declines all fixes | "No changes made." → STOP |
| Re-review finds new issues | Report — user can run `/resolve-review` again |
---
**End of /resolve-review Command**
