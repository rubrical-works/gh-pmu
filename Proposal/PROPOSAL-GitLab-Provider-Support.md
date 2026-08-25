# Proposal: GitLab Provider Support

**Status:** Draft
**Created:** 2026-08-25
**Author:** AI Assistant
**Tracking Issue:** #909
**Diagrams:** None
**Companion:** Decision document — "The GitLab Seam" (Artifact)

---

## Executive Summary

gh-pmu is coupled to GitHub in exactly one package. `internal/api` is a total
rewrite; almost everything else survives a port intact, because `cmd/` already
talks to twenty-one narrow, self-declared interfaces rather than to a concrete
client.

The blocking question is not architectural. It is that **GitHub Projects V2 has
no equivalent on GitLab**, and the entire gh-pmu field, filter and board model
is built on it.

| Metric | Value | Source |
|--------|-------|--------|
| Total Go LOC | 79,713 | `wc -l` over non-vendored `*.go` |
| Source LOC | 26,606 | — |
| Test LOC | 53,107 | — |
| Exported methods on `*api.Client` | 68 | source inspection |
| Consumer interfaces in `cmd/` | 21 | source inspection |
| Non-test `*api.Client` references | 1 | source inspection |
| GraphQL query + mutation LOC | 5,059 | `queries.go` + `mutations.go` |
| Tests passing in CI | 2,244 | `go test -v -short ./...` |

---

## Problem Statement

gh-pmu runs only against GitHub. Teams on GitLab cannot use it at all, and there
is currently no articulated position on whether supporting them is feasible,
what it would cost, or what would be lost in translation.

Three specific unknowns block any decision:

1. **How much of the codebase is actually GitHub-specific?** Without a measured
   answer, "port it to GitLab" is unbounded.
2. **Where do Projects V2 custom fields live on GitLab?** GitLab issues carry
   fixed attributes plus labels. There is no custom-field system.
3. **Is a single binary serving both platforms viable, or does GitLab support
   require a fork?** The answer determines whether every future `cmd/` fix has
   to be applied once or twice.

### Measured coupling

| Package | Source | Tests | Fate under a port |
|---------|-------:|------:|-------------------|
| `cmd` | 13,472 | 39,828 | Survives — mocks interfaces, not clients |
| `internal/api` | 6,602 | 11,486 | Rewrite — GitHub schema, top to bottom |
| `internal/config` | 565 | 2,082 | Extend — needs a provider discriminator |
| `internal/testutil` | 505 | — | Rewrite — shells `gh` directly |
| `internal/ui` | 412 | 659 | Survives — pure rendering |
| `internal/integrity` | 290 | 614 | Survives |
| `internal/framework` | 158 | 323 | Survives |
| `internal/defaults` | 87 | 294 | Survives |
| `internal/version` | 7 | 59 | Survives |
| `test/e2e` | — | 2,257 | Rewrite — live GitHub fixtures |

`queries.go` (3,041 lines) and `mutations.go` (2,018) are inline typed structs
with tags such as `graphql:"projectV2(number: $number)"`, bound to GitHub's
schema shape by shape. They are not adaptable.

### The seam already exists

Each command declares only the slice of the API it needs, and `*api.Client`
satisfies each one structurally:

```go
// cmd/board.go:20
type boardClient interface {
    GetProject(owner string, number int) (*api.Project, error)
    GetProjectItemsForBoard(projectID string, filter *api.BoardItemsFilter) ([]api.BoardItem, error)
    SearchRepositoryIssues(owner, repo string, filters api.SearchFilters, limit int) ([]api.Issue, error)
    GetProjectFieldsForIssues(projectID string, issueIDs []string) (map[string][]api.FieldValue, error)
}
```

Exactly one non-test line in `cmd/` names the concrete client. Pressure to keep
commands testable built the provider boundary before anyone asked for a second
provider.

### Concept correspondence

