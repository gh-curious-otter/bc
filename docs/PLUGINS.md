# Plugin Development Guide

Extend bc with custom functionality through plugins.

## Overview

Plugins are self-contained modules that extend bc's capabilities. They can add new commands, integrate external tools, or customize agent behavior.

## Plugin Structure

A plugin consists of a directory with a manifest:

```
.bc/plugins/
└── my-plugin/
    ├── plugin.toml      # Plugin manifest (required)
    ├── main.go          # Go plugin (optional)
    ├── script.sh        # Shell scripts (optional)
    └── README.md        # Documentation
```

## Plugin Manifest

Every plugin needs a `plugin.toml`:

```toml
[plugin]
name = "my-plugin"
version = "1.0.0"
description = "My custom plugin"
author = "Your Name"

# Entry point type: "binary", "script", or "hook"
type = "script"
entry = "script.sh"

# bc version compatibility
bc_version = ">=1.0.0"

# Commands this plugin adds
[[commands]]
name = "my-command"
description = "Does something useful"
usage = "bc my-command [args]"

# Hooks this plugin responds to
[[hooks]]
event = "agent.created"
handler = "on-agent-created.sh"

# Dependencies
[dependencies]
requires = ["curl", "jq"]
```

## Plugin Types

### Script Plugins

Simple shell scripts:

```bash
#!/bin/bash
# .bc/plugins/notify/script.sh

# Called with: bc notify <message>
MESSAGE="$1"
curl -X POST "https://hooks.slack.com/..." \
  -d "{\"text\": \"$MESSAGE\"}"
```

### Binary Plugins

Compiled Go plugins:

```go
// .bc/plugins/analytics/main.go
package main

import (
    "fmt"
    "os"
)

func main() {
    // Plugin receives args after plugin name
    args := os.Args[1:]

    switch args[0] {
    case "report":
        generateReport()
    case "stats":
        showStats()
    }
}
```

### Hook Plugins

Respond to bc events:

```toml
# plugin.toml
[[hooks]]
event = "agent.created"
handler = "hooks/on-create.sh"

[[hooks]]
event = "agent.stopped"
handler = "hooks/on-stop.sh"
```

## Available Events

| Event | Description | Data |
|-------|-------------|------|
| `workspace.init` | Workspace initialized | path |
| `agent.created` | Agent created | name, role |
| `agent.started` | Agent started | name |
| `agent.stopped` | Agent stopped | name |
| `agent.deleted` | Agent deleted | name |
| `channel.message` | Message sent | channel, sender, content |
| `report.state` | State reported | agent, state, message |

## Installing Plugins

### From Local Directory

```bash
bc plugin install ./my-plugin
```

### From Git Repository

```bash
bc plugin install https://github.com/user/bc-plugin-name
```

### From Registry (Future)

```bash
bc plugin install plugin-name
```

## Managing Plugins

```bash
# List installed plugins
bc plugin list

# Show plugin details
bc plugin show my-plugin

# Remove a plugin
bc plugin remove my-plugin

# Update a plugin
bc plugin update my-plugin
```

## Plugin Development Workflow

### 1. Create Plugin Directory

```bash
mkdir -p .bc/plugins/my-plugin
cd .bc/plugins/my-plugin
```

### 2. Create Manifest

```bash
cat > plugin.toml << 'EOF'
[plugin]
name = "my-plugin"
version = "0.1.0"
description = "My first plugin"
type = "script"
entry = "main.sh"
EOF
```

### 3. Create Entry Script

```bash
cat > main.sh << 'EOF'
#!/bin/bash
echo "Hello from my-plugin!"
echo "Arguments: $@"
EOF
chmod +x main.sh
```

### 4. Test

```bash
bc my-plugin test arg1 arg2
```

## Plugin API

### Environment Variables

Plugins receive context via environment:

| Variable | Description |
|----------|-------------|
| `BC_WORKSPACE` | Workspace root |
| `BC_PLUGIN_DIR` | Plugin directory |
| `BC_AGENT_ID` | Current agent (if in session) |
| `BC_COMMAND` | Command that triggered plugin |

### Standard Input

For hook handlers, event data is passed via stdin:

```bash
#!/bin/bash
# Read event data from stdin
EVENT_DATA=$(cat)

# Parse with jq
AGENT_NAME=$(echo "$EVENT_DATA" | jq -r '.agent')
EVENT_TYPE=$(echo "$EVENT_DATA" | jq -r '.event')
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Missing dependency |

## Example Plugins

### Slack Notifier

```toml
# .bc/plugins/slack-notify/plugin.toml
[plugin]
name = "slack-notify"
version = "1.0.0"
description = "Send notifications to Slack"
type = "script"
entry = "notify.sh"

[config]
webhook_url = ""  # Set via bc config
```

```bash
#!/bin/bash
# notify.sh
WEBHOOK="${BC_SLACK_WEBHOOK:-$SLACK_WEBHOOK_URL}"
MESSAGE="$*"

curl -X POST "$WEBHOOK" \
  -H "Content-Type: application/json" \
  -d "{\"text\": \"[bc] $MESSAGE\"}"
```

### Git Stats

```toml
# .bc/plugins/git-stats/plugin.toml
[plugin]
name = "git-stats"
version = "1.0.0"
description = "Git repository statistics"
type = "script"
entry = "stats.sh"

[[commands]]
name = "git-stats"
description = "Show git statistics"
```

```bash
#!/bin/bash
# stats.sh
echo "=== Git Statistics ==="
echo "Commits today: $(git log --since='1 day ago' --oneline | wc -l)"
echo "Files changed: $(git diff --stat HEAD~10 | tail -1)"
echo "Contributors: $(git shortlog -sn | wc -l)"
```

### Auto-Review Hook

```toml
# .bc/plugins/auto-review/plugin.toml
[plugin]
name = "auto-review"
version = "1.0.0"
description = "Auto-request code review on PR"
type = "hook"

[[hooks]]
event = "report.done"
handler = "on-done.sh"
```

```bash
#!/bin/bash
# on-done.sh
EVENT=$(cat)
AGENT=$(echo "$EVENT" | jq -r '.agent')
MESSAGE=$(echo "$EVENT" | jq -r '.message')

# Check if PR-related
if [[ "$MESSAGE" == *"PR #"* ]]; then
  PR_NUM=$(echo "$MESSAGE" | grep -o 'PR #[0-9]*' | head -1 | tr -d 'PR #')
  gh pr edit "$PR_NUM" --add-reviewer "@team/reviewers"
fi
```

## Best Practices

1. **Keep it simple**: Single responsibility per plugin
2. **Handle errors**: Always check return codes
3. **Document**: Include README with examples
4. **Test**: Verify plugin works in isolation
5. **Version**: Use semantic versioning
6. **Dependencies**: Declare all requirements

## Debugging

```bash
# Run plugin with verbose output
BC_DEBUG=1 bc my-plugin args

# Check plugin loading
bc plugin list --verbose

# View plugin logs
bc logs --type plugin
```

## Distribution

### Package Your Plugin

```bash
cd .bc/plugins/my-plugin
tar -czf my-plugin-1.0.0.tar.gz *
```

### Publish to GitHub

1. Create repository: `bc-plugin-<name>`
2. Include plugin.toml at root
3. Tag releases: `v1.0.0`

Users can then install:

```bash
bc plugin install https://github.com/user/bc-plugin-name
```
