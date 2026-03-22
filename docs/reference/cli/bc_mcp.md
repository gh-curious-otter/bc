## bc mcp

Manage MCP server configurations

### Synopsis

Manage Model Context Protocol (MCP) server configurations.

MCP servers provide tools and resources to AI agents. Configurations are
stored per-workspace and can be referenced by roles.

Examples:
  bc mcp list                                     # List all MCP servers
  bc mcp add github --command npx --args "@modelcontextprotocol/server-github"
  bc mcp add sqlite --command npx --args "@modelcontextprotocol/server-sqlite,/path/to/db"
  bc mcp add remote --transport sse --url "https://api.example.com/mcp"
  bc mcp add github --command npx --env "GITHUB_TOKEN=tok_123"
  bc mcp show github                              # Show server details
  bc mcp remove github                            # Remove a server
  bc mcp disable github                           # Disable a server
  bc mcp enable github                            # Re-enable a server

### Options

```
  -h, --help   help for mcp
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc mcp add](bc_mcp_add.md)	 - Add an MCP server configuration
* [bc mcp disable](bc_mcp_disable.md)	 - Disable an MCP server configuration
* [bc mcp enable](bc_mcp_enable.md)	 - Enable an MCP server configuration
* [bc mcp list](bc_mcp_list.md)	 - List MCP server configurations
* [bc mcp register](bc_mcp_register.md)	 - Register bc as an MCP server in agent settings.json
* [bc mcp remove](bc_mcp_remove.md)	 - Remove an MCP server configuration
* [bc mcp serve](bc_mcp_serve.md)	 - Start bc as an MCP server
* [bc mcp show](bc_mcp_show.md)	 - Show MCP server configuration details

