## bc daemon logs

Show bcd server or process logs

### Synopsis

Show logs for the bcd server or a named workspace process.

Without an argument, shows bcd server logs.
With a name, shows the named workspace process logs.

Examples:
  bc daemon logs           # bcd server logs
  bc daemon logs postgres  # workspace process logs

```
bc daemon logs [name] [flags]
```

### Options

```
  -h, --help       help for logs
      --tail int   Number of lines to show (default 50)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc daemon](bc_daemon.md)	 - Manage workspace processes and the bcd server

