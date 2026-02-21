# RFC 004: Agent Marketplace

**Issue:** #1406
**Author:** eng-03
**Status:** Draft
**Created:** 2026-02-22

## Summary

Create a marketplace for discovering, installing, and sharing specialized agents with pre-configured roles, tools, and workflows.

## Motivation

- Users need specialized agents for different domains (frontend, DevOps, security)
- Community can share proven agent configurations
- Reduces setup time for common workflows
- Enables monetization for professional agents
- Builds ecosystem around bc

## Design Principles

1. **Curated Quality** - Marketplace agents are reviewed and rated
2. **Security First** - Sandboxed by default, clear permission model
3. **Simple Distribution** - One-command install, no complex dependencies
4. **Composable** - Agents can be extended and customized locally

## Scope

### In Scope (MVP)

| Feature | Description |
|---------|-------------|
| Agent Packages | Bundled role + prompts + tools |
| Browse/Search | CLI and TUI marketplace browser |
| Install/Update | `bc market install <agent>` |
| Ratings | Community star ratings |
| Categories | frontend, backend, devops, security, etc. |

### Out of Scope (MVP)

| Feature | Rationale |
|---------|-----------|
| Paid agents | Focus on free ecosystem first |
| Web UI | CLI/TUI sufficient for MVP |
| Agent builder | Use existing role/prompt system |
| Teams/orgs | Future iteration |

## Technical Design

### Agent Package Format

Agents are distributed as directories with `agent.toml` manifest:

```toml
[agent]
name = "frontend-react"
version = "1.2.0"
description = "React/TypeScript specialist with testing expertise"
author = "bcommunity"
license = "MIT"
repository = "https://github.com/bcommunity/agent-frontend-react"

[agent.bc]
min_version = "0.3.0"

[agent.metadata]
categories = ["frontend", "react", "typescript"]
tags = ["testing", "jest", "rtl"]
stars = 4.5
installs = 1250

[role]
base = "engineer"
name = "frontend-react"
description = "React/TypeScript frontend engineer"

[capabilities]
# Inherits from engineer, adds these
tools = ["npm", "vite", "jest", "playwright"]
languages = ["typescript", "javascript", "css"]

[prompts]
# Custom prompt sections
system = "prompts/system.md"
task_format = "prompts/task.md"
code_review = "prompts/review.md"

[tools]
# Pre-configured tools
[tools.npm]
allowed_commands = ["install", "run", "test", "build"]

[tools.vite]
allowed_commands = ["dev", "build", "preview"]
```

### Package Structure

```
~/.bc/marketplace/
└── frontend-react/
    ├── agent.toml
    ├── prompts/
    │   ├── system.md
    │   ├── task.md
    │   └── review.md
    ├── tools/
    │   └── lint-fix.sh
    ├── examples/
    │   └── sample-task.md
    └── README.md
```

### CLI Commands

```bash
# Browse marketplace
bc market browse                    # Interactive TUI browser
bc market search "react"            # Search by keyword
bc market list --category=frontend  # Filter by category

# Agent management
bc market install frontend-react    # Install from marketplace
bc market install ./my-agent        # Install local package
bc market update frontend-react     # Update to latest
bc market remove frontend-react     # Uninstall

# Use marketplace agent
bc agent create my-ui --role=frontend-react  # Create with marketplace role
bc agent start my-ui                          # Start normally

# Publish (for creators)
bc market publish ./my-agent        # Submit for review
bc market versions frontend-react   # List available versions
```

### TUI Integration

Add "Marketplace" section to drawer:

```
WORKSPACE
● Dashboard
○ Agents
○ Channels
○ Costs

MARKETPLACE
○ Browse
○ Installed
○ Updates (2)
```

### Marketplace Browser View

```
┌─ Marketplace ───────────────────────────────────────────────┐
│ [/] Search: react_                                          │
│                                                             │
│ TOP AGENTS                                                  │
│ ┌─────────────────────────────────────────────────────────┐│
│ │ frontend-react              ★★★★☆ 4.5  ↓1.2k           ││
│ │ React/TypeScript specialist                             ││
│ │ @bcommunity · frontend, react                          ││
│ ├─────────────────────────────────────────────────────────┤│
│ │ devops-k8s                  ★★★★★ 4.8  ↓890            ││
│ │ Kubernetes & Helm deployment expert                     ││
│ │ @cloudteam · devops, kubernetes                        ││
│ ├─────────────────────────────────────────────────────────┤│
│ │ security-audit              ★★★★☆ 4.3  ↓560            ││
│ │ Security vulnerability scanner                          ││
│ │ @secteam · security, audit                             ││
│ └─────────────────────────────────────────────────────────┘│
│                                                             │
│ j/k: nav | Enter: details | i: install | /: search         │
└─────────────────────────────────────────────────────────────┘
```

### Agent Detail View

