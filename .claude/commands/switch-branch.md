---
version: "v0.91.0"
allowed-tools: Bash
description: Switch branch context (project)
argument-hint: "[branch-name]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
Switch between branch contexts.
```bash
node .claude/scripts/shared/switch-branch.js "$ARGUMENTS"
```
After switching, report the new context to the user.
