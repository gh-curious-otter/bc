# bc — Agent Orchestration System
# Makefile for building, testing, linting, and deploying all components.
#
# Components:
#   bc       Go CLI binary (cmd/bc)
#   bcd      Go server daemon with embedded web UI (cmd/bcd)
#   tui      React/Ink terminal UI (tui/)
#   web      React web dashboard (web/) — embedded into bcd
#   landing  Next.js marketing site (landing/)
#
# Usage:
#   make help              Show all targets
#   make build             Build bc + bcd + tui + web + landing
#   make test              Run all tests
#   make lint              Run all linters
#   make check             Full quality gate (fmt + vet + lint + test)
#   make deploy ENV=local  Deploy bcd to environment
#   make integrate         check + build (CI equivalent)

# =============================================================================
# .PHONY declarations (grouped by category)
# =============================================================================

.PHONY: help version
.PHONY: build build-bc build-bcd build-tui build-web build-landing
.PHONY: test test-bcd test-bc test-tui test-web test-landing test-ui
.PHONY: lint lint-bcd lint-bc lint-tui lint-web lint-landing lint-ui
.PHONY: fmt vet check coverage bench
.PHONY: deploy deploy-bcd deploy-landing
.PHONY: release release-bcd release-bc
.PHONY: gen deps clean clean-deps
.PHONY: ci-local integrate
.PHONY: security vuln
.PHONY: docker docker-bcd docker-bcdb docker-agent-base docker-agent docker-agents
.PHONY: dev dev-bcd dev-web dev-landing
.PHONY: install

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
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "  \033[36mdocker-agent-NAME       \033[0m Build agent image for provider ($(AGENT_PROVIDERS))"
	@echo ""
	@echo "Environment (ENV):"
	@echo "  local       127.0.0.1:9374 (default)"
	@echo "  dogfood     127.0.0.1:9374"
	@echo "  production  0.0.0.0:9374"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    Build everything"
	@echo "  make test-bcd                 Run Go tests only"
	@echo "  make deploy ENV=dogfood       Deploy bcd to dogfood"
	@echo "  make docker-agent-gemini      Build Gemini agent image"

version: ## Show version info
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

# =============================================================================
# Build
# =============================================================================

build: build-bc build-bcd build-tui build-landing ## Build everything

build-bc: gen ## Build bc CLI binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_VERSION)" -o $(BUILD_DIR)/bc ./cmd/bc

build-bcd: gen build-web ## Build bcd server binary (embeds web UI)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_VERSION)" -o $(BUILD_DIR)/bcd ./cmd/bcd

build-tui: ## Build TUI package
	cd tui && bun install && bun run build

build-web: ## Build React web UI → server/web/dist/
	cd web && bun install && bun run build
	@rm -rf server/web/dist
	@cp -r web/dist server/web/dist

build-landing: ## Build Next.js landing page
	cd landing && bun install && bun run build

install: build-bc ## Install bc to $GOPATH/bin
	cp $(BUILD_DIR)/bc $(shell $(GO) env GOPATH)/bin/

# =============================================================================
# Test
# =============================================================================

test: test-bcd test-ui ## Run all tests

test-bcd: ## Run Go tests with race detector
	$(GO) test -race ./...

test-bc: test-bcd ## Alias for test-bcd (shared Go codebase)

test-tui: ## Run TUI tests
	cd tui && bun install && bun test

test-web: ## Run web UI tests (vitest)
	cd web && bun install && bun run test

test-landing: ## Run landing page tests (Playwright)
	cd landing && bun run test

test-ui: test-tui test-web test-landing ## Run all UI tests

coverage: ## Run Go tests with coverage report
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

bench: ## Run Go benchmarks
	$(GO) test -bench=. -benchmem -count=1 ./...

# =============================================================================
# Lint & Format
# =============================================================================

fmt: ## Format Go code
	gofmt -s -w $$(find . -name '*.go' -not -path './.bc/*' -not -path './vendor/*')

vet: ## Run go vet
	$(GO) vet ./...

lint: lint-bcd lint-ui ## Run all linters

lint-bcd: ## Run golangci-lint on Go code
	golangci-lint run ./...

lint-bc: lint-bcd ## Alias for lint-bcd (shared Go codebase)

lint-tui: ## Lint TUI code
	cd tui && bun run lint

lint-web: ## Lint web UI code
	cd web && bun run lint

lint-landing: ## Lint landing page code
	cd landing && bun run lint

lint-ui: lint-tui lint-web lint-landing ## Run all UI linters

# =============================================================================
# Check & CI
# =============================================================================

