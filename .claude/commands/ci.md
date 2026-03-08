---
version: "v0.58.0"
description: Manage GitHub Actions CI workflows interactively (project)
argument-hint: "[list|validate|add|recommend|watch] (no args shows status)"
---

<!-- EXTENSIBLE -->
# /ci
Interactive CI workflow management for GitHub Actions. View workflow status, manage CI features, and validate YAML configuration.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command ci`
---
## Prerequisites
- `.github/workflows/` directory (created if adding features)
- GitHub Actions enabled in repository
---
## Arguments
| Argument | Description |
|----------|-------------|
| *(none)* | Show workflow status (default) |
| `list` | List available CI features |
| `validate` | Validate workflow YAML files |
| `add <feature>` | Add a CI feature to workflows |
| `recommend` | Analyze project and suggest improvements |
| `watch [--sha <commit>]` | Monitor CI run status for a commit |
---
## Subcommands
### `/ci` (no arguments) - View Workflow Status
Display summary of existing GitHub Actions workflows: names, triggers, OS targets, language versions.
```bash
node .claude/scripts/shared/ci-status.js
```
### `/ci list` - List Available CI Features
Show which CI features can be added. Lists all 11 available features with enabled/disabled status, grouped by tier.
```bash
node .claude/scripts/shared/ci-list.js
```
### `/ci validate` - Validate Workflow YAML
Check workflow files for syntax errors, deprecated actions, missing concurrency groups, hardcoded secrets, permissive permissions. Groups findings by severity.
```bash
node .claude/scripts/shared/ci-validate.js
```
### `/ci add <feature>` - Add CI Feature

<!-- USER-EXTENSION-START: pre-add -->
<!-- USER-EXTENSION-END: pre-add -->

1. Validate feature name against `ci-features.json`
2. Detect project language via `ci-detect-lang.js`
3. Auto-detect target workflow file via `ci-detect-workflow.js`
4. Confirm target file with user
5. Add feature using `ci-modify.js` (creates backup first)
6. Report changes

<!-- USER-EXTENSION-START: post-add -->
<!-- USER-EXTENSION-END: post-add -->

```bash
node .claude/scripts/shared/ci-add.js <feature>
```
### `/ci recommend` - Analyze and Recommend

<!-- USER-EXTENSION-START: pre-recommend -->
<!-- USER-EXTENSION-END: pre-recommend -->

1. Analyze project stack via `ci-analyze.js`
2. Inventory existing workflows via `ci-recommend.js`
3. Compare against best practices, categorize as [Add], [Remove], [Alter], [Improve]
4. Present numbered menu via `ci-recommend-ui.js`
5. Apply selected recommendations via `ci-apply.js`

<!-- USER-EXTENSION-START: post-recommend -->
<!-- USER-EXTENSION-END: post-recommend -->

```bash
node .claude/scripts/shared/ci-analyze.js
node .claude/scripts/shared/ci-recommend.js
```
### `/ci watch` - Monitor CI Run
Monitor a GitHub Actions workflow run by commit SHA and report structured results.
| Argument | Required | Default | Description |
|----------|----------|---------|-------------|
| `--sha <commit>` | No | `HEAD` | Commit SHA to monitor |
| `--timeout <seconds>` | No | `300` | Max wait time |
| `--poll <seconds>` | No | `15` | Polling interval |
```bash
node .claude/scripts/shared/ci-watch.js --sha $SHA [--timeout $TIMEOUT] [--poll $POLL]
```
**Exit codes:** 0=pass, 1=fail, 2=timeout, 3=no-run-found, 4=cancelled
---
## Execution Instructions
### Step 1: Parse Subcommand
```bash
SUBCOMMAND="${1:-status}"
```
### Step 2: Verify CI Scripts Installed
Before routing, check if CI scripts exist:
```bash
ls .claude/scripts/shared/ci-status.js 2>/dev/null
```
**If script does not exist:**
```
CI scripts not installed. The /ci command requires the ci-cd-pipeline-design skill.

To install: /install-skill ci-cd-pipeline-design
To set up CI manually: create .github/workflows/ and add workflow YAML files.
```
→ **STOP** (do not attempt to run missing scripts)
### Step 3: Route to Handler
| Subcommand | Action |
|------------|--------|
| *(none)* or `status` | Execute `ci-status.js` |
| `list` | Execute `ci-list.js` |
| `validate` | Execute `ci-validate.js` |
| `add <feature>` | Execute `ci-add.js <feature>` |
| `recommend` | Execute `ci-analyze.js` + `ci-recommend.js` flow |
| `watch [--sha X]` | Execute `ci-watch.js --sha X` (default: HEAD) |
| Other | Error: `Unknown subcommand: $1` |

<!-- USER-EXTENSION-START: custom-subcommands -->
<!-- USER-EXTENSION-END: custom-subcommands -->

---
## Error Handling
| Situation | Response |
|-----------|----------|
| No `.github/workflows/` | "No .github/workflows/ directory found" |
| Empty workflows directory | "No workflow files found" |
| YAML parse error | Report file and error, continue with others |
| Unknown subcommand | "Unknown subcommand: {name}. Use: ci, ci list, ci validate, ci watch" |
---
**End of /ci Command**
