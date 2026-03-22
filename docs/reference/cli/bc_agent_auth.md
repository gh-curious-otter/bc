## bc agent auth

Authenticate an agent for Docker containers

### Synopsis

Run OAuth login for a specific agent. Each agent has its own isolated
credentials directory. Opens a browser for authentication.

Usage:
  bc agent auth my-agent        # Login for a specific agent
  bc agent auth my-agent status # Check auth status

```
bc agent auth <agent-name> [flags]
```

### Options

```
  -h, --help   help for auth
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

