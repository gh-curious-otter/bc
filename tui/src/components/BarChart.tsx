/**
 * BarChart - Simple horizontal bar chart for data visualization
 * Issue #1046 - Data visualization components
 */

import React, { memo, useMemo } from 'react';
import { Box, Text } from 'ink';

export interface BarChartItem {
  /** Label for the bar */
  label: string;
  /** Numeric value */
  value: number;
  /** Optional color override */
  color?: string;
}

export interface BarChartProps {
  /** Array of items to display */
  data: BarChartItem[];
  /** Width of the bar portion in characters (default: 20) */
  barWidth?: number;
  /** Show value numbers (default: true) */
  showValues?: boolean;
  /** Show percentages instead of absolute values */
  showPercent?: boolean;
  /** Maximum label width (default: 12) */
  labelWidth?: number;
  /** Bar character (default: '█') */
  barChar?: string;
  /** Empty bar character (default: '░') */
  emptyChar?: string;
  /** Default bar color (default: 'cyan') */
  defaultColor?: string;
  /** Title for the chart */
  title?: string;
}

// Default color palette for bars
const DEFAULT_COLORS = ['cyan', 'green', 'yellow', 'magenta', 'blue', 'red'];

/**
 * BarChart component - horizontal bar chart
 */
export const BarChart = memo(function BarChart({
  data,
  barWidth = 20,
  showValues = true,
  showPercent = false,
  labelWidth = 12,
  barChar = '█',
  emptyChar = '░',
  defaultColor = 'cyan',
  title,
}: BarChartProps): React.ReactElement {
  // Calculate max value for scaling
  const maxValue = useMemo(() => {
    if (data.length === 0) return 1;
    return Math.max(...data.map((d) => d.value), 1);
  }, [data]);

  // Calculate total for percentages
  const total = useMemo(() => {
    return data.reduce((sum, d) => sum + d.value, 0);
  }, [data]);

  if (data.length === 0) {
    return (
      <Box flexDirection="column">
        {title && <Text bold>{title}</Text>}
        <Text dimColor>No data</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      {title && <Text bold>{title}</Text>}
      {data.map((item, idx) => {
        const filledWidth = Math.round((item.value / maxValue) * barWidth);
        const emptyWidth = barWidth - filledWidth;
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- item.color can be undefined per BarChartItem type
        const color = item.color ?? DEFAULT_COLORS[idx % DEFAULT_COLORS.length] ?? defaultColor;
        const percent = total > 0 ? ((item.value / total) * 100).toFixed(0) : '0';

        return (
          <Box key={item.label}>
            <Text>{item.label.padEnd(labelWidth).slice(0, labelWidth)}</Text>
            <Text dimColor> </Text>
            <Text color={color}>{barChar.repeat(filledWidth)}</Text>
            <Text dimColor>{emptyChar.repeat(emptyWidth)}</Text>
            {showValues && !showPercent && (
              <Text dimColor> ({item.value})</Text>
            )}
            {showPercent && (
              <Text dimColor> {percent}%</Text>
            )}
          </Box>
        );
      })}
    </Box>
  );
});

/**
 * MiniBarChart - Compact bar chart for inline use
 * Shows bars without labels, suitable for small spaces
 */
export interface MiniBarChartProps {
  /** Array of numeric values */
  data: number[];
  /** Width in characters (default: 16) */
  width?: number;
  /** Color of the bars */
  color?: string;
  /** Show total count */
  showTotal?: boolean;
}

export const MiniBarChart = memo(function MiniBarChart({
  data,
  width = 16,
  color = 'cyan',
  showTotal = false,
}: MiniBarChartProps): React.ReactElement {
  const maxValue = Math.max(...data, 1);
  const total = data.reduce((sum, v) => sum + v, 0);

  // Calculate bar width per item
  const itemWidth = Math.max(1, Math.floor(width / data.length));

  if (data.length === 0) {
    return <Text dimColor>{'─'.repeat(width)}</Text>;
  }

  return (
    <Box>
      {data.map((value, idx) => {
        const height = Math.round((value / maxValue) * 8);
        const chars = ['▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'];
        const char = chars[Math.min(height, 7)] ?? '▁';
        return (
          <Text key={idx} color={color}>
            {char.repeat(itemWidth)}
          </Text>
        );
      })}
      {showTotal && <Text dimColor> ({total})</Text>}
    </Box>
  );
});

/**
 * Distribution chart using symbols instead of bars
 * Good for showing role/status distribution in compact space
 */
export interface DistributionChartProps {
  /** Array of items with label and count */
  data: { label: string; count: number; color?: string }[];
  /** Symbol to use (default: '⊙') */
  symbol?: string;
  /** Maximum symbols per row (default: 8) */
  maxSymbols?: number;
  /** Show counts (default: true) */
  showCounts?: boolean;
}

export const DistributionChart = memo(function DistributionChart({
  data,
  symbol = '⊙',
  maxSymbols = 8,
  showCounts = true,
}: DistributionChartProps): React.ReactElement {
  if (data.length === 0) {
    return <Text dimColor>No data</Text>;
  }

  const maxCount = Math.max(...data.map((d) => d.count));
  const scale = maxCount > maxSymbols ? maxSymbols / maxCount : 1;

  return (
    <Box flexDirection="column">
      {data.map((item, idx) => {
        const symbolCount = Math.max(1, Math.round(item.count * scale));
        const color = item.color ?? DEFAULT_COLORS[idx % DEFAULT_COLORS.length];

        return (
          <Box key={item.label}>
            <Text>{item.label.padEnd(10)}: </Text>
            <Text color={color}>{symbol.repeat(symbolCount)}</Text>
            {showCounts && <Text dimColor> ({item.count})</Text>}
          </Box>
        );
      })}
    </Box>
  );
});

export default BarChart;
