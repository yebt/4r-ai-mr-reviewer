---
title: Telegram notifications
description: Get pinged when a review or release finishes, straight to a Telegram chat or topic.
---

4R can notify a **Telegram** chat when work finishes — a review completing, a
release landing — so you don't have to sit and poll the UI.

## Register a target

In **Settings → Telegram**, add a target with:

- **Bot token** — from [@BotFather](https://t.me/BotFather).
- **Chat** — a user, group, or channel id, with an optional **topic thread** for
  forum-style groups.

Bot tokens are AES-256-GCM encrypted at rest and never returned by the API.

### Resolve instead of copying ids

Rather than hunting for chat and thread ids by hand, 4R can **resolve** recent
chats and threads straight from the bot's `getUpdates` — send your bot a message,
then pick the destination from the resolved list. You can also send a **test
message** to confirm the wiring before relying on it.

## Notification rules

Notifications are driven by **rules** (**Settings → Notifications**) that match
events to targets. Events include review and release completion. Set a default
target and add rules for the events you care about.

## What ships vs. roadmap

- ✅ **Notifications** — completion pings, test messages, chat/thread resolving.
- 🗺️ **Bot commands** — triggering a review and publishing results *from* chat is
  on the roadmap; today the bot notifies, it doesn't take commands.
