---
version: "v0.62.1"
description: Review a PRD with tracked history (project)
argument-hint: "#issue [--with ...] [--mode ...] [--force]"
---

<!-- EXTENSIBLE -->
# /review-prd
Reviews a PRD document linked from a GitHub issue. Delegates setup to `review-preamble.js`, keeping this spec focused on evaluation and model judgment. Document file updates (Reviews metadata, Review Log) are handled inline; issue body updates, comment posting, and label assignment are handled by the calling orchestrator.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command review-prd`
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue body must reference the PRD file path
---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number linked to the PRD (e.g., `#42` or `42`) |
| `--with` | No | Comma-separated domain extensions (e.g., `--with security,performance`) or `--with all` |
| `--mode` | No | Transient review mode override: `solo`, `team`, or `enterprise` |
| `--force` | No | Force re-review even if issue has `reviewed` label |
---
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Include Extensions:** Add todo for each non-empty `USER-EXTENSION` block
3. **Track Progress:** Mark todos `in_progress` → `completed` as you work
4. **Post-Compaction:** Re-read spec and regenerate todos after context compaction
---
## Workflow
### Step 1: Setup (Preamble Script)
```bash
node ./.claude/scripts/shared/review-preamble.js $ISSUE [--with extensions] [--mode mode] [--force]
```
Parse JSON output. If `ok: false`: report `errors[0].message` → **STOP**.
If `earlyExit: true`: report review count and **STOP**.
Extract: `context` (issue data, reviewNumber, PRD file path), `criteria`, `extensions`, `warnings`.
Read the PRD file at the extracted path. If file not found → **STOP**.
**Extension Loading:** The preamble handles extension loading from `.claude/metadata/review-extensions.json`. Unknown extension IDs produce warnings; missing registry or malformed JSON falls back to standard review only.

### Step 1b: Locate Test Plan
Check for `Test-Plan-*.md` in the same directory as the PRD file.
If test plan exists: read it for cross-reference during evaluation.
If no test plan found: warn and continue with PRD-only review (non-blocking).

<!-- USER-EXTENSION-START: pre-review -->
<!-- USER-EXTENSION-END: pre-review -->

### Step 2: Evaluate Criteria

<!-- USER-EXTENSION-START: criteria-customize -->
<!-- USER-EXTENSION-END: criteria-customize -->

**Step 2a: Auto-Evaluate Objective Criteria**
Re-read `.claude/metadata/prd-review-criteria.json` from disk (not memory). For each criterion, use `autoCheckMethod` to evaluate the PRD (and test plan if present). Emit ✅/⚠️/❌ with evidence. Evaluates requirements completeness, user story format, acceptance criteria, NFR adequacy (performance, security, scalability), cross-references, and story numbering.
**`requiresTestPlan` filtering:** Skip criteria with `requiresTestPlan: true` when no test plan.
**Graceful degradation:** If criteria file missing or malformed, warn and use inline defaults. Per-criterion validation: skip criteria missing `autoCheckMethod`. All failures non-blocking.

**Step 2b: Ask Subjective Criteria**
Load subjective criteria from `prd-review-criteria.json`. **Decomposition context preview:** Extract epic/story structure from PRD and display before asking. Use `AskUserQuestion` with each criterion's `question`, `header`, and `options`. Partial reviews valid. **Solo mode:** skip entirely.

**Step 2c: Extension Criteria** (if `--with` specified)
Evaluate extension domain criteria loaded by preamble. Auto-evaluate objective; ask subjective.

**Step 2d: Determine Recommendation**
- **Ready for backlog creation** — No blocking concerns
- **Ready with minor revisions** — Small issues
- **Needs revision** — Should be addressed first
- **Needs major rework** — Fundamental issues
Extension findings can **escalate** the recommendation but cannot downgrade it.
**Applicability Filtering:** Omit extension domain sections that produce no applicable findings. Only domains with findings appear in `**Extensions Applied:**`. If no domains produce findings when `--with` used, fall back to standard review with warning. At least one domain section must appear when `--with` is used.

### Step 3: Update PRD File
**Update `**Reviews:** N` field:** Increment if exists, add `**Reviews:** 1` after metadata fields if not.
**Update Review Log:** Append row to existing `## Review Log` table. If section missing, insert before `**End of PRD**` marker (or append at end if no marker — DD14 fallback).
```markdown
| # | Date | Reviewer | Findings Summary |
|---|------|----------|------------------|
| N | YYYY-MM-DD | Claude | [Brief one-line summary] |
```
Each review appends a new row. **Never edit or delete existing rows.**

### Step 4: Write Findings
Write structured findings to `.tmp-$ISSUE-findings.json` for the calling orchestrator. Include recommendation and all evaluated criteria.
For non-`--with` runs, append discoverability tip:
```
Tip: Use --with security,performance to add domain-specific review criteria.
Available: security, accessibility, performance, chaos, contract, qa, seo, privacy (or --with all)
```

<!-- USER-EXTENSION-START: post-review -->
<!-- USER-EXTENSION-END: post-review -->

---
## Error Handling
| Situation | Response |
|-----------|----------|
| Preamble `ok: false` | Report `errors[0].message` → STOP |
| PRD file not found | Report path error → STOP |
| Test plan not found | Warning, continue with PRD-only review |
| Issue closed | Ask user (from preamble context) |
| File write fails | Report error → STOP |
---
**End of /review-prd Command**
