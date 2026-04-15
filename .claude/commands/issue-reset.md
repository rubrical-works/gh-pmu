---
version: "v0.86.0"
description: Reset bug/enhancement/prd/proposal/epic issue to clean slate (project)
argument-hint: "#issue [--dry-run]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /issue-reset
Reset a bug/enhancement/PRD/proposal/epic issue to clean slate. Removes reviews, resolutions, test plans, downstream files. Epics recursively reset sub-issues.
---
## Prerequisites
- `gh pmu` installed
- `.gh-pmu.json` configured
---
## Arguments
| Argument | Description |
|----------|-------------|
| `#issue` | Issue number (e.g., `#42`) |
| `--dry-run` | Preview without executing |
---
## Workflow
### Step 1: Validate Type
```bash
gh pmu view $ISSUE --json=number,title,labels,body,state
```
**Allowed:** `bug`, `enhancement`, `prd`, `proposal`, `epic`
**Rejected:** `story`, `branch`, others

**Rejected:**
```
Error: /issue-reset only applies to bug, enhancement, prd, proposal, and epic issues.
Issue #$ISSUE has label "$LABEL" which is not supported.
```
→ **STOP**
### Step 1a: Epic Detection
**If `epic` label:**
1. ```bash
   gh pmu sub list $ISSUE
   ```
2. Record sub-issue list for recursive Step 4
### Step 2: Analyze Scope
| Item | Detection |
|------|-----------|
| AC checkboxes | Count `[x]` in body |
| Reviews counter | `**Reviews:** N` |
| Auto-generated sections | `#### Proposed Solution`, `#### Proposed Fix` |
| Review comments | `## Issue Review`, `## Proposal Review`, `## PRD Review` |
| `reviewed` label | Labels array |
| Test plan files | `Construction/Test-Plans/` referencing `#$ISSUE` |

**Proposal additionally:**
| Item | Detection |
|------|-----------|
| Associated PRD | Body `PRD: #N` or `PRD/{name}/PRD-{name}.md` |
| PRD → backlog | PRD has sub-issues `in_progress`/`in_review`/`done` |

**Epic additionally:**
| Item | Detection |
|------|-----------|
| Sub-issues | From Step 1a |
| Sub-issue scope | Per sub-issue (same as above) |
### Step 3: Dry-Run or Confirmation
**`--dry-run`:**
```
Dry run — /issue-reset #$ISSUE

Would reset:
  - Uncheck $N AC checkboxes
  - Reset Reviews counter (currently $N)
  - Remove $N auto-generated sections
  - Delete $N review comments
  - Remove 'reviewed' label: $YES_NO
  - Delete $N test plan files
  [proposal only]
  - Delete PRD issue #$PRD_NUM: $YES_NO
  - Delete PRD files: $FILE_LIST
  [epic only]
  - Reset $M sub-issues recursively

No changes made.
```
→ **STOP**

**Not dry-run:** `AskUserQuestion`:
```
This will reset issue #$ISSUE ($TITLE):
  - Uncheck $N AC checkboxes
  - Reset Reviews counter
  - Delete $N review comments
  - Remove 'reviewed' label
  - Delete $N test plan files
  [proposal-specific items if applicable]
  [epic: "Reset $M sub-issues recursively"]

Proceed with reset?
```
**Declines:** → **STOP**
### Step 4: Execute
**4a: Reset body**
```bash
gh pmu view $ISSUE --body-stdout > .tmp-$ISSUE.md
```
- `[x]` → `[ ]`
- `**Reviews:** N` → `**Reviews:** 0`
- Remove `#### Proposed Solution` (if auto-generated)
- Remove `#### Proposed Fix` (if auto-generated)
```bash
gh pmu edit $ISSUE -F .tmp-$ISSUE.md && rm .tmp-$ISSUE.md
```
**4b: Move to backlog**
```bash
gh pmu move $ISSUE --status backlog
```
**4c: Delete review comments**
```bash
gh api repos/{owner}/{repo}/issues/$ISSUE/comments --paginate
```
For each matching `## Issue Review`, `## Proposal Review`, `## PRD Review`:
```bash
gh api -X DELETE repos/{owner}/{repo}/issues/comments/$COMMENT_ID
```
**4d: Remove reviewed label**
```bash
gh issue edit $ISSUE --remove-label reviewed
```
(Silent skip if absent)
**4e: Delete test plans**
For each file in `Construction/Test-Plans/` referencing `#$ISSUE`:
```bash
git rm "$FILE"
```
**4f: Proposal — PRD cleanup**
When a proposal has been converted to backlog, clean up the PRD tracker.
If `proposal` label + associated PRD:
- **PRD NOT converted:** Delete PRD issue and files
  ```bash
  gh issue close $PRD_ISSUE --comment "Deleted by /issue-reset #$ISSUE"
  gh pmu move $PRD_ISSUE --status done
  git rm -r "PRD/{name}/"
  ```
- **PRD converted:** Warn, ask user:
  ```
  Warning: PRD #$PRD_NUM has $N backlog issues in progress.
  Deleting the PRD will orphan these issues.

  Options:
  1. Keep PRD (only reset the proposal)
  2. Delete PRD and orphan backlog issues
  3. Cancel reset
  ```
**4g: Epic — Recursive**
If `epic` label: For each sub-issue, apply 4a–4e. Report:
```
Resetting sub-issue #N: $TITLE... done
```
After all sub-issues, apply 4a–4e to the epic itself.
### Step 5: Commit
If files deleted:
```bash
git commit -m "Refs #$ISSUE — reset issue artifacts (test plans, PRD files)"
```
### Step 6: Post Audit Comment
```bash
gh issue comment $ISSUE --body "Issue reset by /issue-reset on $DATE. Removed: $N review comments, $M test plan files. Status: backlog."
```
### Step 7: Report
```
Reset complete: Issue #$ISSUE — $TITLE

Actions taken:
  ✓ AC checkboxes unchecked ($N)
  ✓ Reviews counter reset to 0
  ✓ Auto-generated sections removed ($N)
  ✓ Review comments deleted ($N)
  ✓ 'reviewed' label removed
  ✓ Test plan files deleted ($M)
  ✓ Status: backlog
  ✓ Audit comment posted
  [epic only]
  ✓ Sub-issues reset: $M

Issue is ready for a fresh start.
```
**STOP.**
---
## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." → STOP |
| Invalid type | "Only bug, enhancement, prd, proposal, epic issues can be reset." → STOP |
| User declines | "Reset cancelled." → STOP |
| No review comments | Skip, report "0 review comments" |
| No test plan files | Skip, report "0 test plan files" |
| `git rm` fails | Report error, continue |
| PRD has backlog | Warn and ask |
| Epic has no sub-issues | Reset epic only, report "0 sub-issues" |
---
**End of /issue-reset Command**
