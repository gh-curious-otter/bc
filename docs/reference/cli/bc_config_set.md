## bc config set

Set a configuration value

### Synopsis

Set a specific configuration value using dot notation.

The value type is automatically inferred (string, number, boolean).

Examples:
  bc config set providers.default 6
  bc config set providers.default claude
  bc config set runtime.backend docker
  bc config set tools.claude.command "claude --force"

```
bc config set <key> <value> [flags]
```

### Options

```
  -h, --help   help for set
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc config](bc_config.md)	 - Manage workspace configuration

