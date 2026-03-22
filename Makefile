# bc — Agent Orchestration System
# Makefile for building, testing, linting, and deploying all components.
#
# Components:
#   go   → bc (CLI binary), bcd (server daemon with embedded web UI)
#   ts   → tui (React/Ink terminal UI), web (React dashboard), landing (Next.js site)
#
# Naming convention: <verb>-<component>[-<runtime>]
#   component = bc, bcd, tui, web, landing
#   runtime   = -local (host machine), -docker (Docker image/container)
#   go, ts    = language aggregates for CI/CD convenience
#
# Usage:
#   make help                          Show all targets
#   make build                         Build everything locally
#   make test                          Run all tests
#   make lint                          Run all linters
#   make check                         Full quality gate
#   make integrate                     Full CI equivalent

# =============================================================================
# .PHONY declarations
# =============================================================================

.PHONY: help version
# Aggregates
.PHONY: build test lint fmt vet coverage bench deps check scan gen clean release run deploy
# Go language aggregates
.PHONY: build-go-local test-go lint-go fmt-go vet-go coverage-go bench-go deps-go check-go scan-go gen-go
# Go components — local
.PHONY: build-bc-local build-bcd-local release-bc-local release-bcd-local
.PHONY: run-bc-local install-bc-local deploy-bcd-local
# Go components — docker
.PHONY: build-bcd-docker build-bcdb-docker
.PHONY: build-agent-base-docker build-agent-docker build-agents-docker
# TS language aggregates
.PHONY: build-ts-local test-ts lint-ts fmt-ts vet-ts coverage-ts bench-ts deps-ts check-ts scan-ts gen-ts
# TS components
.PHONY: build-tui-local build-web-local build-landing-local
.PHONY: test-tui test-web test-landing
.PHONY: lint-tui lint-web lint-landing
.PHONY: run-tui-local run-web-local run-landing-local
.PHONY: deploy-landing-local
# Misc
.PHONY: ci-local integrate clean-artifacts clean-deps install

.DEFAULT_GOAL := help

# =============================================================================
# Variables
# =============================================================================

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_DIR ?= bin
GO ?= go
COVERAGE_THRESHOLD ?= 60

# Deploy environment: local, dogfood, production
ENV ?= local

# Docker registry and image tag
REGISTRY ?= bc
IMAGE_TAG ?= latest

# Agent providers for Docker images
AGENT_PROVIDERS := claude gemini codex aider opencode openclaw cursor

# ldflags
LDFLAGS_VERSION = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
LDFLAGS_RELEASE = -s -w $(LDFLAGS_VERSION)

# Environment-specific addresses
ADDR_local      := 127.0.0.1:9374
ADDR_dogfood    := 127.0.0.1:9374
ADDR_production := 0.0.0.0:9374

DEPLOY_ADDR = $(ADDR_$(ENV))

# =============================================================================
# Help
# =============================================================================

help: ## Show all targets
	@echo "bc — Agent Orchestration System ($(VERSION))"
	@echo ""
	@echo "Naming: make <verb>-<component>[-<runtime>]"
	@echo "  component = bc | bcd | tui | web | landing"
	@echo "  runtime   = -local (host) | -docker (container)"
	@echo "  go | ts   = language aggregates for CI/CD"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-34s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "  \033[36mbuild-agent-NAME-docker           \033[0m Build agent Docker image ($(AGENT_PROVIDERS))"
	@echo ""
	@echo "Environment (ENV=local|dogfood|production):"
	@echo "  local       127.0.0.1:9374 (default)"
	@echo "  dogfood     127.0.0.1:9374"
	@echo "  production  0.0.0.0:9374"
	@echo ""
	@echo "Examples:"
	@echo "  make build                           Build everything locally"
	@echo "  make test-go                         Run Go tests only"
	@echo "  make deploy-bcd-local ENV=dogfood    Deploy bcd to dogfood"
	@echo "  make build-agent-gemini-docker       Build Gemini agent Docker image"

version: ## Show version info
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

# =============================================================================
# Top-level aggregates
# =============================================================================

build: build-go-local build-ts-local ## Build everything locally
test: test-go test-ts ## Run all tests
lint: lint-go lint-ts ## Run all linters
fmt: fmt-go fmt-ts ## Format all code
vet: vet-go vet-ts ## Vet all code
coverage: coverage-go coverage-ts ## Run all coverage
bench: bench-go bench-ts ## Run all benchmarks
deps: deps-go deps-ts ## Install all dependencies
check: check-go check-ts ## Full quality gate (go + ts)
scan: scan-go scan-ts ## Run all security scans
gen: gen-go gen-ts ## Run all code generation
release: release-bc-local release-bcd-local ## Build release binaries (stripped, optimized)
install: install-bc-local ## Install bc to $GOPATH/bin
clean: clean-artifacts ## Remove all build artifacts

