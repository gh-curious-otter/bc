## bc agent peek

Show recent output from an agent

### Synopsis

Capture and display recent output from an agent's session.

Examples:
  bc agent peek eng-01              # Show last 500 lines
  bc agent peek eng-01 --lines 100  # Show last 100 lines
  bc agent peek eng-01 --follow     # Stream live output (Ctrl+C to stop)

```
bc agent peek <agent> [flags]
```

### Options

```
  -f, --follow      Stream live output (like tail -f)
  -h, --help        help for peek
      --lines int   Number of lines to show (default 500)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

