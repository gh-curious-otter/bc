## bc config user

Manage user-level configuration (~/.bcrc)

### Synopsis

Manage user-level configuration stored in ~/.bcrc.

User configuration provides defaults that apply across all bc workspaces:
  - Your nickname for channel messages
  - Default role for new agents
  - Preferred AI tools

Workspace config (.bc/settings.toml) takes precedence over user config.

Examples:
  bc config user init   # Create ~/.bcrc with guided prompts
  bc config user show   # Show user config
  bc config user path   # Show user config path

```
bc config user [flags]
```

### Options

```
  -h, --help   help for user
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc config](bc_config.md)	 - Manage workspace configuration
* [bc config user init](bc_config_user_init.md)	 - Create user configuration file (~/.bcrc)
* [bc config user path](bc_config_user_path.md)	 - Show user configuration file path
* [bc config user show](bc_config_user_show.md)	 - Show user configuration

