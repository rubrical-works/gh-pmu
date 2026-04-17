---
version: "v0.88.0"
description: Start working on issues with validation and auto-task extraction (project)
argument-hint: "#issue [#issue...] [--assign] [--nonstop] [--wait] | all in <status>"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /work
`/work` command spec (hybrid shell + auto-loaded execution rule). Execution logic lives in `.claude/rules/08-work-execution.md` and auto-loads once per session. Consolidated from the `/workit` parallel-evaluation prototype (#2329, retired in #2368).

---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue assigned to a branch (use `/assign-branch` first, or pass `--assign`)

---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes (one of) | Single issue number (`#42` or `42`) |
| `#issue #issue...` | | Multiple issue numbers (`#42 #43 #44`) |
| `all in <status>` | | All issues in given status (`all in backlog`) |
| `--assign` | No | Assign issue(s) to current branch before starting work |
| `--nonstop` | No | Epic/branch tracker: skip per-sub-issue STOP, process all to `in_review` continuously |
| `--wait` | No | Wait for pending CI to pass before starting work |

---
## Execution Logic

Follow `.claude/rules/08-work-execution.md` (auto-loaded at session start). That rule defines:
- Phase 1 / Phase 2 task creation
- Step 1: preamble via `work-preamble.js`
- Steps 1a, 1b, 2, 2a, 3, 3b, 4, 4a, 4b, 4c, 5, 6, 6a
- Pre-Agent / Sub-Agent / Commit-per-AC gates
- Autonomous epic & branch tracker processing (default vs. `--nonstop`)
- STOP boundary and post-STOP cleanup

---
## Error Handling
**STOP errors:** Issue not found, no branch assignment, `gh pmu` failure, `ALREADY_ASSIGNED` (different branch), `WORKSTREAM_CONFLICT` (use `/assign-branch`).
**Non-blocking:** PRD tracker not found, framework file missing, no acceptance criteria, issue already in_progress.

---
**End of /work Command**
