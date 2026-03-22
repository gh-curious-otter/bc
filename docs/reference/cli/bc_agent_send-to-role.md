## bc agent send-to-role

Send a message to all agents of a specific role

### Synopsis

Send a message to all running agents that have the specified role.

Examples:
  bc agent send-to-role engineer "run the tests"
  bc agent send-to-role manager "check status"
  bc agent send-to-role tech-lead "review PRs"

```
bc agent send-to-role <role> <message> [flags]
```

### Options

```
  -h, --help   help for send-to-role
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

