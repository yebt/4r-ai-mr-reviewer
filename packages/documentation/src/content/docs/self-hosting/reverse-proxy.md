---
title: Reverse proxy & TLS
description: Terminate HTTPS and serve the SPA and API under one domain.
---

Put a TLS-terminating reverse proxy in front of the backend: it serves the built
SPA, forwards `/api` to the Go server, and handles certificates. Bind the backend
to localhost (`AIR_HTTP_ADDR=127.0.0.1:8080`) so only the proxy can reach it.

## Caddy

Caddy gets you automatic HTTPS with almost no config:

```text
review.example.com {
    encode gzip

    # SPA static files (packages/spa/dist)
    root * /opt/ai-reviewer/spa
    file_server

    # API → backend
    handle /api/* {
        reverse_proxy 127.0.0.1:8080
    }

    # SPA client-side routing fallback
    try_files {path} /index.html
}
```

## nginx

```nginx
server {
    listen 443 ssl http2;
    server_name review.example.com;

    ssl_certificate     /etc/letsencrypt/live/review.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/review.example.com/privkey.pem;

    root /opt/ai-reviewer/spa;      # packages/spa/dist
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host              $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    }

    location / {
        try_files $uri $uri/ /index.html;   # SPA fallback
    }
}
```

:::note[The `/api` prefix]
The SPA calls `/api/...`. Strip or keep the prefix consistently: the backend serves
paths **without** `/api` (e.g. `/reviews`), and the dev server proxies `/api →` the
backend. In production, route `/api/*` to the backend and let it see the stripped
path, matching the dev proxy.
:::

## Trusting the proxy

When a proxy terminates TLS, set **`AIR_TRUST_PROXY=true`** so the backend honors
`X-Forwarded-Proto` / `X-Forwarded-For` for the cookie `Secure` flag and login
rate-limit IP.

:::danger
Only set `AIR_TRUST_PROXY=true` when a **trusted** proxy sets those headers. If the
backend is reachable directly, a client could spoof `X-Forwarded-*` to weaken the
cookie `Secure` flag or the rate limiter. Keep it `false` otherwise.
:::
