# Project Audit — findings & fix checklist

Consolidated from a 4-lens audit (reliability, risk, resilience, frontend/leftovers) over the
current on-disk state. Report-only findings; tick each box as it is fixed.

**Deployment context:** this is a self-hosted tool assumed to run locally/single-user today.
Some severities (notably auth) change if it is ever exposed to a network — noted inline.

Severity legend: 🔴 P0 (user-facing bug / outage / security) · 🟠 P1 (correctness, cost/data loss)
· 🟡 P2 (robustness/quality) · ⚪ P3 (minor/cleanup).

---

## 🔴 P0 — Reported bugs, outage & security

- [x] **1. GitLab 400 `line_code can't be blank` on inline publish.** Root cause is NOT the
  include-summary interaction (that path is correctly gated). Inline discussions send only
  `position[new_line]` with no `old_line`/`old_path` (`internal/adapters/gitlab/publish.go:22-44`),
  and `internal/review/engine/parse.go:41-50` copies the model's `line` with no validation that it
  is an *added* diff line. A finding on a context line → 400.
  **Fix:** validate `f.Line` against the diff hunks before taking the inline branch
  (`internal/app/reviews/publish.go:84-91`), fall back to a general note otherwise, populate
  `old_line`/`old_path` for context lines, and add a test for the context-line position.

- [x] **2. File path missing from posted comment when finding has no line.** `formatFinding`
  (`internal/app/reviews/publish.go:155-168`) never emits `File`/`Line`; general notes lose all
  location context. **Fix:** prepend the file path in the general-note branch (or in `formatFinding`
  when `Line == 0`).

- [~] **3. No auth/authz on any HTTP endpoint + default bind on all interfaces.**
  *(Partial: default bind moved to `127.0.0.1:8080`. The auth middleware is still open.)*
  `internal/http/server.go:44-89` (no middleware), `internal/config/config.go:27`
  (`:8080`), `cmd/server/main.go:85`. Anyone reaching the port can publish to MRs with the stored
  PAT, delete accounts/reviews, add a provider. Defensible on localhost; serious if exposed.
  **Fix (minimum now):** default `AIR_HTTP_ADDR` to `127.0.0.1:8080`. Add a bearer/app-password
  middleware on mutating routes before any non-local exposure.

- [x] **4. Zero panic recovery anywhere → a panic crashes the whole process.**
  `internal/jobs/runner.go:106-117` (`go runner.Start`), `internal/app/profiles/service.go:221-229`
  (`triggerDistill` goroutine). No supervisor in the repo. **Fix:** `defer recover()` around the
  runner's handler invocation and the `triggerDistill` goroutine body, failing just that job/profile.

## 🟠 P1 — Correctness / cost & data loss

- [x] **5. Review state-machine recovery gaps** (two related holes):
  - `Save` failure after a successful review leaves the review `running` forever; the job is already
    `error`, so `RequeueRunning` never re-runs it (`internal/app/reviews/service.go:229-246`,
    `internal/jobs/runner.go:106-117,68`). Paid LLM output lost, no self-heal.
  - `Handle` has no terminal-status guard, so after a crash `RequeueRunning` can re-execute an
    already-`Done` review (dup LLM cost, wipes `published` flags, breaks humanization `finding_index`)
    (`internal/app/reviews/service.go:203-247`).
  **Fix:** early-return in `Handle` if `rv.Status.Terminal()`; on `Save` failure fall back to
  `SetStatus(..., StatusError, ...)`; consider a startup sweep for stuck `running` reviews.

- [x] **6. LLM calls have no retry/back-off.** A single 429/5xx on pass 3/4 discards all prior paid,
  successful passes (`internal/adapters/ai/http.go:33-65`, `internal/review/engine/multipass.go:51-79`).
  **Fix:** bounded exponential back-off on 429/5xx (respect `Retry-After`), and/or persist per-pass
  results incrementally.

- [x] **7. Bulk publish drops finding humanized overrides.** `Publish selected` / `Comment all` /
  sticky bar / confirm never forward `findingOverrides`, so a selected+humanized finding publishes
  the generated (robot) text (`packages/spa/src/pages/reviews/[id].vue:173-191`). The per-card
  Publish button does forward them → inconsistent behavior. Same class as the (already-fixed) summary
  override bug. **Fix:** build `findingOverrides` from each selected finding's `selectedFindingTab`,
  mirroring `withSummaryOverride`.

- [x] **8. Summary can double-post on retry after a partial write failure.** If `CreateNote`
  succeeds but `MarkSummaryPublished` then fails, `SummaryPublished` stays false → a retry re-posts
  (`internal/app/reviews/publish.go:61-72`). The findings loop below it already guards this.
  **Fix:** apply the same "persist what posted, even on error" pattern to the summary branch.

