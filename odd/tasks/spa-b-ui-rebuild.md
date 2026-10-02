# Feature: spa-b UI rebuild (fresh visual world)

Project: 4r-ai-mr-reviewer · Package: `packages/spa-b` · Branch: (feature branch, TBD)
Created: 2026-09-22

## Objective
Rebuild the 4R web UI from scratch in `packages/spa-b` as a fresh, responsive,
component-driven system. Milestone 1: **login + a unified configuration/settings
view + the responsive base shell**. Stack fixed by the user: **Reka UI (headless)
+ Tailwind v4 + atomic design**, with a **fresh visual identity** (NOT a port of
the current UnoCSS dark/lime identity).

## Why
The current `packages/spa` is desktop-first with a bolted-on mobile tab bar, a
hand-rolled UI kit, no design system, and six disconnected settings pages. spa-b
is a clean slate to build a real atomic design system on headless primitives with
two deliberate UI lines (mobile + desktop) and researched navigation patterns.

## Scope (Milestone 1)
- Foundation: Reka UI + Tailwind v4 + icons; theme/token layer; atomic-design base.
- Responsive base shell: desktop sidebar + content; mobile bottom-nav / command
  palette / mobile-specific patterns. Two UI lines with shared logic.
- Login page wired to the REAL `/auth/*` contract (incl. auth-disabled fail-open).
- Unified Settings shell (NEW IA) grouping the config areas; wire the first section.

Out of scope for M1 (later milestones): full port of every settings section, Flow,
Reviews, Repos detail, Profiles distill, Telegram, notifications rules.

## Constraints
- TDD: **Trimmed TDD ("recortado")** — user override 2026-09-22 (supersedes strict for
  this feature): focused tests for valuable LOGIC/behavior only (RED→GREEN on critical
  paths), skip exhaustive/presentational tests, move fast. Runners: `vitest` +
  `@vue/test-utils` (unit), `playwright` (e2e/render).
- **Parallelize** (user directive): independent tasks run as concurrent writers on
  DISJOINT file paths; parent runs the integrated type-check/build after each wave.
- **Research before implementing** (user directive): UI/UX interaction patterns (command
  palette, bottom sheet, mobile nav, drawer, toasts, forms…) are researched into
  `packages/spa-b/docs/ui-patterns.md` BEFORE the components that use them; components are
  built as reusable design-system pieces.
- Artifact language: English (code, comments, UI copy, tests).
- Do NOT commit/push without explicit user request (per repo policy).
- Design authority: `impeccable` skill (new-work playbook) drives the visual world.
- Real API contract must be honored exactly (see "Contract reference" below).

## Design decisions (log)
- [decided] Aesthetic = "Radix-minimal, dual-theme, usability-first" (user): minimalist,
  light AND dark both first-class, Radix-Themes-style token scales (neutral gray + 1 accent,
  12-step, accessible), Reka UI (the Radix approach ported to Vue), soft radii/hairline
  borders/subtle depth, calm motion. Code/diffs via a highlight lib (Shiki). "Highly and
  deeply usable tool" — usability over decoration.
- [decided] Stack = Reka UI + Tailwind v4 + atomic design (user-fixed).
- [decided] Fresh visual world, not a port of spa's UnoCSS lime/dark identity (user).
- [decided] spa-b CENTRALIZES UX directly (user): one coherent UX hub — unified shell,
  central navigation (command palette + shell as the single entry to everything),
  consistent patterns everywhere. Not the scattered standalone pages of today.
- [note] Existing spa uses UnoCSS Wind4, not Tailwind — flagged; irrelevant since fresh world.

## Contract reference (from map, verbatim-critical)
- `GET /auth/status` → `{ authEnabled: boolean, authenticated: boolean }` (boot probe; fail-open on network error).
- `POST /auth/login` body `{ password }` → `{ authenticated }` (+ httpOnly `air_session` cookie). 401 invalid, 429 rate-limited (10/min/IP).
- `POST /auth/logout` → `{ authenticated:false }` (clears cookie).
- `credentials: 'same-origin'`; base = `import.meta.env.VITE_API_URL ?? '/api'`.
- Any 401 on non-`/auth/*` → global onUnauthorized → clear state → redirect `/login?redirect=<path>` (sanitized: in-app absolute, never `//host`, never `/login*`).
- Settings areas to eventually rebuild: Accounts, AI Providers (richest DTO: kind/baseUrl/model/apiKey(write-only)/isDefault/temperature/models[]), Telegram, Profiles, Repos, Notifications+Vault, Skills(read-only).

