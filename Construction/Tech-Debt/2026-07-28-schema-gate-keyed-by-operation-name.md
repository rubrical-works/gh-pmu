# Tech Debt: Schema gate coverage is keyed by operation name, not by call site

**Logged:** 2026-07-28
**Priority:** Low
**Related Issue:** #895

## Description

`internal/api/schema_operations_test.go` proves every GraphQL document this
package sends validates against the vendored schema. It builds a map keyed by
*operation name* (`docs[name] = d`), and `TestNamedOperations_CoverageIsComplete`
asserts every `gql.Query("...")` name found in the sources appears in that map.

Two different methods currently send documents under the same operation name
`GetUserID`: `GetOwnerID` (its user fallback) and `getUserID`. Because the map is
name-keyed, driving either one satisfies coverage for both, and only the
last-captured document is validated.

## Current State

The documents are structurally identical today — both select
`user(login: $login) { id }` — so the incidental coverage is real. #895 added a
direct invocation for `getUserID` to make the dependency explicit, but that does
not change the underlying mechanic: a later edit that adds a field to one of the
two documents would be validated only if that method happened to run last.

## Desired State

Coverage keyed by call site rather than operation name, so each method's document
is validated independently. Alternatively, give the two methods distinct
operation names (`GetOwnerUserID` for the `GetOwnerID` fallback), which would let
the existing name-keyed map work correctly.

## Remediation Effort

Small. Renaming one operation string plus its invocation entry is a few lines;
re-keying the map by call-site label is a contained change to one test file.

## Risks if Unaddressed

- A field added to `getUserID` or `GetOwnerID` can ship unvalidated against the
  vendored schema, which is the exact failure the gate exists to catch.
- The gate reads as complete coverage while providing partial coverage — the same
  hazard the `knownInvalidOperations` quarantine comment warns about.

---
*Tracked during completion of #895*
