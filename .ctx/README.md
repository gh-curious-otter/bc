# bc Context Documentation

Context documentation for building **bc**, a new agent orchestrator inspired by Gas Town.

---

## Purpose

This directory contains architectural documentation, design patterns, and reference materials extracted from the Gas Town (gt) multi-agent orchestration system. These documents serve as the foundation for building bc - a streamlined agent orchestrator that incorporates lessons learned from Gas Town's development.

---

## Document Index

| Document | Description |
|----------|-------------|
| [01-architecture-overview.md](01-architecture-overview.md) | Core concepts: Town, Rig, Mayor, Deacon, Witness, Refinery, Polecats, Crew. Hierarchical organization, data flow, and key design principles including ZFC and GUPP. |
| [02-agent-types.md](02-agent-types.md) | Detailed documentation of all six agent types: Mayor (coordinator), Deacon (daemon), Witness (monitor), Refinery (merge queue), Polecats (ephemeral workers), and Crew (human workers). |
| [03-cli-reference.md](03-cli-reference.md) | Complete `gt` CLI reference: work management (sling, convoy, done, hook), agent operations, communication (mail, nudge, broadcast, handoff), services, workspace, and diagnostics. |
| [04-data-models.md](04-data-models.md) | Data structures: configuration files (town.json, rigs.json), runtime state, event log format, Beads integration (issues, MRs, agents), and mail system. |
| [05-workflows.md](05-workflows.md) | Operational workflows: work assignment flow, polecat lifecycle, merge queue processing, communication patterns, and manual intervention procedures. |
| [06-gtn-tui.md](06-gtn-tui.md) | gtn TUI documentation: Bubble Tea architecture, views (status, sessions, convoys, MQ, costs), components (table, modal, tree), keyboard shortcuts, and gt client wrapper. |
| 07-design-lessons.md | *(Planned)* Key lessons and design decisions from Gas Town development for bc. |

---

## Quick Reference

### Key Concepts

Gas Town is a multi-agent orchestration system that solves agent coordination through **git-backed persistence** (work survives restarts), **tmux-based liveness** (ZFC - session existence = agent alive), and **hook-based work assignment** (GUPP - if work is on your hook, you run it). The hierarchy flows from Town (workspace) to Rigs (projects) to Agents (Mayor coordinates, Polecats execute, Witness monitors, Refinery merges).

### Essential Commands

```bash
# Start/stop services
gt up                          # Start all services
gt down                        # Stop all services

# Work assignment
gt sling <bead-id> <rig>       # Assign work to polecat
gt convoy create "Name" <ids>  # Bundle work items
gt done                        # Complete work and submit to MQ

# Agent interaction
gt prime                       # Load context for current role
gt hook                        # Check current work assignment
gt nudge <target> <message>    # Send message to agent
gt mail inbox                  # Check mail

# Monitoring
gt status                      # Town overview
gt agents                      # List active agents
gt mq list                     # View merge queue
```

### Critical Data Structures

| Structure | Location | Purpose |
|-----------|----------|---------|
| `town.json` | `mayor/town.json` | Town configuration and identity |
| `rigs.json` | `mayor/rigs.json` | Registry of all project rigs |
| `issues.jsonl` | `.beads/issues.jsonl` | Beads issue tracking database |
| `routes.jsonl` | `.beads/routes.jsonl` | Issue prefix to rig routing |
| `.events.jsonl` | Town root | Append-only event log |

---

## Source Materials

| Source | Location | Description |
|--------|----------|-------------|
| **Gas Town repo** | [github.com/steveyegge/gastown](https://github.com/steveyegge/gastown) | Original gt orchestrator codebase |
| **gtn TUI** | `~/Projects/gtn/gtn` | Terminal UI for Gas Town (Bubble Tea) |
| **Beads** | `bd` command | Git-backed issue tracking CLI |

---

## Next Steps for bc Development

1. **Define core abstractions** - Identify which Gas Town concepts to keep, simplify, or replace
2. **Design agent model** - Decide on agent types and their responsibilities
3. **Plan persistence layer** - Determine state storage (git-backed, SQLite, or hybrid)
4. **Build CLI foundation** - Set up Cobra CLI structure with core commands
5. **Implement work assignment** - Hook-based work routing with Beads integration
6. **Add TUI** - Consider porting gtn patterns or building fresh with Bubble Tea
7. **Document design decisions** - Capture lessons learned in 07-design-lessons.md
