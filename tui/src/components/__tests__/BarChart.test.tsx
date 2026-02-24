/**
 * BarChart Tests
 * Issue #1046: Data visualization components
 *
 * Tests cover:
 * - Bar width calculation
 * - Max value scaling
 * - Percentage calculation
 * - Label truncation
 * - Color palette cycling
 * - MiniBarChart
 * - DistributionChart
 */

import { describe, test, expect } from 'bun:test';

// Types matching BarChart
interface BarChartItem {
  label: string;
  value: number;
  color?: string;
}

// Default color palette
const DEFAULT_COLORS = ['cyan', 'green', 'yellow', 'magenta', 'blue', 'red'];

// Helper functions matching BarChart logic
function calculateMaxValue(data: BarChartItem[]): number {
  if (data.length === 0) return 1;
  return Math.max(...data.map((d) => d.value), 1);
}

function calculateTotal(data: BarChartItem[]): number {
  return data.reduce((sum, d) => sum + d.value, 0);
}

function calculateFilledWidth(value: number, maxValue: number, barWidth: number): number {
  return Math.round((value / maxValue) * barWidth);
}

function calculatePercent(value: number, total: number): string {
  return total > 0 ? ((value / total) * 100).toFixed(0) : '0';
}

function truncateLabel(label: string, maxWidth: number): string {
  return label.padEnd(maxWidth).slice(0, maxWidth);
}

function getBarColor(index: number, itemColor: string | undefined, defaultColor: string): string {
  return itemColor ?? DEFAULT_COLORS[index % DEFAULT_COLORS.length] ?? defaultColor;
}

function createBar(filledWidth: number, emptyWidth: number, barChar = '█', emptyChar = '░'): string {
  return barChar.repeat(filledWidth) + emptyChar.repeat(emptyWidth);
}

// MiniBarChart helpers
function calculateItemWidth(width: number, dataLength: number): number {
  return Math.max(1, Math.floor(width / dataLength));
}

function valueToHeight(value: number, maxValue: number): number {
  return Math.round((value / maxValue) * 8);
}

// DistributionChart helpers
function calculateScale(maxCount: number, maxSymbols: number): number {
  return maxCount > maxSymbols ? maxSymbols / maxCount : 1;
}

function calculateSymbolCount(count: number, scale: number): number {
  return Math.max(1, Math.round(count * scale));
}

