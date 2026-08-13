# 4R — Full Functional Specification

> **Scope of this document.** This is a *functional* specification of what the
> application does today and where it is heading. It describes behavior,
> entities, states, rules, and flows — **not** visual design. There is nothing
> here about colors, typography, screen layout, component distribution, or
> interaction styling; those live in `DESIGN.md`. Every capability below is
> either **Covered** (built and wired), **Partial** (built with a documented
> limitation), or **Projected** (intended but not yet implemented).
>
> **Status legend:** ✅ Covered · 🟡 Partial · 🔭 Projected / Gap
>
> Generated from a source-level audit of `packages/server` (Go backend) and
> `packages/spa` (Vue SPA). Supersedes `packages/server/SPECS.md`, which was
> frozen at the initial scaffold (2026-07-06) and reflects only the review
> rubric, not the shipped feature set.

---

## 1. Product overview

4R is a self-hosted assistant that reviews GitLab merge requests with an LLM and
automates the release chores around them. It has three functional pillars:

1. **AI review** — on demand or on a webhook, it runs a four-lens review (Risk,
   Readability, Reliability, Resilience) over an MR's diff and produces scored,
   actionable findings that can be published back to GitLab.
2. **Release routines** — one-click, gated automation that reacts to, approves,
   tags, and merges/releases an MR, computing the next semver from conventional
   commits.
3. **Configuration & integrations** — GitLab accounts, LLM providers, reviewer
   "voice" profiles, notification rules, and a Telegram bot, all backed by an
   encrypted secret store.

It is designed as a **single-user, self-hosted tool**: authentication and
hardening are deliberately modest, and several endpoints return secrets in the
clear to the operator who owns the instance.

---

## 2. Architecture (functional view)

- **Backend** — Go, hexagonal: `domain` (entities + pure rules) → `app`
  (services / use cases) → `adapters` (GitLab, OpenRouter, AI providers, SQLite,
  crypto, Telegram). HTTP layer (`internal/http`) exposes a JSON API; all routes
  sit behind a session gate.
- **Persistence** — SQLite with numbered migrations. Secrets live in a separate
  encrypted table; the domain never touches ciphertext.
- **Async work** — two independent loops:
  - a **job runner** (worker pool, default concurrency 1) drains a review queue;
  - a **single routines worker** serializes all routine runs so two routines
    never race a repo's tags.
- **Frontend** — Vue 3 `<script setup>` + TypeScript SPA, Pinia stores,
  file-based routes. Talks only to the JSON API.
- **Crash recovery** — both loops re-queue `running` work on startup; terminal
  writes are compare-and-set so a recovered job can never double-charge or
  overwrite a finished record.

---

## 3. Domain glossary

| Term | Meaning |
|------|---------|
| **Account** | A GitLab connection (base URL + token) used to reach projects. |
| **Provider** | An LLM backend (kind + model + key) used to run reviews/distillation. |
| **Profile** | A writing-*voice* persona used to humanize review prose (not a severity persona). |
| **Repo** | A tracked GitLab project, bound to one account and optionally a provider+model. |
| **Review** | One AI review of one MR, producing findings + score + recommendation. |
| **Finding** | A single issue in a review: dimension, severity, location, why, fix, blocking. |
| **Routine / Run** | An automated release action over an MR (approve-and-tag or release). |
| **Notification Rule** | A binding of an event + scope to a delivery channel (Telegram). |
| **Target** | A Telegram destination (chat + optional thread + bot token ref). |
| **Secret** | An encrypted value (token/key) referenced by name from an entity. |

---

## 4. Authentication & session ✅

Two independent mechanisms share the same configured password but do different
jobs: the **vault** (§5) protects secrets *at rest*; the **auth manager** gates
the *HTTP API*.

- **Model** — stateless signed tokens, no server-side session table. Token =
  `base64url(expiry).base64url(HMAC-SHA256(expiry, sha256(password)))`, so
  sessions survive restarts.
- **Enable/disable** — a non-empty configured password enables auth; an empty
  password disables it and every route passes.
- **Login** `POST /auth/login` — rate-limited **10 attempts / IP / 60s** (429 over
  limit); wrong password sleeps a fixed 200ms then 401; success sets an
  `air_session` cookie (`HttpOnly`, `SameSite=Lax`, `Secure` only over real
  HTTPS).
- **Logout** `POST /auth/logout` — clears the cookie.
- **Status** `GET /auth/status` — `{authEnabled, authenticated}`; never requires auth.
- **Exemptions** — `/auth/*`, `/health`, `/telegram/webhook`, and
  `POST /webhooks/gitlab/*` (self-secured by their own secrets) bypass the gate.
- **Proxy trust** — `X-Forwarded-Proto`/`X-Forwarded-For` are honored only when
  `trustProxy` is enabled (CWE-290 mitigation).

**Gaps / projection**
- 🔭 No user accounts, roles, or multi-tenant separation — single shared password.
- 🔭 No password-change or session-revocation endpoint at runtime (password is a boot config value).
- 🟡 Login hardening is intentionally modest (delay + IP cap, no lockout/backoff escalation).

---

## 5. Vault & secret store ✅

- **Contract** — `secret.Store` (`Set/Get/Exists/Delete/List`); adapters encrypt
  at rest, the domain only ever sees plaintext. `List` returns names only.
