## bc workspace list

List discovered workspaces

### Synopsis

List all bc workspaces on this machine.

Searches:
  - Global registry (~/.bc/workspaces.json)
  - Common directories (~/Projects, ~/Developer, ~/dev, ~/code, ~/repos, ~/src)
  - Additional paths specified with --scan

Examples:
  bc workspace list                    # List all workspaces
  bc workspace list --json             # Output as JSON
  bc workspace list --scan ~/work      # Include additional path
  bc workspace list --no-cache         # Skip registry, scan only

```
bc workspace list [flags]
```

### Options

```
      --depth int      Maximum scan depth (default 3)
  -h, --help           help for list
      --no-cache       Skip registry, scan filesystem only
      --scan strings   Additional paths to scan
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc workspace](bc_workspace.md)	 - Manage bc workspaces

