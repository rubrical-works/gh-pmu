---
version: "v0.67.2"
description: Review a PRD with tracked history (project)
argument-hint: "#issue [--with ...] [--mode ...] [--force]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /review-prd
Reviews a PRD document linked from a GitHub issue. Self-contained: handles document updates, issue finalization, and AC check-off.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command review-prd`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
- Issue body references PRD file path
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to PRD |
| `--with` | No | Comma-separated domain extensions or `--with all` |
| `--mode` | No | Transient override: `solo`, `team`, or `enterprise` |
| `--force` | No | Force re-review even if issue has `reviewed` label |
---
## Execution Instructions
**REQUIRED:** Generate TodoWrite todos. Include extensions. Track progress. Post-compaction: re-read spec.
---
## Workflow
### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/review-preamble.js $ISSUE --no-redirect [--with extensions] [--mode mode] [--force]
```
- `ok: false`: STOP. `earlyExit: true` (issue has `reviewed` label, no `--force`): report count, early exit, STOP.
Extract: `context`, `criteria`, `extensions`, `warnings`. Read PRD file. Not found: STOP.
### Step 1b: Locate Test Plan
Check for `Test-Plan-*.md` in PRD directory. Found: read for cross-reference. Missing: warn, continue.

<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 2: Evaluate Criteria

<!-- USER-EXTENSION-START: criteria-customize -->
<!-- USER-EXTENSION-END: criteria-customize -->

**Step 2a: Auto-Evaluate Objective Criteria**
Re-read `.claude/metadata/prd-review-criteria.json` from disk. Evaluate PRD (and test plan). Emit pass/warn/fail.
Skip `requiresTestPlan: true` criteria when no test plan. Graceful degradation. All non-blocking.
**Step 2b: Ask Subjective Criteria**
Display epic/story structure before asking. Use AskUserQuestion. **Solo:** skip.
**Step 2c: Extension Criteria** (if `--with`)
**Step 2d: Recommendation**
Ready for backlog creation / Ready with minor revisions / Needs revision / Needs major rework.
Extensions escalate only. Applicability filtering.
### Step 3: Update PRD File
Increment `**Reviews:** N`. Append to `## Review Log` (DD14 fallback). Never edit existing rows.
### Step 4: Finalize (Self-Contained)
```bash
node ./.claude/scripts/shared/review-finalize.js $ISSUE -F .tmp-$ISSUE-findings.json
```
Handles metadata, comment, labels. Clean up. Non-`--with` tip appended.
### Step 5: AC Check-Off (Conditional)
**Only "Ready for" recommendation:**
```bash
node .claude/scripts/shared/review-ac-checkoff.js --issue $ISSUE --findings .tmp-$ISSUE-findings.json
```
No status transition. Not "Ready for": skip.

<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

### Closing Notification
Output `closingNotification`.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | STOP |
| PRD not found | STOP |
| Test plan not found | Warning |
| Issue closed | Ask user |
| File write fails | STOP |
---
**End of /review-prd Command**
