## bc config show

Show configuration

### Synopsis

Display the current workspace configuration.

If a key is specified, shows only that section. Otherwise shows entire config.

Examples:
  bc config show                  # Show all config
  bc config show tools            # Show tools section
  bc config show tools.claude     # Show specific tool config
  bc config show --json           # Output as JSON

```
bc config show [key] [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc config](bc_config.md)	 - Manage workspace configuration

