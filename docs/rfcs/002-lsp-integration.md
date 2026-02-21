# RFC 002: LSP Integration for Code Intelligence

**Issue:** #1403
**Author:** eng-04
**Status:** Draft
**Created:** 2026-02-22

## Summary

Design LSP (Language Server Protocol) integration for bc to provide real-time code intelligence features like diagnostics, completions, and navigation.

## Motivation

bc agents work with code but lack awareness of code structure. LSP integration would enable:
- Real-time error detection before agents commit broken code
- Code navigation for agents to understand codebase structure
- Intelligent completions when composing commands
- Diagnostics that agents can report and act on

## Design Principles

1. **Agent-Centric** - LSP data serves agents, not just the TUI
2. **Language Agnostic** - Support any language with an LSP server
3. **Non-Blocking** - LSP operations shouldn't slow down bc commands
4. **Optional** - bc works without LSP, enhanced with it

## Architecture

### Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         bc CLI                               │
├──────────────────┬──────────────────┬───────────────────────┤
│   Agent System   │    TUI Views     │    LSP Manager        │
│                  │                  │                        │
│  ┌────────────┐  │  ┌────────────┐  │  ┌─────────────────┐  │
│  │ eng-01     │  │  │ Editor     │◄─┼──│ LSP Client Pool │  │
│  │ eng-02     │  │  │ View       │  │  │                 │  │
│  │ ...        │  │  └────────────┘  │  │ ┌─────────────┐ │  │
│  └────────────┘  │  ┌────────────┐  │  │ │ gopls       │ │  │
│        │         │  │ Diagnostics│◄─┼──│ │ tsserver    │ │  │
│        ▼         │  │ Panel      │  │  │ │ rust-analyzer│ │  │
│  ┌────────────┐  │  └────────────┘  │  │ └─────────────┘ │  │
│  │ bc lsp     │◄─┼──────────────────┼──│                 │  │
│  │ diagnostics│  │                  │  └─────────────────┘  │
│  └────────────┘  │                  │                        │
└──────────────────┴──────────────────┴───────────────────────┘
```

### LSP Client Pool

The LSP Manager maintains a pool of language server connections:

```go
type LSPManager struct {
    servers map[string]*LSPClient  // language -> client
    config  LSPConfig
}

type LSPClient struct {
    Language string
    Command  string      // e.g., "gopls", "tsserver"
    Conn     *jsonrpc2.Conn
    Caps     ServerCapabilities
}
```

### Language Server Configuration

```toml
# config.toml
[lsp]
enabled = true
auto_start = true  # Start servers on workspace init

[lsp.servers.go]
command = "gopls"
args = ["serve"]
filetypes = ["go"]

[lsp.servers.typescript]
command = "typescript-language-server"
args = ["--stdio"]
filetypes = ["ts", "tsx", "js", "jsx"]

[lsp.servers.python]
command = "pylsp"
filetypes = ["py"]

[lsp.servers.rust]
command = "rust-analyzer"
filetypes = ["rs"]
```

## Features

### 1. Diagnostics

Get real-time errors and warnings:

```bash
# Show diagnostics for workspace
bc lsp diagnostics

# Show diagnostics for specific file
bc lsp diagnostics pkg/agent/agent.go

# Watch diagnostics (live updates)
bc lsp diagnostics --watch

# JSON output for agent consumption
bc lsp diagnostics --json
```

Output:
```
pkg/agent/agent.go:45:12 error: undefined: ctx
pkg/agent/agent.go:67:5 warning: unused variable 'tmp'
tui/src/app.tsx:23:8 error: Property 'foo' does not exist
```

### 2. Code Navigation

Navigate codebase structure:

```bash
# Go to definition
bc lsp definition pkg/agent/agent.go:45:12
# Output: pkg/workspace/workspace.go:23:6

# Find references
bc lsp references pkg/agent/Agent
# Output: 15 references found

# Show symbols in file
bc lsp symbols pkg/agent/agent.go
# Output: type Agent, func NewAgent, func (a *Agent) Start, ...
```

### 3. Completions

Get code completions:

```bash
# Get completions at position
bc lsp completions pkg/agent/agent.go:45:12
# Output: ctx, context, contextKey, ...
```

### 4. Agent Integration

Agents can query LSP data:

```bash
# Agent checks code before commit
bc lsp diagnostics --severity error --exit-code
# Exit 0 if no errors, 1 if errors