- **Cipher** — AES-256-GCM. Password mode derives the key via
  PBKDF2-HMAC-SHA256, 600,000 iterations (OWASP 2023), 16-byte salt. Sealed blob
  is `nonce || ciphertext`.
- **Vault key lifecycle** — two modes:
  - **Password mode** — key derived from the app password; a constant verifier is
    sealed at init and re-opened at unlock to detect a wrong password.
  - **Key-file mode** (empty password) — a random 32-byte key persisted to a
    `0600` file beside the DB.
- **Wiring** — unlocked once at startup; first run initializes, later runs unlock.
- **Consumers** — accounts, providers, telegram store tokens/keys here by
  reference (`account:<id>:token`, `provider:<id>:apikey`, etc.).

**Gaps / projection**
- ✅ Runtime key management — `GET /vault/status` and `POST /vault/password` change the master password, set/remove it, or re-key the key file, re-encrypting every secret in one transaction (auth-gated). 🔭 No runtime *lock* state (the vault stays unlocked after boot). A password change requires updating `AIR_PASSWORD` before the next restart (surfaced as a warning); key-file re-key needs no config change.
- 🟡 Key-file mode keeps the master key next to the data (encrypted at rest but not operator-gated) — a documented trade-off.

---

## 6. GitLab accounts ✅

- **Entity** — `Account{ID, Name, BaseURL, TokenRef, CreatedAt}`. The token lives
  in the secret store, never on the entity, never in any response DTO.
- **Add** `POST /accounts` — validates name/baseURL/token non-empty and requires a
  **secure base URL**; seals the token first, rolls back the orphan secret if the
  row insert fails.
- **List** `GET /accounts` — accounts only, never tokens.
- **Delete** `DELETE /accounts/{id}` — removes row then token secret.
- **Project search** `GET /accounts/{id}/projects?search=` — live GitLab call so
  the add-repo form can pick a project instead of pasting a URL; returns
  `{id, name, pathWithNamespace, webUrl}`. Upstream failure → 502.

**Gaps / projection**
- 🔭 No update/edit endpoint — accounts are add / delete / list only. To change a token or URL you delete and recreate.

---

## 7. LLM providers ✅

- **Entity** — `Provider{ID, Name, Kind, BaseURL?, Model, APIKeyRef, IsDefault,
  Temperature?, Models[], CreatedAt}`. Key stored by reference.
- **Kinds** — `openai-compat` (Groq/OpenAI/Moonshot/Kimi…), `anthropic`,
  `gemini` (OpenAI-compatible wire + Gemini default URL), `openrouter`
  (OpenAI-compatible wire + OpenRouter default URL).
- **Add** `POST /providers` — validates name/key non-empty, kind, secure base URL;
  the **first** provider (or an explicit `makeDefault`) becomes default.
- **List** `GET /providers` — keys never returned.
- **Update** `PATCH /providers/{id}` — empty `apiKey` keeps the stored key;
  non-empty rotates it.
- **Set default** `POST /providers/{id}/default`.
- **Delete** `DELETE /providers/{id}` — row then key secret.
- **Test connection** `POST /providers/test` — one minimal live probe (`"ping"`,
  16 max-tokens, 20s timeout) to catch a bad URL/key/model at config time. Uses
  the supplied key, or falls back to the stored key when only an `ID` is given
  (so an edit form can test without re-typing the key). Persists nothing; always
  returns HTTP 200 with `{ok:true}` or `{ok:false, error}` — a failed probe is a
  valid result, not an API error.

### 7.1 OpenRouter model catalog ✅
- `GET /openrouter/models` — proxies `openrouter.ai/api/v1/models` (no upstream
  auth), 10s timeout, 8 MiB body cap, decoded to `{id, name, contextLength}` and
  sorted by id.
- **Cache** — mutex-guarded 1h TTL; a refetch error **falls back to stale data**
  if any exists, else 502. Powers model pickers in provider/repo config.

**Gaps / projection**
- 🟡 `Test` verifies reachability/credentials/model only — not quota or full review capability.
- 🔭 Catalog fetching is OpenRouter-only; other kinds rely on a manual `Models[]` preset list.

---

## 8. Reviewer voice profiles ✅

A profile is a **writing-voice persona** used when humanizing review prose — not
a reviewer-severity persona.

- **Entity** — `Profile{ID, Name, Language, Formality, Emojis, Samples[],
  StyleGuide, StyleGuideStatus, StyleGuideError, timestamps}`. Samples are raw
  pasted writing (stored in clear — not secret). `StyleGuideStatus` ∈
  `none | pending | ready | error`.
- **CRUD** — `POST/GET/GET{id}/PATCH{id}/DELETE{id} /profiles`. Only `Name` is
  required; `StyleGuide` is server-managed and never accepted or overwritten on
  input.
- **Distillation** — creating (or redistilling) a profile with samples triggers
  an async LLM pass that "distills" the samples + knobs (language, formality,
  emoji) into a durable **style guide**:
  - runs on a fresh background context with a 2-minute timeout (the request ctx
    dies when the HTTP response is sent);
  - panic-recovered — a panic persists an `error` status instead of crashing;
  - uses the **default provider** and its model (requires a model to be set);
  - status transitions `pending → ready | error`.
