## bc agent logs

Show agent event history

### Synopsis

Show the event log history for a specific agent.

Examples:
  bc agent logs eng-01               # Show all events
  bc agent logs eng-01 --since 1h    # Show events from last hour

```
bc agent logs <agent> [flags]
```

### Options

```
  -h, --help           help for logs
      --since string   Show events since duration (e.g., 1h, 30m)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

