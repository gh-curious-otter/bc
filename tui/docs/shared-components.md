# Shared Components Design

**Author:** eng-03
**Phase:** 2 Prep
**Status:** Draft

## Overview

This document defines shared components that will be reused across multiple views (Dashboard, Agents, Channels, etc.). Using ink-ui as a foundation where possible.

## Dependencies

```json
{
  "dependencies": {
    "ink": "^4.4.0",
    "ink-ui": "^2.0.0",
    "react": "^18.0.0"
  }
}
```

## Component Catalog

### 1. DataTable

A flexible table component for displaying structured data.

```typescript
// tui/src/components/DataTable.tsx

interface Column<T> {
  key: keyof T;
  header: string;
  width?: number;
  align?: 'left' | 'center' | 'right';
  render?: (value: T[keyof T], row: T) => React.ReactNode;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  selectedIndex?: number;
  onSelect?: (row: T, index: number) => void;
  emptyMessage?: string;
  showHeader?: boolean;
}

function DataTable<T>({ columns, data, selectedIndex, onSelect, emptyMessage, showHeader = true }: DataTableProps<T>) {
  // Implementation using Box/Text with flexbox layout
}
```

**Usage:**
```tsx
<DataTable
  columns={[
    { key: 'name', header: 'AGENT', width: 15 },
    { key: 'role', header: 'ROLE', width: 12 },
    { key: 'state', header: 'STATE', width: 10, render: (v) => <StatusBadge status={v} /> },
    { key: 'uptime', header: 'UPTIME', width: 10 },
    { key: 'task', header: 'TASK' },
  ]}
  data={agents}
  selectedIndex={selectedIndex}
  onSelect={handleAgentSelect}
/>
```

### 2. StatusBadge

Colored status indicator matching bc CLI state colors.

```typescript
// tui/src/components/StatusBadge.tsx

type AgentStatus = 'idle' | 'working' | 'done' | 'stuck' | 'error' | 'stopped';
type HealthStatus = 'healthy' | 'degraded' | 'unhealthy' | 'stuck';

interface StatusBadgeProps {
  status: AgentStatus | HealthStatus;
  showIcon?: boolean;
}

const STATUS_COLORS: Record<string, string> = {
  // Agent states
  idle: 'cyan',
  working: 'green',
  done: 'green',
  stuck: 'magenta',
  error: 'red',
  stopped: 'yellow',
  // Health states
  healthy: 'green',
  degraded: 'yellow',
  unhealthy: 'red',
};

const STATUS_ICONS: Record<string, string> = {
  working: '●',
  idle: '○',
  done: '✓',
  stuck: '!',
  error: '✗',
  stopped: '◌',
  healthy: '✓',
  degraded: '!',
  unhealthy: '✗',
};

function StatusBadge({ status, showIcon = false }: StatusBadgeProps) {
  return (
    <Text color={STATUS_COLORS[status]}>
      {showIcon && STATUS_ICONS[status] + ' '}
      {status}
    </Text>
  );
}
```

### 3. Panel

Bordered container with optional title.

```typescript
// tui/src/components/Panel.tsx

interface PanelProps {
  title?: string;
  children: React.ReactNode;
  borderColor?: string;
  focused?: boolean;
  width?: number | string;
  height?: number | string;
}

function Panel({ title, children, borderColor = 'gray', focused, width, height }: PanelProps) {
  return (
    <Box
      flexDirection="column"
      borderStyle="single"
      borderColor={focused ? 'blue' : borderColor}
      width={width}
      height={height}
    >
      {title && (
        <Box marginBottom={1}>
          <Text bold>{title}</Text>
        </Box>
      )}
      {children}
    </Box>
  );
}
```

### 4. MetricCard

Compact metric display for summary dashboards.

