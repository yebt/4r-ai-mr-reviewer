---
title: Your first review
description: Configure an account, a provider and a repo, then run a review end to end.
---

This walks through the full loop once: **configure → review → publish**.

## 1. Add a GitLab account

In **Settings → Accounts**, add your GitLab instance:

- **Base URL** — e.g. `https://gitlab.com` or your self-managed host.
- **Personal access token** — with the `api` scope (add `read_repository` if you
  want deep reviews; see [Running reviews](/guides/running-reviews/)).

The token is encrypted at rest immediately and never returned by the API.

## 2. Add an AI provider

In **Settings → Providers**, add a provider and mark one as the **default**:

- **Kind** — OpenAI-compatible or Anthropic.
- **Base URL & model** — e.g. Groq / OpenAI / OpenRouter endpoint and a model id.
- **API key** — encrypted at rest.

You can **test** the connection from the form before saving. See
[Providers & models](/guides/providers-and-models/) for precedence rules
(default → per-repo → per-review).

## 3. Track a repository

In **Repositories**, add the GitLab project you want to review (by URL). Optionally
set a **per-repo provider/model** and a **default voice profile** for humanizing.

## 4. Run a review

Open the repo's **Flow** workspace. In the **MRs** tab, pick an open merge request
and press **Review**. In the launch modal choose:

- **Provider / model** — or leave the repo/default.
- **Context mode** — **fast** (diff over the API) or **deep** (shallow clone for
  more context).

The review runs asynchronously. Each 4R lens runs as its own **pass**, so progress
streams in. When it finishes you get:

- A **summary** and a **deterministic score** (0–100).
- A **recommendation** — approve or request changes.
- A list of **located findings**, each with a lens, severity, `file:line`, and an
  issue / why / fix breakdown.

## 5. Publish what matters

From the review detail view, select the findings worth posting and **publish** —
they become inline discussions on the merge request. Already-published findings
are never re-posted on a later publish.

Want the feedback in your own tone? Capture a [voice profile](/guides/humanize/)
and generate a humanized version of the summary or any finding before publishing.

## What next?

- [Running reviews](/guides/running-reviews/) — fast vs deep, lifecycle, scoring.
- [Creating merge requests](/guides/merge-requests/) — draft an MR with AI.
- [Release routines](/guides/release-routines/) — automate dev/main releases.