- [x] **9. Humanize persistence failure discards the already-generated (paid) text.** On `Add`
  failure, `HumanizeFinding`/`HumanizeSummary` return an empty struct + error instead of the computed
  text (`internal/app/humanize/service.go:57-83,88-111`). **Fix:** return the computed text alongside
  a persistence-failed flag, or retry `Add` a bounded number of times.

## 🟡 P2 — Robustness / quality

- [ ] **10. Internal error details returned verbatim to the client.** `writeErr`
  (`internal/http/handlers.go:99-106`) leaks DB op names, upstream GitLab/LLM bodies, filesystem
  paths. **Fix:** map to a generic client message; log the full wrapped error server-side.

- [ ] **11. `tab_index` race in humanization persistence.** The `Add` comment claims transaction
  safety that vanilla SQLite doesn't provide (no `SetMaxOpenConns(1)`, no `UNIQUE`)
  (`internal/adapters/sqlite/humanization_store.go:28-59`, `migrations/0011_humanizations.sql`,
  `db.go:20-38`). Double-click Humanize → two rows with `tab_index=0`. **Fix:** add
  `UNIQUE(review_id, target, finding_index, tab_index)` or compute `MAX(tab_index)+1` atomically.

- [x] **12. No `https://` scheme enforcement on provider/account `BaseURL`.** A `http://` URL sends
  the API key / GitLab PAT in cleartext (`internal/adapters/ai/openai.go`, `anthropic.go`,
  `gitlab/client.go`). **Fix:** require `https://` on persist (allow-list `http://localhost` for local).

- [x] **13. `Modal.vue` has no focus trap and no `aria-labelledby`.** Tab leaks to the background
  page; the title `<h2>` has no `id` linked to the dialog. Used by phone Filters and publish-confirm.
  **Fix:** trap Tab within the panel, mark background inert while open, wire `aria-labelledby`.

- [x] **14. Classic card `opacity-60` dims still-active buttons.** Published classic cards fade the
  live Humanize / "Publish again" buttons to look disabled — regression vs the triage card's
  documented decision (`packages/spa/src/modules/reviews/components/FindingCard.vue:100-101`).
  **Fix:** drop `opacity-60`, keep the `bg-ok/5` tint.

- [x] **15. Two backend comments now lie.** "It is ephemeral / nothing is persisted" in
  `internal/app/humanize/service.go:1-6` and `internal/http/handlers.go:453-456` — humanize runs are
  now persisted. **Fix:** update the comments.

- [x] **16. `git clone --branch <ref>` uses an MR-controlled ref with no explicit validation.**
  `internal/adapters/gitlab/clone.go:37-41` (ref = MR source branch). Mitigated only by upstream
  git/GitLab ref-format rules, not enforced here. **Fix:** validate `ref` (reject leading `-`,
  restrict charset) before use.

## ⚪ P3 — Minor / cleanup

- [ ] **17. No observability/alerting.** All failures go to `log.Printf`; no metrics, health/readiness
  endpoint, or alert thresholds. **Fix:** add error-rate/latency metrics + a stuck-`running` sweep.
- [x] **18. Type/wire mismatch.** `HumanizationsResponse.findings` typed `Record<number, ...>` but
  the server emits string object keys (`packages/spa/src/shared/api/types.ts:101-104`,
  `handlers.go:687`). **Fix:** type it `Record<string, FindingHumanized[]>`.
- [x] **19. `hydrateHumanized` can clobber an in-flight humanize.** Navigating away/back while a
  humanize call is pending overwrites from the server (which lacks the not-yet-persisted run)
  (`store.ts:257-263`). **Fix:** skip/merge hydrate when `humanizing` is active.
- [x] **20. Duplicated finding-card script (~50 lines).** `FindingCard.vue` and
  `FindingCardTriage.vue` share identical props/bindings/`publish()`. Fix #7 would need editing both.
  **Fix:** extract a `useFindingCard(props)` composable.
- [ ] **21. Small cleanups.** Store returns raw refs never read directly; N `matchMedia` listeners
  (one per triage card); `id.New()` panics on `crypto/rand` failure with no recover boundary;
  unbounded map growth in cancel bookkeeping on a rare error path; cancel-vs-error can misclassify a
  genuine failure as `cancelled` (`internal/app/reviews/service.go:229-241`).

---

## Verified clean (no action)

Vault AES-256-GCM + PBKDF2 (600k iters), clone token via `GIT_ASKPASS` (never in `argv`),
parameterized queries incl. the `IN(...)` builder, delete cascade correctness, atomic job `Claim`,
fail-fast migrations, response 16MB cap + client timeouts, malformed-LLM-JSON handled without panics,
`resolveIndices`/publish tri-state logic, and `Retry` (always a fresh review, so humanization
`finding_index` stays valid under normal flow).
