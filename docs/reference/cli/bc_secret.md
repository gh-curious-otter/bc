## bc secret

Manage encrypted secrets

### Synopsis

Manage encrypted secrets for the workspace.

Secrets store API keys and tokens used by tools, MCP servers, and agents.
Values are encrypted at rest with AES-256-GCM. The API never exposes
secret values in list/show operations.

Other configs reference secrets with ${secret:NAME} syntax:
  [tools.claude-code]
  env = { ANTHROPIC_API_KEY = "${secret:ANTHROPIC_API_KEY}" }

Examples:
  bc secret set ANTHROPIC_API_KEY                    # Prompt for value
  bc secret set ANTHROPIC_API_KEY --value "sk-..."   # Set directly
  bc secret set GITHUB_TOKEN --from-env GITHUB_TOKEN # Import from env var
  bc secret list                                     # List names (no values)
  bc secret show ANTHROPIC_API_KEY                   # Show metadata
  bc secret show ANTHROPIC_API_KEY --reveal          # Show actual value
  bc secret delete ANTHROPIC_API_KEY                 # Delete a secret

### Options

```
  -h, --help   help for secret
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc secret delete](bc_secret_delete.md)	 - Delete a secret
* [bc secret get](bc_secret_get.md)	 - Get a secret value (prints to stdout)
* [bc secret list](bc_secret_list.md)	 - List secrets (names and metadata only)
* [bc secret set](bc_secret_set.md)	 - Create or update a secret
* [bc secret show](bc_secret_show.md)	 - Show secret metadata

