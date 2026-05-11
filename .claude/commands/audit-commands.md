---
version: "v0.91.1"
description: Audit project command specs for LLM processing reliability (project)
argument-hint: "[all|<command-name>|<group description>]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /audit-commands
Audit command specs for formatting weaknesses impacting LLM processing reliability. Respects managed/extensible/local boundary — users only see findings for content they own.
**Skill Dependency:** Loads `command-spec-audit` skill for rubric.
## Prerequisites
- `gh pmu` installed, `.gh-pmu.json` configured
- `framework-config.json` with `frameworkPath` set
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `all` | No | Audit every auditable command in `.claude/commands/` |
| `<command-name>` | No | Single command (e.g., `work`, `my-custom-cmd`) |
| `<group description>` | No | NL grouping (e.g., "review commands") |

No argument → prompt for scope.
## Execution Instructions
**REQUIRED:**
1. **Load Skill:** Read `command-spec-audit` skill's `SKILL.md` for rubric
2. **Create Task List:** `TaskCreate` from workflow steps
3. **Track Progress:** Mark tasks `in_progress` → `completed`
## Workflow
### Step 1: Load Manifest and Classify Commands
Read `{frameworkPath}/framework-manifest.json` (`frameworkPath` from `framework-config.json`), extract `managedCommands` and `extensibleCommands`.

List all `.md` in `.claude/commands/`. Classify each:

| Category | Source | Audit Scope |
|----------|--------|-------------|
| **Managed** | In `managedCommands` | **Skip** — hub-managed |
| **Extensible** | In `extensibleCommands` | **Extension points only** — `USER-EXTENSION` blocks |
| **Local** | Not in either array | **Full spec audit** |

Report classification summary.
### Step 2: Resolve Scope
| Input | Resolution |
|-------|-----------|
| `all` | All extensible + local (managed always skipped) |
| `<command-name>` | `.claude/commands/<name>.md` |
| `<group description>` | Match by NL against frontmatter descriptions |

Managed → `"Command '<name>' is managed by the hub and cannot be audited."` → skip.
Not found → `"Command '<name>' not found."` → **STOP**.
### Step 3: Audit Each Command
#### 3a: Extensible — Extension Point Audit
1. **Read** full command file
2. **Extract** content within `USER-EXTENSION-START/END` blocks
3. All empty → `"No extension content: <command-name>"` → skip
4. Content exists → evaluate Category 4 criteria: formatting, conflicts, size (>50 lines), marker integrity
5. **Record** criterion, severity, extension point name + line range, finding, recommendation

#### 3b: Local — Full Spec Audit
1. **Read** full content
2. **Evaluate** against 4 rubric categories: Structural Integrity (5), Decision Formatting (4), Execution Reliability (6), Extension Points (6)
3. **Record** criterion, severity, location, finding, recommendation
### Step 4: Report or Create Issue
Zero findings → `"No issues found: <command-name>"`, skip.

Findings exist → create enhancement issue:
```bash
gh pmu create --title "[Audit]: <command-name> — N findings" --label enhancement --status backlog -F .tmp-body.md
rm .tmp-body.md
```

Issue body:
```markdown
## Command Spec Audit: <command-name>

**Source:** `.claude/commands/<command-name>.md`
**Audited:** YYYY-MM-DD
**Rubric:** `command-spec-audit` skill
**Audit Type:** {extensible (extensions only) | local (full spec)}

### Findings

| # | Criterion | Severity | Location | Finding | Recommendation |
|---|-----------|----------|----------|---------|----------------|
| 1 | ... | High | ... | ... | ... |

**Summary:** N High, N Medium, N Low
```
### Step 5: Summary Report
```
Audit Complete
  Commands skipped (managed): N
  Commands audited: N
    Extensible (extensions only): N
    Local (full spec): N
  Commands with findings: N
  Issues created: N
  Severity breakdown: N High, N Medium, N Low
```
**STOP.** Do not implement fixes — audit-only.
## Error Handling
| Situation | Response |
|-----------|----------|
| No commands found | "No command files found in .claude/commands/." → STOP |
| `framework-manifest.json` not found | "Warning: not found at {frameworkPath}. Treating all as local." → continue |
| File unreadable | "Cannot read: <path>." → skip, continue |
| Skill not loaded | "Warning: command-spec-audit skill not found. Using inline criteria." → continue |
| `gh pmu create` fails | "Failed to create issue: {error}" → report, continue |
| Managed command requested | "Managed by hub, cannot be audited." → skip |

**End of /audit-commands Command**
