# bc Makefile
# Build, test, and development commands

# Binary name
BINARY_NAME := bc
BINARY_PATH := bin/$(BINARY_NAME)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOVET := $(GOCMD) vet

# Build info (injected via ldflags)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Platforms for cross-compilation
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

# Default target
.DEFAULT_GOAL := help

##@ Development

.PHONY: build
build: ## Build the binary
	@mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/bc

.PHONY: run
run: build ## Build and run the binary
	./$(BINARY_PATH)

.PHONY: install
install: ## Install to GOPATH/bin
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) ./cmd/bc

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin/
	rm -rf dist/
	rm -f coverage.out coverage.html

##@ Testing

.PHONY: test
test: ## Run tests
	$(GOTEST) -v -race ./...

.PHONY: test-short
test-short: ## Run tests (short mode)
	$(GOTEST) -v -short ./...

.PHONY: coverage
coverage: ## Run tests with coverage
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: bench
bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

##@ Code Quality

.PHONY: fmt
fmt: ## Format code
	$(GOFMT) -s -w .

.PHONY: fmt-check
fmt-check: ## Check code formatting
	@test -z "$$($(GOFMT) -l .)" || (echo "Code not formatted. Run 'make fmt'" && exit 1)

.PHONY: vet
vet: ## Run go vet
	$(GOVET) ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint installed)
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: brew install golangci-lint" && exit 1)
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: brew install golangci-lint" && exit 1)
	golangci-lint run --fix ./...

.PHONY: check
check: fmt-check vet test ## Run all checks (format, vet, test)

##@ Dependencies

.PHONY: deps
deps: ## Download dependencies
	$(GOMOD) download

.PHONY: tidy
tidy: ## Tidy and verify dependencies
	$(GOMOD) tidy
	$(GOMOD) verify

.PHONY: update
update: ## Update dependencies
	$(GOGET) -u ./...
	$(GOMOD) tidy

##@ Build Variants

.PHONY: build-debug
build-debug: ## Build with debug symbols
	@mkdir -p bin
	$(GOBUILD) -gcflags="all=-N -l" -o $(BINARY_PATH) ./cmd/bc

.PHONY: build-release
build-release: ## Build optimized release binary
	@mkdir -p bin
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -o $(BINARY_PATH) ./cmd/bc

.PHONY: build-all
build-all: ## Build for all platforms
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath \
		-o dist/$(BINARY_NAME)-$${platform%/*}-$${platform#*/} ./cmd/bc; \
		echo "Built: dist/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}"; \
	done

##@ Release

.PHONY: version
version: ## Show version info
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

##@ Help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
