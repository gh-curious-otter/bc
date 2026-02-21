# RFC 002: File Explorer for TUI

**Issue:** #1401
**Author:** eng-05
**Status:** Draft
**Created:** 2026-02-21

## Summary

Add a file explorer view to the bc TUI that provides visual navigation of agent worktrees and the main workspace, enabling quick file access and navigation.

## Motivation

bc's multi-agent architecture creates multiple git worktrees. Currently:
- Users must navigate worktrees via CLI (`bc worktree list`)
- No visual overview of file changes across agents
- Competitors (OpenCode, Cursor) have integrated file explorers

A file explorer would:
- Visualize agent worktrees side-by-side
- Show file changes and diffs at a glance
- Enable quick navigation without leaving TUI

## Design Principles

1. **Agent-Centric** - Organize by agent worktree, not raw filesystem
2. **Read-First** - Start with navigation; editing is future scope
3. **Performance** - Lazy-load directories, cache file trees
4. **Responsive** - Work at 80x24 minimum terminal size

## Feature Scope

### Phase 1: Basic File Tree (MVP)

| Feature | Description | Priority |
|---------|-------------|----------|
| Worktree Selector | Switch between agent worktrees | P0 |
| Directory Tree | Expandable/collapsible tree view | P0 |
| File Preview | Read-only file content preview | P0 |
| Keyboard Navigation | j/k/Enter/Esc navigation | P0 |
| Git Status Indicators | Modified/added/deleted markers | P1 |

### Phase 2: Enhanced Navigation

| Feature | Description | Priority |
|---------|-------------|----------|
| File Search | Fuzzy search across worktree | P1 |
| Recent Files | Quick access to recently viewed | P2 |
| Bookmarks | Pin frequently used files | P2 |
| Multi-Worktree Diff | Compare files across agents | P2 |

### Phase 3: Editing (Future)

| Feature | Description | Priority |
|---------|-------------|----------|
| Basic Editing | Simple text editing in TUI | P3 |
| Syntax Highlighting | Language-aware highlighting | P3 |
| Agent Handoff | Send file to agent for editing | P3 |

## UI Design

### Layout Options

**Option A: Drawer Integration (Recommended)**
```
┌─────────────────────────────────────────────────────────┐
│  bc Dashboard                                           │
├──────────┬──────────────────────────────────────────────┤
│ Drawer   │  Files (eng-01)                              │
│          │  ┌─────────────────────────────────────────┐ │
│ 1 Dash   │  │ ▸ src/                                  │ │
│ 2 Agents │  │   ├─ components/                        │ │
│ 3 Chans  │  │   │  ├─ Button.tsx                     │ │
│ 4 Files◀ │  │   │  └─ Form.tsx                       │ │
│ 5 Costs  │  │   └─ utils/                            │ │
│          │  │      └─ helpers.ts                     │ │
│          │  │ ▸ tests/                               │ │
│          │  │ ▸ docs/                                │ │
│          │  └─────────────────────────────────────────┘ │
│          │  Preview: src/components/Button.tsx         │
│          │  ┌─────────────────────────────────────────┐ │
│          │  │ import React from 'react';              │ │
│          │  │ export const Button = () => {...}       │ │
│          │  └─────────────────────────────────────────┘ │
├──────────┴──────────────────────────────────────────────┤
│ j/k: nav │ Enter: expand/preview │ Tab: switch pane    │
└─────────────────────────────────────────────────────────┘
```

**Option B: Split View**
```
┌──────────────────────┬──────────────────────────────────┐
│  File Tree           │  Preview                         │
│  ──────────          │  ────────                        │
│  eng-01/             │  // Button.tsx                   │
│  ├─ src/             │  import React from 'react';      │
│  │  ├─ components/   │                                  │
│  │  │  ├─ Button.tsx │  export const Button = () => {   │
│  │  │  └─ Form.tsx   │    return <button>Click</button> │
│  │  └─ utils/        │  };                              │
│  └─ tests/           │                                  │
└──────────────────────┴──────────────────────────────────┘
```

### Worktree Selector

```
┌─────────────────────────────────────┐
│ Worktree: [eng-01 ▼]                │
│  ○ main (workspace root)            │
│  ● eng-01 ✱ 3 modified              │
│  ○ eng-02                           │
│  ○ eng-03 ✱ 1 modified              │
└─────────────────────────────────────┘
```

### Git Status Icons

| Icon | Meaning |
|------|---------|
| `✱` | Modified |
| `+` | Added (new file) |
| `−` | Deleted |
| `?` | Untracked |
| `!` | Conflict |

## Technical Design

### Component Structure

```
tui/src/views/
├── FilesView.tsx          # Main view component
├── components/
│   ├── FileTree.tsx       # Tree navigation
│   ├── FilePreview.tsx    # File content display
│   ├── WorktreeSelector.tsx
│   └── FileSearch.tsx     # Fuzzy search
└── hooks/
    ├── useFileTree.ts     # Directory traversal
    ├── useFileContent.ts  # File reading with caching
    └── useGitStatus.ts    # Git status for worktree
```

### Service Layer

```typescript
// services/files.ts
interface FileService {
  listWorktrees(): Promise<Worktree[]>;
  readDirectory(path: string): Promise<DirectoryEntry[]>;
  readFile(path: string): Promise<string>;
  getGitStatus(worktree: string): Promise<GitStatus>;
}
```

### Performance Considerations

1. **Lazy Loading** - Only load directories when expanded
2. **File Caching** - Cache file contents with TTL
3. **Debounced Navigation** - Prevent excessive file reads
4. **Size Limits** - Don't preview files > 100KB

## Implementation Plan

### Milestone 1: Skeleton (1-2 PRs)
- [ ] Add "Files" option to drawer (view #4)
- [ ] Create FilesView.tsx with placeholder
- [ ] Add useWorktrees hook

### Milestone 2: Tree View (2-3 PRs)
- [ ] Implement FileTree component
- [ ] Add directory expansion/collapse
- [ ] Keyboard navigation (j/k/Enter/Esc)
- [ ] Connect to bc worktree data

### Milestone 3: Preview (1-2 PRs)
- [ ] Implement FilePreview component
- [ ] Add syntax highlighting (basic)
- [ ] Handle large files gracefully

### Milestone 4: Git Status (1-2 PRs)
- [ ] Implement useGitStatus hook
- [ ] Add status indicators to tree
- [ ] Show modified file count in selector

## Open Questions

1. **Editing scope**: Should Phase 3 editing be in-scope or separate RFC?
2. **Cross-worktree diff**: Priority for comparing files across agents?
3. **External editor**: Support opening in $EDITOR?
4. **File operations**: Allow create/rename/delete or read-only?

## Alternatives Considered

### 1. External File Manager Integration
- Pros: No TUI development needed
- Cons: Context switching, loses agent-centric view

### 2. Terminal-in-Terminal (like tmux)
- Pros: Full shell access
- Cons: Complex, duplicates tmux functionality

### 3. Web-Based File Explorer
- Pros: Richer UI possibilities
- Cons: Breaks CLI-first philosophy

## Success Criteria

- [ ] File explorer accessible from TUI drawer
- [ ] Can navigate all agent worktrees
- [ ] File preview works for common file types
- [ ] Keyboard-only navigation is efficient
- [ ] Git status visible at a glance
- [ ] Works at 80x24 terminal size

## References

- Issue: #1401
- Competitive: OpenCode file explorer, VS Code explorer
- Architecture: docs/ARCHITECTURE.md (TUI section)
