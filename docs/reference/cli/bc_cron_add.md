## bc cron add

Add a new cron job

### Synopsis

Create a new scheduled cron job.

One of --agent+--prompt or --command is required.

Examples:
  bc cron add daily-lint --schedule "0 9 * * *" --agent qa-01 --prompt "Run make lint and report"
  bc cron add hourly-check --schedule "0 * * * *" --command "make check"
  bc cron add weekday-standup --schedule "0 9 * * 1-5" --agent root --prompt "Send standup"

```
bc cron add <name> [flags]
```

### Options

```
      --agent string      Target agent name
      --command string    Shell command to run (alternative to --agent+--prompt)
      --disabled          Create job in disabled state
  -h, --help              help for add
      --prompt string     Prompt to send to the agent
      --schedule string   5-field cron expression (required)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc cron](bc_cron.md)	 - Manage scheduled agent tasks

