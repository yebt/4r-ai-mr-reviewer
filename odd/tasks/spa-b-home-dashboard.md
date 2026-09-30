# spa-b Home dashboard

## Objective
Replace the placeholder Home (`packages/spa-b/src/pages/index.vue`) with a real,
fast dashboard that surfaces routine-run health and quick navigation.

## Problem / Why
Home is an explicit placeholder ("until the real dashboard lands"). It is the
landing route, so it must load fast. The tool's daily value is spotting runs
that need a human (blocked / awaiting confirmation) and jumping into them.

## Scope
- Consume only CHEAP existing data: `useRunsStore().runs` (one global
  `/routines?limit` call, already polling-aware) + `useReposStore().repos` count.
- NO backend changes. NO expensive reviews N-repo fan-out on landing (kept as a
  quick-nav card only; a cheap global reviews count is a possible follow-up).
- Atomic design: pure selectors live in the runs module; the page composes
  existing design-system atoms + `bg-bg-panel` panels (no Card atom exists).

## Constraints
- Runs store polls → `pausePolling()` on `onUnmounted` (same leak-guard as the
  list/detail pages).
- Deep-import stores (`@modules/repos/store`); barrels export components/types only.
- Artifacts in English. TDD recortado: test pure selectors only.

## Tasks
- [x] T1 — `src/modules/runs/dashboard.ts`: pure selectors `runStats(runs)`
  (trimmed to `{total, active, attention}` — dropped unused/mislabelled
  done/failed), `attentionRuns`, `recentRuns`. Exported via runs barrel.
  `dashboard.spec.ts`: 7 tests, green.
- [x] T2 — Rewrote `src/pages/index.vue`: stat tiles, Needs-attention queue +
  Recent-activity feed (both deep-link `/runs/{id}`), quick-nav, loading/empty/
  error states, `onUnmounted` polling teardown.
- [x] T3 — Verified: type-check 0, 258 unit tests, build 0. Commit b2663e4
  (compact, no author). Screenshot pending capture.

## Route
Delegated-direct: one writer (explore-then-write, 2+ files). Parent integrated
verification. Runner: `bun run test:unit`.

## Acceptance
- Home renders stat cards, attention list, recent activity, quick nav.
- Attention + recent rows deep-link to the run detail pages.
- Fast load (no reviews fan-out). type-check 0, build 0, all unit tests green.

## Progress
- DONE (commit b2663e4). Writer built T1/T2; parent trimmed `RunStats` to the
  3 rendered fields (removed the mislabelled `failed`=cancelled + unused `done`)
  and re-verified. type-check 0, 258 tests (7 new), build 0. Branch not pushed.
- Design decision: Home stays fast — only the cheap global `/routines` call +
  repos count. Reviews = quick-nav card only (no N-repo fan-out on landing). A
  cheap global reviews-count endpoint would be the follow-up to add review stats.
