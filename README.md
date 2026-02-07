# bc - Multi-Agent Orchestration for Claude Code

A simpler, more controllable agent orchestrator for coordinating multiple Claude Code agents with predictable behavior and cost awareness.

## Status

**Version:** v2 in development (Lint-Zero complete, 763 violations fixed)

| Milestone | Status |
|-----------|--------|
| Lint-Zero | Complete |
| Epic 1.1: Workspace Restructure | Complete |
| Epic 1.2: Root Agent Singleton | In Progress |

## Features

- **Hierarchical Agent System** - Root, Product Manager, Manager, Tech Lead, Engineers, and QA agents work together
- **Git Worktrees** - Each agent gets an isolated worktree for conflict-free parallel development
- **Work Queue** - Integrated with beads for task tracking and assignment
- **TUI Dashboard** - Real-time visualization of agent status and progress
- **Cost-Aware** - Built-in controls for managing API costs
- **Multi-Tool Support** - Works with Claude Code, Cursor, Codex, and more
- **TOML Configuration** - Clean, human-readable workspace configuration (v2)
- **Per-Agent State** - Individual state files for concurrent access without conflicts (v2)

## Quick Start

```bash
# Initialize a bc workspace in your project
bc init

# Start the root agent
bc up

# View agent status
bc status

# Open TUI dashboard
bc home

# Stop all agents
bc down
```

## Installation

```bash
# Build from source
make build

# Install to GOPATH/bin
make install
```

### Prerequisites

- Go 1.25.1+
- tmux
- Claude Code (or other supported AI agent)
- beads (issue tracker, required for v2)

## CLI Reference

### Workspace Commands

| Command | Description |
|---------|-------------|
| `bc init [dir]` | Initialize a new bc workspace |
| `bc up` | Start agents with default roster |
| `bc down` | Stop all running agents |
| `bc status` | Show agent status |
| `bc dashboard` | Show workspace dashboard with stats |
| `bc home` | Open TUI dashboard |

### Agent Commands

| Command | Description |
|---------|-------------|
| `bc attach <agent>` | Attach to an agent's tmux session |
| `bc spawn <name>` | Spawn a new agent dynamically |
| `bc send <agent> <msg>` | Send a message to an agent |
| `bc report <state> [msg]` | Report agent state (for agents) |

### Work Queue Commands

| Command | Description |
|---------|-------------|
| `bc queue` | List all work items |
| `bc queue add <title>` | Add a work item |
| `bc queue assign <id> <agent>` | Assign work to an agent |
| `bc queue load` | Populate queue from beads issues |
| `bc queue complete <id>` | Mark work item as done |

### Worktree Commands

| Command | Description |
|---------|-------------|
| `bc worktree list` | List all agent worktrees |
| `bc worktree check` | Verify agent is in correct worktree |

### Other Commands

| Command | Description |
|---------|-------------|
| `bc logs` | View event logs |
| `bc merge` | Merge management commands |
| `bc channel` | Communication channel management |
| `bc stats` | View statistics |
| `bc version` | Print version information |

## Agent Hierarchy

```
                         ┌──────────────────┐
                         │       Root       │  Level 0 (singleton)
                         │  (orchestrates)  │
                         └────────┬─────────┘
                                  │
           ┌──────────────────────┼──────────────────────┐
           │                      │                      │
           ▼                      ▼                      ▼
  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
  │ Product Manager │    │    Manager      │    │   Tech Lead     │
  │ (creates epics) │    │ (assigns work)  │    │ (code review)   │
  └────────┬────────┘    └────────┬────────┘    └─────────────────┘
           │                      │
           │         ┌────────────┼────────────┐
           │         │            │            │
           │         ▼            ▼            ▼
           │   ┌──────────┐ ┌──────────┐ ┌──────────┐
           │   │ eng-01   │ │ eng-02   │ │ eng-03   │  Level 2
           │   └──────────┘ └──────────┘ └──────────┘
           │
           │         ┌──────────────────────────┐
           │         ▼                          ▼
           │   ┌──────────┐              ┌──────────┐
           │   │  qa-01   │              │  qa-02   │  Level 2
           │   └──────────┘              └──────────┘
           │
           └──────── Can have multi-level tree structure
```

### Agent Roles

| Role | Level | Purpose |
|------|-------|---------|
| **Root** | 0 | Singleton orchestrator, top-level merge integration |
| **Product Manager** | 1 | Creates epics, prioritizes work, spawns managers |
| **Manager** | 1 | Breaks down epics, assigns to engineers and QA |
| **Tech Lead** | 1 | Reviews code, makes architectural decisions |
| **Engineer** | 2 | Implements tasks, reports progress |
| **QA** | 2 | Tests implementations, validates quality |

