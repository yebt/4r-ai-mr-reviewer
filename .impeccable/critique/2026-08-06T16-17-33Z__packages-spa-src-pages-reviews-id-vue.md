---
target: review detail view (/reviews/[id])
total_score: 26
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 2
timestamp: 2026-08-06T16-17-33Z
slug: packages-spa-src-pages-reviews-id-vue
---
# Critique — Review Detail Surface (`/reviews/[id]`)

Method: dual-agent (A: design-review · B: detector+evidence). Browser visualization skipped — no running instance (needs `make dev` + Go backend + config); source + detector only.

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 4 | Excellent phased polling + live reasoning; only gap is no `aria-live` |
| 2 | Match System / Real World | 4 | GitLab-native (`!iid`, discussion), 4R lens naming, deterministic score |
| 3 | User Control and Freedom | 2 | Publish to a real MR is irreversible with no undo; `Esc`/clear only in phone modal |
| 4 | Consistency and Standards | 2 | "Comment all" vs "Publish selected"; desktop inline vs phone modal; ad-hoc humanize tabs |
| 5 | Error Prevention | 2 | Delete/Discard confirm, but the frequent, team-visible publish fires unguarded on desktop |
| 6 | Recognition Rather Than Recall | 3 | Opaque `V1/V2` tab labels; global profile held in memory |
| 7 | Flexibility and Efficiency | 2 | No keyboard shortcuts, no range-select, no "select all blocking" |
| 8 | Aesthetic and Minimalist | 2 | ~20 controls above the list; two competing lime primaries; One-Signal Rule broken |
| 9 | Error Recovery | 4 | Exemplary error state: tokens, empty-response hint, raw output, retry-clones |
| 10 | Help and Documentation | 1 | No inline help / first-run guidance despite newcomer self-hoster being first-class |
| **Total** | | **26/40** | **Acceptable — concentrated fixes, not a rebuild** |

## Design Specificity Verdict — PASS (with a misplaced signature)

**LLM assessment:** Strongly authored for THIS product. R1–R4 lens filter chips with per-dimension counts, phased Risk/Readability/Reliability/Resilience `(n/4)` progress, GitLab vocabulary, blocking=flame severity edges, humanize-into-your-voice, deterministic score — none of this transplants to a generic tool. BUT the design system's own headline "4R readouts" (ScoreMeter gauge, RecommendationBar distribution) are absent from the core surface: `SummaryCard` renders the score as plain text `72/100`; the gauges are used only on `pages/index.vue`. The IA is unmistakably 4R; the instrument-panel *face* is missing from the one screen that most needs it.

**Deterministic scan:** Detector clean — **0 findings** on both the review surface and the whole `packages/spa/src` tree (exit 0). No design-system drift, no decorative/fake content, no anti-patterns. The only "hex in components" hit is a false positive (`&#8209;` HTML entity in AppSidebar). No TODO/FIXME/lorem/dummy in the review components. The clean detector *confirms* the specificity verdict: this is a coherent, product-specific system, not a template.

**Visual overlays:** none — no running instance available; no user-visible overlay was produced.

## Overall Impression

This is a strong, product-true surface with genuinely excellent waiting and failure states — and two real problems that cluster at the emotional peak and the safety-critical moment. The single biggest opportunity: the review's *peak* (score arrives) is under-celebrated as plain text while a built, CVD-validated gauge sits unused, and the review's *most consequential action* (publishing to a live teammate-visible MR) fires on desktop with no confirmation and no undo. Fix those two and the surface jumps a band.

## What's Working

1. **Async waiting UX (the best beat on the surface).** Phased `(n/4)` progress, an auto-expanding ReasoningPanel that follows the live phase, and desktop "notify when done" make a long job feel alive and safely abandonable.
2. **Failure diagnosability.** The error state surfaces tokens, an empty-response hint, captured reasoning, and collapsible raw model output with retry-clones. Heuristic-9 excellent.
3. **Product-true triage IA.** FindingsToolbar filters by R1–R4 with live per-dimension/severity counts and blocking=flame; `useFindingFilters` sorts blocking-first. The 4R spine, made operable.

## Priority Issues

**[P0] No-confirmation desktop publish.**
- Why it matters: A self-hoster can post findings onto a live MR — visible to their whole team, not un-postable from the UI — with one click/Enter, no undo. The app's most consequential action is its *least* guarded on its primary surface. Phone gets a focus-trapped confirm modal; desktop (the power user's environment) gets nothing.
- Fix: gate desktop "Comment all" / "Publish selected" behind a focus-trapped confirm (reuse the phone `Modal`/`useConfirm` path) showing target count, include-summary, and which MR; add an explicit success state (and, ideally, a brief undo window).
- Suggested command: `/impeccable harden`

