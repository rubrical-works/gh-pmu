---
version: "v0.84.0"
description: Audit project command specs for LLM processing reliability (project)
argument-hint: "[all|<command-name>|<group description>]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /audit-commands
Audit command specs for formatting weaknesses impacting LLM processing reliability. Respects managed/extensible/local boundary.
**Skill Dependency:** Loads `command-spec-audit` skill for evaluation rubric.
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
- `framework-config.json` with `frameworkPath` set
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `all` | No | Audit every auditable command |
| `<command-name>` | No | Audit a single command |
| `<group description>` | No | NL grouping (e.g., "review commands") |
If no argument, prompt user for scope.
## Execution Instructions
**REQUIRED:** Before executing:
1. **Load Skill:** Read `command-spec-audit` skill's `SKILL.md`
2. **Generate Todo List:** Use `TodoWrite` from workflow steps
3. **Track Progress:** Mark todos `in_progress` -> `completed`
## Workflow
### Step 1: Load Manifest and Classify Commands
Read `{frameworkPath}/framework-manifest.json`, extract `managedCommands` and `extensibleCommands`. List `.md` files in `.claude/commands/`. Classify:
| Category | Source | Audit Scope |
|----------|--------|-------------|
| **Managed** | In `managedCommands` | Skip entirely |
| **Extensible** | In `extensibleCommands` | Extension points only |
| **Local** | Not in either | Full spec audit |
Report classification summary.
### Step 2: Resolve Scope
| Input | Resolution |
|-------|-----------|
| `all` | All extensible + local (managed always skipped) |
| `<command-name>` | `.claude/commands/<name>.md` |
| `<group description>` | Match by NL against frontmatter descriptions |
Managed -> report and skip. Not found -> STOP.
### Step 3: Audit Each Command
#### Step 3a: Extensible Commands -- Extension Point Audit
1. Read full command, extract content within each `<!-- USER-EXTENSION-START: {point} -->` / `<!-- USER-EXTENSION-END: {point} -->` block
2. All blocks empty -> skip
3. Content exists -> evaluate against **Category 4: Extension Points** rubric: formatting, conflicts, size (>50 lines), marker integrity
4. Record: criterion, severity, location, finding, recommendation
#### Step 3b: Local Commands -- Full Spec Audit
1. Read full command
2. Evaluate all 4 rubric categories: Structural Integrity (5), Decision Formatting (4), Execution Reliability (6), Extension Points (6)
3. Record: criterion, severity, location, finding, recommendation
### Step 4: Report or Create Issue
Zero findings -> report and skip. Findings exist -> create enhancement issue:
```bash
gh pmu create --title "[Audit]: <command-name> -- N findings" --label enhancement --status backlog -F .tmp-body.md
rm .tmp-body.md
```
Issue body: command name, source path, date, rubric, audit type, findings table (criterion/severity/location/finding/recommendation), severity summary.
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
**STOP.** Audit-only -- do not implement fixes.
## Error Handling
| Situation | Response |
|-----------|----------|
| No commands found | STOP |
| `framework-manifest.json` missing | Treat all as local, continue |
| Command unreadable | Skip, continue |
| Skill not loaded | Use inline criteria, continue |
| `gh pmu create` fails | Report, continue |
| Managed command requested | Report and skip |
**End of /audit-commands Command**
