# Tech Debt: Config.Save emits an empty `release` key into configs that lacked one

**Date:** 2026-08-19
**Issue:** #901 (surfaced by, not caused by)
**Status:** Open

## What

`Config.Release` is declared as a non-pointer struct tagged `omitempty`
(`internal/config/config.go:25`):

```go
Release Release `yaml:"release,omitempty" json:"release,omitempty"`
```

`encoding/json` does not honour `omitempty` for struct values — only for empty
strings, zero numbers, nil pointers, and empty maps/slices/interfaces. A zero
`Release` therefore marshals as `"release": {}` rather than being omitted.

The consequence: any `cfg.Save(dir)` on a config that had no `release` key adds
one. Against the integrity check that is a real, if cosmetic, drift report —
`Added: release` — attributable to the tool rather than the user.

`Acceptance` and `Metadata` on the same struct are pointers (`:26-27`) and behave
correctly. `Release` is the outlier.

## How it surfaced

Not by looking for it. #901 added `--resolve-view`, which writes `project.view`
via `Save`, and an E2E test asserted the resolved view produced no drift. The run
failed reporting `Added: release` — the view key was correctly excluded, and a
different key drifted instead.

## Why it was not fixed here

Out of #901's scope, which is `project.view`. The fix touches config
serialization on every `Save` call site, so it deserves its own change with its
own regression coverage rather than riding along on a feature branch.

Practical impact is currently low: the repository's own `.gh-pmu.json` already
carries a `release` key, as do configs written by `gh pmu init`. Only a config
that never had one is affected — which is exactly what the E2E fixture is.

The E2E test was narrowed to assert what #901 actually promises (`project.view`
never appears in a drift report) rather than the broader "no drift detected",
with the reason recorded inline at `test/e2e/config_test.go` so it is not
mistaken for a weakened assertion.

## Fix when picked up

Make `Release` a pointer (`*Release`), matching `Acceptance` and `Metadata`, and
update the call sites that read `cfg.Release` fields to nil-check. Alternatively
implement `MarshalJSON` on `Config` to drop zero-valued struct members, though
that is more machinery than the problem warrants.

Either way, add a test asserting that loading and re-saving a config without a
`release` key produces byte-identical output — the same round-trip guarantee
`project.view` got in #901.
