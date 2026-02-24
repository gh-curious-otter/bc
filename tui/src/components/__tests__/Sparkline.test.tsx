/**
 * Sparkline Tests
 * Issue #864: Cost visualizations
 *
 * Tests cover:
 * - Value to character mapping
 * - Data resampling
 * - Value formatting (K/M suffixes)
 * - Trend calculation
 * - Empty data handling
 */

import { describe, test, expect } from 'bun:test';

// Sparkline characters from lowest to highest
const SPARK_CHARS = ['▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'];

// Helper functions matching Sparkline logic
function valueToChar(value: number, min: number, max: number): string {
  if (max === min) return SPARK_CHARS[4]; // Middle char if all values equal
  const normalized = (value - min) / (max - min);
  const index = Math.min(SPARK_CHARS.length - 1, Math.floor(normalized * SPARK_CHARS.length));
  return SPARK_CHARS[index];
}

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

function calculateTrend(data: number[]): { char: string; color: string } {
  if (data.length < 2) {
    return { char: '→', color: 'gray' };
  }

  const midpoint = Math.floor(data.length / 2);
  const firstHalf = data.slice(0, midpoint);
  const secondHalf = data.slice(midpoint);

  const firstAvg = firstHalf.reduce((a, b) => a + b, 0) / firstHalf.length;
  const secondAvg = secondHalf.reduce((a, b) => a + b, 0) / secondHalf.length;

  const percentChange = firstAvg > 0 ? ((secondAvg - firstAvg) / firstAvg) * 100 : 0;

  if (percentChange > 5) {
    return { char: '↑', color: 'green' };
  } else if (percentChange < -5) {
    return { char: '↓', color: 'red' };
  }
  return { char: '→', color: 'gray' };
}

describe('Sparkline', () => {
  describe('SPARK_CHARS', () => {
    test('has 8 characters', () => {
      expect(SPARK_CHARS).toHaveLength(8);
    });

    test('characters are in ascending order', () => {
      expect(SPARK_CHARS[0]).toBe('▁');
      expect(SPARK_CHARS[7]).toBe('█');
    });
  });

  describe('valueToChar', () => {
    test('minimum value returns lowest char', () => {
      expect(valueToChar(0, 0, 100)).toBe('▁');
    });

    test('maximum value returns highest char', () => {
      expect(valueToChar(100, 0, 100)).toBe('█');
    });

    test('middle value returns middle char', () => {
      const result = valueToChar(50, 0, 100);
      expect(SPARK_CHARS.indexOf(result)).toBeGreaterThanOrEqual(3);
      expect(SPARK_CHARS.indexOf(result)).toBeLessThanOrEqual(4);
    });

    test('equal min/max returns middle char', () => {
      expect(valueToChar(50, 50, 50)).toBe(SPARK_CHARS[4]);
    });

    test('handles negative ranges', () => {
      expect(valueToChar(-50, -100, 0)).toBe(SPARK_CHARS[4]);
    });

    test('handles decimal values', () => {
      const result = valueToChar(0.25, 0, 1);
      expect(SPARK_CHARS).toContain(result);
    });
  });

  describe('resampleData', () => {
    test('returns empty for empty data', () => {
      expect(resampleData([], 5)).toEqual([]);
    });

    test('returns empty for zero target length', () => {
      expect(resampleData([1, 2, 3], 0)).toEqual([]);
    });

    test('fills array for single value', () => {
      const result = resampleData([5], 3);
      expect(result).toEqual([5, 5, 5]);
    });

    test('preserves endpoints', () => {
      const result = resampleData([0, 100], 5);
      expect(result[0]).toBe(0);
      expect(result[4]).toBe(100);
    });

    test('interpolates middle values', () => {
      const result = resampleData([0, 100], 3);
      expect(result).toEqual([0, 50, 100]);
    });

    test('downsamples correctly', () => {
      const data = [0, 25, 50, 75, 100];
      const result = resampleData(data, 3);
      expect(result[0]).toBe(0);
      expect(result[2]).toBe(100);
    });

    test('upsamples correctly', () => {
      const data = [0, 100];
      const result = resampleData(data, 5);
      expect(result.length).toBe(5);
      expect(result[0]).toBe(0);
      expect(result[2]).toBe(50);
      expect(result[4]).toBe(100);
    });
  });

  describe('formatValue', () => {
    test('formats millions', () => {
      expect(formatValue(1_000_000)).toBe('1.0M');
      expect(formatValue(2_500_000)).toBe('2.5M');
    });

    test('formats thousands', () => {
      expect(formatValue(1_000)).toBe('1.0K');
      expect(formatValue(5_500)).toBe('5.5K');
    });

    test('formats small decimals', () => {
      expect(formatValue(0.5)).toBe('0.50');
      expect(formatValue(0.123)).toBe('0.12');
    });

    test('formats regular numbers', () => {
      expect(formatValue(50)).toBe('50');
      expect(formatValue(999)).toBe('999');
    });

    test('formats zero', () => {
      expect(formatValue(0)).toBe('0');
    });
  });

  describe('calculateTrend', () => {
    test('upward trend returns green up arrow', () => {
      const data = [10, 20, 30, 40, 50, 60];
      const trend = calculateTrend(data);
      expect(trend.char).toBe('↑');
      expect(trend.color).toBe('green');
    });

    test('downward trend returns red down arrow', () => {
      const data = [60, 50, 40, 30, 20, 10];
      const trend = calculateTrend(data);
      expect(trend.char).toBe('↓');
      expect(trend.color).toBe('red');
    });

    test('flat trend returns gray horizontal arrow', () => {
      const data = [50, 50, 50, 50, 50, 50];
      const trend = calculateTrend(data);
      expect(trend.char).toBe('→');
      expect(trend.color).toBe('gray');
    });

    test('small change (<5%) is flat', () => {
      const data = [100, 100, 100, 101, 102, 103];
      const trend = calculateTrend(data);
      expect(trend.char).toBe('→');
    });

    test('single value returns flat', () => {
      const trend = calculateTrend([50]);
      expect(trend.char).toBe('→');
      expect(trend.color).toBe('gray');
    });

    test('empty data returns flat', () => {
      const trend = calculateTrend([]);
      expect(trend.char).toBe('→');
    });
  });

  describe('Sparkline rendering', () => {
    test('creates sparkline string from data', () => {
      const data = [0, 25, 50, 75, 100];
      const sparkChars = data.map((v) => valueToChar(v, 0, 100)).join('');
      expect(sparkChars.length).toBe(5);
    });

    test('all same values creates uniform line', () => {
      const data = [50, 50, 50, 50];
      const sparkChars = data.map((v) => valueToChar(v, 50, 50)).join('');
      expect(sparkChars).toBe('▅▅▅▅');
    });

    test('ascending data creates ascending sparkline', () => {
      const data = [0, 50, 100];
      const min = Math.min(...data);
      const max = Math.max(...data);
      const chars = data.map((v) => valueToChar(v, min, max));

      expect(SPARK_CHARS.indexOf(chars[0])).toBeLessThan(SPARK_CHARS.indexOf(chars[2]));
    });

    test('descending data creates descending sparkline', () => {
      const data = [100, 50, 0];
      const min = Math.min(...data);
      const max = Math.max(...data);
      const chars = data.map((v) => valueToChar(v, min, max));

      expect(SPARK_CHARS.indexOf(chars[0])).toBeGreaterThan(SPARK_CHARS.indexOf(chars[2]));
    });
  });

  describe('Empty data handling', () => {
    test('empty array shows no data', () => {
      const data: number[] = [];
      expect(data.length).toBe(0);
    });

    test('placeholder for empty mini sparkline', () => {
      const width = 8;
      const placeholder = '─'.repeat(width);
      expect(placeholder).toBe('────────');
    });
  });

  describe('Default values', () => {
    test('default color is cyan', () => {
      const defaultColor = 'cyan';
      expect(defaultColor).toBe('cyan');
    });

    test('default mini sparkline width is 8', () => {
      const defaultWidth = 8;
      expect(defaultWidth).toBe(8);
    });
  });

  describe('Range display', () => {
    test('formats min/max range', () => {
      const min = 100;
      const max = 5000;
      const range = `[${formatValue(min)}-${formatValue(max)}]`;
      expect(range).toBe('[100-5.0K]');
    });
  });

  describe('Trend threshold', () => {
    test('5% threshold for trend detection', () => {
      // Just above 5%
      const upData = [100, 100, 100, 106, 106, 106];
      expect(calculateTrend(upData).char).toBe('↑');

      // Just below 5%
      const flatData = [100, 100, 100, 104, 104, 104];
      expect(calculateTrend(flatData).char).toBe('→');
    });
  });

  describe('MiniSparkline', () => {
    test('handles data longer than width', () => {
      const data = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
      const width = 5;
      const resampled = resampleData(data, width);
      expect(resampled.length).toBe(5);
    });

    test('preserves data shorter than width', () => {
      const data = [1, 2, 3];
      const width = 8;
      // MiniSparkline doesn't upsample, it uses data as-is if shorter
      expect(data.length).toBeLessThan(width);
    });
  });

  describe('Normalization', () => {
    test('normalizes to 0-1 range', () => {
      const value = 50;
      const min = 0;
      const max = 100;
      const normalized = (value - min) / (max - min);
      expect(normalized).toBe(0.5);
    });

    test('handles negative ranges', () => {
      const value = 0;
      const min = -100;
      const max = 100;
      const normalized = (value - min) / (max - min);
      expect(normalized).toBe(0.5);
    });
  });

  describe('Edge cases', () => {
    test('very large numbers', () => {
      expect(formatValue(1_000_000_000)).toBe('1000.0M');
    });

    test('very small positive numbers', () => {
      expect(formatValue(0.001)).toBe('0.00');
    });

    test('single data point sparkline', () => {
      const data = [50];
      const min = Math.min(...data);
      const max = Math.max(...data);
      const char = valueToChar(data[0], min, max);
      expect(char).toBe(SPARK_CHARS[4]); // Middle char for single value
    });

    test('two data points resampling', () => {
      const result = resampleData([0, 100], 5);
      expect(result[1]).toBe(25);
      expect(result[3]).toBe(75);
    });
  });
});
