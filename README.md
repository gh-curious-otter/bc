# bc - Multi-Agent Orchestration for Claude Code

A simpler, more controllable agent orchestrator for coordinating multiple Claude Code agents with predictable behavior and cost awareness.

## Features

- **Hierarchical Agent System** - Root, Product Manager, Manager, Tech Lead, Engineers, and QA agents work together
- **Git Worktrees** - Each agent gets an isolated worktree for conflict-free parallel development
- **Channels** - Real-time messaging between agents
- **TUI Dashboard** - Real-time visualization of agent status and progress
- **Multi-Tool Support** - Works with Claude Code, Cursor, Codex, and more

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

## Quick Start

```bash
# Initialize a bc workspace
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

## Basic Usage

```bash
# Spawn agents
bc spawn pm-01 --role product-manager
bc spawn eng-01 --role engineer

# Send messages
bc send eng-01 "implement login feature"

# Attach to agent session
bc attach eng-01

# View logs
bc logs
```

## Documentation

For detailed documentation, see:

- [Architecture Overview](.ctx/01-architecture-overview.md)
- [Agent Roles](.ctx/02-agent-types.md)
- [CLI Reference](.ctx/03-cli-reference.md)
- [Data Models](.ctx/04-data-models.md)
- [Workflows](.ctx/05-workflows.md)
- [Contributing](CONTRIBUTING.md)

## License

TBD
