# Tech Debt: Value structs tagged omitempty still marshal into .gh-pmu.json

**Logged:** 2026-08-19
**Priority:** Low
**Related Issue:** #902

## Description

`encoding/json` does not treat an all-zero struct as empty. A struct field
declared by value and tagged `omitempty` is therefore written on every `Save`,
even when nothing set it:

```go
Defaults     Defaults          `yaml:"defaults,omitempty" json:"defaults,omitempty"`
```

This is the exact defect #902 removed for `Release`. `Defaults` and `Triage`'s
nested `Apply TriageApply` carry the same shape. `Acceptance` and `Metadata`
avoid it by being pointers, which `omitempty` does honour.

## Current State

A config that sets no defaults still gains `"defaults": {}` on the next write.
`TestSave_OmitsReleaseKey` (`internal/config/config_test.go`) asserts only that
`release` is gone; the saved JSON it prints on failure shows `"defaults": {}`
sitting alongside it.

The consequence is cosmetic today — no drift is reported, because the key is
written identically on both sides of a comparison. It becomes real when the key
appears on one side only: a config written by an older or newer build, or a
fixture that was hand-authored without it. That asymmetry is what made the #901
e2e drift assertion untightenable until #902 landed.

## Desired State

Either make the field a pointer (`*Defaults`), matching `Acceptance` and
`Metadata`, or give `Config` a `MarshalJSON` that drops zero-valued sub-objects.
The pointer route is consistent with the two fields that already get this right,
at the cost of nil checks at each read site.

## Remediation Effort

Small for `Defaults` — one field, plus nil guards wherever `cfg.Defaults` is read.
`TriageApply` is nested inside a map value and needs the same treatment.

## Risks if Unaddressed

- The next config key added by value repeats the defect, and the reviewer has no
  written reason to catch it.
- Any future comparison between a config this tool wrote and one it did not will
  surface the phantom key as a difference.

---
*Tracked during completion of #902*
