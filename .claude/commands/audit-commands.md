---
version: "v0.61.0"
description: Audit command spec formatting for LLM processing reliability (project)
argument-hint: "[all|<command-name>|<group description>]"
---
<!-- MANAGED -->

# /audit-commands
Audit command specification files for formatting weaknesses that impact LLM processing reliability. Creates one enhancement issue per command with findings.
**Skill Dependency:** Loads `command-spec-audit` skill for evaluation rubric.

## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `all` | No | Audit every command in `.claude/commands/` |
| `<command-name>` | No | Audit a single command (e.g., `work`, `done`) |
| `<group description>` | No | NL grouping (e.g., "release commands", "review commands") |
If no argument provided, prompt the user for scope.

## Execution Instructions
**REQUIRED:** Before executing:
1. **Load Skill:** Read `Skills/command-spec-audit/SKILL.md` for the evaluation rubric
2. **Generate Todo List:** Use `TodoWrite` to create todos from workflow steps
3. **Track Progress:** Mark todos `in_progress` -> `completed` as you work

## Workflow

### Step 1: Resolve Scope
Parse the argument to determine which commands to audit:
| Input | Resolution |
|-------|-----------|
| `all` | List all `.md` files in `.claude/commands/` |
| `<command-name>` | Resolve to `.claude/commands/<name>.md` |
| `<group description>` | Match commands by NL description against command descriptions from frontmatter |
**If command not found:** Report `"Command '<name>' not found."` -> **STOP**

### Step 2: Audit Each Command
For each in-scope command file:
1. **Read** the full command file content
2. **Evaluate** against all 4 rubric categories from `command-spec-audit` skill:
   - **Structural Integrity** (5 criteria)
   - **Decision Formatting** (4 criteria)
   - **Execution Reliability** (6 criteria)
   - **Extension Points** (6 criteria)
3. **Record findings** with: criterion name, severity, location (line/section), finding description, recommendation
**Evaluation approach:** Read the command file section by section. For each criterion in the rubric, check the detection heuristic against the actual content. Record all findings — do not fix anything.

### Step 3: Report or Create Issue
**If zero findings:** Report `"No issues found: <command-name>"` and skip to next command.
**If findings exist:** Create one enhancement issue per command:
```bash
gh pmu create --title "[Audit]: <command-name> — N findings" --label enhancement --status backlog -F .tmp-body.md
rm .tmp-body.md
```
Issue body format:
```markdown
## Command Spec Audit: <command-name>

**Source:** `.claude/commands/<command-name>.md`
**Audited:** YYYY-MM-DD
**Rubric:** `command-spec-audit` skill

### Findings

| # | Criterion | Severity | Location | Finding | Recommendation |
|---|-----------|----------|----------|---------|----------------|
| 1 | ... | High | ... | ... | ... |

**Summary:** N High, N Medium, N Low
```

### Step 4: Summary Report
After all commands audited:
```
Audit Complete
  Commands audited: N
  Commands with findings: N
  Issues created: N
  Severity breakdown: N High, N Medium, N Low
```
**STOP.** Do not implement fixes — this is an audit-only command.

## Error Handling
| Situation | Response |
|-----------|----------|
| No commands found | "No command files found in .claude/commands/." -> STOP |
| Command file unreadable | "Cannot read: <path>." -> skip, continue |
| Skill not loaded | "Warning: command-spec-audit skill not found. Using inline criteria." -> continue |
| `gh pmu create` fails | "Failed to create issue: {error}" -> report, continue to next |
**End of /audit-commands Command**