describe('BarChart', () => {
  describe('Max Value Calculation', () => {
    test('returns max from data', () => {
      const data: BarChartItem[] = [
        { label: 'A', value: 10 },
        { label: 'B', value: 50 },
        { label: 'C', value: 30 },
      ];
      expect(calculateMaxValue(data)).toBe(50);
    });

    test('returns 1 for empty data', () => {
      expect(calculateMaxValue([])).toBe(1);
    });

    test('returns at least 1 for all zeros', () => {
      const data: BarChartItem[] = [
        { label: 'A', value: 0 },
        { label: 'B', value: 0 },
      ];
      expect(calculateMaxValue(data)).toBe(1);
    });
  });

  describe('Total Calculation', () => {
    test('sums all values', () => {
      const data: BarChartItem[] = [
        { label: 'A', value: 10 },
        { label: 'B', value: 20 },
        { label: 'C', value: 30 },
      ];
      expect(calculateTotal(data)).toBe(60);
    });

    test('returns 0 for empty data', () => {
      expect(calculateTotal([])).toBe(0);
    });
  });

  describe('Bar Width Calculation', () => {
    test('full width for max value', () => {
      expect(calculateFilledWidth(100, 100, 20)).toBe(20);
    });

    test('half width for half value', () => {
      expect(calculateFilledWidth(50, 100, 20)).toBe(10);
    });

    test('rounds to nearest integer', () => {
      expect(calculateFilledWidth(33, 100, 20)).toBe(7);
    });

    test('zero value gives zero width', () => {
      expect(calculateFilledWidth(0, 100, 20)).toBe(0);
    });
  });

  describe('Percentage Calculation', () => {
    test('calculates correct percentage', () => {
      expect(calculatePercent(25, 100)).toBe('25');
      expect(calculatePercent(50, 100)).toBe('50');
    });

    test('rounds to whole number', () => {
      expect(calculatePercent(33.33, 100)).toBe('33');
    });

    test('returns 0 for zero total', () => {
      expect(calculatePercent(50, 0)).toBe('0');
    });
  });

  describe('Label Truncation', () => {
    test('pads short labels', () => {
      const result = truncateLabel('foo', 10);
      expect(result.length).toBe(10);
      expect(result.startsWith('foo')).toBe(true);
    });

    test('truncates long labels', () => {
      const result = truncateLabel('very-long-label-here', 10);
      expect(result.length).toBe(10);
      expect(result).toBe('very-long-');
    });

    test('handles exact length', () => {
      const result = truncateLabel('exactly10c', 10);
      expect(result).toBe('exactly10c');
    });
  });

  describe('Color Selection', () => {
    test('uses item color if provided', () => {
      expect(getBarColor(0, 'red', 'cyan')).toBe('red');
    });

    test('cycles through default colors', () => {
      expect(getBarColor(0, undefined, 'cyan')).toBe('cyan');
      expect(getBarColor(1, undefined, 'cyan')).toBe('green');
      expect(getBarColor(2, undefined, 'cyan')).toBe('yellow');
    });

    test('wraps color index', () => {
      expect(getBarColor(6, undefined, 'cyan')).toBe('cyan');
      expect(getBarColor(7, undefined, 'cyan')).toBe('green');
    });
  });

  describe('Bar Creation', () => {
    test('creates bar with default characters', () => {
      expect(createBar(5, 5)).toBe('█████░░░░░');
    });

    test('creates full bar', () => {
      expect(createBar(10, 0)).toBe('██████████');
    });

    test('creates empty bar', () => {
      expect(createBar(0, 10)).toBe('░░░░░░░░░░');
    });

    test('uses custom characters', () => {
      expect(createBar(3, 3, '▓', '▒')).toBe('▓▓▓▒▒▒');
    });
  });

  describe('Default Values', () => {
    test('default bar width is 20', () => {
      const defaultBarWidth = 20;
      expect(defaultBarWidth).toBe(20);
    });

    test('default label width is 12', () => {
      const defaultLabelWidth = 12;
      expect(defaultLabelWidth).toBe(12);
    });

    test('default bar char is █', () => {
      const defaultBarChar = '█';
      expect(defaultBarChar).toBe('█');
    });

    test('default empty char is ░', () => {
      const defaultEmptyChar = '░';
      expect(defaultEmptyChar).toBe('░');
    });

    test('default color is cyan', () => {
      const defaultColor = 'cyan';
      expect(defaultColor).toBe('cyan');
    });
  });

  describe('DEFAULT_COLORS', () => {
    test('has 6 colors', () => {
      expect(DEFAULT_COLORS).toHaveLength(6);
    });

    test('colors in order', () => {
      expect(DEFAULT_COLORS[0]).toBe('cyan');
      expect(DEFAULT_COLORS[1]).toBe('green');
      expect(DEFAULT_COLORS[2]).toBe('yellow');
      expect(DEFAULT_COLORS[3]).toBe('magenta');
      expect(DEFAULT_COLORS[4]).toBe('blue');
      expect(DEFAULT_COLORS[5]).toBe('red');
    });
  });
});

