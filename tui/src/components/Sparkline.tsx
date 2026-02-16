/**
 * Sparkline - Mini line chart for trend visualization
 * Issue #864 - Cost visualizations
 */

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

export default Sparkline;
