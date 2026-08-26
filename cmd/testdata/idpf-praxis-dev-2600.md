## Bug Report

> **Scope is wider than this report states — the section-anchoring half already mis-parses every
> proposal issue in this tracker (found 2026-08-23).**
>
> The description below frames the defect around checkboxes quoted inside fenced blocks. The
> missing **section anchor** is the half that is live right now, with no fence involved.
>
> `/proposal` emits a `### Lifecycle` checklist into every proposal issue it creates:
>
> ```
> - [ ] Proposal reviewed
> - [ ] Ready for PRD conversion
> ```
>
> Proposal issues have no `**Acceptance Criteria:**` section at all, so `parseAcceptanceCriteria()`
> returns those two lifecycle markers as acceptance criteria, with `source: 'acceptance_criteria'`
> — indistinguishable from real ones. Verified against #2604: two checkbox lines, both outside any
> fence, both lifecycle markers. This reproduces on **every** proposal issue, not just that one.
>
> A fix that adds only fence tracking would leave this untouched. Both halves are needed.
>
> **Relationship to #2604: prerequisite for backlog conversion, deliberately NOT folded.**
> That proposal carries 46 acceptance criteria that become story ACs, and each flows through this
> parser into `/work` Step 4 verification, Step 4b's force-move prohibition, Step 6a's AC-checkbox
> audit, and per-AC subtask creation. There is **no file overlap** — this issue's scope is
> six shared scripts plus a new `lib/` module (17 declared paths) — but still shares no file with #2604,
> so it is a dependency, not a collision. *(Premise corrected 2026-08-23 Review #3: this previously read
> "`work-preamble.js` plus one test". The no-overlap conclusion survives; its stated basis did not.)*
> Folding it into #2604
> was considered and rejected: it would invert the order, delay a repo-wide fix behind one feature,
> and discard the review findings this issue carries (re-reviewed 2026-08-23; check the live label rather
> than this line — it has been stale once already. When findings are open, clear them with
> `/resolve-review 2600` before working it).

**Description:**

**Prior Art:** found — this exact defect class was fixed once before in this repo, in the file whose solution this issue now reuses (searched: fenced code block, fence-aware, checkbox parsing, acceptance criteria parser, section anchoring, markdown scanner, `extractSection`, `parseAcceptanceCriteria`, `checkbox-scan`; 8 of 19 surfaces resolved, `partial: false`; issue history including closed)

| Reference | Relationship |
|---|---|
| #2523 (closed) | **Direct precedent.** Fixed a fence-blindness defect in `extractFilesToModify`, and is *why* `extractSection`'s `fenceAware` option exists at all. Regression coverage lives at `tests/scripts/shared/scope-drift-check.test.js:631,650`. This issue does not duplicate it — it generalises that solution from one caller to six. |
| #2540 (closed) | **Same defect class, different file.** `classifyMarker` matched the prior-art marker anywhere in the body, so prose naming it reported a completed sweep. An unanchored global regex matching outside its intended section — the identical failure mode, in `prior-art-marker.js`. Fixed there in isolation; no shared scanner resulted, which is the gap this issue closes. |
| #2542 (closed) | Sibling of #2540 — `classifyMarker` treating a `PARTIAL` marker as complete. Cited as evidence that marker/section scanning has been patched per-caller three times now. |

**Why this is warranted rather than already-shipped:** every prior fix was scoped to its own caller. Nothing was extracted, so each new caller re-derived the rule or missed it. That is the pattern this issue ends.

`parseAcceptanceCriteria()` in `.claude/scripts/shared/work-preamble.js` collects every checkbox-shaped line in the entire issue body, with no awareness of fenced code blocks and no scoping to the `**Acceptance Criteria:**` section. Illustrative checkboxes quoted inside a fence — common in bug reports that show a checklist as evidence — are returned as acceptance criteria.

The implementation is a bare global regex over the whole body:

```js
function parseAcceptanceCriteria(body) {
  const result = { source: 'acceptance_criteria', items: [] };
  if (!body) return result;

  const regex = /^- \[([ x])\] (.+)$/gm;
  let match;
  while ((match = regex.exec(body)) !== null) {
    result.items.push({ text: match[2].trim(), checked: match[1] === 'x' });
  }
  return result;
}
```

