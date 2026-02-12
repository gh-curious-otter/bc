# bc - AI Agent Orchestration for Software Development

`bc` is a CLI-first orchestration system for coordinating teams of AI agents to work on software development projects. It provides a structured, observable, and persistent environment for AI-driven engineering.

## Core Philosophy

- **CLI-First**: Every feature is accessible and scriptable through the `bc` command line.
- **Agent Agnostic**: Works with any AI agent that can run in a terminal (Claude Code, Cursor, Codex, Gemini).
- **Organic Growth**: Start with a single `root` agent and grow your team conversationally.
- **Persistent Memory**: Agents learn from experiences and accumulate knowledge over time.
- **Isolated Workspaces**: Each agent operates in its own `git worktree` for conflict-free development.

## Installation

### Prerequisites

- Go 1.22+
- `tmux`
- A configured AI agent tool (e.g., Claude Code, Cursor)

### Build from Source

```bash
make build
make install
```

## Quick Start

```bash
# 1. Initialize workspace
bc init

# 2. Start the root agent
bc up

# 3. Check status
bc status

# 4. Create an engineer agent
bc agent create --role engineer

# 5. Send work to the agent
bc agent send swift-falcon "Implement the login feature"

# 6. Stop all agents
bc down
```

## Commands

### Workspace Lifecycle

| Command | Description |
|---------|-------------|
| `bc init` | Initialize a new workspace |
| `bc up` | Start agents |
| `bc down` | Stop all agents |
| `bc status` | Show agent status |
| `bc stats` | Show workspace statistics |

### Agent Management

| Command | Description |
|---------|-------------|
| `bc agent create` | Create a new agent |
| `bc agent list` | List all agents |
| `bc agent show <name>` | Show agent details |
| `bc agent send <name> <msg>` | Send message to agent |
| `bc agent attach <name>` | Attach to agent session |
| `bc agent peek <name>` | Show recent output |
| `bc agent stop <name>` | Stop an agent |
| `bc agent delete <name>` | Delete an agent |
| `bc agent rename <old> <new>` | Rename an agent |

### Communication

| Command | Description |
|---------|-------------|
| `bc channel create <name>` | Create a channel |
| `bc channel list` | List channels |
| `bc channel send <ch> <msg>` | Send to channel |
| `bc channel show <name>` | Show channel details |

### Teams

| Command | Description |
|---------|-------------|
| `bc team create <name>` | Create a team |
| `bc team list` | List teams |
| `bc team show <name>` | Show team details |
| `bc team add <team> <agent>` | Add agent to team |
| `bc team remove <team> <agent>` | Remove agent from team |
| `bc team rename <old> <new>` | Rename a team |

### Roles

| Command | Description |
|---------|-------------|
| `bc role create --name <n>` | Create a role |
| `bc role list` | List roles |
| `bc role show <name>` | Show role details |
| `bc role edit <name>` | Edit role in $EDITOR |
| `bc role delete <name>` | Delete a role |

### Scheduled Tasks (Demons)

| Command | Description |
|---------|-------------|
| `bc demon create <name>` | Create scheduled task |
| `bc demon list` | List demons |
| `bc demon show <name>` | Show demon details |
| `bc demon run <name>` | Manually trigger demon |
| `bc demon edit <name>` | Edit demon config |
| `bc demon enable <name>` | Enable a demon |
| `bc demon disable <name>` | Disable a demon |
| `bc demon logs <name>` | Show execution history |

### Background Processes

| Command | Description |
|---------|-------------|
| `bc process start <cmd>` | Start a process |
| `bc process list` | List processes |
| `bc process show <name>` | Show process details |
| `bc process stop <name>` | Stop a process |
| `bc process restart <name>` | Restart a process |
| `bc process logs <name>` | Show process logs |
| `bc process attach <name>` | Attach to process |

### Memory

| Command | Description |
|---------|-------------|
| `bc memory show` | Show agent memory |
| `bc memory record <desc>` | Record an experience |
| `bc memory learn <cat> <text>` | Add a learning |
| `bc memory search <query>` | Search memories |
| `bc memory prune` | Remove old entries |

### Configuration

| Command | Description |
|---------|-------------|
| `bc config show` | Show configuration |
| `bc config get <key>` | Get config value |
| `bc config set <key> <val>` | Set config value |
| `bc config edit` | Edit config in $EDITOR |

### Other

| Command | Description |
|---------|-------------|
| `bc worktree list` | List agent worktrees |
| `bc worktree prune` | Remove orphaned worktrees |
| `bc cost show` | Show cost records |
| `bc cost summary` | Show cost summary |
| `bc logs` | View event log |
| `bc version` | Show version |

## Configuration

Configuration is stored in `.bc/config.toml`. Key settings:

```toml
[workspace]
name = "my-project"

[tools]
default = "claude"

[tools.claude]
command = "claude --dangerously-skip-permissions"
enabled = true

[roster]
engineers = 4
tech_leads = 2
```

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.
