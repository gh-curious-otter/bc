# bc Documentation

Welcome to the bc documentation. **bc** is a CLI-first orchestration system for coordinating teams of AI agents in software development.

## What is bc?

bc enables you to:

- **Orchestrate AI agents** - Manage teams of Claude Code, Cursor, or other AI assistants
- **Coordinate work** - Assign tasks, track progress, communicate between agents
- **Isolate workspaces** - Each agent gets its own git worktree for parallel development
- **Track costs** - Monitor API usage and set budgets per agent or team

## Quick Start

```bash
# Initialize a bc workspace
bc init

# Start the AI agent team
bc up

# View the dashboard
bc home

# Check status
bc status
```

## Philosophy

- **CLI-First**: Every feature is scriptable via the command line
- **Agent Agnostic**: Works with Claude Code, Cursor, Codex, Gemini, or any terminal AI
- **Organic Growth**: Start with one agent, grow conversationally
- **Persistent Memory**: Agents learn and accumulate knowledge
- **Isolated Workspaces**: Each agent gets its own git worktree

## Documentation Overview

| Section | Description |
|---------|-------------|
| [Quick Start](QUICKSTART.md) | 5-minute setup guide |
| [Commands Reference](COMMANDS.md) | Complete CLI reference |
| [Architecture](ARCHITECTURE.md) | System design and data flow |
| [Troubleshooting](TROUBLESHOOTING.md) | Common issues and solutions |

## Support

- **GitHub Issues**: [rpuneet/bc/issues](https://github.com/rpuneet/bc/issues)
- **Source Code**: [rpuneet/bc](https://github.com/rpuneet/bc)
