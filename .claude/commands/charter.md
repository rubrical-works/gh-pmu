---
version: "v0.62.1"
description: View, create, or manage project charter
argument-hint: "[update|refresh|validate]"
---
<!-- MANAGED -->

# /charter
Context-aware charter command. Shows summary if exists, starts creation if missing.

## Usage
| Command | Description |
|---------|-------------|
| `/charter` | Show charter summary (if exists) or start creation (if missing) |
| `/charter update` | Update specific charter sections |
| `/charter refresh` | Re-extract from code, merge with existing |
| `/charter validate` | Check current work against charter scope |

## Execution Instructions
**REQUIRED:** Before executing:
1. **Generate Todo List:** Parse workflow steps, use `TodoWrite` to create todos
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
3. **Post-Compaction:** If resuming, re-read this spec and regenerate todos

## Template Detection
**Pattern:** `/{[a-z][a-z0-9-]*}/`
| Scenario | Handling |
|----------|----------|
| ANY placeholder present | Treat as template |
| Empty sections, no placeholders | Treat as complete |

## Workflow

### /charter (No Arguments)
**Step 1: Check for charter**
```bash
test -f CHARTER.md
```
**Step 2: If exists, check for template placeholders**
```bash
grep -E '\{[a-z][a-z0-9-]*\}' CHARTER.md
```
**If TEMPLATE (has placeholders):** → Proceed to Step 3 (Extraction/Inception)
**If COMPLETE (no placeholders):**
1. Read and display charter summary
2. Show: Project name, vision, current focus, tech stack
3. Mention: "Run `/charter update` to modify, `/charter validate` to check scope"
**Step 3: If no charter OR template detected**
**Charter is mandatory.** Automatically proceed:
1. Check if codebase has existing code
2. If has code → Use extraction mode
3. If empty → Use inception mode
4. Proceed directly to charter creation (no skip option)

### Extraction Mode (Existing Projects)
**Step 1:** Load `Skills/codebase-analysis/SKILL.md`
**Step 2:** Analyze codebase (tech stack, architecture, test parsing, NFR detection)
**Step 3:** Present findings, ask user to confirm/adjust
**Step 4:** Generate CHARTER.md and Inception/ artifacts from confirmed findings

### Inception Mode (New Projects)

#### Essential Questions (Always Asked)
| # | Question | Maps To |
|---|----------|---------|
| 1 | What are you building? (1-2 sentences) | CHARTER.md Vision |
| 2 | What problem does it solve? | Inception/Charter-Details.md Problem Statement |
| 3 | What technology/language? | CHARTER.md Tech Stack |
| 4 | What's in scope for v1? (3-5 items) | CHARTER.md In Scope |
| 5 | What testing framework? (conditional) | Inception/Test-Strategy.md Framework |
**Note:** Q5 only asked for testable projects (skip for docs/config repos).

#### Testing Framework Question (Conditional)
| Tech Stack | Ask? | Options |
|------------|:----:|---------|
| TypeScript/JS/Node | Yes | Jest, Vitest, Bun test |
| Python | Yes | pytest, unittest |
| Go | Yes | testing, testify |
| Rust | Yes | cargo test |
| Java/Kotlin | Yes | JUnit, TestNG |
| C#/.NET | Yes | xUnit, NUnit, MSTest |
| Documentation-only | Skip | N/A |
**Skip Detection:** If Q3 contains "documentation", "docs", "config", "terraform", "ansible" → skip Q5, set framework to "N/A - non-code project"

#### Deployment Platform Question (Q3a — Conditional)
**Trigger:** Deployable project detected from Q3 answer — web framework (React, Next.js, Express, Flask, Rails, Django, etc.), frontend build tool, Docker, or user description containing "web app", "API", "service", "site".
**Skip:** CLI tools, libraries, documentation-only projects, infrastructure repos (terraform, ansible, pulumi).
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
1. Write `deploymentTarget` to `framework-config.json`:
   - Vercel → `"vercel"`, Railway → `"railway"`, DigitalOcean → `"digitalocean"`, Render → `"render"`
   - Other/Not decided → `"other"`, Self-hosted/Not applicable → `null`
