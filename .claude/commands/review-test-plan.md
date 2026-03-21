---
version: "v0.67.2"
description: Review a test plan against its PRD (project)
argument-hint: "#issue [--mode ...] [--force]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /review-test-plan
Reviews a TDD test plan cross-referencing the source PRD for coverage completeness. Self-contained: handles document updates, finalization, and approval gate AC check-off.
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
- Issue body contains `**Test Plan:**` and `**PRD:**` file links
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to test plan |
| `--mode` | No | Transient override: `solo`, `team`, or `enterprise` |
| `--force` | No | Force re-review even if issue has `reviewed` label |
---
## Execution Instructions
**REQUIRED:** Create TodoWrite todos. Track progress. Post-compaction: re-read spec.
---
## Workflow
### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/review-preamble.js $ISSUE --no-redirect [--mode mode] [--force]
```
- `ok: false`: STOP. `earlyExit: true` (issue has `reviewed` label, no `--force`): report count, early exit, STOP.
Extract: `context`, `criteria`, `warnings`. Read both test plan and PRD. Either not found: STOP.

<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 2: Evaluate Criteria
**Step 2a: Auto-Evaluate Objective Criteria**
Re-read `.claude/metadata/test-plan-review-criteria.json` from disk. Use `shouldEvaluate()` to filter by reviewMode.
**Coverage Analysis (P0):** Map PRD ACs to test cases. Report as structured findings.
Graceful degradation with inline defaults. All non-blocking.
**Step 2b: Ask Subjective Criteria**
Use AskUserQuestion. Partial valid. **Solo:** skip. Coverage gaps as bullet-points (for `/resolve-review` compatibility).
**Step 2c: Recommendation**
Ready for approval / Ready with minor gaps / Needs revision / Needs major rework.
### Step 3: Update Test Plan File
Increment `**Reviews:** N`. Append to `## Review Log`. Never edit existing rows.
### Step 4: Finalize (Self-Contained)
```bash
node ./.claude/scripts/shared/review-finalize.js $ISSUE -F .tmp-$ISSUE-findings.json
```
Handles metadata, comment, labels. Clean up.
### Step 5: Approval Gate AC Check-Off (Conditional)
**Only "Ready for approval":**
```bash
node .claude/scripts/shared/review-ac-checkoff.js --issue $ISSUE --findings .tmp-$ISSUE-findings.json --move-status in_review
```
Not ready: skip.

<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

### Closing Notification
Output `closingNotification`.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | STOP |
| Test plan not found | STOP |
| PRD not found | STOP |
| Issue closed | Ask user |
| File write fails | STOP |
---
**End of /review-test-plan Command**
