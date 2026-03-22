## bc daemon

Manage workspace processes and the bcd server

### Synopsis

Manage long-lived workspace processes (databases, servers, etc.)
and the bcd coordination server.

  bc daemon start          — start the bcd HTTP server
  bc daemon run --name db  — run a workspace process
  bc daemon list           — list running workspace processes
  bc daemon stop [name]    — stop bcd server or a named process
  bc daemon status         — check bcd server health
  bc daemon logs [name]    — view bcd or process logs

### Options

```
  -h, --help   help for daemon
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc daemon list](bc_daemon_list.md)	 - List workspace processes
* [bc daemon logs](bc_daemon_logs.md)	 - Show bcd server or process logs
* [bc daemon restart](bc_daemon_restart.md)	 - Restart a workspace process
* [bc daemon rm](bc_daemon_rm.md)	 - Remove a stopped workspace process
* [bc daemon run](bc_daemon_run.md)	 - Run a named workspace process
* [bc daemon start](bc_daemon_start.md)	 - Start the bcd daemon
* [bc daemon status](bc_daemon_status.md)	 - Show bcd server status
* [bc daemon stop](bc_daemon_stop.md)	 - Stop the bcd server or a named workspace process