| GitHub | GitLab | Verdict | Cost |
|--------|--------|---------|------|
| Labels (name / color / description) | Labels | Clean | Group labels add an inheritance dimension |
| Milestones | Milestones | Clean | — |
| Issue number | `iid` | Clean | — |
| `owner/repo` — two parts | `group/subgroup/…/project` | Signature change | ~40 of 68 methods; `validateOwner` and `validateRepo` reject `/` |
| Node ID `[A-Za-z0-9_=-]` | `gid://gitlab/Issue/123` | Validator change | `validateNodeID` rejects every GitLab GID |
| Sub-issues (native, preview header) | Work items and epics | Tier-gated | `split` and `sub` degrade by billing plan |
| **Projects V2 custom fields** | **No equivalent** | **No analog** | The entire field, filter and board model |

---

## Proposed Solution

Extract the 68-method client surface into `internal/provider.Provider`, keep the
existing GitHub client as `provider/github`, add `provider/gitlab`, and select
between them on a config key. Commands never learn which platform they are
talking to.

This is recommended over a fork specifically because it preserves the 39,828
lines of command tests, which already mock interfaces and therefore survive the
change untouched.

### Encoding Projects V2 onto GitLab

The workable encoding is **scoped labels**. A label such as
`status::in-progress` is mutually exclusive with every other `status::` label on
the same issue, which is a genuine single-select analog. Native attributes cover
two more field types outright.

| gh-pmu field | GitLab encoding |
|--------------|-----------------|
| Status | scoped label — `status::in-progress` |
| Priority | scoped label — `priority::high` |
| Estimate | native issue `weight` (integer) |
| Sprint / Iteration | native iterations |
| Text and date fields | no analog — drop, or bury in the description |

`.gh-pmu.json` already abstracts alias → field → value, so it can carry a field
*kind* alongside the field name. That is a small change to a file already being
migrated by the version-refresh work in #905. **The lossiness is not small, and
it is permanent.**

### Two consequences to price in

**Tier gating.** Epics, iterations and weights sit behind GitLab Premium and
Ultimate. A free-tier user gets a materially degraded tool, and half the field
model disappears.

**Batch reads die.** GitLab's GraphQL coverage is thinner than its REST API, so
a REST-first provider is the realistic path. That removes the basis for
`GetSubIssuesBatch`, `GetParentIssueBatch` and
`GetIssuesWithProjectFieldsBatch`, which exist specifically to exploit GraphQL
aliasing. REST turns one call into N, and `board`, `list` and `move --recursive`
all regress.

### Phasing

Phase 0 exists to kill the project cheaply if the encoding does not hold. The
order matters more than the estimates.

| Phase | Work | Cost |
|-------|------|------|
| 0 | Spike `GetProject`, `GetProjectItems`, `SetProjectItemField` against one real GitLab board using scoped labels. Build nothing else until this is answered. | 1–2 days |
| 1 | Extract the 68 methods behind `internal/provider.Provider`. Mechanical; the 2,244 passing tests are the safety net and no assertion should change. | ~1 week |
| 2 | Close the GitHub leaks through the interface (see criteria below). | ~2,000 LOC |
| 3 | Build the GitLab provider plus a `Capabilities()` probe so commands degrade loudly rather than half-working. | Months; ~5–6k LOC + tests |
| 4 | Provider conformance suite — one behavior table, run against both providers. | Ongoing |

---

## Implementation Criteria

### Phase 0 — encoding spike (gate)

- [ ] `GetProject`, `GetProjectItems` and `SetProjectItemField` implemented
      against a real GitLab board using scoped labels
- [ ] Written finding on whether the scoped-label encoding holds; if it does
      not, this proposal is rejected and no further phase begins

### Phase 1 — extract the seam

- [ ] `internal/provider.Provider` declares the 68-method surface
- [ ] Existing GitHub client relocated to `provider/github`, satisfying it
- [ ] `provider: github | gitlab` key folded into the #905 config-version
      migration, not added as a second migration path
- [ ] All 2,244 tests pass with zero assertion changes

### Phase 2 — close the leaks

