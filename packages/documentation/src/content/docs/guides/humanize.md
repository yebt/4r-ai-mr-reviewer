---
title: Humanize — your voice
description: Capture your writing style into profiles and rewrite review feedback in your own voice.
---

Review feedback reads better when it sounds like a person, not a linter.
**Humanize** rewrites a finished review's parts — the summary and each finding —
in an author's writing voice, without changing the technical substance.

## Voice profiles

A **profile** captures a writing voice. You create one in **Settings → Profiles**
with:

- **Writing samples** — a few pieces of your own writing pasted in.
- **Language**, **formality**, and whether the voice uses **emojis**.

From the samples and those knobs, the backend **distills a style guide**
asynchronously (one LLM pass). A profile's status moves `pending → ready`; only a
**ready** profile can humanize. If distillation fails you can re-trigger it.

## Rewriting a review

On a finished review, pick a profile and humanize:

- **A finding** — its *issue / why / fix* parts are rewritten in the voice, each
  part kept separate. An empty part stays empty; nothing is invented or moved
  between parts.
- **The summary** — rewritten as prose in the voice.

Each humanized result is **persisted**, so it survives a page reload and appears as
its own tab. The original generated text is always kept alongside.

## Publishing humanized text

When you publish findings, you can post the **humanized** version instead of the
generated one — per finding, and for the summary. Pick the voice that fits the
audience, then send.

:::note[Per-repo default profile]
A tracked repository can carry a **default voice profile**, so the humanize picker
pre-selects it for that repo's reviews.
:::

## Where else voice is used

The same voice profiles can style an [AI-drafted merge request
description](/guides/merge-requests/) — choose a profile there to write the MR body
in that voice, or leave it in plain English.
