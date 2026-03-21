---
name: go-tests
description: Run Go tests with go test
extensionPoints:
  - pre-validation
  - pre-gate
appliesTo:
  - prepare-release:pre-validation
  - prepare-beta:pre-validation
  - merge-branch:pre-gate
prerequisites:
  - go.mod
---

### Run Tests

```bash
go test ./...
```

**If tests fail, STOP and fix before continuing.**
