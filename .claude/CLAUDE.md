# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Start

**Prerequisites**: Go 1.25.4+, tmux, golangci-lint, make. For TUI: Bun.

**Naming convention**: `make <verb>-<lang|component>[-<runtime>]`
- **lang** = `go` | `ts` (language aggregates)
- **component** = `bc` | `bcd` | `tui` | `web` | `landing`
- **runtime** = `-local` (host machine) | `-docker` (container)

**Build**
```bash
make build                         # Build everything locally (go + ts)
make build-go-local                # Build all Go binaries (bc + bcd)
make build-bc-local                # Build bc CLI binary
make build-bcd-local               # Build bcd server (embeds web UI)
make build-ts-local                # Build all TS packages (tui + web + landing)
make build-bcd-docker              # Build bcd server Docker image
make build-agent-gemini-docker     # Build Gemini agent Docker image
make build-agents-docker           # Build all agent Docker images
make release                       # Optimized release binaries (stripped)
```

**Test**
```bash
make test                                       # Run all tests (go + ts)
make test-go                                    # Run Go tests with race detector
go test -race -run TestAgentStart ./pkg/agent/  # Run a specific Go test
make test-ts                                    # Run all TS tests (tui + web + landing)
make test-tui                                   # Run TUI tests
make test-web                                   # Run web UI tests (vitest)
make coverage-go                                # Go coverage report (60% threshold)
make bench-go                                   # Run Go benchmarks
```

**Lint & Check**
```bash
make lint                  # Run all linters (go + ts)
make lint-go               # Run golangci-lint
make lint-ts               # Run all TS linters (tui + web + landing)
make fmt-go                # Format Go code with gofmt
make vet-go                # Run go vet
make check                 # Full quality gate (go + ts)
make check-go              # Go quality gate: gen + fmt + vet + lint + test
make check-ts              # TS quality gate: lint + test
make integrate             # Full CI equivalent: check + build
```

**Run & Deploy**
```bash
make run-bc-local                  # Run bc CLI from source (go run)
make run-web-local                 # Run web UI dev server (hot reload)
make run-landing-local             # Run landing dev server (hot reload)
make deploy-bcd-local              # Deploy bcd server locally
make deploy-bcd-local ENV=dogfood  # Deploy to dogfood environment
```

**Utilities**
```bash
make deps-go               # Download and tidy Go dependencies
make deps-ts               # Install all TS dependencies (bun install)
make scan-go               # Run govulncheck
make gen-go                # Generate Go code (currently no-op)
make install-bc-local      # Install bc to $GOPATH/bin
make clean                 # Remove all build artifacts
make clean-deps            # Remove artifacts + node_modules
```

## Architecture

**bc** is a CLI-first AI agent orchestration system built in Go with a TypeScript/React TUI. It coordinates teams of AI agents (Claude Code, Gemini, Cursor, etc.) working in isolated tmux sessions with per-agent git worktrees.

### Package Layout

- **cmd/bc/main.go** → entry point, injects version via ldflags, delegates to internal/cmd
- **internal/cmd/** → all Cobra CLI commands in a single package. Commands are `*Cmd` variables registered via `init()`. Access workspace via `getWorkspace(cmd)` helper.
- **pkg/** → reusable packages (agent, workspace, channel, cost, events, memory, tmux, git, etc.)
- **config/** → generated from config.toml via `make gen` (uses cfgx tool)
- **tui/src/** → React/Ink terminal UI with 14 views, compiled to CommonJS in tui/dist/
- **prompts/** → default role prompt templates
- **docker/** → per-provider Dockerfiles (claude, gemini, codex, aider, opencode, openclaw, cursor)

### Key Concepts

- **Agents**: Isolated AI assistants in tmux sessions, each with own git worktree. Have roles (root, engineer, manager) with capabilities. State in `.bc/agents/<name>/`.
- **Workspace**: Project dir with `.bc/` subdirectory for config, state, logs. Supports v2 (TOML) and legacy v1 (JSON) config formats.
- **Channels**: SQLite-backed persistent inter-agent communication with reactions.
- **Memory**: Per-agent persistent knowledge (experiences, learnings).
- **Runtime backends**: Agents run in either tmux sessions or Docker containers, configured via `[runtime]` in config.toml.
- **Roles**: Defined in `.bc/roles/*.md` with capabilities (create_agents, assign_work, implement_tasks, etc.) and hierarchy.

### Config Generation

Config code is generated from `config.toml` using the `cfgx` tool (`go generate ./...`). The `make build` target runs `make gen-go` as a prerequisite. After modifying config.toml, always run `make gen-go`.

## Implementation Details

### Command Structure
- Single `internal/cmd` package with one file per command group (agent.go, channel.go, etc.)
- Cobra framework with `*Cmd` variables and `init()` registration
- Root command: opens TUI if workspace exists, prompts init if not, shows help in non-interactive mode
- Global flags: `-v/--verbose`, `--json`

### Database
- SQLite for persistent storage (channels, cost, events) in `.bc/` directory
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
make build-agent-docker                # Build default agent Docker image (claude)
make build-agent-gemini-docker         # Build specific provider Docker image
make build-agents-docker               # Build all agent Docker images
```

## Architecture Patterns

- cmd imports pkg, never vice versa; pkg packages are self-contained
- Workspace access: `workspace.Load(rootDir)` / `workspace.Init(rootDir)`
- Agent operations: `ws.Agents(ctx)`, `agent.Start(ctx, ws, name, role)`
- Channel communication: `ws.Channel(name)`, `ch.Send(agentName, message)`
- Use interfaces for loose coupling between packages
