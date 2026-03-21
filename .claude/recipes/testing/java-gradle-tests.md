---
name: java-gradle-tests
description: Run Java tests with Gradle
extensionPoints:
  - pre-validation
  - pre-gate
appliesTo:
  - prepare-release:pre-validation
  - prepare-beta:pre-validation
  - merge-branch:pre-gate
prerequisites:
  - build.gradle
---

### Run Tests

```bash
gradle test
```

**If tests fail, STOP and fix before continuing.**