## Architecture

bc creates isolated workspaces for each agent using git worktrees:

```
project/
├── .bc/                         # bc workspace directory
│   ├── config.toml              # Workspace configuration (v2: TOML)
│   ├── roles/                   # Role definitions with prompts
│   │   ├── root.md              # Root agent role (required)
│   │   ├── manager.md
│   │   ├── engineer.md
│   │   └── qa.md
│   ├── agents/                  # Per-agent state files (v2)
│   │   ├── root.json            # Root singleton state
│   │   ├── manager-atlas.json
│   │   └── engineer-01.json
│   ├── worktrees/               # Per-agent git worktrees
│   │   ├── root/
│   │   ├── manager-atlas/
│   │   └── engineer-01/
│   ├── channels/                # Communication channels
│   │   └── general.jsonl
│   ├── memory/                  # Per-agent memory (planned)
│   ├── bin/                     # Git wrapper scripts
│   └── events.jsonl             # Append-only event log
└── prompts/                     # Default role prompts
    ├── root.md
    ├── product_manager.md
    ├── manager.md
    ├── engineer.md
    └── qa.md
```

### Key Design Principles

1. **Worktree Isolation** - Each agent works in its own git worktree, preventing merge conflicts
2. **tmux Sessions** - Agents run in tmux for session persistence and easy attachment
3. **Event Sourcing** - All actions logged for debugging and replay
4. **Beads Integration** - Work items sync with beads issue tracker (required in v2)
5. **Per-Agent State** - Individual state files prevent lock contention (v2)
6. **Root Singleton** - Single root agent orchestrates all work (v2)

For detailed architecture documentation, see [`.ctx/`](.ctx/).

## Configuration

### Startup Options

```bash
# Start with custom agent counts
bc up --engineers 5 --qa 3

# Use a different AI agent
bc up --agent cursor
```

### Custom Prompts

Create role-specific prompts in the `prompts/` directory:

- `prompts/root.md` - Root agent instructions
- `prompts/product_manager.md` - Product manager instructions
- `prompts/manager.md` - Manager instructions
- `prompts/engineer.md` - Engineer instructions
- `prompts/qa.md` - QA instructions

## Development

### Build Commands

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `bin/bc` |
| `make test` | Run tests with race detector |
| `make coverage` | Run tests with coverage report |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code |
| `make check` | Run all checks (fmt, vet, test) |
| `make clean` | Remove build artifacts |

### Project Structure

```
bc/
├── cmd/bc/              # CLI entry point
├── config/              # Generated config (cfgx)
├── internal/
│   ├── cmd/             # Cobra command implementations
│   └── tui/             # Application-specific TUI views
├── pkg/
│   ├── agent/           # Agent lifecycle, roles, state management
│   ├── beads/           # Beads issue tracker integration
│   ├── channel/         # Broadcast messaging channels
│   ├── events/          # Event sourcing system
│   ├── git/             # Git operations (worktrees, branches)
│   ├── github/          # GitHub API integration
│   ├── log/             # Structured logging
│   ├── queue/           # Work queue management
│   ├── stats/           # Cost and usage statistics
│   ├── tmux/            # Terminal multiplexer wrapper
│   ├── tui/             # Generic TUI components (Bubble Tea)
│   │   ├── runtime/     # TUI runtime protocol
│   │   └── style/       # Theme and styling
│   └── workspace/       # Workspace config, roles, registry
├── prompts/             # Default role prompts
├── .ctx/                # Architecture documentation
├── .github/workflows/   # CI/CD
├── Makefile
└── README.md
```

## Typical Workflow

```bash
# 1. Initialize workspace (creates v2 structure with root agent)
bc init

# 2. Start the root agent
bc up

# 3. Spawn additional agents as needed
bc spawn pm-01 --role product-manager
bc spawn manager-atlas --role manager --parent pm-01
bc spawn eng-01 --role engineer --parent manager-atlas

# 4. Add work items (or import from beads)
bc queue add "Implement user authentication"
bc queue load   # Import from beads

# 5. Monitor progress
bc status       # Quick status
bc dashboard    # Detailed dashboard
bc home         # Interactive TUI

# 6. Interact with agents
bc attach root           # Attach to root session
bc send manager-atlas "prioritize auth tasks"

# 7. Review and integrate
bc queue        # Check work status
bc merge list   # View pending merges

# 8. Stop when done
bc down
```

## Documentation

- [Architecture Overview](.ctx/01-architecture-overview.md)
- [Agent Roles](.ctx/02-agent-types.md)
- [CLI Reference](.ctx/03-cli-reference.md)
- [Data Models](.ctx/04-data-models.md)
- [Workflows](.ctx/05-workflows.md)

## License

TBD
