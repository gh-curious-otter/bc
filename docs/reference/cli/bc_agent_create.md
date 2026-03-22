## bc agent create

Create a new agent

### Synopsis

Create and start a new agent.

If no name is provided, a random memorable name is generated (e.g., swift-falcon).

Examples:
  bc agent create --role engineer              # Create with random name
  bc agent create worker-01                    # Create with explicit name
  bc agent create eng-01 --role engineer       # Create engineer
  bc agent create qa-01 --role qa --tool cursor # Create QA with Cursor

```
bc agent create [name] [flags]
```

### Options

```
      --env string       Path to env file (KEY=VALUE per line)
  -h, --help             help for create
      --parent string    Parent agent ID (must have permission to create this role)
      --role string      Agent role (required). Use 'bc role list' to see available roles
      --runtime string   Runtime backend override: tmux or docker
      --team string      Team name (alphanumeric)
      --tool string      Agent tool (claude, gemini, cursor, codex, opencode, openclaw, aider)
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

