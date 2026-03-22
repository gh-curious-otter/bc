## bc daemon start

Start the bcd daemon

### Synopsis

Start the bc coordination daemon (bcd).

bcd is an HTTP server that manages agent, channel, and workspace state.
By default it listens on :4880. Use -d to run in the background.

Examples:
  bc daemon start          # Foreground (blocks)
  bc daemon start -d       # Background (daemonized)

```
bc daemon start [flags]
```

### Options

```
  -d, --daemonize   Run in background (daemonized)
  -h, --help        help for start
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc daemon](bc_daemon.md)	 - Manage workspace processes and the bcd server

