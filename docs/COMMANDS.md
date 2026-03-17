# Command Reference

Complete reference for all bc CLI commands.

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format |
| `-v, --verbose` | Enable verbose output |
| `--help` | Show help for any command |

## Workspace Commands

### bc init

Initialize a new workspace.

```bash
bc init                    # Interactive initialization
bc init --quick            # Quick setup with defaults
bc init --preset solo      # Use solo preset
bc init --preset small-team  # Small team (1 manager, 4 engineers)
bc init --preset full-team   # Full team (PM, managers, engineers, QA)
```

### bc up

Start the root agent.

```bash
bc up                      # Start root agent
```

### bc down

Stop all agents.

```bash
bc down                    # Stop all agents gracefully
```

### bc status

Show workspace and agent status.

```bash
bc status                  # Basic status
bc status --activity       # Include recent activity
bc status --with-worktrees # Show worktree paths
```

### bc workspace stats

Show workspace statistics.

```bash
bc workspace stats         # Workspace metrics
```

### bc home

Open the TUI dashboard.

```bash
bc home                    # Launch terminal UI
```

## Agent Commands

### bc agent create

Create a new agent.

```bash
bc agent create <name> --role <role>
bc agent create eng-01 --role engineer
bc agent create mgr-01 --role manager --team backend
```

Options:
- `--role` - Agent role (required): engineer, manager, tech-lead, ux, product-manager
- `--team` - Team assignment

### bc agent list

List all agents.

```bash
bc agent list              # Basic list
bc agent list --json       # JSON output
```

### bc agent show

Show agent details.

```bash
bc agent show <name>       # Full details
bc agent show eng-01
```

### bc agent send

Send a message to an agent.

```bash
bc agent send <name> <message>
bc agent send eng-01 "Implement feature X"
```

### bc agent broadcast

Send a message to all agents.

```bash
bc agent broadcast <message>
bc agent broadcast "Team meeting in 5 minutes"
```

### bc agent send-to-role

Send a message to all agents with a specific role.

```bash
bc agent send-to-role <role> <message>
bc agent send-to-role engineer "New coding standards in effect"
```

### bc agent send-pattern

Send a message to agents matching a pattern.

```bash
bc agent send-pattern <pattern> <message>
bc agent send-pattern "eng-*" "Sprint planning"
```

### bc agent attach

Attach to an agent's tmux session.

```bash
bc agent attach <name>     # Interactive session
bc agent attach eng-01
```

Press `Ctrl+B D` to detach.

### bc agent peek

View recent agent output without attaching.

```bash
bc agent peek <name>       # Last 50 lines
bc agent peek <name> -n 100  # Last 100 lines
```

### bc agent health

Show agent health status.

```bash
bc agent health            # All agents
bc agent health <name>     # Specific agent
```

### bc agent stop

Stop a running agent.

```bash
bc agent stop <name>
bc agent stop eng-01
```

### bc agent start

Start a stopped agent.

```bash
bc agent start <name>
bc agent start eng-01
```

### bc agent delete

Delete an agent.

```bash
bc agent delete <name>
bc agent delete eng-01
```

### bc agent rename

Rename an agent.

```bash
bc agent rename <old> <new>
bc agent rename eng-01 senior-eng-01
```

## Report Commands

### bc agent report

Report agent state (run from within agent session).

```bash
bc agent reportidle "Waiting for work"
bc agent reportworking "Implementing feature X"
bc agent reportdone "Feature complete"
bc agent reportstuck "Blocked on API access"
bc agent reporterror "Build failed"
```

States: `idle`, `working`, `done`, `stuck`, `error`

## Channel Commands

### bc channel create

Create a communication channel.

```bash
bc channel create <name>
bc channel create eng
bc channel create eng --description "Engineering team channel"
```

### bc channel list

List all channels.

```bash
bc channel list
```

### bc channel send

Send a message to a channel.

```bash
bc channel send <channel> <message>
bc channel send eng "Starting sprint 5"
```

### bc channel history

View channel message history.

```bash
bc channel history <channel>
bc channel history eng --limit 50
```

### bc channel add

Add a member to a channel.

```bash
bc channel add <channel> <agent>
bc channel add eng eng-01
```

### bc channel remove

Remove a member from a channel.

```bash
bc channel remove <channel> <agent>
```

### bc channel join

Current agent joins a channel.

```bash
bc channel join <channel>
```

### bc channel leave

Current agent leaves a channel.

```bash
bc channel leave <channel>
```

### bc channel delete

Delete a channel.

```bash
bc channel delete <channel>
```

## Team Commands

### bc team create

Create a team.

```bash
bc team create <name>
bc team create backend
```

### bc team list

List all teams.

```bash
bc team list
```

### bc team add

Add an agent to a team.

```bash
bc team add <team> <agent>
bc team add backend eng-01
```

### bc team remove

Remove an agent from a team.

```bash
bc team remove <team> <agent>
```

### bc team delete

Delete a team.

```bash
bc team delete <team>
```

### bc team export

Export team configuration.

```bash
bc team export <team> > team-config.json
```

### bc team import

Import team configuration.

