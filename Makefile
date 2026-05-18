.PHONY: help build build-backend build-ui test lint run run-dev clean docker-build docker-push

# Sensible defaults
IMAGE ?= llm-orchestrator
TAG   ?= latest
PORT  ?= 8080

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ─── Build ────────────────────────────────────────────────────────────────────

build: build-backend build-ui ## Build everything (backend + UI)

build-backend: ## Compile the Go server binary
	CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server

build-ui: ## Build the React UI (output: ui/dist)
	cd ui && npm ci && npm run build

# ─── Test ─────────────────────────────────────────────────────────────────────

test: ## Run all Go tests
	go test ./... -race -count=1

test-cover: ## Run tests and open coverage report
	go test ./... -race -coverprofile=coverage.out && go tool cover -html=coverage.out

lint: ## Run golangci-lint (install: https://golangci-lint.run/usage/install/)
	golangci-lint run ./...

# ─── Run ──────────────────────────────────────────────────────────────────────

run: build-backend ## Build and start the production server
	SERVER_PORT=:$(PORT) ./server

run-dev: ## Start backend (hot-reload with Air) + UI dev server in parallel
	@echo "Starting backend with Air and UI dev server..."
	@air -c .air.toml & cd ui && npm run dev

# ─── Docker ───────────────────────────────────────────────────────────────────

docker-build: ## Build the Docker image
	docker build -t $(IMAGE):$(TAG) .

docker-push: ## Push to a registry (set IMAGE and TAG)
	docker push $(IMAGE):$(TAG)

docker-run: ## Run the Docker image locally (requires .env)
	docker run --rm -p $(PORT):8080 --env-file .env $(IMAGE):$(TAG)

# ─── Misc ─────────────────────────────────────────────────────────────────────

clean: ## Remove compiled binaries and UI build output
	rm -f server orch
	rm -rf ui/dist ui/dist-ssr

migrate-up: ## Apply all pending database migrations
	migrate -path=migrations -database "$(DATABASE_URL)" up

migrate-down: ## Roll back the last database migration
	migrate -path=migrations -database "$(DATABASE_URL)" down 1
