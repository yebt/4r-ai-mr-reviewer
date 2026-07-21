# Telegram bot interaction — design

Status: **design (not enabled yet).** The receiver is built webhook-first; going
live is a later step (register the webhook once the server is publicly
reachable). Everything below is testable locally by POSTing a fake `Update` to
the endpoint — no public URL needed to develop it.

## Goal

Let a trusted user drive the app from a Telegram chat:

1. **View repos** — list the configured repositories.
2. **View a repo's MRs** — list its open merge requests.
3. **Trigger a review** — start a review on a chosen MR (fast/deep).
4. **View a review's content** — status, score, recommendation, summary, findings.
5. **List reviews** — recent reviews across repos.
6. **Open a review** — drill into one from the list.

Publishing findings from chat is intentionally out of scope for this first pass.

## Mechanism: webhook (long-polling rejected)

Telegram delivers updates by **HTTP POST to a URL we register** (`setWebhook`).

- **Endpoint:** `POST /telegram/webhook` — receives an `Update` JSON body.
- **Endpoint auth:** on `setWebhook` we set a `secret_token`; Telegram then sends
  it back on every call in the `X-Telegram-Bot-Api-Secret-Token` header. The
  handler **rejects any request** whose header doesn't match the stored secret.
  This is what keeps random internet traffic from injecting fake updates.
- **Enablement is deferred.** The endpoint exists and is unit-testable now; it
  only receives real traffic once (a) the server is reachable over HTTPS and
  (b) we call `setWebhook(url, secret)` on the bot token. Until then it's dormant.
- **Local/dev testing:** `curl -X POST /telegram/webhook -H 'X-Telegram-Bot-Api-Secret-Token: <secret>' -d '<fake update>'`
  exercises the whole flow without Telegram.

> Long-polling was rejected: it needs a persistent `getUpdates` loop that would
> also collide with the **Resolve** feature (single `getUpdates` consumer per
> token) and offers no benefit here since we accept building a webhook endpoint.

## Bot identity & token

Replying (`sendMessage`, answering callback queries) needs a bot token. We
designate **one interactive bot token** — the webhook is registered on it and all
replies use it.

- MVP: reuse the bot token of a designated Telegram target (a `isBot`/`interactive`
  flag on the target), or a dedicated "bot" setting. Decision to finalize at build
  time; the receiver just needs "the token to reply with".

## Authorization (critical)

The webhook can be hit by anyone who knows the URL+secret, and the bot can be
messaged by any Telegram user. Two gates:

1. **Transport:** the secret-token header (above) authenticates that the request
   is really from Telegram.
2. **Actor allowlist:** every update carries `message.from.id` (user) and
   `message.chat.id`. We only act on updates whose chat/user is on an
   **allowlist**. MVP allowlist source: the chat IDs of the configured
   notification targets (chats you already trust), or an explicit allowed-chats
   setting. Unlisted actors get a polite "not authorized" reply (or silence).

No write action (trigger review) runs for an unauthorized actor.

## Interaction model: inline keyboards + callback queries

Button-driven, not typed commands. The bot sends messages with inline keyboards;
a tap sends a **callback query** with compact `callback_data` we route on.

Entry points (typed commands, minimal):

- `/start` / `/menu` → main menu: `[ Repos ]  [ Reviews ]`

Navigation (all via buttons):

```
Main menu
 ├─ Repos ───────────► list repos as buttons        cb: repo:<repoId>
 │                       └─ tap repo ──────────────► repo view
 ├─ Reviews ─────────► list recent reviews           cb: rv:<reviewId>
 │                       └─ tap review ────────────► review view

Repo view (repo:<repoId>)
 ├─ open MRs as buttons                              cb: mr:<repoId>:<iid>
 │    └─ tap MR ─────► MR view
 └─ [ Recent reviews ]                               cb: rr:<repoId>

MR view (mr:<repoId>:<iid>)
 ├─ [ Review · fast ]                                cb: go:<repoId>:<iid>:fast
 └─ [ Review · deep ]                                cb: go:<repoId>:<iid>:deep
      └─ triggers create → reply "queued/started" + [ View review ] cb: rv:<reviewId>

Review view (rv:<reviewId>)
 └─ status, score, recommendation, summary, findings count
    (auto-updates on next open; no live push in v1)
```

### `callback_data` scheme

Telegram caps `callback_data` at **64 bytes**. Our IDs are 128-bit hex (~32
chars), so one ID per callback fits; `go:<repoId>:<iid>:<mode>` (~44 chars) also
fits. Prefixes: `repo` `mr` `rr` (repo-reviews) `rv` (review) `go` (trigger).

Parsing is a small dispatcher: split on `:`, switch on the prefix, load the
entity, render the next view (edit the message in place with the new keyboard, or
send a new message).

## Reply rendering

- Parse mode **MarkdownV2** or **HTML** (HTML is easier to escape reliably).
- Review view: recommendation + score header, the summary, then findings as a
  compact list (dimension · severity · file:line · issue). Reuse the existing
  finding-to-text formatting where possible.

## Server shape (build sketch, when approved)

- `internal/adapters/telegram/client.go` — add `SendMessage` with an inline
  keyboard, `AnswerCallbackQuery`, `EditMessageText`, `SetWebhook`/`DeleteWebhook`.
- `internal/app/bot/` (new) — `Service` that takes a parsed `Update`, checks the
  allowlist, dispatches on message text / callback_data, calls the existing
  reviews/repos services, and replies via the telegram client. Pure logic →
  unit-testable with fake updates.
- `internal/http/handlers_telegram_webhook.go` — `POST /telegram/webhook`:
  validate the secret header, decode the `Update`, hand it to `bot.Service`,
  return 200 quickly (Telegram retries on non-2xx; do slow work async).
- Reuse `reviews.Service` (Create, Get, List, ListOpenMergeRequests) and
  `repos.Service` (List) — no new review logic.

## Rollout slices

- **5a — Design (this doc).**
- **5b — Bot core, webhook-ready but dormant.** Endpoint + secret validation +
  allowlist + dispatcher + the six actions (repos, MRs, trigger, review content,
  list reviews, open one). Fully testable via POSTed fake updates + Go tests. Not
  yet receiving real traffic.
- **5c — Enable webhook.** Add a `setWebhook`/`deleteWebhook` admin action (SPA
  button or one-off), point Telegram at the public URL, go live. Couples with the
  GitLab webhook auto-trigger work (#18) since both need the server reachable.
- **Later — Publish findings from chat.**

## Open decisions (finalize at 5b)

- Where the interactive bot token lives (target flag vs dedicated setting).
- Allowlist source (target chat IDs vs explicit setting).
- Reply parse mode (HTML recommended).
- Message editing vs new messages on navigation (editing is cleaner but more code).
