---
version: "v0.80.0"
description: Comprehensive code review with manifest-driven incremental tracking (project)
argument-hint: "[--full] [--status] [--scope <globs>] [--batch <N>] [--with <domains>] [--suggest]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /code-review
Performs methodical, charter-aligned code review with manifest-driven incremental tracking. Previously reviewed unchanged files are skipped.
**Note:** Reviews **source code files** only. Use `/review-issue`, `/review-prd`, `/review-proposal`, or `/review-test-plan` for other artifacts.
## Prerequisites
- `CHARTER.md` exists and is configured (run `/charter` if missing)
- `framework-config.json` exists in project root
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| *(none)* | | Normal incremental mode -- skip approved+unchanged files |
| `--full` | | Bypass manifest, review all discovered files |
| `--status` | | Report manifest statistics only, then STOP |
| `--scope <globs>` | | Comma-separated file patterns to limit scope |
| `--batch <N>` | | Review N files then stop; next run picks up where left off |
| `--with <domains>` | | Comma-separated domain extensions or `--with all` |
| `--suggest` | | Analyze charter and codebase, recommend applicable domains (mutually exclusive with `--with`) |
Flags can combine: `--scope "src/**/*.js" --batch 10 --with security`
## Execution Instructions
**REQUIRED:** Before executing:
1. Parse workflow steps, use `TodoWrite` to create todos
2. Mark todos `in_progress` -> `completed` as you work
3. After compaction, re-read this spec and regenerate todos
## Workflow
### Step 1: Parse Arguments
Accept: no arguments (incremental), `--full`, `--status`, `--scope "globs"`, `--batch N`, `--with domains`, `--suggest`
`--suggest` and `--with` are mutually exclusive. If both provided, report error and STOP.
If invalid arguments, report error and STOP.
### Step 2: Load Manifest
Read `.code-review-manifest.json` from project root.
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
      "issueRefs": [],
      "domains": []
    }
  }
}
```
**Status values:** `pending` (never reviewed), `approved` (clean), `flagged` (has findings), `deferred` (user skipped)
If not found: create empty manifest. If malformed: warn, continue as `--full`.
### Step 2b: Status Report (--status)
If `--status`: report approved/flagged/pending/deferred/new counts. Directory breakdown if >20 files. **STOP** after report.
### Step 3: Discover Source Files
Auto-detect from charter tech stack: JS/TS (`**/*.js`, `**/*.ts`, `**/*.jsx`, `**/*.tsx`), Python (`**/*.py`), Go (`**/*.go`), Rust (`**/*.rs`), Java (`**/*.java`)
**Exclude:**
| Category | Directories |
|----------|------------|
| Dependencies | `node_modules/`, `vendor/`, `Pods/`, `packages/` |
| Python | `__pycache__/`, `.venv/`, `venv/`, `site-packages/`, `.tox/` |
| Build output | `dist/`, `build/`, `out/`, `target/`, `bin/`, `obj/` |
| Framework builds | `.next/`, `.nuxt/`, `.svelte-kit/`, `.angular/` |
| Java/Gradle | `.gradle/`, `.maven/` |
| Test coverage | `coverage/`, `.nyc_output/` |
| System | `.git/` |
Exclude test files (use `/bad-test-review`). If `--scope`: use those globs instead, still apply excludes.
**Language detection:** 1. CHARTER.md tech stack 2. Root configs 3. Extension counts
### Step 4: Filter by Manifest (Incremental Mode)
SHA-256 per file, compare against manifest:
| File State | Manifest Entry | Hash Match? | Action |
|------------|---------------|-------------|--------|
| New file | Not in manifest | N/A | **Queue** |
| Existing | `approved` | Yes | **Skip** |
| Existing | `approved` | No | **Queue** re-review |
| Existing | `flagged` | Yes | **Skip** |
| Existing | `flagged` | No | **Queue** re-review |
| Existing | `deferred` | Any | **Skip** |
| Deleted | In manifest | N/A | **Remove** from manifest |
**Charter change:** Hash differs -> re-review all. **Domain change:** `--with` specified, files missing requested domain -> re-review. **`--full`:** Queue all.
### Step 5: Load Charter-Aligned Review Criteria
Read `CHARTER.md` for goals, conventions, quality, tech stack, security.
| Category | What to Check |
|----------|--------------|
| **Correctness** | Logic errors, edge cases, off-by-one, null handling |
| **Security** | Injection, XSS, auth bypass, sensitive data, OWASP top 10 |
| **Maintainability** | Complexity, duplication, coupling, cohesion, readability |
| **Naming** | Variable/function/file naming per charter |
| **Error handling** | Missing try/catch, unhandled promises, silent failures |
| **Documentation** | Missing JSDoc/docstrings for public APIs |
### Step 5b: Skill Loading
Check `projectSkills` in `framework-config.json`, match `.claude/metadata/skill-keywords.json`:
| Skill | When Loaded |
|-------|-------------|
| `anti-pattern-analysis` | Reviewing implementation files |
| `error-handling-patterns` | Error handling patterns detected |
| `codebase-analysis` | Reviewing module boundaries |
| `test-writing-patterns` | Reviewing test-adjacent files |
Lazy loading, supplementary.
### Step 5a: Charter-Aware Domain Filtering
When `--with all` or `--with <domains>`:
1. Check `activeDomains` in `framework-config.json` (takes precedence)
2. If none, read `CHARTER.md` + `.claude/metadata/domain-signals.json`
3. Call `filterDomainsByCharter(requestedDomains, charterContent, domainSignalsJson, config)` from `load-review-extensions.js`
4. Log skipped domains
5. Pass applicable domains to Step 5c
If `domain-signals.json` missing/malformed: no filtering (all pass).
### Step 5d: Domain Suggestion (--suggest)
If `--suggest` (mutually exclusive with `--with`):
1. Read `CHARTER.md` + `.claude/metadata/domain-signals.json`
2. Call `suggestDomains(charterContent, domainSignalsJson)` from `load-review-extensions.js`
3. `AskUserQuestion`: "Accept suggested ({domains})", "Modify selection", "Skip domains"
4. Accepted -> feed into `--with` pipeline (Step 5a->5c)
5. Modified -> user specifies, proceed
6. Skipped -> standard review only
### Step 5c: Domain Extension Loading (--with)
If `--with` specified (or from `--suggest`):
1. Read `.claude/metadata/review-extensions.json`
2. `all` loads all extensions; comma-separated loads specific
3. Call `loadCodeReviewExtensions(projectDir, domainIds)` from `./.claude/scripts/shared/lib/load-review-extensions.js`
   Returns: `{ ok, domains: { [id]: { description, domain, questions[] } }, warnings[] }`
   - `ok: false` -> log error, fall back to standard
   - `ok: true` -> use `questions[]` per domain as review criteria
   - Report warnings (non-blocking)
4. Unknown IDs: warn with list (`security, accessibility, performance, chaos, contract, qa, seo, privacy, observability, i18n, api-design`)
All errors fall back to standard review (non-blocking). If `--with` not specified: skip.
### Step 6: Per-File Review
| Field | Description |
|-------|-------------|
| File path | Relative path |
| Line range | Start-end lines |
| Category | correctness, security, maintainability, naming, error-handling, documentation, or domain |
| Severity | `high`, `medium`, `low`, `info` |
| Description | What the issue is |
| Recommendation | How to fix it |
**Severity:** High=security/correctness; Medium=maintainability/convention; Low=style/naming; Info=suggestion
With `--with`: apply domain criteria after standard review. Domain findings can escalate but not downgrade severity.
### Step 7: Batch Mode Support
`--batch N`: limit N files, save manifest after batch, report progress.
### Step 8: Issue Creation
**MANDATORY:** Use `/bug` or `/enhancement` slash commands. Never raw `gh issue create`.
| Finding Type | Command |
|-------------|---------|
| Correctness/security defect | `/bug` |
| Missing error handling, refactoring, convention | `/enhancement` |
1. Present findings summary
2. `AskUserQuestion`: per finding, per group, or skip
3. Invoke `Skill("bug", "<title>")` or `Skill("enhancement", "<title>")`
4. Group related findings by root cause
5. Record issue refs in manifest
**Info-level** reported but not offered as issues.
### Step 8b: Security Finding Label
If `--with security`/`--with all` and security findings have high/medium severity:
```bash
gh issue edit $ISSUE --add-label=security-finding
```
If all clean, do not apply label.
### Step 9: Structured Report
Save to `Construction/Code-Reviews/YYYY-MM-DD-report.md` **after** issue creation. Create directory if needed. Include: summary, findings by severity with issue numbers, per-file status, statistics, issues created section.
### Step 10: Manifest Update
1. Write `.code-review-manifest.json`
2. Update `charter.contentHash`
3. No findings -> `approved`; has findings -> `flagged`
4. Record `reviewedAt`, `findingCount` per file
5. Record `domains[]` with `--with` (merge, don't replace)
6. Preserve skipped; remove deleted
### Step 11: Final Summary
```
Code Review Complete
--------------------
Files reviewed: N
Findings: X (H high, M medium, L low, I info)
Issues created: Y
Report: Construction/Code-Reviews/YYYY-MM-DD-report.md
Manifest: .code-review-manifest.json updated
Next: Run --status to see cumulative progress
```
**STOP** -- command complete.
## Error Handling
| Situation | Response |
|-----------|----------|
| CHARTER.md not found | "No charter found. Run `/charter` first." -> STOP |
| No source files found | "No reviewable source files found in scope." -> STOP |
| Manifest malformed | "Manifest corrupted. Running full review." -> `--full` |
| Source file unreadable | Warn, skip, continue |
| Issue creation fails | Warn, include in report, continue |
| `--scope` matches no files | "Scope pattern matched no files: {pattern}" -> STOP |
| `framework-config.json` missing | Warn, continue without skill loading |
| `--with` unknown domain | Warn with list, skip unknown, continue |
| `--with` registry missing | Warn, fall back to standard review |
| `--with` criteria missing | Skip domain, warn, continue |
| `--suggest` + `--with` | "Mutually exclusive." -> STOP |
| `domain-signals.json` missing | Warn, skip filtering |
| All domains filtered | Warn, fall back to standard review |
**End of /code-review Command**
