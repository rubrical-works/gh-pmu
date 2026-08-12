---
version: "v0.96.1"
allowed-tools: Bash, AskUserQuestion
description: "Assign or remove issues from a branch: [#issue...] [branch/...] [--add-ready] [--remove] (project)"
argument-hint: "[#issue...] [branch/name] [--add-ready] [--remove]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
Assign issues to a branch.
```bash
node .claude/scripts/shared/assign-branch.js "$ARGUMENTS"
```
## Handling "NO_BRANCH_FOUND" Output
If the script outputs `NO_BRANCH_FOUND`, no open branches exist. It also outputs:
1. **CONTEXT:** — last version, issue labels, user input
2. **SUGGESTIONS:** — formatted `number|branch|description`

On this output:
1. Parse SUGGESTIONS lines for branch options
2. Use `AskUserQuestion` — present `(recommended)` first, include descriptions, "Other" for custom name
3. After selection, create the branch:
   ```bash
   gh pmu branch start --name "<selected-branch>"
   ```
4. Re-run the original assign-branch command
## Example Flow
```
NO_BRANCH_FOUND
SUGGESTIONS:
1|patch/v0.15.1|Next patch version (bug fixes only) (recommended)
2|release/v0.16.0|Next minor version (new features)
```
1. AskUserQuestion with: "patch/v0.15.1 (Recommended)", "release/v0.16.0"
2. User selects → `gh pmu branch start --name "patch/v0.15.1"`
3. Re-run original command
## Normal Output
If branches exist, report the result directly.
## Errors
| Error | Cause | Resolution |
|-------|-------|------------|
| `Issue #N is a branch tracker and cannot be assigned.` | Target carries the `branch` label (branch tracker, not implementation work). Script exits non-zero before any mutation. | Target the sub-issues on the branch instead, or use `gh pmu sub list` to enumerate them. |
| `"X" is not an open branch. No issues were assigned.` | Slash-bearing token read as a branch name matches no open tracker. Script exits non-zero before any mutation. | Re-run with a listed open branch, or omit the branch to use the current one. |

**Why a slash command can be misread as a branch (#2554):** branch args are identified by shape — non-flag, non-numeric, contains a slash — so a token like `/work` in trailing chat text is picked up as the branch name. It is rejected because it matches no open tracker; nothing is written. Re-invoke with only the issue numbers.
