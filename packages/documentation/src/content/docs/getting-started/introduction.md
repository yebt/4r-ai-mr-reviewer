---
title: Introduction
description: What 4R is, how it is built, and what is in scope today.
---

**4R** is a self-hosted AI code reviewer for **GitLab merge requests**. It reviews
every change through four independent lenses — **R**isk, **R**eadability,
**R**eliability, **R**esilience — so the issues that matter most surface first,
without slowing down changes that are genuinely safe.

## The idea

You point 4R at a GitLab account and an AI provider, track a repository, and run a
review on any open merge request. For each review the engine:

1. Loads the **4R rule sets** (one per lens).
2. Builds context from the MR — the diff (**fast**) or a shallow clone (**deep**).
3. Runs each lens as its own pass and collects **located, structured findings**
   (file, line, severity, and an issue / why / fix split).
4. Computes a **deterministic 0–100 score** and an approve / request-changes
   recommendation **from the findings** — not by asking the model for a verdict.

You then decide which findings to publish back to the MR as inline discussions,
and can rewrite any of them — or the summary — in your own writing voice first.

## How it is built

A monorepo. The backend owns all state and is the single contract every client
consumes over HTTP.

```
packages/
  server/          Go backend — hexagonal, SQLite (single binary), REST API
  spa/             Vue 3 + TypeScript + Vite + UnoCSS + Pinia web client
  landing-base/    Marketing landing (Astro)
  documentation/   These docs (Astro + Starlight)
```

- **Backend** — Go + SQLite (`modernc.org/sqlite`, pure-Go, so it compiles to a
  single binary), an encrypted secret vault, a bounded job runner, and the 4R
  engine behind strategy interfaces (fast/deep context × single/multi-pass).
  Adapters for GitLab (clone + REST) and Telegram (Bot API) live at the edges.
- **Web** — file-based routing, feature modules, and a borderless
  technical-minimal design system.

## What is in scope today

| Area | Status |
|---|---|
| GitLab MR review (fast + deep) | ✅ Shipped |
| Deterministic scoring & phased multi-pass | ✅ Shipped |
| Multiple providers (OpenAI-compatible + Anthropic) | ✅ Shipped |
| Humanize (voice profiles) | ✅ Shipped |
| Selective publishing | ✅ Shipped |
| AI-drafted merge request creation | ✅ Shipped |
| Release routines (dev + main flows) | ✅ Shipped |
| Telegram notifications | ✅ Shipped |
| Secrets encrypted at rest (AES-256-GCM) | ✅ Shipped |
| GitHub support | 🗺️ Roadmap |
| Webhook auto-trigger | 🗺️ Roadmap |
| Telegram bot commands (trigger from chat) | 🗺️ Roadmap |
| Auth & multi-user | 🧪 Partial — API auth ships; login UI and multi-user are roadmap |

:::caution[Single-user]
4R is designed to run for one person / one team on trusted infrastructure. It is
GitLab-first and single-user today. See [Authentication](/self-hosting/authentication/)
before exposing it beyond localhost.
:::

Next: [Quick start](/getting-started/quick-start/).
