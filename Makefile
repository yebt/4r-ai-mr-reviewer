# ai-reviewer — developer tasks
.DEFAULT_GOAL := help

SERVER := packages/server
WEB    := packages/spa
BIN    := $(CURDIR)/bin
BINARY := $(BIN)/air-server

.PHONY: help run run-server run-spa dev build test vet fmt tidy clean

help: ## Show this help
	@echo "ai-reviewer — make targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Config via env: AIR_HTTP_ADDR (default :8080), AIR_DB_PATH,"
	@echo "                AIR_PASSWORD (empty = key-file mode), AIR_SKILLS_DIR"
	@echo ""
	@echo "Example: make dev   # backend + SPA together"

run: run-server ## Alias for run-server

run-server: ## Start the API server (reads AIR_* env vars)
	cd $(SERVER) && go run ./cmd/server

run-spa: ## Start the SPA dev server (Vite)
	cd $(WEB) && bun run dev

dev: ## Run backend + SPA together (Ctrl-C stops both)
	@echo "backend → :8080   SPA → :5173"
	@trap 'kill 0' EXIT; \
		( cd $(SERVER) && go run ./cmd/server ) & \
		( cd $(WEB) && bun run dev ) & \
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
