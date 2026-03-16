# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Start

**Prerequisites**: Go 1.25.1+, tmux, golangci-lint, make. For TUI: Bun.

**Build**
```bash
make build              # Build binary to bin/bc (runs go generate first)
make build-release      # Optimized release build (stripped symbols)
```

**Test**
```bash
make test                                    # Run all tests with race detector
go test -race -run TestAgentStart ./pkg/agent/  # Run a specific test
make coverage                                # Generate coverage report
make bench                                   # Run benchmarks
```

**Lint & Check**
```bash
make fmt                # Format code with gofmt
make lint               # Run golangci-lint (strict)
make check              # Full check suite: gen + fmt + vet + lint + test
```

**TUI**
```bash
make build-tui          # Build TUI (cd tui && bun install && bun run build)
make test-tui           # Run TUI tests
make lint-tui           # Lint TUI code
bun test src/hooks/__tests__/useStatus.test.tsx  # Run specific TUI test (from tui/)
```

## Architecture

**bc** is a CLI-first AI agent orchestration system built in Go with a TypeScript/React TUI. It coordinates teams of AI agents (Claude Code, Gemini, Cursor, etc.) working in isolated tmux sessions with per-agent git worktrees.

### Package Layout

- **cmd/bc/main.go** → entry point, injects version via ldflags, delegates to internal/cmd
- **internal/cmd/** → all Cobra CLI commands in a single package. Commands are `*Cmd` variables registered via `init()`. Access workspace via `getWorkspace(cmd)` helper.
- **pkg/** → reusable packages (agent, workspace, channel, cost, events, provider, container, tmux, etc.)
- **pkg/provider/** → per-provider files (claude.go, gemini.go, etc.) implementing Provider interface with Binary(), BuildCommand(), InstallHint()
- **config/** → generated from config.toml via `make gen` (uses cfgx tool)
- **tui/src/** → React/Ink terminal UI with 9 views, k9s-style `:command` navigation, compiled to CommonJS in tui/dist/
- **prompts/** → default role prompt templates
- **docker/** → per-provider Dockerfiles (claude, gemini, codex, aider, opencode, openclaw, cursor)

### Key Concepts

- **Agents**: Isolated AI assistants running in tmux sessions or Docker containers. Use `claude -w <name>` for built-in worktree isolation (no custom git worktree management). State stored in SQLite (including root agent as `is_root=1`). Have roles (root, engineer, manager) with capabilities.
- **Workspace**: Project dir with `.bc/` subdirectory for config, state (SQLite), logs. Supports v2 (TOML) and legacy v1 (JSON) config formats.
- **Channels**: SQLite-backed persistent inter-agent communication with reactions.
- **Providers**: Per-provider implementations in pkg/provider/ (claude, gemini, cursor, aider, codex, opencode, openclaw). Each owns its behavior via Provider interface; optional ContainerCustomizer interface for Docker-specific config.
- **Runtime backends**: Default is Docker. Configured via `[runtime]` in config.toml. Tmux also supported.
- **Roles**: Defined in `.bc/roles/*.md` with capabilities (create_agents, assign_work, implement_tasks, etc.) and hierarchy.

### Config Generation

Config code is generated from `config.toml` using the `cfgx` tool (`go generate ./...`). The `make build` target runs `make gen` as a prerequisite. After modifying config.toml, always run `make gen`.

## Implementation Details

### Command Structure
- Single `internal/cmd` package with one file per command group (agent.go, channel.go, etc.)
- Cobra framework with `*Cmd` variables and `init()` registration
- Root command: opens TUI if workspace exists, prompts init if not, shows help in non-interactive mode
- Global flags: `-v/--verbose`, `--json`

### Database
- SQLite for all persistent storage (agent state, channels, cost, events) in `.bc/` directory using WAL mode
- Tables created with `IF NOT EXISTS` for idempotency
- JSON encoding for complex data types

### Testing Patterns
- Table-driven tests preferred
- `TestMain()` in `internal/cmd/` and `pkg/agent/` sets up global `RoleCapabilities` and `RoleHierarchy` maps
- Integration tests use `setupIntegrationWorkspace()` and `seedAgents()` helpers
- E2E tests use live tmux sessions (agent_e2e_test.go, channel_e2e_test.go)
- TUI: test exported helper functions and type interfaces, not hooks directly (hooks can't be tested without DOM in Ink)

### Error Handling
- Never ignore errors — use explicit handling or `//nolint:errcheck` with justification
- `noctx` linter enforces context.Context propagation through all call chains

## Code Style

- gofmt with -s (simplify)
- goimports with local prefix `github.com/rpuneet/bc` (import grouping: stdlib, external, local)
- Short receiver names: `w` for workspace, `a` for agent, `c` for channel
- Avoid package-level variables except for cobra commands
- Struct field alignment matters for memory efficiency (govet fieldalignment)

## Linting

Strict golangci-lint config in `.golangci.yml`. Key linters:
- **errcheck**: all errors handled (type assertions too)
- **govet**: enable-all (includes fieldalignment, shadow, etc.)
- **gosec**: security issues (G104 excluded)
- **noctx**: context propagation
- **staticcheck, bodyclose, prealloc, unconvert, misspell, ineffassign, unused**

Exclusions: deprecated queue/beads migration, test file magic numbers, main.go globals.

## Git Conventions

- Branch naming: `feat/`, `fix/`, `docs/` prefixes
- Conventional commits format
- Run `make check` before committing

## Docker Agent Images

```bash
make build-agent-image          # Build default (claude) agent image
make build-agent-image-gemini   # Build specific provider image
make build-agent-images         # Build all provider images
```

## Architecture Patterns

- cmd imports pkg, never vice versa; pkg packages are self-contained
- Workspace access: `workspace.Load(rootDir)` / `workspace.Init(rootDir)`
- Agent operations: `ws.Agents(ctx)`, `agent.Start(ctx, ws, name, role)`
- Channel communication: `ws.Channel(name)`, `ch.Send(agentName, message)`
- Use interfaces for loose coupling between packages
