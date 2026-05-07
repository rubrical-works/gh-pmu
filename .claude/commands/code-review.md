---
version: "v0.91.0"
description: Comprehensive code review with manifest-driven incremental tracking (project)
argument-hint: "[--full] [--status] [--scope <globs>] [--batch <N>] [--with <domains>] [--suggest]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /code-review
Methodical charter-aligned code review with manifest-driven incremental tracking. Previously approved+unchanged files skipped.
**Note:** Source code files only. Use `/review-issue`, `/review-prd`, `/review-proposal`, `/review-test-plan` for other artifacts.
## Prerequisites
- `CHARTER.md` exists (run `/charter` if missing)
- `framework-config.json` exists
## Arguments
| Argument | Description |
|----------|-------------|
| *(none)* | Incremental mode — skip approved+unchanged |
| `--full` | Bypass manifest, review all discovered |
| `--status` | Report manifest stats only, STOP |
| `--scope <globs>` | Comma-separated file patterns to limit scope |
| `--batch <N>` | Review N files then stop |
| `--with <domains>` | Comma-sep domain extensions or `all` |
| `--suggest` | Recommend domains (mutually exclusive with `--with`) |

Combinable: `--scope "src/**/*.js" --batch 10 --with security`
## Execution
**REQUIRED before executing:**
1. Parse workflow steps → `TaskCreate`
2. Mark tasks `in_progress` → `completed`
3. **Post-Compaction:** re-read this spec, regenerate tasks
## Workflow
### Step 1: Parse Arguments
Accept: none, `--full`, `--status`, `--scope`, `--batch N`, `--with`, `--suggest`. `--suggest` and `--with` mutually exclusive → error+STOP. Invalid → error+STOP.
### Step 2: Load Manifest
Read `.code-review-manifest.json`.
**Read** `.claude/scripts/shared/lib/code-review-manifest-schema.json` for manifest structure, required fields, and status enum values.
Not found → create empty. Malformed → warn, continue as `--full`.
### Step 2b: Status Report (--status)
Read manifest, discovery for counting only, report approved/flagged/pending/deferred/new counts. Directory breakdown if >20 files. **STOP**.
### Step 3: Discover Source Files
Resolve default include patterns from charter tech stack:
1. `detectTechStack(projectRoot)` from `.claude/scripts/shared/lib/detect-tech-stack.js`
2. `getGlobPatternsForTechs(techs)` from same module
3. Cross-reference CHARTER.md tech stack; include any additional patterns

**Read** `.claude/metadata/code-review-excludes.json` for default exclude patterns. Flatten `categories[].directories`. `env: "dev"` applies only in `idpf-praxis-dev`; `env: "deployed"` only in user projects; omitted = both. Test files always excluded (use `/bad-test-review`).
`--scope` → use those globs (still apply excludes).
### Step 4: Filter by Manifest (Incremental)
SHA-256 each file, compare to manifest:
| File State | Manifest | Hash | Action |
|------------|----------|------|--------|
| New | Absent | — | **Queue** |
| Existing | `approved` | Yes | **Skip** |
| Existing | `approved` | No | **Queue** re-review |
| Existing | `flagged` | Yes | **Skip** unchanged |
| Existing | `flagged` | No | **Queue** re-review |
| Existing | `deferred` | Any | **Skip** |
| Deleted | Present | — | **Remove** |

**Charter change:** CHARTER.md hash differs → re-review all.
**Domain change:** `--with` queues files previously approved without requested domain.
**`--full`:** Queue all discovered.
### Step 5: Load Charter-Aligned Review Criteria
Read `CHARTER.md` for goals, conventions, quality, tech stack, security.
| Category | Checks |
|----------|--------|
| **Correctness** | Logic, edge cases, off-by-one, null |
| **Security** | Injection, XSS, auth bypass, data exposure, OWASP top 10 |
| **Maintainability** | Complexity, duplication, coupling, cohesion, readability |
| **Naming** | Charter standards |
| **Error handling** | Missing try/catch, unhandled promises, silent failures |
| **Documentation** | JSDoc/docstrings per charter |
### Step 5b: Skill Loading
Check `projectSkills` in `framework-config.json`. Re-read `.claude/metadata/skill-keywords.json` and match keywords. Skills load lazily — supplementary only. If no skills installed, continue with charter-only criteria.
### Step 5a: Charter-Aware Domain Filtering
When `--with all`/`--with <domains>`:
1. Check `activeDomains` in `framework-config.json` → precedence (intersect requested)
2. Else read `CHARTER.md` (Tech Stack, In Scope) + `.claude/metadata/domain-signals.json`
3. Call `filterDomainsByCharter(requestedDomains, charterContent, domainSignalsJson, config)` from `load-review-extensions.js`
4. Log skipped: `"Skipping {domains} — not applicable per {source}"`
5. Pass applicable to Step 5c

