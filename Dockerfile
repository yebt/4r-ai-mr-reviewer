# syntax=docker/dockerfile:1
#
# Multi-target build for a Dokploy (Docker Compose) deploy — see
# docs/DEPLOY-DOKPLOY.md. Two images come out of this file:
#   - target `api`: the Go API server (single static binary, no CGO)
#   - target `web`: nginx serving the built SPA and reverse-proxying the API
# docker-compose.yml wires them together on one domain (same-origin, which the
# cookie-based auth requires).

# ---------------------------------------------------------------------------
# Build the Go API server. modernc.org/sqlite is pure Go, so CGO stays off and
# the result is a fully static binary that runs on a minimal base image.
# ---------------------------------------------------------------------------
FROM golang:1.26-alpine AS api-build
WORKDIR /src
COPY packages/server/go.mod packages/server/go.sum ./
RUN go mod download
COPY packages/server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/air-server ./cmd/server

# --- API runtime image -----------------------------------------------------
FROM alpine:3.20 AS api
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 air \
 && mkdir -p /data && chown air:air /data
COPY --from=api-build /out/air-server /usr/local/bin/air-server
USER air
# The API binds all interfaces inside the container; only the web image (same
# Docker network) reaches it. The DB and the vault key-file live under /data,
# which MUST be a persistent volume (losing the key-file = losing every stored
# GitLab token and provider API key).
ENV AIR_HTTP_ADDR=":8080" \
    AIR_DB_PATH="/data/ai-reviewer.db" \
    AIR_KEYFILE_PATH="/data/ai-reviewer.db.key"
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/usr/local/bin/air-server"]

# ---------------------------------------------------------------------------
# Build the Vue SPA (Vite → dist). build-only skips the type-check that CI runs.
# ---------------------------------------------------------------------------
FROM oven/bun:1-alpine AS web-build
WORKDIR /src
COPY packages/spa/package.json packages/spa/bun.lock ./
RUN bun install --frozen-lockfile
COPY packages/spa/ ./
RUN bun run build-only

# --- Web runtime image: nginx serves the SPA and proxies the API -----------
FROM nginx:1.27-alpine AS web
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=web-build /src/dist /usr/share/nginx/html
EXPOSE 80
