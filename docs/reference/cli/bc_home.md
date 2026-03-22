## bc home

Open the bc TUI dashboard

### Synopsis

Open the bc terminal user interface (TUI) dashboard.

The TUI provides a visual interface for managing agents, channels,
costs, and other bc features using keyboard navigation.

Requirements:
  - bun (or node) must be installed
  - TUI must be built (run 'make build-tui-local' if needed)

Navigation:
  [1-4]  Switch tabs (Dashboard, Agents, Channels, Costs)
  [j/k]  Navigate lists (down/up)
  [?]    Show help
  [q]    Quit

Examples:
  bc home          # Open TUI dashboard

```
bc home [flags]
```

### Options

```
  -h, --help   help for home
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator

