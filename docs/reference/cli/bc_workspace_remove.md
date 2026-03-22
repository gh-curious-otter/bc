## bc workspace remove

Remove a workspace from the registry

### Synopsis

Unregister a workspace from the global registry.

This does not delete the workspace, just removes it from the registry.

Examples:
  bc workspace remove fe                    # Remove by alias
  bc workspace remove ~/projects/frontend   # Remove by path

```
bc workspace remove <alias|path> [flags]
```

### Options

```
  -h, --help   help for remove
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc workspace](bc_workspace.md)	 - Manage bc workspaces

