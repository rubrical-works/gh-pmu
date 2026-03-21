---
version: "v0.67.2"
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
Multiple flags can be combined: `--scope "src/**/*.js" --batch 10 --with security`
## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Track Progress:** Mark todos `in_progress` -> `completed` as you work
3. **Post-Compaction:** If resuming after context compaction, re-read this spec and regenerate todos
## Workflow
### Step 1: Parse Arguments
Accept: no arguments (incremental), `--full`, `--status`, `--scope "globs"`, `--batch N`, `--with domains`, `--suggest`
`--suggest` and `--with` are mutually exclusive. If both provided, report error and STOP.
If invalid arguments provided, report error and STOP.
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
If manifest not found: create empty manifest (first run). If malformed: warn and continue as `--full`.
### Step 2b: Status Report (--status)
If `--status` flag: read manifest, run discovery for counting only, report approved count, flagged count, pending count, deferred count, and new (unreviewed) count. Show directory breakdown if > 20 files. **STOP** after report.
### Step 3: Discover Source Files
Scan codebase using Glob patterns. Auto-detect from charter tech stack:
- JS/TS: `**/*.js`, `**/*.ts`, `**/*.jsx`, `**/*.tsx`; Python: `**/*.py`; Go: `**/*.go`; Rust: `**/*.rs`; Java: `**/*.java`
**Default include patterns** (auto-detect from charter tech stack). **Default exclude patterns:**
| Category | Directories |
|----------|------------|
| Dependencies | `node_modules/`, `vendor/`, `Pods/`, `packages/` |
| Python | `__pycache__/`, `.venv/`, `venv/`, `site-packages/`, `.tox/` |
| Build output | `dist/`, `build/`, `out/`, `target/`, `bin/`, `obj/` |
| Framework builds | `.next/`, `.nuxt/`, `.svelte-kit/`, `.angular/` |
| Java/Gradle | `.gradle/`, `.maven/` |
| Test coverage | `coverage/`, `.nyc_output/` |
| System | `.git/` |
- Test files (reviewed by `/bad-test-review` instead)
If `--scope` provided: use those globs instead of defaults. Still apply excludes.
**Language detection:** 1. Check CHARTER.md tech stack 2. Scan root configs 3. Count extensions
### Step 4: Filter by Manifest (Incremental Mode)
Compute SHA-256 per file and compare against manifest:
| File State | Manifest Entry | Hash Match? | Action |
|------------|---------------|-------------|--------|
| New file | Not in manifest | N/A | **Queue** |
| Existing | `approved` | Yes | **Skip** |
| Existing | `approved` | No | **Queue** re-review |
| Existing | `flagged` | Yes (unchanged) | **Skip** — flagged unchanged |
| Existing | `flagged` | No (changed) | **Queue** re-review |
| Existing | `deferred` | Any | **Skip** |
| Deleted | In manifest | N/A | **Remove** from manifest |
**Charter change:** If CHARTER.md hash differs, re-review all files.
**Domain change:** When `--with` specified, files previously approved without requested domain are queued for re-review even if unchanged.
**`--full` mode:** Queue every discovered file.
### Step 5: Load Charter-Aligned Review Criteria
Read `CHARTER.md` for project goals, conventions, quality standards, tech stack, security requirements.
| Category | What to Check |
|----------|--------------|
| **Correctness** | Logic errors, edge cases, off-by-one, null handling |
| **Security** | Injection, XSS, auth bypass, sensitive data exposure, OWASP top 10 |
| **Maintainability** | Complexity, duplication, coupling, cohesion, readability |
| **Naming conventions** | Variable/function/file naming per charter standards |
| **Error handling** | Missing try/catch, unhandled promises, silent failures |
| **Documentation** | Missing JSDoc/docstrings for public APIs (per charter requirements) |
### Step 5b: Skill Loading
Check `projectSkills` in `framework-config.json`. Re-read `.claude/metadata/skill-keywords.json` from disk and match keywords:
| Skill | Domain | When Loaded |
|-------|--------|-------------|
| `anti-pattern-analysis` | Code smells, design pattern violations | Reviewing implementation files |
| `error-handling-patterns` | Error handling consistency | Error handling patterns detected |
| `codebase-analysis` | Architecture review, structural analysis | Reviewing module boundaries |
| `test-writing-patterns` | Test quality, assertion patterns | Reviewing test-adjacent files |
Skills loaded lazily. Supplementary, not required.
### Step 5a: Charter-Aware Domain Filtering
When `--with all` or `--with <domains>` is specified, filter domains by project applicability before loading extensions:
1. Check `activeDomains` in `framework-config.json` — if present, takes precedence (only load domains in both `activeDomains` and requested list)
2. If no `activeDomains`, read `CHARTER.md` (Tech Stack, In Scope sections) and `.claude/metadata/domain-signals.json`
3. Call `filterDomainsByCharter(requestedDomains, charterContent, domainSignalsJson, config)` from `load-review-extensions.js`
4. Log skipped domains: `"Skipping {domains} — not applicable per {source}"`
5. Pass only applicable domains to Step 5c
`--with all` becomes "all applicable domains" rather than all unconditionally.
**Error handling:** If `domain-signals.json` is missing or malformed, fall back to no filtering (all domains pass through).
### Step 5d: Domain Suggestion (--suggest)
If `--suggest` specified (mutually exclusive with `--with`):
1. Read `CHARTER.md` and `.claude/metadata/domain-signals.json`
2. Call `suggestDomains(charterContent, domainSignalsJson)` from `load-review-extensions.js`
3. Present ranked recommendations via `AskUserQuestion`:
   - Options: "Accept suggested ({domains})", "Modify selection", "Skip domains"
   - Display reasoning per domain (high/medium/low/none relevance)
