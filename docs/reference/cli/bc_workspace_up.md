## bc workspace up

Start all roster agents

### Synopsis

Start all agents defined in [roster] of .bc/settings.json.

Agents that are already running are skipped. Missing role files are
created from built-in defaults automatically.

Examples:
  bc workspace up          # Start roster agents
  bc ws up                 # Short alias

```
bc workspace up [flags]
```

### Options

```
  -h, --help   help for up
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc workspace](bc_workspace.md)	 - Manage bc workspaces

