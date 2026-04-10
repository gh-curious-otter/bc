# Agent SDK Research — bc Integration Opportunities

## Agent SDK Landscape

### 1. Claude Agent SDK (Anthropic)
- **What**: Open-source foundation powering Claude Code — same agent loop, tools, context management
- **Languages**: Python (`claude-agent-sdk-python` v0.1.48), TypeScript (`@anthropic-ai/claude-agent-sdk` v0.2.71)
- **Core concepts**: Tools, hooks, MCP servers, subagents
- **Built-in tools**: Read, Write, Edit, Bash, Glob, Grep, WebSearch, WebFetch, Agent, NotebookEdit
- **Key feature**: Programmable version of Claude Code — same capabilities, full control
- Links: [Overview](https://platform.claude.com/docs/en/agent-sdk/overview) | [GitHub](https://github.com/anthropics/claude-agent-sdk-python) | [Blog](https://www.anthropic.com/engineering/building-agents-with-the-claude-agent-sdk)

### 2. Cursor Cloud Agents API
- **What**: Headless background agents that work on repos programmatically
- **Access**: Via Cursor API key — headless CLI + Background Agent API
- **Key feature**: Self-hosted option — run agents in your own network, code never leaves your machines
- **SDK**: npm `@nothumanwork/cursor-agents-sdk` (unofficial), also Go SDK available
- **MCP support**: `cursor-cloud-agent-mcp` bridges Cursor agents into MCP
- Links: [API Docs](https://cursor.com/docs/cloud-agent/api/endpoints) | [Cursor APIs](https://cursor.com/docs/api)

### 3. Google ADK (Agent Development Kit)
- **What**: Modular framework optimized for Gemini + Google ecosystem, but model-agnostic
- **Languages**: Python, TypeScript, Java, Go — all at 1.0.0
- **Key features**: Code execution, Google Search, context caching, computer use, multi-agent orchestration
- **Deployment**: Containerize to Cloud Run or Vertex AI Agent Engine
- **Philosophy**: "Agent development should feel like software development"
- Links: [Docs](https://google.github.io/adk-docs/) | [GitHub](https://github.com/google/adk-python)

### 4. OpenAI Codex SDK
- **What**: Programmatic control of Codex — CI/CD integration, custom agent workflows
- **Architecture**: Codex CLI runs as MCP server, orchestrated by OpenAI Agents SDK
- **Config**: `AGENTS.md` for custom instructions (like CLAUDE.md)
- **Pricing**: codex-mini at $1.50/M input, $6/M output (75% cache discount)
- **Key feature**: GPT-5.2-Codex — specialized code model within general-purpose architecture
- Links: [SDK Docs](https://developers.openai.com/codex/sdk) | [Agents SDK Guide](https://developers.openai.com/codex/guides/agents-sdk)

### 5. OpenCode
- **What**: Open-source Go-based terminal coding agent, 95K+ stars
- **Architecture**: Client/server — HTTP API, any client can connect
- **Agent types**: Primary (Build, Plan) + custom subagents
- **Models**: 75+ LLMs supported, free to use
- **Key feature**: HTTP server mode — other apps/agents interact over HTTP
- Links: [Docs](https://opencode.ai/docs/) | [GitHub](https://github.com/opencode-ai/opencode)

### 6. Aider
- **What**: Open-source terminal AI pair programming, 39K stars
- **Philosophy**: "Thinks in git" — every edit is a commit, every session is a branch
- **Models**: 100+ LLMs supported
- **No native SDK** — but controllable via AgentAPI (HTTP wrapper)
- Links: [GitHub](https://github.com/Aider-AI/aider)

---

## Universal/Adapter Layers

### Coder AgentAPI
- **What**: Go HTTP server that wraps multiple agents (Claude Code, Goose, Aider, Gemini, Amp, Codex)
- **How**: In-memory terminal emulator translates HTTP calls to keystrokes, parses output
- **Endpoints**: GET conversation, POST message, GET status, SSE stream
- Links: [GitHub](https://github.com/coder/agentapi)

### Rivet Sandbox Agent SDK
- **What**: Universal SDK — single HTTP/SSE API for Claude Code, Codex, OpenCode, Amp
- **Built in**: Rust, 15MB static binary, no runtime deps
- **Solves**: API fragmentation (JSONL vs JSON-RPC vs HTTP+SSE)
- **Runs in**: Docker, E2B, Vercel Sandboxes, Daytona
- Links: [Docs](https://sandboxagent.dev/) | [GitHub](https://github.com/rivet-dev/sandbox-agent)

---

## Claude Agent SDK — Deep Dive for bc

### Current bc approach vs Claude Agent SDK

| What bc does now | How | With Claude Agent SDK |
|---|---|---|
| Start agent | `docker run` + tmux + `claude --dangerously-skip-permissions` | `sdk.query(prompt, options)` — no tmux, no docker exec |
| Set prompt/role | Write `CLAUDE.md` file to worktree | Pass `systemPrompt` parameter directly |
| Add MCP servers | Write `.mcp.json` or `claude mcp add` via docker exec | Pass `mcpServers` in options — in-process or external |
| Add custom tools | Write `.claude/commands/*.md` files | Define `@tool` decorated functions inline |
| Agent attach | `tmux attach -t <session>` | `session.receiveMessages()` — stream events programmatically |
| Resume session | `claude --resume <uuid>` via tmux | `sdk.query(prompt, { resume: sessionId })` |
| Subagents | Agent spawns via Claude's built-in Agent tool | Define subagents inline with `agents` option |
| Monitor state | Parse tmux output for spinner/prompt symbols | Hook into `PostToolUse`, `Stop`, `Notification` events |
| Permissions | `--dangerously-skip-permissions` flag | `permissionMode: "none"` or custom permission hooks |
| File safety | Hope agent doesn't break things | `fileCheckpointing: true` — automatic backups, rollback |

### What the SDK unlocks that bc can't do today

1. **No tmux/docker exec needed** — Direct programmatic control. No terminal emulation, no keystroke parsing, no output scraping. Clean API calls.

2. **Structured event streaming** — Instead of parsing terminal output to detect "working"/"idle"/"stuck", you get typed events: `PreToolUse`, `PostToolUse`, `Stop`, `Notification`, etc.

3. **In-process MCP servers** — Define bc's tools (send_message, report_status) as `@tool` decorated Python/TS functions. No separate MCP server process needed.

4. **Session forking** — `forkSession` lets you branch a conversation to explore different approaches from the same point. Could enable "try two solutions in parallel."

5. **Custom permission hooks** — Instead of all-or-nothing permissions, you can intercept each tool call and approve/deny programmatically. bc could enforce per-role permissions.

6. **File checkpointing** — Automatic file backups before edits. If an agent breaks something, programmatic rollback to any previous state.

7. **Hooks for everything** — `PreToolUse` (intercept before tool runs), `PostToolUse` (react after), `UserPromptSubmit` (modify prompts), `SubagentStart/Stop`. bc could use these for:
   - Cost tracking per tool call
   - Activity events in dashboard
   - Permission enforcement
   - Automatic PR creation after edits

### Architecture comparison

```
Current:  bc → docker run → tmux → claude CLI → terminal output → regex parsing
SDK:      bc → Claude Agent SDK → typed events + structured control
```

### Code example — starting an agent with SDK

```python
# Current bc: 15+ steps (create container, mount volumes, seed settings,
# start tmux, send claude command, parse output...)

# With SDK:
from claude_agent_sdk import ClaudeAgent

agent = ClaudeAgent(
    system_prompt=role.prompt,
    mcp_servers=[bc_mcp_server, github_mcp_server],
    allowed_tools=["Read", "Write", "Edit", "Bash", "mcp__bc__*"],
    permission_mode="none",
    hooks={
        "PostToolUse": lambda event: emit_activity(agent_name, event),
        "Stop": lambda event: update_agent_state(agent_name, "idle"),
    }
)

async for event in agent.query("Check channels and start working"):
    dashboard.stream(event)  # Live activity in web UI
```

### Migration path for bc

1. Keep Docker for isolation (worktrees, filesystem)
2. Replace tmux + claude CLI with SDK inside the container
3. bcd talks to agents via SDK instead of terminal scraping
4. Hooks replace the current state detection regex
5. In-process MCP tools replace the SSE MCP server for agent-local tools

### Key benefits

- Eliminates MCP identity bugs (no more wrong agent name in .mcp.json)
- Eliminates terminal parsing failures
- Eliminates stuck detection heuristics (hooks give exact state)
- Eliminates file sync issues (.mcp.json, settings.json race conditions)
- Native cost tracking per tool call via hooks
- Live structured activity events for dashboard

### Considerations

- bc is written in Go; SDK is Python/TypeScript — would need a sidecar or rewrite
- Docker isolation still valuable for security
- tmux is still useful for human "agent attach" debugging
- Could run SDK as sidecar process inside Docker container, bcd communicates via HTTP

---

## Relevance to bc

bc already does what AgentAPI and Rivet do — orchestrates multiple coding agents with a unified interface. The key integration opportunities:

1. **Claude Agent SDK** — replace tmux/docker exec with native SDK control
2. **Coder AgentAPI** — similar architecture; could share patterns or adopt
3. **Rivet Sandbox Agent** — bc could adopt their universal event schema for provider abstraction
4. **ADK** — native framework for Gemini agents
5. **Codex SDK** — native framework for OpenAI agents

The provider abstraction layer shipped in #2852-#2854 positions bc well to integrate any of these SDKs per provider.

---

## Auth Model Change: OAuth → API Key

### Current (Claude CLI)
- Uses **OAuth** — agents need a Claude.ai account login
- Requires mounting `~/.claude/` and `claude.json` for session persistence
- Agents share the account's rate limits (Pro/Team subscription)
- Costs go through Claude subscription billing
- Plugins (GitHub, etc.) tied to OAuth account

### With SDK
- Uses **`ANTHROPIC_API_KEY`** — no OAuth, no account login
- No `~/.claude/` mount needed — no auth state to persist
- Rate limits from API plan, not account
- Costs through API billing (pay-per-token)
- No Claude.ai plugins — but MCP servers replace them:
  - GitHub plugin → `@modelcontextprotocol/server-github` MCP
  - Web search → built-in `WebSearch` tool in SDK
  - File operations → built-in `Read`/`Write`/`Edit` tools

### What this eliminates from Docker setup

| Volume mount | Needed with SDK? | Why |
|---|---|---|
| `-v worktree:/workspace` | **Yes** | Agent still needs code access |
| `-v agent-dir/.claude:/home/agent/.claude` | **No** | No OAuth state to persist |
| `-v claude.json:/home/agent/.claude.json` | **No** | No OAuth tokens |
| `-v bc-shared-tmp:/tmp/bc-shared` | **Yes** | Screenshot sharing still useful |

### Cost implications

| | Claude CLI (current) | Claude Agent SDK |
|---|---|---|
| Auth | OAuth (Claude Pro/Team) | API key |
| Billing | Subscription ($20-100/mo/seat) | Pay-per-token |
| Input tokens | Included in sub | $3/M (Sonnet), $15/M (Opus) |
| Output tokens | Included in sub | $15/M (Sonnet), $75/M (Opus) |
| Rate limits | Account tier | API tier (higher with scale) |
| Caching | Automatic | 90% discount on cached prompts |
| Best for | Few agents, lots of usage | Many agents, controlled usage |

For bc with 5+ agents doing heavy work, API billing could be significant. But it gives:
- Precise per-agent cost tracking (native, no JSONL parsing)
- No shared rate limit bottleneck
- Programmatic cost controls (stop agent at $X)
- Model selection per agent (cheap model for simple tasks)