There is no fence tracking, no section anchor, and `source` is reported as `acceptance_criteria` unconditionally — so a caller cannot tell a real criterion from a quoted one.

Reproduced in isolation:

```
input body:
    Some prose.

    ```
    - [ ] quoted example box
    ```

    **Acceptance Criteria:**

    - [ ] real criterion

output:
    { "source": "acceptance_criteria", "items": [
        { "text": "quoted example box", "checked": false },
        { "text": "real criterion",     "checked": false } ] }
```

The quoted example is returned **first**, ahead of the real criterion.

**Observed live on #2594.** That issue quotes a PRD tracker's 3-item lifecycle checklist inside a fence as evidence for the defect it reports. The preamble returned **10** items where only **7** were acceptance criteria; the 3 phantom entries (`PRD reviewed`, `Test plan approved (see #{test_plan_issue})`, `Ready for backlog creation`) were the quoted evidence. They sorted ahead of the real ACs because they appear earlier in the body.

**Why this matters more than a miscount: `/work` and `gh pmu` disagree.**

`gh pmu move <n> --status in_review` validates unchecked checkboxes and **does** ignore fenced blocks — #2594 moved to `in_review` with a plain move, no `--force`, while those three lines were still unchecked. So the two components parse the same body by different rules:

| Component | Fenced `- [ ]` lines |
|---|---|
| `work-preamble.js` `parseAcceptanceCriteria()` | counted as acceptance criteria |
| `gh pmu move --status in_review` | ignored |

The `/work` execution rule builds its per-AC subtask list, its commit-per-AC gate, and its Step 4 verification pass from the preamble's list. Phantom criteria therefore produce subtasks for work that does not exist, and — the sharper risk — invite the operator to "resolve" them. Checking a phantom box means editing quoted evidence inside a bug report so that it no longer matches the source it quotes. On #2594 the phantoms were quoted from `CommandsSrc/create-prd.md`; checking them would have silently falsified the reproduction case.

The disagreement also means the failure is invisible from the `/work` side. Because `gh pmu` ignores the fenced lines, the `in_review` transition succeeds and nothing ever surfaces the overcount.

**Version:**

0.96.2

**Steps to Reproduce:**

1. Create an issue whose body contains a fenced code block with one or more `- [ ] ...` lines, plus a real `**Acceptance Criteria:**` section below it.
2. Run `node .claude/scripts/shared/work-preamble.js --issue <N>`.
3. Read `autoTask.items` — it contains the fenced lines alongside the real criteria, with `source: "acceptance_criteria"` for all of them.
4. Compare against `gh pmu move <N> --status in_review`, which validates only the real ones.

**Expected Behavior:**

`parseAcceptanceCriteria()` skips lines inside fenced code blocks, and preferably scopes extraction to the `**Acceptance Criteria:**` section. Its notion of a checkbox matches the one `gh pmu` enforces at the `in_review` gate, so `/work` and `gh pmu` cannot disagree about how many criteria an issue has.

**Actual Behavior:**

Every `- [ ]` line anywhere in the body is returned as an acceptance criterion, including quoted examples inside fences and checklists belonging to other sections. `/work` builds subtasks and gates from the inflated list while `gh pmu` validates the correct one, so the two silently diverge.

**Scope:**

