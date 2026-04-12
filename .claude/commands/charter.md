---
version: "v0.85.0"
description: View, create, or manage project charter
argument-hint: "[update|refresh|validate|--create-domain-entities]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /charter
Context-aware. Shows summary if exists, starts creation if missing.
## Usage
| Command | Description |
|---------|-------------|
| `/charter` | Show summary or start creation |
| `/charter update` | Update specific sections |
| `/charter refresh` | Re-extract from code, merge |
| `/charter validate` | Check current work against scope |
| `/charter --create-domain-entities` | Regenerate `domain-entities.json` |
## Execution
**REQUIRED:** `TodoWrite` todos from workflow steps; mark `in_progress`→`completed`. After compaction, re-read spec and regenerate todos.

## framework-config.json Writes — Use the Helper
ALL writes to `framework-config.json` (incl. `update`/`refresh`/`validate`) MUST go through `framework-config.js`. Covers `deploymentTarget`, `projectSkills`, `reviewMode`, `activeDomains`, any other field. Raw `fs.writeFileSync` is forbidden — helper validates against `.claude/metadata/framework-config.schema.json` (ajv draft-07) and rejects schema-invalid output.
```javascript
const fwconfig = require('./.claude/scripts/shared/lib/framework-config.js');
const config = fwconfig.read(process.cwd());
config.deploymentTarget = 'vercel';
config.projectSkills = [...new Set([...(config.projectSkills || []), ...newSkills])].sort();
fwconfig.write(process.cwd(), config); // throws on validation failure
```
If `fwconfig.write` throws, surface validation error and stop — do NOT retry with raw file I/O.
## Template Detection
Pattern: `/{[a-z][a-z0-9-]*}/`. ANY placeholder → template. No placeholders → complete.
## Workflow
### /charter (No Args)
1. `test -f CHARTER.md`
2. If exists, check placeholders: `grep -E '\{[a-z][a-z0-9-]*\}' CHARTER.md`
   - **TEMPLATE:** → Step 3
   - **COMPLETE:** Display summary (name, vision, current focus, tech stack). Mention `/charter update` and `/charter validate`.
3. **No charter OR template:** Charter is mandatory. Has code → extraction mode; empty → inception mode. Proceed directly (no skip).
### Extraction Mode (Existing)
1. Load `{frameworkPath}/Skills/codebase-analysis/SKILL.md`
2. Analyze codebase (tech stack, architecture, test parsing, NFR detection)
3. Present findings, ask user to confirm/adjust
4. Generate CHARTER.md and Inception/ artifacts
### Inception Mode (New)
#### Essential Questions (always)
| # | Question | Maps To |
|---|----------|---------|
| 1 | What are you building? (1-2 sentences) | CHARTER.md Vision |
| 2 | What problem does it solve? | Inception/Charter-Details.md Problem Statement |
| 3 | What technology/language? | CHARTER.md Tech Stack |
| 4 | What's in scope for v1? (3-5 items) | CHARTER.md In Scope |
| 5 | What testing framework? (conditional) | Inception/Test-Strategy.md Framework |

Q5 only for testable projects (skip docs/config repos).
#### Testing Framework (conditional)
| Tech Stack | Ask? | Options |
|------------|:----:|---------|
| TypeScript/JS/Node | Yes | Jest, Vitest, Bun test |
| Python | Yes | pytest, unittest |
| Go | Yes | testing, testify |
| Rust | Yes | cargo test |
| Java/Kotlin | Yes | JUnit, TestNG |
| C#/.NET | Yes | xUnit, NUnit, MSTest |
| Documentation-only | Skip | N/A |

