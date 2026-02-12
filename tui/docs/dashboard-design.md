# Dashboard View Component Design

**Author:** eng-03
**Phase:** 2 Prep
**Status:** Draft

## Overview

The Dashboard view provides a high-level overview of the bc workspace, aggregating data from multiple bc CLI commands into a single view.

## Data Sources

The Dashboard calls these bc CLI commands (all with `--json` flag):

| Command | Data | Refresh Rate |
|---------|------|--------------|
| `bc status --json` | Workspace summary, agent counts | 2s |
| `bc agent list --json` | Agent list with details | 2s |
| `bc agent health --json --detect-stuck` | Health status per agent | 5s |
| `bc channel list --json` | Channel list | 10s |
| `bc cost summary --workspace` | Cost totals | 30s |

## Component Structure

```
Dashboard
├── Header
│   ├── WorkspaceName
│   └── LastRefreshed
├── SummaryCards (horizontal row)
│   ├── TotalAgentsCard
│   ├── ActiveAgentsCard
│   ├── WorkingAgentsCard
│   └── TotalCostCard
├── MainContent (vertical stack)
│   ├── AgentsPanel
│   │   └── AgentTable
│   │       └── AgentRow[]
│   ├── ChannelsPanel
│   │   └── ChannelList
│   │       └── ChannelItem[]
│   └── RecentActivityPanel
│       └── ActivityFeed
│           └── ActivityItem[]
└── Footer
    └── QuickCommands
```

## Component Specifications

### 1. Header

```typescript
interface HeaderProps {
  workspaceName: string;
  lastRefreshed: Date;
}
```

**Display:**
```
bc · my-workspace                    Last updated: 2s ago
```

### 2. SummaryCards

```typescript
interface SummaryCardsProps {
  total: number;
  active: number;
  working: number;
  totalCostUSD: number;
}
```

**Display:**
```
┌─────────────┬─────────────┬─────────────┬─────────────┐
│ 8 Agents    │ 6 Active    │ 4 Working   │ $12.34      │
│   total     │   running   │   on task   │   spent     │
└─────────────┴─────────────┴─────────────┴─────────────┘
```

### 3. AgentsPanel

```typescript
interface Agent {
  name: string;
  role: string;
  state: 'idle' | 'working' | 'done' | 'stuck' | 'error' | 'stopped';
  task: string;
  uptime: string;
  health: 'healthy' | 'degraded' | 'unhealthy' | 'stuck';
}

interface AgentsPanelProps {
  agents: Agent[];
  onSelectAgent: (name: string) => void;
}
```

**Display:**
```
Agents
───────────────────────────────────────────────────────
AGENT           ROLE         STATE      UPTIME    TASK
eng-01          engineer     working    2h 15m    Fix auth bug...
eng-02          engineer     idle       1h 30m    -
tech-lead-01    tech-lead    working    3h 45m    Review PRs...
qa-01           qa           done       45m       Tests passed
```

**State Colors:**
- `working` → green
- `idle` → cyan
- `done` → green
- `stuck` → magenta
- `error` → red
- `stopped` → yellow

### 4. ChannelsPanel

```typescript
interface Channel {
  name: string;
  members: string[];
  messageCount: number;
  lastActivity?: Date;
}

interface ChannelsPanelProps {
  channels: Channel[];
  onSelectChannel: (name: string) => void;
}
```

**Display:**
```
Channels
───────────────────────────────────────────────────────
#engineering     4 members   12 messages   5m ago
#standup         8 members   45 messages   1h ago
#reviews         3 members   3 messages    10m ago
```

### 5. RecentActivityPanel

```typescript
interface ActivityItem {
  channel: string;
  sender: string;
  message: string;
  time: Date;
}

interface RecentActivityPanelProps {
  items: ActivityItem[];
  maxItems?: number; // default: 5
}
```

**Display:**
```
Recent Activity
───────────────────────────────────────────────────────
[#engineering] eng-01: PR #541 ready for review (5m ago)
[#standup] tech-lead: Sprint complete, starting Phase 2 (1h ago)
[#reviews] qa-01: All tests passing ✓ (2h ago)
```

### 6. Footer/QuickCommands

```typescript
interface QuickCommandsProps {
  commands: Array<{ key: string; label: string; action: string }>;
}
```

**Display:**
```
───────────────────────────────────────────────────────
[a] agents  [c] channels  [s] status  [q] quit  [?] help
```

## Keyboard Navigation

| Key | Action |
|-----|--------|
| `a` | Switch to Agents view |
| `c` | Switch to Channels view |
| `s` | Refresh status |
| `q` | Quit TUI |
| `?` | Show help |
| `↑/↓` | Navigate panels |
| `Enter` | Select/expand item |
| `Tab` | Cycle focus between panels |

