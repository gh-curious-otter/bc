## bc agent report

Report agent state (called by agents)

### Synopsis

Report the calling agent's current state. This command must be run from within an agent session.

Valid states: idle, working, done, stuck, error

For stuck state, use --reason to provide detailed context:
  bc agent report stuck --reason "database connection timeout"
  bc agent report stuck --reason "auth fails" --reproduction "login with test user" --severity critical

Examples:
  bc agent report working "fixing auth bug"
  bc agent report done "auth bug fixed"
  bc agent report stuck "need database credentials"
  bc agent report stuck --reason "TUI freezes on channel select" --severity high

```
bc agent report <state> [message] [flags]
```

### Options

```
  -h, --help                  help for report
      --reason string         Detailed reason for stuck state
      --reproduction string   Steps to reproduce the issue
      --severity string       Issue severity (critical, high, medium, low) (default "medium")
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

