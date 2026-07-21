# Proposal: Failure Diagnostic Capture

**Status:** Draft
**Created:** 2026-07-21
**Author:** API-Integration-Specialist
**Tracking Issue:** #892
**Diagrams:** None

---

## Problem Statement

When a gh-pmu operation fails against the live GitHub API, the diagnostic
evidence is discarded. What survives is whatever the developer manually
reconstructs afterwards — and the repository contains the scars of that
reconstruction.

`internal/api/errors.go:17-29` documents the 2026-05-14 ProjectV2 field-resolver
incident: an auto-rollout of Created/Closed/Updated timestamp fields whose
backing rows returned null for NON_NULL `name`/`dataType`/`id`, crashing bulk
`fields(first: N)` fetches while per-item reads kept working. That analysis was
performed by hand, from a failure that had already scrolled past. The resulting
four-path classifier (`IsFieldResolverUnavailable`, `errors.go:173-192`) is the
artifact of that labour. #888 on the current branch is another instance of the
same family.

**The sharper problem: the error classifiers themselves are unverified
assumptions about wire format.** Three of them string-match formats that neither
GitHub nor shurcoolGraphQL guarantees:

| Location | Assumed string | Consequence if it drifts |
|----------|----------------|--------------------------|
| `errors.go:133-135` | `"non-200 OK status code: 502\|503\|504"` | `IsTransient5xx` returns false; retry silently stops happening |
| `errors.go:190` | `"Something went wrong while executing"` | Field-resolver fallback never engages; bulk fetch fails hard |
| `errors.go:97` | `"rate limit"` / `"rate_limited"` on 403 | Secondary rate limits no longer retried |

Each failure mode is **silent**. Nothing goes red; retry simply stops occurring
and the tool degrades. Vendored-schema validation (#886) cannot see this — it
validates what we *ask for*, not how failures come back. The `cmd/scenario_test.go`
fixtures cannot see it either, because they encode the same assumptions.

This is the residual gap knowingly accepted in
`Construction/Design-Decisions/2026-07-14-vcr-record-replay-not-adopted.md`,
approached from the side that does not require recording production traffic.

## Proposed Solution

An **opt-in, env-var-gated diagnostic capture** that writes one file per failed
operation to a fixed location. Failures are rare, so volume is low and no
database is warranted. Capture is **whitelist-based**: fields are added
deliberately, never stripped after the fact.

### Gating and location

- **Enable:** `GH_PMU_DIAG=1` (environment variable only).
- **Output:** a fixed, non-configurable directory; gitignored in the same commit
  that introduces it.
- **Disabled by default.** No output, no filesystem access, no behaviour change.

Explicitly rejected during design (see Alternatives): a boolean plus an output
**path** in `.gh-pmu.json`.

### Capture whitelist (Phase 1)

| Captured | Excluded |
|----------|----------|
| Operation name | Response `data` payload |
| Query/mutation document text | GraphQL variable **values** |
| Variable **names and GraphQL types** | `Authorization` and other credential headers |
| HTTP status code | Repository/issue titles and bodies |
| GraphQL `errors[]` block | Node IDs |
| Non-credential response headers | |
| gh-pmu version, timestamp, classifier verdicts | |

The classifier verdicts (`IsRetryable`, `IsTransient5xx`,
`IsFieldResolverUnavailable`, `IsRateLimited`, `IsAuthError`) are recorded
alongside the raw status and error body. **That pairing is the feedback loop**:
a captured event where the raw response is a 502 but `IsTransient5xx` returned
false is direct evidence of classifier drift, and converts into a regression
test immediately.

### Phasing

**Phase 1 — gh-pmu GraphQL failures.** Capture at the `internal/api` transport
boundary, reusing the existing `api.SetTestTransport` seam. Self-contained; all
evidence above is Phase 1 evidence.

**Phase 2 — IDPF framework tooling.** The same capture contract applied to
`.claude/scripts/shared/*.js` failures (preamble scripts, hooks), feeding the
Jest suite. Shares the event format and redaction discipline; shares no code.
See Open Questions — Phase 2 targets charter-excluded framework infrastructure
and may belong in a different repository.

### What this is not

It is **reactive**. It does not find bugs; it makes bugs that already occurred
far cheaper to convert into regression tests, and it makes silent classifier
drift observable. It does not address 200-response wrong-shape failures — that
gap remains as accepted in the VCR decision, with `test/e2e` as backstop.

## Implementation Criteria

- [ ] Capture is disabled unless `GH_PMU_DIAG=1`; with it unset there is no
      filesystem access and no behaviour change on any code path
- [ ] Output directory is fixed in code and not derivable from `.gh-pmu.json`,
      command-line flags, or any repository-supplied input
- [ ] The output directory is added to `.gitignore` in the same commit that
      introduces capture
- [ ] Capture is whitelist-based: a test asserts that response `data`, variable
      **values**, and `Authorization` headers are absent from a written event
- [ ] Each event records both the raw failure (status, `errors[]`) and every
      classifier verdict, enabling drift detection
- [ ] Capture failures (unwritable directory, disk full) never fail or alter the
      user-facing operation — diagnostics are strictly best-effort
- [ ] No new module dependencies (`go.mod` direct requirements stay at 7)
- [ ] Event format documented in TESTING.md, including how to convert a captured
      event into a `cmd/scenario_test.go` scenario
- [ ] Phase 2 scoped in a follow-up issue, not implemented under this proposal

## Alternatives Considered

- **Boolean + output path in `.gh-pmu.json` (the originally proposed shape):**
  Rejected. `.gh-pmu.json` is **tracked** (`git ls-files` confirms) and gh-pmu is
  a cross-repository tool, so a config-supplied write path is an
  attacker-controlled file write arriving via `git clone` and triggered by a
  network failure. Confining the path would require a `..`/absolute/UNC/
  drive-relative/ADS-safe validator on a Windows-primary project — a large
  correctness burden to save a config field, during a cycle whose charter focus
  is security hardening.
- **Boolean in tracked `.gh-pmu.json`, fixed path:** Rejected. The toggle is a
  machine-local developer preference in a shared committed file: enabling it
  dirties the tree, churns `.gh-pmu.checksum` and trips integrity verification,
  and one careless commit enables collection for everyone who clones.
- **Untracked local config file (`.gh-pmu.local.json`):** Deferred, not
  rejected. It is the right home if the knob ever grows past on/off (capture
  verbosity, error-class selection, retention). An env var suffices for a
  boolean and requires no schema commitment.
- **Build tag (`//go:build diag`):** Rejected as primary. Safest — the code
  would not exist in shipped binaries — but it forfeits the core value, which is
  capturing the failure on the machine where it happened and which the developer
  cannot reproduce.
- **VCR record/replay, incl. SQLite-backed stateful fake:** Previously rejected
  (#884 decision record) and re-examined during this design. Bulk capture of
  successes is high-volume, secret-dense, and duplicates existing coverage;
  #884's own finding is that assumption risk concentrates in request
  construction, where recorded responses add nothing.
- **Schema-driven fakers (`graphql-faker` et al.):** Rejected. Random
  type-conforming data is strictly weaker than the offline validation
  `internal/api/schema_operations_test.go` already performs against the vendored
  schema, and adds a dependency plus a running process.

## Impact Assessment

- **Scope:** `internal/api/` (transport-boundary capture, event serialisation,
  redaction whitelist), `.gitignore`, `TESTING.md`. No changes to `cmd/`, config
  schema, or `.gh-pmu.json`. Phase 2 would touch `.claude/scripts/shared/` —
  charter-excluded framework infrastructure, scoped separately.
- **Risk:** **Low.** Default-off, no dependency, no config-schema surface, no
  user-visible behaviour change, best-effort by construction. The residual risk
  is redaction correctness, mitigated by whitelist-not-blacklist design plus an
  explicit negative-assertion test.
- **Effort:** Phase 1 ~8 story points (capture hook 3, redaction + tests 3,
  docs 2). Phase 2 unestimated pending the repository question below.

## Open Questions

| # | Question | Impact |
|---|----------|--------|
| 1 | Does Phase 2 belong in gh-pmu at all? `.claude/**` is explicitly excluded from charter scope validation (rule `04-charter-enforcement.md`), so framework tooling capture may belong in the framework repository. | Determines whether Phase 2 is a follow-up issue here or a proposal elsewhere |
| 2 | Retention — do captured events expire, or is pruning manual? | Unbounded growth on a developer machine with capture left on |
| 3 | Should captured events be shareable (attached to issues), and if so does that raise the redaction bar? | Whitelist is currently sized for local-only inspection |

## References

- `Construction/Design-Decisions/2026-07-14-vcr-record-replay-not-adopted.md` — revisit triggers
- `Construction/Design-Decisions/2026-07-14-vendored-schema-validation-approach.md`
- `Proposal/PROPOSAL-Integration-Test-Alternatives.md` (Adopted, #876) — predecessor context
- #884 (mock-based coverage), #886 (vendored-schema validation), #888 (ProjectV2FieldCommon field selection)
