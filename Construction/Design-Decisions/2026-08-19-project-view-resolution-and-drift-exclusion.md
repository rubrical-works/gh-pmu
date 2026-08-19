# Design Decision: project.view — Owner Resolution, Drift Exclusion, Opt-In Write

**Date:** 2026-08-19
**Issue:** #901
**Status:** Accepted

## Context

`.gh-pmu.json` gained `project.view`, the number of the project's first Backlog
view with a `BOARD_LAYOUT` layout. It is needed to build view-scoped URLs
(`{projectUrl}/views/{number}`) and cannot be assumed to be `1`: view numbers are
creation ordinals that are never backfilled, so org-owned boards routinely start
at `2`. Resolving it on every command would cost an extra GraphQL round-trip, so
it is cached in the config — following the `metadata.fields[]` precedent from
#853.

Three choices dominated the work, and each had a plausible alternative that was
rejected for a concrete reason.

## Decision 1: One `repositoryOwner` query, not try-user-then-org

`GetProject` resolves owners by attempting `getUserProject`, falling back to
`getOrgProject`, and joining both failures with `errors.Join`
(`internal/api/queries.go:40,46,85`). Reusing that idiom would have been the
consistent choice.

**Rejected.** A joined error cannot distinguish a login that resolves to no user
or organization from an owner that exists but has no board at that number — the
misattribution #861 fixed for a different caller. #901 requires those two to stay
separable, because collapsing them makes a typo'd owner read as a missing view
and sends the user to fix the wrong argument.

`repositoryOwner(login:) { ... on ProjectV2Owner { projectV2(number:) } }` gets
both owner types in one round-trip, and leaves the two null cases individually
observable: a null owner leaves `__typename` empty, a null project leaves its
non-null `ID` empty. They map to `ErrOwnerNotFound` and `ErrProjectNotFound`.

A third outcome — the board exists but has no Backlog view — is deliberately
**not** an error. GitHub has no `createProjectV2View` mutation (confirmed
independently in `backlog/product-backlog.md` Story 2.2 and
`PRD/Implemented/gh-pm-unified-PRD.md:100`), so it is a fact to report, never
something to repair. It surfaces as `found == false` with a nil error.

## Decision 2: Exclude `project.view` from the drift comparison

The issue framed this as open: exclude the key from the comparison, or keep the
write explicit so the user expects the diff. It also carried an acceptance
criterion asserting that calling `integrity.UpdateChecksumForConfig` after the
write would stop the next run reporting drift.

**That criterion was wrong, and the review caught it.** `runIntegrityCheck` never
consults the stored checksum. It reads the committed HEAD blob via `gitShowFile`
(`cmd/integrity_check.go:39`) and hands it to `CompareContent` (`:46`), which
sha256s local against committed. Refreshing the checksum file cannot influence
that comparison at all. Worse, `cmd/integrity_check.go:67` escalates drift to a
hard error under strict mode, so a tool-written key would block every subsequent
command until the user committed `.gh-pmu.json`.

**Chosen: exclusion.** `diffJSON` now drops changes whose dotted path is in
`driftExcludedKeys` and reports how many it dropped, so `CompareContent` can tell
"only our own bookkeeping moved" (not drift) from "changed in a way `diffMaps`
could not name" (still drift — formatting-only edits keep reporting, as before).

Scoped to the exact path `project.view`. #792 added deliberate `project.*`
alerting to `config verify`; a broader exclusion would have silently regressed
it. A top-level `view` key is unaffected, and `compareCriticalFields` — which
watches `project.owner`, `project.number` and `repositories` — never looked at
`view` and is untouched.

The exclusion suppresses drift reporting, so anything added to that list stops
being watched. Keep it minimal.

## Decision 3: Resolution is opt-in, `config verify` stays read-only

The issue's acceptance criteria named `gh pmu config verify` as the resolve site.
Wiring it into every verify run was tried first and rejected on two counts.

**Rejected.** `verify` is documented as "Verify config integrity against git
HEAD" (`cmd/config.go:37-38`). A verification run that rewrites the thing it
verifies inverts that contract, and it would give a command that needs neither
network nor auth a hard dependency on both — including in unit tests, which
started making live API calls the moment it was wired in.

**Chosen:** a `--resolve-view` flag. Plain `verify` never writes. The flag both
enables resolution and forces re-resolution of a value already present, which is
consistent with the requirement that a hand-edited value is authoritative and
that the explicit path is the only one allowed to replace it.

When the flag is passed, resolution runs *before* the local file is read. The
comparison hashes whatever is on disk at that moment, so resolving afterwards
would make the same run report the write it had just performed.

A resolve failure warns and continues rather than failing the command. Making an
unresolvable cache field fatal would be a worse outcome than leaving it unset. A
`Save` failure is the one loud case, and it deliberately leaves the checksum
alone: recording a checksum for a file that was not written would describe a
state that is not on disk.

## Consequences

- `project.view` can be resolved once and cached without ever tripping the
  integrity check, in strict mode or otherwise.
- Missing owner, missing project and missing Backlog view remain three separate,
  individually actionable outcomes.
- `verify` keeps working offline and unauthenticated in its default form.
- `driftExcludedKeys` is a small, load-bearing list. Each future entry trades
  drift visibility for quiet, and should be justified the way this one was.
- A separate pre-existing defect surfaced during E2E work and was left alone —
  see `Construction/Tech-Debt/2026-08-19-config-save-emits-empty-release-key.md`.
