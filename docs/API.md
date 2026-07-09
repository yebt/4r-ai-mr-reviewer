# ai-reviewer — HTTP API

The backend (`packages/server`) exposes a JSON HTTP API. It is the single
contract every client (TUI, SPA, Telegram bot) consumes.

- **Base URL (default):** `http://localhost:8080`
- **Content type:** `application/json`
- **Auth:** none yet — MVP is single-user and local. The app password only
  unlocks the secret vault at startup; it is not an HTTP credential.

Secrets (GitLab tokens, provider API keys) are **write-only** through the API:
you send them on create, but they are never returned in any response.

## Configuration (environment variables)

| Variable | Default | Purpose |
|---|---|---|
| `AIR_HTTP_ADDR` | `:8080` | Listen address |
| `AIR_DB_PATH` | `ai-reviewer.db` | SQLite database file |
| `AIR_KEYFILE_PATH` | `<db>.key` | Master key file (key-file mode) |
| `AIR_PASSWORD` | _(empty)_ | Unlocks the vault; empty → key-file mode |
| `AIR_SKILLS_DIR` | _(empty)_ | Override dir for the 4R skill files |

## Conventions

- IDs are 32-char hex strings.
- Errors return `{"error": "message"}` with an appropriate status:
  `404` not found, `400` bad request, `502` upstream (GitLab) failure.
- `provider.kind` is `openai-compat` (Groq, OpenAI, Moonshot, Kimi, OpenRouter)
  or `anthropic` (Claude).
- `review.mode` is `fast` (diff + touched files) or `deep` (shallow clone).

---

## Health

### `GET /health`
```json
{ "status": "ok" }
```

---

## Accounts (GitLab)

### `POST /accounts` → `201`
```json
{ "name": "work", "baseUrl": "https://gitlab.com", "token": "glpat-xxxxx" }
```
Response:
```json
{ "id": "…", "name": "work", "baseUrl": "https://gitlab.com", "createdAt": "…" }
```

### `GET /accounts` → `200`
Array of accounts (never includes the token).

### `DELETE /accounts/{id}` → `204`
Removes the account, its token, and cascades to its repos.

---

## Providers (AI)

### `POST /providers` → `201`
```json
{
  "name": "groq",
  "kind": "openai-compat",
  "baseUrl": "https://api.groq.com/openai/v1",
  "model": "llama-3.3-70b-versatile",
  "apiKey": "gsk_xxxxx",
  "makeDefault": true,
  "temperature": null,
  "models": ["llama-3.3-70b-versatile", "moonshotai/kimi-k2"]
}
```
The first provider created becomes the default automatically. For `anthropic`,
`baseUrl` may be omitted (defaults to `https://api.anthropic.com`).

`temperature` is optional: `null` (or omitted) means the review will **not send
a temperature**, so the model uses its default — required by models that reject
any other value. `models` is a list of preset model names offered when
configuring a repository.

### `GET /providers` → `200`
Array of providers (never includes the API key).

### `PATCH /providers/{id}` → `200`
Edits a provider. `apiKey` is optional — an empty string keeps the stored key.
`temperature` (`null` = don't send) and `models` are the advanced settings.
```json
{ "name": "groq-eu", "kind": "openai-compat", "baseUrl": "https://eu", "model": "llama-3.3", "apiKey": "", "temperature": 0.2, "models": ["llama-3.3"] }
```

### `POST /providers/{id}/default` → `200`
Makes this provider the sole default.

### `DELETE /providers/{id}` → `204`

---

## Repositories

### `POST /repos` → `201`
```json
{
  "name": "web",
  "url": "https://gitlab.com/group/project",
  "accountId": "…",
  "providerId": "",
  "model": ""
}
```
`providerId`/`model` are optional. Empty `providerId` uses the default provider;
empty `model` uses the provider's model.

### `GET /repos` → `200`
Array of repos.

### `PATCH /repos/{id}/assign` → `200`
```json
{ "providerId": "…", "model": "gpt-4o" }
```
Empty `providerId` clears the assignment (falls back to default).

### `DELETE /repos/{id}` → `204`

### `GET /repos/{id}/merge-requests` → `200`
Live-fetches the repo's **open** MRs from GitLab.
```json
[ { "iid": 7, "title": "Add login", "state": "opened",
    "sourceBranch": "feat", "targetBranch": "main",
    "webUrl": "…", "author": "yahir" } ]
```

### `GET /repos/{id}/reviews` → `200`
The repo's reviews, newest first (without findings).

---

## Reviews

### `POST /reviews` → `201`
```json
{ "repoId": "…", "mrIid": 7, "mode": "fast" }
```
Creates a **pending** review and enqueues it. The review runs asynchronously;
poll `GET /reviews/{id}` for status (`pending → running → done | error`).

### `GET /reviews/{id}` → `200`
Full review including findings:
```json
{
  "id": "…", "repoId": "…", "mrIid": 7, "contextMode": "fast",
  "status": "done", "summary": "…",
  "recommendation": "request_changes", "score": 75,
  "inputTokens": 1200, "outputTokens": 300,
  "findings": [
    { "index": 0, "dimension": "risk", "severity": "high",
      "file": "auth.go", "line": 42, "issue": "hardcoded secret",
      "why": "…", "fix": "…", "blocking": true, "published": false }
  ],
  "createdAt": "…", "updatedAt": "…"
}
```

### `POST /reviews/{id}/retry` → `201`
Clones the review's configuration into a fresh pending review. The original
(errored) review is kept for history.

### `POST /reviews/{id}/publish` → `200`
Publishes selected findings to the merge request as inline discussions (or a
general note when a finding has no line), plus a summary note.
```json
{ "all": true }
```
or select specific findings by their `index`:
```json
{ "indices": [0, 2] }
```
Published findings are marked so a re-publish will not duplicate them.

---

## Typical flow

1. `POST /accounts` — add a GitLab account.
2. `POST /providers` — add an AI provider (first is default).
3. `POST /repos` — track a repo, optionally pin a provider/model.
4. `GET /repos/{id}/merge-requests` — find an open MR.
5. `POST /reviews` — start a review for that MR.
6. `GET /reviews/{id}` — poll until `done`.
7. `POST /reviews/{id}/publish` — push the findings you approve.
