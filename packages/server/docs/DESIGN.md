# DESIGN — ai-reviewer

Lightweight technical design. Source of truth for the MVP backend.
Scope: GitLab-first, single user, manual trigger, single-pass review.

## Decisions (locked)

| Area | Decision |
|---|---|
| Language | Go 1.26 |
| Design | API-first. One backend owns everything; clients (TUI, SPA, Telegram bot) only consume the HTTP API. |
| Phases | 1) server → 2) TUI (Bubbletea) → 3) SPA (Vue) → 4) Telegram bot |
| Data | SQLite behind an abstract repository layer (migratable to Postgres). Driver: `modernc.org/sqlite` (pure Go, no cgo → single-binary deploy). |
| App auth | Optional local password; if set, it derives the key that encrypts secrets at rest. |
| Secrets | Stored by the backend, encrypted at rest. UX: show whether a token is stored, allow revoke. |
| Jobs | Job table with an adapter (DB default, Redis later). MVP: SQLite table + worker goroutines. Retry CLONES the failed job, never deletes it. |
| Review engine | Strategy pattern on two axes: ContextStrategy (`fast` = diff + touched files / `deep-lite` = shallow clone) × ReviewStrategy (`single-pass` MVP / `multi-pass` later with prompt caching). |
| Review output | Findings: `dimension`, `severity`, `location (file:line)`, `issue`, `why`, `fix`, `blocking`. Plus overall score + recommendation. |
| Publish | Selectable per review: inline discussions (GitLab position API) and/or summary note. "Comment all" button. |
| AI providers | OpenAI-compatible (Groq, OpenAI, Moonshot, Kimi, OpenRouter) share one HTTP impl; Claude is a second adapter. Unified `Provider` port. |

### Deferred (post-MVP)
- Adaptive contextual memory layer 3 (learn from accept/reject). MVP keeps layer 1 (4R skills) + layer 2 (per-repo instructions, minimal).
- GitHub, OAuth, multiple accounts.
- Webhook auto-trigger (enabled when deployed on VPS). Telegram: notify-only in MVP, interactive later.
- Multi-pass review + prompt caching. Agentic deep review with sandboxed system tools.

## Architecture — hexagonal / screaming

```
cmd/server/main.go            # composition root
internal/
  config/                     # env + file config
  domain/                     # entities + ports (interfaces). No I/O.
    account/                  #   GitLab account
    provider/                 #   AI provider config
    repo/                     #   tracked repository
    review/                   #   review, finding, job
    secret/                   #   secret value + cipher port
  app/                        # use cases (orchestration over ports)
  adapters/
    sqlite/                   # repository implementations + migrations
    crypto/                   # secret encryption (cipher port impl)
    gitlab/                   # GitLab REST client
    ai/                       # AI provider adapters (openai-compat, claude)
    jobs/                     # job queue (db adapter; redis later)
    review/                   # ContextStrategy + ReviewStrategy impls
  http/                       # API handlers = the client contract
migrations/                   # *.sql
docs/
  DESIGN.md
  skills/r-*.md               # 4R review rules (loaded by the engine)
```

Rule: `domain` depends on nothing. `adapters` implement domain ports. `app`
orchestrates ports. `http` calls `app`. Dependencies point inward.

## Build order — vertical slices

- [x] **0. Skeleton** — module, layout, config, SQLite conn + migration runner, crypto (secret cipher), one domain port + adapter as the pattern seed. Compiles + a test.
- [x] **1. Secrets & accounts** — encrypted secret store, optional app password (vault), GitLab accounts CRUD, AI providers CRUD + default.
- [x] **2. Repos** — add repo from URL, assign provider/model, CRUD.
- [ ] **3. GitLab client** — list MRs, fetch diff + touched files (fast), shallow clone (deep-lite).
- [ ] **4. AI adapter** — openai-compatible + Claude behind one Provider port.
- [ ] **5. Review engine** — ContextStrategy + ReviewStrategy, load skills, structured findings, score. Jobs table + worker goroutines (retry clones).
- [ ] **6. Publish** — selective findings → GitLab inline discussions + summary note.
- [ ] **7. HTTP API** — expose all of the above (the TUI contract).

Each slice must compile and carry its own tests before moving on.
