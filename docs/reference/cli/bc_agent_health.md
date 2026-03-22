## bc agent health

Check agent health status

### Synopsis

Check health status of agents including tmux session and state freshness.

Examples:
  bc agent health              # Check all agents
  bc agent health eng-01       # Check specific agent
  bc agent health --json       # Output as JSON
  bc agent health --detect-stuck --alert eng  # Detect stuck and alert

```
bc agent health [agent] [flags]
```

### Options

```
      --alert string          Send alert to channel when stuck agents detected (requires --detect-stuck)
      --detect-stuck          Enable stuck detection analysis
  -h, --help                  help for health
      --json                  Output as JSON
      --max-failures int      Max consecutive failures before considered stuck (default 3)
      --timeout string        Stale state threshold (e.g., 30s, 2m) (default "60s")
      --work-timeout string   Work timeout for stuck detection (e.g., 30m, 1h) (default "30m")
```

### Options inherited from parent commands

```
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

