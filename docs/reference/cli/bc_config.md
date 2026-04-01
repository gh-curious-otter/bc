## bc config

Manage workspace configuration

### Synopsis

Commands for managing workspace configuration (.bc/settings.json).

Configuration uses a hierarchical key structure with dot notation:
  workspace.name
  providers.claude.command
  providers.default

Examples:
  bc config show                        # Show all config
  bc config get providers.default           # Get a specific value
  bc config set providers.default 6      # Set a value
  bc config list                        # List all config keys
  bc config edit                        # Open config in editor
  bc config validate                    # Validate config file
  bc config reset                       # Reset to defaults

```
bc config [flags]
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

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc config edit](bc_config_edit.md)	 - Edit configuration file
* [bc config get](bc_config_get.md)	 - Get a configuration value
* [bc config list](bc_config_list.md)	 - List all configuration keys
* [bc config reset](bc_config_reset.md)	 - Reset configuration to defaults
* [bc config set](bc_config_set.md)	 - Set a configuration value
* [bc config show](bc_config_show.md)	 - Show configuration
* [bc config user](bc_config_user.md)	 - Manage user-level configuration (~/.bcrc)
* [bc config validate](bc_config_validate.md)	 - Validate configuration file

