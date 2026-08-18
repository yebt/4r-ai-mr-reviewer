---
title: Deploy
description: Ship the single Go binary to your own server and run it as a service.
---

The backend compiles to a **single self-contained binary** with an embedded SQLite
database — no runtime dependencies, no external services. Deployment is: build,
copy, run.

## 1. Build the binary

On a machine with Go 1.26+:

```sh
make build          # → ./bin/ai-reviewer
```

Or cross-compile for your server's OS/arch, e.g. Linux amd64:

```sh
cd packages/server
GOOS=linux GOARCH=amd64 go build -o ai-reviewer ./cmd/server
```

Copy the resulting binary to the server (e.g. `/opt/ai-reviewer/ai-reviewer`).

## 2. Serve the web UI

The Go server exposes the **API**. Build the SPA and serve its static files with
your web server / proxy (or a static host), pointing its `/api` calls at the
backend. See [Reverse proxy & TLS](/self-hosting/reverse-proxy/) for a combined
setup that serves the SPA and proxies the API under one domain.

```sh
cd packages/spa
bun run build        # → dist/
```

## 3. Configure

Set the environment. A minimal production setup:

```sh
# /opt/ai-reviewer/.env
AIR_HTTP_ADDR=127.0.0.1:8080        # bind local; a proxy fronts it
AIR_DB_PATH=/var/lib/ai-reviewer/ai-reviewer.db
AIR_PASSWORD=…                      # unlocks the secret vault
AIR_TRUST_PROXY=true                # ONLY behind a trusted TLS proxy
AIR_AUTH_PASSWORD=…                 # see the Authentication guide first
```

See [Configuration](/reference/configuration/) for every variable.

## 4. Run as a service (systemd)

```ini
# /etc/systemd/system/ai-reviewer.service
[Unit]
Description=4R AI MR Reviewer
After=network.target

[Service]
WorkingDirectory=/opt/ai-reviewer
EnvironmentFile=/opt/ai-reviewer/.env
ExecStart=/opt/ai-reviewer/ai-reviewer
Restart=on-failure
User=ai-reviewer
StateDirectory=ai-reviewer

[Install]
WantedBy=multi-user.target
```

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now ai-reviewer
sudo systemctl status ai-reviewer
```

## 5. Next

- [Reverse proxy & TLS](/self-hosting/reverse-proxy/) — terminate HTTPS and serve
  the SPA + API under one domain.
- [Authentication](/self-hosting/authentication/) — before exposing it publicly.
- [Backups](/self-hosting/backups/) — the database **and** the master key.
