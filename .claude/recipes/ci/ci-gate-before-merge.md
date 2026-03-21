---
name: ci-gate-before-merge
description: Block merge/tag until CI passes
extensionPoints:
  - post-pr-create
  - pre-tag
  - post-prepare
appliesTo:
  - prepare-release:post-pr-create
  - prepare-release:pre-tag
  - prepare-beta:post-prepare
  - prepare-beta:pre-tag
  - merge-branch:post-pr-create
prerequisites:
  - wait-for-ci.js (included with framework)
  - "GitHub Actions workflows configured (run /ci to check status)"
---

### Wait for CI

```bash
node .claude/scripts/shared/wait-for-ci.js
```

**If CI fails, STOP and report the error.**

Common failures:
- Build errors: Check compilation output
- Test failures: Run tests locally to debug
- Lint errors: Run linter with --fix
