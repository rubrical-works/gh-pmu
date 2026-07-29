# Design Decision: An unresolvable --assignee aborts the command

**Date:** 2026-07-28
**Status:** Accepted
**Context:** Issue #895 — [Enhancement]: resolve --assignee to the authenticated GitHub login

## Decision

`CreateIssueWithOptions` now returns an error when any `--assignee` value cannot
be resolved, instead of writing a warning to stderr and creating the issue
without that assignee. This reverses **#872 finding 4**.

## Rationale

#872 found that a failed assignee lookup was being dropped silently and made it
warn with its reason, so a transient API failure would not look identical to
"user not found". That fixed *visibility* but left the *outcome* wrong: the
command still exited 0 and still returned a successfully-created issue that was
missing the assignee the caller asked for. Any script reading the exit code saw
success.

The deciding detail is ordering. Assignee resolution runs *before* the
`createIssue` mutation, so failing there creates nothing — there is no
half-created issue to reconcile, no partial state to explain, and no cleanup
path to write. The abort is strictly safer than the warning it replaces, which
is what makes reversing a prior deliberate decision defensible rather than
merely a change of taste.

## Alternatives Considered

- **Keep warn-and-skip, make only `@me` fatal**: preserves #872 untouched and is
  the smallest change, but leaves the silent-wrong-result defect in place for
  every typo'd or renamed login. Rejected — that is the same class of bug #872
  was trying to address, just narrowed.
- **Create the issue, then exit non-zero**: honours #872's "creation still
  proceeds" literally while letting scripts detect the failure. Rejected — it
  produces exactly the half-right artifact the abort avoids, and leaves the user
  to notice and fix the assignment manually.
- **Distinct exit codes for partial vs total failure**: rejected as out of scope.
  `main.go` exits 1 for every error and no numeric scheme exists to extend;
  inventing one here would be a cross-cutting convention introduced by a single
  feature. Partial and total failures are distinguished by *message* instead,
  following `subRemoveBatchError` (`cmd/sub.go`).

## Consequences

- An invalid login that previously warned and let creation proceed now fails the
  command with exit 1. This is a user-visible behavior change and warrants a
  release note in the next v1.5.x tag.
- Batch resolution is all-or-nothing: a command that cannot honour every assignee
  it was given creates nothing, rather than proceeding with a subset that looks
  successful while differing from what was asked.
- `TestCreateIssueWithOptions_AssigneeLookupFailureWarns` was rewritten as
  `..._AssigneeLookupFailureAborts`, with the reversal recorded in its comment so
  the next reader finds the reasoning rather than an unexplained flip.

## Issues Encountered

`@me` turned out to already work on one path: `list` routes through the Search
API when a repository filter is available, and GitHub resolves the
`assignee:@me` qualifier server-side. The same flag silently matched nothing on
the client-side fallback. The issue was originally written as though `@me` never
resolved anywhere; it was corrected during review to describe unifying divergent
behavior, which also meant the fix had to preserve a working path rather than
only repair a broken one.

Resolution needed the account node id, not just the login, because `createIssue`
takes assignee ids. Caching login and id together avoids paying two lookups per
value — `@me` still costs two the first time, since `viewer{login}` returns no
id, but the resolved login is cached under its own key so a subsequent explicit
mention of that same account is free.

---
*Documented during completion of #895*
