.PHONY: dev build build-release build-all clean gen test coverage bench fmt vet lint check deps help version build-tui test-tui lint-tui build-web build-bcd build-bcd-image build-bcdb-image build-agent-base build-agent-image build-agent-images build-landing dev-landing

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build output directory (override with BUILD_DIR=/tmp/bc-build for Docker agents)
BUILD_DIR ?= bin

# ldflags for version injection
LDFLAGS_VERSION = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

help:
	@echo "Available targets:"
	@echo "  dev           - Run in development mode"
	@echo "  build         - Build the binary with version info"
	@echo "  build-release - Build optimized release binary with version info"
	@echo "  build-all     - Build everything (bc, bcd, TUI, web UI)"
	@echo "  clean         - Remove all build artifacts (bin, dist, node_modules, coverage)"
	@echo "  gen           - Generate config code from config.toml"
	@echo "  test          - Run tests"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  bench         - Run benchmarks"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run golangci-lint"
	@echo "  check         - Run all checks (gen + fmt + vet + lint + test)"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  version       - Show version info that will be embedded"
	@echo ""
	@echo "TUI targets (requires bun):"
	@echo "  build-tui     - Build the TUI package"
	@echo "  test-tui      - Run TUI tests"
	@echo "  lint-tui      - Lint TUI code"
	@echo ""
	@echo "Server targets:"
	@echo "  build-web     - Build React web UI (cd web && bun run build)"
	@echo "  build-bcd     - Build bcd server binary (embeds web UI)"
	@echo ""
	@echo "Landing page targets:"
	@echo "  build-landing - Build static landing page to landing/dist/"
	@echo "  dev-landing   - Run local dev server at http://localhost:8080"
	@echo ""
	@echo "Docker agent targets:"
	@echo "  build-agent-image       - Build default (claude) agent image"
	@echo "  build-agent-image-NAME  - Build specific agent image (claude, gemini, codex, aider, opencode, openclaw, cursor)"
	@echo "  build-agent-images      - Build all agent images"
	@echo ""
	@echo "Version variables (can be overridden):"
	@echo "  VERSION=$(VERSION)"
	@echo "  COMMIT=$(COMMIT)"
	@echo "  DATE=$(DATE)"

version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

dev:
	go run ./cmd/bc

build: gen
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS_VERSION)" -o $(BUILD_DIR)/bc ./cmd/bc

build-web:
	@echo "Building React web UI..."
	cd web && bun install && bun run build
	@rm -rf server/web/dist
	@cp -r web/dist server/web/dist
	@echo "Web UI copied to server/web/dist/ for embedding"

build-bcd: gen build-web
	@echo "Building bcd server (with embedded web UI)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS_VERSION)" -o $(BUILD_DIR)/bcd ./cmd/bcd

build-release: gen
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o $(BUILD_DIR)/bc ./cmd/bc

build-all: build build-tui build-bcd
	@echo "All binaries built: bc, bcd (with web UI and TUI)"

clean:
	rm -rf $(BUILD_DIR)/ dist/
	rm -rf tui/node_modules tui/dist
	rm -rf web/node_modules web/dist server/web/dist
	rm -rf landing/dist
	rm -f coverage.out

gen:
	go generate ./...

test:
	go test -race ./...

coverage: gen
	go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

bench:
	go test -bench=. -benchmem ./...

fmt:
	gofmt -s -w $(shell find . -name '*.go' -not -path './.bc/*')

vet:
	go vet ./...

lint:
	golangci-lint run ./...

check: gen fmt vet lint test

deps:
	go mod download
	go mod tidy

# Server images (bcd + bcdb)
build-bcd-image:
	@echo "Building bc-bcd:latest..."
	docker build -t bc-bcd:latest -f docker/Dockerfile.bcd .

build-bcdb-image:
	@echo "Building bc-bcdb:latest..."
	docker build -t bc-bcdb:latest -f docker/Dockerfile.bcdb .

build-server-images: build-bcd-image build-bcdb-image
	@echo "Server images built (bc-bcd, bc-bcdb)"

# TUI targets (requires bun)
build-tui:
	@echo "Building TUI..."
	cd tui && bun install && bun run build

test-tui:
	@echo "Testing TUI..."
	cd tui && bun test

lint-tui:
	@echo "Linting TUI..."
	cd tui && bun run lint

# Landing page
build-landing:
	@echo "Building landing page..."
	@mkdir -p landing/dist
	@cp landing/index.html landing/dist/
	@cp -r landing/assets landing/dist/
	@echo "Landing page built to landing/dist/"

dev-landing:
	@echo "Starting landing page dev server at http://localhost:8080"
	@cd landing && python3 -m http.server 8080

# Docker agent images (per-provider)
AGENT_PROVIDERS := claude gemini codex aider opencode openclaw cursor

build-agent-base:
	@echo "Building bc-agent-base image..."
	docker build -t bc-agent-base:latest -f docker/Dockerfile.base .

build-agent-image: build-agent-base build-agent-image-claude
	@echo "Default agent image built (claude)"

build-agent-image-%: build-agent-base
	@echo "Building bc-agent-$* image..."
	docker build -t bc-agent-$*:latest -f docker/Dockerfile.$* .

build-agent-images: build-agent-base
	@for p in $(AGENT_PROVIDERS); do \
		echo "Building bc-agent-$$p..."; \
		docker build -t bc-agent-$$p:latest -f docker/Dockerfile.$$p . || exit 1; \
	done
	@echo "All agent images built."
