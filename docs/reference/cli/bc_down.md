## bc down

Stop bc agents

### Synopsis

Stop all running bc agents.

This will gracefully stop all agent tmux sessions.

Examples:
  bc down          # Stop all agents
  bc down --force  # Force kill without cleanup

```
bc down [flags]
```

### Options

```
      --force   Force kill without cleanup
  -h, --help    help for down
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator

