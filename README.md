# bc - AI Agent Orchestration for Software Development

[![Build Status](https://github.com/rpuneet/bc/actions/workflows/ci.yml/badge.svg)](https://github.com/rpuneet/bc/actions)
[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

> **Mission control for AI agents.** Coordinate teams of AI agents working together on your codebase.

`bc` is a CLI-first orchestration system for coordinating teams of AI agents to work on software development projects. It provides a structured, observable, and persistent environment for AI-driven engineering.

<p align="center">
  <img src="docs/demo.gif" alt="bc demo showing workspace initialization, agent creation, TUI dashboard, channel communication, and cost tracking" width="720" />
</p>

*Multi-agent orchestration: initialize workspace, spawn engineers, coordinate via channels, track costs*

## Core Philosophy

- **CLI-First**: Every feature is accessible and scriptable through the `bc` command line.
- **Agent Agnostic**: Works with any AI agent that can run in a terminal (Claude Code, Cursor, Codex, Gemini).
- **Organic Growth**: Start with a single `root` agent and grow your team conversationally.
- **Persistent Memory**: Agents learn from experiences and accumulate knowledge over time.
- **Isolated Workspaces**: Each agent operates in its own `git worktree` for conflict-free development.

## Why bc?

| Feature | bc | Single-Agent Tools |
|---------|:--:|:------------------:|
| Multiple parallel agents | ✅ | ❌ |
| Role-based hierarchy | ✅ | ❌ |
| Inter-agent communication | ✅ | ❌ |
| Git worktree isolation | ✅ | ❌ |
| Cost tracking per agent | ✅ | Limited |
| Persistent agent memory | ✅ | Session-only |
| TUI dashboard | ✅ | Varies |

## Supported AI Agents

bc works with any AI agent that runs in a terminal. Use `--tool` to specify:

```bash
bc agent create worker-01 --tool claude    # default
bc agent create worker-01 --tool gemini
bc agent create worker-01 --tool cursor
```

| Agent | Tool Flag | Status | Notes |
|-------|-----------|--------|-------|
| [Claude Code](https://claude.ai/code) | `--tool claude` | ✅ Default | Full support, tested extensively |
| [Gemini](https://ai.google.dev/) | `--tool gemini` | ✅ Supported | Google AI models |
| [Cursor](https://cursor.sh) | `--tool cursor` | ✅ Supported | Via terminal mode |
| [Aider](https://aider.chat) | `--tool aider` | ✅ Supported | Any model backend |
| [Codex CLI](https://github.com/openai/codex) | `--tool codex` | ✅ Supported | OpenAI models |
| [OpenCode](https://github.com/opencode-ai/opencode) | `--tool opencode` | ✅ Supported | Terminal agent |
| [OpenClaw](https://github.com/openclaw/openclaw) | `--tool openclaw` | ✅ Supported | Autonomous agent |
| Custom agents | Configure in config.toml | ✅ Supported | Any CLI tool |

## Installation

### Quick Install (coming soon)

```bash
# Homebrew (macOS/Linux) - coming soon
brew install rpuneet/tap/bc

# Or build from source
git clone https://github.com/rpuneet/bc && cd bc && make install
```

### Prerequisites

- Go 1.22+
- `tmux`
- A configured AI agent tool (e.g., Claude Code, Cursor)

### Build from Source

```bash
git clone https://github.com/rpuneet/bc
cd bc
make build
make install
```

### Shell Completions

Enable tab completion for bc commands, agent names, and channel names:

```bash
# Bash
bc completion bash > /etc/bash_completion.d/bc
# or on macOS with Homebrew:
bc completion bash > $(brew --prefix)/etc/bash_completion.d/bc

# Zsh (add to fpath)
bc completion zsh > "${fpath[1]}/_bc"
# or for Oh My Zsh:
bc completion zsh > ~/.oh-my-zsh/completions/_bc

# Fish
bc completion fish > ~/.config/fish/completions/bc.fish
```

## Quick Start

```bash
# 1. Run bc - prompts to initialize if no workspace exists
bc

# 2. Or explicitly initialize
bc init

# 3. Start the root agent
bc up

# 4. Open the TUI dashboard
bc home

# 5. Check status
bc status

# 6. Create an engineer agent
bc agent create --role engineer

# 7. Send work to the agent
bc agent send swift-falcon "Implement the login feature"

# 8. Stop all agents
bc down
```

**Smart Default**: Running `bc` with no arguments opens the TUI dashboard if a workspace exists, or prompts you to initialize one if not.

## Commands

### Workspace Lifecycle

| Command | Description |
|---------|-------------|
| `bc` | Open TUI dashboard (or prompt to init) |
| `bc init` | Initialize a new workspace |
| `bc up` | Start agents |
| `bc down` | Stop all agents |
| `bc home` | Open TUI dashboard |
| `bc status` | Show agent status |
| `bc stats` | Show workspace statistics |

### Agent Management

| Command | Description |
|---------|-------------|
| `bc agent create` | Create a new agent |
| `bc agent list` | List all agents |
| `bc agent show <name>` | Show agent details |
| `bc agent send <name> <msg>` | Send message to agent |
| `bc agent broadcast <msg>` | Send message to all agents |
| `bc agent send-to-role <role> <msg>` | Send to all agents of a role |
| `bc agent send-pattern <pat> <msg>` | Send to agents matching pattern |
| `bc agent attach <name>` | Attach to agent session |
| `bc agent peek <name>` | Show recent output |
| `bc agent health` | Show agent health status |
| `bc agent stop <name>` | Stop an agent |
| `bc agent delete <name>` | Delete an agent |
| `bc agent rename <old> <new>` | Rename an agent |

### Agent Reporting

| Command | Description |
|---------|-------------|
| `bc report <state> [msg]` | Report agent state (idle, working, done, stuck, error) |

### Communication

| Command | Description |
|---------|-------------|
| `bc channel create <name>` | Create a channel |
| `bc channel list` | List channels |
| `bc channel show <name>` | Show channel details |
| `bc channel send <ch> <msg>` | Send to channel |
| `bc channel add <ch> <agent>` | Add member to channel |
| `bc channel remove <ch> <agent>` | Remove member from channel |
| `bc channel join <ch>` | Join channel (current agent) |
| `bc channel leave <ch>` | Leave channel (current agent) |
| `bc channel history <ch>` | Show channel message history |
| `bc channel react <ch> <msg-id>` | React to a channel message |
| `bc channel delete <name>` | Delete a channel |

### Teams

| Command | Description |
|---------|-------------|
| `bc team create <name>` | Create a team |
| `bc team list` | List teams |
| `bc team show <name>` | Show team details |
| `bc team add <team> <agent>` | Add agent to team |
| `bc team remove <team> <agent>` | Remove agent from team |
| `bc team rename <old> <new>` | Rename a team |
| `bc team delete <name>` | Delete a team |

### Roles

| Command | Description |
|---------|-------------|
| `bc role create --name <n>` | Create a role |
| `bc role list` | List roles |
| `bc role show <name>` | Show role details |
| `bc role edit <name>` | Edit role in $EDITOR |
| `bc role delete <name>` | Delete a role |
| `bc role validate` | Validate all role files |

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
| `bc memory show [agent]` | Show agent memory |
| `bc memory list` | List all agent memories |
| `bc memory record <desc>` | Record an experience |
| `bc memory learn <cat> <text>` | Add a learning |
| `bc memory forget <topic>` | Remove a learning topic |
| `bc memory search <query>` | Search memories |
| `bc memory prune` | Remove old entries |
| `bc memory clear [agent]` | Clear agent memory |
| `bc memory export [agent]` | Export memory to JSON |
| `bc memory import <file>` | Import memories from file |

### Configuration

| Command | Description |
|---------|-------------|
| `bc config show` | Show configuration |
| `bc config get <key>` | Get config value |
| `bc config set <key> <val>` | Set config value |
| `bc config list` | List all config keys |
| `bc config edit` | Edit config in $EDITOR |
| `bc config validate` | Validate config file |
| `bc config reset` | Reset to defaults |

### Cost Tracking

| Command | Description |
|---------|-------------|
| `bc cost show [agent]` | Show cost records |
| `bc cost summary` | Show cost summary |
| `bc cost by-agent` | Show costs grouped by agent |
| `bc cost budget` | Manage cost budgets |
| `bc cost dashboard` | Show comprehensive cost dashboard |
| `bc cost project` | Project future costs |
| `bc cost trends` | Show spending trends |

### Worktrees

| Command | Description |
|---------|-------------|
| `bc worktree list` | List agent worktrees |
| `bc worktree check` | Verify agent's worktree |
| `bc worktree prune` | Remove orphaned worktrees |

### Event Log

| Command | Description |
|---------|-------------|
| `bc logs` | View event log |
| `bc logs --agent <name>` | Filter by agent |
| `bc logs --type <type>` | Filter by event type |
| `bc logs --since <dur>` | Events since duration |

### Other

| Command | Description |
|---------|-------------|
| `bc version` | Show version |
| `bc help` | Show help |

## Configuration

Configuration is stored in `.bc/config.toml`. Key settings:

```toml
[workspace]
name = "my-project"

[user]
nickname = "@yourname"  # Shown in channel messages instead of 'cli'

[tools]
default = "claude"

[tools.claude]
command = "claude --dangerously-skip-permissions"
enabled = true

[tools.gemini]
command = "gemini --yolo"
enabled = true

[tools.cursor]
command = "cursor-agent --force --print"
enabled = false

[tools.aider]
command = "aider --yes"
enabled = false

[roster]
engineers = 4
tech_leads = 2

[performance]
# TUI polling intervals in milliseconds (min: 500ms)
poll_interval_agents = 2000    # Agent status updates
poll_interval_channels = 3000  # Channel message polling
poll_interval_costs = 5000     # Cost data refresh
```

### User Nickname

Your nickname is displayed in channel messages when sending from the CLI:

```bash
# Set your nickname (must start with @, max 15 chars)
bc config set user.nickname @alice

# Messages now show your nickname
bc channel send eng "Hello team!"
# Output: [@alice] Hello team!
```

## TUI Features

The `bc home` dashboard provides a full terminal UI with:

- **Responsive Layout**: Works at minimum 80x24 terminal size
- **14 Views**: Dashboard, Agents, Channels, Costs, Commands, Roles, Logs, Worktrees, Workspaces, Demons, Processes, Memory, Routing, Help
- **Command Palette**: Quick access to all actions via `Ctrl+K`
- **Keyboard Navigation**: Number keys for views, Tab to cycle, j/k in drawer/lists
- **Channel Features**:
  - `@mention` autocomplete with Tab completion
  - Role-based name colors and emoji prefixes
  - Arrow key scrolling in message history
- **Activity Timeline**: Real-time agent activity on dashboard

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `1-9`, `0`, `-` | Jump to view (Dashboard=1, Agents=2, ..., Processes=-) |
| `M` | Memory view |
| `R` | Routing view |
| `?` | Open Help view |
| `Ctrl+K` | Open command palette |
| `Tab` / `Shift+Tab` | Next/previous view |
| `j/k` or `↑/↓` | Navigate drawer/lists |
| `g` / `G` | Jump to first/last item in drawer |
| `m` | Compose message (in Channels view) |
| `i` | Toggle detail pane |
| `@` | Start mention autocomplete |
| `Enter` | Send message / Select item |
| `Esc` | Go back / Cancel |
| `Ctrl+R` | Refresh all data |
| `q` | Quit |

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.
