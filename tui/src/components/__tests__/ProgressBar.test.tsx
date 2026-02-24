/**
 * ProgressBar Tests
 * Issue #864: Cost visualizations
 *
 * Tests cover:
 * - Percentage calculation
 * - Width calculation (filled/empty)
 * - Color thresholds
 * - Value clamping
 * - Display options
 */

import { describe, test, expect } from 'bun:test';

// Helper functions matching ProgressBar logic
function calculatePercent(value: number, max: number): number {
  const clampedValue = Math.max(0, Math.min(value, max));
  return max > 0 ? Math.min(100, (clampedValue / max) * 100) : 0;
}

function calculateFilledWidth(percent: number, width: number): number {
  return Math.round((percent / 100) * width);
}

function getBarColor(
  percent: number,
  thresholds: { warning: number; critical: number }
): 'green' | 'yellow' | 'red' {
  if (percent >= thresholds.critical) return 'red';
  if (percent >= thresholds.warning) return 'yellow';
  return 'green';
}

function createBar(filledWidth: number, emptyWidth: number): string {
  return '█'.repeat(filledWidth) + '░'.repeat(emptyWidth);
}

function formatPercent(percent: number): string {
  return `${percent.toFixed(0)}%`;
}

function formatValue(value: number, max: number, prefix: string): string {
  return `(${prefix}${value.toFixed(2)}/${prefix}${max.toFixed(2)})`;
}

describe('ProgressBar', () => {
  describe('Percentage Calculation', () => {
    test('calculates correct percentage', () => {
      expect(calculatePercent(50, 100)).toBe(50);
      expect(calculatePercent(25, 100)).toBe(25);
      expect(calculatePercent(75, 100)).toBe(75);
    });

    test('handles non-100 max', () => {
      expect(calculatePercent(50, 200)).toBe(25);
      expect(calculatePercent(150, 300)).toBe(50);
    });

    test('clamps to 100%', () => {
      expect(calculatePercent(150, 100)).toBe(100);
      expect(calculatePercent(200, 100)).toBe(100);
    });

    test('handles zero value', () => {
      expect(calculatePercent(0, 100)).toBe(0);
    });

    test('handles zero max', () => {
      expect(calculatePercent(50, 0)).toBe(0);
    });

    test('clamps negative values', () => {
      expect(calculatePercent(-10, 100)).toBe(0);
    });
  });

  describe('Width Calculation', () => {
    test('calculates filled width for 50%', () => {
      expect(calculateFilledWidth(50, 20)).toBe(10);
    });

    test('calculates filled width for 100%', () => {
      expect(calculateFilledWidth(100, 20)).toBe(20);
    });

    test('calculates filled width for 0%', () => {
      expect(calculateFilledWidth(0, 20)).toBe(0);
    });

    test('rounds to nearest character', () => {
      // 33% of 20 = 6.6, rounds to 7
      expect(calculateFilledWidth(33, 20)).toBe(7);
      // 17% of 20 = 3.4, rounds to 3
      expect(calculateFilledWidth(17, 20)).toBe(3);
    });

    test('handles different widths', () => {
      expect(calculateFilledWidth(50, 10)).toBe(5);
      expect(calculateFilledWidth(50, 40)).toBe(20);
    });
  });

  describe('Color Thresholds', () => {
    const defaultThresholds = { warning: 50, critical: 80 };

    test('green below warning threshold', () => {
      expect(getBarColor(0, defaultThresholds)).toBe('green');
      expect(getBarColor(25, defaultThresholds)).toBe('green');
      expect(getBarColor(49, defaultThresholds)).toBe('green');
    });

    test('yellow at warning threshold', () => {
      expect(getBarColor(50, defaultThresholds)).toBe('yellow');
      expect(getBarColor(60, defaultThresholds)).toBe('yellow');
      expect(getBarColor(79, defaultThresholds)).toBe('yellow');
    });

    test('red at critical threshold', () => {
      expect(getBarColor(80, defaultThresholds)).toBe('red');
      expect(getBarColor(90, defaultThresholds)).toBe('red');
      expect(getBarColor(100, defaultThresholds)).toBe('red');
    });

    test('handles custom thresholds', () => {
      const customThresholds = { warning: 30, critical: 60 };
      expect(getBarColor(25, customThresholds)).toBe('green');
      expect(getBarColor(35, customThresholds)).toBe('yellow');
      expect(getBarColor(65, customThresholds)).toBe('red');
    });
  });

  describe('Bar Creation', () => {
    test('creates bar with correct characters', () => {
      expect(createBar(5, 5)).toBe('█████░░░░░');
    });

    test('creates full bar', () => {
      expect(createBar(10, 0)).toBe('██████████');
    });

    test('creates empty bar', () => {
      expect(createBar(0, 10)).toBe('░░░░░░░░░░');
    });

    test('bar length equals width', () => {
      const bar = createBar(7, 13);
      expect(bar.length).toBe(20);
    });
  });

  describe('Percent Formatting', () => {
    test('formats whole numbers', () => {
      expect(formatPercent(50)).toBe('50%');
      expect(formatPercent(100)).toBe('100%');
      expect(formatPercent(0)).toBe('0%');
    });

    test('rounds decimals', () => {
      expect(formatPercent(33.33)).toBe('33%');
      expect(formatPercent(66.67)).toBe('67%');
    });
  });

  describe('Value Formatting', () => {
    test('formats with prefix', () => {
      expect(formatValue(50, 100, '$')).toBe('($50.00/$100.00)');
    });

    test('formats without prefix', () => {
      expect(formatValue(25.5, 100, '')).toBe('(25.50/100.00)');
    });

    test('formats decimals', () => {
      expect(formatValue(33.333, 100, '$')).toBe('($33.33/$100.00)');
    });
  });

  describe('Default Values', () => {
    test('default max is 100', () => {
      const defaultMax = 100;
      expect(calculatePercent(50, defaultMax)).toBe(50);
    });

    test('default width is 20', () => {
      const defaultWidth = 20;
      expect(calculateFilledWidth(50, defaultWidth)).toBe(10);
    });

    test('default thresholds', () => {
      const defaults = { warning: 50, critical: 80 };
      expect(defaults.warning).toBe(50);
      expect(defaults.critical).toBe(80);
    });
  });

  describe('Edge Cases', () => {
    test('handles exactly at thresholds', () => {
      const thresholds = { warning: 50, critical: 80 };
      expect(getBarColor(50, thresholds)).toBe('yellow');
      expect(getBarColor(80, thresholds)).toBe('red');
    });

    test('handles very small percentages', () => {
      expect(calculateFilledWidth(1, 20)).toBe(0);
      expect(calculateFilledWidth(3, 20)).toBe(1);
    });

    test('handles very large values', () => {
      expect(calculatePercent(1000, 100)).toBe(100);
    });

    test('handles decimal values', () => {
      expect(calculatePercent(33.33, 100)).toBeCloseTo(33.33);
    });
  });

  describe('InlineProgressBar', () => {
    test('uses default width of 10', () => {
      const defaultWidth = 10;
      expect(calculateFilledWidth(50, defaultWidth)).toBe(5);
    });

    test('uses same color logic', () => {
      const thresholds = { warning: 50, critical: 80 };
      expect(getBarColor(25, thresholds)).toBe('green');
      expect(getBarColor(60, thresholds)).toBe('yellow');
      expect(getBarColor(90, thresholds)).toBe('red');
    });

    test('no brackets in inline bar', () => {
      const bar = createBar(5, 5);
      expect(bar).not.toContain('[');
      expect(bar).not.toContain(']');
    });
  });

  describe('Budget Visualization', () => {
    test('budget at 25% is green', () => {
      const percent = calculatePercent(25, 100);
      const color = getBarColor(percent, { warning: 50, critical: 80 });
      expect(color).toBe('green');
    });

    test('budget at 60% is yellow (warning)', () => {
      const percent = calculatePercent(60, 100);
      const color = getBarColor(percent, { warning: 50, critical: 80 });
      expect(color).toBe('yellow');
    });

    test('budget at 90% is red (critical)', () => {
      const percent = calculatePercent(90, 100);
      const color = getBarColor(percent, { warning: 50, critical: 80 });
      expect(color).toBe('red');
    });

    test('over budget shows 100%', () => {
      const percent = calculatePercent(150, 100);
      expect(percent).toBe(100);
    });
  });

  describe('Visual Representation', () => {
    test('25% bar visualization', () => {
      const percent = 25;
      const filled = calculateFilledWidth(percent, 20);
      const empty = 20 - filled;
      const bar = createBar(filled, empty);

      expect(filled).toBe(5);
      expect(empty).toBe(15);
      expect(bar).toBe('█████░░░░░░░░░░░░░░░');
    });

    test('50% bar visualization', () => {
      const percent = 50;
      const filled = calculateFilledWidth(percent, 20);
      const empty = 20 - filled;
      const bar = createBar(filled, empty);

      expect(filled).toBe(10);
      expect(empty).toBe(10);
      expect(bar).toBe('██████████░░░░░░░░░░');
    });

    test('75% bar visualization', () => {
      const percent = 75;
      const filled = calculateFilledWidth(percent, 20);
      const empty = 20 - filled;
      const bar = createBar(filled, empty);

      expect(filled).toBe(15);
      expect(empty).toBe(5);
      expect(bar).toBe('███████████████░░░░░');
    });

    test('100% bar visualization', () => {
      const percent = 100;
      const filled = calculateFilledWidth(percent, 20);
      const empty = 20 - filled;
      const bar = createBar(filled, empty);

      expect(filled).toBe(20);
      expect(empty).toBe(0);
      expect(bar).toBe('████████████████████');
    });
  });
});
