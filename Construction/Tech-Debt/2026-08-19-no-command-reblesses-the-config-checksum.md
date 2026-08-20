# Tech Debt: No command re-blesses .gh-pmu.checksum after an intentional edit

**Logged:** 2026-08-19
**Priority:** Low
**Related Issue:** #902

## Description

`internal/integrity` hashes `.gh-pmu.json` whole (SHA-256) and stores the digest
in `.gh-pmu.checksum`. Any deliberate edit to the config — by hand or by another
tool — invalidates that digest, and `gh pmu config verify` then reports
`Checksum: MISMATCH` until it is refreshed.

Nothing refreshes it as its stated purpose. `integrity.UpdateChecksumForConfig`
has three callers, and each does it as a side effect of some other job:

| Caller | Its actual job |
|---|---|
| `cmd/accept.go:94` | Records terms acceptance; rewrites the `acceptance` block (user, date, version) |
| `cmd/config.go:270` | Resolves and persists `project.view` (`--resolve-view`) |
| `cmd/field.go:202` | Field operations |

`config verify` itself is read-only and has no `--update` flag.

## Current State

Removing `"release": {}` from the repository config during #902 left the stored
digest stale with no blessed way to fix it. The workaround was to call
`integrity.UpdateChecksumForConfig` directly through a throwaway `main` inside
the module. That is the right code path but not a route a user has.

The alternatives are worse: `gh pmu accept --yes` refreshes the digest but
rewrites acceptance metadata as a side effect, and hand-writing the hex digest
bypasses the package that owns the format.

## Desired State

A `--update` (or `--accept`) flag on `gh pmu config verify` that recomputes and
stores the digest after showing what changed. Keeping it on `verify` means the
user sees the drift report before blessing it, which is the point of the gate.

## Remediation Effort

Small. One flag, one call to the existing `integrity.UpdateChecksumForConfig`,
guarded so it runs after the report rather than instead of it.

## Risks if Unaddressed

- Users who edit the config legitimately learn to ignore `Checksum: MISMATCH`,
  which is the signal the integrity check exists to raise.
- The next contributor in this position either reaches for `accept --yes` and
  silently rewrites acceptance metadata, or hand-edits the digest file.

---
*Tracked during completion of #902*
