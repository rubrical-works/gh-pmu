# Design Decision: init Atomicity — Defer with Rollback

**Date:** 2026-05-05
**Status:** Accepted
**Context:** Issue #847 — init is non-atomic; destination project created up front leaves orphan on post-create failures

## Decision

`gh pmu init --source-project N` adopts the **defer** atomicity model:

1. Pre-flight checks and project creation run as before. If `CopyProjectFromTemplate` fails, no on-disk state has been written; nothing to roll back.
2. After project creation, all remaining setup (field validation, optional-field creation, label loop, refetch, config write) runs inside an atomicity boundary in `runInitPostCreate` (`cmd/init_atomic.go`).
3. **`writeConfigWithMetadata` is the last step.** If any prior step hard-fails, `.gh-pmu.json` is never touched.
4. **On any hard failure inside the boundary:**
   - Emit the structured failure trailer (`destinationProjectNumber=N`, `failedStep=...`, `configRewritten=bool`) on stderr.
   - Call `client.DeleteProject(newProject.ID)` to roll back the orphan. Best-effort — a rollback failure is logged but does not mask the original error.
5. **The label loop stays warn+continue** (per AC5). #849's retry layer transparently handles transient 5xx; persistent label failures are repo-config issues the user fixes externally. Treating them as hard errors would force users to clean up partial state before each retry — worse UX for non-essential setup.

## Rationale

The bug was: `init` creates a destination GitHub Project early, then performs several steps that can hard-fail. Any failure between project creation and the final config write left the user with an orphan project on the org and (in the worst case) a `.gh-pmu.json` rewritten to point at it.

Two atomicity strategies were considered:

- **Rollback variant:** snapshot `.gh-pmu.json` to `.gh-pmu.json.bak` before writing, restore on failure.
- **Defer variant (chosen):** write `.gh-pmu.json` only as the very last step, after every fallible operation has succeeded. On failure, no on-disk state needs restoration.

## Alternatives Considered

- **Rollback (snapshot + restore):** rejected. Adds a backup-file lifecycle, races on concurrent writes, and Windows file-locking edge cases. The defer variant achieves the same atomicity guarantee without introducing a new on-disk artifact.
- **Pre-flight validation only:** validate the source project's fields and target repo's labels *before* creating the destination project, so we never enter a state where rollback is needed. Rejected as scope creep — it adds GraphQL round-trips for every init even when the upstream is healthy, and doesn't fully eliminate post-create failure modes (e.g., the optional-field-creation path).
- **Make the label loop fail-fast (#847 AC5 option B):** rejected. With #849's retry layer transparent to callers, surviving label-loop errors are persistent (permission, malformed labels, sustained outage) rather than transient. Fail-fast on those errors would block init entirely on a non-essential issue and force the user to clean up partial state. Warn+continue lets init finish; users fix labels externally and re-run if needed.

## Consequences

**Positive:**
- Single source of truth for "did init succeed?" — the absence of `.gh-pmu.json` (or its unchanged pre-run state) is sufficient signal.
- No `.gh-pmu.json.bak` artifact left around if the process is killed mid-write.
- Callers (e.g., px-manager `#895`) get a structured failure trailer with `destinationProjectNumber`, `failedStep`, and `configRewritten` fields they can branch on without screen-scraping.
- Future-proof: adding new fallible steps means adding a new `failedStep` enum value and slotting the call inside the boundary; no new rollback bookkeeping per step.

**Negative / Trade-offs:**
- `client.DeleteProject` is best-effort. If it fails (network blip during rollback), the user is left with the orphan AND the trailer pointing at it. Mitigated by emitting a clear `warning: rollback DeleteProject(...) failed` line so the user knows to clean up manually.
- `writeConfigWithMetadata` itself can leave a partially-written file on Windows (POSIX rename atomicity isn't guaranteed cross-platform). The trailer flags `configRewritten=true` defensively in this case so callers know not to trust the file.
- A failure between `CopyProjectFromTemplate` succeeding and `runInitPostCreate` returning means the project briefly existed and was deleted — the orphan window is small but non-zero. Acceptable.

## Issues Encountered

None during implementation. The biggest design clarification was deciding whether the label loop should participate in the atomicity boundary; AC5 of #847 explicitly raised this as a sub-decision and we documented the warn+continue rationale.

## Files Touched

- `cmd/init_atomic.go` (new) — `runInitPostCreate` + `initPostCreateClient` interface
- `cmd/init.go` — wire `runInitPostCreate` into `runInitNonInteractive`; add `--source-project` guard + `--force` flag
- `cmd/init_failure.go` (new) — structured failure trailer + `failedStep` enum
- `internal/api/mutations.go` — `DeleteProject` mutation
- `cmd/init_test.go` — atomicity tests, guard tests, trailer tests

---
*Documented during completion of #847.*
