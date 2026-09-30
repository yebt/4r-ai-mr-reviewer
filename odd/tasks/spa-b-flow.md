# spa-b — Flow module (per-repo workspace)

## Objective
Rebuild the old UI's Flow in spa-b: `/flow` repo picker + `/flow/:repoId`
workspace (MRs · Reviews · Actions) from which the user launches reviews,
releases (dev / main) and AI-drafted MRs. Settings → Repos stays config-only.
User decision: "Flow como en la vieja" (primary nav entry "Flow").

## Evidence
Mapped old spa Flow (pages/flow/index.vue, [repoId].vue, ReleaseModal,
CreateMergeRequestModal, ReviewLaunchModal, RoutineRunDetail). Backend has every
endpoint (verified shapes): GET /repos/{id}/merge-requests, POST /reviews,
POST /repos/{id}/routines/release, /release-main, GET /repos/{id}/routines/preview-tag,
GET /repos/{id}/branches, POST /repos/{id}/merge-requests/generate, POST /repos/{id}/merge-requests,
GET /repos/{id}/reviews, GET /repos/{id}/routines, GET /repos/{id}/preflight.
No backend gaps — all work is spa-b client.

## Tasks
- [x] F0 — Safety parity: "Merge now" in `pages/runs/[id].vue` gets a second
  danger ConfirmDialog naming target branch + tag (old RoutineRunDetail.vue:44-58).
- [x] S1 — Flow shell: nav entry, `/flow` searchable repo picker w/ attention
  badges, `/flow/:repoId` header + repo switcher + tabs (?tab=) MRs · Reviews ·
  Actions, MR list (GET merge-requests), repo-scoped reviews + runs tabs.
- [x] S2 — Launch review dialog (POST /reviews; provider/model/mode).
- [x] S3 — Release (dev) + Release to main dialog: bump, live preview-tag,
  emojis (DEFAULT thumbsup+seedling, +custom), includeDev, branch pickers,
  removeSourceBranch / mergeWhenPipelineSucceeds (per-flow defaults).
- [x] S4 — New MR dialog: branch pickers (searchable), voice profile, provider/model,
  AI generate (read-only) → editable → create (GitLab write), dirty-close guard.
- [x] S5 — repo settings in workspace header (provider/model assign, webhook,
  preflight permissions report), reusing repos-module components (user: "siguiente fase").
- [x] F1 — "Show archived" toggle in repo Reviews/Actions tabs (?archived=1).
- [x] F2 — client-side cap of 10 reaction emojis in ReleaseDialog (backend maxEmojis=10).
- [x] F3 — reviews list lazy row menu (same pattern as runs ce74c35).
- [x] P1 — Ctrl+K command palette: search repos and jump to `/flow/:repoId` (user request).

## Route / checks
Delegated direct writers per slice (F0 ‖ S1 parallel, disjoint files).
TDD trimmed (repo convention): unit-test pure helpers, keep specs green.
Per slice: type-check + test:unit + build, real-browser check, compact commit.

## Progress
- F0 DONE 9eccc9d — Merge now behind danger ConfirmDialog naming source→target + tag.
- S1 DONE 5ff6805 — nav Flow, /flow picker (attention from cached recent pages —
  limitation documented), /flow/:repoId tabs (?tab= sync, survives reload), MR list,
  repo-scoped reviews/runs (+polling). Browser-verified 1280 + 390: no overflow,
  0 console errors; empty states correct (0 open MRs / only-archived reviews match API).
  Fixed repo switcher wrapping (w-48 → w-full sm:w-64). 304 tests, type-check 0, build 0.
- Follow-up noted: repo tabs hide archived reviews/runs (backend supports ?archived=1).
- S2 DONE d4825b5 — Review button per MR → ReviewLaunchDialog (provider/model/mode
  defaults from repo → default provider; tested helper reviewLaunch.ts), POST /reviews →
  navigate to review. Backend does NOT dedupe manual launches → dialog warns when the MR
  has a pending/running review ("Start another review"). Browser-verified with mocked MR
  + blocked POST (no real review fired), 1280 + 390. 72 tests flow/reviews, type-check 0.
- S3 DONE b25ce16 — ReleaseDialog dev/main + Combobox atom + release.ts (27 tests).
  Browser-verified with ALL writes intercepted: payloads exact (dev MWPS true, main false,
  removeSourceBranch false, emojis thumbsup+seedling), outward-effects copy shown, inline
  error on blocked submit, 1280+390 no overflow, 0 console errors. Preview real (4.59.0).
  Parent fix: main-flow preview waits for resolved branches (was 1 wasted GitLab call; on
  repos without `main` e.g. sensei/master it now asks to pick branches instead of erroring).
  143 tests, type-check 0. Pre-gate GitLab writes: dev react+approve; main create_mr,
  approve, react (service.go step ledgers).
- Not enforced client-side: backend emoji cap 10 (surfaces as inline 400).
- S4 DONE bc49f6e — NewMergeRequestDialog (branch Combobox, target default development>main>master,
  ready voice profiles, provider/model via reviewLaunch helpers, Generate read-only w/ overwrite
  confirm, dirty-close "Discard this draft?", Create = GitLab write → invalidates repo MR list).
  newMergeRequest.ts 21 tests; 377 full suite (writer), type-check 0, build 0. Browser-verified with
  generate MOCKED (no tokens) + create BLOCKED, 1280+390, 0 console errors. Backend has no dedupe for
  an existing MR on the same branch pair — GitLab's own 400 surfaces inline.
- Core Flow (S1–S4) COMPLETE.
- P1 DONE 4e254ec — command-source registry (shared/composables/useCommandSources.ts, filter/cap
  tested); Flow registers "Repositories" group → /flow/:id. Browser: "deshi fr" + Enter lands on
  deshi front; arrows cross groups; 0 errors. Layering kept (design system imports no modules).
- S5 DONE (see git log) — gear menu: Provider & model (RepoForm reused unmodified), Webhook
  (WebhookDialog reused unmodified), Check permissions (new PreflightDialog + tested preflight.ts).
  Browser 1280+390 with writes blocked (none fired); live preflight deshi front: Owner, 5 ok.
  393 tests, type-check 0, build 0.
- Style: 74ffe0b dropped side-tab border on release preview (impeccable hook).
- FEATURE COMPLETE. /skills skipped (user).
- F3 DONE 62a10ec — reviews list lazy row menus + lifted Discard confirm. Browser found a real bug:
  Enter on a row's ⋯ navigated to the review (row @keydown.enter caught the bubbling key) — fixed
  with @keydown.stop on the actions wrapper; verified 30 triggers/0 menus at rest, Enter opens,
  Esc returns focus, Discard confirm opens (cancelled), 0 writes.
- F1+F2 DONE (see git log) — Webcloster: reviews 0 → 7 archived (dimmed + badge); actions 29 →
  "No archived runs for this repository."; release emoji cap: 10 selected, Add disabled, hint
  shown. 180 tests flow/reviews/runs, type-check 0, build 0. 0 writes, 0 console errors.
