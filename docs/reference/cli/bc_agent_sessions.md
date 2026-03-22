## bc agent sessions

List session history for an agent

### Synopsis

Show stored session IDs for an agent.

The current session ID (if captured) is listed first, followed by archived
session IDs from previous runs.

Examples:
  bc agent sessions eng-01       # List session IDs
  bc agent sessions eng-01 --json

```
bc agent sessions <agent> [flags]
```

### Options

```
  -h, --help   help for sessions
      --json   Output as JSON
```

### Options inherited from parent commands

```
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

