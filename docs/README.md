# bc Documentation

Welcome to the bc documentation. bc is a CLI-first orchestration system for coordinating teams of AI agents in software development.

## Getting Started

| Document | Description |
|----------|-------------|
| [Quick Start](QUICKSTART.md) | 5-minute setup guide |
| [Commands Reference](COMMANDS.md) | Complete CLI reference |

## Core Concepts

| Document | Description |
|----------|-------------|
| [Architecture](ARCHITECTURE.md) | System design and data flow |
| [Roles & Responsibilities](roles-responsibilities.md) | Agent role hierarchy |
| [Hierarchical Agents](hierarchical-agents.md) | Team structure patterns |
| [Memory System](memory-system.md) | Persistent agent memory |
| [Channel Conventions](channel-conventions.md) | Team communication |

## Extensions

| Document | Description |
|----------|-------------|
| [Plugins](PLUGINS.md) | Plugin development guide |
| [MCP Integration](MCP.md) | Model Context Protocol |

## Reference

| Document | Description |
|----------|-------------|
| [Troubleshooting](TROUBLESHOOTING.md) | Common issues and solutions |
| [TUI Design](tui-design-proposal.md) | Terminal UI specification |
| [TUI Technical](tui-technical-design.md) | TUI implementation details |

## Quick Links

- **Initialize workspace**: `bc init`
- **Start agents**: `bc up`
- **View dashboard**: `bc home`
- **Check status**: `bc status`
- **Get help**: `bc --help`

## Philosophy

- **CLI-First**: Every feature is scriptable via the command line
- **Agent Agnostic**: Works with Claude Code, Cursor, Codex, Gemini, or any terminal AI
- **Organic Growth**: Start with one agent, grow conversationally
- **Persistent Memory**: Agents learn and accumulate knowledge
- **Isolated Workspaces**: Each agent gets its own git worktree

## Support

- GitHub Issues: [github.com/rpuneet/bc/issues](https://github.com/rpuneet/bc/issues)
- Documentation: [github.com/rpuneet/bc/docs](https://github.com/rpuneet/bc/docs)
