---
title: Providers & models
description: Configure AI providers, set defaults, and understand the precedence rules.
---

4R talks to AI providers over their APIs. You configure one or more, mark one as
the **default**, and optionally override per repo or per review.

## Supported kinds

- **OpenAI-compatible** — any endpoint that speaks the OpenAI chat API, including
  **Groq, OpenAI, Moonshot / Kimi, and OpenRouter**.
- **Anthropic** — Claude models.

## Adding a provider

In **Settings → Providers**, add a provider with its **kind**, **base URL**,
**model**, and **API key**. You can **test** the connection before saving — it runs
one minimal live call so credential, endpoint, or model errors surface at
configuration time instead of inside a long review.

The first provider you add becomes the default automatically. API keys are
AES-256-GCM encrypted at rest and never returned by the API.

:::note[OpenRouter model browser]
When you use OpenRouter, the provider form can browse the OpenRouter model catalog
so you can pick a model instead of typing an id.
:::

## Precedence

For any given review, the provider and model are resolved in order:

```
provider:  per-review override  →  repo provider  →  default provider
model:     per-review override  →  repo model     →  provider's model
```

- **Default provider** — used when nothing more specific is set.
- **Per-repo** — set a provider/model on a tracked repository to pin its reviews.
- **Per-review** — the launch modal lets you override both for a single run.

The **resolved model is persisted on the review**, so the UI always shows which
model actually ran — even when it came from the repo or default provider.

## Temperature & reasoning

Each provider carries an optional **temperature**. Reasoning capture is controlled
globally by `AIR_REASONING_BUDGET` (see [Running reviews](/guides/running-reviews/)
and [Configuration](/reference/configuration/)).