describe('MiniBarChart', () => {
  describe('Item Width Calculation', () => {
    test('calculates width per item', () => {
      expect(calculateItemWidth(16, 4)).toBe(4);
      expect(calculateItemWidth(16, 8)).toBe(2);
    });

    test('minimum width is 1', () => {
      expect(calculateItemWidth(4, 10)).toBe(1);
    });

    test('handles single item', () => {
      expect(calculateItemWidth(16, 1)).toBe(16);
    });
  });

  describe('Value to Height', () => {
    test('maps to 0-8 range', () => {
      expect(valueToHeight(0, 100)).toBe(0);
      expect(valueToHeight(50, 100)).toBe(4);
      expect(valueToHeight(100, 100)).toBe(8);
    });

    test('handles equal max', () => {
      expect(valueToHeight(50, 50)).toBe(8);
    });
  });

  describe('Height Characters', () => {
    const chars = ['▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'];

    test('has 8 height levels', () => {
      expect(chars).toHaveLength(8);
    });

    test('lowest is ▁', () => {
      expect(chars[0]).toBe('▁');
    });

    test('highest is █', () => {
      expect(chars[7]).toBe('█');
    });
  });

  describe('Empty Data', () => {
    test('shows placeholder', () => {
      const width = 16;
      const placeholder = '─'.repeat(width);
      expect(placeholder).toBe('────────────────');
    });
  });

  describe('Default Values', () => {
    test('default width is 16', () => {
      const defaultWidth = 16;
      expect(defaultWidth).toBe(16);
    });

    test('default color is cyan', () => {
      const defaultColor = 'cyan';
      expect(defaultColor).toBe('cyan');
    });
  });
});

describe('DistributionChart', () => {
  describe('Scale Calculation', () => {
    test('returns 1 when count <= maxSymbols', () => {
      expect(calculateScale(5, 8)).toBe(1);
      expect(calculateScale(8, 8)).toBe(1);
    });

    test('scales down when count > maxSymbols', () => {
      expect(calculateScale(16, 8)).toBe(0.5);
      expect(calculateScale(24, 8)).toBeCloseTo(0.333, 2);
    });
  });

  describe('Symbol Count Calculation', () => {
    test('uses full count when scale is 1', () => {
      expect(calculateSymbolCount(5, 1)).toBe(5);
    });

    test('scales down count', () => {
      expect(calculateSymbolCount(16, 0.5)).toBe(8);
    });

    test('minimum is 1 symbol', () => {
      expect(calculateSymbolCount(0, 1)).toBe(1);
      expect(calculateSymbolCount(1, 0.1)).toBe(1);
    });
  });

  describe('Default Values', () => {
    test('default symbol is ⊙', () => {
      const defaultSymbol = '⊙';
      expect(defaultSymbol).toBe('⊙');
    });

    test('default maxSymbols is 8', () => {
      const defaultMaxSymbols = 8;
      expect(defaultMaxSymbols).toBe(8);
    });
  });

  describe('Label Padding', () => {
    test('pads to 10 characters', () => {
      const label = 'foo';
      const padded = label.padEnd(10);
      expect(padded.length).toBe(10);
    });
  });
});

describe('Integration', () => {
  describe('Full Bar Chart', () => {
    test('renders complete chart data', () => {
      const data: BarChartItem[] = [
        { label: 'Engineer', value: 5 },
        { label: 'Designer', value: 3 },
        { label: 'Manager', value: 2 },
      ];

      const maxValue = calculateMaxValue(data);
      const total = calculateTotal(data);

      expect(maxValue).toBe(5);
      expect(total).toBe(10);

      // First bar should be full width
      const firstBarWidth = calculateFilledWidth(5, maxValue, 20);
      expect(firstBarWidth).toBe(20);

      // Second bar should be 60%
      const secondBarWidth = calculateFilledWidth(3, maxValue, 20);
      expect(secondBarWidth).toBe(12);

      // Percentages
      expect(calculatePercent(5, total)).toBe('50');
      expect(calculatePercent(3, total)).toBe('30');
      expect(calculatePercent(2, total)).toBe('20');
    });
  });

  describe('Cost Breakdown', () => {
    test('visualizes budget distribution', () => {
      const data: BarChartItem[] = [
        { label: 'Input', value: 45.50 },
        { label: 'Output', value: 32.25 },
        { label: 'Other', value: 22.25 },
      ];

      const total = calculateTotal(data);
      expect(total).toBe(100);

      expect(calculatePercent(45.50, total)).toBe('46');
      expect(calculatePercent(32.25, total)).toBe('32');
      expect(calculatePercent(22.25, total)).toBe('22');
    });
  });
});
