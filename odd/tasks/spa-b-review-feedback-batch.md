# spa-b — feedback batch (bugs + UX)

## Objective
Address the user's batch on the new spa-b UI: 3 bugs + several UX upgrades on
reviews, runs, the shell, and the add-repo form.

## Diagnosed (no code here)
- **Humanize 502**: NOT frontend. Humanize uses the DEFAULT provider
  (`s.providers.Default`); the default is "GeminiTest" with an empty `model`, so
  the backend errors `no model set on default provider` → masked as 502. Fix is
  config: set a model on GeminiTest or make a configured provider default.
  (Optional backend follow-up: surface that error as a clear 4xx, not 502.)

## Items
- [ ] B1 desborde — finding text (long unbroken strings, e.g. `pak_000…`)
  overflows the card. `break-words`/`overflow-wrap:anywhere` on finding text +
  file:line. (reviews detail)
- [ ] B2 runs delete stuck — the `⋯` DropdownMenu stays open and the Delete
  ConfirmDialog overlaps/gets stuck. Fix the dropdown+confirm nesting in
  `RunsListSection.vue`. (runs)
- [ ] B3 sidebar scroll — should be independent from the main content scroll
  (desktop shell). (shell)
- [ ] U1 copy code line/reference — add a copy action per finding (file:line +
  the finding markdown), like old spa's `buildFindingMarkdown`. (reviews detail)
- [ ] U2 reviews detail structure + score visual — better card hierarchy + a
  visual ScoreMeter for score/recommendation. (reviews detail)
- [ ] U3 humanize-all modal — a "Humanize all" button that opens a modal with a
  profile picker + Cancel; profile selection lives there (not an always-visible
  header select). (reviews detail)
- [ ] U4 reviews list — repo info on cards + repo/status filters (like runs).
  (reviews list — WAVE 2, shares module with reviews-detail)
- [ ] U5 add-repo search — RepoForm URL field → search account projects via
  `GET /accounts/{id}/projects`. (repos)

## Waves (parallel, file-disjoint)
- Wave 1 (concurrent): W1 reviews-detail (B1,U1,U2,U3) · W2 shell+runs (B2,B3) ·
  W3 repo-search (U5).
- Wave 2: U4 reviews list (after W1 — shares reviews module).

## Route
Delegated-direct, parallel writers on disjoint file sets. Parent integrated
verify per wave (type-check + test:unit --run + build). Compact commits, no author.

## Progress
- 502 diagnosed + reported (config, GeminiTest default has no model).
- Wave 1 DONE + committed: C1 2aeeec0 (B1,U1,U2,U3 review detail), C2 7b744bc
  (B2,B3 sidebar+runs delete), C3 5ee5b92 (U5 repo search), C4 b989560
  (notifications add-rule alignment — extra img #33 fix). Combined verify:
  type-check 0, 281 tests, build 0.
- Wave 2 DONE + committed: C5 221a817 (U4 reviews list repo name + repo/status
  filter bar). Combined verify: type-check 0, 282 tests, build 0.
- ALL items done. Commits: 2aeeec0, 7b744bc, 5ee5b92, b989560, 221a817.
- Pending: visual screenshot verification of the whole batch (in progress).
