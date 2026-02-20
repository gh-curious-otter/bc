import { useContext, memo } from 'react';
import { Text } from 'ink';
import ThemeContext from '../theme/ThemeContext';
import type { ThemeColors } from '../theme/types';
import { useHighContrast, getHighContrastColor } from '../hooks/useAccessibility';

// Agent states from eng-04's implementation
export type AgentState = 'idle' | 'starting' | 'working' | 'done' | 'stuck' | 'error' | 'stopped';

// Health states added by eng-03
export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy';

export interface StatusBadgeProps {
  state: string;
  /** Show icon before state (default: true) */
  showIcon?: boolean;
  /** Show text label after icon (default: true) - Issue #1220 colorblind accessibility */
  showLabel?: boolean;
  /** Use compact display (icon only) */
  compact?: boolean;
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

/**
 * High contrast color mappings for colorblind accessibility
 * Issue #1220: More distinct colors for visibility
 */
const highContrastColors: Record<string, 'success' | 'error' | 'warning' | 'info'> = {
  // Success/healthy states
  done: 'success',
  healthy: 'success',
  // Error states
  error: 'error',
  stuck: 'error',
  unhealthy: 'error',
  // Warning states
  starting: 'warning',
  degraded: 'warning',
  // Neutral states
  idle: 'info',
  working: 'info',
  stopped: 'info',
};

/**
 * Status symbols for colorblind accessibility
 * Issue #1220: Icons provide visual cues independent of color
 */
const stateSymbols: Record<string, string> = {
  idle: '○',
  starting: '◐',
  working: '●',
  done: '✓',
  stuck: '!',
  error: '✗',
  stopped: '◌',
  healthy: '✓',
  degraded: '⚠',
  unhealthy: '✗',
};

/**
 * Text labels for critical states (uppercase for emphasis)
 * Issue #1220: Always include text status, not just colored indicators
 */
const stateLabels: Record<string, string> = {
  idle: 'idle',
  starting: 'starting',
  working: 'working',
  done: 'done',
  stuck: 'STUCK',
  error: 'ERROR',
  stopped: 'stopped',
  healthy: 'healthy',
  degraded: 'DEGRADED',
  unhealthy: 'UNHEALTHY',
};

/**
 * StatusBadge - Colored status indicator matching bc CLI
 *
 * Supports theming when wrapped in ThemeProvider, otherwise uses fallback colors.
 * Merged from eng-04 (#561) and eng-03 (#562), updated for theming (#558)
 *
 * Issue #1220: Colorblind-friendly visual cues
 * - Icons always shown alongside colors
 * - Text labels accompany colored elements
 * - High contrast mode support (BC_HIGH_CONTRAST=1)
 *
 * Memoized for performance - Issue #1003 Phase 3 optimization.
 */
export const StatusBadge = memo(function StatusBadge({
  state,
  showIcon = true,
  showLabel = true,
  compact = false,
}: StatusBadgeProps) {
  const themeContext = useContext(ThemeContext);
  const highContrast = useHighContrast();
  const symbol = stateSymbols[state] ?? '?';
  const label = stateLabels[state] ?? state;

  // Get color from theme, high contrast, or fallback
  let color: string;
  if (highContrast) {
    const hcType = highContrastColors[state] ?? 'info';
    color = getHighContrastColor(hcType);
  } else if (themeContext) {
    const colorKey = getThemeColorKey(state);
    color = colorKey ? themeContext.theme.colors[colorKey] : 'white';
  } else {
    color = fallbackColors[state] ?? 'white';
  }

  // Compact mode: icon only
  if (compact) {
    return (
      <Text color={color}>
        {symbol}
      </Text>
    );
  }

  return (
    <Text color={color}>
      {showIcon ? `${symbol} ` : ''}{showLabel ? label : ''}
    </Text>
  );
});

export default StatusBadge;
