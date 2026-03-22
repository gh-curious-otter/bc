## bc doctor

Health checks and diagnostics

### Synopsis

Run health checks on your bc workspace and dependencies.

Checks workspace config, agent state, databases, tools, and git worktrees.

Categories:
  workspace   .bc/ directory, settings.toml, role files
  database    SQLite integrity and table existence
  agents      Running agents, stale sessions, missing worktrees
  tools       tmux, git, and AI provider installations
  git         Worktree validity and orphaned worktrees

Examples:
  bc doctor                          # Full health check
  bc doctor check workspace          # Check specific category
  bc doctor fix                      # Auto-fix fixable issues
  bc doctor fix --dry-run            # Preview fixes
  bc doctor fix --category git       # Fix specific category

Exit codes:
  0  All checks passed or only warnings
  1  One or more checks failed

```
bc doctor [flags]
```

### Options

```
  -h, --help   help for doctor
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc doctor check](bc_doctor_check.md)	 - Check a specific health category
* [bc doctor fix](bc_doctor_fix.md)	 - Auto-fix fixable issues

