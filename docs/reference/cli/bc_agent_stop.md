## bc agent stop

Stop an agent

### Synopsis

Stop a specific agent and its tmux session.

Examples:
  bc agent stop eng-01       # Stop eng-01
  bc agent stop eng-01 --force  # Force stop

```
bc agent stop <agent> [flags]
```

### Options

```
      --force   Force stop without cleanup
  -h, --help    help for stop
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

