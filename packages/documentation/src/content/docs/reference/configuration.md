---
title: Configuration
description: Every backend environment variable, with defaults and purpose.
---

The backend is configured entirely through environment variables. All are
optional — 4R runs with sensible defaults out of the box.

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `AIR_HTTP_ADDR` | `:8080` | Listen address. |
| `AIR_DB_PATH` | `ai-reviewer.db` | SQLite database file. |
| `AIR_PASSWORD` | _(empty)_ | Unlocks the secret vault. Empty → **key-file mode**. |
| `AIR_KEYFILE_PATH` | `<db>.key` | Master key file (key-file mode). |
| `AIR_SKILLS_DIR` | _(empty)_ | Override directory for the 4R rule files. |
| `AIR_REVIEW_CONCURRENCY` | `2` | Max reviews running in parallel (min 1). |
| `AIR_REASONING_BUDGET` | `0` | Per-phase reasoning capture. `0` = off. A positive value is the Anthropic thinking-token budget **and** enables capture of reasoning from OpenAI-compatible providers (clamped to 32768). |
| `AIR_AUTH_PASSWORD` | _(empty)_ | Enables API auth (password + signed-cookie sessions). Empty → auth disabled, every route open. |
| `AIR_AUTH_SESSION_HOURS` | `168` | Session-cookie lifetime in hours (7 days); clamped to `1`..`8760`. |
| `AIR_TRUST_PROXY` | `false` | Trust client `X-Forwarded-Proto` / `X-Forwarded-For`. Set `true` **only** behind a trusted TLS-terminating proxy. |

## The secret vault

Secrets (GitLab tokens, provider API keys, bot tokens) are encrypted at rest with
AES-256-GCM. The master key is derived one of two ways:

- **Key-file mode** (default, `AIR_PASSWORD` empty) — a random master key is stored
  in `AIR_KEYFILE_PATH` (next to the database by default). Keep this file safe and
  back it up with the database.
- **Password mode** (`AIR_PASSWORD` set) — the key is derived from your password
  (PBKDF2-HMAC-SHA256). Lose the password and the secrets can't be decrypted.

You can change the master password or re-key at runtime from **Settings →
Security**. See [Backups](/self-hosting/backups/) for what to preserve.

## Auth footguns

Before enabling `AIR_AUTH_PASSWORD`, read [Authentication](/self-hosting/authentication/):

- **Don't lock yourself out** — the SPA login UI is required once auth is on.
- **Sessions are stateless** — a leaked token stays valid until it expires; rotate
  `AIR_AUTH_PASSWORD` to revoke all sessions immediately.
- **Leave `AIR_TRUST_PROXY=false`** unless a trusted proxy terminates TLS in front.
