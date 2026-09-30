# spa-b reviews — publish findings (slice 1)

## Objective
Let the reviews detail page publish findings + the summary to the live GitLab MR.
Slice 1 of the publish/humanize feature (user chose incremental). Humanize is a
separate later slice.

## Problem / Why
`packages/spa-b` has NO publish UI (the detail page's own comment says
"Humanize/publish are deferred"). The backend endpoint exists and posts real MR
comments — high-consequence, irreversible — so every publish is confirm-gated.

## Backend contract (verified)
`POST /reviews/{id}/publish` body `{all?, indices?[], includeSummary?, summaryOverride?, findingOverrides?[{index,text}]}`
→ 200 `{status:"published"}`; errors → 502. Semantics:
- Per-finding: `{indices:[index], includeSummary:false}`.
- Summary only: `{indices:[], includeSummary:true}`.
- Publish all: `{all:true}` → posts only not-yet-published findings + the summary
  if not yet published (idempotent). Finding.published / review.summaryPublished
  are the durable "already posted" flags.
- Precondition: review.status must be `done`.

## Scope (slice 1 — NO multi-select, NO humanize, NO overrides)
- Per-finding "Publish" button (only when `!finding.published`; "Published" badge otherwise).
- Summary "Publish" button (only when `!summaryPublished`; badge otherwise).
- Toolbar "Publish all" (disabled when nothing unpublished).
- Every action behind `ConfirmDialog` ("posts to the live MR, can't be undone").
- Only shown when review.status === 'done'.
- After publish, refresh the detail query so `published` flags flip reactively.

## Constraints
- Artifacts in English. TDD recortado: test the pure gating helper only.
- Type the full `PublishSelection` (incl. overrides) for forward-compat, but slice 1
  only uses all/indices/includeSummary.

## Tasks
- [x] T1 — `src/modules/reviews/publish.ts`: `unpublishedFindingIndices`,
  `hasUnpublished`; 6 trimmed tests. Exported via barrel.
- [x] T2 — `api.publishReview(id, selection)` + `PublishSelection`; `store.publish`
  mutation (toast "Published to the MR" + invalidate `['reviews']`).
- [x] T3 — `[id].vue`: per-finding / summary / "Publish all" buttons, each
  confirm-gated, done-status guard, `pendingPublishTarget`, `refetch()` after success.
  Published state shows a success Badge.
- [x] T4 — Verified: type-check 0, 264 tests (6 new), build 0. Commit b224e4a
  (compact, no author). Screenshot pending capture.

## Route
Delegated-direct: one writer (explore-then-write). Runner `bun run test:unit`.

## Acceptance
- Publish a finding / the summary / all, each confirm-gated; buttons reflect
  published state after refetch. type-check 0, build 0, unit tests green.

## Progress
- Slice 1 DONE (commit b224e4a). type-check 0, 264 tests, build 0. Branch not pushed.

## Slice 2 — humanize (IN PROGRESS)
Rewrite summary + findings in a ready profile's author voice; publish the chosen
version.

### Backend (verified)
- `POST /reviews/{id}/humanize` `{profileId, target:"summary"}` → `{summary}`;
  `{profileId, target:"finding", index}` → `{issue,why,fix}`. Persisted.
  Preconditions: review done + profile `styleGuideStatus==='ready'` (else 409);
  index OOR → 400; unknown review/profile → 404; LLM/parse → 502.
- `GET /reviews/{id}/humanizations` → `{summary:[{summary}], findings:{[idxStr]:[{issue,why,fix}]}}`
  (accumulated runs, in tab order).

### Override reassembly (safety-critical — matches server publish.go)
- Finding override = `buildFindingBody(finding, parts)`:
  `**[{DIM} · {SEV_UPPER}]** {issue}\n\n` + (`**Why:** {why}\n\n` if why) +
  (`**Suggested fix:** {fix}\n` if fix) + (`\n_Blocking._` if blocking).
  DIM map: risk=R1 Risk, readability=R2 Readability, reliability=R3 Reliability,
  resilience=R4 Resilience. (mirrors formatFinding)
- Summary override = the RAW humanized summary text (old UI behavior, no header).
- Only send an override when the active tab != Original.

### Tasks
- [x] S2-T1 — `humanize.ts` (`DIMENSION_LABELS`, `buildFindingBody`=byte-for-byte
  formatFinding, `ORIGINAL`) + 10 tests.
- [x] S2-T2 — `api.humanizeFinding/humanizeSummary/getHumanizations` + types
  (FindingHumanized/SummaryHumanized/Humanizations).
- [x] S2-T3 — `useReviewHumanize`: ready-profile picker (pre-seed repo default →
  first ready, sticky), humanize fns w/ per-finding pending, tabs (GET + local
  append), active-tab state, override builders.
- [x] S2-T4 — `[id].vue` + `HumanizeTabs.vue`: profile Select, per-card Humanize +
  tab strip + active-version body, Humanize-all, overrides merged into the 3
  publish handlers (incl. publish-all → respects active tabs).
- [x] S2-T5 — Verified: type-check 0, 274 tests (10 new), build 0. Parent
  re-checked buildFindingBody vs server + the publish merge. Commit 33caa59
  (compact, no author). Screenshot pending.
