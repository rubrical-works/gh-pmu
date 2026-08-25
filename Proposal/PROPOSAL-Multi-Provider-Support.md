# Proposal: Multi-Provider Support

**Version:** 1.2
**Status:** Draft
**Created:** 2026-08-25
**Updated:** 2026-08-25 — retitled from GitLab Provider Support (see Revision History)
**Author:** AI Assistant
**Tracking Issue:** #909
**Diagrams:** None
**Companion:** Decision document — "The GitLab Seam" (Artifact)

---

## Scope

This proposal decides two things:

1. **Should gh-pmu grow a provider abstraction**, so that commands run against
   platforms other than GitHub?
2. **Which platform validates it first**, and in what order do others follow?

It recommends yes to the first, and **GitLab Premium** to the second. Other
candidates — OneDev, Gitea, Forgejo — are assessed here as comparators that
establish whether the abstraction is worth building at all, not as parallel
proposals. See **Candidate Platforms**.

A platform earns its own proposal once the abstraction exists and it becomes an
implementation decision rather than a comparison — with
`**Predecessor:** PROPOSAL-Multi-Provider-Support.md`, matching the convention
in `PROPOSAL-Integration-Test-Alternatives.md`. For OneDev the trigger is
**Phase 2 complete and an instance running**, since its API is documented only
from a live instance and any estimate before that is a guess.

---

## Executive Summary

gh-pmu is coupled to GitHub in exactly one package. `internal/api` is a total
rewrite; almost everything else survives a port intact, because `cmd/` already
talks to twenty-one narrow, self-declared interfaces rather than to a concrete
client.

The blocking question is not architectural. It is where the gh-pmu field, filter
and board model lands on a platform that is not GitHub.

A platform survey on 2026-08-25 (see **Candidate Platforms**) answered that
better than expected for GitLab and worse than expected for the Gitea family:

- **GitLab now has native custom fields for work items**, generally available
  since GitLab 18.0. This is a close structural analog to Projects V2 and
  replaces the lossy scoped-label encoding this proposal originally assumed.
- **The price is tier.** Custom fields, scoped labels, weight and iterations are
  all Premium/Ultimate — including on self-managed. GitLab Free is not a viable
  target for gh-pmu's model.
- **Gitea and Forgejo are disqualified for `gh pmu board`.** Neither exposes any
  project/board REST API; both have stalled attempts and open proposals.

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
| **Projects V2 custom fields** | **Work item custom fields** (GA 18.0) | Close analog, **Premium+** | Group-scoped not project-scoped; no date type; quota limits |

---

## Candidate Platforms

Surveyed 2026-08-25. Ranked by the property that actually decides the port:
**is the board's column axis backed by a queryable field, or is it manual card
placement?** `gh pmu board` reconstructs a board from field values
(`GetProjectItemsForBoard` returns `BoardItem{Number, Title, State, Status,
Priority, Repository}`), so a board that is only UI state cannot be served.

| Platform | Licence | Board column axis | Field model | Verdict |
|----------|---------|-------------------|-------------|---------|
| **GitLab** (Premium+) | MIT core / EE | Label lists (Free); status, assignee, milestone, iteration lists (Premium+) | **Native custom fields**, GA 18.0 | **Recommended target** |
| **OneDev** | MIT | **Any single-valued custom field** | Native typed custom fields | **Strongest structural fit**; concentrated maintainership |
| GitLab (Free / CE) | MIT | Label lists only | Plain labels only | Degraded — see below |
| **Gitea** | MIT | Manual placement | Exclusive scoped labels | **No project/board API at all** |
| **Forgejo** | GPLv3+ | Manual placement | Exclusive scoped labels | **No project/board API at all** |
| Phorge | Apache-2.0 | Mostly manual | Maniphest custom fields | Not evaluated further |
| Plane / Taiga / Redmine | AGPL / GPL | Field-backed | Varies | Not forges — excluded |

### GitLab

Tier boundaries verified against current documentation:

| Capability | Tier |
|------------|------|
| Multiple issue boards | Free |
| Label lists on boards | **Free** |
| Scoped labels (`::` mutual exclusivity) | **Premium, Ultimate** |
| Work item custom fields | **Premium, Ultimate** (GA 18.0) |
| Issue weight | **Premium, Ultimate** |
| Iterations | **Premium, Ultimate** |
| Swimlanes, WIP limits, assignee/milestone/iteration/status lists | **Premium, Ultimate** |

**On Premium and above**, gh-pmu project fields map onto work item custom
fields nearly one-to-one. Supported types are single-select, multi-select,
number and text. Constraints to design around:

- **No date type.** Date fields still have no analog.
- **Group-scoped, not project-scoped.** Fields are defined on a top-level group
  and apply across its subgroups and projects — a genuine modelling difference
  from Projects V2, where fields belong to one project.
- **Quotas:** 50 active custom fields per top-level group, 10 per work item
  type, text values ≤ 1,024 characters, select fields ≤ 50 options.