**[P1] The peak has no instrument.**
- Why it matters: the deterministic 0–100 score is the product's trust anchor, and on the core surface it's bare text. DESIGN.md's signature readouts are thrown away exactly where they'd land hardest.
- Fix: put the already-built, CVD-validated `ScoreMeter` + `RecommendationBar` into `SummaryCard` as the results peak.
- Suggested command: `/impeccable bolder`

**[P1] Results-view overload + broken One-Signal Rule.**
- Why it matters: ~20 controls stack above the finding list, and two `btn-accent` lime primaries compete on screen; the repeated lime section ticks dilute the "one live signal" doctrine. Content that matters (the findings) is buried under chrome.
- Fix: collapse the desktop controls zone into a selection action-bar that appears only when `selected.length > 0` (mirror the phone sticky bar), demote repeated lime ticks to hairline, keep exactly one primary.
- Suggested command: `/impeccable distill`

**[P2] No keyboard triage path.**
- Why it matters: for a power user triaging dozens of findings, the whole loop is mouse-driven checkboxes — no `j/k` move, `x` toggle, shift-range, `Esc` clear, or `Enter` publish. A "triage tool" that can't be triaged from the keyboard.
- Fix: roving-focus list navigation + selection shortcuts + `Esc`-clears-selection on desktop.
- Suggested command: `/impeccable optimize`

**[P2] Async status is invisible to assistive tech.**
- Why it matters: the "Reviewing {phase} (n/4)…" region has no `aria-live`, and `notifyDone` only fires when the tab is hidden — a screen-reader user hears nothing on phase change or completion.
- Fix: wrap the status/phase line in `aria-live="polite"`; announce publish success/error.
- Suggested command: `/impeccable harden`

## Cognitive Load — 6 of 8 failed (high)

- FAIL Single focus · FAIL Chunking ≤4 · PASS Grouping · FAIL Visual hierarchy · FAIL One-thing-at-a-time · FAIL Minimal choices ≤4 · FAIL Working memory · PASS Progressive disclosure.
- Worst overload point: the desktop controls zone + FindingsToolbar stacked together — ~20 controls before a single finding is read. Working-memory trap: a selection silently persists *outside* the current filter (`selectAllVisible` union), so filtering then publishing can post findings the user can no longer see.

## Persona Red Flags

**Alex (fast power triage):** No keyboard shortcuts on this surface; no shift-click range select; `Esc` doesn't clear selection on desktop (only inside the phone-only Modal); publishing a filtered blocking set is a 3-step dance with no one-click "publish all blocking"; `V1/V2` humanize tabs don't say which voice.

**Sam (accessibility):** No live-region for async status (silent phase progress + completion); keyboard-only desktop publish fires an irreversible external action on Enter with no focus-managed confirm; mixed tab semantics (proper `Tabs.vue` roving `tablist` elsewhere, but humanize tabs are plain `aria-pressed` buttons). Credit: severity is *not* color-only — the colored edge is always paired with a text chip (Labelled-Status rule honored), and review-control inputs carry dynamic `:aria-label`s (B confirmed 65 aria-labels, label-wrapped selects).

## Minor Observations

- `ScoreMeter`/`RecommendationBar`/`Sparkline` are built, CVD-validated, and unused on the core surface (only `index.vue`).
- Inconsistent verb for one act: "Comment all" vs "Publish selected"/"Publish summary".
- The global `publishing` ref disables *all* publish buttons during any publish — the user can't tell which action is in flight.
- `FindingCard` (classic) and `FindingCardTriage` duplicate humanize/publish/tab logic; classic has no filters, so that view loses triage entirely.
- Reduced-opacity muted text appears 37× (`text-muted/50|60|70`); several are `aria-hidden` decorative separators, but `FindingCardTriage:128` and `ReviewList:48` (`/70`) are real small-text-contrast watch-items (ties to the prior audit's P2).

## Questions to Consider

1. If the 0–100 score is the product's trust anchor, why is it plain text on the core surface while a validated ScoreMeter gauge sits unused — isn't that gauge the entire "instrument panel" promise and the emotional peak of the review?
2. Publishing to a teammate-visible MR is the least-reversible thing the app does — why does desktop fire it unguarded while the phone gets a modal? Which environment actually needs the guardrail?
3. If a self-hoster triages 40 findings in a sitting, does a "triage tool" whose entire selection loop is mouse-driven checkboxes actually earn the name?
