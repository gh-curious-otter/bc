.PHONY: dev build build-bcd build-release build-bcd-release build-all clean clean-deps gen test coverage bench fmt vet lint check check-all deps install help version
.PHONY: build-tui test-tui lint-tui build-web lint-web dev-web build-landing dev-landing lint-landing test-landing
.PHONY: build-server-images build-bcd-image build-bcdb-image build-agent-base build-agent-image build-agent-images
.PHONY: security vuln deploy-dogfood

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build output directory (override with BUILD_DIR=/tmp/bc-build for Docker agents)
BUILD_DIR ?= bin

# ldflags for version injection
LDFLAGS_VERSION = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
LDFLAGS_RELEASE = -s -w $(LDFLAGS_VERSION)

# Go binary — use system default
GO ?= go

# Coverage threshold (matches CI ci.yml "Check coverage threshold" step)
COVERAGE_THRESHOLD ?= 60

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Docker targets:"
	@echo "  build-agent-image-NAME  Build specific agent image (claude, gemini, codex, aider, opencode, openclaw, cursor)"
	@echo ""
	@echo "Version: $(VERSION)  Commit: $(COMMIT)"

version: ## Show version info that will be embedded
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

# =============================================================================
# Core Go targets
# =============================================================================

dev: ## Run bc in development mode
	$(GO) run ./cmd/bc

build: gen ## Build bc binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_VERSION)" -o $(BUILD_DIR)/bc ./cmd/bc

build-bcd: gen build-web ## Build bcd server binary (embeds web UI)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_VERSION)" -o $(BUILD_DIR)/bcd ./cmd/bcd

build-release: gen ## Build optimized bc + bcd release binaries
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_RELEASE)" -o $(BUILD_DIR)/bc ./cmd/bc

build-bcd-release: gen build-web ## Build optimized bcd release binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="$(LDFLAGS_RELEASE)" -o $(BUILD_DIR)/bcd ./cmd/bcd

build-all: build build-tui build-bcd build-landing ## Build everything (bc, bcd, TUI, web, landing)

install: build ## Install bc to $GOPATH/bin
	cp $(BUILD_DIR)/bc $(shell $(GO) env GOPATH)/bin/

gen: ## Run go generate (no-op, kept for compatibility)
	@true

deps: ## Download and tidy dependencies
	$(GO) mod download
	$(GO) mod tidy

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)/ dist/
	rm -rf tui/dist web/dist server/web/dist landing/.next landing/out
	rm -f coverage.out coverage.html

clean-deps: clean ## Remove build artifacts AND node_modules
	rm -rf tui/node_modules web/node_modules landing/node_modules

# =============================================================================
# Testing & Quality
# =============================================================================

test: ## Run Go tests with race detector
	$(GO) test -race ./...

coverage: ## Run tests with coverage report
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

bench: ## Run benchmarks
	$(GO) test -bench=. -benchmem -count=1 ./...

fmt: ## Format Go code
	gofmt -s -w $$(find . -name '*.go' -not -path './.bc/*' -not -path './vendor/*')

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

check: gen fmt vet lint test ## Run all Go checks (gen + fmt + vet + lint + test)

check-all: check lint-tui test-tui lint-web ## Run all checks (Go + TUI + web)

ci-local: ## Run full CI pipeline locally (replaces GitHub Actions when billing exhausted)
	@echo "=== CI Local Pipeline ==="
	@echo "--- Step 1: Generate ---" && echo "skipped (no generators)"
	@echo "--- Step 2: Format ---" && gofmt -s -l $$(find . -name '*.go' -not -path './.bc/*' -not -path './vendor/*') | (grep . && echo "FAIL: files need formatting" && exit 1 || echo "PASS")
	@echo "--- Step 3: Vet ---" && $(GO) vet ./...
	@echo "--- Step 4: Lint ---" && golangci-lint run ./...
	@echo "--- Step 5: Test (fast) ---" && mkdir -p server/web/dist && echo '<!-- stub -->' > server/web/dist/index.html && $(GO) test -race $$($(GO) list ./... | grep -v /internal/cmd$$)
	@echo "--- Step 6: Build ---" && $(GO) build -ldflags="$(LDFLAGS_RELEASE)" -o $(BUILD_DIR)/bc ./cmd/bc
	@echo "--- Step 7: Verify ---" && $(BUILD_DIR)/bc version
	@echo ""
	@echo "=== CI Local: ALL PASS ==="

# =============================================================================
# Security scanning
# =============================================================================

vuln: ## Run govulncheck for known vulnerabilities
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

security: vuln ## Run all security checks
	@echo "Security checks passed."

# =============================================================================
# TUI targets (requires bun)
# =============================================================================

build-tui: ## Build TUI package
	cd tui && bun install && bun run build

test-tui: ## Run TUI tests
	cd tui && bun install && bun test

lint-tui: ## Lint TUI code
	cd tui && bun run lint

# =============================================================================
# Web UI targets (requires bun)
# =============================================================================

build-web: ## Build React web UI and copy to server/web/dist/
	cd web && bun install && bun run build
	@rm -rf server/web/dist
	@cp -r web/dist server/web/dist

lint-web: ## Lint web UI code
	cd web && bun run lint

dev-web: ## Run web UI dev server (hot reload)
	cd web && bun run dev

# =============================================================================
# Landing page (Next.js, requires bun)
# =============================================================================

build-landing: ## Build landing page
	cd landing && bun install && bun run build

dev-landing: ## Run landing dev server (hot reload)
	cd landing && bun run dev

lint-landing: ## Lint landing page code
	cd landing && bun run lint

test-landing: ## Run landing page Playwright tests
	cd landing && bun run test

# =============================================================================
# Docker targets
# =============================================================================

# Server images
build-bcd-image: ## Build bcd server Docker image
	docker build -t bc-bcd:latest -f docker/Dockerfile.bcd .

build-bcdb-image: ## Build bcdb Postgres Docker image
	docker build -t bc-bcdb:latest -f docker/Dockerfile.bcdb .

build-server-images: build-bcd-image build-bcdb-image ## Build all server images

# Agent images
AGENT_PROVIDERS := claude gemini codex aider opencode openclaw cursor

build-agent-base: ## Build agent base image
	docker build -t bc-agent-base:latest -f docker/Dockerfile.base .

build-agent-image: build-agent-base build-agent-image-claude ## Build default (claude) agent image

build-agent-image-%: build-agent-base
	docker build -t bc-agent-$*:latest -f docker/Dockerfile.$* .

build-agent-images: build-agent-base ## Build all agent images
	@for p in $(AGENT_PROVIDERS); do \
		echo "Building bc-agent-$$p..."; \
		docker build -t bc-agent-$$p:latest -f docker/Dockerfile.$$p . || exit 1; \
	done
	@echo "All agent images built."

# =============================================================================
# Deploy targets
# =============================================================================

deploy-dogfood: build-bcd ## Deploy dogfood: pull main, rebuild bcd, restart
	@echo "--- Deploying dogfood ---"
	@if [ -f .bc/bcd.pid ]; then \
		PID=$$(cat .bc/bcd.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			echo "Stopping bcd (PID $$PID)..."; \
			kill $$PID; \
			sleep 2; \
		fi; \
	fi
	@echo "Starting bcd..."
	@nohup ./bin/bcd --addr 127.0.0.1:9000 > /tmp/bcd-dogfood.log 2>&1 &
	@sleep 1
	@echo "Dogfood deployed at :9000 (PID $$(cat .bc/bcd.pid 2>/dev/null || echo unknown))"