- [ ] `owner, repo string` replaced by a `ProjectRef` carrying a full path
- [ ] `validateNodeID` replaced by a provider-supplied validator
- [ ] `Metadata.Fields[].ID` (a ProjectV2 field-ID cache with no GitLab
      counterpart) becomes provider-shaped
- [ ] `X-Github-Next` preview header becomes a provider concern
- [ ] `cmd/create.go` and `cmd/history.go` no longer call go-gh REST directly

### Phase 3 — GitLab provider

- [ ] `provider/gitlab` satisfies the full interface
- [ ] `Capabilities()` probe implemented; commands requiring absent capabilities
      fail with a clear message rather than partially succeeding
- [ ] Field-kind support (`scoped_label` / `weight` / `milestone` / `iteration`)
      in config

### Phase 4 — conformance

- [ ] Single conformance table of provider-independent behaviors, executed
      against both providers
- [ ] Existing `internal/api` tests rehomed as GitHub provider tests
- [ ] `cmd/` tests unchanged

---

## Alternatives Considered

- **Hard fork to `glab-pmu`** — copy the repo, delete `internal/api`, write a
  GitLab client against the same command code. Fastest to a first working
  version with no abstraction tax, but 13,472 lines of command logic diverge
  permanently and every fix gets applied twice, forever. **Rejected** on
  maintenance cost.

- **GitLab-native rewrite** — stop pretending Projects V2 ports; design around
  what GitLab actually has (scoped labels, boards, iterations, epics), reusing
  only the CLI ergonomics. The most honest option regarding the model mismatch,
  but it discards 53,107 lines of tests encoding years of hard-won behavior —
  the single most valuable asset in the repository. **Rejected** on what it
  throws away.

- **Do nothing** — remains the correct outcome if Phase 0 shows the scoped-label
  encoding does not hold. This proposal is explicitly structured so that
  answering one question cheaply can close it.

---

## Impact Assessment

- **Scope:** `internal/api` (rewrite, 6,602 LOC), `internal/config` (provider
  discriminator), `internal/testutil` and `test/e2e` (rewrite), plus ~2,000 LOC
  of leak-closing across `cmd/`. `internal/ui`, `internal/framework`,
  `internal/defaults`, `internal/integrity` and `internal/version` unaffected.
- **Risk:** **High.** The Projects V2 mapping is lossy and irreversible once
  users depend on it; GitLab tier gating removes half the field model for
  free-tier users; batch-read loss is a user-visible performance regression on
  `board`, `list` and `move --recursive`.
- **Effort:** Phase 0 is 1–2 days and is the only commitment being asked for
  now. Phases 1–2 are roughly three weeks. Phase 3 is multi-month, dominated by
  field-model edge cases rather than by code volume.

### Open decisions

These change what gets built and do not get easier by being deferred:

1. Does scoped-label status encoding survive a real board? If not, stop at
   Phase 0.
2. Free tier, or Premium and up? It decides which commands can exist at all.
3. Are we willing to lose batch reads, and the performance that goes with them?
4. Where does the binary live? gh-pmu is a `gh` extension — `project_name:
   gh-pmu`, module `github.com/rubrical-works/gh-pmu`, display name `gh pmu`.
   GitLab has no equivalent host, so it becomes standalone, and we inherit the
   auth, host resolution and token config that go-gh currently provides free.
5. Does offline schema validation survive? The vendored 73,084-line
   `testdata/graphql/schema.docs.graphql` and its provenance gate need a GitLab
   counterpart, or the charter's offline-validation guarantee quietly lapses on
   one side.

---

## Verification Notes

**Repository figures** in this document are measured from the working tree at
`pmu/1.5.3`: line counts via `wc -l` over non-vendored `*.go`; method and
interface counts by source inspection of `internal/api` and `cmd`; the test
count from `go test -v -short ./...` (1,638 top-level functions plus 606
subtests, 2 skipped, 0 failing).

**GitLab capability claims** — tier gating, work-item hierarchy, GraphQL
coverage, scoped-label semantics — are **not** verified against current GitLab
documentation and must be re-checked before Phase 0 begins. No prior-art sweep
was run for this proposal (`--prior-art` not requested).