## Data Flow

```
┌─────────────────────────────────────────────────────┐
│                   Dashboard Component               │
│                                                     │
│  ┌─────────────┐   ┌─────────────┐   ┌──────────┐ │
│  │ useAgents() │   │ useChannels()│  │ useCost()│ │
│  └──────┬──────┘   └──────┬──────┘   └────┬─────┘ │
│         │                 │               │        │
│         v                 v               v        │
│  ┌─────────────────────────────────────────────┐  │
│  │              Service Layer                   │  │
│  │  runBcCommand('agent', ['list', '--json'])  │  │
│  │  runBcCommand('channel', ['list', '--json'])│  │
│  │  runBcCommand('cost', ['summary', '--json'])│  │
│  └──────────────────────┬──────────────────────┘  │
│                         │                          │
│                         v                          │
│  ┌─────────────────────────────────────────────┐  │
│  │              bc CLI (subprocess)             │  │
│  └─────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

## Service Layer Functions

```typescript
// tui/src/services/bc.ts

export async function getStatus(): Promise<StatusResponse> {
  const result = await runBcCommand('status', ['--json']);
  return JSON.parse(result);
}

export async function getAgents(): Promise<Agent[]> {
  const result = await runBcCommand('agent', ['list', '--json']);
  return JSON.parse(result);
}

export async function getAgentHealth(name?: string): Promise<AgentHealth[]> {
  const args = ['health', '--json', '--detect-stuck'];
  if (name) args.push(name);
  const result = await runBcCommand('agent', args);
  return JSON.parse(result);
}

export async function getChannels(): Promise<Channel[]> {
  const result = await runBcCommand('channel', ['list', '--json']);
  return JSON.parse(result);
}

export async function getCostSummary(): Promise<CostSummary> {
  const result = await runBcCommand('cost', ['summary', '--workspace', '--json']);
  return JSON.parse(result);
}
```

## Hooks

```typescript
// tui/src/hooks/useDashboard.ts

export function useDashboard() {
  const agents = useAgents({ refreshInterval: 2000 });
  const channels = useChannels({ refreshInterval: 10000 });
  const cost = useCost({ refreshInterval: 30000 });

  const summary = useMemo(() => ({
    total: agents.data?.length ?? 0,
    active: agents.data?.filter(a => a.state !== 'stopped').length ?? 0,
    working: agents.data?.filter(a => a.state === 'working').length ?? 0,
    totalCostUSD: cost.data?.totalCostUSD ?? 0,
  }), [agents.data, cost.data]);

  return {
    agents,
    channels,
    cost,
    summary,
    isLoading: agents.isLoading || channels.isLoading || cost.isLoading,
    error: agents.error || channels.error || cost.error,
  };
}
```

## Shared Components Needed

Components that should be extracted for reuse:

1. **Box** - Bordered container with title
2. **Table** - Flexible table component
3. **StatusBadge** - Colored status indicator
4. **Card** - Summary metric card
5. **Spinner** - Loading indicator
6. **ErrorBoundary** - Error display component
7. **KeyHint** - Keyboard shortcut hint

## Error Handling

```typescript
// Display when bc command fails
<Box borderStyle="single" borderColor="red">
  <Text color="red">Error: {error.message}</Text>
  <Text dimColor>Press 'r' to retry or 'q' to quit</Text>
</Box>
```

## Loading States

```typescript
// Display while fetching data
<Box>
  <Spinner type="dots" />
  <Text> Loading workspace data...</Text>
</Box>
```

## Implementation Priority

1. **P0 (MVP):**
   - Header with workspace name
   - SummaryCards (counts only)
   - AgentsPanel basic table

2. **P1 (Enhanced):**
   - Cost in SummaryCards
   - ChannelsPanel
   - RecentActivityPanel

3. **P2 (Polish):**
   - Keyboard navigation
   - Panel focus states
   - Help overlay

## Dependencies

- `ink` - React for CLI
- `ink-spinner` - Loading spinners
- `ink-table` - Table rendering (or custom)
- `ink-text-input` - Text input (future)

## Testing Strategy

1. **Unit tests:** Each component with mocked data
2. **Integration tests:** Service layer with mocked bc CLI
3. **Snapshot tests:** Layout verification

## Next Steps

1. eng-02 delivers service layer (#537)
2. eng-02 delivers useAgents/useChannels hooks (#538, #539)
3. eng-03 implements Dashboard component using hooks
4. Integration testing with real bc CLI
