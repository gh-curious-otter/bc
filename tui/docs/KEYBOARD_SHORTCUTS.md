# TUI Keyboard Shortcuts Audit

Issue #1130 - Standardize keyboard shortcuts across views

## Current State Summary

### Universal Shortcuts (Consistent)

| Key | Action | Views |
|-----|--------|-------|
| `j` / `↓` | Navigate down | All list views |
| `k` / `↑` | Navigate up | All list views |
| `g` | Go to first item | Most list views |
| `G` | Go to last item | Most list views |
| `q` | Quit/exit view | All views |
| `ESC` | Back/close/cancel | All views |
| `Enter` | Select/open item | Most views |
| `r` | Refresh data | Most views |

### View-Specific Shortcuts

#### Dashboard
| Key | Action |
|-----|--------|
| `a` | Go to Agents view |
| `c` | Go to Channels view |
| `$` | Go to Costs view |
| `Ctrl+P` | Toggle performance panel |

#### LogsView
| Key | Action |
|-----|--------|
| `/` | Search |
| `s` | Cycle severity filter |
| `a` | Cycle agent filter |
| `t` | Cycle time filter |
| `c` | Clear all filters |

#### AgentsView
| Key | Action |
|-----|--------|
| `a` | Attach to agent |
| `x` | Stop agent (with confirm) |
| `X` | Force stop agent |
| `R` | Restart agent |

#### RolesView
| Key | Action |
|-----|--------|
| `/` | Search roles |
| `d` | Delete role (with confirm) |

#### CommandsView
| Key | Action |
|-----|--------|
| `/` | Search commands |
| `Tab` | Next section |
| `f` | Toggle favorite |
| `c` | Clear output |

#### ProcessesView
| Key | Action |
|-----|--------|
| `l` | View logs |
| `r` | Restart process |

#### WorktreesView
| Key | Action |
|-----|--------|
| `p` | Prune orphaned worktrees |
| `o` | Toggle orphaned-only filter |

#### TeamsView
| Key | Action |
|-----|--------|
| `Space` | Expand/collapse team |

#### CostDashboard
| Key | Action |
|-----|--------|
| `1/2/3` | Switch tabs (agent/model/team) |
| `b` | Set budget |
| `e` | Edit budget |

#### ActivityView
| Key | Action |
|-----|--------|
| `d` | Day view |
| `w` | Week view |
| `m` | Month view |

#### AgentDetailView
| Key | Action |
|-----|--------|
| `i` / `m` | Send message |
| `1/2/3` | Switch tabs |
| `Tab` | Next tab |

#### WorkspaceSelectorView
| Key | Action |
|-----|--------|
| `v` | View workspace details |

## Inconsistencies Found

### 1. Search (`/`)
- **Has search**: LogsView, CommandsView, RolesView
- **Missing search**: AgentsView, TeamsView, ProcessesView, DemonsView
- **Recommendation**: Add `/` search to all list views

### 2. Delete/Stop Actions
- AgentsView uses `x` for stop
- RolesView uses `d` for delete
- **Recommendation**: Standardize to `d` for delete/stop, `x` for force

### 3. Tab Navigation
- CostDashboard, AgentDetailView use `1/2/3`
- CommandsView uses `Tab`
- **Recommendation**: Support both - `1/2/3` for direct access, `Tab` for cycling

### 4. Refresh (`r`)
- Used consistently in most views
- **Status**: Good - keep as is

### 5. Help (`?`)
- Not implemented in any view
- **Recommendation**: Add `?` to show keyboard shortcut overlay

## Proposed Unified Scheme

### Tier 1: Universal (All Views)
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate |
| `g/G` | First/last |
| `Enter` | Select |
| `ESC` | Back |
| `q` | Quit |
| `r` | Refresh |
| `?` | Show help |

### Tier 2: List Views
| Key | Action |
|-----|--------|
| `/` | Search/filter |
| `c` | Clear filters |

### Tier 3: Item Actions
| Key | Action |
|-----|--------|
| `d` | Delete (with confirm) |
| `e` | Edit |
| `x` | Force action (no confirm) |

### Tier 4: Tab Views
| Key | Action |
|-----|--------|
| `1-9` | Jump to tab N |
| `Tab` | Next tab |
| `Shift+Tab` | Previous tab |

## Implementation Priority

1. **P1**: Add `?` help overlay to all views
2. **P2**: Add `/` search to missing list views
3. **P3**: Standardize delete/stop to `d`
4. **P4**: Add Tab cycling to tab views

## Files to Modify

- `src/views/AgentsView.tsx` - Add `/` search, change `x` to `d`
- `src/views/TeamsView.tsx` - Add `/` search
- `src/views/ProcessesView.tsx` - Add `/` search
- `src/views/DemonsView.tsx` - Add `/` search
- `src/components/HelpOverlay.tsx` - New component for `?`
- All views - Add `?` handler
