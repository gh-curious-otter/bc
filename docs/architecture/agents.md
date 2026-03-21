# Agent Architecture

## What is an Agent?

An AI coding assistant running in an isolated tmux session or Docker container with its own git worktree. Agents have roles (engineer, manager, root) that define their capabilities, prompts, and permissions.

## State Machine

```
         create
           |
           v
      [starting] ──→ [idle] ←──→ [working]
           |            |              |
           v            v              v
        [error]    [stopped]      [stopped]
           |            |              |
           └────────────┴──── delete ──┘
```

Valid transitions:
- starting → idle (session alive)
- starting → error (session failed)
- idle → working (hook: tool_use started)
- working → idle (hook: tool_use completed)
- idle/working → stopped (user stop or session died)
- stopped/error → starting (restart)

State updates come from Claude Code hooks (`POST /api/agents/{name}/hook`) which fire on tool use start/stop events.

## Runtime Backends

### Tmux (`pkg/runtime/tmux.go`)
- Creates `bc-<agent>` tmux session
- `send-keys -l` for message injection (literal mode, `--` prevents arg injection)
- `pipe-pane` for log streaming to `.bc/logs/<agent>.log`
- `capture-pane` for output capture (peek, session ID extraction)

### Docker (`pkg/container/container.go`)
- Each agent gets a container: `bc-<hash>-<agent>`
- Tmux runs inside the container for session management
- Resource limits: 2 CPUs, 2048MB memory (configurable)
- Network: host mode by default (configurable)
- Volumes: workspace mounted read-write, agent auth dir at `/home/agent/.claude/`

## Agent Lifecycle

### Create + Start (`SpawnAgentWithOptions`)
1. Validate name (`IsValidAgentName` — alphanumeric, hyphens, underscores)
2. Create git worktree (`git worktree add`)
3. Setup role files (CLAUDE.md, settings.json, .mcp.json, commands/, skills/, agents/, rules/)
4. Create tmux session or Docker container
5. Start provider command (e.g., `claude --dangerously-skip-permissions`)
6. Persist state to SQLite

### Stop (`StopAgent`)
1. Capture session ID (parse Claude's `--resume <uuid>` output)
2. Archive session ID with timestamp
3. Kill tmux session or Docker container
4. Update state to stopped

### Delete (`DeleteAgent`)
1. Stop if running
2. Remove from state DB
3. **Known bug:** Does not clean up Docker container, git worktree, or branch (#2038)

### Session Resume
On restart, if a valid Claude session UUID exists (36 chars, `[0-9a-f]{8}-...`), the agent starts with `--continue` to resume the conversation. Fixed in #2169 — previously crashed when tmux session names were mistaken for UUIDs.

## Roles

Defined in `.bc/roles/*.md` with YAML frontmatter:

```yaml
---
name: engineer
description: Implements features and fixes bugs
parent_roles: []
mcp_servers: [playwright]
secrets: [GITHUB_TOKEN]
plugins: [github, commit-commands]
prompt_create: "You are an engineer agent..."
settings:
  model: opus
commands:
  lint: "Run linting on the codebase"
---

Main role prompt body (becomes CLAUDE.md in the agent's worktree)
```

Roles support BFS inheritance via `parent_roles`. Child settings override parent.

## Manager (`pkg/agent/agent.go`)

The Manager holds all agent state behind a `sync.RWMutex`. Key concern: the lock is held during slow Docker/tmux subprocess calls (#2106). `RefreshState()` runs on every `GET /api/agents` and shells out to `docker ps` / `tmux list-sessions` for each runtime while holding the write lock.

## Permissions (RBAC)

Defined in `pkg/agent/agent.go:116-193`. Three levels:
- **Root** (level -1): all permissions
- **Manager** (level 0): create/stop/restart agents, send commands, create channels
- **Engineer** (level 1+): view logs, send commands/messages

**Not enforced at API layer** — any HTTP client can call any endpoint regardless of role.
