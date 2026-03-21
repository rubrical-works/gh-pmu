---
name: php-phpunit-tests
description: Run PHP tests with PHPUnit
extensionPoints:
  - pre-validation
  - pre-gate
appliesTo:
  - prepare-release:pre-validation
  - prepare-beta:pre-validation
  - merge-branch:pre-gate
prerequisites:
  - composer.json with phpunit
---

### Run Tests

```bash
composer test
```

**If tests fail, STOP and fix before continuing.**
