---
version: "v0.62.0"
description: Review issues with type-specific criteria (project)
argument-hint: "#issue [#issue...] [--with ...] [--mode ...] [--force]"
---

<!-- EXTENSIBLE -->
# /review-issue
Reviews one or more GitHub issues with type-specific criteria. Delegates setup to `review-preamble.js` and cleanup to `review-finalize.js`, keeping this spec focused on evaluation and model judgment.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command review-issue`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | One or more issue numbers (e.g., `#42` or `42 43 44`) |
| `--with` | No | Comma-separated domain extensions (e.g., `--with security,performance`) or `--with all` |
| `--mode` | No | Transient review mode override: `solo`, `team`, or `enterprise` |
| `--force` | No | Force re-review even if issue has `reviewed` label |
Accepts multiple issue numbers: `/review-issue #42 #43 #44` — reviews each sequentially.
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction
---
## Workflow
**For multiple issues:** Process each issue sequentially through Steps 1–3.
### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/review-preamble.js $ISSUE [--with extensions] [--mode mode] [--force]
```
Parse JSON output. If `ok: false`: report `errors[0].message` → **STOP** (skip to next if batch).
If `context.redirect` is set: invoke the corresponding skill (e.g., `Skill("review-proposal")`, `Skill("review-prd")`, `Skill("review-test-plan")`) → **STOP**.
If issue is closed (`context.issue.state === "closed"`): ask user to confirm before proceeding.
If `earlyExit: true` (issue has `reviewed` label, no `--force`): report review count and **STOP**.
Extract: `context` (issue type, reviewNumber, title, labels, body), `criteria` (common from `.claude/metadata/review-mode-criteria.json`, typeSpecific from `.claude/metadata/review-criteria.json`), `extensions`, `warnings`.
**Extension Loading:** The preamble handles extension loading from `.claude/metadata/review-extensions.json`. Unknown extension IDs produce warnings; missing registry or malformed JSON falls back to standard review only.

<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 2: Evaluate Criteria

<!-- USER-EXTENSION-START: criteria-customize -->
<!-- USER-EXTENSION-END: criteria-customize -->

**Step 2a: Auto-Evaluate Objective Criteria**
For each **objective** criterion from `criteria.common` and `criteria.typeSpecific`, evaluate by reading the issue content. Re-read `.claude/metadata/review-criteria.json` from disk (not memory) if criteria are stale. Emit ✅/⚠️/❌ with evidence. Use `autoCheck` field for evaluation guidance.

**Step 2a-ii: Auto-Generate Proposed Solution/Fix (Bug, Enhancement, Story)**
**Trigger:** `proposed-solution` or `proposed-fix-described` check is ❌/⚠️. Does NOT apply to epic types.
**Placeholder detection:** Treat as missing if under 20 chars or matches "TBD", "To be documented", "...", or empty.
When triggered: analyze codebase, generate **Approach**, **Files to modify**, **Implementation steps**, **Testing considerations**. Present as `#### Proposed Solution (Auto-Generated)` (enhancement/story) or `#### Proposed Fix (Auto-Generated)` (bug). When NOT triggered: issue already has substantive content (>20 chars, no placeholder).

**Step 2a-iii: Epic-Specific Evaluation**
For epic type: `sub-issue-review` criterion requires recursive review of sub-issues through Steps 2a-2b, including auto-generate (Step 2a-ii) with per-sub-issue body updates. `construction-context` criterion scans `Construction/Design-Decisions/` and `Construction/Tech-Debt/` for files referencing sub-issue numbers. No Construction context found — report gracefully.

**Step 2b: Ask Subjective Criteria**
For **subjective** criteria applicable to current reviewMode, use `AskUserQuestion`. Re-read `.claude/metadata/review-mode-criteria.json` from disk (not memory) for question/options fields. **Solo mode:** skip entirely.

**Step 2c: Extension Criteria** (if `--with` specified)
Evaluate extension domain criteria loaded by preamble.

**Step 2c-ii: Security Finding Label**
If `--with security` or `--with all` was specified and any security extension finding has ⚠️ or ❌ status, apply the `security-finding` label:
```bash
gh issue edit $ISSUE --add-label=security-finding
```
If all security findings are ✅ (no issues detected), do not apply the label.

**Step 2d: Determine Recommendation**
- **Ready for work** — No blocking concerns
- **Needs minor revision** — Small issues
- **Needs revision** — Should be addressed before starting work
- **Needs major rework** — Fundamental issues

### Step 3: Finalize (Script)
Write findings JSON to `.tmp-$ISSUE-findings.json` and call:
```bash
node ./.claude/scripts/shared/review-finalize.js $ISSUE -F .tmp-$ISSUE-findings.json
```
The finalize script handles: body metadata update (`**Reviews:** N` increment), structured review comment posting, label assignment (`reviewed`/`pending`), and epic sub-issue label propagation. Clean up temp file after. Report summary from finalize output.
For non-`--with` runs, append discoverability tip:
```
Tip: Use --with security,performance to add domain-specific review criteria.
Available: security, accessibility, performance, chaos, contract, qa, seo, privacy (or --with all)
```
**Extensions Applied** in the review comment lists only domains that produced findings (omit domains with no applicable findings). At least one domain section must appear when `--with` is used; if no domains produce findings, fall back to standard review with warning.

<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

---
## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | Report `errors[0].message` → STOP |
| Issue not found | Preamble returns error → STOP |
| Issue closed | Ask user (from preamble context) |
| Unknown label | Preamble uses generic criteria |
| Finalize fails | Report error, body may already be updated |
---
**End of /review-issue Command**