```typescript
// tui/src/components/MetricCard.tsx

interface MetricCardProps {
  value: number | string;
  label: string;
  color?: string;
  prefix?: string;  // e.g., '$' for cost
  suffix?: string;  // e.g., '%' for percentage
}

function MetricCard({ value, label, color = 'white', prefix = '', suffix = '' }: MetricCardProps) {
  return (
    <Box flexDirection="column" paddingX={1}>
      <Text bold color={color}>
        {prefix}{value}{suffix}
      </Text>
      <Text dimColor>{label}</Text>
    </Box>
  );
}
```

**Usage:**
```tsx
<Box>
  <MetricCard value={8} label="Agents" />
  <MetricCard value={6} label="Active" color="green" />
  <MetricCard value={4} label="Working" color="cyan" />
  <MetricCard value="12.34" label="Cost" prefix="$" />
</Box>
```

### 5. KeyHint

Keyboard shortcut hint display.

```typescript
// tui/src/components/KeyHint.tsx

interface KeyHintProps {
  keyChar: string;
  label: string;
}

function KeyHint({ keyChar, label }: KeyHintProps) {
  return (
    <Box marginRight={2}>
      <Text>[</Text>
      <Text bold color="blue">{keyChar}</Text>
      <Text>] {label}</Text>
    </Box>
  );
}

// Footer component using KeyHint
interface FooterProps {
  hints: Array<{ key: string; label: string }>;
}

function Footer({ hints }: FooterProps) {
  return (
    <Box borderStyle="single" borderTop borderBottom={false} borderLeft={false} borderRight={false}>
      {hints.map(h => <KeyHint key={h.key} keyChar={h.key} label={h.label} />)}
    </Box>
  );
}
```

### 6. LoadingIndicator

Loading state with spinner (uses ink-ui Spinner).

```typescript
// tui/src/components/LoadingIndicator.tsx
import { Spinner } from 'ink-ui';

interface LoadingIndicatorProps {
  message?: string;
}

function LoadingIndicator({ message = 'Loading...' }: LoadingIndicatorProps) {
  return (
    <Box>
      <Spinner type="dots" />
      <Text> {message}</Text>
    </Box>
  );
}
```

### 7. ErrorDisplay

Error message display with retry option.

```typescript
// tui/src/components/ErrorDisplay.tsx

interface ErrorDisplayProps {
  error: Error | string;
  onRetry?: () => void;
}

function ErrorDisplay({ error, onRetry }: ErrorDisplayProps) {
  const message = typeof error === 'string' ? error : error.message;

  return (
    <Box flexDirection="column" borderStyle="single" borderColor="red" padding={1}>
      <Text color="red" bold>Error</Text>
      <Text color="red">{message}</Text>
      {onRetry && (
        <Text dimColor>Press 'r' to retry</Text>
      )}
    </Box>
  );
}
```

### 8. ActivityItem

Single activity/message display for feeds.

```typescript
// tui/src/components/ActivityItem.tsx

interface ActivityItemProps {
  channel: string;
  sender: string;
  message: string;
  time: Date;
}

function ActivityItem({ channel, sender, message, time }: ActivityItemProps) {
  const timeAgo = formatTimeAgo(time);
  const truncatedMsg = message.length > 50 ? message.slice(0, 47) + '...' : message;

  return (
    <Box>
      <Text color="cyan">[#{channel}]</Text>
      <Text> </Text>
      <Text bold>{sender}</Text>
      <Text>: {truncatedMsg} </Text>
      <Text dimColor>({timeAgo})</Text>
    </Box>
  );
}

function formatTimeAgo(date: Date): string {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}
```

### 9. ProgressIndicator

Progress bar for operations (uses ink-ui ProgressBar).

```typescript
// tui/src/components/ProgressIndicator.tsx
import { ProgressBar } from 'ink-ui';

interface ProgressIndicatorProps {
  progress: number; // 0-100
  label?: string;
}

function ProgressIndicator({ progress, label }: ProgressIndicatorProps) {
  return (
    <Box flexDirection="column">
      {label && <Text>{label}</Text>}
      <ProgressBar value={progress} />
    </Box>
  );
}
```

## Hooks

### useKeyboardNav

Navigation hook for keyboard-driven interfaces.

