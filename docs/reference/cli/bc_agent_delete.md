## bc agent delete

Permanently delete an agent

### Synopsis

Permanently delete an agent from the workspace.

This removes the agent's tmux session, channel memberships,
and agent state. Memory is preserved by default for recovery.

Use --force to delete an agent without stopping it first.
Use --purge to also delete the agent's memory directory.

Examples:
  bc agent delete eng-01              # Delete stopped agent (preserves memory)
  bc agent delete eng-01 --force      # Force delete (any state)
  bc agent delete eng-01 --purge      # Delete including memory
  bc agent delete eng-01 --force --purge  # Force delete with full cleanup

```
bc agent delete <agent> [flags]
```

### Options

```
      --force   Force delete running agent without stopping first
  -h, --help    help for delete
      --purge   Also delete agent's memory directory
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

