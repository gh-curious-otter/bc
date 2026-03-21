# bc Documentation

bc is a CLI-first orchestration system for coordinating teams of AI coding agents across multiple repositories.

## Backend

| Document | Description |
|----------|-------------|
| [System Overview](backend/overview.md) | Architecture layers, components, data flow diagrams |
| [Agents](backend/agents.md) | Lifecycle state machine, runtimes, worktrees, roles |
| [MCP Server](backend/mcp.md) | Resources, tools, transports, notifications |

## Database

| Document | Description |
|----------|-------------|
| [Schema & Storage](database/database.md) | Complete DDL, ER diagram, indexes, migrations, filesystem layout |

## Frontend

| Document | Description |
|----------|-------------|
| [Web UI](frontend/web-ui.md) | React SPA, component tree, routing, state management |
| [TUI](frontend/tui.md) | React Ink terminal UI, navigation, hooks |
| [Design System](frontend/design-system.md) | Solar Flare palette, tokens, shared component library |
| [Networking](frontend/networking.md) | Client-server communication, SSE events, protocols |

## Infrastructure

| Document | Description |
|----------|-------------|
| [CI/CD](infrastructure/ci-cd.md) | GitHub Actions, test pipeline, release workflow |
| [Deployment](infrastructure/deployment.md) | Docker containers, runtime configuration |

## API Reference

| Document | Description |
|----------|-------------|
| [REST API](api/rest.md) | All HTTP endpoints with params, body, response schemas |

## Guides

| Document | Description |
|----------|-------------|
| [Quick Start](guides/quickstart.md) | 5-minute setup |
| [Channels](guides/channels.md) | Message types, conventions, PR workflow |
| [Troubleshooting](guides/troubleshooting.md) | Common errors and fixes |
| CLI Reference | Run `bc --help` for complete command listing |

## Engineering Reviews

| Document | Description |
|----------|-------------|
| [Backend](reviews/backend.md) | Architecture, API, data layer, performance |
| [Frontend](reviews/frontend.md) | Web UI, TUI, landing page |
| [Infrastructure](reviews/infrastructure.md) | CI/CD, Docker, deployment |

## Archive

Old v1 docs in [bak/](bak/).
