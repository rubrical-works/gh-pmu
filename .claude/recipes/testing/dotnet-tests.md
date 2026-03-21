---
name: dotnet-tests
description: Run .NET tests with dotnet CLI
extensionPoints:
  - pre-validation
  - pre-gate
appliesTo:
  - prepare-release:pre-validation
  - prepare-beta:pre-validation
  - merge-branch:pre-gate
prerequisites:
  - "*.csproj or *.sln"
---

### Run Tests

```bash
dotnet test
```

**If tests fail, STOP and fix before continuing.**
