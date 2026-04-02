---
version: "v0.77.4"
description: Generate session statistics report with development velocity metrics
argument-hint: "[--since YYYY-MM-DD] [--until YYYY-MM-DD]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /idpf-stats
Generate session statistics by analyzing git history, GitHub issues, and test counts. Produces ASCII tables: volume, testing, throughput, issue categorization.
---
## Prerequisites
- Git repository initialized
- `gh` CLI installed (for issue breakdown)
- `.gh-pmu.json` configured (optional)
---
## Arguments
| Argument | Required | Default | Description |
|----------|----------|---------|-------------|
| `--since` | No | Today | Start date `YYYY-MM-DD` |
| `--until` | No | Now | End date `YYYY-MM-DD` |
---
## Workflow
### Step 1: Determine Time Range
Default: today since midnight. Compute display label (single day vs range).
### Step 2: Gather Git Metrics
```bash
git log --after="$since_date" --until="$until_date" --oneline | wc -l
git log --after="$since_date" --until="$until_date" --pretty=format: --name-only | sort -u | grep -v '^$' | wc -l
git log --after="$since_date" --until="$until_date" --pretty=format: --numstat | awk '{ added += $1; removed += $2 } END { print added, removed }'
git log --after="$since_date" --until="$until_date" --format="%aI" | tail -1
git log --after="$since_date" --until="$until_date" --format="%aI" | head -1
```
No commits: set all to 0, skip throughput.
### Step 3: Gather Issue Metrics
Extract issue numbers from commit messages:
```bash
git log --after="$since_date" --until="$until_date" --pretty=format:"%s %b" | grep -oE '#[0-9]+' | sort -u
```
Query labels per issue. Categorize:
| Label | Category |
|-------|----------|
| `bug` | Bug fixes |
| `enhancement` | Enhancements |
| `security` | Security hardening |
| `code-review`, `reviewed` | Code review |
| `infrastructure`, `ci`, `devops` | Infrastructure |
| `documentation`, `docs` | Documentation |
| (none) | Other |
No issues: skip table. No `gh`/config: skip entirely.
### Step 4: Gather Test Metrics
Count test files and cases. Derive "tests before" from new test files added in period.
### Step 5: Compute Throughput
Commits/hour, lines added/hour, issues/hour. Min 1 hour denominator. Prefix `~` for approximation.
### Step 6: Render Output
Unicode box-drawing tables. Numbers right-aligned, commas for large numbers. Tables: Volume, Testing, Throughput, Issue Breakdown by Category.
### Step 7: Edge Case -- Empty Report
No activity: `"No activity found in the specified time range."` No empty tables.
---
## Error Handling
| Condition | Behavior |
|-----------|----------|
| Not a git repo | STOP |
| Invalid date format | STOP |
| `--until` before `--since` | STOP |
| No commits | "No activity found" |
| `gh` unavailable | Skip issue breakdown |
| No test files | Show zeros |
| Git command fails | Report and continue |
---
**End of /idpf-stats Command**