2. If a platform was selected (not "other" or null), auto-install the corresponding deployment skill:
   ```bash
   node .claude/scripts/shared/install-skill.js <skill-name>
   ```
   | Platform | Skill |
   |----------|-------|
   | Vercel | `vercel-project-setup` |
   | Railway | `railway-project-setup` |
   | DigitalOcean | `digitalocean-app-setup` |
   | Render | `render-project-setup` |
3. After skill install, query `recipe-tech-mapping.json` for deployment recipes matching the platform and display available recipes

#### Complexity-Triggered Questions
| Trigger | Follow-Up |
|---------|-----------|
| **Web app** | "Will users need to log in?" / "What data will you store?" |
| **API service** | "Who will consume this API?" |
| **Multi-user** | "What access levels are needed?" |
| **Data handling** | "Any sensitive/personal data?" / "Compliance requirements?" |
| **External integrations** | "What external services?" / "Any constraints?" |
**Max 1-2 complexity questions to avoid overwhelming user.**

#### Dynamic Follow-Up Generation
- Analyze baseline answers for gaps/ambiguities
- Simple projects: 0-1 follow-ups; Complex: 2-4
- Skip questions already answered indirectly
| Project Complexity | Total Questions |
|--------------------|-----------------|
| Simple (CLI, utility) | 4 essential only |
| Medium (web app, API) | 4-6 questions |
| Complex (multi-service) | 6-8 questions |

#### Review Mode Question (Always Asked)
After essential questions and complexity follow-ups, ask about review mode:
**ASK USER (single-select via AskUserQuestion):**
```
What review mode should be used for this project?
- Solo: Single developer - skip team-oriented criteria
- Team (Recommended): 2-10 developers - include sizing, priorities, dependencies
- Enterprise: Large teams - all criteria plus effort estimation and risk assessment
```
**Default:** "team" if user doesn't select or non-interactive.
**After answer:**
1. Write `reviewMode` to `framework-config.json` (lowercase: "solo", "team", or "enterprise")
2. Show confirmation with mode-specific explanation

#### Artifact Generation from Answers
**Answer-to-Artifact Mapping:**
| Answer | Primary Artifact |
|--------|------------------|
| What building? | CHARTER.md → Vision |
| What problem? | Inception/Charter-Details.md → Problem Statement |
| What technology? | CHARTER.md → Tech Stack |
| What's in scope? | CHARTER.md → In Scope |
| Testing framework? | Inception/Test-Strategy.md → Framework |
| Review mode? | framework-config.json → reviewMode |
**Generation Process:**
1. Create lifecycle directory structure:
   ```bash
   mkdir -p Inception Construction/Test-Plans Construction/Design-Decisions Construction/Tech-Debt Transition
   ```
2. Generate CHARTER.md (Vision, Tech Stack, In Scope, Status: Draft)
3. Generate Inception/ artifacts (Charter-Details, Tech-Stack, Scope-Boundaries, Constraints, Architecture, Test-Strategy, Milestones)
4. Create Construction/ structure with .gitkeep files and README.md
5. Create Transition/ artifacts (Deployment-Guide, Runbook, User-Documentation)
6. Use "TBD" for sections without answers
7. Commit all artifacts: "Initialize project charter and lifecycle structure"
**Note:** Directories created after questions to avoid orphaned directories if user abandons mid-flow.

### /charter update
**Step 1:** Read current CHARTER.md and Inception/Charter-Details.md
**Step 2:** Ask what to update (Vision, Current Focus, Tech Stack, Scope, Milestones, Deployment Target)
**Step 3:** Apply updates, sync to CHARTER.md if vision changes, update Last Updated date
**Step 4:** If Tech Stack modified, trigger skill and recipe suggestions (NEW items only). Also detect new default skills not in current `projectSkills` (via `getDefaultSkills()` from `manage-skills.js`) and add them additively.
**Step 4b:** If Deployment Target selected: read existing `deploymentTarget` from `framework-config.json`. If changing platforms, uninstall old deployment skill and install new one. Update `deploymentTarget` in config. If no previous target existed, treat as fresh install.