**Skip:** Q3 contains "documentation", "docs", "config", "terraform", "ansible" → skip Q5, framework = "N/A - non-code project".
#### Deployment Platform (Q3a — conditional)
**Trigger:** Deployable from Q3 — web framework (React, Next.js, Express, Flask, Rails, Django, etc.), frontend build tool, Docker, or description containing "web app"/"API"/"service"/"site".
**Skip:** CLI, libraries, docs-only, infra repos (terraform, ansible, pulumi).
**ASK USER (single-select via AskUserQuestion):**
```
Where will this project be deployed?
- Vercel — Best for frontend, Next.js, static sites
- Railway — Best for full-stack apps, background workers
- DigitalOcean (App Platform) — Best for multi-component apps with databases
- Render — Best for web services with managed infrastructure
- Other/Not decided — No deployment skill installed
- Self-hosted/Not applicable — No deployment skill installed
```
**After answer:**
1. Write `deploymentTarget`: Vercel→`"vercel"`, Railway→`"railway"`, DigitalOcean→`"digitalocean"`, Render→`"render"`, Other→`"other"`, Self-hosted→`null`.
2. If platform selected (not "other"/null), copy skill from hub:
   ```bash
   cp -r {frameworkPath}/.claude/skills/<skill-name>/ .claude/skills/<skill-name>/
   ```
   | Platform | Skill |
   |----------|-------|
   | Vercel | `vercel-project-setup` |
   | Railway | `railway-project-setup` |
   | DigitalOcean | `digitalocean-app-setup` |
   | Render | `render-project-setup` |

   Add skill to `projectSkills` (additive merge, sorted alphabetically).
3. Query `recipe-tech-mapping.json` for matching deployment recipes; display.
#### Complexity-Triggered
| Trigger | Follow-Up |
|---------|-----------|
| **Web app** | "Will users need to log in?" / "What data will you store?" |
| **API service** | "Who will consume this API?" |
| **Multi-user** | "What access levels are needed?" |
| **Data handling** | "Any sensitive/personal data?" / "Compliance requirements?" |
| **External integrations** | "What external services?" / "Any constraints?" |

Max 1-2 to avoid overwhelm.
#### Dynamic Follow-Up
- Analyze baselines for gaps/ambiguities
- Simple: 0-1; Complex: 2-4
- Skip questions answered indirectly

| Complexity | Total Questions |
|--------------------|-----------------|
| Simple (CLI, utility) | 4 essential only |
| Medium (web app, API) | 4-6 |
| Complex (multi-service) | 6-8 |
#### Schema-Driven Domain Questions (after complexity triggers)
```javascript
const { generateQuestions } = require('.claude/scripts/shared/lib/schema-driven-questions.js');
const schemaQuestions = generateQuestions(answersCollectedSoFar);
```
Use `AskUserQuestion` with each question's `header`/`question` fields. Answers feed `domain-entities.json` (Step 7). If schema not found / no questions, skip silently.

**During `/charter update`:** Use `generateUpdateQuestions(currentDomainEntities)` — only missing/empty fields; existing values preserved.
#### Review Mode (always)
**ASK USER (single-select):**
```
What review mode should be used for this project?
- Solo: Single developer - skip team-oriented criteria
- Team (Recommended): 2-10 developers - include sizing, priorities, dependencies
- Enterprise: Large teams - all criteria plus effort estimation and risk assessment
```
**Default:** "team" if not selected / non-interactive.
**After:** Write `reviewMode` (lowercase: "solo"/"team"/"enterprise"). Show mode-specific confirmation.
#### Domain Profiling (conditional)
**Trigger:** Any project type except documentation-only.

**Step 1: Auto-detect** from tech stack and project type:
| Project Signal | Pre-check Domains |
|----------------|------------------|
| Web app / frontend / API | Security, Accessibility, SEO, API-Design |
| Multi-service / distributed | Observability, Contract-Testing |
| Stores personal data / compliance | Privacy |
| Multi-language / international | i18n |
| Performance-sensitive | Performance |
| Chaos/resilience mentioned | Chaos |
| Automated testing mentioned | QA-Automation |

**Step 2: ASK USER (multi-select):**
```
Which domain review criteria should auto-apply to this project?
Pre-checked domains are based on your tech stack. Adjust as needed.
```
Present all 11 domains pre-checked per detection.

**Step 3: After answer:**
1. Write `activeDomains` (lowercase IDs: `"security"`, `"accessibility"`, `"seo"`, `"privacy"`, `"observability"`, `"i18n"`, `"api-design"`, `"performance"`, `"chaos"`, `"contract"`, `"qa"`)
2. Report: `"Active domains: {list}. These will auto-apply to review commands."`