# =============================================================================
# Build — Go (local)
# =============================================================================

build-go-local: build-bc-local build-bcd-local ## Build all Go binaries locally

build-bc-local: gen-go ## Build bc CLI binary (local)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_VERSION)" -o $(BUILD_DIR)/bc ./cmd/bc

build-bcd-local: gen-go build-web-local ## Build bcd server binary (local, embeds web UI)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_VERSION)" -o $(BUILD_DIR)/bcd ./cmd/bcd

# =============================================================================
# Build — Go (docker)
# =============================================================================

build-bcd-docker: ## Build bcd server Docker image
	docker build -t $(REGISTRY)-bcd:$(IMAGE_TAG) -f docker/Dockerfile.bcd .

build-bcdb-docker: ## Build bcdb Postgres Docker image
	docker build -t $(REGISTRY)-bcdb:$(IMAGE_TAG) -f docker/Dockerfile.bcdb .

build-agent-base-docker: ## Build agent base Docker image
	docker build -t $(REGISTRY)-agent-base:$(IMAGE_TAG) -f docker/Dockerfile.base .

build-agent-docker: build-agent-base-docker build-agent-claude-docker ## Build default agent Docker image (claude)

build-agent-%-docker: build-agent-base-docker ## Build agent Docker image for provider
	docker build -t $(REGISTRY)-agent-$*:$(IMAGE_TAG) -f docker/Dockerfile.$* .

build-agents-docker: build-agent-base-docker ## Build all agent Docker images
	@for p in $(AGENT_PROVIDERS); do \
		echo "Building $(REGISTRY)-agent-$$p..."; \
		docker build -t $(REGISTRY)-agent-$$p:$(IMAGE_TAG) -f docker/Dockerfile.$$p . || exit 1; \
	done
	@echo "All agent images built."

# =============================================================================
# Build — TypeScript (local)
# =============================================================================

build-ts-local: build-tui-local build-web-local build-landing-local ## Build all TS packages locally

build-tui-local: ## Build TUI package (local)
	cd tui && bun install && bun run build

build-web-local: ## Build React web UI → server/web/dist/ (local)
	cd web && bun install && bun run build
	@rm -rf server/web/dist
	@cp -r web/dist server/web/dist

build-landing-local: ## Build Next.js landing page (local)
	cd landing && bun install && bun run build

# =============================================================================
# Test — Go
# =============================================================================

test-go: ## Run Go tests with race detector
	$(GO) test -race ./...

