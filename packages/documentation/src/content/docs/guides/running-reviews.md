---
title: Running reviews
description: Fast vs deep context, the review lifecycle, and reasoning capture.
---

## Launching a review

From a repository's **Flow** workspace, open the **MRs** tab, pick an open merge
request, and press **Review**. The launch modal lets you choose the provider,
model, and context mode for this run; leave them to fall back to the repo or
default provider.

## Fast vs deep context

The **context mode** decides how much the model sees:

| Mode | What it sends | When to use |
|---|---|---|
| **Fast** | The MR diff and the touched files, fetched over the GitLab API. | Most reviews. Quick, no clone. |
| **Deep** | A **shallow clone** of the branch for broader repository context. | When findings need surrounding code the diff alone doesn't show. |

:::caution[Deep needs `read_repository`]
Deep reviews clone over HTTPS, so the GitLab token must include the
**`read_repository`** scope in addition to `api`. An `api`-only token passes fast
reviews but is rejected at clone time.
:::

## The lifecycle

Reviews run **asynchronously**; the UI polls for status. A review moves through
states you can act on:

- **Running** — passes execute; progress streams per lens. You can **cancel** (it
  is cooperative — the run flips to `cancelled` shortly after).
- **Done** — summary, score, recommendation, and findings are ready to publish.
- **Error** — something failed. **Retry** clones the review into a fresh run and
  **keeps the failed one for history** (a new review id — humanizations and other
  per-review state are never carried across).
- **Archive / Delete** — tidy the lists from the detail view or the list views.

## Concurrency

The backend runs a bounded number of reviews in parallel — set by
`AIR_REVIEW_CONCURRENCY` (default **2**, minimum 1). Launch more than that and the
extra reviews queue and start as slots free up.

## Reasoning capture (optional)

Set `AIR_REASONING_BUDGET` to a positive value to capture per-phase model
reasoning:

- `0` (default) — off; no reasoning is stored for any provider.
- A positive value — used as the Anthropic *thinking-token* budget **and** enables
  capture of reasoning returned by OpenAI-compatible providers. Clamped to 32768.

See [Configuration](/reference/configuration/) for the full variable reference.

## Publishing

Once a review is done, select the findings worth posting and **publish** — each
becomes an inline discussion anchored to its `file:line`. You can also post them
all. Already-published findings are never re-posted. Optionally
[humanize](/guides/humanize/) the text first.
