---
name: cross-os-testing
description: Run CI tests across Ubuntu, Windows, and macOS using a matrix strategy
extensionPoints:
  - pre-validation
  - pre-gate
appliesTo:
  - prepare-release:pre-validation
  - prepare-beta:pre-validation
  - merge-branch:pre-gate
prerequisites:
  - "Cross-OS testing enabled in CI: /ci add cross-os-testing"
---

### Verify Cross-OS CI

```bash
gh run list --limit 1 --json jobs --jq '.[0].jobs[] | select(.name | test("ubuntu|windows|macos"; "i")) | {name, status, conclusion}'
```

Confirm all OS matrix jobs passed:
- **Ubuntu:** ✅
- **Windows:** ✅
- **macOS:** ✅

**If any OS job failed or is missing, STOP and investigate before continuing.**

Common cross-OS failures:
- Path separators: Use `path.join()` instead of string concatenation
- Case sensitivity: macOS/Windows are case-insensitive, Linux is not
- Symlinks: Windows requires elevated permissions for symlinks
- Line endings: Git autocrlf can cause test mismatches on Windows
