# Deploying ai-reviewer with Dokploy

This guide deploys ai-reviewer on a VPS using [Dokploy](https://dokploy.com)
(Docker Compose). It goes live behind HTTPS on one domain, which is what the
GitLab webhook auto-trigger and the Telegram bot need, and what the cookie-based
login requires (same-origin).

## Topology

ai-reviewer is two pieces: a **Go API server** and a **Vue SPA**. They must be
served from **one origin** because the session is an httpOnly cookie sent
same-origin. The provided setup does exactly that in one Compose project:

```
Internet ──HTTPS──▶ Dokploy/Traefik ──▶ web (nginx) ──┬─ /            → SPA static files
                                                       ├─ /api/*       → api:8080 (prefix stripped)
                                                       ├─ /webhooks/*  → api:8080 (GitLab)
                                                       ├─ /telegram/webhook → api:8080 (Telegram bot)
                                                       └─ /health      → api:8080
                                                              │
                                                          api (Go) ── /data volume (SQLite DB + vault key)
```

- `web` (nginx) is the only service exposed; attach your domain to it.
- `api` is reached only over the internal Docker network (`api:8080`).
- The whole thing is defined by `Dockerfile`, `docker/nginx.conf`, and
  `docker-compose.yml` at the repo root.

## Prerequisites

- A Dokploy server (installed on your VPS) and a domain/subdomain pointing at it.
- This repository reachable by Dokploy (GitHub/GitLab, or a public/deploy-key
  clone).
- A GitLab **Personal Access Token** with `api` + `read_repository` scope (added
  later in the UI, not at deploy time) and at least one AI provider API key.

## 1. Create the app in Dokploy

1. Dokploy → your project → **Create Service → Compose**.
2. Point it at this repository (branch `main`) with **Compose Path**
   `docker-compose.yml`.
3. Build type: **Dockerfile / Compose build** (the compose file builds both
   images from the repo — no registry needed).

## 2. Environment variables

Set these under the Compose app's **Environment** tab. Only two really matter to
start (`AIR_AUTH_PASSWORD` and, implicitly, the persistent volume); the rest have
safe defaults.

| Variable | Required | Default | What it does |
|---|---|---|---|
| `AIR_AUTH_PASSWORD` | **Strongly recommended** | _(empty = no login)_ | Password to log into the web UI/API. **Leaving it empty means anyone with the URL controls your GitLab.** Set it for any public deploy. |
| `AIR_AUTH_SESSION_HOURS` | no | `168` | Login session lifetime (hours). |
| `AIR_TRUST_PROXY` | yes (set to `true`) | `false` | Trust `X-Forwarded-Proto` so the session cookie is marked `Secure` on HTTPS. Required behind Dokploy/Traefik. Already `true` in the compose file. |
| `AIR_TELEGRAM_WEBHOOK_SECRET` | only for the Telegram bot | _(empty = bot dormant)_ | Shared secret validating inbound Telegram bot updates. Notifications (outbound) do **not** need this. |
| `AIR_REVIEW_CONCURRENCY` | no | `2` | How many reviews run in parallel. |
| `AIR_REASONING_BUDGET` | no | `0` | Thinking-token budget; `>0` captures the model's per-phase reasoning. |
| `AIR_ROUTINE_MERGE_TIMEOUT` | no | `1800` | Seconds a release routine waits for a merge/pipeline before it blocks (resumable). |
| `AIR_PASSWORD` | no | _(empty = key-file mode)_ | Optional password to encrypt the secret vault instead of the on-disk key-file. See **Security** below. |
| `AIR_DB_PATH`, `AIR_KEYFILE_PATH`, `AIR_HTTP_ADDR` | no | set in the image | Point at `/data` inside the container — leave as-is. |

> GitLab tokens and provider API keys are **not** env vars. You add them in the
> UI after first login; they're encrypted into the vault on the `/data` volume.

## 3. Persistence (do not skip)

The `api` service stores everything under `/data`, mounted from the named volume
`air-data` in the compose file:

- `ai-reviewer.db` — all your repos, accounts, reviews, routines, rules.
- `ai-reviewer.db.key` — the vault **master key** (key-file mode).

**Losing `/data` loses your data, and losing the key-file makes every stored
GitLab token and API key permanently undecryptable.** Make sure the volume is
persistent (it is, by default) and back it up (see **Backups**).

## 4. Attach the domain and deploy

1. In the Compose app → **Domains**, add your domain and attach it to the **`web`**
   service, port **80**. Enable HTTPS (Let's Encrypt) — Dokploy/Traefik terminate
   TLS and forward `X-Forwarded-Proto=https`, which is why `AIR_TRUST_PROXY=true`.
2. **Deploy.** The first build compiles the Go binary and the SPA (a few minutes).
3. Open `https://your-domain` → you should see the login screen (if
   `AIR_AUTH_PASSWORD` is set) or the Overview.

Health check: `https://your-domain/health` returns `200`.

## 5. First-run setup (in the UI)

1. Log in with `AIR_AUTH_PASSWORD`.
2. **GitLab accounts** → add your GitLab base URL + PAT (`api` + `read_repository`).
3. **AI providers** → add a provider (OpenAI-compatible, Anthropic, or Gemini)
   with its API key.
4. **Repositories** → add a repo (search it by project once the account is set).
5. Run a review from an open merge request.

## 6. Webhooks (need the public domain)

Now that the server is reachable, you can enable the automation that needs a
public URL:

- **GitLab auto-trigger** — per repo → **Webhook** tab → Enable. Copy the URL
  (`https://your-domain/webhooks/gitlab/<repoId>`) and the Secret token into the
  GitLab project's **Settings → Webhooks**, checking **Merge request events**.
  Optionally turn on "Require confirmation" so triggered reviews wait for approval.
- **Telegram bot go-live** — set `AIR_TELEGRAM_WEBHOOK_SECRET`, then register the
  bot webhook at `https://your-domain/telegram/webhook` (bot slice 5c).

Telegram **notifications** (Settings → Notifications) work without any of this —
they only need a bot token + chat id on a notifier target.

## 7. Security notes

- **Set `AIR_AUTH_PASSWORD`.** Without it the API is open to anyone who finds the
  URL, and it can approve/merge/tag on your GitLab.
- **Vault key-file vs password.** By default the master key sits in
  `/data/ai-reviewer.db.key` (key-file mode) — simple, but anyone with the volume
  can decrypt your tokens. Setting `AIR_PASSWORD` encrypts the vault with that
  password instead (no key on disk), at the cost of keeping the password in
  Dokploy's env. Pick per your threat model; if you switch modes, existing
  secrets must be re-entered.
- Keep the domain on HTTPS; the cookie is `Secure` + `HttpOnly`.

## 8. Backups

Back up the `air-data` volume (the `/data` directory) — both files. A simple
approach is a periodic copy of the volume, or `docker run --rm -v air-data:/data
-v "$PWD":/backup alpine tar czf /backup/air-data.tgz -C /data .` on the host.

## 9. Updating

Push to `main`, then hit **Redeploy** in Dokploy. Database migrations run
automatically on startup; the `/data` volume carries your data across
deploys.

## 10. Validate the build locally first (optional but recommended)

Before deploying, confirm the images build on your machine:

```sh
docker compose build          # builds both images from the repo
docker compose up             # api on the internal net, web on :80
# open http://localhost — set AIR_AUTH_PASSWORD in your shell/env first if you
# want the login screen.
```

Then push and deploy in Dokploy.
