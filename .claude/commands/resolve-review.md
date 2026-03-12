---
version: "v0.61.0"
description: Resolve review findings for an issue (project)
argument-hint: "#issue"
---
<!-- MANAGED -->

# /resolve-review
Parse the latest review findings on an issue and systematically resolve each one. Works with findings from `/review-issue`, `/review-proposal`, `/review-prd`, and `/review-test-plan`.

## Prerequisites
- `gh pmu` extension installed
- `.gh-pmu.json` configured in repository root
- Issue has at least one review comment (from `/review-issue`, `/review-proposal`, `/review-prd`, or `/review-test-plan`)

## Arguments
| Argument | Required | Description |
|----------|----------|-------------|
| `#issue` | Yes | Issue number (e.g., `#42` or `42`) |

## Execution Instructions
**REQUIRED:** Before executing:
1. **Create Todo List:** Use `TodoWrite` to create todos from the steps below
2. **Track Progress:** Mark todos `in_progress` → `completed` as you work
3. **Resume Point:** If interrupted, todos show where to continue

## Workflow

### Step 1: Fetch Review Comment
Retrieve the latest review comment from the issue:
```bash
gh api repos/{owner}/{repo}/issues/$ISSUE/comments --jq 'reverse | .[0:10]'
```
Scan comments in reverse chronological order for a review comment. Match by header pattern:
| Pattern | Review Type |
|---------|-------------|
| `## Issue Review #N` | Issue review (`/review-issue`) |
| `## Proposal Review #N` | Proposal review (`/review-proposal`) |
| `## PRD Review #N` | PRD review (`/review-prd`) |
| `## Test Plan Review #N` | Test plan review (`/review-test-plan`) |
**If no review comment found:**
```
No review found on issue #$ISSUE. Run one of these first:
  /review-issue #$ISSUE
  /review-proposal #$ISSUE
  /review-prd #$ISSUE
  /review-test-plan #$ISSUE
```
→ **STOP**

### Step 2: Detect Review Type and Recommendation
From the matched comment, extract:
- **Review type** (Issue, Proposal, PRD, or Test Plan)
- **Review number** (from `#N` in header)
- **Recommendation** (from `### Recommendation` section)
| Recommendation | Action |
|----------------|--------|
| Ready for work / Ready for implementation / Ready for backlog creation | Report "Already ready — no action needed." → **STOP** |
| Needs minor revision | Proceed to Step 3 |
| Needs revision | Proceed to Step 3 |
| Needs major rework | Proceed to Step 3 |

### Step 3: Parse Findings
Parse findings based on review type.

#### Issue Reviews (emoji-based)
Extract lines from `### Findings` section matching these patterns:
| Marker | Status | Action |
|--------|--------|--------|
| `✅` | Passed | Skip — no action needed |
| `⚠️` | Needs attention | Queue for resolution |
| `❌` | Missing/incorrect | Queue for resolution |
**Extract the criterion name** from each finding line (text after the emoji and optional bold marker).
Example parsing:
```
- ⚠️ **Title** — Doesn't reflect both changes
  → Finding: "Title needs rewording"
  → Detail: "Doesn't reflect both changes"

- ❌ **Priority** — Not set
  → Finding: "Priority not set"
  → Detail: "Not set"
```

#### Proposal Reviews (section-based)
Extract items from **Concerns** and **Recommendations** sections:
- Each bullet under `**Concerns:**` → queue for resolution
- Each bullet under `**Recommendations:**` → queue for resolution
- Items under `**Strengths:**` → skip (positive findings)

#### PRD Reviews (section-based)
Same parsing as Proposal Reviews, plus:
- Check `**Test Plan:**` field — if "Not found", queue as finding

#### Test Plan Reviews (section-based)
Same parsing as Proposal Reviews — extract Concerns and Recommendations bullets.
- Coverage gaps appear as bullet-point concerns (e.g., "Story 2.3 AC ... has no corresponding test case")
- Check `### Coverage Summary` for overall coverage status

### Step 4: Classify Findings
For each queued finding, classify as **auto-fixable** or **needs-user-input**.

#### Auto-Fixable Findings
| Finding Pattern | Auto-Fix Strategy |
|-----------------|-------------------|
| Priority not set / Priority missing | `gh pmu move $ISSUE --priority p2` (suggest default, ask confirm) |
| Missing labels | `gh issue edit $ISSUE --add-label {label}` |
| Body format doesn't match template | Reformat body to canonical template |
| Missing acceptance criteria section | Add AC section skeleton to body |
| Missing reproduction steps (bug) | Add template section to body |
| Test Plan not found (PRD) | Flag for user — cannot auto-generate |

#### Needs-User-Input Findings
| Finding Pattern | User Action |
|-----------------|-------------|
| Title needs rewording | Propose new title, ask user to accept/edit |
| Description insufficient | Ask user to provide additional detail |
| Scope not well-defined | Ask user to clarify scope boundaries |
| Success criteria not measurable | Ask user to refine criteria |
| Value proposition unclear | Ask user to articulate value |
| Any subjective quality judgment | Present the finding, ask user to resolve |

