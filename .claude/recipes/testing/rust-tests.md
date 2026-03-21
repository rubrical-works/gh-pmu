---
name: rust-tests
description: Run Rust tests with cargo
extensionPoints:
  - pre-validation
  - pre-gate
appliesTo:
  - prepare-release:pre-validation
  - prepare-beta:pre-validation
  - merge-branch:pre-gate
prerequisites:
  - Cargo.toml
---

### Run Tests

```bash
cargo test
```

**If tests fail, STOP and fix before continuing.**
