---
version: "v0.62.1"
allowed-tools: Bash
description: Switch branch context (project)
argument-hint: "[branch-name]"
---
<!-- MANAGED -->
Switch between branch contexts.
Run the switch-branch script:
```bash
node .claude/scripts/shared/switch-branch.js "$ARGUMENTS"
```
After switching, report the new context to the user.
