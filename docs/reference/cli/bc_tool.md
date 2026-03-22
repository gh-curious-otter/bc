## bc tool

Manage AI tool providers

### Synopsis

Add, remove, and inspect AI tool providers for agent spawning.

Examples:
  bc tool list              # Show all tools with status
  bc tool add myagent       # Add a custom tool
  bc tool show claude       # Show tool details
  bc tool setup claude      # Install and configure a tool
  bc tool status claude     # Check installation status
  bc tool upgrade claude    # Upgrade an installed tool
  bc tool delete mytool     # Remove a custom tool
  bc tool run claude --help # Run a tool directly

### Options

```
  -h, --help   help for tool
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc tool add](bc_tool_add.md)	 - Add a tool to the workspace
* [bc tool delete](bc_tool_delete.md)	 - Remove a tool from the workspace
* [bc tool edit](bc_tool_edit.md)	 - Edit a tool's configuration
* [bc tool list](bc_tool_list.md)	 - List all configured tools and their status
* [bc tool run](bc_tool_run.md)	 - Run a tool directly
* [bc tool setup](bc_tool_setup.md)	 - Install and configure a tool
* [bc tool show](bc_tool_show.md)	 - Show detailed information about a tool
* [bc tool status](bc_tool_status.md)	 - Check installation status of a tool
* [bc tool upgrade](bc_tool_upgrade.md)	 - Upgrade an installed tool

