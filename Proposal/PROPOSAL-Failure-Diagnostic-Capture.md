# Proposal: Failure Diagnostic Capture

**Status:** Draft
**Created:** 2026-07-21
**Updated:** 2026-07-23 (analysis review: location fixed, retention capped, header whitelist named, Phase 2 removed)
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
- **Output:** fixed in code at `os.UserConfigDir()/gh-pmu/diag/`
  (`~/.config/gh-pmu/diag/` on Linux, `%AppData%\gh-pmu\diag\` on Windows).
  Outside any repository working tree, so no `.gitignore` entry is needed and
  captures can never be committed. Each event records the repository
  (`owner/name`) it was captured against, preserving correlation.
- **Retention:** at most **100 event files**; the oldest is deleted on write.
  Bounded disk use even with capture left on permanently. Deletion is
  best-effort like everything else in the capture path.
- **Disabled by default.** No output, no filesystem access, no behaviour change.

Explicitly rejected during design (see Alternatives): a boolean plus an output
**path** in `.gh-pmu.json`; a repo-relative output directory; the OS temp
directory.

### Capture whitelist

| Captured | Excluded |
|----------|----------|
| Operation name | Response `data` payload |
| Query/mutation document text (see caveat below) | GraphQL variable **values** |
| Variable **names and GraphQL types** | `Authorization` and other credential headers |
| HTTP status code | All response headers not on the named list |
| GraphQL `errors[]` block | Repository/issue titles and bodies |
| Named response headers **only**: `X-GitHub-Request-Id`, `Retry-After`, `X-RateLimit-*`, `Content-Type` | Node IDs |
| Repository `owner/name` (correlation) | `X-OAuth-Scopes` and other account-descriptive headers |
| gh-pmu version, timestamp, classifier verdicts | |

Headers are whitelisted by **name**, not by class: an unlisted header is
excluded by default, so newly introduced sensitive headers (the way
`X-OAuth-Scopes` describes token scope inventory) cannot leak into events.

**Document-text caveat:** capturing the query/mutation document as-is is safe
only if every operation passes values exclusively via GraphQL variables. Any
operation that interpolates values into the document text (e.g. a search
string) turns the document into a value-leak channel — such operations must be
converted to variables or have their document excluded from capture. This is
an implementation criterion, verified by audit, below.

The classifier verdicts (`IsRetryable`, `IsTransient5xx`,
`IsFieldResolverUnavailable`, `IsRateLimited`, `IsAuthError`) are recorded
alongside the raw status and error body. **That pairing is the feedback loop**:
a captured event where the raw response is a 502 but `IsTransient5xx` returned
false is direct evidence of classifier drift, and converts into a regression
test immediately.

To keep that loop complete as the classifier set evolves, verdicts are
enumerated from a **single named registry** (a slice of `{name, func}` pairs)
consumed by the event writer, with a test asserting the registry covers every
exported `Is*` classifier in `internal/api`. A future classifier added to
`errors.go` but omitted from the registry fails the test rather than silently
under-reporting drift.

### Scope

Capture applies to gh-pmu GraphQL failures at the `internal/api` transport
boundary, reusing the existing `api.SetTestTransport` seam. Self-contained.

A previously drafted Phase 2 (the same capture contract applied to
`.claude/scripts/shared/*.js` framework tooling) has been **removed from this
proposal entirely** — it targets charter-excluded framework infrastructure and
is out of scope for gh-pmu. If pursued, it belongs in the framework repository
as its own proposal, referencing this event format as a contract.

### What this is not

It is **reactive**. It does not find bugs; it makes bugs that already occurred
far cheaper to convert into regression tests, and it makes silent classifier
drift observable. It does not address 200-response wrong-shape failures — that
gap remains as accepted in the VCR decision, with `test/e2e` as backstop.

Captured events are **local-only inspection artifacts**, not designed to be
attached to issues as-is. The whitelist is sized accordingly; developers review
and excerpt manually when sharing. Revisit the redaction bar if sharing demand
appears.

## Implementation Criteria

- [ ] Capture is disabled unless `GH_PMU_DIAG=1`; with it unset there is no
      filesystem access and no behaviour change on any code path
- [ ] Output directory is fixed in code at `os.UserConfigDir()/gh-pmu/diag/` and
      not derivable from `.gh-pmu.json`, command-line flags, or any
      repository-supplied input; it lies outside any repository working tree
- [ ] Retention is capped at 100 event files, oldest deleted on write;
      deletion failures are best-effort like all capture failures
- [ ] Capture is whitelist-based: a test asserts that response `data`, variable
      **values**, `Authorization` headers, and any header not on the named
      header whitelist are absent from a written event
- [ ] Response headers are whitelisted by **name** (`X-GitHub-Request-Id`,
      `Retry-After`, `X-RateLimit-*`, `Content-Type`), not by class
- [ ] An audit confirms every captured operation passes values exclusively via
      GraphQL variables; any operation interpolating values into document text
      is converted to variables or has its document excluded from capture
- [ ] Each event records both the raw failure (status, `errors[]`) and every
      classifier verdict, enabling drift detection
- [ ] Classifier verdicts are enumerated from a single named registry; a test
      asserts the registry covers every exported `Is*` classifier in
      `internal/api`
- [ ] Capture failures (unwritable directory, disk full) never fail or alter the
      user-facing operation — diagnostics are strictly best-effort
- [ ] No new module dependencies (`go.mod` direct requirements stay at 7)
- [ ] Event format documented in TESTING.md, including how to convert a captured
      event into a `cmd/scenario_test.go` scenario

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
- **Repo-relative output directory (e.g. `.gh-pmu-diag/`):** Rejected. Easiest
  to correlate with the failing repository, but requires a `.gitignore` entry in
  every repository gh-pmu runs against — captures land un-ignored in any repo
  missing the entry and are one careless `git add -A` from being committed. The
  user-config-dir location makes committing captures structurally impossible;
  correlation is preserved by recording `owner/name` in the event.
- **OS temp directory:** Rejected. No gitignore concern, but platform temp
  cleanup can silently destroy exactly the evidence the feature exists to keep.
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
  redaction whitelist, classifier registry), `TESTING.md`. No changes to `cmd/`,
  config schema, `.gh-pmu.json`, or `.gitignore` (output lives outside the
  working tree).
- **Risk:** **Low.** Default-off, no dependency, no config-schema surface, no
  user-visible behaviour change, best-effort by construction. The residual risk
  is redaction correctness, mitigated by whitelist-not-blacklist design (named
  headers, variables-only document capture) plus an explicit
  negative-assertion test.
- **Effort:** ~8 story points (capture hook 3, redaction + tests 3, docs 2).

## Resolved Questions (2026-07-23 analysis review)

| # | Question | Resolution |
|---|----------|------------|
| 1 | Does Phase 2 (framework tooling capture) belong in gh-pmu? | **Removed entirely** — out of scope. `.claude/**` is charter-excluded; if pursued, it is a separate proposal in the framework repository sharing only the event-format contract. |
| 2 | Retention — do captured events expire, or is pruning manual? | **Cap of 100 files, delete-oldest on write.** Promoted to an implementation criterion. |
| 3 | Should captured events be shareable (attached to issues)? | **Local-only for now.** Whitelist sized for local inspection; developers review before excerpting. Revisit if sharing demand appears. |

## References

- `Construction/Design-Decisions/2026-07-14-vcr-record-replay-not-adopted.md` — revisit triggers
- `Construction/Design-Decisions/2026-07-14-vendored-schema-validation-approach.md`
- `Proposal/PROPOSAL-Integration-Test-Alternatives.md` (Adopted, #876) — predecessor context
- #884 (mock-based coverage), #886 (vendored-schema validation), #888 (ProjectV2FieldCommon field selection)
