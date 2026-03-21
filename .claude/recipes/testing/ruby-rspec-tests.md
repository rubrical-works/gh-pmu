---
name: ruby-rspec-tests
description: Run Ruby tests with RSpec
extensionPoints:
  - pre-validation
  - pre-gate
appliesTo:
  - prepare-release:pre-validation
  - prepare-beta:pre-validation
  - merge-branch:pre-gate
prerequisites:
  - Gemfile with rspec
---

### Run Tests

```bash
bundle exec rspec
```

**If tests fail, STOP and fix before continuing.**
