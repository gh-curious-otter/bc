# MCP Integration Guide

Integrate Model Context Protocol (MCP) servers with bc for enhanced agent capabilities.

## Overview

MCP (Model Context Protocol) is a standard for AI tools to provide capabilities to language models. bc supports running MCP servers and making their tools available to agents.

## What is MCP?

MCP servers expose:
- **Tools**: Functions agents can call (file operations, web search, etc.)
- **Resources**: Data sources agents can read
- **Prompts**: Pre-defined prompt templates

## Configuration

### config.toml

```toml
[mcp]
enabled = true

[[mcp.servers]]
name = "filesystem"
command = "mcp-server-filesystem"
args = ["/path/to/allowed/dir"]
enabled = true

[[mcp.servers]]
name = "github"
command = "mcp-server-github"
env = { GITHUB_TOKEN = "env:GITHUB_TOKEN" }
enabled = true

[[mcp.servers]]
name = "web-search"
command = "mcp-server-brave-search"
env = { BRAVE_API_KEY = "env:BRAVE_API_KEY" }
enabled = false
```

## Managing MCP Servers

### List Servers

```bash
bc mcp server list
```

Output:
```
NAME         STATUS   TOOLS  COMMAND
filesystem   running  5      mcp-server-filesystem
github       running  8      mcp-server-github
web-search   stopped  -      mcp-server-brave-search
```

### Check Status

```bash
bc mcp server status
```

### Start/Stop Servers

```bash
bc mcp server start filesystem
bc mcp server stop github
bc mcp server restart all
```

### View Server Tools

```bash
bc mcp server tools filesystem
```

Output:
```
TOOL              DESCRIPTION
read_file         Read contents of a file
write_file        Write content to a file
list_directory    List directory contents
create_directory  Create a new directory
delete_file       Delete a file
```

## Available MCP Servers

### Official Servers

| Server | Description | Installation |
|--------|-------------|--------------|
| `mcp-server-filesystem` | File operations | `npm i -g @modelcontextprotocol/server-filesystem` |
| `mcp-server-github` | GitHub API | `npm i -g @modelcontextprotocol/server-github` |
| `mcp-server-postgres` | PostgreSQL | `npm i -g @modelcontextprotocol/server-postgres` |
| `mcp-server-sqlite` | SQLite | `npm i -g @modelcontextprotocol/server-sqlite` |
| `mcp-server-brave-search` | Web search | `npm i -g @anthropic/server-brave-search` |
| `mcp-server-puppeteer` | Browser automation | `npm i -g @anthropic/server-puppeteer` |

### Installing Servers

```bash
# Via npm
npm install -g @modelcontextprotocol/server-filesystem

# Via pip
pip install mcp-server-fetch

# Via cargo
cargo install mcp-server-git
```

## Server Configuration Examples

### Filesystem Server

```toml
[[mcp.servers]]
name = "filesystem"
command = "mcp-server-filesystem"
args = [
  "${BC_WORKSPACE}",           # Allow workspace access
  "${HOME}/Documents"          # Additional allowed path
]
```

### GitHub Server

```toml
[[mcp.servers]]
name = "github"
command = "mcp-server-github"
env = { GITHUB_TOKEN = "env:GITHUB_TOKEN" }
```

Required: `GITHUB_TOKEN` environment variable.

### PostgreSQL Server

```toml
[[mcp.servers]]
name = "postgres"
command = "mcp-server-postgres"
args = ["postgresql://user:pass@localhost/mydb"]
```

### Custom Server

```toml
[[mcp.servers]]
name = "my-tools"
command = "/path/to/my-mcp-server"
args = ["--config", "/path/to/config.json"]
env = { MY_API_KEY = "env:MY_API_KEY" }
working_dir = "/path/to/workdir"
```

## Agent Usage

When agents have MCP tools available, they can use them naturally:

```
Agent: I need to read the config file.
[MCP filesystem] read_file("config.toml")
[Result] Contents of config.toml...

Agent: Let me check the open issues.
[MCP github] list_issues("rpuneet/bc", {"state": "open"})
[Result] 5 open issues...
```

## Security

### Sandboxing

MCP servers run with restricted access:

```toml
[[mcp.servers]]
name = "filesystem"
command = "mcp-server-filesystem"
args = ["/allowed/path"]  # Only this path is accessible
```

### Environment Variables

Use `env:` prefix to reference environment variables securely:

```toml
env = { API_KEY = "env:MY_API_KEY" }  # Reads from environment
```

Never hardcode secrets in config.toml.

### Permissions

Control which agents can use which servers:

```toml
[[mcp.servers]]
name = "github"
command = "mcp-server-github"
allowed_roles = ["engineer", "tech-lead"]  # Role restriction
```

## Developing MCP Servers

### Server Specification

MCP servers communicate via JSON-RPC over stdio:

```
Client → Server: {"jsonrpc": "2.0", "method": "tools/list", "id": 1}
Server → Client: {"jsonrpc": "2.0", "result": {...}, "id": 1}
```

### Python Example

```python
# my_mcp_server.py
from mcp import Server, Tool

server = Server("my-server")

@server.tool()
async def greet(name: str) -> str:
    """Greet someone by name."""
    return f"Hello, {name}!"

if __name__ == "__main__":
    server.run()
```

### TypeScript Example

```typescript
// my_mcp_server.ts
import { Server, Tool } from "@modelcontextprotocol/sdk";

const server = new Server("my-server");

server.tool("greet", "Greet someone", {
  name: { type: "string", description: "Name to greet" }
}, async ({ name }) => {
  return `Hello, ${name}!`;
});

server.run();
```

### Go Example

```go
// main.go
package main

import "github.com/modelcontextprotocol/sdk-go"

func main() {
    server := mcp.NewServer("my-server")

    server.Tool("greet", "Greet someone", func(args map[string]any) (any, error) {
        name := args["name"].(string)
        return fmt.Sprintf("Hello, %s!", name), nil
    })

    server.Run()
}
```

## Debugging

### Server Logs

```bash
# View MCP server output
bc mcp server logs filesystem

# Tail logs
bc mcp server logs filesystem --follow
```

### Test Server Connection

```bash
bc mcp server test filesystem
```

### Manual Tool Call

```bash
bc mcp call filesystem read_file '{"path": "README.md"}'
```

## Troubleshooting

### Server Won't Start

1. Check if command is in PATH
2. Verify required environment variables
3. Check config.toml syntax
4. View server logs: `bc mcp server logs <name>`

### Tools Not Available

1. Verify server is running: `bc mcp server status`
2. Check server tools: `bc mcp server tools <name>`
3. Restart server: `bc mcp server restart <name>`

### Permission Denied

1. Check allowed paths in server config
2. Verify API key/token is set
3. Check role restrictions

## Best Practices

1. **Minimal Permissions**: Only allow necessary paths/actions
2. **Environment Variables**: Use `env:` for secrets
3. **Role Restrictions**: Limit server access by agent role
4. **Logging**: Enable logging for audit trail
5. **Testing**: Test tools manually before agent use

## Resources

- [MCP Specification](https://modelcontextprotocol.io/docs/specification)
- [Official MCP Servers](https://github.com/modelcontextprotocol/servers)
- [MCP SDK (TypeScript)](https://github.com/modelcontextprotocol/sdk-ts)
- [MCP SDK (Python)](https://github.com/modelcontextprotocol/sdk-python)