### /charter refresh
**Step 1:** Load `Skills/codebase-analysis/SKILL.md`
**Step 2:** Analyze codebase
**Step 3:** Compare with existing Inception/ artifacts, identify differences
**Step 4:** Present diff, ask for confirmation
**Step 5:** Merge changes, commit "Charter refresh"
**Step 6:** Trigger skill and recipe suggestions. Detect new default skills not in current `projectSkills` (via `getDefaultSkills()` from `manage-skills.js`) — add them additively and report additions. Also, if tech stack changed, trigger keyword-based skill suggestions (NEW items only).

### /charter validate
**Step 1:** Load CHARTER.md and Inception/Scope-Boundaries.md
**Step 2:** Identify current work (issue, recent commits, staged changes)
**Step 3:** Compare against in-scope/out-of-scope items
**Step 4:** Report
| Finding | Action |
|---------|--------|
| Aligned | Proceed normally |
| Possibly out of scope | Ask user to confirm intent |
| Clearly out of scope | Suggest updating charter or revising work |

## Project Skills Selection
After charter creation, suggest relevant skills based on defaults and tech stack using `.claude/metadata/skill-keywords.json`.
**Step 1:** Re-read `.claude/metadata/skill-keywords.json` from disk (not memory) — contains `defaultSkills`, `skillKeywords`, and `groupKeywords`. Also re-read `.claude/metadata/skill-registry.json` from disk (not memory) for descriptions. Use `getDefaultSkills()` from `manage-skills.js` to load defaults.
**If `skill-keywords.json` missing:** Warn and skip (non-blocking).
**Step 1b: Load Default Skills:** Read `defaultSkills` array from `skill-keywords.json`. These are universally applicable skills that apply regardless of tech stack. Add all defaults to the candidate list before keyword matching. If `defaultSkills` is missing or empty, continue without defaults (non-blocking).
**Step 2:** Match tech stack keywords against skillKeywords entries (case-insensitive, whole-word). Collect all skills with at least 1 keyword match as candidates — no false positive from partial string matching. Also match groupKeywords — if group keyword matches, add ALL group.skills. Merge keyword-matched candidates with defaults. Deduplicate against existing `projectSkills`.
**If tech stack unknown:** Still present defaults. **If zero keyword matches found:** Present defaults only and continue.
**Step 3:** Present candidates via `AskUserQuestion` with multi-select. Show skill name and description for each candidate. **Default skills are pre-selected** (checked by default). Users can deselect defaults if desired. Mark default skills with `[default]` in the description.
**Step 3b: Existing Project — Additive Merge:** Read existing `projectSkills`, filter already-present candidates. If all relevant skills (including defaults) are enabled, report and skip. Present only NEW candidates. Merge additively.
**Step 4:** Store confirmed skills in `framework-config.json` `projectSkills` array, sorted alphabetically. Additive merge with existing.
**Step 4b:** Deploy skills via `node .claude/scripts/shared/install-skill.js <skill-names...>`
**Step 5:** Report installed skills

## Extension Recipe Suggestions
After skill selection, suggest relevant extension recipes.
**Triggers:** `/charter` (creation), `/charter update` (if Tech Stack modified), `/charter refresh`
**Skip if:** `"extensionSuggestions": false` or no release commands installed
**Step 1:** Re-read `.claude/metadata/recipe-tech-mapping.json` from disk (not memory)
**Step 2:** Match tech stack against indicators and groupMappings
**Step 3:** Filter already-installed recipes (check extension points for content)
**Step 4: ASK USER:**
```
Extension Recipes Available:
- nodejs-tests: Run npm test before release validation
- dependency-audit: Check for vulnerabilities
Install? (y/n/select)
```
**Step 5:** Implement selected recipes (insert template between `USER-EXTENSION-START/END` markers)
**Step 6:** Report results
| Edge Case | Handling |
|-----------|----------|
| Extension point has content | Skip: "{point} already configured" |
| No release commands | Skip: "Extension recipes require release commands" |
| All suggestions installed | Report: "Extension recipes are up to date" |

## Token Budget
| Artifact | Tokens |
|----------|--------|
| CHARTER.md | ~150-200 |
| Charter-Details.md | ~1,200-1,500 |
| Scope-Boundaries.md | ~500-800 |
| skill-keywords.json | ~300-500 |

## Related Commands
- `/charter update` - Modify charter sections
- `/charter refresh` - Sync charter with codebase
- `/charter validate` - Check scope alignment
**End of /charter Command**
