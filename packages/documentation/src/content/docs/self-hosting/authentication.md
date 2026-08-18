---
title: Authentication
description: Enabling API auth safely — sessions, rotation, and proxy trust.
---

By default 4R runs with **auth disabled**: every route is open. That's fine on
localhost or a private network. Before exposing it more widely, enable API auth —
and read these footguns first.

## Enabling auth

Set **`AIR_AUTH_PASSWORD`**. The API then requires a valid **signed-cookie
session**, obtained by logging in with that password. Session lifetime is
`AIR_AUTH_SESSION_HOURS` (default 168 = 7 days, clamped `1`..`8760`).

## Footguns

:::caution[Don't lock yourself out]
Once auth is on, the **SPA login UI is required** to use the app. Make sure your
deployed SPA build includes the login flow before setting `AIR_AUTH_PASSWORD`, or
you'll be shut out of the web UI.
:::

**Sessions are stateless.** A session is a self-contained signed token. Logout is
client-side only — a **leaked token stays valid until it expires**. To revoke
**every** outstanding session immediately, **change `AIR_AUTH_PASSWORD`**: that
rotates the signing key and invalidates all existing tokens.

**Proxy trust.** Leave `AIR_TRUST_PROXY=false` unless a trusted proxy terminates
TLS in front of the backend. When it's `false`, the cookie `Secure` flag and the
login rate-limit IP come only from the real connection, so a client can't spoof
`X-Forwarded-Proto` / `X-Forwarded-For` to weaken them. See
[Reverse proxy & TLS](/self-hosting/reverse-proxy/).

## Rate limiting

Login attempts are rate-limited per client IP, so a bad `AIR_AUTH_PASSWORD` guess
can't be brute-forced quickly. With `AIR_TRUST_PROXY=true`, the limiter uses the
forwarded client IP; otherwise it uses the direct connection IP.

## Recommended posture

- **Localhost / private LAN** — auth optional.
- **Public / shared** — `AIR_AUTH_PASSWORD` set, behind a TLS proxy with
  `AIR_TRUST_PROXY=true`, backend bound to `127.0.0.1`.
- Treat 4R as **single-user** — multi-user and OAuth are on the roadmap.