### Step 5: Resolve Findings
Process findings in two passes: **auto-apply unambiguous fixes** first, then **ask the user only about subjective resolutions**.
**Pass 1: Auto-Apply (no user question)**
For auto-fixable findings from Step 4, apply the fix immediately and report the action. Do NOT use `AskUserQuestion` for these — the whole point is autonomous resolution.
**Safe auto-fixes (apply immediately):**
| Finding | Auto-Apply Action |
|---------|-------------------|
| Priority not set | `gh pmu move $ISSUE --priority p2` — apply default, report |
| Missing labels | `gh issue edit $ISSUE --add-label {label}` — add inferred label, report |
**Body-modifying auto-fixes (confirm first):**
These changes modify the issue body content. Show a preview and ask for confirmation before applying:
| Finding | Auto-Fix Action | Confirmation |
|---------|-----------------|--------------|
| Missing AC section | Add `## Acceptance Criteria\n- [ ] TODO` skeleton | "Add AC section skeleton to body? (y/n)" |
| Missing repro steps (bug) | Add `## Steps to Reproduce\n1. \n2. \n3. ` skeleton | "Add repro steps template to body? (y/n)" |
| Body format issues | Reformat body to canonical template | Show diff preview, "Apply reformatting? (y/n)" |
**Why confirm body changes:** Metadata changes (priority, labels) are additive and easily reversed. Body changes overwrite content and are harder to undo — a confirmation gate prevents unintended loss of user-crafted text.
Report each auto-applied fix immediately:
```
Auto-resolved:
  ✓ Priority set to P2 (default)
  ✓ Added label: enhancement
  ✓ Added AC section skeleton to issue body (confirmed)
```
**Pass 2: Ask User (subjective resolutions only)**
For needs-user-input findings, use `AskUserQuestion`:
```javascript
AskUserQuestion({
  questions: [{
    question: `Finding: ${findingDescription}\nReviewer note: ${reviewerNote}`,
    header: "Resolution",
    options: [
      { label: "Accept suggestion", description: `Apply suggested change: ${suggestedFix}` },
      { label: "Provide alternative", description: "Specify your own resolution for this finding" },
      { label: "Skip", description: "Leave this finding unresolved" }
    ],
    multiSelect: false
  }]
});
```
- **Accept suggestion** → Apply the suggested resolution, report: `"✓ {change applied}"`
- **Provide alternative** → *Conditional follow-up:* Ask conversationally what the user's alternative is, then apply
- **Skip** → Report: `"⊘ Skipped: {finding}"`
**For title rewording specifically:**
1. Read the current title and issue body
2. Propose a new title based on the issue content
3. Present via AskUserQuestion with: "Accept suggested title", "Edit title", "Skip"
4. If "Edit title" selected, ask conversationally for the new title
5. Apply: `gh issue edit $ISSUE --title "{new title}"`
**Progress reporting:**
```
Resolving 4 findings from Issue Review #1...

  Auto-resolved (2 findings):
    ✓ Priority set to P2 (default)
    ✓ Added AC section skeleton

  User input needed (2 findings):

    1/2 ⚠️ Title needs rewording
        → [AskUserQuestion: Accept suggestion / Provide alternative / Skip]
        → ✓ Title updated

    2/2 ⚠️ Description could be more detailed
        → [AskUserQuestion: Accept suggestion / Provide alternative / Skip]
        → ✓ Description updated

All findings resolved.
```

### Step 6: Re-Run Review
After all findings are resolved, re-run the appropriate review command:
| Review Type | Re-Run Command |
|-------------|----------------|
| Issue | `/review-issue #$ISSUE --force` |
| Proposal | `/review-proposal #$ISSUE --force` |
| PRD | `/review-prd #$ISSUE --force` |
| Test Plan | `/review-test-plan #$ISSUE --force` |
**Invoke the command** using the Skill tool with `--force` to bypass the early-exit gate (the issue may still have the `reviewed` label from a prior cycle). The re-review will verify all findings are resolved and post an updated review comment.
**Pending label cleanup:** If the re-review recommendation starts with "Ready for" (passes), the re-invoked review command will apply `reviewed` and remove `pending` via its own Step 5.5/6.5. If the re-review still does NOT start with "Ready for", the `pending` label remains (the review command applies it). No additional label logic is needed here — the re-invoked review command handles the label swap:
```bash
# Handled by the re-invoked review command's Step 5.5/6.5:
gh issue edit $ISSUE --add-label=reviewed --remove-label=pending   # if Ready for
gh issue edit $ISSUE --add-label=pending --remove-label=reviewed   # if not Ready for
```
Report final status:
```
/resolve-review #$ISSUE complete.
  Findings resolved: N
  Re-review: [recommendation from new review]
```

## Error Handling
| Situation | Response |
|-----------|----------|
| Issue not found | "Issue #N not found." → STOP |
| No review comment found | "No review found. Run /review-issue, /review-proposal, /review-prd, or /review-test-plan first." → STOP |
| Already ready for work | "Already ready — no action needed." → STOP |
| `gh pmu` command fails | "Failed to update issue: {error}" → STOP |
| `gh issue edit` fails | "Failed to update issue: {error}" → STOP |
| User declines all fixes | "No changes made. Review findings remain unresolved." → STOP |
| Re-review finds new issues | Report — user can run `/resolve-review` again |
**End of /resolve-review Command**
