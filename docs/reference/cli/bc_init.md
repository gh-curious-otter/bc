## bc init

Initialize a new bc v2 workspace

### Synopsis

Initialize a new bc v2 workspace in the specified directory (or current directory).

This creates a .bc directory with v2 configuration for managing agents.

v2 workspace structure:
  .bc/
    settings.json  # Workspace configuration
    roles/         # Agent role definitions
      root.md      # Root agent role
    agents/        # Per-agent state files

Examples:
  bc init                        # Interactive wizard
  bc init --quick                # Quick init with defaults
  bc init --preset solo          # Use solo developer preset
  bc init --preset small-team    # Use small team preset
  bc init --preset full-team     # Use full team preset
  bc init ~/Projects/myapp       # Initialize specific directory

```
bc init [directory] [flags]
```

### Options

```
  -h, --help            help for init
      --preset string   Use preset configuration (solo, small-team, full-team)
      --quick           Quick init with defaults (skip wizard)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator

