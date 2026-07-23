# Design Decision: VCR Record/Replay Not Adopted

**Date:** 2026-07-14
**Issue:** #884
**Status:** Accepted (revisit trigger below)

## Context

Live-API integration tests caused a GitHub account lockout (2025-12-08) and are
permanently barred from CI (#876). #884 set out to replace the lost regression
protection with safe, deterministic coverage across three tracks. Track 2
proposed VCR-style record/replay: record real GraphQL responses once locally,
sanitize them, commit the cassettes, and replay them deterministically in CI —
e.g. `dnaeon/go-vcr` at the `http.RoundTripper` level, slotting into the
existing `api.SetTestTransport` hook.

The motivating argument is sound: hand-written mocks encode *our assumptions*
about GitHub's responses, and assumptions drift from reality silently.

#884's AC permits either adopting VCR **or** a documented decision not to. This
records the decision not to adopt, for now.

## Decision

**Do not adopt VCR record/replay.** Keep two existing mechanisms, which
together cover VCR's realistic value at lower cost:

1. **Transport-level fakes** — `internal/api/retry_integration_test.go` already
   proves canned `http.Response` sequences work through real go-gh plumbing
   (`fakeRoundTripper`). This is VCR's replay half, without the dependency or
   the recording workflow.
2. **Command-level scenario tests** — `cmd/scenario_test.go` (#884) drives the
   real stack through a mock GraphQL server, exercising genuine query
   construction and response decoding.

Schema-drift detection — the gap neither mechanism closes — is addressed by
vendored-schema validation (**#886**), which catches field renames and typos
with zero API traffic and no recorded fixtures to maintain.

## Alternatives Considered

1. **Adopt `dnaeon/go-vcr` now** — rejected for this release. Adds a dependency
   during a v1.4.x cycle focused on stability and security hardening, plus an
   ongoing cassette-maintenance workflow: re-record on every intentional
   response-shape change, and sanitize tokens and node IDs correctly *every*
   time. A sanitization miss commits a credential to git history.
2. **Record cassettes by hand** — rejected. That is what the existing fixtures
   already are; calling them cassettes adds ceremony, not fidelity.
3. **Rely on live tests before releases** — already the policy for `test/e2e`
   (#876), and unchanged by this decision. It is a complement, not a
   substitute: it is manual and depends on developer discipline.

## Consequences

**Accepted:**
- No new dependency; no cassette-maintenance burden; no sanitization risk.
- Fixtures remain hand-written and therefore remain assumptions. If GitHub
  changes a response *shape* in a way the schema still permits, our fixtures
  keep passing and the scenario tests will not notice.
- #886 narrows this: it catches drift in what we *ask for*, not in what GitHub
  *returns*. That residual gap is real and knowingly accepted; `test/e2e`
  before releases is the backstop.

**Revisit trigger.** Adopt VCR if either happens:
- A production bug reaches users that a recorded real response would have
  caught — i.e. our fixtures were wrong about GitHub's actual response, and
  neither #886 nor e2e caught it.
- Fixture maintenance in `cmd/scenario_test.go` starts costing more than
  re-recording would, e.g. the same shape is duplicated across many scenarios
  and drifts inconsistently.

## Issues Encountered

Two findings from #884 shaped this decision:

- **Coverage was never the constraint.** `cmd` sat at 77.0% before the scenario
  tests and 77.0% after, because the pre-existing wrapper smoke tests already
  *executed* the `run*` bodies while asserting nothing. The problem was
  assertion quality, not untested lines — so a fidelity mechanism like VCR was
  never the highest-value next step. See the adjusted accounting in TESTING.md.
- **Assumption risk is concentrated in query construction, not response
  shape.** The scenario tests caught a simulated regression that dropped
  `repo:` scoping from the search builder — which would have made `list` search
  all of GitHub. That class of bug is on the request side, where VCR's recorded
  *responses* add nothing.

---
*Documented during completion of #884*
