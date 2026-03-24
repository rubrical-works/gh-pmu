---
name: ci-status-check
description: Quick CI status without blocking workflow
extensionPoints:
  - post-analysis
  - post-prepare
appliesTo:
  - prepare-release:post-analysis
  - prepare-beta:post-analysis
  - prepare-release:post-prepare
  - prepare-beta:post-prepare
prerequisites:
  - gh CLI (GitHub CLI)
  - "GitHub Actions workflows configured (run /ci to check status)"
---

### Check CI Status

```bash
gh run list --limit 1 --json status,conclusion,name
```

Report status but do not block workflow.
