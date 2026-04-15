---
version: "v0.87.0"
description: Hybrid command+rule parallel evaluation of /work (project)
argument-hint: "#issue [#issue...] [--assign] [--nonstop] [--wait] | all in <status>"
copyright: "Rubrical Works (c) 2026"
---

<!-- MANAGED -->
# /workit
Hybrid command+rule parallel evaluation of `/work`. This shell is minimal; the execution logic lives in `.claude/rules/08-workit-execution.md` and auto-loads once per session. Ships alongside `/work` for token-footprint measurement (#2329).

---
## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue assigned to a branch (use `/assign-branch` first, or pass `--assign` to auto-assign)

---
## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes (one of) | Single issue number (e.g., `#42` or `42`) |
| `#issue #issue...` | | Multiple issue numbers (e.g., `#42 #43 #44`) |
| `all in <status>` | | All issues in given status (e.g., `all in backlog`) |
| `--assign` | No | Assign issue(s) to current branch before starting work |
| `--nonstop` | No | Epic/branch tracker: skip per-sub-issue STOP, process all to `in_review` continuously |
| `--wait` | No | Wait for pending CI to pass before starting work |

---
## Execution Logic

Follow `.claude/rules/08-workit-execution.md` (auto-loaded at session start). That rule defines:

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
**End of /workit Command**
