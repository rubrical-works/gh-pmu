---
version: "v0.84.0"
description: Generate session statistics report with development velocity metrics
argument-hint: "[--today] [--date YYYY-MM-DD] [--since YYYY-MM-DD] [--until YYYY-MM-DD] [--repos /path/a,/path/b] [--repos-edit] [--save]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /idpf-stats
Generate session statistics by analyzing git history, GitHub issues, and test counts. Produces ASCII tables: volume, testing, throughput, issue categorization.
**Scope:** Framework development and user projects — deployed to users.
## Prerequisites
- Git repository initialized
- `gh` CLI installed (for issue breakdown)
- `.gh-pmu.json` configured (optional — issue breakdown skipped if missing)
## Arguments
| Argument | Required | Default | Description |
|----------|----------|---------|-------------|
| `--today` | No | — | Explicit alias for default. Sets `since`=midnight local today, `until`=now. Cannot combine with `--since`/`--until`/`--date`. |
| `--date` | No | — | Shortcut for a specific full day `YYYY-MM-DD`. Sets `since`=`T00:00:00`, `until`=`T23:59:59`. Deterministic. Cannot combine with `--since`/`--until`/`--today`. |
| `--since` | No | Today (midnight) | Start date `YYYY-MM-DD` |
| `--until` | No | Now | End date `YYYY-MM-DD` |
| `--repos` | No | — | Comma-separated directories. No value = cached list. |
| `--repos-edit` | No | — | Present cached directory list for interactive add/remove |
| `--save` | No | — | Save report to `idpf-stats/YYYY-MM-DD.md` |
**Examples:**
- `/idpf-stats` — Today's session (single repo)
- `/idpf-stats --today` — Same as above, explicit/self-documenting
- `/idpf-stats --date 2026-04-06` — Specific full day (deterministic)
- `/idpf-stats --since 2026-03-15` — Stats since March 15
- `/idpf-stats --repos /path/a,/path/b` — Multi-repo stats
- `/idpf-stats --repos` — Multi-repo using cached list
- `/idpf-stats --repos-edit` — Edit cached directory list
- `/idpf-stats --save` — Display and save report to `idpf-stats/`
### How the Work Day is Calculated
`/idpf-stats` treats the "work day" as a time range anchored to local timezone, not UTC. Four modes:
| Mode | `since` | `until` | Notes |
|---|---|---|---|
| **Default** (no flags) | Midnight local today (00:00:00) | "now" | Range grows as you re-run later in the day. |
| **`--today`** | Midnight local today (00:00:00) | "now" | Identical to default. Explicit/self-documenting. |
| **`--date YYYY-MM-DD`** | `T00:00:00` local | `T23:59:59` local | Deterministic — same output regardless of when run. |
| **`--since` / `--until`** | `T00:00:00` of given date | `T23:59:59` of given date (or "now" if omitted) | Multi-day ranges. |
**Timezone:** Dates parsed/formatted in local timezone via `getTzOffset()`. A commit at 11:55 PM local on 2026-04-06 appears in `--date 2026-04-06`, not the next day.
**Midnight boundaries:** `--date` uses `00:00:00`–`23:59:59`, so a commit exactly at midnight is counted in the day it starts. Default/`--today` use "now", so future-dated commits (clock skew) are excluded.
**`--today` vs `--date <today>`:** `--today` at 3pm vs 6pm shows different counts (range grew). `--date 2026-04-06` is deterministic but misses commits after the run. Snapshot of "today so far" → `--today`. Full historical day → `--date` after the day ends.
## Workflow
### Step 1: Collect Metrics
Run the stats collection script:
```bash
node .claude/scripts/shared/stats-collect.js [--since YYYY-MM-DD] [--until YYYY-MM-DD] [--repos /path/a,/path/b] [--repos-edit]
```
Parse JSON output. The script handles:
- Timezone-aware date parsing (local timezone, never UTC)
- Git command execution for all metrics
- Multi-repo collection and aggregation (when `--repos` present)
- Directory validation (non-git dirs produce warnings, skipped)
- Directory list caching in `.claude/metadata/idpf-stats-repos.json`
- Edge cases (no commits, no issues, no test files, single-commit throughput)
**`--repos` behavior:**
- `--repos /path/a,/path/b` — collect from specified dirs, cache list
- `--repos` (no value) — reuse cached list
- `--repos-edit` — present cached list via `AskUserQuestion` for add/remove, then collect
**Auto-detection:** If no `--repos` flag but `idpf-stats/repos.json` exists at project root, auto-load repo list and run multi-repo mode.
**Multi-repo output:** Returns `{ aggregate, perRepo, warnings }` instead of flat result. `aggregate` same shape as single-repo. `perRepo` is per-directory array. `warnings` lists skipped non-git dirs.
**If script fails:** Report error and STOP.
**If no commits:** Script returns `volume.commits: 0` (or `aggregate.volume.commits: 0`) — skip to Step 4.
### Step 2: Gather Issue Categories
**Requires:** `gh` CLI. Verify with `gh auth status`.
**Single-repo:** Issues are `["#42", "#43"]` strings:
```bash
gh issue view $N --json labels --jq '.labels[].name' 2>/dev/null
```
**Multi-repo:** Issues are `[{ "number": 42, "repo": "owner/repo" }]`:
```bash
gh issue view $N --repo $REPO --json labels --jq '.labels[].name' 2>/dev/null
```
Categorize using `labelCategories` from `.claude/metadata/stats-config.json`:
| Label | Category |
|-------|----------|
| `bug` | Bug fixes |
| `enhancement` | Enhancements |
| `security` | Security hardening |
| `code-review`, `reviewed` | Code review findings |
| `infrastructure`, `ci`, `devops` | Infrastructure |
| `documentation`, `docs` | Documentation |
| (no matching label) | Other |
- No issues referenced: skip issue breakdown table
- `gh` not installed: skip issue metrics entirely, report "Issue breakdown unavailable (gh CLI not found)."
- `.gh-pmu.json` NOT required — `gh` uses global auth, `--repo owner/name` works for any accessible repo
### Step 3: Render Output
Display report header and tables using ASCII box-drawing characters.
**Table rules:**
- Unicode box-drawing: `┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼ │ ─`
- Column widths auto-sized, header separated by `├──┼──┤`
- Numbers right-aligned, text left-aligned, large numbers with commas
- Positive deltas `+`, negative `-`
**Title:** From `displayRange`: single day → `"Today's Session Stats ({since})"`, range → `"Session Stats ({since} to {until})"`
**Output structure:**
```
{Title} ({Date Range})

Volume
┌──────────────────────┬──────────────────┐
│        Metric        │      Value       │
├──────────────────────┼──────────────────┤
│ Commits              │ {volume.commits} │
│ Issues closed/worked │ {issues.length}  │
│ Files changed        │ {volume.filesChanged} │
│ Lines added          │ +{volume.linesAdded}  │
│   Code               │ +{byType.Code.added}  │
│   Documentation      │ +{byType.Documentation.added} │
│   Config             │ +{byType.Config.added}│
│   Other              │ +{byType.Other.added} │
│ Lines removed        │ -{volume.linesRemoved}│
│ Net lines            │ +/-{volume.netLines}  │
└──────────────────────┴──────────────────┘

Testing
┌────────────────────┬───────┐
│       Metric       │ Value │
├────────────────────┼───────┤
│ Tests before today │ {testing.testsBefore} │
│ Tests now          │ {testing.testCases}   │
│ New tests added    │ +{testing.newTestFiles}│
│ Test files         │ {testing.testFiles}   │
└────────────────────┴───────┘

Throughput
┌──────────────────┬───────┐
│      Metric      │ Value │
├──────────────────┼───────┤
│ Commits/hour     │ ~{throughput.commitsPerHour}  │
│ Lines added/hour │ ~{throughput.linesPerHour}    │
│ Issues/hour      │ ~{throughput.issuesPerHour}   │
└──────────────────┴───────┘

Issue Breakdown by Category
┌──────────────────────┬───────┬──────────────────────────────┐
│       Category       │ Count │            Issues            │
├──────────────────────┼───────┼──────────────────────────────┤
│ {category}           │ {N}   │ {#N, #N, ...}                │
└──────────────────────┴───────┴──────────────────────────────┘
```
**Line-type sub-rows:** Indented under "Lines added" for each `byType` category where `added > 0`. Use `extensionCategories` from `stats-config.json`.
**Code-to-documentation ratio:** After Volume table, if both non-zero: `"Code:Docs ratio — {N}:1"` with assessment.
**Conditional sections:**
- **Throughput:** Only if `throughput` not null
- **Issue Breakdown:** Only if issues non-empty and `gh` available
- **Testing:** Always (zeros if no test files)
- **Velocity Assessment:** Only if `throughput` not null
### Step 3a: Multi-Repo Rendering
When output contains `perRepo`:
1. Render aggregate tables using `aggregate` data (same format as above)
2. Add per-repo breakdown:
```
Per-Repository Breakdown
┌──────────────────────┬─────────┬───────────┬─────────┬──────────┐
│      Repository      │ Commits │ Files     │ +Lines  │ -Lines   │
├──────────────────────┼─────────┼───────────┼─────────┼──────────┤
│ /path/to/repo-a      │ {N}     │ {N}       │ +{N}    │ -{N}     │
│ /path/to/repo-b      │ {N}     │ {N}       │ +{N}    │ -{N}     │
├──────────────────────┼─────────┼───────────┼─────────┼──────────┤
│ Total                │ {N}     │ {N}       │ +{N}    │ -{N}     │
└──────────────────────┴─────────┴───────────┴─────────┴──────────┘
```
3. Report warnings for skipped non-git directories
### Step 3b: Velocity Assessment
**Trigger:** `throughput` not null.
Use `assessVelocity()` from `stats-collect.js` with `velocityBenchmarks` from `stats-config.json`:
```
Velocity Assessment
┌──────────────────┬────────┬────────────┐
│      Metric      │  Rate  │  Rating    │
├──────────────────┼────────┼────────────┤
│ Commits/hour     │ ~{N}   │ 🟢/🟡/🔴  │
│ Lines added/hour │ ~{N}   │ 🟢/🟡/🔴  │
│ Issues/hour      │ ~{N}   │ 🟢/🟡/🔴  │
└──────────────────┴────────┴────────────┘
Estimated throughput: ~{multiplier}x typical developer velocity
Assumptions: median human benchmarks — {commits median} commits/hr, {lines median} lines/hr,
{issues median} issues/hr (configurable in stats-config.json velocityBenchmarks)
```
Rating: 🟢 above `high`, 🟡 between `low`/`high`, 🔴 below `low`.
**Assumptions disclosure mandatory.** Show benchmark medians and source. Custom benchmarks: note "custom benchmarks".
### Step 3b-ii: File Type Breakout
**Trigger:** `byExtension` from `parseNumstat` has entries.
Render after all other tables as detail appendix:
```
File Type Breakout
┌───────────┬────────┬────────┐
│ Extension │ +Lines │ -Lines │
├───────────┼────────┼────────┤
│ .md       │ +8,200 │ -1,100 │
│ .js       │ +5,800 │   -900 │
│ .test.js  │ +2,300 │    -50 │
│ .json     │ +1,100 │   -200 │
└───────────┴────────┴────────┘
```
**Rules:**
- Sorted by lines added (descending) — pre-sorted by `parseNumstat`
- Omit extensions with zero added and zero removed
- `.test.js`/`.spec.js` separate from `.js` (handled by `getExtension()`)
- Multi-repo: show aggregate `byExtension`
### Step 3c: Save Report (--save)
**Trigger:** `--save` flag present.
1. Create `idpf-stats/` directory if needed
2. Filename: single day `YYYY-MM-DD.md`, range `YYYY-MM-DD--YYYY-MM-DD.md`
3. Write markdown report with:
   - Emoji headers: 📊 Volume, 🧪 Testing, ⚡ Throughput, 📁 Issue Breakdown
   - Markdown tables, 🟢🟡🔴 velocity indicators
   - Velocity Assessment with multiplier and benchmark assumptions
   - Code-to-documentation ratio, File Type Breakout as final section
4. Report: `"Saved to {filepath}"`
**Note:** `idpf-stats/` is project-local, never deployed. Should be in `.gitignore` for user projects, tracked in framework dev repos.
### Step 4: Edge Case — Empty Report
If `volume.commits` is 0:
```
{Title} ({Date Range})

No activity found in the specified time range.
```
Do not render empty tables.
## Error Handling
| Condition | Behavior |
|-----------|----------|
| Not a git repo | Script returns error → STOP |
| Invalid date format | "Invalid date format. Use YYYY-MM-DD." STOP |
| `--until` before `--since` | "End date must be after start date." STOP |
| No commits in range | "No activity found" |
| `gh` unavailable | Skip issue breakdown |
| No test files | Show zeros |
| Script fails | Report failure, continue with available data |
**End of /idpf-stats Command**
