---
title: Quick start
description: Prerequisites, run 4R locally, and open the web UI.
---

Get 4R running locally in a couple of minutes.

## Prerequisites

- **Go 1.26+**
- **Node 22+** (or **bun**)
- **git**
- A **GitLab** account with a personal access token
- An **AI provider** API key (an OpenAI-compatible provider such as Groq, OpenAI,
  Moonshot, Kimi or OpenRouter, or Anthropic)

:::tip[GitLab token scopes]
`api` is enough for **fast** reviews. **Deep** reviews clone the repo over HTTPS,
so the token also needs **`read_repository`**. An `api`-only token passes fast
reviews but is rejected at clone time.
:::

## Run it

From the repository root:

```sh
# backend + SPA together (backend :8080, SPA :5173)
make dev

# …or separately
make run-server
make run-spa
```

Then open **<http://localhost:5173>**. The Vite dev server proxies `/api` to the
backend, so there is no CORS setup to do.

## First run & the vault

On first start, 4R creates its SQLite database (`ai-reviewer.db`) and an encrypted
**secret vault** that stores your GitLab tokens and provider keys. By default it
runs in **key-file mode** — a master key file is generated next to the database
(`<db>.key`). To use a password instead, set `AIR_PASSWORD`; see
[Configuration](/reference/configuration/) and [Backups](/self-hosting/backups/).

:::caution
Keep the key file (or your `AIR_PASSWORD`) safe. Without it the encrypted secrets
in the database cannot be decrypted.
:::

## Next steps

- [Your first review](/getting-started/first-review/) — configure an account, a
  provider and a repo, then run a review end to end.
- [Configuration](/reference/configuration/) — every backend environment variable.
- [Deploy self-hosted](/self-hosting/deploy/) — ship the single binary to a server.
