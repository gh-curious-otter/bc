.PHONY: dev build clean gen test bench fmt vet lint check deps help

help:
	@echo "Available targets:"
	@echo "  dev        - Run in development mode"
	@echo "  build      - Build the binary"
	@echo "  clean      - Remove build artifacts"
	@echo "  gen        - Generate config code from config.toml"
	@echo "  test       - Run tests"
	@echo "  bench      - Run benchmarks"
	@echo "  fmt        - Format code"
	@echo "  vet        - Run go vet"
	@echo "  lint       - Run golangci-lint"
	@echo "  check      - Run all checks (gen + fmt + vet + lint + test)"
	@echo "  deps       - Download and tidy dependencies"

dev:
	go run ./cmd/bc

build: gen
	go build -o bin/bc ./cmd/bc

clean:
	rm -rf bin/

gen:
	go generate ./...

test:
	go test -race ./...

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
