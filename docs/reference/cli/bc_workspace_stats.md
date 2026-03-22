## bc workspace stats

Show workspace statistics

### Synopsis

Display statistics about the current workspace including work item
metrics, agent utilization, and completion rates.

Examples:
  bc workspace stats             # human-readable summary
  bc workspace stats --json      # JSON output for scripting
  bc workspace stats --save      # save stats snapshot to .bc/stats.json

```
bc workspace stats [flags]
```

### Options

```
  -h, --help   help for stats
      --json   Output as JSON
      --save   Save stats snapshot to disk
```

### Options inherited from parent commands

```
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc workspace](bc_workspace.md)	 - Manage bc workspaces

