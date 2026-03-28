---
version: "v0.76.0"
description: Review issues with type-specific criteria (project)
argument-hint: "#issue [#issue...] [--with ...] [--mode ...] [--force]"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /review-issue
Reviews one or more GitHub issues with type-specific criteria. Delegates setup to `review-preamble.js` and cleanup to `review-finalize.js`.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command review-issue`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | One or more issue numbers |
| `--with` | No | Comma-separated domain extensions or `--with all` |
| `--mode` | No | Transient override: `solo`, `team`, or `enterprise` |
| `--force` | No | Force re-review even if issue has `reviewed` label |
Multiple issues: reviews sequentially.
---
## Execution Instructions
**REQUIRED:** Generate TodoWrite todos. Include extensions. Track progress. Post-compaction: re-read spec.
---
## Workflow
**Multiple issues:** Process each through Steps 1-3.
### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/review-preamble.js $ISSUE [--with extensions] [--mode mode] [--force]
```
- `ok: false`: report error → **STOP**
- `context.redirect`: invoke corresponding skill with all original args (`#$ISSUE [--with extensions] [--mode mode] [--force]`) → **STOP**
- Closed: ask user
- `earlyExit: true` (reviewed label, no `--force`): report count → **STOP**
Extract: `context`, `criteria`, `extensions`, `warnings`.

<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 2: Evaluate Criteria

<!-- USER-EXTENSION-START: criteria-customize -->
<!-- USER-EXTENSION-END: criteria-customize -->

**Step 2a: Auto-Evaluate Objective Criteria**
Evaluate from `criteria.common` and `criteria.typeSpecific`. Re-read `.claude/metadata/review-criteria.json` from disk if stale. Emit pass/warn/fail with evidence.
**Step 2a-ii: Auto-Generate Proposed Solution/Fix**
Trigger: proposed-solution/fix check fails. NOT for epics. Placeholder: <20 chars or TBD/empty.
When triggered: analyze codebase, generate approach, files, steps, testing.
**Step 2a-iii: Epic-Specific Evaluation**
`sub-issue-review`: recursive review with auto-generate and body updates.
`construction-context`: scan Construction directories. Report gracefully.
**Step 2b: Ask Subjective Criteria**
Re-read `.claude/metadata/review-mode-criteria.json` from disk. Use AskUserQuestion. **Solo:** skip.
**Step 2c: Extension Criteria** (if `--with`)
**Step 2c-ii:** Security finding label if warn/fail detected.
**Step 2d: Recommendation**
Ready for work / Needs minor revision / Needs revision / Needs major rework.
### Step 3: Finalize (Script)
Write findings to `.tmp-$ISSUE-findings.json`. **Read** `.claude/scripts/shared/lib/findings-schema.json` for contract structure, required fields, status values, and recommendation values. Solo mode: `userEvaluated` is `[]`.
```bash
node ./.claude/scripts/shared/review-finalize.js $ISSUE -F .tmp-$ISSUE-findings.json
```
Handles body metadata, comment, labels, epic propagation. Clean up.
Non-`--with` tip: Available extensions listed.
**Extensions Applied** lists only domains with findings. At least one domain section when `--with` used; if none produce findings, fall back to standard with warning.

<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

### Step 4: Closing Notification
Output `closingNotification`. Multi-issue: combine.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | STOP |
| Issue not found | STOP |
| Issue closed | Ask user |
| Unknown label | Generic criteria |
| Finalize fails | Report error |
---
**End of /review-issue Command**
