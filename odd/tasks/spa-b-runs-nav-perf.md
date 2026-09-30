# spa-b — Runs navigation perf

## Problem (measured, real browser, load 0.9)
Warm settings→runs (0 network requests, data cached) blocks the main thread
~172ms in dev / ~106ms in a production build (`vite preview`); first visit
~512ms (prod) incl. the lazy route chunk. API is not the cause
(`/api/routines` 15ms).

CPU profile (warm nav, dev): Vue mount 95ms (dev checks), Vue devtools
`startMeasure/endMeasure` `now()` ~24ms (dev-only), `formatDateTime` 10ms
(`toLocaleString(undefined, opts)` builds an Intl formatter per call),
per-row Reka `DropdownMenuRoot`+Trigger mounted for every row even when never
opened, virtualizer measurement ~10ms.

## Tasks
- [x] T1 — `runs/format.ts`: cache one module-level `Intl.DateTimeFormat`
  for `formatDateTime` (same output). Keep format.spec green.
- [x] T2 — `RunsListSection.vue`: lazy row action menu — a plain `⋯` button
  per row; mount the Reka DropdownMenu only for the row whose menu is open
  (`activeMenuId`), open immediately, unmount on close. Preserve Archive /
  Cancel (nested ConfirmDialog) / Delete (lifted confirm) and a11y.
- [x] T3 — warm lazy route chunks on idle after app start (router route
  component loaders via requestIdleCallback).
- [x] T4 — parent re-measure dev + prod (nav.mjs), full verify, commit.

## Route
Delegated direct: one writer (T1–T3, 3+ files). Parent measures (T4).
TDD: trimmed (repo convention) — keep specs green; unit-test pure helpers.

## Progress
- DONE, commit ce74c35. Measured before→after (real browser):
  warm block 172→111ms dev, 106→67ms prod; click→rows 236→162 dev, 167→135 prod;
  first visit 642→511 dev, 512→431 prod, route chunks 18/12→0 (prefetched).
  Browser check: 16 rows → 0 Reka menus mounted; keyboard open, focus in menu,
  Esc closes, focus returns to row button; 0 console errors. runs+core 61 tests,
  full suite 299 (writer), type-check 0, build 0. Cancel confirm lifted like Delete.
- Remaining dev cost is mostly dev-only (Vue devtools startMeasure/performance.now,
  dev prop validation). Follow-ups: make vite-plugin-vue-devtools opt-in; apply the
  same lazy row menu to the reviews list.
