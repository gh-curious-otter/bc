## bc workspace migrate

Migrate a v1 workspace to v2

### Synopsis

Migrate a bc v1 workspace (.bc/config.json) to v2 (.bc/settings.toml).

bc v2 uses a TOML-based config format. The migration:
  - Reads .bc/config.json (v1 format)
  - Writes .bc/config.json.bak (backup of original)
  - Writes .bc/settings.toml  (v2 format, best-effort field mapping)

Agent state (JSON files) are migrated automatically the next time they
are opened — no manual step needed.

Examples:
  bc workspace migrate          # Check and prompt for migration
  bc workspace migrate ~/myapp  # Check a specific path
  bc workspace migrate --yes    # Migrate without prompting

```
bc workspace migrate [directory] [flags]
```

### Options

```
      --dry-run   Show what would be migrated without making changes
  -h, --help      help for migrate
      --yes       Perform migration without prompting
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc workspace](bc_workspace.md)	 - Manage bc workspaces

