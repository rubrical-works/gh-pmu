---
version: "v0.58.0"
description: Comprehensive code review with manifest-driven incremental tracking (project)
argument-hint: "[--full] [--status] [--scope <globs>] [--batch <N>]"
---
<!-- MANAGED -->
# /code-review
Performs a methodical, charter-aligned code review across a codebase with manifest-driven incremental tracking. Previously reviewed and unchanged files are skipped — only new or changed files are reviewed on each invocation.
**Note:** This command reviews **source code files**, not issues, PRDs, proposals, or test plans. Use `/review-issue`, `/review-prd`, `/review-proposal`, or `/review-test-plan` for those.
## Prerequisites
- `CHARTER.md` exists and is configured (run `/charter` if missing)
- `framework-config.json` exists in project root
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| *(none)* | | Normal incremental mode — skip approved+unchanged files |
| `--full` | | Bypass manifest, review all discovered files |
| `--status` | | Report manifest statistics only, then STOP |
| `--scope <globs>` | | Comma-separated file patterns to limit scope (e.g., `--scope "src/**/*.ts,lib/**/*.js"`) |
| `--batch <N>` | | Review N files then stop; next run picks up where left off |
Multiple flags can be combined: `--scope "src/**/*.js" --batch 10`
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse the workflow steps in this spec, then use `TodoWrite` to create todos
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
3. **Post-Compaction:** If resuming after context compaction, re-read this spec and regenerate todos
## Workflow
### Step 1: Parse Arguments
Accept these argument forms:
- No arguments — normal incremental mode
- `--full` — ignore manifest, review all files
- `--status` — report only, no review (handled in Step 2b)
- `--scope "src/**/*.ts,lib/**/*.js"` — comma-separated multiple globs to filter scope
- `--batch 10` — review at most N files, then stop
If invalid arguments provided, report error and STOP.
### Step 2: Load Manifest
Read `.code-review-manifest.json` from project root.
**Manifest schema:**
```json
{
  "version": 1,
  "lastRun": "2026-02-16",
  "charter": { "contentHash": "sha256:abc123..." },
  "files": {
    "src/utils/helper.js": {
      "contentHash": "sha256:def456...",
      "status": "approved",
      "reviewedAt": "2026-02-15",
      "findingCount": 0,
      "findings": [],
      "issueRefs": []
    }
  }
}
```
**Status values:** `pending` (never reviewed), `approved` (clean), `flagged` (has findings), `deferred` (user skipped)
**If manifest not found:** Create empty manifest and proceed (first run).
**If manifest malformed:** Warn: "Manifest corrupted. Running full review." Continue as `--full`.
### Step 2b: Status Report (--status)
If `--status` flag is present:
1. Read manifest (do NOT discover or review files)
2. Count new files not yet in manifest by running discovery (Step 3) but only for counting
3. Report file counts by status — approved count, flagged count, pending count, deferred count, and new (unreviewed) count
4. Show directory breakdown if > 20 files tracked
**STOP** after status report — do not proceed to review.
### Step 3: Discover Source Files
Scan codebase for reviewable source files using Glob patterns.
**Default include patterns** (auto-detect from charter tech stack):
- JavaScript/TypeScript: `**/*.js`, `**/*.ts`, `**/*.jsx`, `**/*.tsx`
- Python: `**/*.py`; Go: `**/*.go`; Rust: `**/*.rs`; Java: `**/*.java`
**Default exclude patterns:**
- `node_modules/`, `dist/`, `build/`, `out/`, `.git/`, `vendor/`, `__pycache__/`, `coverage/`, `.next/`, `.nuxt/`
- Test files (reviewed by `/bad-test-review` instead)
**If `--scope` provided:** Use comma-separated glob patterns instead of defaults. Still apply exclude patterns.
**Language detection:** 1. Check `CHARTER.md` tech stack 2. Scan root configs (`package.json` → JS/TS) 3. Count file extensions
### Step 4: Filter by Manifest (Incremental Mode)
For each discovered file, compute SHA-256 content hash and compare against manifest:
| File State | Manifest Entry | Hash Match? | Action |
|------------|---------------|-------------|--------|
| New file | Not in manifest | N/A | **Queue** for review |
| Existing | `approved` | Yes (unchanged) | **Skip** — already approved |
| Existing | `approved` | No (changed) | **Queue** for re-review |
| Existing | `flagged` | Yes (unchanged) | **Skip** — flagged unchanged, already tracked |
| Existing | `flagged` | No (changed) | **Queue** for re-review |
| Existing | `deferred` | Any | **Skip** — user deferred |
| Deleted | In manifest | N/A | **Remove** from manifest |
**Charter change detection:** If CHARTER.md hash differs from manifest's `charter.contentHash`, re-review all files.
**`--full` mode:** Bypass all manifest filtering. Queue every discovered file.
### Step 5: Load Charter-Aligned Review Criteria
Read `CHARTER.md` to extract project goals, conventions, quality standards, tech stack, security requirements.
| Category | What to Check |
|----------|--------------|
| **Correctness** | Logic errors, edge cases, off-by-one, null handling |
| **Security** | Injection, XSS, auth bypass, sensitive data exposure, OWASP top 10 |
| **Maintainability** | Complexity, duplication, coupling, cohesion, readability |
| **Naming conventions** | Variable/function/file naming per charter standards |
| **Error handling** | Missing try/catch, unhandled promises, silent failures |
| **Documentation** | Missing JSDoc/docstrings for public APIs (per charter requirements) |
### Step 5b: Skill Loading
Check `projectSkills` in `framework-config.json` for relevant skills.
Re-read `.claude/metadata/skill-keywords.json` from disk (not memory) and match keywords against review categories:
| Skill | Domain | When Loaded |
|-------|--------|-------------|
| `anti-pattern-analysis` | Code smells, design pattern violations | When reviewing implementation files |
| `error-handling-patterns` | Error handling consistency | When error handling patterns detected |
| `codebase-analysis` | Architecture review, structural analysis | When reviewing module boundaries |
| `test-writing-patterns` | Test quality, assertion patterns | When reviewing test-adjacent files |
Skills loaded lazily — only read `SKILL.md` when file matches skill's domain. Skills are supplementary, not required.
### Step 6: Per-File Review
For each queued file, read content and perform structured analysis.
| Field | Description |
|-------|-------------|
| File path | Relative path |
| Line range | Start-end lines (e.g., `42-55`) |
| Category | correctness, security, maintainability, naming, error-handling, documentation |
| Severity | `high`, `medium`, `low`, `info` |
| Description | What the issue is |
| Recommendation | How to fix it |
**Severity:** High = security/correctness bug; Medium = maintainability/convention; Low = style/naming; Info = suggestion
Present findings incrementally as each file is reviewed.
### Step 7: Batch Mode Support
When `--batch N` is specified, `--batch` limits review to N files per invocation:
1. Limit to N files, then stop
2. Save manifest after batch (Step 10)
3. Report: `Reviewed N of M queued files. Run again to continue.`
### Step 8: Structured Report
Save report to `Construction/Code-Reviews/YYYY-MM-DD-report.md`. Create directory if needed.
**Report format** — findings grouped by severity, then by file:
- Summary: files reviewed, finding counts, approved/flagged
- Findings by Severity (High/Medium/Low/Info tables)
- Per-File Status table
- Aggregate Statistics
### Step 9: Issue Creation
| Finding Type | Issue Command |
|-------------|---------------|
| Correctness/security defect | `/bug` |
| Missing error handling, refactoring, convention | `/enhancement` |
1. Present findings summary after all reviews
2. Use `AskUserQuestion` for user choice: create per finding, per group, or skip
3. Group related findings in same file when they share root cause
4. Record issue references in manifest (`issueRefs` per file)
**Info-level findings** reported but not offered as issues.
### Step 10: Manifest Update
1. Write updated `.code-review-manifest.json`
2. Update `charter.contentHash`
3. Set status: no findings → `approved`, has findings → `flagged`
4. Record `reviewedAt` and `findingCount` per file
5. Preserve skipped file entries; remove deleted file entries
### Step 11: Final Summary
```
Code Review Complete
────────────────────
Files reviewed: N
Findings: X (H high, M medium, L low, I info)
Issues created: Y
Report: Construction/Code-Reviews/YYYY-MM-DD-report.md
Manifest: .code-review-manifest.json updated

Next: Run --status to see cumulative progress
```
**STOP** — command complete.
## Error Handling
| Situation | Response |
|-----------|----------|
| CHARTER.md not found | "No charter found. Run `/charter` first." → STOP |
| No source files found | "No reviewable source files found in scope." → STOP |
| Manifest malformed | "Manifest corrupted. Running full review." → continue as `--full` |
| Source file unreadable | Warn and skip, continue |
| Issue creation fails | Warn, include in report, continue |
| `--scope` matches no files | "Scope pattern matched no files: {pattern}" → STOP |
| `framework-config.json` missing | Warn, continue without skill loading |
**End of /code-review Command**
