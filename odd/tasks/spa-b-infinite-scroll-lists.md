# spa-b — cursor infinite scroll for all lists (+ run-detail stepper)

## Objective
All list views use cursor (keyset) infinite scroll with skeletons + lightweight
row rendering. Backend gains keyset pagination; frontend gains a reusable
infinite-list composable. Also: run-detail Steps → vertical timeline/stepper.
User chose "todo junto".

## Cursor contract (NON-BREAKING, uniform)
- `GET /routines?limit&cursor[&archived]` (EXISTING — must stay backward compatible;
  old `packages/spa` reads it as a bare array) and NEW `GET /reviews?limit&cursor&archived`
  (global, across all repos) BOTH return:
  - Body: the JSON **array** of items (unchanged shape; runs still `RoutineRun[]`).
  - Header: **`X-Next-Cursor`** — opaque cursor for the next page, empty/absent = end.
- Cursor = base64url of `"<created_at RFC3339Nano>|<id>"`. Keyset:
  `WHERE (created_at < c.ts) OR (created_at = c.ts AND id < c.id)`
  `ORDER BY created_at DESC, id DESC LIMIT ?`. Default limit 30, clamp ≤ 100.
- Frontend reads body array + `X-Next-Cursor` header → `{ items, nextCursor }`.

## Waves
- **A (parallel)**: W_backend (Go: runs keyset + new global reviews endpoint + cursor util + tests) ‖ W_stepper (run-detail Steps → timeline).
- **B (after A backend committed)**: W_infra_runs — `client.requestPage` (reads X-Next-Cursor) + `useInfiniteList` composable + runs list infinite scroll + lazy `⋯` menu + skeletons + polling reconcile.
- **C (after B committed)**: W_reviews — reviews store → new global endpoint via `useInfiniteList`; ReviewsListSection infinite scroll; keep filters (client-side over loaded pages, v1).

## Decisions / notes
- Runs polling + infinite scroll: poll refetches page 1 (no cursor) and MERGES by id
  into the accumulated list (update in place, prepend new); never resets scroll or
  drops loaded pages.
- Filters (runs/reviews) stay client-side over loaded items for v1 (only sees loaded
  pages) — server-side filtering is a follow-up.
- Virtualization: user said optional → DEFERRED (infinite scroll + lazy menus +
  page size handle the perf). Follow-up if needed.

## Tasks
- [ ] A1 backend: runs keyset + reviews global endpoint + cursor util + Go tests.
- [ ] A2 run-detail Steps → vertical timeline/stepper component.
- [ ] B1 useInfiniteList + client.requestPage + runs list infinite scroll + lazy menu.
- [ ] C1 reviews list infinite scroll on the new global endpoint.
- [ ] Verify per wave (front: type-check+tests+build; back: go build+test), compact commits, screenshots.

## Progress — CODE COMPLETE (all committed, all green)
- A: 37d8e9d run-detail timeline · 477a9af backend keyset + global GET /reviews.
- B: 2d02140 runs list cursor infinite scroll (useInfiniteList + requestPage + lazy
  menu confirmed via reka Presence + polling merge).
- C: 3a2045d reviews list cursor infinite scroll on the new global endpoint.
- Verify: frontend type-check 0, 293 tests, build 0; backend go build 0, 572 tests.
- PENDING: e2e/screenshot needs :8082 REBUILT with 477a9af (old binary lacks the new
  endpoints + X-Next-Cursor). Coordinate restart with user.
