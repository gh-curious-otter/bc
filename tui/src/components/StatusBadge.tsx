import { useContext } from 'react';
import { Text } from 'ink';
import ThemeContext from '../theme/ThemeContext';
import type { ThemeColors } from '../theme/types';

// Agent states from eng-04's implementation
export type AgentState = 'idle' | 'starting' | 'working' | 'done' | 'stuck' | 'error' | 'stopped';

// Health states added by eng-03
export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy';

export interface StatusBadgeProps {
  state: AgentState | HealthStatus | string;
  showIcon?: boolean;
}

/**
 * Map state to theme color key
 */
function getThemeColorKey(state: string): keyof ThemeColors | null {
  switch (state) {
    case 'idle':
    case 'stopped':
      return 'agentIdle';
    case 'starting':
      return 'warning';
    case 'working':
      return 'agentWorking';
    case 'done':
    case 'healthy':
      return 'agentDone';
    case 'stuck':
    case 'error':
    case 'unhealthy':
      return 'agentError';
    case 'degraded':
      return 'warning';
    default:
      return null;
  }
}

/**
 * Fallback colors when not using ThemeProvider
 */
const fallbackColors: Record<string, string> = {
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
 *
 * Supports theming when wrapped in ThemeProvider, otherwise uses fallback colors.
 * Merged from eng-04 (#561) and eng-03 (#562), updated for theming (#558)
 */
export function StatusBadge({ state, showIcon = true }: StatusBadgeProps) {
  const themeContext = useContext(ThemeContext);
  const symbol = stateSymbols[state] || '?';

  // Get color from theme or fallback
  let color: string;
  if (themeContext) {
    const colorKey = getThemeColorKey(state);
    color = colorKey ? themeContext.theme.colors[colorKey] : 'white';
  } else {
    color = fallbackColors[state] || 'white';
  }

  return (
    <Text color={color}>
      {showIcon ? `${symbol} ` : ''}{state}
    </Text>
  );
}

export default StatusBadge;