**Step 4 (`/charter refresh`):** Re-evaluate vs auto-detection from updated tech stack.
**Step 5 (`/charter update`):** Allow add/remove via multi-select.
#### Artifact Generation
| Answer | Primary Artifact |
|--------|------------------|
| What building? | CHARTER.md → Vision |
| What problem? | Inception/Charter-Details.md → Problem Statement |
| What technology? | CHARTER.md → Tech Stack |
| What's in scope? | CHARTER.md → In Scope |
| Testing framework? | Inception/Test-Strategy.md → Framework |
| Review mode? | framework-config.json → reviewMode |
| Active domains? | framework-config.json → activeDomains |

**Process:**
1. Create lifecycle dirs:
   ```bash
   mkdir -p Inception Construction/Test-Plans Construction/Design-Decisions Construction/Tech-Debt Transition
   ```
2. Generate CHARTER.md (Vision, Tech Stack, In Scope, Status: Draft)
3. Generate Inception/ artifacts (Charter-Details, Tech-Stack, Scope-Boundaries, Constraints, Architecture, Test-Strategy, Milestones)
4. Construction/ structure with .gitkeep and README.md
5. Transition/ artifacts (Deployment-Guide, Runbook, User-Documentation)
6. Use "TBD" where unanswered
7. Generate `domain-entities.json` at project root:
   ```javascript
   const { generateFromCharter } = require('.claude/scripts/shared/generate-domain-entities.js');
   const charter = fs.readFileSync('CHARTER.md', 'utf8');
   const entities = generateFromCharter(charter, version);
   // Validate against .claude/metadata/domain-entities-schema.json before writing
   // If validation fails, warn and skip (non-blocking)
   ```
8. Commit: "Initialize project charter and lifecycle structure"
9. Hint: `"Tip: Run /charter --create-domain-entities to regenerate domain-entities.json after manual charter edits."`

**Note:** Dirs created after questions to avoid orphans if user abandons.
### /charter update
1. Read current CHARTER.md and Inception/Charter-Details.md
2. Ask what to update (Vision, Current Focus, Tech Stack, Scope, Milestones, Deployment Target)
3. Apply, sync to CHARTER.md if vision changes, update Last Updated date
3a. Regenerate `domain-entities.json` via `generateFromCharter()` from `.claude/scripts/shared/generate-domain-entities.js`. Validate before writing. If missing, generate it (existing project migration). Include `"$schema"` first property pointing to `.claude/metadata/domain-entities-schema.json`.
3b. Hint: `"Tip: Run /charter --create-domain-entities to regenerate domain-entities.json after manual charter edits."`
4. If Tech Stack modified: trigger skill and recipe suggestions (NEW only). Detect new default skills not in current `projectSkills` (read `defaultSkills` from `.claude/metadata/skill-keywords.json`, cross-referenced with `.claude/metadata/skill-registry.json`) — copy from `{frameworkPath}/.claude/skills/`, add additively.
4b. If Deployment Target selected: read existing `deploymentTarget`. If platform changed, remove old deployment skill dir and copy new one from `{frameworkPath}/.claude/skills/<skill-name>/`. Update `deploymentTarget` and `projectSkills` (remove old, add new). No prior target → fresh install.
### /charter refresh
1. Load `{frameworkPath}/Skills/codebase-analysis/SKILL.md`
2. Analyze codebase
3. Compare with existing Inception/ artifacts, identify differences
4. Present diff, ask for confirmation
5. Merge changes, commit "Charter refresh"
5a. Regenerate `domain-entities.json` via `generateFromCharter()`. Run `verifyEntityCounts()` — report mismatches between charter counts and filesystem counts. Ask before updating charter counts. Validate and write. Include `"$schema"` first property.
5b. Hint: `"Tip: Run /charter --create-domain-entities to regenerate domain-entities.json after manual charter edits."`
6. Trigger skill/recipe suggestions. Detect new default skills not in current `projectSkills` (read `defaultSkills` from `.claude/metadata/skill-keywords.json`) — copy from `{frameworkPath}/.claude/skills/`, add additively, report. If tech stack changed, trigger keyword-based skill suggestions (NEW only).
### /charter validate
1. Load CHARTER.md and Inception/Scope-Boundaries.md
2. Identify current work (issue, recent commits, staged changes)
3. Compare against in-scope/out-of-scope
4. Report

| Finding | Action |
|---------|--------|
| Aligned | Proceed normally |
| Possibly out of scope | Ask user to confirm intent |
| Clearly out of scope | Suggest updating charter or revising work |
### /charter --create-domain-entities
Standalone regeneration of `domain-entities.json` from current charter.

