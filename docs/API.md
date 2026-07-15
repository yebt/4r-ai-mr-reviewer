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

### `GET /skills`
The 4R review rule sets currently loaded by the engine (read-only).
```json
{ "risk": "# R1 — Risk…", "readability": "…", "reliability": "…", "resilience": "…" }
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

## Humanization profiles

A humanization profile captures a user's writing voice, used in a later slice to
rephrase review comments in their own style. Samples are **not** secret and are
stored in the clear (unlike provider API keys).

`styleGuide` is **server-managed**: it is an LLM-distilled cache and is read-only
(it cannot be set by the client). Whenever a profile is created or updated **with
non-empty `samples`**, the server distills those samples plus the knobs
(`language`, `formality`, `emojis`) into `styleGuide` via **one asynchronous LLM
call on the default provider**. The CRUD request does **not** block on it: the
response returns immediately with `styleGuideStatus: "pending"`.

Two read-only fields report distillation state:

- `styleGuideStatus` — one of:
  - `""` (none): no samples, so nothing to distill (any prior guide is cleared).
  - `"pending"`: distillation was triggered and is in flight.
  - `"ready"`: `styleGuide` holds the distilled result.
  - `"error"`: the last attempt failed; see `styleGuideError`.
- `styleGuideError` — the failure message when `styleGuideStatus` is `"error"`,
  empty otherwise.

Distillation survives restarts: profiles left `pending` are re-triggered at
startup.

### `POST /profiles` → `201`
```json
{
  "name": "casual-es",
  "language": "es-AR",
  "formality": "casual",
  "emojis": true,
  "samples": ["che, mirá esto", "buenísimo el cambio"]
}
```
Only `name` is required. `language` and `formality` are free text (e.g.
`casual`, `neutral`, `formal`). The response includes `styleGuide`,
`styleGuideStatus`, `styleGuideError`, `createdAt` and `updatedAt`. With samples,
`styleGuideStatus` is `"pending"` on return (distillation runs in the
background); without samples it is `""` (none).

### `GET /profiles` → `200`
Array of profiles, ordered by name.

### `GET /profiles/{id}` → `200`
A single profile.

### `PATCH /profiles/{id}` → `200`
Edits a profile. `name` is required; all fields are replaced with the payload.
`styleGuide` is ignored (server-managed). If the payload carries samples,
distillation is re-triggered (`styleGuideStatus: "pending"`); if samples are
removed, the guide is cleared (`styleGuideStatus: ""`).
```json
{ "name": "casual-es", "language": "es", "formality": "formal", "emojis": false, "samples": ["a", "b"] }
```

### `DELETE /profiles/{id}` → `204`

### `POST /profiles/{id}/redistill` → `200`
Manually re-runs style-guide distillation for a profile. Sets the status to
`pending` and triggers the async LLM call. Returns `404` if the profile does not
exist.
```json
{ "status": "pending" }
```

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
The repo's **active** reviews, newest first (without findings). Pass
`?archived=1` (or `?archived=true`) to return the repo's **archived** reviews
instead; the default returns active only.

---

## Reviews

### `POST /reviews` → `201`
```json
{ "repoId": "…", "mrIid": 7, "mode": "fast" }
```
Creates a **pending** review and enqueues it. The review runs asynchronously as
a **multi-pass 4R review** — one focused model call per lens (Risk → Readability
→ Reliability → Resilience). Poll `GET /reviews/{id}` for status
(`pending → running → done | error | cancelled`) and `phase` (the current lens
while running: `risk`/`readability`/`reliability`/`resilience`, empty otherwise).

### `GET /reviews/{id}` → `200`
Full review including findings. `status` is one of
`pending`/`running`/`done`/`error`/`cancelled`:
```json
{
  "id": "…", "repoId": "…", "mrIid": 7, "contextMode": "fast",
  "status": "done", "phase": "", "archived": false, "summaryPublished": false, "summary": "…",
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

### `DELETE /reviews/{id}` → `204`
Hard-removes the review and all its findings. Returns `404` if it does not exist.

### `POST /reviews/{id}/retry` → `201`
Clones the review's configuration into a fresh pending review. The original
(errored) review is kept for history.

### `POST /reviews/{id}/cancel` → `200`
Requests cooperative cancellation of a **pending** or **running** review. The
request returns immediately; a running review aborts its in-flight model call and
flips to the `cancelled` terminal state shortly after (observe it by polling
`GET /reviews/{id}`). Returns `409` if the review is already in a terminal state
(`done`/`error`/`cancelled`), and `404` if it does not exist.
```json
{ "status": "cancelling" }
```

### `POST /reviews/{id}/archive` → `200`
Soft-hides the review from the active list (`GET /repos/{id}/reviews`) while
keeping its full history. Returns `404` if the review does not exist.
```json
{ "status": "archived" }
```

### `POST /reviews/{id}/unarchive` → `200`
Restores an archived review to the active list. Returns `404` if the review does
not exist.
```json
{ "status": "unarchived" }
```

### `POST /reviews/{id}/humanize` → `200`
Rewrites **one target** of a finished review — a single finding, or the summary —
in a humanization profile's author voice, returning the rewrite as structured
JSON. Each successful rewrite is **persisted** as a new tab (see
`GET /reviews/{id}/humanizations`) so it survives a page reload; the rewrite is
still returned inline so the caller can show it immediately. It uses the
**default provider** (like style-guide distillation), not the repo's provider.

Callers rewrite one target per request, so a finding card and the summary are
humanized by independent, concurrent calls. The `target` field selects what to
rewrite:

Rewrite a single finding by its zero-based `index`:
```json
{ "profileId": "…", "target": "finding", "index": 0 }
```
Response — each part of the finding is rewritten separately so nothing is lost
when the caller reassembles the comment body. Only the VOICE/phrasing changes;
the technical substance is preserved. A part that is empty in the original stays
empty (it is never fabricated):
```json
{
  "issue": "…issue in the author's voice…",
  "why": "…why it matters, in the author's voice…",
  "fix": "…fix in the author's voice…"
}
```

Rewrite the review summary:
```json
{ "profileId": "…", "target": "summary" }
```
Response:
```json
{ "summary": "…rephrased summary…" }
```

Guards and error codes:
- `400` — `target` is missing or not `finding`/`summary`, or the finding `index`
  is outside the review's finding range.
- `404` — the review or the profile does not exist.
- `409` — the review is not `done`, or the profile's style guide is not `ready`
  (the message includes the actual status, e.g. `style guide not ready (pending)`).
- `502` — the upstream LLM call failed or its output could not be parsed.

### `GET /reviews/{id}/humanizations` → `200`
Returns every **persisted** humanize run of a review, grouped so a client can
rehydrate its humanize tabs after a page reload. `summary` is the ordered list of
summary rewrites; `findings` maps a finding's zero-based index (as a string key,
since JSON object keys are strings) to its ordered list of rewrites. Within each
group the tabs preserve the order in which they were produced. Empty groups are
omitted from `findings`; `summary` is `[]` when the summary was never humanized.
```json
{
  "summary": [{ "summary": "…rephrased summary…" }],
  "findings": {
    "0": [{ "issue": "…", "why": "…", "fix": "…" }],
    "3": [{ "issue": "…", "why": "…", "fix": "…" }]
  }
}
```
Known limitation: a finding rewrite is keyed by the finding's index at rewrite
time. Retrying a review regenerates its findings, so rewrites produced before a
retry may point at a now-stale index; they are returned as-is and not reconciled.

