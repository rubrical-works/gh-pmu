---
version: "v0.72.0"
allowed-tools: Bash, AskUserQuestion
description: "Assign or remove issues from a branch: [#issue...] [branch/...] [--add-ready] [--remove] (project)"
argument-hint: "[#issue...] [branch/name] [--add-ready] [--remove]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
Assign issues to a branch.
Run the assign-branch script:
```bash
node .claude/scripts/shared/assign-branch.js "$ARGUMENTS"
```
## Handling "NO_BRANCH_FOUND" Output
If the script outputs `NO_BRANCH_FOUND`:
1. Parse the SUGGESTIONS lines (`number|branch|description`)
2. Use `AskUserQuestion` to let user select a branch (recommended first, include "Other" for custom)
3. Create branch: `gh pmu branch start --name "<selected-branch>"`
4. Re-run the original assign-branch command
## Normal Output
If branches exist, report the result directly.
