# RFC 005: Multi-Provider Agent Architecture

**Issue:** #1429, #1430
**Author:** eng-03
**Status:** Draft
**Created:** 2026-02-22

## Summary

Design a provider abstraction layer enabling bc to orchestrate agents from multiple AI platforms (Claude, Gemini, OpenClaw, Cursor, Codex) through a unified interface.

## Motivation

- bc's value is orchestration, not being an AI itself
- Different AI tools have different strengths
- Users want to use their preferred AI providers
- Multi-provider enables cost optimization (cheaper models for simple tasks)
- Competitive advantage: no other tool orchestrates multiple AI agents

## Design Principles

1. **Provider Agnostic** - Core bc logic independent of provider
2. **Unified Interface** - Same `bc agent` commands for all providers
3. **Provider Strengths** - Leverage each provider's unique capabilities
4. **Fallback Support** - Gracefully handle provider unavailability
5. **Cost Awareness** - Track costs per provider

## Scope

### In Scope (MVP)

| Feature | Description |
|---------|-------------|
| Provider Interface | Abstract interface for all providers |
| Claude Provider | Current implementation, refactored |
| OpenClaw Provider | CLI-based agent integration |
| Gemini Provider | API-based Google AI |
| Provider Config | Per-agent provider settings |

### Out of Scope (MVP)

| Feature | Rationale |
|---------|-----------|
| Cursor Integration | IDE-based, complex subprocess |
| Codex | OpenAI API deprecation concerns |
| Auto-routing | Future enhancement |
| Provider marketplace | Depends on RFC 004 |

## Technical Design

### Provider Interface

```go
// pkg/agent/provider/provider.go

// Provider defines the interface for AI agent providers
type Provider interface {
    // Name returns the provider identifier
    Name() string

    // Initialize sets up provider (API keys, connections)
    Initialize(ctx context.Context, config ProviderConfig) error

    // Spawn creates a new agent session
    Spawn(ctx context.Context, agent *Agent, task string) (Session, error)

    // Status returns current agent state
    Status(ctx context.Context, session Session) (AgentState, error)

    // SendMessage sends task/message to agent
    SendMessage(ctx context.Context, session Session, message string) error

    // ReadOutput reads agent's response/output
    ReadOutput(ctx context.Context, session Session) ([]Output, error)

    // Stop gracefully stops agent
    Stop(ctx context.Context, session Session) error

    // Kill forcefully terminates agent
    Kill(ctx context.Context, session Session) error

    // Capabilities returns provider capabilities
    Capabilities() Capabilities
}

// Session represents an active agent session
type Session interface {
    ID() string
    Provider() string
    PID() int  // Process ID for CLI providers
}

// Capabilities describes what a provider supports
type Capabilities struct {
    Streaming    bool     // Real-time output
    Tools        []string // Available tools
    MaxContext   int      // Max context window
    CostTracking bool     // Cost reporting supported
    Models       []string // Available models
}
```

### Provider Types

```
┌─────────────────────────────────────────────────────────────┐
│                    bc Orchestration Layer                    │
├─────────────────────────────────────────────────────────────┤
│   AgentManager   ChannelManager   MemoryManager   Costs     │
└────────────────────────────┬────────────────────────────────┘
                             │
                    ┌────────┴────────┐
                    │ Provider Router │
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
   ┌────┴────┐         ┌────┴────┐         ┌────┴────┐
   │  CLI    │         │   API   │         │  IDE    │
   │Provider │         │Provider │         │Provider │
   └────┬────┘         └────┬────┘         └────┬────┘
        │                   │                    │
   ┌────┴────┐         ┌────┴────┐         ┌────┴────┐
   │OpenClaw │         │ Claude  │         │ Cursor  │
   │ Aider   │         │ Gemini  │         │         │
   │OpenCode │         │ Codex   │         │         │
   └─────────┘         └─────────┘         └─────────┘
```

### CLI Provider (OpenClaw, Aider, OpenCode)

CLI providers run as subprocesses in tmux sessions:

```go
// pkg/agent/provider/cli.go

type CLIProvider struct {
    name    string
    binary  string  // Path to CLI binary
    args    []string
    tmux    *tmux.Manager
}

func (p *CLIProvider) Spawn(ctx context.Context, agent *Agent, task string) (Session, error) {
    // Create tmux session
    sessionName := fmt.Sprintf("bc-%s-%s", p.name, agent.Name)

    // Build command
    cmd := append([]string{p.binary}, p.args...)
    cmd = append(cmd, task)

    // Launch in tmux
    err := p.tmux.CreateSession(sessionName, strings.Join(cmd, " "))
    if err != nil {
        return nil, err
    }

    return &CLISession{
        id:       sessionName,
        provider: p.name,
        tmux:     p.tmux,
    }, nil
}
```

### API Provider (Claude, Gemini)

API providers use HTTP clients:

```go
// pkg/agent/provider/api.go

type APIProvider struct {
    name     string
    client   *http.Client
    endpoint string
    apiKey   string
    model    string
}

func (p *APIProvider) Spawn(ctx context.Context, agent *Agent, task string) (Session, error) {
    // Create conversation/session via API
    resp, err := p.client.Post(p.endpoint+"/conversations", "application/json", ...)
    if err != nil {
        return nil, err
    }

    // Parse session ID from response
    var result struct {
        ID string `json:"conversation_id"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    return &APISession{
        id:       result.ID,
        provider: p.name,
        client:   p.client,
    }, nil
}
```

### Provider Registry

```go
// pkg/agent/provider/registry.go

