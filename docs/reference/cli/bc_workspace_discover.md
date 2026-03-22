## bc workspace discover

Discover and register workspaces

### Synopsis

Scan filesystem for bc workspaces and add them to the registry.

This updates ~/.bc/workspaces.json with newly found workspaces.

Examples:
  bc workspace discover                # Scan default locations
  bc workspace discover --scan ~/work  # Include additional path

```
bc workspace discover [flags]
```

### Options

```
      --depth int      Maximum scan depth (default 3)
  -h, --help           help for discover
      --scan strings   Additional paths to scan
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc workspace](bc_workspace.md)	 - Manage bc workspaces

