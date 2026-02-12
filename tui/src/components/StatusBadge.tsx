
import { Text } from 'ink';

// Agent states from eng-04's implementation
export type AgentState = 'idle' | 'starting' | 'working' | 'done' | 'stuck' | 'error' | 'stopped';

// Health states added by eng-03
export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy';

export interface StatusBadgeProps {
  state: AgentState | HealthStatus | string;
  showIcon?: boolean;
}

const stateColors: Record<string, string> = {
  // Agent states
  idle: 'gray',
  starting: 'yellow',
  working: 'blue',
  done: 'green',
  stuck: 'red',
  error: 'red',
  stopped: 'gray',
  // Health states
  healthy: 'green',
  degraded: 'yellow',
  unhealthy: 'red',
};

const stateSymbols: Record<string, string> = {
  idle: '○',
  starting: '◐',
  working: '●',
  done: '✓',
  stuck: '!',
  error: '✗',
  stopped: '◌',
  healthy: '✓',
  degraded: '!',
  unhealthy: '✗',
};

/**
 * StatusBadge - Colored status indicator matching bc CLI
 * Merged from eng-04 (#561) and eng-03 (#562)
 */
export function StatusBadge({ state, showIcon = true }: StatusBadgeProps) {
  const color = stateColors[state] || 'white';
  const symbol = stateSymbols[state] || '?';

  return (
    <Text color={color}>
      {showIcon ? `${symbol} ` : ''}{state}
    </Text>
  );
}

export default StatusBadge;