### `POST /reviews/{id}/publish` → `200`
Publishes selected findings to the merge request as inline discussions (or a
general note when a finding has no line), plus a summary note.
```json
{ "all": true }
```
`all` posts only findings that are **not yet published**, so a repeated
"publish all" never re-comments what is already on the merge request. To
re-post a specific finding deliberately, select it by `index` — explicit
indices are honored as-is (a re-selection may re-post an already-published one):
```json
{ "indices": [0, 2] }
```
The summary note posts on the **first** publish only. Pass the optional
`includeSummary` flag to override this: `true` re-posts the summary even if it
was already posted (re-selectable), `false` suppresses it. When omitted, the
summary posts only while `summaryPublished` is still `false`.
```json
{ "all": true, "includeSummary": true }
```
Optionally override the posted text with humanized versions. `summaryOverride`
(string) replaces the generated summary body, and `findingOverrides`
(`[{ "index", "text" }]`) replaces the generated body of each listed finding.
When provided, the humanized text is posted **as-is** — it is treated as a
self-contained comment, so the dimension/severity/blocking header is **not**
prepended. Omit them to keep the generated bodies unchanged.
```json
{
  "all": true,
  "summaryOverride": "Nice work overall — a couple of things to tighten up.",
  "findingOverrides": [
    { "index": 0, "text": "This log line leaks the API token; let's redact it." }
  ]
}
```
Published findings are marked so a re-publish will not duplicate them, and
`summaryPublished` flips to `true` once the summary has been posted.

---

## Typical flow

1. `POST /accounts` — add a GitLab account.
2. `POST /providers` — add an AI provider (first is default).
3. `POST /repos` — track a repo, optionally pin a provider/model.
4. `GET /repos/{id}/merge-requests` — find an open MR.
5. `POST /reviews` — start a review for that MR.
6. `GET /reviews/{id}` — poll until `done`.
7. `POST /reviews/{id}/publish` — push the findings you approve.
