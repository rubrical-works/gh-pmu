---
name: coverage-gate
description: Block release if coverage drops below threshold
extensionPoints:
  - pre-validation
  - post-validation
appliesTo:
  - prepare-release:pre-validation
  - prepare-release:post-validation
  - prepare-beta:pre-validation
prerequisites:
  - Coverage tool configured (nyc, c8, jest --coverage, etc.)
  - Coverage threshold defined in package.json or config
technology:
  - node
---

### Coverage Gate

Enforce minimum coverage threshold before release:

```bash
# Example: Jest coverage with threshold
npx jest --coverage --coverageThreshold='{"global":{"branches":80,"functions":80,"lines":80,"statements":80}}'

# Example: nyc check-coverage
nyc check-coverage --lines 80 --functions 80 --branches 80
```

**If coverage drops below threshold, STOP the release.**
