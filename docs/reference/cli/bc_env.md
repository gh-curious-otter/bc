## bc env

Manage workspace environment variables

### Synopsis

Configure environment variables for agent sessions.

Environment variables are stored in settings.toml and injected into agent
sessions at startup. Use --provider to set per-provider env vars.

Priority (highest wins): agent --env file > provider env > workspace env.

Examples:
  bc env set SHARED_VAR global                           # workspace [env]
  bc env set --provider claude CLAUDE_CODE_USE_BEDROCK 1 # [providers.claude.env]
  bc env list                                            # all env vars
  bc env list --provider claude                          # claude-only env vars
  bc env get SHARED_VAR
  bc env unset SHARED_VAR
  bc env unset --provider claude CLAUDE_CODE_USE_BEDROCK

### Options

```
  -h, --help              help for env
      --provider string   Target a specific provider (e.g., claude, gemini)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc env get](bc_env_get.md)	 - Get an environment variable value
* [bc env list](bc_env_list.md)	 - List environment variables
* [bc env set](bc_env_set.md)	 - Set an environment variable
* [bc env unset](bc_env_unset.md)	 - Remove an environment variable

