# RFC 003: Desktop Application

**Issue:** #1404
**Author:** eng-03
**Status:** Draft
**Created:** 2026-02-22

## Summary

Wrap bc's TUI in a desktop application for cross-platform accessibility, while preserving the terminal-first philosophy.

## Motivation

- bc's TUI is already rich and functional
- Desktop wrapper enables non-terminal users
- Competitors (OpenCode, Cursor) have desktop apps
- System integration (notifications, tray, file associations)

## Design Principles

1. **Terminal-First Preserved** - Desktop wraps TUI, doesn't replace it
2. **Thin Wrapper** - Minimal desktop-specific code
3. **Cross-Platform** - Windows, macOS, Linux from single codebase
4. **Optional** - Desktop is convenience, CLI remains primary

## Scope

### In Scope (MVP)

| Feature | Description |
|---------|-------------|
| TUI Embedding | Run bc TUI in embedded terminal |
| Window Management | Resize, minimize, close |
| System Tray | Background operation, quick access |
| Notifications | Agent alerts, task completion |
| Auto-launch | Start on system boot (optional) |

### Out of Scope (MVP)

| Feature | Rationale |
|---------|-----------|
| Native UI rewrite | TUI already works, avoid duplication |
| Code editor | Use external editors, not reinvent |
| File browser | TUI FilesView handles this (#1401) |
| Mobile app | Different platform, separate effort |

## Technical Design

### Architecture

```
┌─────────────────────────────────────────┐
│ Desktop App (Electron/Tauri)            │
├─────────────────────────────────────────┤
│ ┌─────────────────────────────────────┐ │
│ │ Embedded Terminal (xterm.js/PTY)    │ │
│ │ ┌─────────────────────────────────┐ │ │
│ │ │ bc TUI (React/Ink)              │ │ │
│ │ └─────────────────────────────────┘ │ │
│ └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│ System Integration Layer                │
│ - Notifications                         │
│ - Tray icon                            │
│ - Auto-update                          │
└─────────────────────────────────────────┘
```

### Framework Options

| Option | Pros | Cons |
|--------|------|------|
| **Tauri** (Recommended) | Lightweight (Rust), small binary | Less ecosystem than Electron |
| Electron | Mature, rich ecosystem | Large binary (~150MB) |
| Wails | Go-native, small binary | Younger ecosystem |

### Key Components

```
desktop/
├── src-tauri/          # Rust backend
│   ├── main.rs         # App entry
│   ├── tray.rs         # System tray
│   └── notifications.rs
├── src/                # Web frontend
│   ├── Terminal.tsx    # xterm.js wrapper
│   └── App.tsx         # Shell
├── tauri.conf.json     # Config
└── package.json
```

## Implementation Plan

### Phase 1: Basic Wrapper (2-3 PRs)
1. Tauri project setup with embedded terminal
2. Run `bc home` on launch
3. Basic window management

### Phase 2: System Integration (2-3 PRs)
1. System tray with menu
2. Desktop notifications
3. Auto-launch preference

### Phase 3: Polish (1-2 PRs)
1. Auto-update mechanism
2. App icons and branding
3. Installer packages (DMG, MSI, AppImage)

## Open Questions

1. **Tauri vs Electron?** - Tauri recommended for smaller binary
2. **Separate repo or monorepo?** - Suggest `bc-desktop/` in monorepo
3. **Distribution channel?** - GitHub Releases, Homebrew Cask?
4. **Code signing?** - Required for macOS, Windows
5. **Auto-update frequency?** - Match bc releases?

## Alternatives Considered

### Alternative 1: Native UI Rewrite
Build desktop UI from scratch instead of embedding TUI.

**Rejected:** Duplicates effort, TUI already functional, inconsistent UX between CLI and desktop.

### Alternative 2: Web App
Host TUI via web server, access in browser.

**Rejected:** No system integration (tray, notifications), requires server.

### Alternative 3: No Desktop App
Keep bc terminal-only.

**Viable but:** Limits adoption for users preferring GUI.

## Success Metrics

- Desktop app installs and runs on all 3 platforms
- TUI renders correctly in embedded terminal
- System tray shows agent status
- Notifications fire for key events
- Binary size < 30MB (Tauri) or < 150MB (Electron)

## Timeline

| Phase | Estimate | Dependencies |
|-------|----------|--------------|
| Phase 1 | 4-6 PRs | None |
| Phase 2 | 3-4 PRs | Phase 1 |
| Phase 3 | 2-3 PRs | Phase 2 |

Total: 9-13 PRs over multiple sprints.

## References

- [Tauri](https://tauri.app/) - Rust-based desktop framework
- [xterm.js](https://xtermjs.org/) - Terminal emulator for web
- [node-pty](https://github.com/microsoft/node-pty) - PTY bindings