check: gen fmt vet lint-bcd test-bcd ## Go quality gate (gen + fmt + vet + lint + test)

ci-local: ## Run full CI pipeline locally
	@echo "=== CI Local Pipeline ==="
	@echo "--- Step 1: Generate ---" && $(GO) generate ./...
	@echo "--- Step 2: Format ---" && gofmt -s -l $$(find . -name '*.go' -not -path './.bc/*' -not -path './vendor/*') | (grep . && echo "FAIL: files need formatting" && exit 1 || echo "PASS")
	@echo "--- Step 3: Vet ---" && $(GO) vet ./...
	@echo "--- Step 4: Lint ---" && golangci-lint run ./...
	@echo "--- Step 5: Test (fast) ---" && mkdir -p server/web/dist && echo '<!-- stub -->' > server/web/dist/index.html && $(GO) test -race $$($(GO) list ./... | grep -v /internal/cmd$$)
	@echo "--- Step 6: Build ---" && $(GO) build -ldflags="$(LDFLAGS_RELEASE)" -o $(BUILD_DIR)/bc ./cmd/bc
	@echo "--- Step 7: Verify ---" && $(BUILD_DIR)/bc version
	@echo ""
	@echo "=== CI Local: ALL PASS ==="

integrate: check lint-ui test-ui build ## Full integration: check + lint + test + build

# =============================================================================
# Release
# =============================================================================

release: release-bc release-bcd ## Build release binaries (stripped, optimized)

release-bc: gen ## Build optimized bc binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_RELEASE)" -o $(BUILD_DIR)/bc ./cmd/bc

release-bcd: gen build-web ## Build optimized bcd binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_RELEASE)" -o $(BUILD_DIR)/bcd ./cmd/bcd

# =============================================================================
# Deploy
# =============================================================================

deploy: deploy-bcd ## Deploy bcd (ENV=local|dogfood|production)

deploy-bcd: build-bcd ## Deploy bcd server to $(ENV) at $(DEPLOY_ADDR)
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

deploy-landing: build-landing ## Deploy landing page (placeholder)
	@echo "Landing page built. Deploy via your hosting provider."

# =============================================================================
# Dev servers
# =============================================================================

dev: dev-bcd ## Run bcd in development mode

dev-bcd: ## Run bc CLI in dev mode
	$(GO) run ./cmd/bc

dev-web: ## Run web UI dev server (hot reload)
	cd web && bun run dev

dev-landing: ## Run landing dev server (hot reload)
	cd landing && bun run dev

# =============================================================================
# Docker — Images
# =============================================================================

docker: docker-bcd docker-bcdb ## Build all server Docker images

docker-bcd: ## Build bcd server Docker image
	docker build -t $(REGISTRY)-bcd:$(IMAGE_TAG) -f docker/Dockerfile.bcd .

docker-bcdb: ## Build bcdb Postgres Docker image
	docker build -t $(REGISTRY)-bcdb:$(IMAGE_TAG) -f docker/Dockerfile.bcdb .

# --- Agent images ---

docker-agent-base: ## Build agent base image
	docker build -t $(REGISTRY)-agent-base:$(IMAGE_TAG) -f docker/Dockerfile.base .

docker-agent: docker-agent-base docker-agent-claude ## Build default agent image (claude)

docker-agent-%: docker-agent-base ## Build agent image for provider (e.g., docker-agent-gemini)
	docker build -t $(REGISTRY)-agent-$*:$(IMAGE_TAG) -f docker/Dockerfile.$* .

docker-agents: docker-agent-base ## Build all agent images
	@for p in $(AGENT_PROVIDERS); do \
		echo "Building $(REGISTRY)-agent-$$p..."; \
		docker build -t $(REGISTRY)-agent-$$p:$(IMAGE_TAG) -f docker/Dockerfile.$$p . || exit 1; \
	done
	@echo "All agent images built."

# =============================================================================
# Security
# =============================================================================

vuln: ## Run govulncheck for known vulnerabilities
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

security: vuln ## Run all security checks
	@echo "Security checks passed."

# =============================================================================
# Utilities
# =============================================================================

gen: ## Generate code (currently no-op)
	@true

deps: ## Download and tidy Go dependencies
	$(GO) mod download
	$(GO) mod tidy

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)/ dist/
	rm -rf tui/dist web/dist server/web/dist landing/.next landing/out
	rm -f coverage.out coverage.html

clean-deps: clean ## Remove build artifacts AND node_modules
	rm -rf tui/node_modules web/node_modules landing/node_modules