# Agent memory includes code structure
bc lsp outline pkg/agent/
# Returns hierarchical structure for agent context
```

### 5. TUI Integration

New TUI components:

| Component | Description |
|-----------|-------------|
| DiagnosticsPanel | Shows errors/warnings in sidebar |
| SymbolBrowser | Navigate file symbols |
| HoverCard | Show type info on hover (cursor position) |
| GotoDefinition | Quick navigation with Enter |

## Implementation Plan

### Phase 1: Core Infrastructure
1. LSP client library (jsonrpc2 over stdio)
2. Language server process management
3. Basic textDocument/didOpen, didChange, didClose
4. `bc lsp` command group

### Phase 2: Diagnostics
5. textDocument/publishDiagnostics handling
6. `bc lsp diagnostics` command
7. DiagnosticsPanel in TUI
8. Diagnostics caching and invalidation

### Phase 3: Navigation
9. textDocument/definition
10. textDocument/references
11. textDocument/documentSymbol
12. Symbol browser in TUI

### Phase 4: Agent Integration
13. Diagnostic checks in pre-commit hooks
14. LSP data in agent memory/context
15. Agent commands for code navigation

## Technical Details

### LSP Protocol Subset

Initial implementation covers:

| Method | Direction | Description |
|--------|-----------|-------------|
| initialize | C→S | Initialize server |
| initialized | C→S | Confirm initialization |
| shutdown | C→S | Shutdown server |
| textDocument/didOpen | C→S | File opened |
| textDocument/didChange | C→S | File changed |
| textDocument/didClose | C→S | File closed |
| textDocument/publishDiagnostics | S→C | Diagnostics notification |
| textDocument/definition | C→S | Go to definition |
| textDocument/references | C→S | Find references |
| textDocument/documentSymbol | C→S | Document symbols |

### File Synchronization

Options for keeping LSP servers in sync:

1. **Full sync**: Send entire file on change (simple, works everywhere)
2. **Incremental sync**: Send diffs (efficient, requires server support)
3. **Watch mode**: Let server watch filesystem (gopls preference)

Initial implementation: Full sync with workspace-level file watching.

### Process Management

```go
// Start language server
func (m *LSPManager) StartServer(lang string) error {
    cfg := m.config.Servers[lang]
    cmd := exec.Command(cfg.Command, cfg.Args...)
    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()

    conn := jsonrpc2.NewConn(ctx,
        jsonrpc2.NewBufferedStream(stdout, stdin),
        m.handler)

    m.servers[lang] = &LSPClient{Conn: conn, ...}
    return nil
}
```

## Alternatives Considered

### 1. Direct Parser Integration
- Pros: No external dependencies
- Cons: Reinventing wheel, limited language support

### 2. Tree-sitter
- Pros: Fast, incremental parsing
- Cons: No semantic analysis, different per language

### 3. LSP (Chosen)
- Pros: Standard protocol, rich ecosystem, semantic analysis
- Cons: External process overhead, server management

## Dependencies

### Go Libraries
- `go.lsp.dev/jsonrpc2` - JSON-RPC 2.0 implementation
- `go.lsp.dev/protocol` - LSP protocol types

### Language Servers (User-provided)
- gopls (Go)
- typescript-language-server (TypeScript/JavaScript)
- pylsp (Python)
- rust-analyzer (Rust)
- clangd (C/C++)

## Success Metrics

- LSP servers start within 2 seconds
- Diagnostics update within 500ms of file change
- No noticeable CLI latency when LSP is enabled
- At least 3 language servers tested and documented
- Agent error rate decreases by detecting issues pre-commit

## Open Questions

1. Should LSP run per-worktree or workspace-wide?
2. How to handle language server crashes gracefully?
3. Should we bundle common language servers?
4. How to expose LSP data to plugins?
5. Memory limits for LSP server processes?

## References

- [Language Server Protocol Specification](https://microsoft.github.io/language-server-protocol/)
- [go.lsp.dev](https://go.lsp.dev/) - Go LSP libraries
- [gopls](https://pkg.go.dev/golang.org/x/tools/gopls) - Go language server
- [OpenCode LSP Integration](https://docs.opencode.dev/features/lsp) - Competitor implementation
