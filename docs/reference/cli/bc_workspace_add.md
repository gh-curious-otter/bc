## bc workspace add

Add a workspace to the registry

### Synopsis

Register a workspace in the global registry for quick access.

Examples:
  bc workspace add .                        # Add current directory
  bc workspace add ~/projects/frontend      # Add by path
  bc workspace add . --alias fe             # Add with short alias
  bc workspace add ~/api --alias backend    # Add with alias

```
bc workspace add <path> [flags]
```

### Options

```
      --alias string   Short alias for quick access
  -h, --help           help for add
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc workspace](bc_workspace.md)	 - Manage bc workspaces

