## bc workspace switch

Switch active workspace

### Synopsis

Set the active workspace for cross-workspace operations.

Examples:
  bc workspace switch fe                    # Switch by alias
  bc workspace switch ~/projects/frontend   # Switch by path
  bc workspace switch --clear               # Clear active workspace

```
bc workspace switch <alias|path> [flags]
```

### Options

```
      --clear   Clear active workspace
  -h, --help    help for switch
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc workspace](bc_workspace.md)	 - Manage bc workspaces

