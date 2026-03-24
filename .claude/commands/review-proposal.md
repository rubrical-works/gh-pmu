---
version: "v0.70.0"
description: Review a proposal with tracked history (project)
argument-hint: "#issue [--with ...] [--mode ...] [--force]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /review-proposal
Reviews a proposal document linked from a GitHub issue. Document updates handled inline; issue updates by calling orchestrator.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command review-proposal`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
- Issue body contains `**File:** Proposal/[Name].md`
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to proposal |
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
Extract: `context`, `criteria`, `extensions`, `warnings`. Read proposal file. Not found: STOP.

<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 1b: Construction Context Discovery
Search `Construction/Design-Decisions/` and `Construction/Tech-Debt/` for keywords and issue references. Report gracefully.
### Step 2: Evaluate Criteria

<!-- USER-EXTENSION-START: criteria-customize -->
<!-- USER-EXTENSION-END: criteria-customize -->

**Step 2a: Auto-Evaluate Objective Criteria**
Re-read `.claude/metadata/proposal-review-criteria.json` from disk. Emit pass/warn/fail.
Graceful degradation with inline defaults. Skip criteria missing `autoCheckMethod`. All non-blocking.
**Diagrams check:** Verify referenced diagram files exist on disk.
**Step 2a-gate: Path Analysis Gate**
If `path-analysis-present` is warn/fail: STOP, AskUserQuestion ("Run /paths now" or "Continue without"). Re-evaluate after /paths.
**Step 2b: Ask Subjective Criteria**
Display scope context. Use AskUserQuestion. Skipped as "Skipped". **Solo:** skip.
**Step 2c: Extension Criteria** (if `--with`)
**Step 2d: Recommendation**
Ready for implementation / Ready with minor revisions / Needs revision / Needs major rework.
Extensions escalate only. Applicability filtering.
### Step 3: Update Proposal File
Increment `**Reviews:** N`. Append to `## Review Log` (before `**End of Proposal**` or at end). Never edit existing rows.
### Step 4: Write Findings
Write to `.tmp-$ISSUE-findings.json`. Non-`--with` tip appended.

<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

### Closing Notification
Output `closingNotification`.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | STOP |
| Proposal not found | STOP |
| Issue closed | Ask user |
| File write fails | STOP |
---
**End of /review-proposal Command**
