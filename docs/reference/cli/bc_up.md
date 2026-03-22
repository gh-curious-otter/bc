## bc up

Start bc agents

### Synopsis

Start the bc agent system via the bcd daemon.

Starts the root agent through the running bcd daemon.

Examples:
  bc up                      # Start root agent
  bc up --agent cursor       # Use Cursor AI for agents
  bc up --runtime docker     # Use Docker runtime

```
bc up [flags]
```

### Options

```
      --agent string     Agent type from config (e.g. claude, cursor, cursor-agent, codex)
  -h, --help             help for up
      --runtime string   Runtime backend override: tmux or docker
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator

