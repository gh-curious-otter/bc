/**
 * StatusColors - Consistent color scheme for status indicators across all views
 * Issue #1040: Establish consistent color scheme for status indicators
 *
 * Defines status colors used consistently across Dashboard, Agents, Channels, Costs, Logs views.
 * Combines symbols + colors for accessibility (not color-only coding).
 */

/**
 * Status color definitions with semantic meaning
 */
export const STATUS_COLORS = {
  working: 'cyan',      // Active/Working agents
  idle: 'gray',         // Inactive/Idle agents
  done: 'green',        // Completed/Done state
  error: 'red',         // Error/Failed state
  warning: 'yellow',    // Warning/Attention needed
  info: 'blue',         // Information/General
  pending: 'gray',      // Pending/Not started
  success: 'green',     // Success/OK state
} as const;

export type StatusColor = keyof typeof STATUS_COLORS;

/**
 * Status symbols (for accessibility - symbol + color combination)
 */
export const STATUS_SYMBOLS = {
  working: '⊙',  // Filled circle - actively working
  idle: '○',     // Empty circle - not working
  done: '✓',     // Checkmark - completed
  error: '✗',    // X mark - failed
  warning: '⚠',  // Warning sign
  info: 'ℹ',     // Info symbol
  pending: '−',  // Dash - not started
  success: '✓',  // Checkmark - OK
} as const;

/**
 * Get color for a given status
 */
export function getStatusColor(status: StatusColor): string {
  return STATUS_COLORS[status];
}

/**
 * Get symbol for a given status
 */
export function getStatusSymbol(status: StatusColor): string {
  return STATUS_SYMBOLS[status];
}

/**
 * Get both color and symbol for a status (recommended pattern)
 */
export function getStatusIndicator(status: StatusColor): { color: string; symbol: string } {
  return {
    color: getStatusColor(status),
    symbol: getStatusSymbol(status),
  };
}

/**
 * Health status colors (for progress/health indicators)
 */
export const HEALTH_COLORS = {
  healthy: 'green',      // 80-100% healthy
  warning: 'yellow',     // 50-79% healthy
  critical: 'red',       // <50% healthy
} as const;

/**
 * Cost/budget colors
 */
export const COST_COLORS = {
  normal: 'green',       // <75% of budget used
  warning: 'yellow',     // 75-90% of budget used
  critical: 'red',       // >90% of budget used
} as const;

/**
 * Agent role colors (for visual distinction in lists)
 */
export const ROLE_COLORS: Record<string, string> = {
  engineer: 'cyan',
  manager: 'blue',
  ux: 'magenta',
  root: 'red',
  default: 'white',
};

/**
 * Get color for an agent role
 */
export function getRoleColor(role: string): string {
  return ROLE_COLORS[role] || ROLE_COLORS.default;
}

export default {
  STATUS_COLORS,
  STATUS_SYMBOLS,
  HEALTH_COLORS,
  COST_COLORS,
  ROLE_COLORS,
  getStatusColor,
  getStatusSymbol,
  getStatusIndicator,
  getRoleColor,
};