`--with all` = all applicable.
**Error:** `domain-signals.json` missing/malformed → no filtering.
### Step 5d: Domain Suggestion (--suggest)
1. Read `CHARTER.md` + `.claude/metadata/domain-signals.json`
2. Call `suggestDomains(charterContent, domainSignalsJson)`
3. `AskUserQuestion`: Accept/Modify/Skip with per-domain reasoning
4. Accepted → feed to `--with` pipeline (5a → 5c)
5. Modified → user specifies
6. Skipped → standard review only
### Step 5c: Domain Extension Loading (--with)
1. Read `.claude/metadata/review-extensions.json`
2. Parse: `all` = 8 extensions, or comma-sep
3. Call `loadCodeReviewExtensions(projectDir, domainIds)` from `./.claude/scripts/shared/lib/load-review-extensions.js`
   **Return:** `{ ok, domains: { [id]: { description, domain, questions: string[] } }, warnings }`
   - `ok: false` → log `error`, fallback standard
   - `ok: true` → iterate `Object.entries(result.domains)`
   - Report `warnings` (non-blocking)
4. Use `questions[]` per domain as Code Review Questions
5. Unknown IDs: warn with list derived from the loaded `.claude/metadata/review-extensions.json` registry (source of truth — do not hardcode).

**Error:** All non-blocking → fallback standard review.
No `--with` → skip loading.
### Step 6: Per-File Review
Read each queued file; structured analysis:
| Field | Description |
|-------|-------------|
| File path | Relative |
| Line range | Start-end |
| Category | correctness/security/maintainability/naming/error-handling/documentation/domain |
| Severity | `high`/`medium`/`low`/`info` |
| Description | Issue |
| Recommendation | Fix |

**Severity:** High=security/correctness bug; Medium=maintainability/convention; Low=style/naming; Info=suggestion.
`--with` active: after standard review, apply domain criteria. Tag with domain. Domain can escalate, not downgrade.
### Step 7: Batch Mode
`--batch N`: limit N per invocation, save manifest, report progress.
### Step 8: Issue Creation
**MANDATORY:** Use `/bug` or `/enhancement` slash commands. NEVER raw `gh issue create` — bypasses project board.
| Finding | Command |
|---------|---------|
| Correctness/security defect | `/bug` |
| Error handling, refactoring, convention | `/enhancement` |

1. Present findings summary
2. `AskUserQuestion`: per finding, per group, or skip
3. Invoke `Skill("bug", "<title>")` or `Skill("enhancement", "<title>")`
4. Group findings sharing root cause
5. Record issue refs in manifest

**Info-level** reported but not offered as issues.
### Step 8b: Security Finding Label
If `--with security`/`--with all` and any security finding has ⚠️/❌:
```bash
gh issue edit $ISSUE --add-label=security-finding
```
All security findings ✅ → do not apply.
### Step 9: Structured Report
Save to `Construction/Code-Reviews/YYYY-MM-DD-report.md` **after** issue creation. Create dir if absent. Format: summary, findings grouped by severity with issue #s, per-file status, aggregate statistics, issues-created summary.
Each finding: `Issue: #1234` or `No issue` for info. Add **Issues Created** section.
### Step 10: Manifest Update
1. Write `.code-review-manifest.json`
2. Update `charter.contentHash`
3. Status: no findings→`approved`, findings→`flagged`
4. Record `reviewedAt`, `findingCount`
5. Record `domains` when `--with` active (merge, not replace)
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
**STOP.**
## Error Handling
| Situation | Response |
|-----------|----------|
| CHARTER.md missing | "No charter found. Run `/charter` first." → STOP |
| No source files | "No reviewable source files found in scope." → STOP |
| Manifest malformed | "Manifest corrupted. Running full review." → `--full` |
| File unreadable | Warn, skip |
| Issue creation fails | Warn, include in report |
| `--scope` matches nothing | "Scope pattern matched no files: {pattern}" → STOP |
| `framework-config.json` missing | Warn, no skill loading |
| `--with` unknown domain | Warn with list, skip |
| `--with` registry missing | Warn, fallback standard |
| `--with` criteria missing | Skip domain, warn |
| `--suggest`+`--with` | "`--suggest` and `--with` are mutually exclusive." → STOP |
| `domain-signals.json` missing | Warn, skip filtering |
| All domains filtered | "All requested domains filtered — falling back to standard review only" |

**End of /code-review Command**