```bash
bc team import < team-config.json
```

## Role Commands

### bc role list

List available roles.

```bash
bc role list
```

### bc role show

Show role details.

```bash
bc role show <role>
bc role show engineer
```

## Memory Commands

### bc memory show

Show agent memory.

```bash
bc memory show             # Current agent's memory
bc memory show <agent>     # Specific agent
```

### bc memory learn

Add a learning to memory.

```bash
bc memory learn <category> <content>
bc memory learn patterns "Use table-driven tests"
```

Categories: `patterns`, `bc-usage`, `testing`, `blockers`, `workflows`

### bc memory record

Record an experience.

```bash
bc memory record <content>
bc memory record "Successfully debugged race condition in auth module"
```

### bc memory search

Search memories.

```bash
bc memory search <query>
bc memory search "testing"
```

### bc memory export

Export memory as JSON.

```bash
bc memory export > memory.json
bc memory export <agent> > agent-memory.json
```

## Cost Commands

### bc cost show

Show recent costs.

```bash
bc cost show               # Last 10 entries
bc cost show --limit 50    # More entries
```

### bc cost summary

Show cost summary.

```bash
bc cost summary            # Aggregate statistics
bc cost summary --by agent # Group by agent
```

### bc cost dashboard

Full cost dashboard.

```bash
bc cost dashboard
```

### bc cost add

Manually add a cost entry.

```bash
bc cost add --agent <name> --amount <amount>
```

## Log Commands

### bc logs

View event logs.

```bash
bc logs                    # Recent events
bc logs --tail 50          # Last 50 events
bc logs --agent eng-01     # Filter by agent
bc logs --type agent.spawned  # Filter by event type
```

## Process Commands

### bc process list

List background processes.

```bash
bc process list
```

### bc process stop

Stop a background process.

```bash
bc process stop <pid>
```

## Demon Commands

### bc demon list

List scheduled tasks.

```bash
bc demon list
```

### bc demon run

Manually run a scheduled task.

```bash
bc demon run <name>
```

### bc demon logs

Show task execution history.

```bash
bc demon logs <name>
```

## Worktree Commands

### bc worktree list

List agent worktrees.

```bash
bc worktree list
```

### bc worktree status

Show worktree status.

```bash
bc worktree status <agent>
```

## Plugin Commands

### bc plugin list

List installed plugins.

```bash
bc plugin list
```

### bc plugin install

Install a plugin.

```bash
bc plugin install <name>
bc plugin install <url>
```

### bc plugin remove

Remove a plugin.

```bash
bc plugin remove <name>
```

## MCP Commands

### bc mcp server list

List MCP servers.

```bash
bc mcp server list
```

### bc mcp server status

Show MCP server status.

```bash
bc mcp server status
```

## Remote Commands

### bc remote add

Add a remote host.

```bash
bc remote add <name> <host> --user <user> --key <keypath>
```

### bc remote list

List remote hosts.

```bash
bc remote list
```

### bc remote remove

Remove a remote host.

```bash
bc remote remove <name>
```

### bc remote test

Test connection to remote host.

```bash
bc remote test <name>
```

## Workspace Registry Commands

### bc workspace list

List registered workspaces.

```bash
bc workspace list
```

### bc workspace switch

Switch to another workspace.

```bash
bc workspace switch <name>
```

## Configuration Commands

### bc config show

Show current configuration.

```bash
bc config show
```

### bc config set

Set a configuration value.

```bash
bc config set <key> <value>
```

### bc config export

Export configuration.

```bash
bc config export > config-backup.toml
```

### bc config import

Import configuration.

```bash
bc config import < config-backup.toml
```

## Queue Commands

### bc queue work

Manage work queue.

```bash
bc queue work              # Show incoming work
bc queue work accept <id>  # Accept work item
bc queue work start <id>   # Start working
bc queue work complete <id>  # Mark complete
```

### bc queue merge

Manage merge queue.

```bash
bc queue merge             # Show pending merges
bc queue merge approve <id>  # Approve merge
bc queue merge reject <id>   # Reject with feedback
bc queue merge complete <id>  # Complete merge
```

### bc queue submit

Submit work for review.

```bash
bc queue submit <branch> <message>
```

## Issue Commands

### bc issue view

View issue details.

```bash
bc issue view <id>
bc issue view 123 --comments
```

### bc issue edit

Edit an issue.

```bash
bc issue edit <id> --title "New title"
bc issue edit <id> --add-label bug
bc issue edit <id> --remove-label wontfix
```

### bc issue close

Close an issue.

```bash
bc issue close <id>
bc issue close <id> --reason completed
bc issue close <id> --comment "Fixed in PR #456"
```

### bc issue assign

Assign an issue.

```bash
bc issue assign <id> <user>
bc issue assign 123 @me
bc issue assign 123 --unassign
```

## Audit Commands

### bc audit log

View audit log.

```bash
bc audit log
bc audit log --agent eng-01
bc audit log --action create
```

### bc audit export

Export audit log.

```bash
bc audit export > audit-report.json
```

### bc audit report

Generate audit report.

```bash
bc audit report
bc audit report --format markdown
```
