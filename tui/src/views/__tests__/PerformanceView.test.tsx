/**
 * PerformanceView Tests
 * Issue #1759: Performance/Monitor tab for system observability
 *
 * Tests cover:
 * - Helper functions (formatUptime, formatNumber)
 * - Progress bar calculation logic
 * - Metric row color logic
 * - Utilization calculations
 * - Health status determination
 */

import { describe, test, expect } from 'bun:test';

// Helper functions matching PerformanceView logic
function formatUptime(seconds: number): string {
  if (seconds < 60) {
    return `${String(Math.floor(seconds))}s`;
  }
  const mins = Math.floor(seconds / 60);
  if (mins < 60) {
    return `${String(mins)}m`;
  }
  const hours = Math.floor(mins / 60);
  const remainingMins = mins % 60;
  return `${String(hours)}h ${String(remainingMins)}m`;
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(1)}M`;
  }
  if (n >= 1_000) {
    return `${(n / 1_000).toFixed(1)}K`;
  }
  return n.toString();
}

function calculateProgressBarFilled(percent: number, width: number): number {
  return Math.round((percent / 100) * width);
}

function getUtilizationPercent(working: number, active: number): number {
  return active > 0 ? Math.round((working / active) * 100) : 0;
}

function getHealthPercent(healthyCount: number, total: number): number {
  return total > 0 ? Math.round((healthyCount / total) * 100) : 100;
}

function getMetricColor(avg: number, type: 'avg' | 'max'): string {
  if (type === 'avg') {
    return avg < 16 ? 'green' : avg < 50 ? 'yellow' : 'red';
  }
  return avg < 50 ? 'green' : avg < 100 ? 'yellow' : 'red';
}

function getUtilizationColor(utilization: number): string {
  return utilization >= 80 ? 'green' : utilization >= 50 ? 'yellow' : 'gray';
}

function truncateMetricName(name: string, maxLen: number = 16): string {
  return name.length > maxLen ? name.slice(0, maxLen - 1) + '…' : name;
}

interface PerformanceMetric {
  name: string;
  value: number;
  average: number;
  min: number;
  max: number;
  count: number;
}

describe('PerformanceView', () => {
  describe('formatUptime', () => {
    test('formats seconds correctly', () => {
      expect(formatUptime(30)).toBe('30s');
      expect(formatUptime(59)).toBe('59s');
    });

    test('formats minutes correctly', () => {
      expect(formatUptime(60)).toBe('1m');
      expect(formatUptime(120)).toBe('2m');
      expect(formatUptime(3599)).toBe('59m');
    });

    test('formats hours and minutes correctly', () => {
      expect(formatUptime(3600)).toBe('1h 0m');
      expect(formatUptime(3660)).toBe('1h 1m');
      expect(formatUptime(7200)).toBe('2h 0m');
      expect(formatUptime(7380)).toBe('2h 3m');
    });

    test('handles zero seconds', () => {
      expect(formatUptime(0)).toBe('0s');
    });

    test('handles large values', () => {
      expect(formatUptime(86400)).toBe('24h 0m');
      expect(formatUptime(90061)).toBe('25h 1m');
    });
  });

  describe('formatNumber', () => {
    test('formats small numbers', () => {
      expect(formatNumber(0)).toBe('0');
      expect(formatNumber(100)).toBe('100');
      expect(formatNumber(999)).toBe('999');
    });

    test('formats thousands with K suffix', () => {
      expect(formatNumber(1000)).toBe('1.0K');
      expect(formatNumber(1500)).toBe('1.5K');
      expect(formatNumber(10000)).toBe('10.0K');
      expect(formatNumber(999999)).toBe('1000.0K');
    });

    test('formats millions with M suffix', () => {
      expect(formatNumber(1_000_000)).toBe('1.0M');
      expect(formatNumber(1_500_000)).toBe('1.5M');
      expect(formatNumber(10_000_000)).toBe('10.0M');
    });
  });

  describe('Progress Bar Calculation', () => {
    test('calculates filled segments correctly', () => {
      expect(calculateProgressBarFilled(0, 15)).toBe(0);
      expect(calculateProgressBarFilled(100, 15)).toBe(15);
      expect(calculateProgressBarFilled(50, 15)).toBe(8);
      expect(calculateProgressBarFilled(33, 15)).toBe(5);
    });

    test('handles edge cases', () => {
      expect(calculateProgressBarFilled(0, 0)).toBe(0);
      expect(calculateProgressBarFilled(100, 0)).toBe(0);
    });
  });

  describe('Utilization Calculation', () => {
    test('calculates percentage correctly', () => {
      expect(getUtilizationPercent(3, 10)).toBe(30);
      expect(getUtilizationPercent(5, 10)).toBe(50);
      expect(getUtilizationPercent(10, 10)).toBe(100);
    });

    test('handles zero active agents', () => {
      expect(getUtilizationPercent(0, 0)).toBe(0);
    });

    test('handles zero working agents', () => {
      expect(getUtilizationPercent(0, 10)).toBe(0);
    });
  });

  describe('Health Calculation', () => {
    test('calculates health percentage correctly', () => {
      expect(getHealthPercent(8, 10)).toBe(80);
      expect(getHealthPercent(10, 10)).toBe(100);
      expect(getHealthPercent(5, 10)).toBe(50);
    });

    test('handles zero total agents', () => {
      expect(getHealthPercent(0, 0)).toBe(100);
    });

    test('handles zero healthy agents', () => {
      expect(getHealthPercent(0, 10)).toBe(0);
    });
  });

  describe('Metric Color Logic', () => {
    describe('average metric colors', () => {
      test('green for fast responses (<16ms)', () => {
        expect(getMetricColor(5, 'avg')).toBe('green');
        expect(getMetricColor(15, 'avg')).toBe('green');
      });

      test('yellow for moderate responses (16-49ms)', () => {
        expect(getMetricColor(16, 'avg')).toBe('yellow');
        expect(getMetricColor(49, 'avg')).toBe('yellow');
      });

      test('red for slow responses (>=50ms)', () => {
        expect(getMetricColor(50, 'avg')).toBe('red');
        expect(getMetricColor(100, 'avg')).toBe('red');
      });
    });

    describe('max metric colors', () => {
      test('green for fast max (<50ms)', () => {
        expect(getMetricColor(30, 'max')).toBe('green');
        expect(getMetricColor(49, 'max')).toBe('green');
      });

      test('yellow for moderate max (50-99ms)', () => {
        expect(getMetricColor(50, 'max')).toBe('yellow');
        expect(getMetricColor(99, 'max')).toBe('yellow');
      });

      test('red for slow max (>=100ms)', () => {
        expect(getMetricColor(100, 'max')).toBe('red');
        expect(getMetricColor(200, 'max')).toBe('red');
      });
    });
  });

  describe('Utilization Color Logic', () => {
    test('green for high utilization (>=80%)', () => {
      expect(getUtilizationColor(80)).toBe('green');
      expect(getUtilizationColor(100)).toBe('green');
    });

    test('yellow for moderate utilization (50-79%)', () => {
      expect(getUtilizationColor(50)).toBe('yellow');
      expect(getUtilizationColor(79)).toBe('yellow');
    });

    test('gray for low utilization (<50%)', () => {
      expect(getUtilizationColor(0)).toBe('gray');
      expect(getUtilizationColor(49)).toBe('gray');
    });
  });

  describe('Metric Name Truncation', () => {
    test('short names are not truncated', () => {
      expect(truncateMetricName('render')).toBe('render');
      expect(truncateMetricName('1234567890123456')).toBe('1234567890123456');
    });

    test('long names are truncated with ellipsis', () => {
      expect(truncateMetricName('this-is-a-very-long-metric-name')).toBe('this-is-a-very-…');
    });

    test('exactly max length is not truncated', () => {
      const maxLenName = '1234567890123456';
      expect(truncateMetricName(maxLenName, 16)).toBe(maxLenName);
    });
  });

  describe('Metric Data Structure', () => {
    test('metric has required fields', () => {
      const metric: PerformanceMetric = {
        name: 'render',
        value: 10,
        average: 12.5,
        min: 8,
        max: 25,
        count: 100,
      };

      expect(metric.name).toBe('render');
      expect(metric.average).toBeNumber();
      expect(metric.count).toBeNumber();
    });

    test('metric sorting by name', () => {
      const metrics: PerformanceMetric[] = [
        { name: 'z-metric', value: 10, average: 10, min: 5, max: 15, count: 10 },
        { name: 'a-metric', value: 10, average: 10, min: 5, max: 15, count: 10 },
        { name: 'm-metric', value: 10, average: 10, min: 5, max: 15, count: 10 },
      ];

      const sorted = [...metrics].sort((a, b) => a.name.localeCompare(b.name));
      expect(sorted[0].name).toBe('a-metric');
      expect(sorted[1].name).toBe('m-metric');
      expect(sorted[2].name).toBe('z-metric');
    });

    test('metrics can be sliced to top N', () => {
      const metrics: PerformanceMetric[] = Array(15).fill(null).map((_, i) => ({
        name: `metric-${String(i).padStart(2, '0')}`,
        value: 10,
        average: 10,
        min: 5,
        max: 15,
        count: 10,
      }));

      const top10 = metrics.slice(0, 10);
      expect(top10).toHaveLength(10);
    });
  });

  describe('Keyboard Shortcuts', () => {
    const shortcuts = {
      r: 'refresh',
      c: 'clear metrics',
      'Ctrl+P': 'toggle debug',
      q: 'back',
    };

    test('refresh shortcut', () => {
      expect(shortcuts.r).toBe('refresh');
    });

    test('clear metrics shortcut', () => {
      expect(shortcuts.c).toBe('clear metrics');
    });

    test('toggle debug shortcut', () => {
      expect(shortcuts['Ctrl+P']).toBe('toggle debug');
    });

    test('back shortcut', () => {
      expect(shortcuts.q).toBe('back');
    });
  });

  describe('Health Status Determination', () => {
    test('healthy when no stuck or error agents', () => {
      const stuckCount = 0;
      const errorCount = 0;
      const healthy = stuckCount === 0 && errorCount === 0;
      expect(healthy).toBe(true);
    });

    test('unhealthy when stuck agents exist', () => {
      const stuckCount = 2;
      const errorCount = 0;
      const healthy = stuckCount === 0 && errorCount === 0;
      expect(healthy).toBe(false);
    });

    test('unhealthy when error agents exist', () => {
      const stuckCount = 0;
      const errorCount = 1;
      const healthy = stuckCount === 0 && errorCount === 0;
      expect(healthy).toBe(false);
    });

    test('unhealthy when both stuck and error agents exist', () => {
      const stuckCount = 1;
      const errorCount = 1;
      const healthy = stuckCount === 0 && errorCount === 0;
      expect(healthy).toBe(false);
    });
  });

  describe('Agent State Counts', () => {
    test('computes unhealthy count', () => {
      const stuck = 2;
      const error = 1;
      const unhealthyCount = stuck + error;
      expect(unhealthyCount).toBe(3);
    });

    test('computes healthy count', () => {
      const working = 5;
      const idle = 3;
      const healthyCount = working + idle;
      expect(healthyCount).toBe(8);
    });

    test('attention message when unhealthy', () => {
      const unhealthyCount = 3;
      const message = unhealthyCount > 0
        ? `⚠ ${unhealthyCount} agent${unhealthyCount > 1 ? 's' : ''} need attention`
        : '';
      expect(message).toBe('⚠ 3 agents need attention');
    });

    test('singular message for one agent', () => {
      const unhealthyCount = 1;
      const message = unhealthyCount > 0
        ? `⚠ ${unhealthyCount} agent${unhealthyCount > 1 ? 's' : ''} need attention`
        : '';
      expect(message).toBe('⚠ 1 agent need attention');
    });
  });

  describe('Cost Display Formatting', () => {
    test('formats cost with 4 decimal places', () => {
      const cost = 0.1234;
      const formatted = `$${cost.toFixed(4)}`;
      expect(formatted).toBe('$0.1234');
    });

    test('formats whole number cost', () => {
      const cost = 5;
      const formatted = `$${cost.toFixed(4)}`;
      expect(formatted).toBe('$5.0000');
    });

    test('formats large cost', () => {
      const cost = 123.4567;
      const formatted = `$${cost.toFixed(4)}`;
      expect(formatted).toBe('$123.4567');
    });
  });

  describe('Token Count Display', () => {
    test('computes total tokens', () => {
      const inputTokens = 5000;
      const outputTokens = 2000;
      const totalTokens = inputTokens + outputTokens;
      expect(totalTokens).toBe(7000);
    });

    test('formats token counts with K suffix', () => {
      const inputTokens = 5000;
      const outputTokens = 2000;
      expect(formatNumber(inputTokens)).toBe('5.0K');
      expect(formatNumber(outputTokens)).toBe('2.0K');
    });
  });
});
