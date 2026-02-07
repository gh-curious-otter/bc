# bc Documentation

Technical documentation for **bc** (beads coordinator), a multi-agent orchestration system for Claude Code.

---

## Purpose

This directory contains architecture documentation, design patterns, and reference materials for bc. The system coordinates multiple Claude Code agents working on different tasks with predictable behavior and cost awareness.

---

## Document Index

| Document | Description |
|----------|-------------|
| [01-architecture-overview.md](01-architecture-overview.md) | Core concepts: workspace, agents, work queue, worktrees. Hierarchical roles (Root → PM/Manager → Engineer/QA), tmux-based agent isolation, git-backed state persistence. |
| [02-agent-types.md](02-agent-types.md) | Agent roles and capabilities: Root (singleton), ProductManager, Manager, TechLead, Engineer, QA. Role hierarchy and state machine. |
| [03-cli-reference.md](03-cli-reference.md) | Complete `bc` CLI reference: agent lifecycle (spawn, status, up, down), work queue (queue add/list), communication (send, channel), and diagnostics. |
| [04-data-models.md](04-data-models.md) | Data structures: .bc/ directory layout, config.toml, per-agent state files, events.jsonl, channels. |
| [05-workflows.md](05-workflows.md) | Operational workflows: work assignment, agent lifecycle, merge queue processing, parent-child agent relationships. |
| [06-gtn-tui.md](06-gtn-tui.md) | Reference: gtn TUI patterns (Bubble Tea). Useful for bc TUI development. |
| [07-design-lessons.md](07-design-lessons.md) | Key lessons from Gas Town development applied to bc's simpler design. |
| [08-tui-builder.md](08-tui-builder.md) | Declarative TUI builder pattern for AI-generated terminal interfaces. |
| [cursor-agent-cli.md](cursor-agent-cli.md) | Using Cursor Agent CLI as an alternative to Claude Code in bc. |
| [bc-v1-audit.md](bc-v1-audit.md) | V1 feature audit and status (Lint-Zero complete). |
| [bc-v2-vision-qa.md](bc-v2-vision-qa.md) | V2 vision Q&A clarifications from product owner. |
| [epic-1.1-design.md](epic-1.1-design.md) | Epic 1.1: Workspace Restructure technical design (TOML, roles, per-agent state). |

---

## Quick Reference

### Key Concepts

bc is a multi-agent orchestration system that coordinates Claude Code agents through:
- **Git-backed persistence** - State survives restarts via `.bc/` directory
- **Tmux-based isolation** - Each agent runs in its own tmux session
- **Per-agent worktrees** - Each agent gets its own git worktree to avoid conflicts
- **Hierarchical roles** - PM → Manager → Engineer/QA role hierarchy with capabilities

### Agent Roles

| Role | Level | Capabilities | Can Create |
|------|-------|--------------|------------|
| Root | 0 | create_agents, assign_work, review_work, merge_to_main | PM, Manager, TechLead |
| ProductManager | 1 | create_agents, assign_work, create_epics, review_work | Manager |
| Manager | 1 | create_agents, assign_work, review_work | Engineer, QA |
| TechLead | 1 | review_work, assign_work | (none) |
| Engineer | 2 | implement_tasks | (none) |
| QA | 2 | test_work, review_work | (none) |

### Agent States

| State | Description |
|-------|-------------|
| `idle` | Ready for work |
| `starting` | Session initializing |
| `working` | Actively executing task |
| `done` | Task completed |
| `stuck` | Needs assistance |
| `error` | Encountered error |
| `stopped` | Session terminated |

### Essential Commands

```bash
# Start/stop services
bc up                          # Start coordinator agent
bc down                        # Stop all agents

# Spawn agents
bc spawn pm-01 --role product-manager   # Spawn PM
bc spawn eng-01 --role engineer         # Spawn engineer

# Work queue
bc queue add "Fix login bug"            # Add work item
bc queue list                           # List work items

# Agent interaction
bc send <agent> "message"               # Send message to agent
bc attach <agent>                       # Attach to agent tmux session

# Monitoring
bc status                               # Agent overview
bc stats                                # Workspace statistics
bc logs                                 # View event log
```

### Work Item States

| Status | Description |
|--------|-------------|
| `pending` | Available for assignment |
| `assigned` | Claimed by agent |
| `working` | Being executed |
| `done` | Completed successfully |
| `failed` | Execution failed |

### Critical Data Structures

| Structure | Location | Purpose |
|-----------|----------|---------|
| `agents.json` | `.bc/agents/` | Agent state persistence |
| `queue.json` | `.bc/queue.json` | Work queue items |
| `events.jsonl` | `.bc/events.jsonl` | Append-only event log |
| `channels.json` | `.bc/channels.json` | Communication channels |
| `config.json` | `.bc/config.json` | Workspace configuration |

### Directory Structure

```
.bc/                           # bc workspace root
├── agents/                    # Agent state files
│   └── agents.json            # All agent states
├── bin/                       # Wrapper scripts (git)
├── logs/                      # Agent logs
├── worktrees/                 # Per-agent git worktrees
│   ├── pm-01/                 # PM worktree
│   └── eng-01/                # Engineer worktree
├── config.json                # Workspace config
├── queue.json                 # Work queue
├── channels.json              # Communication channels
└── events.jsonl               # Event log
```

---

## Design Principles

1. **Simplicity** - Two-tier hierarchy (coordinator + workers) vs complex Gas Town hierarchy
2. **Predictability** - Constrained agent actions, explicit capabilities per role
3. **Cost awareness** - Budget tracking (planned) and on-demand execution
4. **Git-native** - All state in git-tracked files for crash recovery
5. **Tmux isolation** - Each agent in isolated tmux session
