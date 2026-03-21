---
name: auto-changelog
description: Generate changelog from commits
extensionPoints:
  - post-analysis
  - pre-commit
appliesTo:
  - prepare-release:pre-commit
  - prepare-beta:pre-commit
prerequisites:
  - generate-changelog.js (included with framework)
  - Conventional commit messages
---

### Generate Changelog

```bash
node .claude/scripts/shared/generate-changelog.js
```

Updates `CHANGELOG.md` with commits since last tag, grouped by type (feat, fix, etc.).