```
┌─ frontend-react ────────────────────────────────────────────┐
│ React/TypeScript specialist with testing expertise          │
│ ★★★★☆ 4.5 (128 ratings) · 1,250 installs                   │
│                                                             │
│ @bcommunity · MIT License                                   │
│ https://github.com/bcommunity/agent-frontend-react          │
│                                                             │
│ DESCRIPTION                                                 │
│ Expert React engineer with deep TypeScript knowledge.       │
│ Specializes in component architecture, testing with         │
│ Jest/RTL, and modern build tooling.                         │
│                                                             │
│ CAPABILITIES                                                │
│ • Languages: TypeScript, JavaScript, CSS                    │
│ • Tools: npm, vite, jest, playwright                        │
│ • Frameworks: React, Next.js, TailwindCSS                   │
│                                                             │
│ VERSIONS                                                    │
│ v1.2.0 (current) · v1.1.0 · v1.0.0                         │
│                                                             │
│ [i] Install   [r] Ratings   [g] GitHub   [q] Back           │
└─────────────────────────────────────────────────────────────┘
```

## Registry Backend

### Architecture

```
┌─────────────────────────────────────────┐
│ bc CLI                                   │
├─────────────────────────────────────────┤
│ marketplace/                             │
│   client.go     # API client            │
│   cache.go      # Local caching         │
│   install.go    # Package installation  │
│   validate.go   # Package validation    │
└─────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│ Registry API (marketplace.bc.dev)       │
├─────────────────────────────────────────┤
│ GET  /agents                            │
│ GET  /agents/:name                      │
│ GET  /agents/:name/versions             │
│ GET  /agents/:name/:version/download    │
│ POST /agents (publish)                  │
│ POST /agents/:name/rate                 │
└─────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│ Storage                                  │
│ • GitHub Releases (packages)            │
│ • SQLite/Postgres (metadata, ratings)   │
│ • CDN (fast downloads)                  │
└─────────────────────────────────────────┘
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/agents` | GET | List/search agents |
| `/agents/:name` | GET | Get agent details |
| `/agents/:name/versions` | GET | List versions |
| `/agents/:name/:ver/download` | GET | Download package |
| `/agents` | POST | Publish new agent |
| `/agents/:name/rate` | POST | Submit rating |

## Security Model

### Package Validation

1. **Manifest Schema** - `agent.toml` must pass validation
2. **No Binary** - Only scripts, prompts, and configs allowed
3. **Checksum** - SHA256 verified on install
4. **Signature** - Optional GPG signature for verified agents

### Trust Levels

| Level | Indicator | Requirements |
|-------|-----------|--------------|
| Verified | ✓ | Author identity confirmed, code reviewed |
| Community | ○ | Published, basic validation passed |
| Unverified | ⚠ | Local or unofficial source |

### Permission Inheritance

Marketplace agents inherit permissions from their base role:
- `engineer` base → standard engineer capabilities
- Custom tools require explicit `--allow-tools` on install

## Implementation Plan

### Phase 1: Local Packages (2-3 PRs)

1. `agent.toml` manifest parser
2. Package loader from local directory
3. `bc market install ./path` command
4. Integrate with `bc agent create --role`

### Phase 2: Remote Registry (3-4 PRs)

5. Registry API client
6. `bc market browse/search` commands
7. Package download and caching
8. `bc market update` command

### Phase 3: TUI & Publishing (2-3 PRs)

9. MarketplaceView in TUI
10. `bc market publish` command
11. Rating system

### Phase 4: Ecosystem (Future)

12. Verified author program
13. Trending/featured agents
14. Collections (curated sets)

## Alternatives Considered

### Alternative 1: GitHub-Only Distribution

Distribute agents via GitHub repositories without central registry.

**Rejected:** Poor discoverability, no ratings, inconsistent quality.

### Alternative 2: npm/Homebrew Model

Use existing package managers.

**Rejected:** bc-specific metadata needed, different lifecycle.

### Alternative 3: Built-In Agents Only

Ship all agents with bc.

**Rejected:** Limits extensibility, slower iteration, larger binary.

## Success Metrics

- 20+ community agents within 6 months
- Average install time < 5 seconds
- User satisfaction rating > 4.0
- Zero security incidents from marketplace agents

## Open Questions

1. **Monetization?** - Should we support paid agents eventually?
2. **Namespacing?** - `@author/agent-name` or flat namespace?
3. **Deprecation policy?** - How to handle unmaintained agents?
4. **Offline support?** - Cache vs always-online?

## References

- [npm Registry](https://docs.npmjs.com/cli/v8/using-npm/registry)
- [VS Code Extension Marketplace](https://code.visualstudio.com/api/working-with-extensions/publishing-extension)
- [Terraform Registry](https://registry.terraform.io/)
- RFC 001: Plugin Ecosystem (related, agents use plugins)
