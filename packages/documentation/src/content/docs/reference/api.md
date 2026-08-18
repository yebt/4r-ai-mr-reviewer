---
title: HTTP API
description: The backend's JSON HTTP API — the contract every client consumes.
---

The backend owns all state and exposes a **JSON HTTP API** that every client (the
web SPA, the Telegram bot, any script) consumes. There is no separate business
logic in the clients — they are thin over this contract.

## Shape

- **Transport** — JSON over HTTP, served at `AIR_HTTP_ADDR` (default `:8080`).
- **Auth** — off by default; when `AIR_AUTH_PASSWORD` is set, routes require a
  valid signed-cookie session (see [Authentication](/self-hosting/authentication/)).
- **Secrets** — write-only. Tokens and API keys are accepted and encrypted, but the
  API **never returns** them.

## Full reference

The complete, authoritative reference lives in the repository:

- **`docs/API.md`** — the full endpoint reference.
- **`docs/api.http`** — a runnable request collection (open it in an HTTP client
  such as the VS Code REST Client or JetBrains HTTP client).

## Example surface

A few representative endpoint groups (see `docs/API.md` for the full list):

| Group | Examples |
|---|---|
| Accounts / providers | `POST /accounts`, `POST /providers`, `POST /providers/test` |
| Repos | `GET /repos`, `GET /repos/{id}/merge-requests`, `GET /repos/{id}/branches` |
| Reviews | `POST /reviews`, `GET /reviews/{id}`, `POST /reviews/{id}/publish` |
| Merge requests | `POST /repos/{id}/merge-requests/generate`, `POST /repos/{id}/merge-requests` |
| Routines | `POST /repos/{id}/routines/release-main`, `GET /repos/{id}/routines/preview-tag` |
| Vault | `GET /vault/status`, `POST /vault/password` |

:::note
The Vite dev server proxies `/api` to the backend during development, so the SPA
calls `/api/...` while the backend serves the paths above.
:::