- Exposed through the **work items GraphQL API** as a widget, with filtering.

**On Free and self-managed CE**, none of the above is available — not even
scoped labels. The only encoding left is plain labels, with mutual exclusivity
enforced client-side by gh-pmu (remove every `status::*`, then add one). That is
two calls instead of one, with no server-side guarantee: a user editing labels
in the web UI can silently put an issue in two states at once. **This is a
degraded mode, not a supported tier.**

### OneDev

The best structural match found, and the only candidate that would not require
encoding project fields as something else.

- **MIT licensed**, ~15.2k GitHub stars, ~7,400 commits, actively released
  (v16.5.8 as of August 2026).
- **Board columns can be backed by any custom field**, subject to one
  constraint: the field must not have `multiple` checked. This is precisely the
  Projects V2 single-select-drives-the-board model.
- **Configurable issue links including parent/child**, working across projects,
  with state-transition rules that consume the hierarchy (auto-close a parent
  when its children finish). Richer than GitHub sub-issues.
- REST API supports searching and updating issues **by custom field**.

Two cautions:

- **The API reference is served only from a running instance** (`/~help/api`),
  not published publicly. The port cannot be scoped from documentation — an
  instance has to be stood up first. Any OneDev estimate before that is a guess.
- **Maintainership is concentrated.** The project has historically been driven
  by a single developer, who has stated a commitment to long-term support and a
  fallback plan of commercial support. Activity is current and the licence is
  MIT, so the risk is bus factor rather than abandonment.

### Gitea and Forgejo — disqualified

Both were attractive on paper: Go, and a REST API deliberately shaped like
GitHub's. Both fail on the board.

