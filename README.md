# bc - Multi-Agent Orchestration for Claude Code

A simpler, more controllable agent orchestrator for coordinating multiple Claude Code agents with predictable behavior and cost awareness.

## Features

- **Agent Hierarchy** - Coordinator, Product Manager, Manager, Engineers, and QA agents work together
- **Git Worktrees** - Each agent gets an isolated worktree for conflict-free parallel development
- **Work Queue** - Integrated with beads for task tracking and assignment
- **TUI Dashboard** - Real-time visualization of agent status and progress
- **Cost-Aware** - Built-in controls for managing API costs
- **Multi-Tool Support** - Works with Claude Code, Cursor, Codex, and more

## Quick Start

```bash
# Initialize a bc workspace in your project
bc init

# Start agents (default: 3 engineers, 2 QA)
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

- Go 1.23+
- tmux
- Claude Code (or other supported AI agent)

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
                    ┌─────────────────┐
                    │   Coordinator   │
                    │  (orchestrates) │
                    └────────┬────────┘
                             │
           ┌─────────────────┼─────────────────┐
           │                 │                 │
           ▼                 ▼                 ▼
  ┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐
  │ Product Manager │ │   Manager   │ │   (Workers)     │
  │ (creates epics) │ │ (assigns)   │ │                 │
  └─────────────────┘ └──────┬──────┘ │                 │
                             │        │                 │
                ┌────────────┼────────┤                 │
                │            │        │                 │
                ▼            ▼        ▼                 │
         ┌──────────┐ ┌──────────┐ ┌──────────┐        │
         │engineer-01│ │engineer-02│ │engineer-03│       │
         └──────────┘ └──────────┘ └──────────┘        │
                                                        │
                ┌───────────────────────┐               │
                ▼                       ▼               │
         ┌──────────┐            ┌──────────┐          │
         │  qa-01   │            │  qa-02   │          │
         └──────────┘            └──────────┘          │
```

### Agent Roles

| Role | Purpose |
|------|---------|
| **Coordinator** | Orchestrates work, assigns tasks, reviews and integrates |
| **Product Manager** | Creates epics, prioritizes work |
| **Manager** | Breaks down epics, assigns to engineers and QA |
| **Engineer** | Implements tasks, reports progress |
| **QA** | Tests implementations, validates quality |

## Architecture

bc creates isolated workspaces for each agent using git worktrees:

```
project/
├── .bc/
│   ├── config.toml         # Workspace configuration
│   ├── state/              # Agent state, queue, events
│   │   ├── agents.json
│   │   ├── queue.json
│   │   └── events.jsonl
│   └── worktrees/          # Agent worktrees
│       ├── coordinator/
│       ├── product-manager/
│       ├── manager/
│       ├── engineer-01/
│       ├── engineer-02/
│       └── qa-01/
└── prompts/                # Role-specific prompts
    ├── coordinator.md
    ├── product_manager.md
    ├── manager.md
    ├── engineer.md
    └── qa.md
```

### Key Design Principles

1. **Worktree Isolation** - Each agent works in its own git worktree, preventing merge conflicts
2. **tmux Sessions** - Agents run in tmux for session persistence and easy attachment
3. **Event Sourcing** - All actions logged for debugging and replay
4. **Beads Integration** - Work items sync with beads issue tracker

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

- `prompts/coordinator.md` - Coordinator instructions
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
├── internal/cmd/        # Cobra command implementations
├── pkg/
│   ├── agent/           # Agent management
│   ├── beads/           # Beads integration
│   ├── channel/         # Communication channels
│   ├── events/          # Event logging
│   ├── log/             # Logging utilities
│   ├── queue/           # Work queue
│   └── workspace/       # Workspace management
├── prompts/             # Default role prompts
├── .ctx/                # Architecture documentation
├── .github/workflows/   # CI/CD
├── Makefile
└── README.md
```

## Typical Workflow

```bash
# 1. Initialize workspace
bc init

# 2. Add work items (or import from beads)
bc queue add "Implement user authentication"
bc queue add "Add unit tests for auth module"
bc queue load   # Import from beads

# 3. Start agents
bc up

# 4. Monitor progress
bc status       # Quick status
bc dashboard    # Detailed dashboard
bc home         # Interactive TUI

# 5. Interact with agents
bc attach coordinator   # Attach to coordinator session
bc send manager "prioritize auth tasks"

# 6. Review and integrate
bc queue        # Check work status
bc merge list   # View pending merges

# 7. Stop when done
bc down
```

## License

TBD
