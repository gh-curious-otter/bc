# Contributing to bc

Thank you for your interest in contributing to bc! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.25.4+
- Bun (for TUI development)
- tmux
- golangci-lint
- make

### Getting Started

```bash
# Clone the repository
git clone https://github.com/rpuneet/bc.git
cd bc

# Install dependencies
make deps

# Build the project
make build

# Run tests
make test

# Install locally (copies bin/bc to $GOPATH/bin)
cp bin/bc $(go env GOPATH)/bin/
```

## Build Commands

Naming convention: `make <verb>-<lang|component>[-<runtime>]` where `lang` = `go` | `ts` (language aggregates), `component` = `bc` | `bcd` | `tui` | `web` | `landing`, `runtime` = `-local` (host) | `-docker` (container).

### Build (local)

| Command | Description |
|---------|-------------|
| `make build` | Build all components locally (bc, bcd, tui, web, landing) |
| `make build-bc-local` | Build bc CLI binary to `bin/bc` |
| `make build-bcd-local` | Build bcd server binary (embeds web UI) |
| `make build-tui-local` | Build TUI package |
| `make build-web-local` | Build React web UI → `server/web/dist/` |
| `make build-landing-local` | Build Next.js landing page |
| `make release` | Build optimized release binaries (stripped symbols) |
| `make install-bc-local` | Install bc to `$GOPATH/bin` |

### Build (Docker)

| Command | Description |
|---------|-------------|
| `make build-bcd-docker` | Build bcd server Docker image |
| `make build-bcdb-docker` | Build bcdb Postgres Docker image |
| `make build-agent-docker` | Build default agent Docker image (claude) |
| `make build-agent-NAME-docker` | Build agent Docker image for provider (claude, gemini, codex, etc.) |
| `make build-agents-docker` | Build all agent Docker images |

### Test

| Command | Description |
|---------|-------------|
| `make test` | Run all tests (go + ts) |
| `make test-go` | Run Go tests with race detector |
| `make test-ts` | Run all TS tests (tui + web + landing) |
| `make test-tui` | Run TUI tests |
| `make test-web` | Run web UI tests (vitest) |
| `make test-landing` | Run landing page tests (Playwright) |
| `make coverage-go` | Run Go tests with coverage report (60% threshold) |
| `make bench-go` | Run Go benchmarks |

### Lint & Quality

| Command | Description |
|---------|-------------|
| `make lint` | Run all linters (go + ts) |
| `make lint-go` | Run golangci-lint on Go code |
| `make lint-ts` | Run all TS linters (tui + web + landing) |
| `make lint-tui` | Lint TUI code |
| `make lint-web` | Lint web UI code |
| `make lint-landing` | Lint landing page code |
| `make fmt-go` | Format Go code with gofmt |
| `make vet-go` | Run go vet |
| `make check` | Full quality gate (go + ts) |
| `make check-go` | Go quality gate (gen + fmt + vet + lint + test) |
| `make check-ts` | TS quality gate (lint + test) |
| `make integrate` | Full CI equivalent: check + build |

### Run & Deploy

| Command | Description |
|---------|-------------|
| `make run-bc-local` | Run bc CLI from source (`go run`) |
| `make run-web-local` | Run web UI dev server (hot reload) |
| `make run-landing-local` | Run landing dev server (hot reload) |
| `make deploy-bcd-local` | Deploy bcd server locally (ENV=local\|dogfood\|production) |
| `make deploy-landing-local` | Deploy landing page locally (placeholder) |

### Utilities

| Command | Description |
|---------|-------------|
| `make gen-go` | Generate Go code from config.toml |
| `make deps-go` | Download and tidy Go dependencies |
| `make deps-ts` | Install all TS dependencies (bun install) |
| `make scan-go` | Run govulncheck for Go vulnerabilities |
| `make install-bc-local` | Install bc to `$GOPATH/bin` |
| `make clean` | Remove all build artifacts |
| `make clean-deps` | Remove build artifacts + node_modules |

Or directly with Bun (from `tui/`, `web/`, or `landing/`):

```bash
cd tui
bun install        # Install dependencies
bun run build      # Build to dist/
bun test           # Run tests
bun run lint       # Lint code
```

### TUI Testing

The TUI uses `bun:test` for testing. Key patterns:

**Testing Hooks Without DOM**

React hooks in Ink/terminal environment don't have DOM access. Test exported helper functions and type interfaces instead of hook behavior:

```typescript
// Test helper functions directly
import { getSeverityColor } from '../useLogs';
expect(getSeverityColor('error')).toBe('red');

// Validate type exports
import type { UseStatusOptions } from '../useStatus';
const options: UseStatusOptions = { pollInterval: 5000 };
expect(options.pollInterval).toBe(5000);
```

**Test File Location**

- Hooks: `tui/src/hooks/__tests__/*.test.tsx`
- Views: `tui/src/views/__tests__/*.test.tsx`
- Components: `tui/src/__tests__/components/*.test.tsx`

**Running Specific Tests**

```bash
bun test src/hooks/__tests__/useStatus.test.tsx
bun test --watch  # Watch mode
```

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

3. **Testing**: Write tests for new functionality. Use table-driven tests where appropriate.

4. **Documentation**: Document exported functions and types.

5. **Naming Conventions**:
   - Short receiver names: `w` for workspace, `a` for agent, `c` for channel
   - Avoid package-level variables except for cobra commands
   - Use descriptive but concise variable names

6. **Struct Alignment**: Run `make lint` to catch fieldalignment issues.

## Project Structure

```
bc/
├── cmd/bc/              # CLI entry point (main.go)
├── config/              # Generated config code (cfgx)
├── internal/
│   └── cmd/             # Cobra command implementations
├── pkg/                 # Reusable packages
│   ├── agent/           # Agent lifecycle, roles, tmux sessions
│   ├── channel/         # SQLite-backed communication
│   ├── cost/            # Cost tracking and budgets
│   ├── demon/           # Scheduled task management
│   ├── events/          # Event logging
│   ├── git/             # Git worktree operations
│   ├── memory/          # Agent memory system
│   ├── process/         # Background process management
│   ├── routing/         # Agent routing patterns
│   ├── team/            # Team management
│   ├── tmux/            # tmux session control
│   ├── ui/              # CLI output formatting (colors, tables)
│   └── workspace/       # Workspace config (v1 JSON, v2 TOML)
├── prompts/             # Default role prompt templates
└── tui/                 # TypeScript/React TUI (Ink)
    ├── src/
    │   ├── __tests__/   # Component and integration tests
    │   ├── components/  # Reusable UI components
    │   ├── hooks/       # React hooks (useAgents, useChannels, etc.)
    │   │   └── __tests__/ # Hook tests
    │   ├── navigation/  # Tab bar, keyboard navigation
    │   ├── services/    # BC CLI wrapper (bc.ts)
    │   ├── views/       # Full-screen views (14 views)
    │   │   └── __tests__/ # View tests
    │   └── app.tsx      # Main TUI application
    └── dist/            # Compiled output (CommonJS)
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

## Architecture Overview

Key concepts to understand before contributing:

- **Agents**: AI assistants running in isolated tmux sessions, each with its own git worktree
- **Workspace**: Project directory with `.bc/` containing config, state, and per-agent data
- **Channels**: SQLite-backed inter-agent communication with persistent history
- **Roles**: Agent capabilities defined in `.bc/roles/*.md` (engineer, manager, etc.)
- **Memory**: Per-agent persistent knowledge (experiences, learnings)

See `CLAUDE.md` for detailed architecture patterns and package documentation.

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

## Questions?

Open an issue or discussion on GitHub.
