---
title: CLI & make targets
description: Make targets for the backend and the SPA's own scripts.
---

## Make targets

Run from the repository root:

```
make            # help
make dev        # backend + SPA together
make run-server # backend only
make run-spa    # SPA only
make build      # compile the server binary to ./bin
make test       # server test suite
make vet        # go vet
make fmt        # go fmt
make clean      # remove build artifacts and local db files
```

## SPA scripts

The web client has its own scripts. Run them from `packages/spa` (with `bun run`
or `npm run`):

| Script | What it does |
|---|---|
| `dev` | Vite dev server (proxies `/api` to the backend). |
| `build` | Production build. |
| `type-check` | `vue-tsc` type check. |
| `test:unit` | Unit tests. |
| `lint` | Lint (oxlint + eslint). |

## The server binary

`make build` produces a single self-contained binary at `./bin/ai-reviewer`. It
embeds SQLite (pure-Go), so there are no runtime dependencies to install — copy the
binary to your server and run it. See [Deploy](/self-hosting/deploy/).
