# Competitor TUI Research: OpenCode, Cursor, and Industry Patterns

**Research by:** eng-02
**Date:** 2026-02-22
**Purpose:** Document TUI patterns from competitors to inform bc UI polish efforts

## Executive Summary

OpenCode and Cursor represent the current state-of-the-art in AI-powered terminal interfaces. This document captures key patterns we should consider adopting for bc's TUI improvements.

## OpenCode TUI Patterns

### Framework
- Built with **Bubble Tea** (Go) for terminal UI
- Uses **OpenTUI** (Zig-based native core with TypeScript bindings) for advanced rendering
- SQLite for persistent storage (sessions, conversations)

### Keybinding System
OpenCode uses `Ctrl+X` as a **leader key** with single-character suffixes:
| Keybinding | Action |
|------------|--------|
| `Ctrl+X c` | Compact/summarize |
| `Ctrl+X e` | Editor access |
| `Ctrl+X h` | Help |
| `Ctrl+X m` | Models list |
| `Ctrl+X n` | New session |
| `Ctrl+X q` | Quit |
| `Ctrl+X s` | Share |
| `Ctrl+X t` | Themes |
| `Ctrl+X x` | Export |

**Recommendation:** Consider adopting leader key pattern for less-common actions to reduce keybinding conflicts.

### Special Prefixes
- `@` prefix for file references (fuzzy search)
- `!` prefix for shell command execution

### Scrolling Configuration
- `scroll_speed`: Controls scroll rate (min: 1, default: 3)
- `scroll_acceleration`: macOS-style smooth scrolling that increases speed with rapid gestures

**Recommendation:** Add configurable scroll speed to bc TUI config.

### Session Management
- `/sessions` command for session listing
- Git-based undo/redo (requires project to be a repo)
- Persistent session storage

### Theme System
- Adaptive theming (Dark, Light, Auto)
- localStorage persistence
- Cross-tab synchronization

## Cursor Patterns

### VS Code Compatibility
- 100% VS Code extension/keybinding compatibility
- Zero-friction migration path

### Key Interaction Modes
| Mode | Keybinding | Purpose |
|------|------------|---------|
| Composer | `Cmd+I` / `Ctrl+I` | Multi-file editing |
| Agent | `Cmd+L` | Full terminal/browser access |
| Inline | `Cmd+K` | Quick agent interface |

### Rule-Based Configuration
- `.cursor/rules/` folder for modular project rules
- Rules auto-applied to all AI interactions
- Team-wide rule sharing

**Recommendation:** Consider similar rule folder structure for bc workspace roles.

### Conditional Keybindings (via Atuin)
Modern TUI pattern: Conditional keybinding execution with boolean expressions:
```
[keymap]
condition = "mode == 'search' && input.length > 0"
```

**Recommendation:** Consider state-dependent keybindings for bc views.

## Industry Trends (2026)

### "Agentic Engineering" Pattern
- Engineers orchestrate agents, not write code
- Focus on rules, plans, commands, hooks
- Predictable AI behavior through configuration

### Configuration Modularity
- Single config file insufficient for complex projects
- Folder-based modular configuration
- Project-level customization

## Patterns to Adopt for bc

### High Priority
1. **Leader Key System** - Use `Ctrl+X` or similar for advanced commands
   - **Status:** Config schema added in `config.toml` under `[tui]` section
   - **Next:** Implement TUI hook for leader key handling
2. **Scroll Configuration** - Add scroll_speed to TUI config
   - **Status:** Config schema added (`scroll_speed`, `scroll_acceleration`)
   - **Next:** Wire to TUI scrollable components
3. **Theme Persistence** - Ensure theme selection persists across sessions
   - **Status:** Config schema added (`theme` field)
   - **Next:** Wire to ThemeContext persistence
4. **Conditional Keybindings** - View-specific key behavior
   - **Status:** Existing via `useInput` hook per-view

### Medium Priority
1. **Special Prefixes** - Consider `@` for file refs, `!` for commands
2. **Session Management UI** - Session list with restore capability
3. **Rule Folder Pattern** - `.bc/rules/` for modular role configuration

### Nice to Have
1. **Scroll Acceleration** - macOS-style smooth scrolling
2. **Export/Share Commands** - Session export functionality
3. **Cross-tab Sync** - Theme/state sync for multiple TUI instances

## References

- [OpenCode GitHub](https://github.com/opencode-ai/opencode)
- [OpenCode TUI Docs](https://opencode.ai/docs/tui/)
- [OpenCode CLI Docs](https://opencode.ai/docs/cli/)
- [Cursor AI Tips](https://github.com/murataslan1/cursor-ai-tips)
- [Cursor IDE 2026 Guide](https://techjacksolutions.com/ai/ai-development/cursor-ide-what-it-is/)
- [Atuin TUI Keybindings](https://blog.atuin.sh/custom-keybindings-for-the-atuin-tui/)
