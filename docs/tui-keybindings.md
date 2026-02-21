# TUI Keybindings

This document describes the keyboard shortcuts available in the bc TUI and how to customize them.

## Basic Navigation

### Global Keys (work in all views)
| Key | Action |
|-----|--------|
| `1-8` | Switch to tab 1-8 |
| `?` | Toggle help view |
| `Tab` | Move focus to next panel |
| `Shift+Tab` | Move focus to previous panel |
| `Esc` | Close modal / Go back |

### List Navigation
| Key | Action |
|-----|--------|
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `g` | Jump to first item |
| `G` | Jump to last item |
| `Enter` | Select / Open details |
| `/` | Search / Filter |

### Scrolling
| Key | Action |
|-----|--------|
| `Ctrl+D` | Scroll down half page |
| `Ctrl+U` | Scroll up half page |
| `Ctrl+F` | Scroll down full page |
| `Ctrl+B` | Scroll up full page |

## Leader Key System

The leader key system allows access to advanced commands without conflicting with view-specific keybindings. By default, `Ctrl+X` is the leader key.

### How It Works
1. Press the leader key (`Ctrl+X`)
2. A small indicator appears showing "Leader..."
3. Press the command key within 1 second
4. The action executes

### Default Leader Key Bindings
| Sequence | Action |
|----------|--------|
| `Ctrl+X h` | Show help |
| `Ctrl+X t` | Open theme selector |
| `Ctrl+X s` | Show session list |
| `Ctrl+X x` | Export current session |
| `Ctrl+X n` | Create new session |
| `Ctrl+X q` | Quit application |

### Configuration

Leader key bindings can be customized in your workspace config:

```toml
# .bc/config.toml

[tui]
# Change the leader key (default: ctrl+x)
leader_key = "ctrl+x"

# Customize leader bindings
[tui.leader_bindings]
h = "help"
t = "themes"
s = "sessions"
x = "export"
n = "new_session"
q = "quit"
```

### Available Actions
| Action | Description |
|--------|-------------|
| `help` | Show help overlay |
| `themes` | Open theme selector |
| `sessions` | Show session management |
| `export` | Export current session to file |
| `new_session` | Start a new session |
| `quit` | Exit the application |

## View-Specific Keys

### Dashboard (1)
| Key | Action |
|-----|--------|
| `r` | Refresh all data |
| `a` | Jump to agents view |
| `c` | Jump to channels view |

### Agents (2)
| Key | Action |
|-----|--------|
| `Enter` | View agent details |
| `s` | Stop selected agent |
| `r` | Restart selected agent |
| `l` | View agent logs |
| `a` | Attach to agent tmux |

### Channels (3)
| Key | Action |
|-----|--------|
| `Enter` | Open channel history |
| `n` | Create new channel |
| `j` | Join channel |
| `m` | Send message |

### Files (4)
| Key | Action |
|-----|--------|
| `Enter` | Open file / Enter directory |
| `h` | Go to parent directory |
| `l` | Enter directory |
| `.` | Toggle hidden files |

### Costs (5)
| Key | Action |
|-----|--------|
| `d` | Daily view |
| `w` | Weekly view |
| `m` | Monthly view |
| `a` | By agent view |

### Logs (6)
| Key | Action |
|-----|--------|
| `f` | Toggle follow mode |
| `l` | Change log level filter |
| `c` | Clear filters |
| `/` | Search logs |

## Scroll Speed Configuration

Configure scroll behavior in your workspace config:

```toml
[tui]
# Scroll speed: 1 (slow) to 10 (fast), default: 3
scroll_speed = 3

# macOS-style scroll acceleration
# Scrolling speeds up with rapid gestures
scroll_acceleration = false
```

## Custom Keybindings (Future)

Future versions will support full keybinding customization:

```toml
[tui.keybindings]
# Example: Remap navigation keys
"ctrl+n" = "next_item"
"ctrl+p" = "prev_item"

# Example: Add custom command
"ctrl+shift+r" = "refresh_all"
```

## Tips

1. **Use Vim-style navigation** - `j/k` for up/down is consistent across all list views
2. **Leader key for rare actions** - Actions you don't use often are behind the leader key
3. **Numbers for tabs** - Quick switch with 1-8 keys
4. **Search everywhere** - `/` activates search in any list view
