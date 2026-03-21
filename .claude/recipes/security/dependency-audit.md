---
name: dependency-audit
description: Check for known vulnerabilities in dependencies
extensionPoints:
  - pre-validation
appliesTo:
  - prepare-release:pre-validation
  - prepare-beta:pre-validation
prerequisites:
  - Audit tool for your language (npm audit, nancy, pip-audit, etc.)
  - "CI integration: /ci add dependency-audit"
---

### Dependency Audit

```bash
# Node.js
npm audit --audit-level=high

# Go
go list -m all | nancy sleuth

# Python
pip-audit
```

**If high/critical vulnerabilities found, STOP and report.**
