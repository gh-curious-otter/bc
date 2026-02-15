# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Start

**Build**
```bash
make build              # Build binary to bin/bc with version info
make build-release      # Optimized release build
```

**Test**
```bash
make test               # Run all tests with race detector
make coverage           # Generate coverage report
make test -k TestName   # Run specific test (use pattern matching)
```

**Development**
```bash
make dev                # Run CLI in development mode (go run)
make gen                # Generate config from config.toml
make fmt                # Format code with gofmt
make lint               # Run golangci-lint (strict)
make check              # Run full check suite (gen + fmt + lint + test)
```

**TUI (TypeScript/React with Ink)**
```bash
cd tui && bun install && bun run build   # Build TUI package
cd tui && bun test                       # Run TUI tests
cd tui && bun run lint                   # Lint TUI code
```

## Project Structure

### Core Architecture

**bc** is an AI agent orchestration CLI built in Go with a TypeScript/React TUI. Key components:

- **cmd/bc/main.go**: Entry point that delegates to internal/cmd
- **internal/cmd/**: All Cobra CLI command implementations (single package, many files)
  - Commands are organized as `*Cmd` variables (agent.go, channel.go, etc.)
  - Each command file contains subcommand handlers
- **pkg/**: Reusable packages imported by commands
  - **agent/**: Agent lifecycle, roles, capabilities, tmux session management
  - **workspace/**: Workspace/project initialization, config (v1 JSON and v2 TOML), roles
  - **channel/**: Agent-to-agent communication channels, SQLite storage
  - **memory/**: Agent memory system, learnings, experiences
  - **cost/**: Cost tracking, budgets, spending analytics
  - **events/**: Event logging and stuck-agent detection
  - **process/**: Background process management
  - **demon/**: Scheduled task management
  - **tmux/**: tmux session control for agent isolation
  - **git/**: Git worktree operations for per-agent isolation
  - **team/**: Team and grouping management
  - **routing/**: Agent routing and pattern matching
  - **stats/**: Workspace statistics
  - **log/**: Logging utilities
  - **names/**: Random agent name generation
- **tui/src/**: React/TypeScript TUI built with Ink
  - Components, hooks, views for terminal UI
  - Compiled to CommonJS in tui/dist/
- **config.toml**: Default configuration
- **prompts/**: Default role prompt templates

### Key Concepts

**Agents**: Isolated AI assistants running in tmux sessions, each with own git worktree. Agents have roles (engineer, manager, etc.) loaded from workspace role files.

**Workspace**: Project directory with `.bc/` subdirectory containing configuration, state, logs, and per-agent workspaces. Supports both v1 (JSON) and v2 (TOML) config formats.

**Channels**: Persistent SQLite-backed inter-agent communication. Messages can have reactions.

**Memory**: Per-agent persistent knowledge (experiences, learnings). Stored in `.bc/` directory.

**Cost Tracking**: All agent commands have tracked costs, with budgets and spending analytics.

**Roles & Capabilities**: Agents have roles (root, engineer, manager, etc.) with capabilities (create_agents, assign_work, implement_tasks, etc.). Defined in workspace role files (.bc/roles/*.md).

## Important Implementation Details

### Command Structure
- All commands are in `internal/cmd` as a single package
- Use Cobra for CLI framework
- Each command file (agent.go, channel.go, etc.) contains command tree for that feature
- Commands use `*Cmd` variables and `init()` to register subcommands
- Access workspace via `getWorkspace(cmd)` and related helpers

### Database
- SQLite used for persistent storage (channels, cost, events)
- Stored in `.bc/` directory (typically `.bc/state.db` or separate files)
- Create tables with `IF NOT EXISTS` for idempotency
- Use JSON encoding for complex data types

### Testing
- Table-driven tests preferred
- Integration tests use actual workspace creation (see agent_integration_test.go)
- E2E tests use tmux sessions (agent_e2e_test.go)
- Strict linting enforced: errcheck, gosec, govet, noctx, etc.

### Error Handling
- Never ignore errors - use explicit handling or `//nolint:errcheck` with justification
- Context must be propagated through call chains (noctx linter enforces this)
- Exported functions should have clear error documentation

### Version Injection
- Version, commit, commit date are injected via ldflags during build
- Set via `main.version`, `main.commit`, `main.date` in cmd/bc/main.go
- Passed to internal/cmd via `SetVersionInfo()` before command execution

### Configuration
- **v2 config** (config.toml): TOML format with tools, workspace, roster, etc.
- **v1 config** (.bc/config.json): Legacy JSON format
- Generated config code in `config/` package via `make gen` (cfgx tool)
- Role files: `.bc/roles/*.md` contain role definitions, capabilities, hierarchy

### Agent Isolation
- Each agent runs in isolated tmux session
- Each agent has isolated git worktree via `git worktree add`
- Memory/state stored per-agent in `.bc/agents/<name>/`

## Development Practices

**Code Style**
- gofmt with -s (simplify)
- Avoid package-level variables except for cobra commands
- Struct field alignment matters (fieldalignment linter)
- Short identifiers preferred (receiver name 'w' for workspace, 'a' for agent)

**Logging**
- Use pkg/log for all logging
- Verbose flag controlled via `-v` or `--verbose`
- JSON output via `--json` flag

**Testing**
- Write tests for new functionality
- Use `make test` with race detector enabled
- Test edge cases and error paths
- Integration tests are acceptable but should clean up after themselves

**Dependencies**
- Keep minimal: BurntSushi/toml, charmbracelet/x, mattn/go-sqlite3, spf13/cobra
- Run `make deps` to download and tidy

## Linting & Quality

Critical lint rules enforced:
- **errcheck**: All errors must be handled
- **gosec**: Security issues must be addressed
- **govet**: No shadowed variables
- **noctx**: Context must be propagated properly
- **fieldalignment**: Struct fields optimally aligned

Run `make lint` to check. Configuration is in `.golangci.yml` with exceptions for deprecated queue/beads migration.

## Architecture Patterns

**Package Dependencies**
- cmd imports pkg packages, not vice versa
- pkg packages are self-contained
- Use interfaces for loose coupling

**Workspace Access**
```go
ws, err := workspace.Load(rootDir)  // Load existing
ws, err := workspace.Init(rootDir)  // Initialize new
```

**Agent Operations**
```go
agents, err := ws.Agents(ctx)       // List agents
ag, err := agent.Start(ctx, ws, name, role)  // Start new
```

**Channel Communication**
```go
ch, err := ws.Channel(name)         // Get channel
err := ch.Send(agentName, message)  // Send message
```

## TUI (tui/)

- Built with Bun (package manager) and TypeScript/React with Ink
- Compiled to CommonJS in tui/dist/
- Main app in src/app.tsx
- Views, components, hooks organized by feature
- Uses theme system for consistent styling
- Tests in src/__tests__/ using Jest/Bun test runner

Run `make build-tui` to build, `make test-tui` to test, `make lint-tui` to lint.

## Common Tasks

**Add a new command**
1. Create function in internal/cmd/newcmd.go following agent.go pattern
2. Register with rootCmd in internal/cmd/root.go
3. Add tests following cmd_test.go pattern
4. Run `make check` before committing

**Add a new package**
1. Create directory in pkg/newpkg/
2. Implement interfaces and functions
3. Add tests alongside implementation
4. Import in commands as needed

**Debug agent session**
```bash
bc agent attach <name>      # Attach to tmux session
bc agent peek <name>        # Show recent output
bc logs --agent <name>      # Show event logs
```

**Profile performance**
```bash
make bench                   # Run benchmarks
make coverage                # Generate coverage report
```

## References

- README.md: Full feature list and command reference
- CONTRIBUTING.md: Contribution guidelines with detailed examples
- config.toml: Configuration file format and defaults
- internal/cmd/AUDIT.md: Security audit notes
- internal/cmd/MEMORY.md: Memory system documentation
