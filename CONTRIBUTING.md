# Contributing to bc

Thank you for your interest in contributing to bc! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.25.1+
- tmux
- golangci-lint
- make

### Getting Started

```bash
# Clone the repository
git clone https://github.com/rpuneet/bc.git
cd bc

# Install dependencies
go mod download

# Build the project
make build

# Run tests
make test

# Install locally
make install
```

## Build Commands

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `bin/bc` |
| `make test` | Run tests with race detector |
| `make coverage` | Run tests with coverage report |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code |
| `make check` | Run all checks (fmt, vet, test) |
| `make clean` | Remove build artifacts |

## Code Style

### Linting

We use `golangci-lint` with strict settings. All code must pass linting before merge.

```bash
# Run linter
make lint

# Configuration is in .golangci.yml
```

### Key Lint Rules

- **errcheck**: All errors must be handled
- **gosec**: Security issues must be addressed
- **govet**: No shadowed variables
- **noctx**: Context must be propagated
- **fieldalignment**: Struct fields optimally aligned

### Code Guidelines

1. **Error Handling**: Never ignore errors. Use explicit handling or `//nolint:errcheck` with justification.

2. **Context Propagation**: Pass `context.Context` through all call chains.

3. **Testing**: Write tests for new functionality. Use table-driven tests where appropriate. See [Testing Guide](#testing-guide) below.

4. **Documentation**: Document exported functions and types.

## Project Structure

```
bc/
├── cmd/bc/              # CLI entry point
├── config/              # Generated config (cfgx)
├── internal/
│   ├── cmd/             # Cobra command implementations
│   └── tui/             # Application-specific TUI views
├── pkg/                 # Reusable packages
│   ├── agent/           # Agent lifecycle management
│   ├── workspace/       # Workspace configuration
│   ├── queue/           # Work queue
│   ├── tui/             # Generic TUI components
│   └── ...
├── prompts/             # Default role prompts
└── .ctx/                # Architecture documentation
```

## Pull Request Process

1. **Branch Naming**: Use descriptive branch names
   - `feat/description` for features
   - `fix/description` for bug fixes
   - `docs/description` for documentation

2. **Commits**: Write clear commit messages
   - Use conventional commits format
   - Reference issues where applicable

3. **Testing**: Ensure all tests pass
   ```bash
   make check
   ```

4. **PR Description**: Include
   - Summary of changes
   - Related issue numbers
   - Test plan

5. **Review**: Address all review feedback

## Architecture Documentation

Before making architectural changes, review the documentation in `.ctx/`:

- [Architecture Overview](.ctx/01-architecture-overview.md)
- [Agent Roles](.ctx/02-agent-types.md)
- [Data Models](.ctx/04-data-models.md)

## Reporting Issues

Use GitHub Issues for:
- Bug reports
- Feature requests
- Documentation improvements

Include:
- Clear description
- Steps to reproduce (for bugs)
- Expected vs actual behavior
- Environment details

## Testing Guide

### Running Tests

```bash
# Run all tests
make test

# Run tests with race detector and verbose output
go test -race -v ./...

# Run integration tests only (CLI commands)
make test-integration

# Run tests with coverage
make coverage

# Run specific test
go test -v ./internal/cmd/... -run TestCostDashboard

# Run tests for a specific package
go test -v ./pkg/cost/...
```

### Test Structure

Tests are organized by package:

- `internal/cmd/*_test.go` - CLI command tests
- `pkg/*_test.go` - Package unit tests

### Writing CLI Command Tests

For CLI commands, use the helper functions in `cmd_test.go`:

```go
func TestMyCommand(t *testing.T) {
    // Set up temporary workspace
    wsDir := setupTestWorkspace(t)

    // Execute command
    output, err := executeCmd("mycommand", "arg1", "--flag", "value")
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }

    // Verify output
    if !strings.Contains(output, "expected text") {
        t.Errorf("expected 'expected text', got: %s", output)
    }
}
```

### Test Output Capture

For command output to be captured in tests, use:
- `cmd.Printf()` instead of `fmt.Printf()`
- `cmd.Println()` instead of `fmt.Println()`
- `tabwriter.NewWriter(cmd.OutOrStdout(), ...)` instead of `tabwriter.NewWriter(os.Stdout, ...)`

### Coverage

Coverage reports are generated in `coverage.out` and uploaded to Codecov in CI.

To view coverage locally:

```bash
make coverage
go tool cover -html=coverage.out
```

## Questions?

Open an issue or discussion on GitHub.
