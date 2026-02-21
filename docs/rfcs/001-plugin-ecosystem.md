# RFC 001: Plugin Ecosystem for bc

**Issue:** #1405
**Author:** eng-01
**Status:** Draft
**Created:** 2026-02-21

## Summary

Design a minimal viable plugin system for bc that enables extensibility without compromising security or simplicity.

## Motivation

bc needs extensibility to support:
- Custom agent behaviors and tools
- Integration with external services (Jira, Linear, Slack)
- Specialized workflows (CI/CD, testing, deployment)
- Community contributions

## Design Principles

1. **Security First** - Plugins run in isolation, cannot access system without permission
2. **Simplicity** - Easy to write, install, and manage plugins
3. **Compatibility** - Plugins work across bc versions with clear API contracts
4. **Performance** - Minimal overhead for plugin execution

## Plugin Architecture

### Plugin Types

| Type | Description | Execution |
|------|-------------|-----------|
| **Hook** | Intercept bc events (agent.start, channel.send) | In-process |
| **Command** | Add new `bc <plugin> <cmd>` commands | Subprocess |
| **Tool** | External tools agents can invoke | Subprocess |
| **View** | Custom TUI views | TUI component |

### Plugin Format

Plugins are directories with a `plugin.toml` manifest:

```toml
[plugin]
name = "jira-integration"
version = "1.0.0"
description = "Jira issue tracking integration"
author = "example"
license = "MIT"

[plugin.bc]
min_version = "0.2.0"

[hooks]
# Hook into bc events
on_agent_start = "hooks/agent_start.sh"
on_channel_send = "hooks/channel_send.sh"

[commands]
# Add new bc commands
jira = { script = "commands/jira.sh", description = "Manage Jira issues" }

[tools]
# Tools agents can use
create_ticket = { script = "tools/create_ticket.sh", description = "Create Jira ticket" }
```

### Plugin Location

```
~/.bc/plugins/
├── jira-integration/
│   ├── plugin.toml
│   ├── hooks/
│   │   ├── agent_start.sh
│   │   └── channel_send.sh
│   ├── commands/
│   │   └── jira.sh
│   └── tools/
│       └── create_ticket.sh
└── slack-notify/
    ├── plugin.toml
    └── ...
```

### Plugin API

Plugins receive context via environment variables and stdin (JSON):

```bash
#!/bin/bash
# hooks/agent_start.sh

# Environment variables
# BC_PLUGIN_NAME=jira-integration
# BC_EVENT=agent.start
# BC_WORKSPACE=/path/to/workspace
# BC_AGENT_NAME=eng-01

# Stdin contains event payload (JSON)
read -r payload
agent_name=$(echo "$payload" | jq -r '.agent.name')
agent_role=$(echo "$payload" | jq -r '.agent.role')

# Plugin can write to stdout (logged)
echo "Agent $agent_name started with role $agent_role"

# Exit code: 0=success, 1=error (logs warning), 2=abort (cancels operation)
exit 0
```

### Plugin Commands

```bash
# List installed plugins
bc plugin list

# Install from directory or URL
bc plugin install ./my-plugin
bc plugin install https://github.com/user/bc-plugin-jira

# Enable/disable
bc plugin enable jira-integration
bc plugin disable jira-integration

# Remove
bc plugin remove jira-integration

# Run plugin command
bc jira list-issues
```

## Security Model

### Sandboxing

- Plugins run as subprocesses with limited environment
- No direct access to bc internals
- File access restricted to workspace and plugin directory
- Network access requires explicit permission in manifest

### Permissions

```toml
[plugin.permissions]
# Required permissions
network = true        # Allow network access
filesystem = "workspace"  # Access: none, workspace, home, all
env_vars = ["JIRA_*"]    # Allowed env var patterns
```

### Validation

- Plugin manifests are validated on install
- Scripts are checked for dangerous patterns
- Checksums verified for remote installs

## Implementation Plan

### Phase 1: Core Infrastructure (MVP)
1. Plugin manifest parser (`plugin.toml`)
2. Plugin loader and registry
3. Hook execution engine
4. `bc plugin` commands (list, install, remove)

### Phase 2: Enhanced Features
5. Command plugins
6. Tool plugins for agents
7. Permission system
8. Plugin validation

### Phase 3: Ecosystem
9. Plugin marketplace/registry
10. Plugin development tools
11. Documentation and examples

## Alternatives Considered

### Go Plugins
- Pros: Native performance, type safety
- Cons: Version coupling, complex distribution, platform-specific

### WASM
- Pros: Sandboxed, portable
- Cons: Complex toolchain, limited Go ecosystem

### External Processes (Chosen)
- Pros: Language agnostic, simple, natural sandboxing
- Cons: IPC overhead, subprocess management

## Success Metrics

- Plugin development takes < 1 hour for simple hooks
- Installation is single command
- No security incidents from plugin execution
- 5+ community plugins within 3 months

## Open Questions

1. Should plugins be able to modify bc's behavior (middleware pattern)?
2. How to handle plugin conflicts?
3. Should we support a plugin registry/marketplace from day 1?
4. How to version plugin API for backwards compatibility?

## References

- [Cobra Plugin System](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md)
- [Terraform Provider Architecture](https://developer.hashicorp.com/terraform/plugin)
- [Git Hooks](https://git-scm.com/docs/githooks)
