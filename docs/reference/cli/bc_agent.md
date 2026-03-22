## bc agent

Manage bc agents

### Synopsis

Manage bc agent lifecycle: create, list, attach, peek, stop, send.

Examples:
  bc agent list                          # List all agents
  bc agent create eng-01 --role engineer # Create new agent
  bc agent attach eng-01                 # Attach to agent session
  bc agent peek eng-01                   # View recent output
  bc agent send eng-01 "run tests"       # Send message to agent
  bc agent stop eng-01                   # Stop agent
  bc agent broadcast "check status"      # Send to all agents
  bc agent send-to-role engineer "test"  # Send to all engineers
  bc agent                               # List all agents (same as bc agent list)
  bc agent send-pattern "eng-*" "hello"  # Send to matching agents

```
bc agent [flags]
```

### Options

```
  -h, --help   help for agent
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc agent attach](bc_agent_attach.md)	 - Attach to an agent's session
* [bc agent auth](bc_agent_auth.md)	 - Authenticate an agent for Docker containers
* [bc agent broadcast](bc_agent_broadcast.md)	 - Send a message to all running agents
* [bc agent cost](bc_agent_cost.md)	 - Show per-agent cost breakdown
* [bc agent create](bc_agent_create.md)	 - Create a new agent
* [bc agent delete](bc_agent_delete.md)	 - Permanently delete an agent
* [bc agent health](bc_agent_health.md)	 - Check agent health status
* [bc agent list](bc_agent_list.md)	 - List all agents
* [bc agent logs](bc_agent_logs.md)	 - Show agent event history
* [bc agent peek](bc_agent_peek.md)	 - Show recent output from an agent
* [bc agent rename](bc_agent_rename.md)	 - Rename an agent
* [bc agent report](bc_agent_report.md)	 - Report agent state (called by agents)
* [bc agent send](bc_agent_send.md)	 - Send a message to an agent
* [bc agent send-pattern](bc_agent_send-pattern.md)	 - Send a message to agents matching a pattern
* [bc agent send-to-role](bc_agent_send-to-role.md)	 - Send a message to all agents of a specific role
* [bc agent sessions](bc_agent_sessions.md)	 - List session history for an agent
* [bc agent show](bc_agent_show.md)	 - Show agent details
* [bc agent start](bc_agent_start.md)	 - Start a stopped agent
* [bc agent stats](bc_agent_stats.md)	 - Show Docker resource stats for an agent
* [bc agent stop](bc_agent_stop.md)	 - Stop an agent

