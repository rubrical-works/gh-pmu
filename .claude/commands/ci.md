---
version: "v0.61.0"
description: Manage GitHub Actions CI workflows interactively (project)
argument-hint: "[list|validate|add|recommend] (no args shows status)"
---
<!-- EXTENSIBLE -->

# /ci
Interactive CI workflow management for GitHub Actions. View workflow status, manage CI features, and validate YAML configuration.
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
**Purpose:** Display a summary of existing GitHub Actions workflows
**Output:**
- Lists all `.github/workflows/*.yml` files
- Shows workflow name, trigger events, OS targets, and language versions
- Reports if no workflows directory exists
- Table format for scannability
**Example:**
```
GitHub Actions Workflows:

┌──────────────────────────────────────────────────────────────┐
│ Name         │ Triggers     │ OS              │ Versions      │
├──────────────────────────────────────────────────────────────┤
│ CI           │ push, pull   │ ubuntu-latest   │ Node 18, 20   │
│ Deploy       │ release      │ ubuntu-latest   │ Node 20       │
└──────────────────────────────────────────────────────────────┘
```
**Implementation:**
```bash
node .claude/scripts/shared/ci-status.js
```

### `/ci list` - List Available CI Features
**Purpose:** Show which CI features can be added to workflows
**Output:**
- Lists all 11 available CI features
- Shows enabled/disabled status for each feature
- Groups features by tier (v1 High Value, v2 Medium Value)
- One-line description for each feature
- Summary count of enabled features
**Example:**
```
Available CI Features:

## v1 — High Value Features

  ✓ enabled     Dependency Caching
                  Cache dependencies to speed up builds

  ✗ disabled    Cross-OS Testing
                  Test across multiple operating systems

...

4 of 11 features enabled
```
**Implementation:**
```bash
node .claude/scripts/shared/ci-list.js
```

### `/ci validate` - Validate Workflow YAML
**Purpose:** Check workflow files for syntax errors, anti-patterns, and security issues
**Output:**
- Validates all `.github/workflows/*.yml` files
- Detects YAML syntax errors
- Identifies deprecated action versions (e.g., checkout@v2)
- Checks for missing concurrency groups on PR workflows
- Scans for hardcoded secrets/tokens
- Reports overly permissive permissions
- Groups findings by severity (error/warning/info)
**Example:**
```
Validating 2 workflow files...

## Warnings

1. [WARNING] test.yml
   PR-triggered workflow missing concurrency group

Found 1 issue: 0 errors, 1 warning, 0 info
```
**Implementation:**
```bash
node .claude/scripts/shared/ci-validate.js
```

### `/ci add <feature>` - Add CI Feature
**Purpose:** Add a CI feature to the appropriate workflow file
<!-- USER-EXTENSION-START: pre-add -->
<!-- USER-EXTENSION-END: pre-add -->
**Workflow:**
1. Validate feature name against registry (`ci-features.json`)
2. Detect project language via `ci-detect-lang.js`
3. Auto-detect target workflow file via `ci-detect-workflow.js`
4. Confirm target file with user before modifying
5. Add feature configuration using YAML-safe modification (`ci-modify.js`)
6. Create backup before modification
7. Report what was changed and in which file
<!-- USER-EXTENSION-START: post-add -->
<!-- USER-EXTENSION-END: post-add -->
**Implementation:**
```bash
node .claude/scripts/shared/ci-add.js <feature>
```

### `/ci recommend` - Analyze and Recommend
**Purpose:** Analyze project stack, compare against best practices, and suggest improvements
<!-- USER-EXTENSION-START: pre-recommend -->
<!-- USER-EXTENSION-END: pre-recommend -->
**Workflow:**
1. Analyze project stack via `ci-analyze.js` (language, test tooling, build system, deployment targets)
2. Inventory existing workflows via `ci-recommend.js`
3. Compare against best practices, categorize findings as [Add], [Remove], [Alter], [Improve]
4. Present numbered menu with selectable items via `ci-recommend-ui.js`
5. Apply selected recommendations sequentially via `ci-apply.js`
6. Report summary of changes
<!-- USER-EXTENSION-START: post-recommend -->
<!-- USER-EXTENSION-END: post-recommend -->
**Implementation:**
```bash
node .claude/scripts/shared/ci-analyze.js
node .claude/scripts/shared/ci-recommend.js
```

### `/ci watch` - Monitor CI Run
**Purpose:** Monitor a GitHub Actions workflow run by commit SHA and report structured results
**Arguments:**
| Argument | Required | Default | Description |
|----------|----------|---------|-------------|
| `--sha <commit>` | No | `HEAD` | Commit SHA to monitor |
| `--timeout <seconds>` | No | `300` | Max wait time |
| `--poll <seconds>` | No | `15` | Polling interval |
**Workflow:**
1. If no `--sha` specified, use `git rev-parse HEAD`
2. Run `ci-watch.js` and display results
3. Report per-workflow conclusion with exit code
**Implementation:**
```bash
node .claude/scripts/shared/ci-watch.js --sha $SHA [--timeout $TIMEOUT] [--poll $POLL]
```
**Exit codes:** 0=pass, 1=fail, 2=timeout, 3=no-run-found, 4=cancelled

## Execution Instructions

### Step 1: Parse Subcommand
```bash
SUBCOMMAND="${1:-status}"  # Default to "status" if no argument
```

### Step 2: Verify CI Scripts Installed
Before routing to any handler, check if the CI scripts exist:
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

### Step 4: Execute Handler
```bash
node .claude/scripts/shared/ci-status.js
```
<!-- USER-EXTENSION-START: custom-subcommands -->
<!-- Add your custom CI subcommands here -->
<!-- USER-EXTENSION-END: custom-subcommands -->

## Error Handling
| Situation | Response |
|-----------|----------|
| No `.github/workflows/` | Report: "No .github/workflows/ directory found" |
| Empty workflows directory | Report: "No workflow files found" |
| YAML parse error | Report file and error message, continue with other files |
| Unknown subcommand | Error: "Unknown subcommand: {name}. Use: ci, ci list, ci validate, ci watch" |
**End of /ci Command**
