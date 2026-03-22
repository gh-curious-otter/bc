## bc secret set

Create or update a secret

### Synopsis

Create or update an encrypted secret.

The value can be provided via --value, --from-env, or --from-file.
If none are specified, reads from stdin.

Note: --value appears in shell history. For sensitive values, prefer:
  bc secret set API_KEY --from-env API_KEY
  echo "sk-abc123" | bc secret set API_KEY

Examples:
  bc secret set API_KEY --value "sk-abc123"
  bc secret set API_KEY --from-env API_KEY
  bc secret set API_KEY --from-file /path/to/key
  echo "sk-abc123" | bc secret set API_KEY

```
bc secret set <name> [flags]
```

### Options

```
      --desc string        Secret description
      --from-env string    Import value from environment variable
      --from-file string   Import value from file
  -h, --help               help for set
      --value string       Secret value (visible in shell history — prefer --from-env or stdin)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc secret](bc_secret.md)	 - Manage encrypted secrets

