# bc

A simpler, more controllable agent orchestrator.

## Vision

Coordinate multiple Claude Code agents with predictable behavior and cost awareness.

## Goals

- Coordinate multiple Claude Code agents
- Persistent work tracking with git
- Simple TUI for visualization
- Cost-aware operation
- Predictable behavior

## Status

Active development.

## Key Differences from Gas Town (Planned)

- Simpler agent hierarchy (2 types: Coordinator + Worker)
- Built-in cost controls with hard limits
- More predictable workflows with explicit action allowlists

## Installation

```bash
# Build from source
make build

# Install to GOPATH/bin
make install

# Or download from releases (when available)
```

## Usage

```bash
# Show help
bc --help

# Show version
bc version
```

## Development

### Prerequisites

- Go 1.23+
- Make
- golangci-lint (optional, for linting)

### Quick Start

```bash
# Download dependencies
make deps

# Build
make build

# Run tests
make test

# Run linter
make lint

# See all available commands
make help
```

### Available Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build the binary to `bin/bc` |
| `make run` | Build and run |
| `make test` | Run tests with race detector |
| `make coverage` | Run tests with coverage report |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code |
| `make check` | Run all checks (fmt, vet, test) |
| `make clean` | Remove build artifacts |
| `make build-release` | Build optimized release binary |
| `make build-all` | Cross-compile for all platforms |

### Project Structure

```
bc/
├── cmd/bc/              # CLI entry point
│   └── main.go
├── internal/            # Private application code
│   └── cmd/             # Cobra command definitions
│       ├── root.go
│       └── root_test.go
├── .ctx/                # Design documentation
├── .github/workflows/   # CI/CD
├── .gitignore
├── .golangci.yml        # Linter configuration
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Documentation

See the `.ctx/` directory for detailed architecture documentation and design decisions based on lessons learned from Gas Town.

## License

TBD
