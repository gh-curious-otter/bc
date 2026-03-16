.PHONY: dev build build-daemon build-release build-daemon-release build-all clean gen test test-daemon coverage bench fmt vet lint check deps help version build-tui test-tui lint-tui build-agent-image build-agent-images daemon-start daemon-stop daemon-status daemon-logs daemon-run

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# ldflags for version injection
LDFLAGS_VERSION = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

help:
	@echo "Available targets:"
	@echo "  dev           - Run in development mode"
	@echo "  build         - Build the binary with version info"
	@echo "  build-daemon  - Build the bcd daemon binary"
	@echo "  build-release - Build optimized release binary with version info"
	@echo "  build-all     - Cross-compile for all platforms to dist/"
	@echo "  clean         - Remove build artifacts"
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
	@echo "Daemon targets:"
	@echo "  daemon-run    - Build and run daemon in foreground"
	@echo "  daemon-start  - Build and start daemon in background"
	@echo "  daemon-stop   - Stop the running daemon"
	@echo "  daemon-status - Show daemon status"
	@echo "  daemon-logs   - Tail daemon log file"
	@echo "  test-daemon   - Run daemon/server tests"
	@echo ""
	@echo "TUI targets (requires bun):"
	@echo "  build-tui     - Build the TUI package"
	@echo "  test-tui      - Run TUI tests"
	@echo "  lint-tui      - Lint TUI code"
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
	go build -ldflags="$(LDFLAGS_VERSION)" -o bin/bc ./cmd/bc

build-daemon: gen
	go build -ldflags="$(LDFLAGS_VERSION)" -o bin/bcd ./cmd/bcd

build-daemon-release: gen
	go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o bin/bcd ./cmd/bcd

build-release: gen
	go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o bin/bc ./cmd/bc

build-all: gen
	@mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bc-darwin-amd64 ./cmd/bc
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bc-darwin-arm64 ./cmd/bc
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bc-linux-amd64 ./cmd/bc
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bc-linux-arm64 ./cmd/bc
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bc-windows-amd64.exe ./cmd/bc
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bcd-darwin-amd64 ./cmd/bcd
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bcd-darwin-arm64 ./cmd/bcd
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bcd-linux-amd64 ./cmd/bcd
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bcd-linux-arm64 ./cmd/bcd
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bcd-windows-amd64.exe ./cmd/bcd

clean:
	rm -rf bin/ dist/

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

# Daemon targets
DAEMON_ADDR ?= 127.0.0.1:9374

daemon-run: build-daemon
	./bin/bcd --addr $(DAEMON_ADDR)

daemon-start: build build-daemon
	./bin/bc daemon start -a $(DAEMON_ADDR)

daemon-stop: build
	./bin/bc daemon stop

daemon-status: build
	./bin/bc daemon status

daemon-logs:
	@if [ -z "$(BC_WORKSPACE)" ]; then \
		echo "Set BC_WORKSPACE or run from a bc workspace directory"; \
		exit 1; \
	fi
	@tail -f $(BC_WORKSPACE)/.bc/bcd.log

test-daemon:
	go test -race ./server/ ./pkg/daemon/

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

# Docker agent images (per-provider)
AGENT_PROVIDERS := claude gemini codex aider opencode openclaw cursor

build-agent-image: build-agent-image-claude
	@echo "Default agent image built (claude)"

build-agent-image-%:
	@echo "Building bc-agent-$* image..."
	docker build -t bc-agent-$*:latest -f docker/Dockerfile.$* .

build-agent-images:
	@for p in $(AGENT_PROVIDERS); do \
		echo "Building bc-agent-$$p..."; \
		docker build -t bc-agent-$$p:latest -f docker/Dockerfile.$$p . || exit 1; \
	done
	@echo "All agent images built."
