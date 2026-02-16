/**
 * Sparkline - Mini line chart for trend visualization
 * Issue #864 - Cost visualizations
 */

import React from 'react';
import { Box, Text } from 'ink';

export interface SparklineProps {
  /** Array of numeric values to chart */
  data: number[];
  /** Width in characters (defaults to data length) */
  width?: number;
  /** Color of the sparkline */
  color?: string;
  /** Show min/max labels */
  showRange?: boolean;
  /** Label for the sparkline */
  label?: string;
}

// Sparkline characters from lowest to highest
const SPARK_CHARS = ['▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'];

/**
 * Map a value to a sparkline character based on min/max range
 */
function valueToChar(value: number, min: number, max: number): string {
  if (max === min) return SPARK_CHARS[4]; // Middle char if all values equal
  const normalized = (value - min) / (max - min);
  const index = Math.min(SPARK_CHARS.length - 1, Math.floor(normalized * SPARK_CHARS.length));
  return SPARK_CHARS[index];
}

/**
 * Render a sparkline chart showing data trends
 */
export function Sparkline({
  data,
  width,
  color = 'cyan',
  showRange = false,
  label,
}: SparklineProps): React.ReactElement {
  if (data.length === 0) {
    return (
      <Box>
        {label && <Text dimColor>{label}: </Text>}
        <Text dimColor>No data</Text>
      </Box>
    );
  }

  const min = Math.min(...data);
  const max = Math.max(...data);

  // Resample data if width is specified and different from data length
  let chartData = data;
  if (width && width !== data.length && data.length > 1) {
    chartData = resampleData(data, width);
  }

  const sparkChars = chartData.map(v => valueToChar(v, min, max)).join('');

  return (
    <Box>
      {label && <Text dimColor>{label}: </Text>}
      <Text color={color}>{sparkChars}</Text>
      {showRange && (
        <Text dimColor> [{formatValue(min)}-{formatValue(max)}]</Text>
      )}
    </Box>
  );
}

/**
 * Resample data array to target length using linear interpolation
 */
function resampleData(data: number[], targetLength: number): number[] {
  if (targetLength <= 0 || data.length === 0) return [];
  if (data.length === 1) return Array<number>(targetLength).fill(data[0]);

  const result: number[] = [];
  const step = (data.length - 1) / (targetLength - 1);

  for (let i = 0; i < targetLength; i++) {
    const pos = i * step;
    const lower = Math.floor(pos);
    const upper = Math.ceil(pos);
    const fraction = pos - lower;

    if (upper >= data.length) {
      result.push(data[data.length - 1]);
    } else {
      result.push(data[lower] + (data[upper] - data[lower]) * fraction);
    }
  }

  return result;
}

/**
 * Format value for display (K/M suffixes for large numbers)
 */
function formatValue(n: number): string {
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(1)}M`;
  }
  if (n >= 1_000) {
    return `${(n / 1_000).toFixed(1)}K`;
  }
  if (n < 1 && n > 0) {
    return n.toFixed(2);
  }
  return n.toFixed(0);
}

/**
 * TrendSparkline - Sparkline with trend direction indicator
 * Shows arrow (↑ ↓ →) based on data trend
 */
export interface TrendSparklineProps {
  /** Array of numeric values to chart */
  data: number[];
  /** Width in characters (defaults to data length) */
  width?: number;
  /** Color of the sparkline */
  color?: string;
  /** Show trend arrow (default: true) */
  showTrend?: boolean;
}

/**
 * Render a sparkline with trend indicator
 */
export function TrendSparkline({
  data,
  width,
  color = 'cyan',
  showTrend = true,
}: TrendSparklineProps): React.ReactElement {
  if (data.length === 0) {
    return (
      <Box>
        <Text dimColor>{'─'.repeat(width ?? 8)} ─</Text>
      </Box>
    );
  }

  const min = Math.min(...data);
  const max = Math.max(...data);

  // Resample data if width is specified
  let chartData = data;
  if (width && width !== data.length && data.length > 1) {
    chartData = resampleData(data, width);
  }

  const sparkChars = chartData.map(v => valueToChar(v, min, max)).join('');

  // Calculate trend by comparing first half to second half
  let trendChar = '→';
  let trendColor = 'gray';

  if (showTrend && data.length >= 2) {
    const midpoint = Math.floor(data.length / 2);
    const firstHalf = data.slice(0, midpoint);
    const secondHalf = data.slice(midpoint);

    const firstAvg = firstHalf.reduce((a, b) => a + b, 0) / firstHalf.length;
    const secondAvg = secondHalf.reduce((a, b) => a + b, 0) / secondHalf.length;

    // 5% threshold to avoid showing trend for noise
    const percentChange = firstAvg > 0 ? ((secondAvg - firstAvg) / firstAvg) * 100 : 0;

    if (percentChange > 5) {
      trendChar = '↑';
      trendColor = 'green';
    } else if (percentChange < -5) {
      trendChar = '↓';
      trendColor = 'red';
    }
  }

  return (
    <Box>
      <Text color={color}>{sparkChars}</Text>
      {showTrend && (
        <Text color={trendColor}> {trendChar}</Text>
      )}
    </Box>
  );
}

/**
 * MiniSparkline - Compact sparkline for inline use in tables
 * No label, fixed width, suitable for table cells
 */
export interface MiniSparklineProps {
  /** Array of numeric values to chart */
  data: number[];
  /** Width in characters (default: 8) */
  width?: number;
  /** Color of the sparkline */
  color?: string;
}

export function MiniSparkline({
  data,
  width = 8,
  color = 'cyan',
}: MiniSparklineProps): React.ReactElement {
  if (data.length === 0) {
    return <Text dimColor>{'─'.repeat(width)}</Text>;
  }

  const min = Math.min(...data);
  const max = Math.max(...data);

  let chartData = data;
  if (data.length > width) {
    chartData = resampleData(data, width);
  }

  const sparkChars = chartData.map(v => valueToChar(v, min, max)).join('');

  return <Text color={color}>{sparkChars}</Text>;
}

export default Sparkline;
