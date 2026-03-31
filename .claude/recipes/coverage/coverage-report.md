---
name: coverage-report
description: Generate coverage report without blocking
extensionPoints:
  - post-validation
  - post-prepare
appliesTo:
  - prepare-release:post-validation
  - prepare-beta:post-validation
  - prepare-release:post-prepare
  - prepare-beta:post-prepare
prerequisites:
  - Coverage tool configured
  - (Optional) Codecov or similar service
  - "CI integration: /ci add coverage-upload"
---

### Coverage Report

Generate coverage report for review:

```bash
# Example: Generate HTML report
nyc report --reporter=html --report-dir=coverage

# Example: Upload to Codecov
bash <(curl -s https://codecov.io/bash)
```

Report location: `coverage/index.html`
