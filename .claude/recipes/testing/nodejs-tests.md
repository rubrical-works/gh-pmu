---
name: nodejs-tests
description: Run Node.js tests with npm
extensionPoints:
  - pre-validation
  - pre-gate
appliesTo:
  - prepare-release:pre-validation
  - prepare-beta:pre-validation
  - merge-branch:pre-gate
prerequisites:
  - package.json with test script
---

### Run Tests

```bash
npm test
```

**If tests fail, STOP and fix before continuing.**
