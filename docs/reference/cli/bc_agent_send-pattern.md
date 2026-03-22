## bc agent send-pattern

Send a message to agents matching a pattern

### Synopsis

Send a message to all running agents whose names match the given pattern.

Pattern uses glob-style matching (* matches any characters).

Examples:
  bc agent send-pattern "engineer-*" "run tests"
  bc agent send-pattern "eng-0*" "check status"
  bc agent send-pattern "*-lead" "review PRs"

```
bc agent send-pattern <pattern> <message> [flags]
```

### Options

```
  -h, --help   help for send-pattern
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

