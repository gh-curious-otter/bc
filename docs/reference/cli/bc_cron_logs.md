## bc cron logs

Show execution history for a cron job

### Synopsis

Display the execution log for a cron job.

Examples:
  bc cron logs daily-lint
  bc cron logs daily-lint --last 5
  bc cron logs daily-lint --json

```
bc cron logs <name> [flags]
```

### Options

```
  -h, --help       help for logs
      --json       Output as JSON
      --last int   Number of entries to show (default 20)
```

### Options inherited from parent commands

```
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc cron](bc_cron.md)	 - Manage scheduled agent tasks