```typescript
// tui/src/hooks/useKeyboardNav.ts
import { useInput } from 'ink';

interface UseKeyboardNavOptions {
  itemCount: number;
  onSelect?: (index: number) => void;
  onEscape?: () => void;
}

function useKeyboardNav({ itemCount, onSelect, onEscape }: UseKeyboardNavOptions) {
  const [selectedIndex, setSelectedIndex] = useState(0);

  useInput((input, key) => {
    if (key.upArrow && selectedIndex > 0) {
      setSelectedIndex(i => i - 1);
    }
    if (key.downArrow && selectedIndex < itemCount - 1) {
      setSelectedIndex(i => i + 1);
    }
    if (key.return && onSelect) {
      onSelect(selectedIndex);
    }
    if (key.escape && onEscape) {
      onEscape();
    }
  });

  return { selectedIndex, setSelectedIndex };
}
```

### useFocusManager

Focus management for multi-panel layouts.

```typescript
// tui/src/hooks/useFocusManager.ts

type PanelId = 'agents' | 'channels' | 'activity' | 'commands';

function useFocusManager(panels: PanelId[]) {
  const [focusedPanel, setFocusedPanel] = useState<PanelId>(panels[0]);

  useInput((input, key) => {
    if (key.tab) {
      const currentIndex = panels.indexOf(focusedPanel);
      const nextIndex = (currentIndex + 1) % panels.length;
      setFocusedPanel(panels[nextIndex]);
    }
  });

  return {
    focusedPanel,
    setFocusedPanel,
    isFocused: (panel: PanelId) => focusedPanel === panel,
  };
}
```

## Theme Configuration

```typescript
// tui/src/theme.ts

export const theme = {
  colors: {
    primary: 'blue',
    secondary: 'cyan',
    success: 'green',
    warning: 'yellow',
    error: 'red',
    muted: 'gray',
  },
  states: {
    working: 'green',
    idle: 'cyan',
    done: 'green',
    stuck: 'magenta',
    error: 'red',
    stopped: 'yellow',
  },
  borders: {
    default: 'single',
    focused: 'double',
  },
};
```

## File Structure

```
tui/src/
├── components/
│   ├── index.ts           # Re-exports all components
│   ├── DataTable.tsx
│   ├── StatusBadge.tsx
│   ├── Panel.tsx
│   ├── MetricCard.tsx
│   ├── KeyHint.tsx
│   ├── Footer.tsx
│   ├── LoadingIndicator.tsx
│   ├── ErrorDisplay.tsx
│   ├── ActivityItem.tsx
│   └── ProgressIndicator.tsx
├── hooks/
│   ├── index.ts
│   ├── useKeyboardNav.ts
│   └── useFocusManager.ts
├── theme.ts
└── utils/
    └── formatters.ts      # formatTimeAgo, truncate, etc.
```

## Usage by Views

| Component | Dashboard | Agents | Channels |
|-----------|-----------|--------|----------|
| DataTable | ✓ | ✓ | ✓ |
| StatusBadge | ✓ | ✓ | - |
| Panel | ✓ | ✓ | ✓ |
| MetricCard | ✓ | - | - |
| KeyHint/Footer | ✓ | ✓ | ✓ |
| LoadingIndicator | ✓ | ✓ | ✓ |
| ErrorDisplay | ✓ | ✓ | ✓ |
| ActivityItem | ✓ | - | ✓ |

## Implementation Priority

**P0 (Required for MVP):**
- DataTable
- StatusBadge
- Panel
- LoadingIndicator
- ErrorDisplay

**P1 (Enhanced UX):**
- MetricCard
- KeyHint/Footer
- useKeyboardNav

**P2 (Polish):**
- ActivityItem
- useFocusManager
- Theme configuration

## References

- [Ink GitHub](https://github.com/vadimdemedes/ink)
- [Ink UI Components](https://github.com/vadimdemedes/ink-ui)
- [Ink Documentation](https://github.com/vadimdemedes/ink#readme)
