/**
 * ProgressBar - Visual progress indicator with color coding
 * Issue #864 - Cost visualizations
 */

import React, { memo } from 'react';
import { Box, Text } from 'ink';

export interface ProgressBarProps {
  /** Current value (0-100 for percentage, or any number with max) */
  value: number;
  /** Maximum value (default: 100) */
  max?: number;
  /** Width of the bar in characters (default: 20) */
  width?: number;
  /** Show percentage text (default: true) */
  showPercent?: boolean;
  /** Show value text like "$50/$100" (default: false) */
  showValue?: boolean;
  /** Prefix for value display (e.g., "$") */
  valuePrefix?: string;
  /** Color thresholds - green below 50%, yellow below 80%, red above */
  colorThresholds?: { warning: number; critical: number };
  /** Custom label to show after the bar */
  label?: string;
}

/**
 * Get color based on percentage and thresholds
 */
function getBarColor(
  percent: number,
  thresholds: { warning: number; critical: number }
): 'green' | 'yellow' | 'red' {
  if (percent >= thresholds.critical) return 'red';
  if (percent >= thresholds.warning) return 'yellow';
  return 'green';
}

/**
 * ProgressBar component with budget-style visualization
 *
 * Memoized for performance - Issue #1003 Phase 3 optimization.
 */
export const ProgressBar = memo(function ProgressBar({
  value,
  max = 100,
  width = 20,
  showPercent = true,
  showValue = false,
  valuePrefix = '',
  colorThresholds = { warning: 50, critical: 80 },
  label,
}: ProgressBarProps): React.ReactElement {
  // Clamp value to valid range and calculate percentage
  const clampedValue = Math.max(0, Math.min(value, max));
  const percent = max > 0 ? Math.min(100, (clampedValue / max) * 100) : 0;
  const filledWidth = Math.round((percent / 100) * width);
  const emptyWidth = width - filledWidth;

  const filled = '█'.repeat(filledWidth);
  const empty = '░'.repeat(emptyWidth);
  const color = getBarColor(percent, colorThresholds);

  return (
    <Box>
      <Text>[</Text>
      <Text color={color}>{filled}</Text>
      <Text dimColor>{empty}</Text>
      <Text>]</Text>
      {showPercent && <Text color={color}> {percent.toFixed(0)}%</Text>}
      {showValue && (
        <Text dimColor>
          {' '}
          ({valuePrefix}
          {value.toFixed(2)}/{valuePrefix}
          {max.toFixed(2)})
        </Text>
      )}
      {label && <Text dimColor> {label}</Text>}
    </Box>
  );
});

/**
 * Inline progress bar for table cells (smaller, no brackets)
 *
 * Memoized for performance - Issue #1003 Phase 3 optimization.
 */
export const InlineProgressBar = memo(function InlineProgressBar({
  value,
  max = 100,
  width = 10,
}: {
  value: number;
  max?: number;
  width?: number;
}): React.ReactElement {
  const percent = max > 0 ? Math.min(100, (value / max) * 100) : 0;
  const filledWidth = Math.round((percent / 100) * width);
  const emptyWidth = width - filledWidth;

  const filled = '█'.repeat(filledWidth);
  const empty = '░'.repeat(emptyWidth);
  const color = getBarColor(percent, { warning: 50, critical: 80 });

  return (
    <Text color={color}>
      {filled}
      <Text dimColor>{empty}</Text>
    </Text>
  );
});

export default ProgressBar;
