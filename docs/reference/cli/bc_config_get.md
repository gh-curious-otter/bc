## bc config get

Get a configuration value

### Synopsis

Get a specific configuration value using dot notation.

Examples:
  bc config get workspace.name
  bc config get providers.default
  bc config get providers.default
  bc config get tools.claude.command

```
bc config get <key> [flags]
```

### Options

```
  -h, --help   help for get
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc config](bc_config.md)	 - Manage workspace configuration

