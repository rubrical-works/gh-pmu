# Design Decision: `.gh-pmu.json` — Preserving Keys the Struct Does Not Model

**Date:** 2026-08-25
**Issue:** #910
**Status:** Accepted

## Context

`.gh-pmu.json` had an asymmetric read and write path. `Load` decoded with plain
`json.Unmarshal` and no `DisallowUnknownFields`, so a key the running binary did
not model was accepted without error. `Config.Save` re-marshalled the struct,
with no `MarshalJSON`, no `json.RawMessage` catch-all and no overflow map — so
whatever was on disk but not in the struct did not come back.

The combination is what made it a defect rather than a documented limitation.
The file loaded cleanly, then came back shorter, and nothing in command output
said so. The user found out from `git diff`, if the file was tracked at all.

`Save` has five call sites, and two of them fire without any explicit save:
`LoadFromDirectoryAndNormalize` writes whenever `framework` is absent, and
`RefreshVersion` (#905) writes whenever the stored version is stale. The first
is the worst of the five — a *load* function that writes, reached by any command
at all.

## Decision 1: Preserve, rather than warn or document

The issue recorded three approaches: preserve unknown keys, detect them at load
and warn while still dropping them, or declare the file strictly struct-shaped
and document the loss.

**Chose preservation.** The issue's own impact analysis is what settles it: it
rates today's exposure as bounded — `release` is the only unmodeled key known to
exist in real configs, and #902 established it is dead — and forward
compatibility as unbounded. Only preservation touches the unbounded half.
Warning and documenting both leave an older binary deleting a newer binary's
keys; they change how loudly it happens, not whether it does.

Preservation is also the treatment this file already got once. #901 gave
`project.view` an `omitempty` round-trip specifically so a write would not drop
it — the same problem solved per-field, for a field the struct does model.
Generalizing that to fields the struct does *not* model is the consistent
reading, not a new policy.

**Cost accepted:** every write now goes through a custom `MarshalJSON`,
including the writes that carry nothing unmodeled, which is every config in
existence today. That is a new code path under existing behavior, and it is
guarded accordingly — see Decision 4.

## Decision 2: Splice the overflow into the encoded struct, do not merge maps

The obvious implementation is to decode into `map[string]json.RawMessage`, merge
the modeled fields back in, and marshal the map. It is shorter and it is wrong.

`encoding/json` emits **struct fields in declaration order** but **map keys in
sorted order**. Marshalling a merged map would have silently realphabetized
every existing config on its next write — `acceptance` first, `version` last, a
diff in every repository, for a change no user asked for and none could explain.

**Chose splicing.** `MarshalJSON` marshals the struct through a method-stripped
`type alias Config` to preserve field order, then writes the captured keys in
before the closing brace. Modeled keys stay exactly where they have always been;
unmodeled keys follow.

Two smaller calls fall out of this:

- **Unknown keys are emitted in sorted order.** The order they had in the source
  file is not recoverable from `encoding/json` without a token-level decoder, and
  nothing depends on it. Sorted is at least deterministic.
- **The spliced output is compact, not indented.** `Save` marshals through
  `json.MarshalIndent`, which re-indents whatever `MarshalJSON` returns, so the
  captured values pick up the surrounding indentation rather than keeping the
  whitespace they happened to have on disk.

## Decision 3: Derive the modeled-key set by reflection, not by hand

`UnmarshalJSON` has to subtract the modeled keys from everything the file
carried. A hand-maintained list of the nine names would work today and rot at
the first added field — and the failure would be quiet in the worst way: the new
field would be marshalled by the struct *and* replayed from the overflow map,
emitting the same key twice.

**Chose reflection over the struct tags**, evaluated once at package init.
Adding a field to `Config` updates the set automatically.

## Decision 4: The seventh criterion is pinned to observed bytes, not to intent

"Key order, indentation and the trailing newline are unchanged from current
behavior" is a claim about the old binary, and it cannot be verified by reading
the new one.

The golden in `TestSave_OutputIsUnchangedWhenNoUnmodeledKeysArePresent` was
captured by running the fixture through the **pre-change** `config.go` and
diffed against the post-change output. They match byte for byte. The golden also
carries the literal `\u003c` that `encoding/json` writes for `<`,
because that escaping is part of the behavior being held still.

Both this golden and the structural guards were mutation-verified — a 2-to-4
space indent change, a swap of two struct field declarations, a forced drop of
the overflow, and an added sixth `Save` caller each turn the relevant test red.
Characterization tests that pass before and after a change prove nothing until
something shows they can fail.

## Decision 5: `release` now survives the write, and that is not a revert of #902

`release` is the one unmodeled key known to exist in real configs. Before this
change it loaded silently and was deleted by the next write. It now loads
silently and stays.

This is a consequence of the general rule, not a special case, and #902 is
intact: the `Release` struct is still gone, nothing reads the block, and
`TestSave_OmitsReleaseKey` still requires that a config which never carried one
does not acquire one. #902's fourth acceptance criterion — a populated block
loads with no error and no warning — is unaffected, and holds by construction:
`Load` takes no writer and returns only `(*Config, error)`, so there is nowhere
for a warning to go.

## Decision 6: The YAML path is left alone

`Load` still parses YAML by file extension, and `yaml.v3` does not call
`UnmarshalJSON`, so a YAML-loaded config captures no overflow.

**Not fixed, deliberately.** `FindConfigFile` resolves `.gh-pmu.json` only and
`Save` writes JSON only, so no reachable path round-trips YAML. Adding a
parallel YAML overflow would be machinery for a path that does not exist.

## Consequences

- A key this binary does not model survives a write by any of the five callers.
- An older binary run against a newer binary's config no longer deletes what it
  does not understand.
- Hand-added keys survive unrelated commands.
- `.gh-pmu.json` output for configs carrying nothing unmodeled is byte-identical
  to what it was.
- Unmodeled keys are carried opaquely and sorted; the tool still reads none of
  them, and preserving one is not a commitment to supporting it.
- A new `Save` caller now fails `TestSaveCallSiteInventoryIsUnchanged` until
  someone confirms it saves a `Config` it loaded rather than one it built. That
  distinction is not decidable from a line-level scan, and the test says so
  rather than implying coverage it does not have.
