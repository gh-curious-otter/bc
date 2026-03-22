## bc daemon stop

Stop the bcd server or a named workspace process

### Synopsis

Stop the bcd server or a named workspace process.

Without an argument, sends a shutdown signal to the bcd HTTP server.
With a name, stops the named workspace process.

Examples:
  bc daemon stop           # Stop bcd server
  bc daemon stop postgres  # Stop workspace process

```
bc daemon stop [name] [flags]
```

### Options

```
  -h, --help   help for stop
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc daemon](bc_daemon.md)	 - Manage workspace processes and the bcd server

