.PHONY: dev build build-release build-all clean gen test coverage bench fmt vet lint check deps help version

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

build-release: gen
	go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o bin/bc ./cmd/bc

build-all: gen
	@mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bc-darwin-amd64 ./cmd/bc
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bc-darwin-arm64 ./cmd/bc
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bc-linux-amd64 ./cmd/bc
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bc-linux-arm64 ./cmd/bc
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w $(LDFLAGS_VERSION)" -o dist/bc-windows-amd64.exe ./cmd/bc

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
	gofmt -s -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

check: gen fmt vet lint test

deps:
	go mod download
	go mod tidy