## Tasks
- [x] F1 — Reka UI 2.10.5 + Tailwind v4.3.3 (+@tailwindcss/vite) + Shiki 4.4.3 + @iconify/vue installed & wired (vite.config, main.css `@import "tailwindcss"`, no-FOUC theme script). Verified: type-check exit 0, build exit 0, lint clean. (delegated writer af1cd72)
- [x] F2 — Tokens in src/shared/assets/tokens.css: gray-12 + accent-12 (indigo) Radix-style scales, semantic aliases (--bg-app/-panel/-panel-raised, --line/-subtle/-strong, --text/-muted, --accent*, status {success,warning,danger,info}{bg,solid,text}), radius/elevation/motion/type — all via one `@theme inline`; light+dark hand-tuned WCAG AA. DESIGN.md is the source of truth. Shiki fine-grained core (992KB→163KB gz, no WASM). (delegated writer af1cd72)
- [x] R0 — UI/UX patterns playbook DONE → `packages/spa-b/docs/ui-patterns.md`. Key: Reka has native `Drawer` (snap points) → bottom sheet uses it, not hand-rolled; command palette = `Combobox`+`useFilter` in `Dialog`; toasts = `Toast*`. NO primitive (must build): bottom tab bar, shell/sidebar grid, `FormField` wrapper, data-table/card reflow, skeletons. Pattern→primitive table in the doc. (research agent)
- [x] F3 — DONE (writer a6e3c7d, 14 tests). 12 atoms + Field molecule + barrel `src/shared/ui/design-system/index.ts`. Button(accent/soft/outline/ghost/danger, sm/md/lg, loading), Input/Textarea (inheritAttrs:false, attrs fall through for aria), Field (useId + aria-describedby/invalid backbone), Select/Checkbox/Switch (Reka), Text/Heading/Icon(@iconify)/Kbd/Badge(status)/Spinner. Gotchas fixed: Vue strips declared props from $attrs; Tailwind v4 needs literal class maps (no dynamic template classes). (writer)
- [x] S1+S2 — DONE (writer afd7a4f, 18 tests). AppShell (useMediaQuery STRUCTURAL swap desktop tree vs mobile tree, not CSS hide), AppSidebar, AppBottomNav (primary tabs + More = Reka Drawer bottom sheet), AppHeader (#actions slot), CommandPalette (Reka Dialog+Combobox+useFilter, ⌘K, Navigate+Actions groups, mobile full-bleed), useCommandPalette (createSharedComposable singleton + useMagicKeys), nav.ts (single source: Home/Reviews/Runs primary, Settings secondary; isNavItemActive/filterNavItems). App.vue bare-vs-shell. Preview → /design; minimal Home at /. (writer)
- [x] A1 — DONE (writer a2a2d57, 15/15 tests). Files: src/shared/api/client.ts (request<T>, ApiError, onUnauthorized), src/modules/auth/{api,errors,redirect,store}.ts + specs, LoginPage.vue placeholder. Guard in src/core/router/index.ts. Surface: useAuthStore{enabled,authenticated,ready; fetchStatus/login(throws)/logout}, errorMessage(e), safeRedirect(raw). A2 NUANCES: /login is registered IMPERATIVELY (router.addRoute → src/modules/auth/pages/LoginPage.vue, meta.public=true) — A2 edits THAT component in place, must NOT add src/pages/login.vue (dup route); login() throws→catch+errorMessage; A2 owns post-login safeRedirect(route.query.redirect). (writer)
- [x] A2 — DONE (writer a369f9c, 4 tests). LoginPage.vue edited in place: Field+Input(password)+Button(loading), try/catch auth.login → errorMessage, success → router.replace(safeRedirect(query.redirect)), clear+refocus on error (via Field id override, no atom hack), bare centered card dual-theme. (writer)
- [x] C1 — DONE (writer a17edbd, 4 tests). src/pages/settings.vue (/settings) + organisms SettingsLayout (Reka Tabs, overflow-x-auto scrollable strip on mobile) + SettingsSectionStub. 6 sections: Providers live (imports ProvidersSection) + 5 stubs. Parent fixed at integration: settings.vue undefined-index guard + replaced fragile node:fs static-source test with a REAL settings.vue mount (mocking @modules/providers). (writer + parent)
- [x] C2 — DONE (writer af00bd9, 7 tests). src/modules/providers/{types,api,store,index}.ts + components/{ProvidersSection,ProviderForm}.vue. Full CRUD + setDefault + testConnection + openrouter model browser. Handles blank apiKey=keep-on-edit, temperature→number|null coercion, state reconciled from server (no double-fetch). Exports `ProvidersSection` from @modules/providers. NOTE: flagged pre-existing lint debt in auth/store.spec.ts (vitest require-mock-type-parameters). (writer)

## Verification (per task)
- `bun run type-check` (vue-tsc), `bun run lint`, `bun run test:unit` (vitest), targeted `playwright` render where visual.
- Spot-check one reported command before delivery.

## Progress / evidence
- 2026-09-22: Explored scaffold + mapped current spa/backend contract. Aesthetic resolved (Radix-minimal dual-theme). Branch `feat/spa-b-ui-rebuild` created off main.
- 2026-09-22: F1+F2 DONE (writer af1cd72). Deps: tailwindcss@4.3.3 +@tailwindcss/vite, reka-ui@2.10.5, shiki@4.4.3, @iconify/vue@5.0.3 +@iconify-json/lucide. tokens.css + DESIGN.md + temporary preview index.vue. Verified: type-check exit 0 (parent re-run too), build exit 0, lint clean.
- 2026-09-22: Visual proof captured (Playwright, light+dark, desktop+mobile) — themes genuinely invert, Radix-style scales correct. DEFECT found + fixed: stray "Hi" scaffold text in src/core/App.vue:4 removed (inline). Screenshots sent to user for direction approval.

## Next step
- 2026-09-22: User APPROVED the visual foundation. Directives: trimmed TDD, parallelize, research-patterns-first.
- WAVE 1 LAUNCHED (parallel, disjoint paths): R0 patterns research (docs/ui-patterns.md) + F3 base atoms + A1 auth logic.
- WAVE 2 (after wave 1 + playbook): S1 centralized responsive shell + S2 ⌘K palette/mobile nav (cite playbook) ∥ A2 login (consumes A1 + atoms).
- WAVE 3: C1 unified settings shell + C2 Providers section.
- Parent integrates + runs full type-check/build after each wave. Preview index.vue removed when S1 lands.
- 2026-09-22: WAVE 1 integrated GREEN (type-check 0, build 0, 29/29 tests). Parent fixed a cross-file type error in client.spec.ts (unknown→ApiError cast) that isolated vitest missed. WAVE 2 LAUNCHED (parallel, disjoint): [shell+nav+⌘K palette] ∥ [login UI]. Known: 846KB pages chunk warning = temp preview (resolves when preview→/design + route splitting).
- 2026-09-22: WAVE 2 integrated GREEN (type-check 0, build 0, 51/51 tests). Milestone-1 core complete: foundation + atoms + auth + shell + ⌘K + login. 7 screenshots captured + reviewed (shell/palette/login light+dark) — sent to user. Palette + shell + login look strong.
  REVIEW NITS (open): (1) Vue DevTools floating pill in shots = dev-only (vite-plugin-vue-devtools), not our UI — ignore. (2) Accent Button uses DARK text on indigo — verify contrast vs white text in a polish pass (touches Button/token, do NOT edit mid-wave).
- 2026-09-22: WAVE 3 LAUNCHED (parallel, disjoint) per user "avanza": [C1 settings shell + Reka Tabs] ∥ [C2 providers module modules/providers/**]. Contract: settings page imports ProvidersSection from @modules/providers. Neither edits atoms/tokens/barrel/nav/router.
- 2026-09-22: WAVE 3 integrated GREEN (type-check 0, build 0, 62/62 tests) after 2 parent fixes (settings.vue undefined guard; settings.spec node:fs static check → real mount). **MILESTONE 1 COMPLETE**: foundation + 12 atoms/Field + auth(client/store/guard) + centralized shell + ⌘K palette + login + unified settings + providers CRUD. All ✓, uncommitted on feat/spa-b-ui-rebuild. Settings screenshots captured + sent (list + Add-provider dialog, light/dark/mobile).
- 2026-09-22: MILESTONE 1 COMMITTED — 4 compact conventional commits, NO author attribution (user: "en compacto y sin autor"), scoped to packages/spa-b, odd/ left untracked: 3cf2ac7 foundation / ef6c6d6 atoms+Field / a95587c auth+login / e2228be shell+palette+settings+providers.
- 2026-09-22: USER FEEDBACK → refinement pass DONE + committed as c56e7b7 `style(spa-b)`. (1) BACKDROP: centralized `.overlay` (main.css) + `--overlay` token (true black 60% + backdrop-blur 6px) on all 5 Reka overlays — also fixed latent bug (old backdrop used gray-12 which flips per theme → brightened dark-mode backdrop). (2) RADII: sm2/md4/lg6/xl8/full; rounded-full audited (all legit pill/circular). (3) ICONS: lucide-vue-next named imports (dynamicIconImports absent in v1.0.0), @iconify removed; name map handled upstream renames (more-horizontal→ellipsis, check-circle-2→circle-check, alert-triangle→triangle-alert, x-circle→circle-x). Verified full: type-check 0, 62/62, build 0. Re-screenshotted palette + dialog — backdrop blur+darken confirmed vs user reference. 5 commits total on feat/spa-b-ui-rebuild (3cf2ac7/ef6c6d6/a95587c/e2228be/c56e7b7).

## Round 2 (post-M1 user feedback, 2026-09-22)
Backend fact: Go API on :8082, auth OFF (authEnabled:false), serves at ROOT (/providers); /api/providers=404.
- [x] SETTINGS SECTIONS Accounts/Telegram/Profiles DONE + committed (fcdd816, feat). 3 parallel writers mirrored the providers colada pattern (optimistic+toast+skeleton), each own module src/modules/{accounts,telegram,profiles}. Parent wired settings.vue tabs. Integrated GREEN: type-check 0, 114/114 tests, build 0. Screenshotted vs REAL :8082 (Work account; deshi/internal telegram targets w/ Default badge; Edu/parchao/walo profiles w/ Ready badges). REMAINING config sections: Repos (webhook/branches/preflight/assign) + Notifications&Security (rules + vault master-key) — more complex, not yet built.
- [x] REVIEW + POLISH PASS (user: "pausa, pule y revisa todo — interacciones, flujos redundantes, cosas que no se acoplen"). Audit (read-only) → 2 parallel polish writers + parent fix. Committed 73dc4c6 (fix) + ac84d8c (refactor, net −72 lines). FIXED: (correctness) double-error-surfacing (forms inline-only, row actions toast), delete loading state (via ConfirmDialog), Profiles Samples a11y (Field). (coherence) accent+status button contrast AA-verified BOTH themes (accent-9 dark 58%→52%, white text 6.49:1; danger/success/info/warning solid tokens fixed), Reviews/Runs placeholder pages (killed 404 dead-ends), /design dev-gated in router, Home uses Heading, quiet ghost Delete (was loud solid-red on every row), Badge neutral consistent, Icon DEV warn. (redundancy) shared ConfirmDialog molecule + resolveErrorMessage util (−90+ dup lines). (polish) Spinner reduced-motion, toasts consistent ("X saved", "Redistilling…"), stale comments/DESIGN.md. GREEN: type-check 0, 116 tests, build 0.
- DEFERRED (flagged to user, not done): deep DRY refactor — createCrudColadaStore<T> factory + CrudSection scaffold + useCrudForm to collapse ~360 lines store/list/form boilerplate across 4 modules (medium risk, tests as safety net).
- Branch commits (13): 3cf2ac7,ef6c6d6,a95587c,e2228be,c56e7b7,2928d8c,688012f,c0c8093,947c930,fcdd816,73dc4c6,ac84d8c + (typed-router chore earlier).
- [x] R2-BUG — Blank "." provider rows = NO vite proxy → /api/* returned index.html → client accepted HTML string → Vue v-for iterated per-char. FIXED inline + committed (fix(spa-b)): vite.config server.proxy /api → VITE_API_TARGET||:8082 with /api strip; client.ts now throws on 2xx non-JSON. (parent)
- [x] R2-INFRA — DONE (writer a687e52, 8 toast tests). favicon.svg (from spa), nprogress (deps+router hooks+accent bar, unlayered CSS override), theme toggle (AppSidebar footer + AppShell mobile header), useToast({success,error,info,dismiss,toasts}) + ToastHost.vue mounted in App.vue, Skeleton atom+barrel, 404 [...path].vue (public, in-shell). CAVEAT TO VERIFY AT INTEGRATION: writer changed App.vue isBare + guard from meta.public===true to route.name==='login' (so 404 isn't bare) — MUST confirm the imperatively-registered /login route is NAMED 'login', else login renders in-shell. (writer)
- [x] R2-PROV — DONE (writer a2f48b4, 15 tests). store.ts on @pinia/colada: useQuery(['providers']) + 4 useMutation (create/update/remove/setDefault) with onMutate optimistic cache patch + snapshot rollback on error + invalidate on settled; toast calls live in the store (single source). Optimistic logic factored into pure tested helpers. ProvidersSection: Skeleton on first load, error+retry state, subtitle fix (no bare "·"). openrouter models = lazy query. (writer)
- [x] R2 INTEGRATED GREEN: type-check 0, build 0, 78/78 tests. Parent fixed a cross-file type error the isolated runs missed: infra writer used `route.name === 'login'` but /login is registered imperatively → not in typed-router RouteNamedMap union (TS2367). Changed App.vue isBare + guard isLoginRoute to `route.path === '/login'` (still distinguishes login from the public 404). 
- Re-screenshotting providers vs REAL :8082 backend (no mock) to confirm names load + 404 + toast, then commit R2 compactly.
- COMMIT CONVENTION (user pref): compact conventional commits, NO Co-Authored-By/attribution. OPEN: accent-button contrast nit (folded into refinement or later); future milestones = real /reviews /runs pages, wire 5 stub settings sections, toasts/skeletons, code-split 846KB chunk.
