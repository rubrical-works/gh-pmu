---
version: "v0.85.0"
description: Audit project command specs for LLM processing reliability (project)
argument-hint: "[all|<command-name>|<group description>]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /audit-commands
Audit command specs for formatting weaknesses affecting LLM reliability. Respects managed/extensible/local boundary.
**Skill Dependency:** `command-spec-audit` (evaluation rubric).
---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured
- `framework-config.json` with `frameworkPath`
---
## Arguments
| Argument | Description |
|----------|-------------|
| `all` | Audit all auditable commands in `.claude/commands/` |
| `<command-name>` | Single command (e.g., `work`) |
| `<group description>` | NL grouping (e.g., "review commands") |

If no argument, prompt user for scope.
---
## Execution
**REQUIRED before executing:**
1. Read `command-spec-audit` skill's `SKILL.md` for rubric
2. `TodoWrite` todos from workflow steps
3. Mark todos `in_progress` → `completed`
---
## Workflow
### Step 1: Load Manifest and Classify
Read `{frameworkPath}/framework-manifest.json`; extract `managedCommands` and `extensibleCommands`. List `.claude/commands/*.md` and classify:

| Category | Source | Audit Scope |
|----------|--------|-------------|
| **Managed** | In `managedCommands` | **Skip** — hub-managed |
| **Extensible** | In `extensibleCommands` | **Extension points only** (`USER-EXTENSION-START/END` content) |
| **Local** | Neither | **Full spec audit** |

Report:
```
Command Classification:
  Managed (skipped): N
  Extensible (extensions only): N
  Local (full audit): N
```
### Step 2: Resolve Scope
| Input | Resolution |
|-------|-----------|
| `all` | All extensible + local (managed skipped) |
| `<command-name>` | `.claude/commands/<name>.md` |
| `<group description>` | NL match against frontmatter descriptions |

**Managed:** Report `"Command '<name>' is managed by the hub and cannot be audited."` → skip.
**Not found:** Report `"Command '<name>' not found."` → **STOP**.
### Step 3: Audit Each Command
#### 3a: Extensible — Extension Point Audit
1. Read file
2. Extract content within each `<!-- USER-EXTENSION-START: {point} -->`/`END` block
3. If all extension blocks empty: Report `"No extension content: <command-name>"` → skip
4. If content exists, evaluate against **Category 4: Extension Points**:
   - Extension formatting (matches parent style)
   - Extension conflicts (duplicates/contradicts built-ins)
   - Extension size (>50 lines)
   - Marker integrity (paired START/END)
5. Record findings: criterion, severity, location (point + lines), finding, recommendation
#### 3b: Local — Full Spec Audit
1. Read file
2. Evaluate against all 4 rubric categories:
   - **Structural Integrity** (5 criteria)
   - **Decision Formatting** (4 criteria)
   - **Execution Reliability** (6 criteria)
   - **Extension Points** (6 criteria)
3. Record findings as above
### Step 4: Report or Create Issue
**Zero findings:** Report `"No issues found: <command-name>"` → next.
**Findings exist:** Create one enhancement issue per command:
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
**STOP.** Audit-only — do not implement fixes.
---
## Error Handling
| Situation | Response |
|-----------|----------|
| No commands found | "No command files found in .claude/commands/." → STOP |
| `framework-manifest.json` missing | "Warning: framework-manifest.json not found at {frameworkPath}. Treating all commands as local." → continue |
| Unreadable file | "Cannot read: <path>." → skip |
| Skill not loaded | "Warning: command-spec-audit skill not found. Using inline criteria." → continue |
| `gh pmu create` fails | "Failed to create issue: {error}" → report, continue |
| Managed requested | "Command '<name>' is managed by the hub and cannot be audited." → skip |
---
**End of /audit-commands Command**
