---
version: "v0.80.0"
description: Manage GitHub Actions CI workflows interactively (project)
argument-hint: "[list|validate|add|recommend] (no args shows status)"
copyright: "Rubrical Works (c) 2026"
---
<!-- EXTENSIBLE -->
# /ci
Interactive CI workflow management for GitHub Actions.
**Extension Points:** See `.claude/metadata/extension-points.json` or run `/extensions list --command ci`
## Prerequisites
- `.github/workflows/` directory (created if adding features)
- GitHub Actions enabled in repository
## Arguments
| Argument | Description |
|----------|-------------|
| *(none)* | Show workflow status (default) |
| `list` | List available CI features |
| `validate` | Validate workflow YAML files |
| `add <feature>` | Add a CI feature to workflows |
| `recommend` | Analyze project and suggest improvements |
| `watch [--sha <commit>]` | Monitor CI run status for a commit |
## Subcommands
### `/ci` (no arguments) - View Workflow Status
Lists all `.github/workflows/*.yml` files with name, triggers, OS targets, and language versions in table format.
```bash
node .claude/scripts/shared/ci-status.js
```
### `/ci list` - List Available CI Features
Shows all 11 CI features with enabled/disabled status, grouped by tier (v1 High Value, v2 Medium Value).
```bash
node .claude/scripts/shared/ci-list.js
```
### `/ci validate` - Validate Workflow YAML
Checks for: YAML syntax errors, deprecated action versions, missing concurrency groups, hardcoded secrets, overly permissive permissions. Groups findings by severity.
```bash
node .claude/scripts/shared/ci-validate.js
```
### `/ci add <feature>` - Add CI Feature

<!-- USER-EXTENSION-START: pre-add -->
<!-- USER-EXTENSION-END: pre-add -->

**Workflow:**
1. Validate feature name against registry (`ci-features.json`)
2. Detect project language via `ci-detect-lang.js`
3. Auto-detect target workflow file via `ci-detect-workflow.js`
4. Confirm target file with user before modifying
5. Add feature using YAML-safe modification (`ci-modify.js`)
6. Create backup before modification
7. Report changes

<!-- USER-EXTENSION-START: post-add -->
<!-- USER-EXTENSION-END: post-add -->

```bash
node .claude/scripts/shared/ci-add.js <feature>
```
### `/ci recommend` - Analyze and Recommend

<!-- USER-EXTENSION-START: pre-recommend -->
<!-- USER-EXTENSION-END: pre-recommend -->

**Workflow:**
1. Analyze project stack via `ci-analyze.js`
2. Inventory existing workflows via `ci-recommend.js`
3. Compare against best practices, categorize as [Add], [Remove], [Alter], [Improve]
4. Present numbered menu via `ci-recommend-ui.js`
5. Apply selected recommendations via `ci-apply.js`
6. Report summary

<!-- USER-EXTENSION-START: post-recommend -->
<!-- USER-EXTENSION-END: post-recommend -->

```bash
node .claude/scripts/shared/ci-analyze.js
node .claude/scripts/shared/ci-recommend.js
```
### `/ci watch` - Monitor CI Run
| Argument | Required | Default | Description |
|----------|----------|---------|-------------|
| `--sha <commit>` | No | `HEAD` | Commit SHA to monitor |
| `--timeout <seconds>` | No | `300` | Max wait time |
| `--poll <seconds>` | No | `15` | Polling interval |
If no `--sha`, use `git rev-parse HEAD`. Exit codes: 0=pass, 1=fail, 2=timeout, 3=no-run-found, 4=cancelled.
```bash
node .claude/scripts/shared/ci-watch.js --sha $SHA [--timeout $TIMEOUT] [--poll $POLL]
```
## Execution Instructions
### Step 1: Parse Subcommand
```bash
SUBCOMMAND="${1:-status}"
```
### Step 2: Verify CI Scripts Installed
```bash
ls .claude/scripts/shared/ci-status.js 2>/dev/null
```
If missing: `CI scripts not installed. The /ci command requires the ci-cd-pipeline-design skill. To install: /install-skill ci-cd-pipeline-design` -> **STOP**
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
<!-- Add your custom CI subcommands here -->
<!-- USER-EXTENSION-END: custom-subcommands -->

## Error Handling
| Situation | Response |
|-----------|----------|
| No `.github/workflows/` | Report: "No .github/workflows/ directory found" |
| Empty workflows directory | Report: "No workflow files found" |
| YAML parse error | Report file and error, continue with others |
| Unknown subcommand | Error with usage hint |
**End of /ci Command**