- **Redistill** `POST /profiles/{id}/redistill` — resets to `pending` and re-runs;
  powers a manual "regenerate" button.
- **Crash recovery** — profiles stuck in `pending` at startup are re-triggered.

**Gaps / projection**
- 🔭 A profile is *not* bound to a repo — it is chosen ad hoc per humanize call (see §11.6).
- 🟡 Distillation depends on the default provider having a model configured; no per-profile provider override.

---

## 9. Repositories ✅

- **Entity** — `Repo{ID, Name, URL, AccountID, ProviderID?, Model?, CreatedAt,
  WebhookSecret, WebhookEnabled, WebhookRequireConfirmation}`.
- **Relationships** — a repo references exactly one **account** (how it's reached)
  and optionally a **provider + model** (how it's reviewed). Empty provider →
  default provider; empty model → provider's default model.
- **Add** `POST /repos` — requires name/URL/account; validates the account exists
  and, if set, the provider exists, before persisting.
- **List** `GET /repos` — DTO adds `webhookPath` = `/webhooks/gitlab/<id>` and
  returns `webhookSecret` in the clear (single-user tool).
- **Assign** `PATCH /repos/{id}/assign` — changes provider + model only; empty
  provider clears the assignment (falls back to default).
- **Webhook config**:
  - `PATCH /repos/{id}/webhook` — `SetWebhook(enabled, requireConfirmation)`;
    enabling with no secret yet **generates** a strong random 32-byte secret;
    disabling **keeps** the secret so re-enabling reuses the URL already pasted
    into GitLab.
  - `POST /repos/{id}/webhook/rotate` — replaces the secret, leaves `enabled`
    unchanged (operator must re-paste into GitLab).
  - `WebhookRequireConfirmation` — hold webhook-triggered reviews in
    `awaiting_approval` for manual approval instead of auto-running.
- **Open merge requests** `GET /repos/{id}/merge-requests` — live list
  `{iid, title, state, sourceBranch, targetBranch, webUrl, author}`; upstream
  error → 502.
- **Branches** `GET /repos/{id}/branches` — branch names, used by the release
  branch picker to warn when `development`/`main` are absent.
- **Preflight** — see §10.

**Gaps / projection**
- 🔭 A repo's **account is not reassignable** — fixed at create time (`Assign` changes only provider+model).
- 🔭 No **profile** field on a repo (the audit corrected the earlier "assign account+provider+profile" framing).

---

## 10. Preflight (permission check) ✅

`GET /repos/{id}/preflight` reports, ahead of time, which write actions the
repo's token + access level actually permit — so a routine doesn't fail
half-way.

- **Fail-closed, three-state** per check: `ok` only when every dependent fact was
  read *and* satisfied; `fail` when read and denied; `unknown` when a required
  fact couldn't be read. A missing scope or a definitively-low access level is a
  hard `fail`, never masked by an unrelated `unknown`.
- **Facts gathered** (best-effort except Project): token scopes (`TokenSelf`;
  OAuth/job/deploy tokens can't introspect → scopes unknown), Project (required —
  yields access level + default branch), protected branches, protected tags.
- **Capabilities & required access**: `comment` & `award_emoji` → Reporter;
  `create_mr` → Developer; `merge_mr` → max(Developer, default-branch protection);
  `create_tag` → max(Developer, protected-tag level).
- **Output** — `{TokenScopes, ScopesKnown, AccessLevel, AccessLevelName,
  DefaultBranch, Checks[]}`. Upstream failure → 502.

---

## 11. Reviews — the core ✅

### 11.1 Entity & fields
`Review{ID, RepoID, MRIID, ContextMode(fast|deep), SourceBranch, TargetBranch,
ProviderID?, Model?, Status, Phase, Archived, SummaryPublished, Summary,
Findings[], Reasonings[], Recommendation, Score, Error, RawOutput,
InputTokens, OutputTokens, timestamps}`.

- **ContextMode** — `fast` (diff + touched files via API) or `deep` (shallow
  clone with full changed-file contents).
- **Phase** — the current 4R lens while running; empty otherwise.
- **Archived** — soft-hide, independent of lifecycle.

### 11.2 Status state machine
States: `awaiting_approval`, `pending`, `running`, `done`, `error`, `cancelled`.
`Terminal = done | error | cancelled`.

- **Manual create** → `pending` + enqueue.
- **Webhook create** → `pending` + enqueue, **unless** the repo requires
  confirmation → `awaiting_approval` (created but *not* enqueued).
- **Approve** — `awaiting_approval → pending` + enqueue; any other state → 409.
- **Worker (`Handle`)** — `pending → running →` `done` (success save) / `error`
  (failure, parse error, or save failure) / `cancelled` (if cancel requested).
  A cancel requested before start short-circuits to `cancelled` with no LLM call.
- **Cancel** — cooperative: sets a flag + fires the running context; `Handle`
  remains the *sole* writer of terminal status. Terminal → 409.
- **Archive** — terminal only (else 409); **Unarchive** always allowed.
- **Retry** — does **not** mutate the old review; clones its config into a fresh
  `pending` review.
- **Delete** — hard-removes review + findings in any state.

### 11.3 Triggers
- **Manual** `POST /reviews` `{repoId, mrIid, mode, providerId?, model?}`.
- **Webhook** `POST /webhooks/gitlab/{id}` — self-secured (constant-time compare
  of `X-Gitlab-Token` vs the repo's stored secret; disabled repo → 200 inert;
  blank stored secret never authenticates). Fires **only** on `merge_request`
  events with action `open`/`reopen`/`update`-with-non-empty-`oldrev` (a push of
  new commits). Merge/close/approve and metadata-only updates are ignored.
- **Duplicate guard** — skips if a review is already pending/running/awaiting for
  that MR.

### 11.4 Provider / model / context resolution
- Provider precedence: review override → repo → default provider.
- Model precedence: review override → repo → provider model; empty → error.
  Resolved model is persisted before running.
- AI client built by kind (`openai_compat`, `gemini`, `openrouter`, `anthropic`).
- Context: deep mode clones and appends full changed-file contents; fast mode
  uses the API diff. Branches are persisted right after context build so they
  survive a later failure.

### 11.5 The 4R pipeline (engine)
- **MultiPass** (the wired default) — one focused LLM call **per dimension** in
  order **Risk → Readability → Reliability → Resilience**, each reported as a
  `Phase`. Each pass uses that dimension's skill text + a shared **precision
  gate** + a JSON contract. Findings are aggregated across passes; the summary is
  composed **locally** (no extra LLM call). Cancellation is checked between
  passes.
- **Precision gate** — report a finding only with a specific changed line/symbol
  *and* a concrete realistic harm; only issues introduced by *this* diff; no
  style/preference/speculative findings; prefer silence over false positives.
- **JSON contract** — the model returns a single JSON object; a tolerant parser
  strips fences/prose. A parse failure captures the raw output + phase + tokens.
- **Reasoning capture** — gated by a reasoning budget: a positive budget requests
  Anthropic extended thinking and persists any provider's reasoning; zero
  captures nothing.

### 11.6 Findings & deterministic scoring
- **Finding** — `{Dimension, Severity(high|medium|low), File, Line(0=file-level),
  Issue, Why, Fix, Blocking, Published}`.
- **Score** — starts at 100; penalties: high −15, medium −5, low −1, plus −10 per
  blocking finding; floored at 0. The **model never self-grades**.
- **Recommendation** — any blocking or any high → `request_changes`; else any
  findings → `comment`; else → `approve`.

### 11.7 Actions
`List` (per repo, `?archived=`), `Create`, `Get` (with findings + reasonings),
`Retry` (→ new review), `Delete`, `Approve`, `Cancel` (returns `cancelling`;
flip observed by polling), `Archive`/`Unarchive`, `Publish` (§11.8),
`Humanize` (§11.9).

### 11.8 Publish to GitLab
- Only a `done` review can publish (else 502).
- **Selection** — `All` (only not-yet-published findings, so "publish all" never
  re-comments), explicit `Indices` (may re-post), `IncludeSummary` tri-state
  (nil = only on first publish), plus `SummaryOverride` / per-index
  `FindingOverrides` (humanized text posted verbatim).
- **Placement** — a finding on an *added* line becomes an **inline discussion**
  anchored via position SHAs; otherwise a **general note** with a `**File:**
  path:line` header (avoids GitLab's "line_code can't be blank" 400 on
  context/deleted lines).
- **Summary note** — `## 4R Review — <recommendation> (score N/100)` + summary +
  finding counts.
- **Idempotency** — posted findings are flagged; on a mid-loop error the
  already-posted indices are persisted first so a retry doesn't double-comment;
  summary marking retries with backoff.

### 11.9 Humanize
- Rewrites one target of a finished review in a profile's voice: a single
  finding's Issue/Why/Fix, or the summary.
- **Preconditions** — review `done` (409), profile style guide `ready` (409),
  finding index in range (400).
- Uses the **default provider** (not the review's). Output is **persisted** as a
  new "tab" so it survives reload; persistence retries and, on persistent
  failure, returns the unsaved text anyway ("losing durability beats losing the
  paid output").
- **History** `GET /reviews/{id}/humanizations` — summary rewrites as an ordered
  list; finding rewrites keyed by finding index; each run auto-assigned a
  `TabIndex` within its group.

**Gaps / projection**
- 🟡 A single-pass `Engine` (all-rules-in-one-call + model-authored summary) exists but is **not wired** — tests only.
- 🔭 No prompt caching / synthesis pass — MultiPass re-sends the full diff + system prompt each lens.
- ✅ Humanize `FindingIndex` is stable by construction: a humanization is bound to one done review, a done review's findings never change in place, and a Retry forks a *new* review (with its own empty humanization set) rather than regenerating findings — so an index can never rebind. (No reconciliation needed; earlier drafts of this spec wrongly listed this as a gap.)
- 🟡 Cancellation is cooperative only — a provider call ignoring `ctx` runs to completion; MultiPass checks cancel only between passes.
- 🟡 Deep mode silently omits unreadable/binary/moved files from full-content context.
- 🟡 A success-then-persist-failure marks the review `error` and loses the paid result (surfaced for retry).
- 🟡 Webhook `update` trigger infers "new commits" from `oldrev`; label/title-only updates won't trigger; no debounce beyond the duplicate guard.

---

## 12. The 4R rubric ✅

Skills are markdown lenses embedded in the binary; an override directory can
replace any rule at runtime without recompiling. Each is a "report only; do not
fix" lens.

- **R1 Risk** — security, privilege boundaries, data safety. Hardcoded secrets,
  frontend-only authorization, unescaped input into HTML/DOM sinks,
  concatenated SQL/command injection, insecure auth cookies. Blocking when it
  could plausibly cause a production breach or data loss.
- **R2 Readability** — clarity & maintainability. Magic numbers/strings, long
  parameter lists, duplicated logic, dead code, intent-hiding names, vague change
  descriptions. Blocking only when too convoluted to safely review.
- **R3 Reliability** — behavior & test risk. Behavior changes without contract
  tests, implementation-centric tests, missing edge cases, swallowed errors,
  unsafe retries, races, weak selectors, leftover `.only`. Blocking when a
  realistic input yields incorrect behavior with no test guarding it.
- **R4 Resilience** — operational failure risk. External calls without
  timeout/retry/fallback/circuit-breaker, cascading failures, missing
  observability on new critical paths, rollback readiness, retry storms.
  Blocking when a reachable production path takes down the system on a dependency
  failure with no recovery.

`GET /skills` exposes the loaded rubric.

---

## 13. Routines (Actions / Releases) ✅

### 13.1 Run & step entities
- **Run** — `{ID, Kind, RepoID, MRIID, Status, Params(immutable input),
  State(mutable accumulator), Steps[], LastError, timestamps, Archived}`.
- **Step** — `{Name, Status, Detail, UpdatedAt}`. Steps are checkpointed after
  every transition (routine actions are irreversible); a resume skips steps
  already `done`/`skipped`.

### 13.2 Statuses & transitions
- **Run statuses** — `pending`, `running`, `blocked` (paused on a failed step,
  resumable), `awaiting_confirmation` (paused on the merge gate, *not* an error),
  `done`, `cancelled`. (`failed`/`skipped` are **step-level only**; `archived` is
  a flag, not a status.)
- Persistence is compare-and-set: a row already terminal is never overwritten
  (`ErrRunFinalized`), and the executor then stops. `RequeueRunning` recovers
  `running → pending` on startup.

### 13.3 The three flows (exact ordered ledgers)
Only two `Kind` values exist — `approve_and_tag` and `release`; the dev and main
flows are both `release`, discriminated by `Flow` (`development` vs `main`).

**a) approve-and-tag** — `react → comment → tag`
1. `react` — award emojis (idempotent; default 👍🌱).
2. `comment` — post a note (default "LGFM").
3. `tag` — requires the MR `merged`; compute next tag once, tag the merge-commit
   SHA (falls back to squash, then head).

**b) release — dev flow** — `verify → react → approve → compute_tag → confirm →
merge → tag → notify`
1. `verify` — no conflicts + latest pipeline green; captures title, pins head SHA.
2. `react` — award emojis.
3. `approve` — approve the MR (idempotent; see §13.6).
4. `compute_tag` — base `HighestSemver`, count MR commits, `NextRelease` (default
   bump `minor`); checkpoints last/next tag + feat/fix counts.
5. `confirm` — interactive gate (§13.4).
6. `merge` — per decision (§13.4).
7. `tag` — tag `NextTag + "-dev"` on the merge SHA.
8. `notify` — best-effort release summary; never fails the run.
   *Creation is dev-flow only and rejects any target ≠ `development`.*

**c) release — main flow** — `compute_tag → create_mr → wait_pipeline → approve →
react → confirm → merge → tag → notify` (starts with `MRIID = 0`)
1. `compute_tag` — base is the highest **pure** release tag (`HighestReleaseSemver`,
   ignores `-dev`); counts source commits not reachable from the base ref; zero
   feat+fix → clean stop ("nothing to release").
2. `create_mr` — create the `development→main` MR (marked "Main Release:"), or
   reuse an already-open marked one; an **unmarked** human MR for the same branch
   pair **blocks** the run (anti-hijack).
3. `wait_pipeline` — re-fetch, block on conflicts, poll pipeline to success; pin
   head SHA (no `verify` step here).
4–7. `approve`, `react`, `confirm`, `merge` — as dev flow.
8. `tag` — tag `NextTag` with **empty** suffix (pure release).
9. `notify`.

### 13.4 Confirmation gate
Both release flows share `confirm`. With no decision yet it returns a sentinel and
the run pauses at `awaiting_confirmation` (step left non-terminal).
`POST /routines/{id}/confirm` accepts `"merge"` or `"wait"` (else 400); run must
be awaiting (else 409); it records the decision and flips to `pending`,
re-entering `confirm`.
- `merge` — triggers the merge exactly once (durable guard), re-fetching the MR
  and refusing if it became unmergeable (stale-head / TOCTOU guard), then polls
  to `merged`.
- `wait` (manual merge) — never merges; only polls for an out-of-band human merge.
- Bounded by a merge-wait timeout (default 10 min); on timeout the run blocks and
  a resume re-polls.

### 13.5 Version / tag computation
- **Selectors** — `HighestSemver` (includes prereleases; dev base) vs
  `HighestReleaseSemver` (pure releases only, ignores `-dev`; main base). Both
  preserve the original string form (a `v` prefix survives).
- **`NextRelease(lastTag, subjects, mode)`** (subjects oldest-first):
  - empty/non-semver base ⇒ `0.0.0`, no `v` prefix; a parseable base keeps its
    own convention;
  - `BumpMinor` (primary) — `minor += feat count`; patch = fixes after the last
    feat (or all fixes if no feat);
  - `BumpMajor` — `(major+1).0.0` regardless of commits;
  - `BumpPatch` — `patch += feat + fix`.
  - Returns total feat/fix counts for display, independent of the bump math.
- **`ClassifyCommit`** — conventional-commit `feat`/`fix` (case-insensitive,
  optional scope/`!`), first line only; token boundary prevents `fixup…`
  matching `fix`.
- **Suffix at tag step** — `-dev` for dev flow, empty for main. Duplicate-tag
  guard treats a leading `v` as insignificant.
- **Tag preview (dry-run)** `GET /repos/{id}/routines/preview-tag?flow&bump&mrIid&source&target`
  — mirrors `compute_tag` exactly without creating a run; returns
  `{nextTag, lastTag, featCount, fixCount}`.

### 13.6 Approve idempotency (hardened)
The `approve` step is idempotent: GitLab answers an already-approved MR with a
401/409, sometimes with an "already" body and **sometimes a bare
`401 Unauthorized`**. On such a status the client actively verifies via
`GET /user` + the approvals endpoint that the current user already approved and
treats it as success; a genuinely bad token also fails those calls, so the
original 401 still surfaces (never masked). A 403 is a real permission failure
and always surfaces.

### 13.7 Lifecycle actions
- **Resume** — `blocked → pending` (else 409); checkpointed steps skip completed
  work.
- **Skip** — only a `blocked` run whose failed step is skippable
  (`react/comment/approve/notify`); essential steps → 409. Marks the step
  `skipped`, run → `pending`.
- **Cancel** — atomic CAS on any non-terminal run; fires the run's cancel func;
  already-terminal → 409.
- **Archive / Unarchive** — archive terminal runs only (else 409); unarchive
  clears the flag.
- **Delete** — any status except `running` (409).
- **Listing** — `GET /repos/{id}/routines` (active or `?archived=1`),
  `GET /routines?limit=&archived=` (recent cross-repo, clamped 1–100, default 20),
  `GET /routines/{id}`.
- **Duplicate guard** — an active run for the same repo+MR+kind → 409; the main
  flow dedupes on flow+target branch instead.

**Gaps / projection**
- 🟡 The `ReleaseNotifier` routing is best-effort; a nil notifier disables notify entirely (see §14 wiring).
- 🟡 Main-flow duplicate check is check-then-act (documented TODO) — mitigated only by the single worker.
- 🟡 The single-worker design serializes all runs; a long merge-wait blocks other runs ("acceptable for now").
- 🟡 `comment` step could double-post on a crash between posting and checkpoint (accepted for a short "LGFM").
- 🟡 `getRoutine` maps any load error (incl. unknown run) to 500 rather than 404.

---

## 14. Notifications ✅

- **Entities** — `Rule{ID, Event, NotifierKind, NotifierID, RepoID?, Enabled,
  CreatedAt}`. Events are a fixed enum: `review.finished`, `release.finished`.
  `NotifierKind` supports exactly one value today: `telegram`.
- **Operations** — `GET /notifications/events`, `GET /notifications/rules`,
  `POST /notifications/rules` (`notifierKind` defaults to telegram; a set
  `repoId` is validated), `PATCH /notifications/rules/{id}` (toggle `enabled`
  only), `DELETE /notifications/rules/{id}`.
- **Validation** — rejects unknown event, unsupported kind, empty/unresolvable
  notifier id, and duplicates on `(event, repoId, kind, id)` → 409. New rules are
  enabled.
- **Fan-out with override semantics** — `Notify(event, repoId, text)` loads
  enabled rules and partitions into repo-scoped vs global: **if any repo-scoped
  rule exists, only those fire and global rules are skipped for that repo**;
  otherwise global rules fire as a fallback. Best-effort — a failed delivery is
  logged and skipped, and the call always succeeds so a finished review/release
  is never blocked.
- **Emitters** — `review.finished` from the reviews service (detached goroutine);
  `release.finished` from the routines service via a `releaseNotifier` adapter
  wired at composition time (this adapter decouples routines from notifications).

**Gaps / projection**
- 🔭 Telegram is the only channel — no email, Slack, webhook-out, etc.
- 🔭 Only two events exist — no per-severity, per-verdict, or per-step notifications.

---

## 15. Telegram ✅ (bot core) / 🟡 (go-live)

### 15.1 Targets
- **Entity** — `Target{ID, Name, ChatID, ThreadID?, TokenRef, IsDefault, IsBot,
  CreatedAt}`. The bot token lives in the secret store. At most one default; at
  most one `IsBot` (the interactive-bot driver).
- **Operations** — create (token sealed first, row rolled back on failure),
  update (blank token keeps stored, non-empty rotates), duplicate (re-encrypted
  under its own ref, never default/bot), list, set-default, set-bot, **test**
  (fixed "✅ ai-reviewer test message"; failure → 502), **resolve** (stateless:
  takes a raw token, calls `getUpdates`, returns discovered chats + forum threads
  for a pick-list; token never persisted or echoed), delete (target + secret +
  best-effort drop of rules pointing at it).
- **Send path** — one `SendTo(targetID, text)` used by both fan-out and test;
  sends HTML (`parse_mode=HTML`); callers pre-escape interpolated values.

### 15.2 Inbound webhook + interactive bot
- **Endpoint** `POST /telegram/webhook` — **dormant unless a global webhook secret
  is configured** (returns 200, does nothing). When set, requires a constant-time
  header-secret match (else 401), decodes the update, and dispatches on a detached
  goroutine (30s ctx, panic recovery), returning 200 immediately.
- **Dispatcher** — resolves the actor chat, **authorizes against the allowlist =
  chat IDs of all configured targets** (unauthorized actors are silently
  ignored), resolves the bot token from the `IsBot` target (none → dormant), then
  routes callback queries or `/`-commands. Never edits messages — every reply is
  fresh.
- **Commands** — `/start`, `/menu`, `/repos`, `/reviews`.
- **Callbacks** — navigate repos/reviews, list a repo's open MRs, trigger a review
  in fast/deep mode, render a review's outcome (header + summary + findings).

**Gaps / projection**
- 🔭 **Go-live ("5c") is not built** — no `setWebhook`/`deleteWebhook`; enabling requires an external `setWebhook` call + setting the global secret. Until then the endpoint is dormant.
- 🔭 No in-place message editing — the bot always sends fresh messages ("auto-updates on next open; no live push").
- 🔭 "Publish findings from chat" is out of scope by design.
- 🟡 The webhook secret is a single global config value, not per-target.

---

## 16. GitLab webhook ingestion ✅

`POST /webhooks/gitlab/{id}` — per-repo, exempt from the global auth middleware
(self-secured).
- Unknown repo → 404; `WebhookEnabled=false` → 200 "ignored"; blank stored secret
  never authenticates; else constant-time compare of `X-Gitlab-Token`.
- Decodes a minimal `MergeRequestEvent`; triggers **reviews only** on
  `open`/`reopen`/`update`-with-`oldrev` (a code push). Dispatch is a detached
  goroutine so GitLab gets a fast 200.
- Routines are **not** created by this webhook — they use their own
  `/repos/{id}/routines/*` endpoints.

---

## 17. Jobs / async runner ✅

- **Entity** — `Job{ID, ReviewID, Status(pending|running|done|error), Attempts,
  LastError, timestamps}`. Its only unit of work is "run the review it points at".
- **Runner** — a worker pool (default concurrency 1, configurable) that
  atomically claims the oldest pending job, wakes a peer on claim (spreads
  bursts), and runs `reviews.Service.Handle`. A handler panic is converted to an
  error (with stack) so one bad review fails only its own job, never the process.
- **Recovery** — `RequeueRunning` on startup re-queues jobs interrupted by a
  crash; `Handle` guards against re-charging an already-terminal review.
- **No auto-retry** — retry is the manual `POST /reviews/{id}/retry` endpoint.
- Routines run on a **separate** worker loop, not through this runner.

---

## 18. Cross-cutting concerns

- **Secrets at rest** — every token/key is AES-256-GCM sealed; never returned by
  list endpoints; DTOs omit secret fields (repo `webhookSecret` is the deliberate
  single-user exception).
- **Idempotency** — award-emoji, approve, publish, and merge are all idempotent so
  a resume/retry never double-acts on GitLab.
- **Crash recovery** — reviews (jobs) and routines both re-queue `running` work on
  startup; terminal writes are compare-and-set.
- **Fail-closed vs fail-soft** — preflight is fail-closed (unknown ≠ ok); caches
  and notifications are fail-soft (serve stale / log-and-skip).
- **Rate limiting** — login is IP-rate-limited with a self-sweeping window.
- **Timeouts** — every outbound call is bounded (AI probe 20s, GitLab client,
  OpenRouter 10s, telegram 10s, distillation 2min, merge-wait 10min).

---

## 19. Coverage matrix

| Area | Status | Notes |
|------|--------|-------|
| Auth / session | ✅ | Single shared password; no roles/users |
| Vault / secrets | ✅ | No runtime key management |
| GitLab accounts | ✅ | No edit endpoint |
| LLM providers | ✅ | Test connection is a minimal probe |
| OpenRouter catalog | ✅ | OpenRouter-only; 1h cache |
| Reviewer profiles | ✅ | Voice-only; not repo-bound |
| Repositories | ✅ | Account not reassignable; no profile field |
| Preflight | ✅ | Fail-closed capability report |
| Reviews (create/run) | ✅ | MultiPass 4R; cooperative cancel |
| Review publish | ✅ | Inline + note fallback; idempotent |
| Humanize | ✅ | Persisted tabs; finding index stable by construction |
| 4R rubric | ✅ | Overridable at runtime |
| Routines: approve-and-tag | ✅ | |
| Routines: release (dev) | ✅ | |
| Routines: release (main) | ✅ | Dedup race (documented) |
| Confirmation gate | ✅ | merge / wait |
| Version / tag preview | ✅ | Dry-run mirrors compute_tag |
| Notifications | ✅ | Telegram-only; 2 events |
| Telegram targets | ✅ | |
| Telegram bot (interactive) | ✅ | Dormant until go-live |
| Telegram go-live (setWebhook) | 🔭 | Not built |
| GitLab webhook (reviews) | ✅ | Code-push events only |
| Jobs / async runner | ✅ | No auto-retry |
| Single-pass engine | 🟡 | Present, not wired |
| Prompt caching / synthesis | 🔭 | Not built |
| Multi-channel notifications | 🔭 | Telegram only |
| Roles / multi-user | 🔭 | Single-user by design |

---

## 20. Consolidated projection (what's missing / next)

Grouped by theme, drawn from the gaps above. This is the backlog surface a new
design should account for — not a commitment.

**Reviews**
- Wire (or delete) the single-pass engine; add prompt caching + an optional
  synthesis pass to cut cost/latency of the N-call MultiPass.
- Debounce / smarter webhook triggering beyond the active-review guard.

**Routines**
- Enforce main-flow dedup with a store constraint (close the check-then-act race).
- Optional parallelism across repos (today one global worker serializes everything).
- Correct `getRoutine` 404 mapping.

**Integrations & config**
- Account edit endpoint (token/URL rotation without delete+recreate).
- Repo account reassignment.
- Optional repo↔profile binding so humanize has a per-repo default voice.
- Model-catalog fetching for non-OpenRouter kinds.

**Notifications & Telegram**
- Build Telegram go-live (`setWebhook`/`deleteWebhook` + admin endpoint) so the
  bot is usable without external setup.
- More channels (email/Slack/webhook-out) and finer events (per-severity, per-step).

**Platform**
- Runtime vault *lock* state (lock/unlock after boot) — change-password and re-key already shipped; a lockable state is the remaining piece.
- Multi-user / roles if the tool ever leaves single-user scope.

---

## 21. Non-goals (by current design)

- Not a multi-tenant SaaS — single shared password, secrets returned to the owner.
- Not a Git host or CI system — it drives GitLab, it does not replace it.
- The reviewer does **not** auto-fix code — every lens reports only.
- No live push to Telegram — the bot reflects state on next interaction.

---

## Appendix A — Endpoint reference

```
GET    /health
GET    /skills

POST   /accounts
GET    /accounts
GET    /accounts/{id}/projects
DELETE /accounts/{id}

GET    /openrouter/models

POST   /providers
POST   /providers/test
GET    /providers
PATCH  /providers/{id}
POST   /providers/{id}/default
DELETE /providers/{id}

POST   /telegram
POST   /telegram/resolve
GET    /telegram
PUT    /telegram/{id}
POST   /telegram/{id}/duplicate
DELETE /telegram/{id}
POST   /telegram/{id}/default
POST   /telegram/{id}/bot
POST   /telegram/{id}/test
POST   /telegram/webhook

POST   /webhooks/gitlab/{id}

GET    /notifications/events
GET    /notifications/rules
POST   /notifications/rules
PATCH  /notifications/rules/{id}
DELETE /notifications/rules/{id}

POST   /profiles
GET    /profiles
GET    /profiles/{id}
PATCH  /profiles/{id}
DELETE /profiles/{id}
POST   /profiles/{id}/redistill

POST   /repos
GET    /repos
PATCH  /repos/{id}/assign
PATCH  /repos/{id}/webhook
POST   /repos/{id}/webhook/rotate
DELETE /repos/{id}
GET    /repos/{id}/merge-requests
GET    /repos/{id}/preflight
GET    /repos/{id}/reviews
GET    /repos/{id}/routines
GET    /repos/{id}/routines/preview-tag
GET    /repos/{id}/branches
POST   /repos/{id}/routines/approve-and-tag
POST   /repos/{id}/routines/release
POST   /repos/{id}/routines/release-main

GET    /routines
GET    /routines/{id}
DELETE /routines/{id}
POST   /routines/{id}/archive
POST   /routines/{id}/unarchive
POST   /routines/{id}/resume
POST   /routines/{id}/skip
POST   /routines/{id}/confirm
POST   /routines/{id}/cancel

POST   /auth/login
POST   /auth/logout
GET    /auth/status

POST   /reviews
GET    /reviews/{id}
DELETE /reviews/{id}
POST   /reviews/{id}/retry
POST   /reviews/{id}/approve
POST   /reviews/{id}/publish
POST   /reviews/{id}/cancel
POST   /reviews/{id}/archive
POST   /reviews/{id}/unarchive
POST   /reviews/{id}/humanize
GET    /reviews/{id}/humanizations
```

## Appendix B — SPA surfaces (functional, no design)

| Route | Purpose |
|-------|---------|
| `/login` | Session login |
| `/` | Home / dashboard |
| `/flow` | Per-repo attention-first cockpit (reviews + routines + settings) |
| `/repos`, `/repos/{id}` | Repo list; repo detail (reviews / open MRs / routines tabs) |
| `/reviews`, `/reviews/{id}` | Review list (grouped attempts); review detail + findings |
| `/actions`, `/actions/{id}` | Routine run list; run detail (ledger, gate, resume/skip/cancel) |
| `/accounts` | GitLab accounts |
| `/providers` | LLM providers + test connection |
| `/profiles` | Reviewer voice profiles |
| `/skills` | The 4R rubric (read-only) |
| `/telegram` | Telegram targets |
| `/settings`, `/more` | Settings hub / secondary nav |
