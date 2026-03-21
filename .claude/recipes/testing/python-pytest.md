---
name: python-pytest
description: Run Python tests with pytest
extensionPoints:
  - pre-validation
  - pre-gate
appliesTo:
  - prepare-release:pre-validation
  - prepare-beta:pre-validation
  - merge-branch:pre-gate
prerequisites:
  - pytest installed
---

### Run Tests

```bash
pytest
```

**If tests fail, STOP and fix before continuing.**
