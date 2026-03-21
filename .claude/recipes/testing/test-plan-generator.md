---
name: test-plan-generator
description: Generate test plan skeleton from branch issues
extensionPoints:
  - pre-tag
appliesTo:
  - prepare-release:pre-tag
  - prepare-beta:pre-tag
prerequisites:
  - generate-test-plan.js (included with framework)
  - gh pmu extension installed
  - Issues assigned to branch
---

### Generate Test Plan

```bash
node .claude/scripts/shared/generate-test-plan.js
```

Creates `Construction/Test-Plans/{version}-test-plan.md` with:
- Test cases derived from branch issues
- Acceptance criteria extracted as expected results
- Test execution tracking table

**Review the generated test plan and complete TODO items.**