4. If accepted: feed suggested domains into `--with` pipeline (proceed to Step 5a → 5c with suggested domains)
5. If modified: user specifies domains, proceed with those
6. If skipped: continue with standard review only (no domain extensions)
### Step 5c: Domain Extension Loading (--with)
If `--with` specified (or domains from `--suggest` accepted):
1. Read `.claude/metadata/review-extensions.json` registry
2. Parse: `all` loads all 8 extensions, comma-separated loads specific ones
3. Call `loadCodeReviewExtensions(projectDir, domainIds)` from `./.claude/scripts/shared/lib/load-review-extensions.js`
   **Return shape:** `{ ok: boolean, domains: { [id]: { description, domain, questions: string[] } }, warnings: string[] }`
   - If `ok: false`: log `result.error`, fall back to standard review
   - If `ok: true`: iterate `Object.entries(result.domains)` for loaded domain questions
   - Report `result.warnings` if any (non-blocking)
4. For each domain in `result.domains`: use `questions[]` array as Code Review Questions
5. Unknown IDs: warn with available list (`security, accessibility, performance, chaos, contract, qa, seo, privacy, observability, i18n, api-design`)
**Error handling:** All errors fall back to standard review only (non-blocking): unknown ID warns and skips, missing criteria file skips domain, missing registry falls back, no Code Review Questions section skips domain.
If `--with` not specified: skip extension loading.
### Step 6: Per-File Review
Read each queued file and perform structured analysis:
| Field | Description |
|-------|-------------|
| File path | Relative path |
| Line range | Start-end lines |
| Category | correctness, security, maintainability, naming, error-handling, documentation, or domain name |
| Severity | `high`, `medium`, `low`, `info` |
| Description | What the issue is |
| Recommendation | How to fix it |
**Severity:** High = security/correctness bug; Medium = maintainability/convention; Low = style/naming; Info = suggestion
When `--with` active: after standard review, apply domain criteria questions. Tag findings with domain name. Domain findings can escalate but not downgrade severity.
### Step 7: Batch Mode Support
`--batch N`: limit to N files per invocation, save manifest after batch, report progress.
### Step 8: Issue Creation
**MANDATORY:** All issue creation MUST use the `/bug` or `/enhancement` slash commands. Never use raw `gh issue create` — it bypasses project board integration and creates orphaned issues.
| Finding Type | Issue Command |
|-------------|---------------|
| Correctness/security defect | `/bug` (invokes `gh pmu create` internally) |
| Missing error handling, refactoring, convention | `/enhancement` (invokes `gh pmu create` internally) |
1. Present findings summary
2. Use `AskUserQuestion` for user choice: per finding, per group, or skip
3. For each approved finding, invoke the Skill tool: `Skill("bug", "<title>")` or `Skill("enhancement", "<title>")`
4. Group related findings sharing root cause
5. Record issue refs in manifest
**Info-level findings** reported but not offered as issues.
### Step 8b: Security Finding Label
If `--with security` or `--with all` was specified and any security domain finding has ⚠️ or ❌ severity, apply the `security-finding` label to each issue created from security findings:
```bash
gh issue edit $ISSUE --add-label=security-finding
```
If all security findings are ✅ (no issues detected), do not apply the label.
### Step 9: Structured Report
Save to `Construction/Code-Reviews/YYYY-MM-DD-report.md` **after** issue creation so issue numbers are available. Create `Construction/Code-Reviews/` directory if it does not exist. Format: summary, findings grouped by severity with issue numbers, per-file status, aggregate statistics, issues created summary.
Each finding entry must include its issue number (e.g., `Issue: #1234`) or `No issue` for info-level findings. Add an **Issues Created** summary section listing all issues created during this review.
### Step 10: Manifest Update
1. Write updated `.code-review-manifest.json`
2. Update `charter.contentHash`
3. Set status: no findings -> `approved`, has findings -> `flagged`
4. Record `reviewedAt` and `findingCount` per file
5. Record `domains` array when `--with` active (merge, don't replace)
6. Preserve skipped entries; remove deleted entries
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
| Manifest malformed | "Manifest corrupted. Running full review." -> continue as `--full` |
| Source file unreadable | Warn and skip, continue |
| Issue creation fails | Warn, include in report, continue |
| `--scope` matches no files | "Scope pattern matched no files: {pattern}" -> STOP |
| `framework-config.json` missing | Warn, continue without skill loading |
| `--with` unknown domain | Warn with available list, skip unknown, continue |
| `--with` registry missing | Warn, fall back to standard review only |
| `--with` criteria file missing | Skip domain, warn, continue |
| `--suggest` + `--with` both given | "`--suggest` and `--with` are mutually exclusive." → STOP |
| `domain-signals.json` missing | Warn, skip charter filtering (all domains pass through) |
| All domains filtered out | Warn: "All requested domains filtered — falling back to standard review only" |
**End of /code-review Command**
