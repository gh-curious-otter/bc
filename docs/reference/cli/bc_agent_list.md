## bc agent list

List all agents

### Synopsis

List all agents with their status, role, and current task.

Examples:
  bc agent list          # List all agents
  bc agent list --json   # Output as JSON
  bc agent list --role engineer  # Filter by role

```
bc agent list [flags]
```

### Options

```
      --full            Include full agent data including prompts (with --json)
  -h, --help            help for list
      --json            Output as JSON (compact by default)
      --role string     Filter by role
      --status string   Filter by status (running, stopped, error)
```

### Options inherited from parent commands

```
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