coverage-go: ## Run Go tests with coverage report
	$(GO) test -race -coverprofile=coverage.out ./...
	@$(GO) tool cover -func=coverage.out | tail -1
	@echo ""
	@COVERAGE=$$($(GO) tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $${COVERAGE}% is below $(COVERAGE_THRESHOLD)% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $${COVERAGE}% meets $(COVERAGE_THRESHOLD)% threshold"; \
	fi

bench-go: ## Run Go benchmarks
	$(GO) test -bench=. -benchmem -count=1 ./...

# =============================================================================
# Test — TypeScript
# =============================================================================

test-ts: test-tui test-web test-landing ## Run all TS tests

test-tui: ## Run TUI tests
	cd tui && bun install && bun test

test-web: ## Run web UI tests (vitest)
	cd web && bun install && bun run test

test-landing: ## Run landing page tests (Playwright)
	cd landing && bun run test

coverage-ts: ## Run TS coverage (NOT IMPLEMENTED)
	@echo "NOT IMPLEMENTED: coverage-ts"

bench-ts: ## Run TS benchmarks (NOT IMPLEMENTED)
	@echo "NOT IMPLEMENTED: bench-ts"

# =============================================================================
# Lint & Format — Go
# =============================================================================

lint-go: ## Run golangci-lint on Go code
	golangci-lint run ./...

fmt-go: ## Format Go code with gofmt
	find . -name '*.go' -not -path './.bc/*' -not -path './vendor/*' | xargs gofmt -s -w

vet-go: ## Run go vet
	$(GO) vet ./...

# =============================================================================
# Lint & Format — TypeScript
# =============================================================================

lint-ts: lint-tui lint-web lint-landing ## Run all TS linters

lint-tui: ## Lint TUI code
	cd tui && bun run lint

lint-web: ## Lint web UI code
	cd web && bun run lint

lint-landing: ## Lint landing page code
	cd landing && bun run lint

fmt-ts: ## Format TS code (NOT IMPLEMENTED)
	@echo "NOT IMPLEMENTED: fmt-ts"

vet-ts: ## Vet TS code (NOT IMPLEMENTED)
	@echo "NOT IMPLEMENTED: vet-ts"

# =============================================================================
# Check & CI
# =============================================================================

check-go: gen-go fmt-go vet-go lint-go test-go ## Go quality gate (gen + fmt + vet + lint + test)

check-ts: lint-ts test-ts ## TS quality gate (lint + test)

ci-local: ## Run full CI pipeline locally
	@echo "=== CI Local Pipeline ==="
	@echo "--- Step 1: gen-go ---"
	@$(MAKE) gen-go
	@echo "--- Step 2: fmt-go ---"
	@$(MAKE) fmt-go
	@echo "--- Step 3: vet-go ---"
	@$(MAKE) vet-go
	@echo "--- Step 4: lint-go ---"
	@$(MAKE) lint-go
	@echo "--- Step 5: test-go ---"
	@$(MAKE) test-go
	@echo "--- Step 6: release-bc-local ---"
	@$(MAKE) release-bc-local
	@echo "--- Step 7: Verify ---"
	@$(BUILD_DIR)/bc version
	@echo ""
	@echo "=== CI Local: ALL PASS ==="

integrate: check build ## Full integration: check + build

# =============================================================================
# Release
# =============================================================================

release-bc-local: gen-go ## Build optimized bc binary (local)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_RELEASE)" -o $(BUILD_DIR)/bc ./cmd/bc

release-bcd-local: gen-go build-web-local ## Build optimized bcd binary (local)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_RELEASE)" -o $(BUILD_DIR)/bcd ./cmd/bcd

# =============================================================================
# Run (foreground, for development)
# =============================================================================

run-bc-local: ## Run bc CLI from source (local)
	$(GO) run ./cmd/bc

run-tui-local: ## Run TUI in dev mode (NOT IMPLEMENTED)
	@echo "NOT IMPLEMENTED: run-tui-local"

run-web-local: ## Run web UI dev server with hot reload (local)
	cd web && bun run dev

run-landing-local: ## Run landing dev server with hot reload (local)
	cd landing && bun run dev

# =============================================================================
# Deploy
# =============================================================================

deploy-bcd-local: build-bcd-local ## Deploy bcd server locally (ENV=local|dogfood|production)
	@if [ -z "$(DEPLOY_ADDR)" ]; then \
		echo "❌ Unknown ENV=$(ENV). Use: local, dogfood, production"; \
		exit 1; \
	fi
	@echo "--- Deploying bcd ($(ENV)) to $(DEPLOY_ADDR) ---"
	@lsof -ti :$(lastword $(subst :, ,$(DEPLOY_ADDR))) | xargs kill 2>/dev/null || true
	@sleep 2
	@nohup ./$(BUILD_DIR)/bcd --addr $(DEPLOY_ADDR) >> .bc/bcd-$(ENV).log 2>&1 &
	@sleep 2
	@if curl -sf http://$(DEPLOY_ADDR)/health > /dev/null 2>&1; then \
		echo "✅ bcd ($(ENV)) deployed at http://$(DEPLOY_ADDR)"; \
	else \
		echo "❌ Deploy failed — check .bc/bcd-$(ENV).log"; \
		exit 1; \
	fi

deploy-landing-local: build-landing-local ## Deploy landing page locally (placeholder)
	@echo "Landing page built. Deploy via your hosting provider."

# =============================================================================
# Install
# =============================================================================

install-bc-local: build-bc-local ## Install bc to $GOPATH/bin
	cp $(BUILD_DIR)/bc $(shell $(GO) env GOPATH)/bin/

# =============================================================================
# Dependencies
# =============================================================================

deps-go: ## Download and tidy Go dependencies
	$(GO) mod download
	$(GO) mod tidy

deps-ts: ## Install all TS dependencies
	cd tui && bun install
	cd web && bun install
	cd landing && bun install

# =============================================================================
# Security scanning
# =============================================================================

scan-go: ## Run Go vulnerability scan (govulncheck)
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

scan-ts: ## Run TS security scan (NOT IMPLEMENTED)
	@echo "NOT IMPLEMENTED: scan-ts"

# =============================================================================
# Code generation
# =============================================================================

gen-go: ## Generate Go code (currently no-op)
	@true

gen-ts: ## Generate TS code (NOT IMPLEMENTED)
	@echo "NOT IMPLEMENTED: gen-ts"

# =============================================================================
# Clean
# =============================================================================

clean-artifacts: ## Remove all build artifacts
	rm -rf $(BUILD_DIR)/ dist/
	rm -rf tui/dist web/dist server/web/dist landing/.next landing/out
	rm -f coverage.out coverage.html

clean-deps: clean-artifacts ## Remove build artifacts AND node_modules
	rm -rf tui/node_modules web/node_modules landing/node_modules
