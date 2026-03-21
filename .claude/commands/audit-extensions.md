---
version: "v0.67.2"
description: Audit extension point content for quality and consistency (project)
argument-hint: "[all|<command-name>|<group description>]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /audit-extensions

Audit extension point content within project commands. Evaluates whether `USER-EXTENSION-START/END` blocks are populated, well-structured, and consistent with the command's workflow. Creates one enhancement issue per command with recommendations.

**Skill Dependency:** Loads `command-spec-audit` skill for extension-specific evaluation criteria (Category 4).

---

## Prerequisites

- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root

---

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `all` | No | Audit every command with extension points |
| `<command-name>` | No | Audit a single command's extensions |
| `<group description>` | No | NL grouping (e.g., "release commands") |

If no argument provided, prompt the user for scope.

---

## Execution Instructions

**REQUIRED:** Before executing:

1. **Load Skill:** Read `Skills/command-spec-audit/SKILL.md` — focus on **Category 4: Extension Points** criteria
2. **Generate Todo List:** Use `TodoWrite` to create todos from workflow steps
3. **Track Progress:** Mark todos `in_progress` -> `completed` as you work

---

## Workflow

### Step 1: Resolve Scope

Same NL scope resolution as `/audit-commands`:

| Input | Resolution |
|-------|-----------|
| `all` | All `.md` files in `.claude/commands/` with extension point markers |
| `<command-name>` | `.claude/commands/<name>.md` |
| `<group description>` | Match by NL against frontmatter descriptions |

**Filter:** Only include commands that have `USER-EXTENSION-START` markers.

### Step 2: Discover Extension Points

For each in-scope command:

1. **Scan** for `<!-- USER-EXTENSION-START: {point} -->` and `<!-- USER-EXTENSION-END: {point} -->` markers
2. **Extract** content between each START/END pair
3. **Classify** each extension as:
   - **Empty** — no content between markers (or only whitespace/comments)
   - **Populated** — has substantive content

Report discovery: `"Found N extension points (M populated, K empty) in <command>"`

### Step 3: Evaluate Extensions

Apply **Category 4: Extension Points** criteria from the `command-spec-audit` skill:

**For empty extensions:**
- Check against high-value extension patterns (e.g., `pre-commit`, `post-implementation`, `pre-create`)
- Recommend common patterns from `/extensions recipes` if applicable

**For populated extensions:**
- **Formatting** — content matches parent command's formatting standards
- **Conflicts** — content duplicates or contradicts built-in step behavior
- **Ordering** — content makes sense relative to workflow position
- **Size** — flag extensions >50 lines that should be scripts
- **Consistency** — similar patterns across commands use same format

Record findings with: criterion, severity, extension point name, finding, recommendation.

### Step 4: Report or Create Issue

**If zero findings:** Report `"No recommendations: <command-name>"` and skip.

**If findings exist:** Create one enhancement issue per command:

```bash
gh pmu create --title "[Audit]: <command-name> extensions — N findings" --label enhancement --status backlog -F .tmp-body.md
rm .tmp-body.md
```

Issue body format:

```markdown
## Extension Audit: <command-name>

**Source:** `.claude/commands/<command-name>.md`
**Audited:** YYYY-MM-DD
**Rubric:** `command-spec-audit` skill (Category 4)

### Extension Points

| Point | Status | Findings |
|-------|--------|----------|
| pre-work | Empty | High-value: recommend pre-validation pattern |
| post-implementation | Populated | 50+ lines — refactor to script |

### Detailed Findings

| # | Criterion | Severity | Point | Finding | Recommendation |
|---|-----------|----------|-------|---------|----------------|
| 1 | ... | ... | ... | ... | ... |

**Summary:** N High, N Medium, N Low
```

### Step 5: Summary Report

After all commands audited:

```
Extension Audit Complete
  Commands scanned: N
  Extension points found: N (M populated, K empty)
  Commands with findings: N
  Issues created: N
  Severity breakdown: N High, N Medium, N Low
```

**STOP.** Do not implement fixes — this is an audit-only command.

---

## Error Handling

| Situation | Response |
|-----------|----------|
| No commands with extensions | "No commands with extension points found." -> STOP |
| Command file unreadable | "Cannot read: <path>." -> skip, continue |
| Skill not loaded | "Warning: command-spec-audit skill not found. Using inline criteria." -> continue |
| Mismatched markers | Report as High severity finding in the command's issue |
| `gh pmu create` fails | "Failed to create issue: {error}" -> report, continue |

---

**End of /audit-extensions Command**
