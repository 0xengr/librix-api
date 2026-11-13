# Librix API — developer tasks
# Run `make` or `make help` to list targets.

SHELL := /bin/bash
.DEFAULT_GOAL := help

BINARY         := api
PKG            := ./cmd/api
DIST           := bin
MIGRATIONS_DIR ?= db/migrations
SQLC_CONFIG    ?= sqlc.yml
LDFLAGS        := -s -w
PLATFORMS      ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
MIGRATION_NAME ?=

# ---------------------------------------------------------------------------
# Shared shell UI: loads .env (so recipes never need DATABASE_URL passed by
# hand), provides colors, a spinner, and small helpers.
# ---------------------------------------------------------------------------
define UI
set -a; [ -f .env ] && . ./.env; set +a; \
if [ -t 1 ]; then TTY=1; R=$$'\033[0m'; B=$$'\033[1m'; D=$$'\033[2m'; G=$$'\033[32m'; Y=$$'\033[33m'; E=$$'\033[31m'; C=$$'\033[36m'; else TTY=0; R=; B=; D=; G=; Y=; E=; C=; fi; \
ok() { printf "  $${G}✔$${R} %s\n" "$$1"; }; \
fail() { printf "  $${E}✖$${R} %s\n" "$$1"; }; \
need_db() { if [ -z "$${GOOSE_DBSTRING:-$${DATABASE_URL:-}}" ]; then fail "DATABASE_URL is empty — set it in .env (copy .env.example if missing)"; exit 1; fi; }; \
with_goose() { export GOOSE_DRIVER="$${GOOSE_DRIVER:-postgres}" GOOSE_DBSTRING="$${GOOSE_DBSTRING:-$$DATABASE_URL}"; }; \
spin() { local label="$$1"; shift; local log; log=$$(mktemp); local start=$$(date +%s%N); "$$@" >"$$log" 2>&1 & local pid=$$!; local frames=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏'); local i=0; if [ "$$TTY" -eq 1 ]; then while kill -0 "$$pid" 2>/dev/null; do printf "\r  $${C}%s$${R} %-40s" "$${frames[$$((i % 10))]}" "$$label"; i=$$((i + 1)); sleep 0.08; done; fi; wait "$$pid"; local rc=$$?; local ms=$$(( ($$(date +%s%N) - start) / 1000000 )); [ "$$TTY" -eq 1 ] && printf "\r\033[K"; if [ "$$rc" -eq 0 ]; then printf "  $${G}✔$${R} %-40s $${D}%sms$${R}\n" "$$label" "$$ms"; else printf "  $${E}✖$${R} %s\n" "$$label"; sed 's/^/    /' "$$log" >&2; fi; rm -f "$$log"; return "$$rc"; }; \
banner() { local t="LIBRIX API"; if [ "$$TTY" -eq 0 ]; then printf "  %s%s%s%s\n" "$$B" "$$C" "$$t" "$$R"; return; fi; local f k; for f in 0 1 2 3 4; do printf "\r  "; for ((k=0; k<$${#t}; k++)); do if [ "$$k" -eq "$$f" ]; then printf "%s%s" "$${B}$${C}" "$${t:$$k:1}"; else printf "%s%s" "$${D}" "$${t:$$k:1}"; fi; done; printf "%s" "$${R}"; sleep 0.05; done; printf "\r  %s%s%s%s\n" "$$B" "$$C" "$$t" "$$R"; }
endef

.PHONY: help tools run build build-local test test-v cover fmt fmt-check vet check \
	lint lint-fix tidy generate sqlc migrate-up migrate-down migrate-status \
	migrate-version migrate-redo migrate-create clean

help: ## Show available targets
	@$(UI); banner; printf "\n  %sBook borrowing API%s\n\n" "$$D" "$$R"; \
	awk -v c="$$C" -v r="$$R" 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %s%-18s%s %s\n", c, $$1, r, $$2}' $(MAKEFILE_LIST)

tools: ## Verify go, goose, sqlc and golangci-lint are installed
	@$(UI); spin "go" go version; spin "goose" goose --version; spin "sqlc" sqlc version; spin "golangci-lint" golangci-lint version

run: ## Run the API locally (go run ./cmd/api)
	@$(UI); printf "  %sRunning %s%s\n\n" "$$B" "$(PKG)" "$$R"; exec go run $(PKG)

build: ## Cross-compile every platform (PLATFORMS) with size and build time
	@$(UI); mkdir -p "$(DIST)"; printf "\n  %sCross-compiling %s%s\n\n" "$$B" "$(BINARY)" "$$R"; \
	printf "  %-18s %10s %10s\n" "PLATFORM" "SIZE" "TIME"; \
	printf "  %s----------------------------------------%s\n" "$$D" "$$R"; \
	for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; out="$(DIST)/$(BINARY)-$$os-$$arch"; \
		[ "$$os" = "windows" ] && out="$$out.exe"; \
		log=$$(mktemp); start=$$(date +%s%N); \
		( CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o "$$out" $(PKG) ) >"$$log" 2>&1 & \
		pid=$$!; frames=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏'); i=0; \
		if [ "$$TTY" -eq 1 ]; then while kill -0 "$$pid" 2>/dev/null; do \
			printf "\r  $${C}%s$${R} %-18s" "$${frames[$$((i % 10))]}" "$$os/$$arch"; i=$$((i + 1)); sleep 0.08; \
		done; fi; \
		wait "$$pid"; rc=$$?; end=$$(date +%s%N); [ "$$TTY" -eq 1 ] && printf "\r\033[K"; \
		if [ "$$rc" -ne 0 ]; then printf "  $${E}✖$${R} %-18s\n" "$$os/$$arch"; sed 's/^/    /' "$$log" >&2; rm -f "$$log"; exit 1; fi; \
		rm -f "$$log"; ms=$$(( (end - start) / 1000000 )); size=$$(du -h "$$out" | cut -f1); \
		printf "  $${G}✔$${R} %-18s %10s %8sms\n" "$$os/$$arch" "$$size" "$$ms"; \
	done; \
	printf "\n  %sArtifacts written to %s/%s\n" "$$D" "$(DIST)" "$$R"

build-local: ## Build for this machine only (bin/api)
	@$(UI); mkdir -p "$(DIST)"; spin "Building $(BINARY) for host" go build -trimpath -ldflags "$(LDFLAGS)" -o "$(DIST)/$(BINARY)" $(PKG)

test: ## Run the test suite
	@$(UI); spin "Running tests" go test ./...

test-v: ## Run the test suite verbosely
	@$(UI); go test -v ./...

cover: ## Run tests and print total coverage
	@$(UI); spin "Running tests with coverage" go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

fmt: ## Format all Go code
	@$(UI); spin "Formatting" go fmt ./...

fmt-check: ## Fail if any Go file is not gofmt-clean
	@$(UI); files=$$(gofmt -l .); if [ -n "$$files" ]; then fail "not formatted: $$files"; exit 1; else ok "formatting clean"; fi

vet: ## Run go vet
	@$(UI); spin "Vetting" go vet ./...

lint: ## Run golangci-lint (config: .golangci.yml)
	@$(UI); spin "Linting" golangci-lint run

lint-fix: ## Run golangci-lint and apply safe fixes
	@$(UI); spin "Linting (fix)" golangci-lint run --fix

check: fmt-check lint test ## Format check, lint, then tests

tidy: ## Tidy go.mod / go.sum
	@$(UI); spin "Tidying modules" go mod tidy

sqlc: ## Generate Go code from SQL with sqlc
	@$(UI); if [ ! -f "$(SQLC_CONFIG)" ] && [ ! -f sqlc.json ]; then fail "no sqlc config (expected $(SQLC_CONFIG)) — see https://docs.sqlc.dev"; exit 1; fi; spin "Generating Go from SQL" sqlc generate

generate: sqlc ## Alias for `make sqlc`

migrate-up: ## Apply all pending migrations (uses DATABASE_URL from .env)
	@$(UI); need_db; with_goose; mkdir -p "$(MIGRATIONS_DIR)"; spin "Applying migrations" goose -dir "$(MIGRATIONS_DIR)" up

migrate-down: ## Roll back the most recent migration
	@$(UI); need_db; with_goose; mkdir -p "$(MIGRATIONS_DIR)"; spin "Rolling back migration" goose -dir "$(MIGRATIONS_DIR)" down

migrate-status: ## Show migration status
	@$(UI); need_db; with_goose; mkdir -p "$(MIGRATIONS_DIR)"; goose -dir "$(MIGRATIONS_DIR)" status

migrate-version: ## Show the current migration version
	@$(UI); need_db; with_goose; mkdir -p "$(MIGRATIONS_DIR)"; goose -dir "$(MIGRATIONS_DIR)" version

migrate-redo: ## Re-run the latest migration (down then up)
	@$(UI); need_db; with_goose; mkdir -p "$(MIGRATIONS_DIR)"; spin "Re-running latest migration" goose -dir "$(MIGRATIONS_DIR)" redo

migrate-create: ## Create a migration: make migrate-create MIGRATION_NAME=add_books
	@$(UI); if [ -z "$(MIGRATION_NAME)" ]; then fail "usage: make migrate-create MIGRATION_NAME=add_books"; exit 1; fi; mkdir -p "$(MIGRATIONS_DIR)"; spin "Creating migration $(MIGRATION_NAME)" goose -dir "$(MIGRATIONS_DIR)" create "$(MIGRATION_NAME)" sql

clean: ## Remove build artifacts and coverage output
	@$(UI); rm -rf "$(DIST)" coverage.out; ok "cleaned"
