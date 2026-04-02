---
version: "v0.79.0"
allowed-tools: Bash
description: Switch branch context (project)
argument-hint: "[branch-name]"
copyright: "Rubrical Works (c) 2026"
---

<!-- MANAGED -->
Switch between branch contexts.

Run the switch-branch script:

```bash
node .claude/scripts/shared/switch-branch.js "$ARGUMENTS"
```

After switching, report the new context to the user.
