## bc doctor fix

Auto-fix fixable issues

### Synopsis

Attempt to automatically repair fixable issues found by 'bc doctor'.

Fixable issues include:
  - Orphaned git worktrees
  - Missing workspace directories

Use --dry-run to preview actions without making changes.

Examples:
  bc doctor fix                      # Fix all fixable issues
  bc doctor fix --dry-run            # Preview fixes
  bc doctor fix --category git       # Fix specific category

```
bc doctor fix [flags]
```

### Options

```
      --category string   Fix only the specified category
      --dry-run           Preview fixes without making changes
  -h, --help              help for fix
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc doctor](bc_doctor.md)	 - Health checks and diagnostics

