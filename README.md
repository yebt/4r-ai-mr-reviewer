![4R — AI code review](assets/banner2.jpg)

# 4R — AI Merge Request Reviewer

Self-hosted AI code review for **GitLab merge requests**, built around the **4R
quality framework** — **R**isk, **R**eadability, **R**eliability, **R**esilience.

You configure your GitLab accounts and an AI provider, track repositories, and
run a review on any open merge request. The engine loads the 4R rule sets,
sends the diff to the model, and produces **located, structured findings** with
a deterministic score and an approve / request-changes recommendation. You then
choose which findings to publish back to the MR as inline discussions.

> Status: **backend MVP complete** (GitLab-first, single-user). Web SPA in
> active development. See the [roadmap](#roadmap).

## Why 4R

Every change is reviewed through four lenses so the issues that matter most are
caught before they reach production — without slowing down changes that are
genuinely safe:

| Lens | Question |
|---|---|
| **R1 · Risk** | Can this harm security, data, or production stability? |
| **R2 · Readability** | Will the next engineer understand it without an hour of digging? |
| **R3 · Reliability** | Does it behave correctly across the realistic range of inputs? |
| **R4 · Resilience** | Does it degrade gracefully when dependencies fail or slow down? |

## Features

- **GitLab MR review** — list open MRs, fetch the diff (fast) or shallow-clone
  for deeper context (deep), run the 4R engine, publish selected findings.
- **Deterministic scoring** — the recommendation and 0–100 score are computed
  from findings, not asked of the model.
- **Multiple AI providers** — OpenAI-compatible (Groq, OpenAI, Moonshot, Kimi,
  OpenRouter) and Anthropic (Claude), with per-repo provider/model and optional
  temperature.
- **Secrets encrypted at rest** — tokens and API keys are AES-256-GCM encrypted;
  the API never returns them.
- **Async jobs** — reviews run in the background with status polling; retry
  clones the review and keeps the failed one for history.
- **Selective publishing** — pick which findings become inline discussions, or
  comment them all.

## Architecture

A monorepo. The backend owns all state and is the single contract every client
consumes over HTTP.

```
packages/
  server/   Go 1.26 backend — hexagonal, SQLite (single binary), REST API
  spa/      Vue 3 + TypeScript + Vite + UnoCSS + Pinia web client
docs/       API reference, design notes, banner prompt
```

- **Backend**: Go + SQLite (`modernc.org/sqlite`, pure-Go → single binary), an
  encrypted secret vault, a job runner, and the 4R engine behind strategy
  interfaces (fast/deep context × single/multi-pass).
- **Web**: file-based routing, feature modules, a borderless technical-minimal
  design system.

## Quick start

Requires **Go 1.26+**, **Node 22+** / **bun**, and **git**.

```sh
# run backend + SPA together (backend :8080, SPA :5173)
make dev

# …or separately
make run-server
make run-spa
```

Then open <http://localhost:5173>. The Vite dev server proxies `/api` to the
backend, so no CORS setup is needed.

### Configuration (backend env vars)

| Variable | Default | Purpose |
|---|---|---|
| `AIR_HTTP_ADDR` | `:8080` | Listen address |
| `AIR_DB_PATH` | `ai-reviewer.db` | SQLite database file |
| `AIR_PASSWORD` | _(empty)_ | Unlocks the secret vault; empty → key-file mode |
| `AIR_KEYFILE_PATH` | `<db>.key` | Master key file (key-file mode) |
| `AIR_SKILLS_DIR` | _(empty)_ | Override dir for the 4R rule files |

## Make targets

```
make            # help
make dev        # backend + SPA together
make run-server # backend only
make run-spa    # SPA only
make build      # compile the server binary to ./bin
make test       # server test suite
```

## API

The HTTP API is the contract for every client. See **[docs/API.md](docs/API.md)**
for the full reference and **[docs/api.http](docs/api.http)** for a runnable
request collection.

## Roadmap

Deferred beyond the current MVP:

- GitHub support, OAuth, multiple accounts
- Webhook auto-trigger (on VPS deploy)
- Telegram bot (notify → trigger → publish)
- Multi-pass review with prompt caching; adaptive contextual memory
- **Humanize** module — capture the user's writing style into profiles and
  generate humanized versions of a review's comments
- Progressive/streaming review feedback; mobile-first responsive redesign