- **Gitea has no project/board API.** The 1.24 API surface contains no project,
  column or card endpoints. Issue #36824 (opened 2026-03-04) requests them and
  remains open; the implementing PR #36008 has been idle since December 2025
  with unaddressed maintainer review. A separate issue (#37151) notes kanban
  ordering is unavailable to any external tool.
- **Forgejo has no project/board API either.** PR #9384 ("complete project board
  API") was not merged. A fresh Project API Proposal (discussions #466, opened
  2026-05-13) sets out 21 use cases, `read:projects` / `write:projects` token
  scopes, and a deliberately staged rollout — read-only first, then basic
  writes, then advanced. A WIP PR (#13714) and refactoring PRs (#13700) exist.
  Nothing has shipped as of August 2026.

Issues, labels and milestones would port cleanly to either. The board would not
port at all, and "Terminal Kanban board visualization" is In Scope in the
charter. **Revisit Forgejo when its staged Project API reaches read-only
parity** — that alone would make `gh pmu board` viable there.

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

**Primary encoding — native custom fields (Premium+).** GitLab work item custom
fields carry the model directly. No lossy translation is required for the field
types gh-pmu actually uses.

| gh-pmu field | GitLab encoding | Fidelity |
|--------------|-----------------|----------|
| Status | custom field, single-select | Direct |
| Priority | custom field, single-select | Direct |
| Estimate | custom field (number), or native `weight` | Direct |
| Sprint / Iteration | native iterations | Direct |
| Text fields | custom field, text (≤ 1,024 chars) | Direct, bounded |
| Date fields | **no analog** | Drop, or encode as text |

**Fallback encoding — labels (Free / CE).** Scoped labels are Premium, so the
free-tier fallback is plain labels with client-enforced exclusivity, and the
single-select invariant becomes gh-pmu's responsibility rather than the
server's. Recommended only as an explicitly degraded mode, if at all.

`.gh-pmu.json` already abstracts alias → field → value, so it can carry a field
*kind* alongside the field name. That is a small change to a file already being
migrated by the version-refresh work in #905.

### Two consequences to price in

**Tier gating.** Custom fields, scoped labels, weight and iterations are all
Premium/Ultimate — on GitLab.com, self-managed and Dedicated alike. Self-hosting
does not buy a way out of it. A Free or CE user gets plain labels and nothing
else, which is not enough to run gh-pmu's model.

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
| 0 | Spike `GetProject`, `GetProjectItems`, `SetProjectItemField` against a real GitLab **Premium** group using work item custom fields, and render one board from field values. Build nothing else until this is answered. | 1–2 days |
| 1 | Extract the 68 methods behind `internal/provider.Provider`. Mechanical; the 2,244 passing tests are the safety net and no assertion should change. | ~1 week |
| 2 | Close the GitHub leaks through the interface (see criteria below). | ~2,000 LOC |
| 3 | Build the GitLab provider plus a `Capabilities()` probe so commands degrade loudly rather than half-working. | Months; ~5–6k LOC + tests |
| 4 | Provider conformance suite — one behavior table, run against both providers. | Ongoing |

---

## Implementation Criteria

### Phase 0 — encoding spike (gate)

- [ ] Access to a GitLab Premium group confirmed (custom fields are not
      available below Premium, so the spike cannot run without one)
- [ ] `GetProject`, `GetProjectItems` and `SetProjectItemField` implemented
      against real work item custom fields via the work items GraphQL API
- [ ] One board rendered from field values, proving `gh pmu board` is
      reconstructible without board-specific APIs
- [ ] Group-scoped field semantics assessed against gh-pmu's per-project field
      model, and the 50-field / 10-per-type quotas checked against real configs
- [ ] Written finding on whether the custom-field mapping holds; if it does not,
      this proposal is rejected and no further phase begins

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
- **Risk:** **Medium-high**, revised down from High after the 2026-08-25 survey.
  Native custom fields remove most of the mapping loss that drove the original
  assessment. What remains: the addressable audience is GitLab Premium and above
  only; custom fields are group-scoped where gh-pmu assumes project-scoped; date
  fields have no analog; and batch-read loss is a user-visible performance
  regression on `board`, `list` and `move --recursive`.
- **Effort:** Phase 0 is 1–2 days and is the only commitment being asked for
  now. Phases 1–2 are roughly three weeks. Phase 3 is multi-month, dominated by
  field-model edge cases rather than by code volume.

### Open decisions

These change what gets built and do not get easier by being deferred:

1. Do work item custom fields survive contact with a real board, including their
   group-scoped semantics and quotas? If not, stop at Phase 0.
2. ~~Free tier, or Premium and up?~~ **Answered by the survey: Premium and up.**
   Free and self-managed CE lack custom fields, scoped labels, weight and
   iterations. The remaining decision is narrower — do we ship a degraded
   label-only Free mode at all, or refuse to run below Premium with a clear
   message? Recommendation: refuse, and say why.
3. Are we willing to lose batch reads, and the performance that goes with them?
4. Is OneDev worth a parallel spike? It is the only surveyed platform whose
   model needs no encoding at all, but its API is documented only from a running
   instance, so scoping it costs standing one up first.
5. Where does the binary live? gh-pmu is a `gh` extension — `project_name:
   gh-pmu`, module `github.com/rubrical-works/gh-pmu`, display name `gh pmu`.
   GitLab has no equivalent host, so it becomes standalone, and we inherit the
   auth, host resolution and token config that go-gh currently provides free.
6. Does offline schema validation survive? The vendored 73,084-line
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

**Platform claims** were unverified in v1.0 of this document. They were checked
against current vendor documentation and issue trackers on **2026-08-25**, and
the survey changed the recommendation materially — see **Candidate Platforms**
and the revision note below.

| Claim | Source |
|-------|--------|
| Board list types and tier badges | [GitLab issue boards](https://docs.gitlab.com/user/project/issue_board/) |
| Scoped labels are Premium/Ultimate | [GitLab labels](https://docs.gitlab.com/user/project/labels/) |
| Custom fields GA 18.0, Premium/Ultimate, types and quotas | [GitLab work item custom fields](https://docs.gitlab.com/user/work_items/custom_fields/) |
| Weight is Premium/Ultimate | [GitLab issue weight](https://docs.gitlab.com/user/project/issues/issue_weight/) |
| Iterations are Premium/Ultimate | [GitLab iterations](https://docs.gitlab.com/user/group/iterations/) |
| Gitea has no project/board API | [Gitea API 1.24](https://docs.gitea.com/api/1.24/), [go-gitea#36824](https://github.com/go-gitea/gitea/issues/36824), [go-gitea#37151](https://github.com/go-gitea/gitea/issues/37151) |
| Forgejo has no project/board API | [forgejo/discussions#466](https://codeberg.org/forgejo/discussions/issues/466), [forgejo!9384](https://codeberg.org/forgejo/forgejo/pulls/9384) |
| OneDev boards, fields, links, licence | [OneDev custom board](https://docs.onedev.io/tutorials/issue/custom-board), [issue links](https://docs.onedev.io/tutorials/issue/issue-links), [REST API](https://docs.onedev.io/restful-api), [theonedev/onedev](https://github.com/theonedev/onedev) |
| OneDev maintainership | [OneDev OD-481](https://code.onedev.io/onedev/server/~issues/481) |

**Still unverified:** OneDev's concrete API surface, which is published only
from a running instance (`/~help/api`). Any OneDev effort estimate is a guess
until an instance exists.

**No prior-art sweep** was run for this proposal (`--prior-art` not requested).

---

## Revision History

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-08-25 | Initial proposal. Assumed scoped labels as the primary encoding; assumed Gitea/Forgejo were plausible targets. |
| 1.1 | 2026-08-25 | Platform survey folded in. Primary encoding changed to native work item custom fields; Free/CE reclassified as non-viable; Gitea and Forgejo disqualified on missing board APIs; OneDev added as strongest structural fit; risk revised High → Medium-high. |
| 1.2 | 2026-08-25 | Retitled from *GitLab Provider Support* and renamed from `PROPOSAL-GitLab-Provider-Support.md`. The document decides the provider abstraction and its first target, not GitLab support alone, and the old title undersold that. Added **Scope**, including the trigger at which a candidate platform earns its own proposal. |
