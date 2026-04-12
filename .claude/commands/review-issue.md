---
version: "v0.85.0"
description: Review issues with type-specific criteria (project)
argument-hint: "#issue [#issue...] [--with ...] [--mode ...] [--force]"
copyright: "Rubrical Works (c) 2026"
---

<!-- EXTENSIBLE -->
# /review-issue
Reviews GitHub issues with type-specific criteria. Delegates setup to `review-preamble.js`, cleanup to `review-finalize.js`.
**Extension Points:** `.claude/metadata/extension-points.json` or `/extensions list --command review-issue`

## Prerequisites
`gh pmu` installed; `.gh-pmu.json` configured.

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue numbers (`#42` or `42 43 44`) |
| `--with` | No | Domain extensions (`security,performance` or `all`) |
| `--mode` | No | Review mode override: `solo`/`team`/`enterprise` |
| `--force` | No | Force re-review even if `reviewed` label present |

Multi: `/review-issue #42 #43 #44` reviews each sequentially.

## Execution Instructions
**REQUIRED:** Routed command — two-phase task creation:
1. **Phase 1:** Single `TaskCreate` for preamble step only.
2. **Phase 2:** After preamble confirms path (no redirect, no early exit), bulk-create tasks for all remaining steps + one per non-empty `USER-EXTENSION` block.
3. **On redirect/early exit:** Mark preamble completed; do NOT create remaining tasks.
4. **Track Progress:** mark each task `in_progress` → `completed` as you work it.
5. **Post-Compaction:** Re-read spec; resume from first incomplete task — no re-routing.

## Workflow
Multi-issue: process each sequentially through Steps 1–3.

### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/review-preamble.js $ISSUE [--with extensions] [--mode mode] [--force]
```
Parse JSON. Branches:
- `ok: false` → report `errors[0].message` → **STOP** (skip to next in batch)
- `context.redirect` set → invoke skill with all original args preserved: `Skill("review-proposal", args: "#$ISSUE [--with extensions] [--mode mode] [--force]")`. Pass `--with`/`--mode`/`--force`. → **STOP**
- `context.issue.state === "closed"` → ask user to confirm before proceeding
- `earlyExit: true` (has `reviewed` label, no `--force`) → report review count → **STOP**

Extract: `context` (type, reviewNumber, title, labels, body), `criteria` (common from `.claude/metadata/review-mode-criteria.json`, typeSpecific from `.claude/metadata/review-criteria.json`), `extensions`, `warnings`.

Extension loading handled by preamble via `.claude/metadata/review-extensions.json`. Unknown IDs warn; missing/malformed → standard review only.

<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 2: Evaluate Criteria

<!-- USER-EXTENSION-START: criteria-customize -->
<!-- USER-EXTENSION-END: criteria-customize -->

**2a: Auto-Evaluate Objective Criteria** — for each objective criterion in `criteria.common` and `criteria.typeSpecific`, evaluate by reading issue content. Re-read `.claude/metadata/review-criteria.json` from disk (not memory) if stale. Emit ✅/⚠️/❌ with evidence using `autoCheck` field for guidance.

**2a-ii: Auto-Generate Proposed Solution/Fix** (Bug, Enhancement, Story; NOT epic)
Trigger: `proposed-solution` or `proposed-fix-described` is ❌/⚠️. Placeholder = under 20 chars or matches "TBD"/"To be documented"/"..."/empty.
When triggered: analyze codebase, generate **Approach**, **Files to modify**, **Implementation steps**, **Testing considerations**. Present as `#### Proposed Solution (Auto-Generated)` (enhancement/story) or `#### Proposed Fix (Auto-Generated)` (bug). Otherwise content already substantive (>20 chars, no placeholder).

**2a-iii: Epic-Specific Evaluation** — for epic type: `sub-issue-review` requires recursive review of sub-issues through 2a–2b including 2a-ii with per-sub-issue body updates. `construction-context` scans `Construction/Design-Decisions/` and `Construction/Tech-Debt/` for files referencing sub-issue numbers. None found → report gracefully.

**2b: Ask Subjective Criteria** — for subjective criteria applicable to current reviewMode, use `AskUserQuestion`. Re-read `.claude/metadata/review-mode-criteria.json` from disk for question/options. Solo mode: skip entirely.

**2c: Extension Criteria** (if `--with`) — evaluate domain criteria loaded by preamble.

**2c-ii: Security Finding Label** — if `--with security`/`--with all` and any security finding ⚠️/❌:
```bash
gh issue edit $ISSUE --add-label=security-finding
```
All ✅ → no label.

**2d: Recommendation** — one of: `Ready for work` (no blocking concerns) / `Needs minor revision` (small) / `Needs revision` (must address before starting) / `Needs major rework` (fundamental).

### Step 3: Finalize (Script)
Write findings JSON to `.tmp-$ISSUE-findings.json`. **Read** `.claude/scripts/shared/lib/findings-schema.json` for contract structure, required fields, status values, recommendation values. Solo mode: `userEvaluated` always `[]`.

```bash
node ./.claude/scripts/shared/review-finalize.js $ISSUE -F .tmp-$ISSUE-findings.json
```
Finalize handles: body metadata (`**Reviews:** N` increment), structured comment posting, label assignment (`reviewed`/`pending`), epic sub-issue label propagation. Clean up temp file. Report summary from output.

For non-`--with` runs, append:
```
Tip: Use --with security,performance to add domain-specific review criteria.
Available: security, accessibility, performance, chaos, contract, qa, seo, privacy (or --with all)
```
**Extensions Applied** in review comment lists only domains producing findings (omit empty). At least one domain section must appear when `--with` used; if none produce findings, fall back to standard review with warning.

<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

### Step 3a: Interdependence Analysis (Multi-Issue Only)
Trigger: 2+ issues reviewed AND all eligible per `typeFilter` in `.claude/metadata/review-interdependence.json`. Eligible: `bug`, `enhancement`, `prd`, `test-plan`. Excluded: `proposal`, `epic` (excluded wins).

After all individual reviews complete:
```javascript
const { analyzeInterdependence, isEligibleForInterdependence } = require('.claude/scripts/shared/review-interdependence.js');
const allEligible = reviewedIssues.every(i => isEligibleForInterdependence(i.labels));
if (allEligible) {
  const result = analyzeInterdependence(reviewedIssues);
}
```
`reviewedIssues` = array of `{ number, title, type, labels, body }` collected during reviews.

Report: **Overlap** (shared scope), **Ordering** (suggested order + rationale), **Conflicts** (contradictory requirements), **Shared Criteria** (ACs in multiple issues).

If findings exist, offer:
```
Interdependence findings for #42, #43, #44:
[findings table]

Suggested order: #43 → #42 → #44

Update issues with cross-references? (y/n)
```
If accepted, add `Refs #N` notes to related issue bodies. No findings → `"No interdependence detected between reviewed issues."` and continue.

Configuration: dimensions and `typeFilter` in `.claude/metadata/review-interdependence.json` (config-driven; add to `eligible`/`excluded` to customize). Single-issue: skipped.

### Step 4: Closing Notification
Output `closingNotification` from finalize. Multi-issue: `"Reviews complete: #42, #43, #44"`.

## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | Report `errors[0].message` → STOP |
| Issue not found | Preamble error → STOP |
| Issue closed | Ask user (from preamble context) |
| Unknown label | Preamble uses generic criteria |
| Finalize fails | Report error; body may already be updated |

**End of /review-issue Command**
