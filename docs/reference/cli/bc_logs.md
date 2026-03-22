## bc logs

Show the event log

### Synopsis

View the bc event log showing agent spawns, stops, work assignments, and reports.

Examples:
  bc logs                     # Show all events
  bc logs --agent eng-01      # Filter by agent
  bc logs --type agent.report # Filter by event type
  bc logs --since 1h          # Events from last hour
  bc logs --tail 20           # Last N events
  bc logs --full              # Show full messages (no truncation)
  bc logs --json              # JSON output

Event Types:
  agent.started    Agent was created and started
  agent.stopped    Agent was stopped
  agent.report     Agent submitted a progress report
  state.working    Agent started working on task
  state.idle       Agent became idle
  state.stuck      Agent is stuck (may need intervention)

Output:
  TIME      AGENT     TYPE           MESSAGE
  10:15:32  eng-01    state.working  Starting implementation
  10:16:45  eng-01    agent.report   Completed feature X

See Also:
  bc status    Quick agent status overview
  bc home      TUI with activity timeline

```
bc logs [flags]
```

### Options

```
      --agent string   Filter by agent name
      --full           Show full messages without truncation
  -h, --help           help for logs
      --since string   Show events since duration ago (e.g. 1h, 30m)
      --tail int       Show last N events
      --type string    Filter by event type (e.g. agent.report)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator

