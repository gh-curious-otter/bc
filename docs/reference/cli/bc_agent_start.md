## bc agent start

Start a stopped agent

### Synopsis

Start a previously stopped agent from its saved state.

This resurrects the agent's tmux session and memory.
The agent must have been previously created and stopped.
By default, resumes the previous session if available.

The agent's tool (claude, gemini, cursor, etc.) is fixed at creation time
and cannot be changed on restart. Use --runtime to switch infrastructure
backends (tmux vs docker) without changing the agent's identity.

Examples:
  bc agent start eng-01                    # Start stopped agent (resumes session)
  bc agent start eng-01 --fresh            # Force new session
  bc agent start eng-01 --runtime docker   # Override runtime backend

```
bc agent start <agent> [flags]
```

### Options

```
      --fresh            Force new session (ignore saved session)
  -h, --help             help for start
      --resume string    Resume a specific session by ID (e.g. --resume cc78cadf-89ce-4820-ab6e-950afd2b6838)
      --runtime string   Runtime backend override: tmux or docker
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc agent](bc_agent.md)	 - Manage bc agents