**Step 1:** Check `CHARTER.md`
- **Exists:** Read, → Step 2
- **Missing:** Trigger full inception flow (`/charter` no args). After completion, `domain-entities.json` is generated as part of Step 7. Report: "Charter created. `domain-entities.json` generated." → **STOP**

**Step 2:** Generate
```javascript
const { generateFromCharter, verifyEntityCounts } = require('.claude/scripts/shared/generate-domain-entities.js');
const charter = fs.readFileSync('CHARTER.md', 'utf8');
const entities = generateFromCharter(charter, version);
```
Write to same dir as `CHARTER.md` with `"$schema"` first property pointing to `.claude/metadata/domain-entities-schema.json` (relative path from file location).

**Step 2b:** Verify entity counts vs filesystem
```javascript
const results = verifyEntityCounts(entities.entities);
```
For each `match: false`, report:
```
⚠️ {entity}: charter says {charterCount}, found {actualCount} on disk
```
If mismatches, ask: `"Update charter counts before writing? (y/n)"`. Yes → update `CHARTER.md` Key Entities table with actual counts, re-run `generateFromCharter()`. No → proceed with charter counts as-is. **Never auto-modify charter without user consent.**

**Step 3:** Validate against schema
- **Schema exists:** Validate. Valid → write. Invalid → warn and write anyway (non-blocking).
- **Missing:** Warn "Schema not found — skipping validation" and write.

**Step 4:** Report
```
Generated domain-entities.json ({entity count} entities)
```
If verification ran, append: `"Count verification: {N} entities checked, {M} mismatches"`

## Project Skills Selection
After charter creation, suggest skills via `.claude/metadata/skill-keywords.json`.
**Step 1:** Re-read `skill-keywords.json` from disk (not memory) — `defaultSkills`, `skillKeywords`, `groupKeywords`. Re-read `skill-registry.json` from disk for descriptions. **If `skill-keywords.json` missing:** warn, skip (non-blocking).
**Step 1b:** Read `defaultSkills` — universally applicable regardless of stack. Add to candidates before keyword matching. Missing/empty → continue (non-blocking).
**Step 2:** Match tech stack keywords against skillKeywords (case-insensitive, whole-word). Collect skills with ≥1 keyword match — no false positives from partial string matching. Match groupKeywords — if matched, add ALL group.skills. Merge with defaults. Deduplicate against existing `projectSkills`.
**Tech stack unknown:** Still present defaults. **Zero matches:** Defaults only.
**Step 3:** Present via `AskUserQuestion` multi-select. Show name and description. **Defaults pre-selected** (can deselect). Mark with `[default]` in description.
**Step 3b: Existing project — additive merge:** Read existing `projectSkills`, filter already-present. All relevant (incl. defaults) enabled → report and skip. Present only NEW. Merge additively.
**Step 4:** Store in `projectSkills`, sorted alphabetically. Additive merge.
**Step 4b:** Deploy by copy:
```bash
cp -r {frameworkPath}/.claude/skills/<skill-name>/ .claude/skills/<skill-name>/
```
COPIED (not symlinked) for per-project customization. Existing dir → skip (preserve customizations).
**Step 5:** Report installed skills.
## Extension Recipe Suggestions
After skill selection, suggest relevant recipes.
**Triggers:** `/charter` (creation), `/charter update` (if Tech Stack modified), `/charter refresh`
**Skip if:** `"extensionSuggestions": false` or no release commands installed
**Step 1:** Re-read `.claude/metadata/recipe-tech-mapping.json` from disk (not memory)
**Step 2:** Match tech stack against indicators and groupMappings
**Step 3:** Filter already-installed (check extension points for content)
**Step 4: ASK USER:**
```
Extension Recipes Available:
- nodejs-tests: Run npm test before release validation
- dependency-audit: Check for vulnerabilities
Install? (y/n/select)
```
**Step 5:** Insert template between `USER-EXTENSION-START/END` markers
**Step 6:** Report

| Edge Case | Handling |
|-----------|----------|
| Extension point has content | Skip: "{point} already configured" |
| No release commands | Skip: "Extension recipes require release commands" |
| All installed | Report: "Extension recipes are up to date" |

**End of /charter Command**
