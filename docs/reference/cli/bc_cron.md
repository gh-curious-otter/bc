## bc cron

Manage scheduled agent tasks

### Synopsis

Manage cron jobs that trigger agent prompts or shell commands on a schedule.

Cron expressions use standard 5-field format:
  ┌────── minute (0-59)
  │ ┌──── hour (0-23)
  │ │ ┌── day of month (1-31)
  │ │ │ ┌ month (1-12)
  │ │ │ │ ┌ day of week (0-6, 0=Sun)
  * * * * *

Examples:
  bc cron add daily-lint --schedule "0 9 * * *" --agent qa-01 --prompt "Run make lint"
  bc cron list                          # List all cron jobs
  bc cron show daily-lint               # Show job details
  bc cron enable daily-lint             # Enable a disabled job
  bc cron disable daily-lint            # Disable without deleting
  bc cron run daily-lint                # Trigger manually
  bc cron logs daily-lint --last 10     # Show last 10 executions
  bc cron remove daily-lint             # Delete a job

### Options

```
  -h, --help   help for cron
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc cron add](bc_cron_add.md)	 - Add a new cron job
* [bc cron disable](bc_cron_disable.md)	 - Disable a cron job
* [bc cron enable](bc_cron_enable.md)	 - Enable a cron job
* [bc cron list](bc_cron_list.md)	 - List all cron jobs
* [bc cron logs](bc_cron_logs.md)	 - Show execution history for a cron job
* [bc cron remove](bc_cron_remove.md)	 - Remove a cron job
* [bc cron run](bc_cron_run.md)	 - Manually trigger a cron job
* [bc cron show](bc_cron_show.md)	 - Show cron job details