- **In scope:**
  - A single fence-aware checkbox scanner in `.claude/scripts/shared/lib/checkbox-scan.js`, consumed by every framework path that reads checkboxes out of an issue body. **Built by lifting the existing `extractSection` out of `scope-drift-check.js` (line 144), not by writing fence tracking a third time** — see Proposed Fix
  - `.claude/scripts/shared/scope-drift-check.js` — its local `extractSection` is removed and re-imported from the shared module, and the duplicate fence loop in `hasFilesToModifyHeader` (line 216) is retired. **Constraint:** this file is pinned dependency-free by `04-deployment-awareness.md` and a file-scoped test, so the shared module inherits that constraint
  - `.claude/scripts/shared/work-preamble.js` — `parseAcceptanceCriteria()` reads through the shared scanner and scopes to the acceptance-criteria section, recognising **all three heading forms this repo emits** — `**Acceptance Criteria:**`, `### Acceptance Criteria`, `## Acceptance Criteria`. Anchoring on the bold form alone returns **zero** criteria for every story, because the canonical story template emits the `###` form (see Proposed Fix)
  - `.claude/scripts/shared/review-ac-checkoff.js` — `checkOffACs()` reads through the same scanner, so a fenced line is neither counted nor written
  - `.claude/scripts/shared/nonstop-audit.js` — `countUncheckedAcs()` / `countTotalAcs()` read through the shared scanner. **Highest-value convert after the two above:** its result drives audit (2), which returns `status: "fail"` and **blocks** the epic `in_review` move, so a fenced checkbox does not merely miscount — it stops an epic on a quotation
  - `.claude/scripts/shared/qa-extract.js` — `extractUncheckedAcs()` reads through the shared scanner. A fenced `- [ ]` here does not just inflate a number: it **creates a real QA sub-issue** on GitHub for a line that was never a criterion
  - `.claude/scripts/shared/review-interdependence.js` — the `acPattern` global regex (line 62) reads through the shared scanner, so quoted criteria stop polluting cross-issue overlap comparison
  - `.claude/scripts/shared/reset-issue-preamble.js` — `analyzeBody()`. **Different defect class; resolved 2026-08-23 Review #3 — convert it.** Its `/[x]/gi` and `/[ ]/g` counts are not line-anchored at all, so they already match `[x]` appearing inline in prose or a table, fenced or not. The counts feed a report rather than a gate, so this is a behaviour change and not purely a fix: `totalBoxes` will drop on any body mentioning `[x]` inline. Accepted deliberately — the numbers become the numbers the field names already claim
  - Jest coverage for fenced blocks (both ``` and ~~~), indented fences, nested/unterminated fences, each of the three heading forms, a section terminated by a sibling `###` heading, and a body with no acceptance-criteria section in any form
  - Helper registration for the new `lib/` module — `framework-manifest.json` `deploymentFiles.scripts["shared/lib"].files`, `constants.js` `INSTALLED_FILES_MANIFEST.scriptsLib.files`, and the `@framework-script {{VERSION}}` JSDoc line (`/work` Step 4e)
- **Out of scope:**
  - `gh pmu`'s own checkbox validation, which already behaves correctly — this issue aligns the framework side to it, not the other way round
  - Retroactively correcting issues already worked with an inflated AC list

<!-- FRAMEWORK-ONLY-START -->
**Deployment Impact:** deployed, and wider than first assessed. Six scanners plus `scope-drift-check.js` sit under `.claude/scripts/shared/`, which is symlinked into every user project — so every project running `/work` on an issue that quotes a checklist gets the inflated criteria list, the silent divergence from `gh pmu`, and (for epics) a blocked `in_review` move plus spurious QA sub-issues. The fix additionally introduces a new `shared/lib` module, which triggers the three off-band `/work` Step 4e registrations — `framework-manifest.json` `deploymentFiles.scripts["shared/lib"].files`, `constants.js` `INSTALLED_FILES_MANIFEST.scriptsLib.files`, and the `@framework-script {{VERSION}}` JSDoc line — or `deployment-parity.test.js` and `manifest-validation.test.js` fail. *(Refreshed 2026-08-23 Review #3: previously described `work-preamble.js` alone.)*
<!-- FRAMEWORK-ONLY-END -->

**Files to modify:**
- `.claude/scripts/shared/lib/checkbox-scan.js`
- `.claude/scripts/shared/work-preamble.js`
- `.claude/scripts/shared/review-ac-checkoff.js`
- `.claude/scripts/shared/nonstop-audit.js`
- `.claude/scripts/shared/qa-extract.js`
- `.claude/scripts/shared/review-interdependence.js`
- `.claude/scripts/shared/reset-issue-preamble.js`
- `.claude/scripts/shared/scope-drift-check.js`
- `tests/scripts/shared/nonstop-audit.test.js`
- `tests/scripts/shared/qa-extract.test.js`
- `tests/scripts/review-interdependence.test.js`
- `tests/scripts/shared/reset-issue-preamble.test.js`
- `tests/scripts/shared/scope-drift-check.test.js`
- `tests/scripts/shared/lib/checkbox-scan.test.js`
- `tests/scripts/shared/work-preamble-ac-fences.test.js`
- `tests/scripts/shared/review-ac-checkoff-fences.test.js`
- `framework-manifest.json`
- `.claude/scripts/framework/constants.js`
- `CHARTER.md`
- `.claude/scripts/shared/generate-test-plan.js`
- `.claude/scripts/shared/mockup-ac-generator.js`
- `tests/fixtures/issue-bodies/issue-2594.md`
- `tests/fixtures/issue-bodies/issue-2503.md`
- `tests/fixtures/issue-bodies/issue-2505.md`
- `Construction/Design-Decisions/2026-08-23-one-checkbox-scanner-lifted-not-rewritten.md`
- `Construction/Tech-Debt/2026-08-23-acceptance-criteria-section-anchoring-still-triplicated.md`

**Scope added during implementation (2026-08-23):** `generate-test-plan.js` and `mockup-ac-generator.js` are a seventh and eighth fence-blind scanner, found by sweeping `.claude/scripts/` for checkbox regexes rather than working from the six-scanner table above. Both extract acceptance criteria from issue bodies, so AC6 ("**every** framework code path") covers them — leaving them would have made AC6 literally false while appearing satisfied, which is the failure Review #3's tightening targeted. Their conversion is narrow: only the fence mask and the checkbox definition move; each keeps its own section-anchoring rule, and the remaining triplication is recorded as tech debt.

**Acceptance Criteria:**

- [x] `parseAcceptanceCriteria()` excludes `- [ ]` and `- [x]` lines inside fenced code blocks — Jest unit test over the function, covering both ``` and ~~~ fences
- [x] A body whose only checkboxes are inside a fence yields zero items rather than phantom criteria — Jest unit test
- [x] Real criteria outside fences are still extracted, with `checked` state preserved, and existing callers see no change for bodies containing no fenced checkboxes — Jest regression test
- [x] An unterminated fence does not cause the remainder of the body to be silently dropped; the chosen behaviour is asserted rather than left incidental — Jest unit test
- [x] The #2594 body is exercised as a fixture and yields exactly 7 items, not 10 — Jest unit test against the recorded body
- [x] Every framework code path extracting checkboxes from an issue body reads through the shared scanner. *(Tightened 2026-08-23 Review #3: this previously accepted a scanner being "listed in the issue as separately affected", which the scope note now does for all six — so the criterion could pass with nothing converted. Enumeration no longer satisfies it.)*
- [x] `nonstop-audit.js` `countUncheckedAcs()` / `countTotalAcs()` read through the shared scanner, and a fenced `- [ ]` line does **not** contribute to `status: "fail"` — Jest unit test. Called out separately because this result blocks the epic `in_review` move, so a fenced quotation halts an epic rather than miscounting it
- [x] `qa-extract.js` `extractUncheckedAcs()` reads through the shared scanner, and **no QA sub-issue is created** for a fenced `- [ ]` line — Jest unit test. Called out separately because this path writes to GitHub, so the defect produces a durable artifact rather than a wrong number
- [x] `review-interdependence.js` `acPattern` reads through the shared scanner, so quoted criteria no longer enter cross-issue overlap comparison — Jest unit test
- [x] `reset-issue-preamble.js` `analyzeBody()` reads through the shared scanner; a test asserts `[x]` appearing inline in prose or a table is no longer counted, and the changed numbers are asserted rather than left incidental — Jest unit test
- [x] `scope-drift-check.js` consumes `extractSection` from the shared module rather than defining its own, the duplicate fence loop in `hasFilesToModifyHeader` is gone, and the existing `scope-drift-check.test.js` suite (including the #2523 fenced-declaration regressions at lines 631 and 650) passes unchanged — Jest regression test
- [x] A repo-wide sweep for fence-toggle regexes under `.claude/scripts/` returns exactly one implementation — Jest structural test. This is the criterion that makes "one shared scanner" verifiable instead of aspirational
- [x] All three acceptance-criteria heading forms are recognised — `**Acceptance Criteria:**`, `### Acceptance Criteria`, `## Acceptance Criteria` — with a Jest unit test per form. A body using the `###` form must not return zero
- [x] A recorded **story** body is exercised as a fixture and yields only its `### Acceptance Criteria` items: #2503 returns **13**, not 22 — Jest unit test. The section must terminate before `### Documentation`, `### Edge Cases` and `### Definition of Done`, so that `- [ ] All acceptance criteria met` is never itself returned as an acceptance criterion
- [x] Every acceptance-criteria heading form emitted anywhere in `CommandsSrc/` and `Templates/` is recognised by the scanner — Jest structural test asserting the scanner's accepted-form list against a sweep of what the repo actually emits. Prevents a future template adding a fourth form that silently returns zero
- [x] `review-ac-checkoff.js` `checkOffACs()` reads checkboxes through the shared scanner, so a fenced `- [ ]` line is neither counted in `total` nor rewritten to `- [x]` — Jest unit test over a body whose **first** checkbox sits inside a fence
- [x] Positional alignment survives a fenced checkbox: because `checkOffACs()` advances `findingIndex` per matched line, a fence-blind early match shifts every later AC onto the wrong finding. A test asserts each real AC receives its own finding's verdict — Jest unit test
- [x] The new `lib/` helper is registered in all three required places (`framework-manifest.json`, `constants.js`, `@framework-script` JSDoc) and `tests/installers/deployment-parity.test.js` plus `manifest-validation.test.js` pass

**Scope widened 2026-08-23 — there are six fence-blind scanners, not two.** Found by sweeping `.claude/scripts/shared/` for checkbox regexes rather than by reasoning about callers:

| File | Site | Consequence of a fenced match |
|---|---|---|
| `work-preamble.js` | `parseAcceptanceCriteria()`, line 303 | inflated criteria list (original report) |
| `review-ac-checkoff.js` | `checkOffACs()`, lines 115, 122 | wrong body written back; positional shift |
| `nonstop-audit.js` | `countUncheckedAcs()`, line 91 | **blocks** the epic `in_review` move |
| `qa-extract.js` | `extractUncheckedAcs()`, line 73 | **creates a spurious QA sub-issue** |
| `review-interdependence.js` | `acPattern`, line 62 | pollutes cross-issue overlap comparison |
| `reset-issue-preamble.js` | `analyzeBody()`, lines 63-64 | report counts; **not line-anchored at all** |

Two of the four newly added scanners have consequences beyond a wrong number: `nonstop-audit.js` halts an epic, and `qa-extract.js` files a GitHub issue. Both are worse outcomes than the miscount this issue was filed for.

**AC6 now reads oddly and should be revisited before work starts.** It currently allows a scanner to be *"either shown to share this function **or** listed in the issue as separately affected."* That second branch was the escape hatch for scanners nobody had enumerated; with these four named and in scope, it no longer applies to them. Either tighten AC6 to require conversion, or add a per-scanner criterion — but do not leave a scanner in scope whose only acceptance test is that it was listed.

**Proposed Fix:**

Track fence state while scanning lines rather than running a global regex over the whole string: walk the body line by line, toggle an `inFence` flag on a line matching `^\s*(```|~~~)`, and skip checkbox matches while the flag is set. That is a small change to an already-small function and needs no new dependency, which matters because `work-preamble.js` is deployed and bound by the runtime dependency contract in `04-deployment-awareness.md`.

Section scoping is the stronger fix: anchor extraction at the acceptance-criteria section and stop at its end. That also excludes unrelated checklists elsewhere in a body, which fence-skipping alone does not. **Which heading forms count, and where the section ends, are load-bearing details — see the amendment below.**

**Resolved 2026-08-23 (Review #3) — ship both halves.** Fence-skipping alone was measured against live bodies and does not fix the half that is actually biting: #2604 and #2609 each return 2 phantom `/proposal` lifecycle markers, **both outside any fence**, so fence tracking leaves them at 2. Section anchoring takes them to 0.

| Body | Today | Fence-skip only | Both (chosen) |
|---|---|---|---|
| #2604 | 2 (phantom) | 2 — unfixed | **0** |
| #2609 | 2 (phantom) | 2 — unfixed | **0** |
| #2594 | 10 | 7 | **7** |
| #2600 | 9 | 9 | **9** |

**Amended 2026-08-23 (post-Review #4) — the anchor must accept three heading forms, not one, and section scoping is worth far more than first measured.**

Everything above assumed the acceptance-criteria section is marked `**Acceptance Criteria:**`. **Stories do not use that form.** The canonical **Story Body Template** — `/add-story` Phase 3, the atomic template `/create-backlog` is explicitly required to apply — emits `### Acceptance Criteria`. Anchoring on the bold form alone therefore returns **zero criteria for every story in this tracker**, which is a worse failure than the overcount this issue was filed for: `/work` would build no per-AC subtasks, the commit-per-AC gate would have nothing to iterate, Step 4 nothing to verify, Step 4b no unchecked boxes to detect, and Step 6a's AC-checkbox audit would report clean. Every gate passes vacuously.

Heading forms actually emitted by this repo:

| Form | Emitted by |
|---|---|
| `**Acceptance Criteria:**` | `/bug`, `/enhancement`, `prd-template.md`, `/add-story` Phase 5 |
| `### Acceptance Criteria` | **`/add-story` Phase 3 — the canonical story template**, `/split-story`, `/mockups` |
| `## Acceptance Criteria` | `/fw-gap-analysis` |

**The scanner accepts all three.** Terminator per form: the bold form ends at the next `**bold:**` marker or any `##`/`###` heading; the heading forms end at the next heading of the same or higher level, **or** at a `**bold:**` marker — because `**Files to modify:**` immediately follows `### Acceptance Criteria` in the story template.

**Getting the terminator right is most of the value here.** Measured against three live story bodies, `parseAcceptanceCriteria` today returns 40–58% phantom items — and not one of them is fenced or a lifecycle marker. They are ordinary story sections that happen to contain checkboxes:

| Story | Returned today | Real ACs | Phantom | Phantom sources |
|---|---|---|---|---|
| #2503 | 22 | **13** | 9 | Documentation 1, Edge Cases 3, Definition of Done 5 |
| #2505 | 12 | **5** | 7 | Documentation 1, Edge Cases 2, Definition of Done 4 |
| #2507 | 7 | **3** | 4 | Documentation 1, Edge Cases 1, Definition of Done 2 |

`Definition of Done` is the sharpest case: it contains `- [ ] All acceptance criteria met`, so the parser returns, *as an acceptance criterion*, a checkbox asserting the acceptance criteria are met. Every story in this tracker has carried that.

This reframes the issue. The fenced-checkbox case it was filed for is real but rare — it needs a bug report that quotes a checklist. **Section-blindness fires on every story, on every `/work` run, today.** The decision to ship both halves was already correct; this is why it was correct.

**Resolved 2026-08-23 — the no-heading rule, which is the part section anchoring must not leave incidental.** A body carrying **none of the three heading forms above** yields **zero items and a distinguishing signal** — a `sectionFound` flag (or equivalent) letting a caller tell *"no AC section"* apart from *"an AC section that was empty"*. Without it, an issue legitimately keeping its criteria elsewhere returns 0 silently, indistinguishable from a proposal issue that correctly has none.

This follows a pattern already in the repo rather than inventing one: `extractFilesToModifySection` returns `{ paths, sectionFound }` for exactly this reason, with the comment *"the caller needs to tell 'no declaration' apart from 'a declaration that parsed to nothing'"* (#2523 Mode 3).

**Resolved 2026-08-23 — build by lifting, not by rewriting.** Fence-aware section extraction already exists here: `scope-drift-check.js:144` has `extractSection(body, isHeader, isTerminator, { fenceAware: true })` — fence tracking *and* section anchoring, the exact pair chosen above — plus a second fence loop in `hasFilesToModifyHeader` at line 216. A repo-wide sweep for fence-toggle regexes under `.claude/scripts/` returns **exactly those two hits and nothing else**: fence awareness exists in one file, twice, exported from neither.

Writing `lib/checkbox-scan.js` independently would make three implementations — the duplication this issue exists to remove, in a new place. So: lift `extractSection` into the shared module, have `scope-drift-check.js` consume it, retire the duplicate loop at line 216, and build checkbox scanning on top. Net fence implementations repo-wide: **2 → 1**.

**Where the fix lands was settled in Review #1: one shared scanner, not a per-caller copy.** `review-ac-checkoff.js` `checkOffACs()` (lines 113-125) scans `/^- \[ \] /` line by line with no fence tracking and then **writes the modified body back** through `gh pmu edit`, making it the more damaging of the two callers. Its harm is not only a wrongly ticked quotation: check-off is positional, advancing `findingIndex` on every matched line, so one fenced match early in a body shifts every subsequent criterion onto the wrong finding's verdict.

The Out-of-scope section previously excluded that file "whose positional check-off was addressed separately in #2594". That premise was false and has been removed. #2594's only commit to the file — `fc3859d3, "skip positional AC check-off for tracker-shaped review types"` — added an `isTrackerType(type)` guard for `prd`/`proposal`/`test-plan`. Issue-shaped reviews (`bug`, `enhancement`, `story`, `epic`) still check off positionally and still cannot see a fence. Note the reflexive case: this issue and #2594 both quote checkbox lines inside fences, so reviewing either one exercises the defect.

Extracting to `lib/checkbox-scan.js` satisfies AC6 by construction rather than by enumeration, and makes a future third caller inherit the rule instead of re-deriving it. It also brings `/work` Step 4e into play — a new `shared/lib/` module needs three off-band registration edits or CI fails — which is why `framework-manifest.json` is declared above: it is an always-protected path, so an undeclared touch halts Step 4c.

**Source:** Observed during `/work 2594 --assign`, 2026-08-20 — the preamble reported 10 acceptance criteria where 7 existed. Reproduced in isolation before filing.
**Reviews:** 4### Files Changed
**Added:**
- Source:
  - `.claude/scripts/shared/lib/checkbox-scan.js`
  - `Construction/Design-Decisions/2026-08-23-one-checkbox-scanner-lifted-not-rewritten.md`
  - `Construction/Tech-Debt/2026-08-23-acceptance-criteria-section-anchoring-still-triplicated.md`
- Tests:
  - `tests/fixtures/issue-bodies/issue-2505.md`
  - `tests/scripts/shared/lib/checkbox-scan.test.js`
  - `tests/scripts/shared/review-ac-checkoff-fences.test.js`
  - `tests/scripts/shared/work-preamble-ac-fences.test.js`
  - `tests/fixtures/issue-bodies/issue-2503.md`
  - `tests/fixtures/issue-bodies/issue-2594.md`

**Modified:**
- Source:
  - `.claude/scripts/framework/constants.js`
  - `CHARTER.md`
  - `framework-manifest.json`
  - `.claude/scripts/shared/generate-test-plan.js`
  - `.claude/scripts/shared/mockup-ac-generator.js`
  - `.claude/scripts/shared/nonstop-audit.js`
  - `.claude/scripts/shared/qa-extract.js`
  - `.claude/scripts/shared/reset-issue-preamble.js`
  - `.claude/scripts/shared/review-interdependence.js`
  - `.claude/scripts/shared/review-ac-checkoff.js`
  - `.claude/scripts/shared/work-preamble.js`
  - `.claude/scripts/shared/scope-drift-check.js`
- Tests:
  - `tests/scripts/review-interdependence.test.js`
  - `tests/scripts/shared/nonstop-audit.test.js`
  - `tests/scripts/shared/qa-extract.test.js`
  - `tests/scripts/shared/reset-issue-preamble.test.js`

