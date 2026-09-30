# ai-reviewer — developer tasks
.DEFAULT_GOAL := help

SERVER  := packages/server
WEB     := packages/spa-b
WEB_OLD := packages/spa
LANDING := packages/landing-base
LANDING_V2 := packages/landing-base-v2
DOCS    := packages/documentation

# The SPA dev server proxies /api to the backend. Derive its target from the
# same AIR_HTTP_ADDR the server listens on (default 127.0.0.1:8080), so
# `make dev` works with or without AIR_HTTP_ADDR set (":8082" → localhost:8082).
API_ADDR   := $(or $(AIR_HTTP_ADDR),127.0.0.1:8080)
API_TARGET := http://$(patsubst :%,localhost:%,$(API_ADDR))
BIN    := $(CURDIR)/bin
BINARY := $(BIN)/air-server

.PHONY: help run run-server run-spa run-spa-host run-spa-old run-landing build-landing run-landing-v2 build-landing-v2 run-docs dev build test vet fmt tidy clean

help: ## Show this help
	@echo "ai-reviewer — make targets:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Config via env: AIR_HTTP_ADDR (default 127.0.0.1:8080), AIR_DB_PATH,"
	@echo "                AIR_PASSWORD (empty = key-file mode), AIR_SKILLS_DIR"
	@echo ""
	@echo "Example: make dev   # backend + SPA (packages/spa-b) together"

run: run-server ## Alias for run-server

run-server: ## Start the API server (reads AIR_* env vars)
	cd $(SERVER) && go run ./cmd/server

run-spa: ## Start the SPA dev server (packages/spa-b)
	cd $(WEB) && VITE_API_TARGET=$(API_TARGET) bun run dev

run-spa-host: ## Start the SPA dev server exposed on the LAN
	cd $(WEB) && VITE_API_TARGET=$(API_TARGET) bun run dev -- --host

run-spa-old: ## Start the legacy SPA (packages/spa) until it is retired
	cd $(WEB_OLD) && VITE_API_TARGET=$(API_TARGET) bun run dev

run-landing: ## Start the landing site dev server (Astro)
	cd $(LANDING) && bun run dev

build-landing: ## Build the landing site (set SITE_URL / PUBLIC_DOCS_URL)
	cd $(LANDING) && bun run build

run-landing-v2: ## Start the v2 landing site dev server (Astro)
	cd $(LANDING_V2) && bun run dev

build-landing-v2: ## Build the v2 landing site (set SITE_URL / PUBLIC_DOCS_URL)
	cd $(LANDING_V2) && bun run build

run-docs: ## Start the documentation site dev server (Astro)
	cd $(DOCS) && bun run dev


dev: ## Run backend + SPA together (Ctrl-C stops both)
	@echo "backend → $(API_ADDR)   SPA (spa-b) → :5173 (proxy → $(API_TARGET))"
	@trap 'kill 0' EXIT; \
		( cd $(SERVER) && go run ./cmd/server ) & \
		( cd $(WEB) && VITE_API_TARGET=$(API_TARGET) bun run dev ) & \
		wait

build: ## Compile the server binary into ./bin/air-server
	@mkdir -p $(BIN)
	cd $(SERVER) && go build -o $(BINARY) ./cmd/server
	@echo "built $(BINARY)"

test: ## Run the full server test suite
	cd $(SERVER) && go test ./...

vet: ## Run go vet
	cd $(SERVER) && go vet ./...

fmt: ## Format the code
	cd $(SERVER) && go fmt ./...

tidy: ## Tidy go.mod / go.sum
	cd $(SERVER) && go mod tidy

clean: ## Remove build artifacts and local db files
	rm -rf $(BIN)
	rm -f $(SERVER)/ai-reviewer.db $(SERVER)/ai-reviewer.db.key $(SERVER)/ai-reviewer.db-*
