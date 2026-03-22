## bc workspace config

Manage workspace configuration

### Synopsis

Manage workspace configuration (.bc/settings.toml).

Examples:
  bc workspace config show                    # Show full config
  bc workspace config get providers.default   # Get a value
  bc workspace config set providers.default claude # Set a value
  bc workspace config validate                # Validate config
  bc workspace config edit                    # Open in $EDITOR

```
bc workspace config [flags]
```

### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc workspace](bc_workspace.md)	 - Manage bc workspaces
* [bc workspace config edit](bc_workspace_config_edit.md)	 - Edit configuration file in $EDITOR
* [bc workspace config get](bc_workspace_config_get.md)	 - Get a configuration value
* [bc workspace config set](bc_workspace_config_set.md)	 - Set a configuration value
* [bc workspace config show](bc_workspace_config_show.md)	 - Show configuration
* [bc workspace config validate](bc_workspace_config_validate.md)	 - Validate configuration file

