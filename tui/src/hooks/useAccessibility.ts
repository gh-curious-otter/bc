/**
 * useAccessibility - Accessibility utilities for colorblind and high contrast support
 * Issue #1220: Add colorblind-friendly visual cues
 *
 * Provides:
 * - High contrast mode detection (BC_HIGH_CONTRAST env var)
 * - Colorblind-safe icon mappings
 * - Pattern characters for visual differentiation
 * - Text label helpers
 */

import { useState, useEffect, useMemo, useContext } from 'react';
import ThemeContext from '../theme/ThemeContext';

// ============================================================================
// High Contrast Mode
// ============================================================================

/**
 * Check if high contrast mode is enabled
 * Respects BC_HIGH_CONTRAST environment variable and config
 */
export function isHighContrastEnabled(): boolean {
  return (
    process.env.BC_HIGH_CONTRAST === '1' ||
    process.env.BC_HIGH_CONTRAST === 'true' ||
    process.env.BC_TUI_HIGH_CONTRAST === '1' ||
    process.env.BC_TUI_HIGH_CONTRAST === 'true'
  );
}

/**
 * Hook to check high contrast mode preference
 */
export function useHighContrast(): boolean {
  const [highContrast, setHighContrast] = useState(isHighContrastEnabled);

  useEffect(() => {
    setHighContrast(isHighContrastEnabled());
  }, []);

  return highContrast;
}

// ============================================================================
// Status Icons - Colorblind Safe
// ============================================================================

/**
 * Status icon mappings for colorblind accessibility
 * Icons provide visual cues independent of color
 */
export const STATUS_ICONS = {
  // Success/Healthy states
  success: '✓',
  healthy: '✓',
  done: '✓',
  complete: '✓',
  active: '●',
  running: '●',
  working: '●',

  // Error/Unhealthy states
  error: '✗',
  failed: '✗',
  unhealthy: '✗',
  stopped: '◌',

  // Warning/Degraded states
  warning: '⚠',
  degraded: '⚠',
  stuck: '!',
  pending: '◐',
  starting: '◐',

  // Neutral states
  idle: '○',
  unknown: '?',
  info: '·',
} as const;

export type StatusIconKey = keyof typeof STATUS_ICONS;

/**
 * Get status icon for a given status/state
 */
export function getStatusIcon(status: string): string {
  const normalized = status.toLowerCase().replace(/[_-]/g, '');
  if (normalized in STATUS_ICONS) {
    return STATUS_ICONS[normalized as StatusIconKey];
  }
  return STATUS_ICONS.unknown;
}

// ============================================================================
// Severity Icons - For Logs and Alerts
// ============================================================================

export const SEVERITY_ICONS = {
  error: '✗',
  warn: '⚠',
  warning: '⚠',
  info: '·',
  debug: '○',
  trace: '·',
} as const;

export type SeverityLevel = keyof typeof SEVERITY_ICONS;

/**
 * Get severity icon
 */
export function getSeverityIcon(severity: string): string {
  const normalized = severity.toLowerCase();
  if (normalized in SEVERITY_ICONS) {
    return SEVERITY_ICONS[normalized as SeverityLevel];
  }
  return SEVERITY_ICONS.info;
}

// ============================================================================
// Progress/Chart Patterns - Pattern Differentiation
// ============================================================================

/**
 * Unicode patterns for charts and progress bars
 * Distinguishable without color
 */
export const PATTERNS = {
  solid: '█',
  dark: '▓',
  medium: '▒',
  light: '░',
  empty: ' ',
  // Progress bar styles
  filled: '━',
  partial: '╸',
  unfilled: '─',
  // Block styles
  full: '■',
  half: '▪',
  quarter: '▫',
} as const;

/**
 * Get pattern characters for a progress level
 * @param level - 0 to 4 indicating fill level
 */
export function getPatternForLevel(level: 0 | 1 | 2 | 3 | 4): string {
  const patterns = [PATTERNS.empty, PATTERNS.light, PATTERNS.medium, PATTERNS.dark, PATTERNS.solid];
  return patterns[level];
}

// ============================================================================
// Text Labels - Always Include Text
// ============================================================================

/**
 * Status text labels for accessibility
 * Should be shown alongside colored indicators
 */
export const STATUS_LABELS = {
  // Agent states
  idle: 'idle',
  starting: 'starting',
  working: 'working',
  done: 'done',
  stuck: 'STUCK',
  error: 'ERROR',
  stopped: 'stopped',

  // Health states
  healthy: 'healthy',
  degraded: 'DEGRADED',
  unhealthy: 'UNHEALTHY',

  // Progress states
  pending: 'pending',
  complete: 'complete',
  failed: 'FAILED',
} as const;

/**
 * Get accessible text label for status
 * Critical states are uppercased for emphasis
 */
export function getStatusLabel(status: string): string {
  const normalized = status.toLowerCase();
  if (normalized in STATUS_LABELS) {
    return STATUS_LABELS[normalized as keyof typeof STATUS_LABELS];
  }
  return status;
}

// ============================================================================
// High Contrast Colors
// ============================================================================

/**
 * High contrast color mappings
 * More distinct colors for visibility
 */
export const HIGH_CONTRAST_COLORS = {
  success: '#00FF00', // Bright green
  error: '#FF0000', // Bright red
  warning: '#FFFF00', // Bright yellow
  info: '#FFFFFF', // White
  primary: '#00FFFF', // Cyan
  secondary: '#FF00FF', // Magenta
  muted: '#808080', // Gray
} as const;

/**
 * Get high contrast color for semantic type
 */
export function getHighContrastColor(
  type: 'success' | 'error' | 'warning' | 'info' | 'primary' | 'secondary' | 'muted'
): string {
  return HIGH_CONTRAST_COLORS[type];
}

// ============================================================================
// Combined Accessibility Hook
// ============================================================================

export interface AccessibilitySettings {
  /** High contrast mode enabled */
  highContrast: boolean;
  /** Get icon for status */
  getIcon: (status: string) => string;
  /** Get label for status */
  getLabel: (status: string) => string;
  /** Get pattern for fill level */
  getPattern: (level: 0 | 1 | 2 | 3 | 4) => string;
  /** Get accessible color (uses high contrast when enabled) */
  getColor: (type: 'success' | 'error' | 'warning' | 'info') => string;
}

/**
 * Combined hook for all accessibility features
 */
export function useAccessibility(): AccessibilitySettings {
  const highContrast = useHighContrast();
  const themeContext = useContext(ThemeContext);

  return useMemo(() => ({
    highContrast,
    getIcon: getStatusIcon,
    getLabel: getStatusLabel,
    getPattern: getPatternForLevel,
    getColor: (type: 'success' | 'error' | 'warning' | 'info') => {
      if (highContrast) {
        return getHighContrastColor(type);
      }
      // Use theme colors if available
      if (themeContext) {
        const colorMap: Record<string, string> = {
          success: themeContext.theme.colors.agentDone,
          error: themeContext.theme.colors.agentError,
          warning: themeContext.theme.colors.warning,
          info: themeContext.theme.colors.text,
        };
        return colorMap[type] || 'white';
      }
      // Fallback colors
      const fallbackMap: Record<string, string> = {
        success: 'green',
        error: 'red',
        warning: 'yellow',
        info: 'gray',
      };
      return fallbackMap[type] || 'white';
    },
  }), [highContrast, themeContext]);
}

export default useAccessibility;
