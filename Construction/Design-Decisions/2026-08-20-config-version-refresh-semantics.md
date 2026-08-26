# Design Decision: config `version` — Refresh Semantics and Repurposing

**Date:** 2026-08-20
**Issue:** #905
**Status:** Accepted

## Context

`.gh-pmu.json` carries a top-level `version` field. Before this change it was
written in exactly two places: `gh pmu init`, and `config.MigrateYAML` — the
one-time YAML→JSON migration. `MigrateYAML` was invoked from `PersistentPreRunE`
on every command, but short-circuited to a no-op unless a legacy `.gh-pmu.yml`
still sat beside the JSON. Since that file was deleted on first hit, the version
stamp fired at most once per repository, ever.

`Config.Save()` round-trips `Version` unchanged, so no other command refreshed
it. The practical result: `version` recorded which binary *created* the config
and then drifted permanently out of date.

## Decision 1: Repurpose the field rather than restore its original use

#700 introduced `version` for a specific purpose — detecting whether `init`
needed re-running after an upgrade. That required comparing the stored value
against the running binary. The comparison consumer was never built; nothing in
the tree reads the field today.

Two directions were available. Build #700's missing consumer, keeping `version`
as a staleness *signal*. Or refresh it automatically, making it a last-written
*stamp*.

**Chose the stamp.** But this forecloses the signal permanently, and that is the
part worth recording: once the field is auto-refreshed, it always equals the
running binary, so it can never again indicate "this config predates your
install". Anything wanting init-staleness detection later needs a second field,
not this one.

The trade is accepted because the staleness signal has gone unbuilt for six
months across multiple releases, while the misleading provenance is live today —
a config reading `1.4.0` under a `1.5.3` binary invites exactly the wrong
conclusion about what wrote it.

## Decision 2: Check always, write only on mismatch

The issue as filed asked for an unconditional stamp on every command — "no
short-circuits". The implemented behavior checks on every non-exempt command but
writes only when `cfg.Version != currentVersion`.

**Rejected the unconditional write.** `Save()` rewrites the whole document, so
an unconditional save would bump the file's mtime on every single `gh pmu`
invocation and re-normalize line endings each time — `Save` emits LF, Windows
checkouts normalize to CRLF. That produces a permanently dirty-looking working
tree for no informational gain: the field's value would be identical either way.

The distinction is deliberate and load-bearing in the ACs: the *check* has no
short-circuit, the *write* does.

## Decision 3: `version` joins `driftExcludedKeys`

Writing `version` diverges the local config from the git HEAD blob.
`runDailyIntegrityCheck` runs in the same `PersistentPreRunE`, three lines after
the refresh, and compares exactly those bytes. Without an exclusion, one command
writes the file and then reports itself as drifted — a warning on every version
bump, and under strict mode a hard error blocking every command until
`.gh-pmu.json` is committed.

This is the same failure #901 fixed for `project.view`, reached through a
different key. The fix is one entry in `driftExcludedKeys`
(`internal/integrity/integrity.go`), and the rationale comment there now covers
both keys.

Matching is on the full dotted path, so nested `acceptance.version` — a separate
field with real re-acceptance gating at `cmd/root.go:213` — remains watched.

**Cost of the exclusion:** a key added to that map stops being monitored for
tampering. `version` is informational and read by nothing, so suppressing it
costs no real alerting. That reasoning does not generalize; the list is meant to
stay minimal.

## Decision 4: Compare-and-save lives in `internal/config`

The natural home for the new logic was `cmd/root.go`, replacing
`runYAMLMigration` in place. That function had zero test coverage —
`cmd/root_test.go` held only help/version/usage tests — and it reads
`os.Getwd()`, which makes both write branches awkward to exercise.

**Chose extraction.** `config.RefreshVersion(dir, currentVersion) (bool, error)`
holds the comparison and the write; `cmd/root.go` keeps a thin wrapper plus a
`refreshConfigVersionInDir` seam that takes the directory outright. Both branches
are now unit-testable without touching the process working directory.

## Decision 5: The refresh honors `exemptCommands`

`checkAcceptance` and `runDailyIntegrityCheck` both skip `init`, `accept`, and
`help`. The refresh follows suit rather than running truly everywhere.

**Rationale:** these are the commands a user reaches for *before* a repo is
configured, or to read documentation. Writing to the config as a side effect of
`gh pmu help` would be surprising, and it is the kind of surprise that shows up
as an unexplained dirty file rather than an error anyone can trace.

## Consequences

- `version` now honestly answers "which gh pmu last wrote this file".
- It can no longer answer "does this config predate my install".
- Read-only commands stay read-only when the version already matches.
- One more key is exempt from drift alerting.
- The removal of `MigrateYAML` retires the last consumer of
  `LegacyYAMLFileName`; `Load` still parses YAML by extension, so a
  hand-supplied `.yml` path is unaffected.
