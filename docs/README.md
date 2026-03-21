# bc Documentation

bc is a CLI-first orchestration system for coordinating teams of AI coding agents across multiple repositories.

## Architecture

| Document | Description |
|----------|-------------|
| [Overview](architecture/overview.md) | System layers, components, data flow diagrams |
| [Database](architecture/database.md) | Complete schema (DDL), ER diagram, indexes, migrations |
| [Agents](architecture/agents.md) | Lifecycle state machine, runtimes, worktrees, roles |
| [MCP](architecture/mcp.md) | Resources, tools, transports, notifications |
| [Web Dashboard](architecture/web-ui.md) | Component tree, routing, state management, SSE |
| [TUI](architecture/tui.md) | Provider tree, navigation, hooks, keyboard |
| [Design System](architecture/design-system.md) | Solar Flare palette, tokens, terminal mapping |
| [Frontend Data Flow](architecture/frontend-data-flow.md) | REST/SSE consumption, caching, delivery |

## API Reference

| Document | Description |
|----------|-------------|
| [REST API](api/rest.md) | All HTTP endpoints with params and response schemas |

## Guides

| Document | Description |
|----------|-------------|
| [Quick Start](guides/quickstart.md) | 5-minute setup |
| [CLI Commands](guides/commands.md) | Full command reference |
| [Channels](guides/channels.md) | Message types, conventions, PR workflow |
| [Memory System](guides/memory.md) | Agent experiences and learnings |
| [Troubleshooting](guides/troubleshooting.md) | Common errors and fixes |

## Development

| Document | Description |
|----------|-------------|
| [Contributing](../CONTRIBUTING.md) | Dev setup, build, test, PR process |

## Engineering Reviews

| Document | Description |
|----------|-------------|
| [Backend](reviews/backend.md) | Architecture, API, data layer, performance |
| [Frontend](reviews/frontend.md) | Web UI, TUI, landing page |
| [Infrastructure](reviews/infrastructure.md) | CI/CD, Docker, deployment |

## Archive

Old docs (v1 architecture, Ink TUI design proposals, competitor research) archived in [bak/](bak/).