var providers = map[string]Provider{}

func Register(p Provider) {
    providers[p.Name()] = p
}

func Get(name string) (Provider, error) {
    p, ok := providers[name]
    if !ok {
        return nil, fmt.Errorf("unknown provider: %s", name)
    }
    return p, nil
}

func init() {
    // Register built-in providers
    Register(&ClaudeProvider{})
    Register(&OpenClawProvider{})
    Register(&GeminiProvider{})
}
```

### Configuration

Agent-level provider config:

```toml
# .bc/config.toml

[providers.claude]
api_key = "${ANTHROPIC_API_KEY}"
model = "claude-sonnet-4-20250514"
max_tokens = 8192

[providers.gemini]
api_key = "${GOOGLE_API_KEY}"
model = "gemini-2.0-flash"

[providers.openclaw]
binary = "openclaw"
# args passed to openclaw CLI
args = ["--auto"]

[providers.cursor]
# Cursor agent mode
mode = "agent"
```

Per-agent provider selection:

```bash
# Create agent with specific provider
bc agent create ui-01 --role engineer --provider claude
bc agent create ops-01 --role devops --provider gemini
bc agent create auto-01 --role engineer --provider openclaw

# Default provider in config
[workspace]
default_provider = "claude"
```

### Cost Tracking

Unified cost tracking across providers:

```go
// pkg/cost/cost.go

type Usage struct {
    Provider    string
    Model       string
    InputTokens int
    OutputTokens int
    Cost        float64 // USD
}

// Provider-specific cost calculation
var CostPerToken = map[string]map[string]float64{
    "claude": {
        "claude-sonnet-4-20250514": 0.003 / 1000, // $3/M input
    },
    "gemini": {
        "gemini-2.0-flash": 0.075 / 1000000, // $0.075/M
    },
    "openclaw": {
        "default": 0.0, // OpenClaw may have own billing
    },
}
```

### TUI Integration

Provider indicator in AgentsView:

```
┌─ Agents ────────────────────────────────────────────────────┐
│                                                             │
│ NAME       ROLE      STATE     PROVIDER   TASK             │
│ ────────── ───────── ───────── ────────── ─────────────────│
│ ▸ eng-01   engineer  working   claude     Fix auth bug     │
│   ops-01   devops    idle      gemini     -                │
│   auto-01  engineer  working   openclaw   Refactor tests   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Provider Implementations

### Claude Provider (Existing)

Current Claude Code implementation, refactored to Provider interface:
- Runs in tmux session
- Uses `claude` CLI binary
- Full tool support
- Cost tracking via API

### OpenClaw Provider (New)

```go
type OpenClawProvider struct {
    binary string
    CLIProvider
}

func NewOpenClawProvider() *OpenClawProvider {
    return &OpenClawProvider{
        binary: "openclaw",
        CLIProvider: CLIProvider{
            name: "openclaw",
        },
    }
}

func (p *OpenClawProvider) Capabilities() Capabilities {
    return Capabilities{
        Streaming:    true,
        Tools:        []string{"bash", "file", "browser"},
        CostTracking: false, // OpenClaw has own billing
        Models:       []string{"default"},
    }
}
```

### Gemini Provider (New)

```go
type GeminiProvider struct {
    APIProvider
}

func NewGeminiProvider(apiKey string) *GeminiProvider {
    return &GeminiProvider{
        APIProvider: APIProvider{
            name:     "gemini",
            endpoint: "https://generativelanguage.googleapis.com/v1beta",
            apiKey:   apiKey,
            model:    "gemini-2.0-flash",
        },
    }
}

func (p *GeminiProvider) Capabilities() Capabilities {
    return Capabilities{
        Streaming:    true,
        MaxContext:   1000000,
        CostTracking: true,
        Models:       []string{"gemini-2.0-flash", "gemini-2.0-pro"},
    }
}
```

## Implementation Plan

### Phase 1: Provider Interface (2-3 PRs)

1. Define Provider interface in `pkg/agent/provider/`
2. Refactor existing Claude implementation
3. Create provider registry
4. Update agent commands to use providers

### Phase 2: CLI Providers (2-3 PRs)

5. Implement CLIProvider base
6. OpenClaw provider
7. OpenCode provider (optional)

### Phase 3: API Providers (2-3 PRs)

8. Implement APIProvider base
9. Gemini provider
10. Cost tracking integration

### Phase 4: Polish (1-2 PRs)

11. TUI provider column
12. Provider health checks
13. Documentation

## Alternatives Considered

### Alternative 1: Plugins for Providers

Use RFC 001 plugin system for providers.

**Rejected:** Providers are core functionality, need tight integration.

### Alternative 2: Single Protocol (MCP)

Use Model Context Protocol for all providers.

**Deferred:** Not all providers support MCP yet. Consider for v2.

### Alternative 3: HTTP Proxy

Proxy all providers through HTTP API.

**Rejected:** Adds latency, complexity. Direct integration better.

## Success Metrics

- Support for 3+ providers at launch
- < 100ms overhead for provider abstraction
- Provider switch does not affect agent state
- Cost tracking accurate within 5%

## Open Questions

1. **Provider failover?** - Auto-switch if provider unavailable?
2. **Provider-specific features?** - How to expose unique capabilities?
3. **Rate limiting?** - Coordinate across multiple API providers?
4. **Credential management?** - Secure storage for multiple API keys?

## References

- [LangChain Provider Pattern](https://python.langchain.com/docs/integrations/llms/)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [OpenRouter API](https://openrouter.ai/docs) - Multi-model routing
