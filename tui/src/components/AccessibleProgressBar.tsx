/**
 * AccessibleProgressBar - Colorblind-friendly progress bar with patterns
 * Issue #1220: Add colorblind-friendly visual cues
 *
 * Uses unicode patterns for visual differentiation beyond color:
 * - Solid blocks for filled portion
 * - Light shade for unfilled portion
 * - Always shows percentage text
 */

import { memo } from 'react';
import { Box, Text } from 'ink';
import { useHighContrast, PATTERNS, getHighContrastColor } from '../hooks/useAccessibility';

export interface AccessibleProgressBarProps {
  /** Progress value 0-100 */
  value: number;
  /** Bar width in characters (default: 20) */
  width?: number;
  /** Label text (shown before bar) */
  label?: string;
  /** Status type for color (default: 'info') */
  status?: 'success' | 'warning' | 'error' | 'info';
  /** Show percentage text (default: true) */
  showPercent?: boolean;
  /** Show numeric value (default: false) */
  showValue?: boolean;
  /** Maximum value for showValue display (default: 100) */
  maxValue?: number;
}

/**
 * Get status-specific pattern
 */
function getStatusPattern(status: string): { filled: string; empty: string } {
  switch (status) {
    case 'success':
      return { filled: PATTERNS.solid, empty: PATTERNS.light };
    case 'warning':
      return { filled: PATTERNS.dark, empty: PATTERNS.light };
    case 'error':
      return { filled: PATTERNS.medium, empty: PATTERNS.light };
    default:
      return { filled: PATTERNS.solid, empty: PATTERNS.light };
  }
}

/**
 * Get status color
 */
function getStatusColor(status: string, highContrast: boolean): string {
  if (highContrast) {
    return getHighContrastColor(status as 'success' | 'error' | 'warning' | 'info');
  }

  switch (status) {
    case 'success':
      return 'green';
    case 'warning':
      return 'yellow';
    case 'error':
      return 'red';
    default:
      return 'blue';
  }
}

/**
 * AccessibleProgressBar - Progress bar with colorblind-friendly patterns
 *
 * Provides visual differentiation through:
 * - Different fill patterns per status
 * - Always visible percentage text
 * - High contrast color support
 */
export const AccessibleProgressBar = memo(function AccessibleProgressBar({
  value,
  width = 20,
  label,
  status = 'info',
  showPercent = true,
  showValue = false,
  maxValue = 100,
}: AccessibleProgressBarProps) {
  const highContrast = useHighContrast();

  // Clamp value to 0-100
  const clampedValue = Math.max(0, Math.min(100, value));
  const filledWidth = Math.round((clampedValue / 100) * width);
  const emptyWidth = width - filledWidth;

  const { filled, empty } = getStatusPattern(status);
  const color = getStatusColor(status, highContrast);

  const filledBar = filled.repeat(filledWidth);
  const emptyBar = empty.repeat(emptyWidth);

  return (
    <Box>
      {label && (
        <Text>{label} </Text>
      )}
      <Text color={color}>{filledBar}</Text>
      <Text dimColor>{emptyBar}</Text>
      {showPercent && (
        <Text> {Math.round(clampedValue)}%</Text>
      )}
      {showValue && (
        <Text dimColor> ({Math.round((clampedValue / 100) * maxValue)}/{maxValue})</Text>
      )}
    </Box>
  );
});

/**
 * BudgetProgressBar - Specialized progress bar for budget/cost display
 * Shows different patterns based on budget utilization
 */
export interface BudgetProgressBarProps {
  /** Current spend */
  current: number;
  /** Budget limit */
  limit: number;
  /** Bar width (default: 15) */
  width?: number;
  /** Warning threshold (default: 80) */
  warningThreshold?: number;
  /** Error threshold (default: 95) */
  errorThreshold?: number;
}

export const BudgetProgressBar = memo(function BudgetProgressBar({
  current,
  limit,
  width = 15,
  warningThreshold = 80,
  errorThreshold = 95,
}: BudgetProgressBarProps) {
  const percent = limit > 0 ? (current / limit) * 100 : 0;

  let status: 'success' | 'warning' | 'error' = 'success';
  if (percent >= errorThreshold) {
    status = 'error';
  } else if (percent >= warningThreshold) {
    status = 'warning';
  }

  return (
    <AccessibleProgressBar
      value={percent}
      width={width}
      status={status}
      showPercent={true}
      showValue={false}
    />
  );
});

export default AccessibleProgressBar;
