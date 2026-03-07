# Figma AI Prompt — bc TUI Redesign

Design a **dark terminal dashboard UI** for "bc" — a mission control app that monitors and commands a fleet of AI coding agents working on a software project. Think **Bloomberg Terminal meets Linear dark mode** — dense, professional, keyboard-driven.

## Specs

- **Canvas**: 1920×1080, dark background `#0D1117`
- **Font**: JetBrains Mono or SF Mono (monospace only)
- **Feel**: Cockpit / air traffic control / submarine sonar. Dense data, surgical color, zero decoration

## Color Palette

- **Backgrounds**: `#07090F` (deepest) → `#0D1117` (base) → `#151B23` (cards/panels) → `#1B2230` (overlays)
- **Borders**: `#1B2230` (ghost, barely visible) · `#2D333B` (subtle) · `#4A9EFF` (focused, pops)
- **Text**: `#F0F3F6` (bright) · `#C9D1D9` (normal) · `#768390` (muted) · `#3D444D` (ghost)
- **Blue** `#4A9EFF` — selection, focus, working state
- **Green** `#3FB950` — success, done, healthy
- **Amber** `#D4A72C` — warning, stuck, caution
- **Red** `#F85149` — error, destructive
- **Violet** `#A371F7` — AI/memory features, badges

## Layout (design all 4 frames)

### Frame 1: Dashboard (main screen)

Top bar: `bc ◆ myproject` logo + tab navigation (Dashboard, Agents, Channels, Costs, Logs, Roles, Memory, Worktrees, Help). Active tab is bright + underlined, rest are muted gray.

Below top bar: 4 metric cards in a row — "12 Total Agents", "5 ◆ Working" (blue), "3 ○ Idle" (gray), "1 ▲ Stuck" (amber). Each card has a braille sparkline `▁▃▅▇█` showing trend. Cards use `#151B23` background.

Main area split 65/35: Left is an "Activity" feed — timestamp, agent name (colored by role: green=engineer, amber=manager, blue=tech-lead), status symbol, and message. Right column has stacked stat panels: Health (92% with mini bar), Cost ($47.23/$500 with progress bar), Roles breakdown.

Bottom status bar: `[:] command  [/] filter  [?] help` on the left, `5 working · $47.23` on the right.

### Frame 2: Agents View

Full-width data table. Columns: STATUS (symbol), NAME, ROLE (colored glyph), STATE (pill badge), COST, UPTIME. Grouped by role with subtle `─── engineers ───` separators. Selected row highlighted in blue with `▸` indicator. Show ~12 agents. Include one "stuck" (amber) and one "error" (red) for contrast.

### Frame 3: Channel Chat View

Breadcrumb: `> Channels › #engineering`. Flat Slack-style messages (NO bubbles/borders). Each message: role glyph + colored sender name + right-aligned timestamp on line 1, indented message body on line 2+. Show 6-8 messages from different roles. Include an `@mention` highlighted in blue. Bottom: message input bar with placeholder text.

### Frame 4: Command Palette Overlay

The Dashboard dimmed to 40% opacity. Centered floating panel (`#1B2230` background, `#2D333B` border) with `: agents█` input at top, 5 fuzzy-matched suggestions below (selected one has `▸` + blue text), and `Tab complete · Esc cancel` hint at bottom.

## Status Symbols (use throughout)

- `◆` working (blue) · `○` idle (gray) · `✓` done (green) · `✗` error (red) · `▲` stuck (amber) · `·` stopped (dim)

## Role Glyphs (colored dots next to agent names)

- `●` engineer (green) · `▲` manager (amber) · `◇` tech-lead (blue) · `■` qa (red) · `◆` root (violet)

## Key Design Rules

1. **No wasted space** — tables and feeds fill all available vertical space
2. **No emoji** — use geometric Unicode glyphs only (◆●▲◇■○✓✗)
3. **Color = meaning** — blue=interactive, green=good, amber=warning, red=bad, violet=AI
4. **Depth via background tiers** — 4 levels of darkness create hierarchy without borders
5. **One top bar + one bottom bar** — no duplicate navigation
6. **Status as pills** — symbol + colored text on muted-color background (e.g., `◆ working` on `#1A3A5C`)
