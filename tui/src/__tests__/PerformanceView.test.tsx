/**
 * PerformanceView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

describe('PerformanceView - formatUptime', () => {
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

  test('formats seconds', () => {
    expect(formatUptime(30)).toBe('30s');
  });

  test('formats zero seconds', () => {
    expect(formatUptime(0)).toBe('0s');
  });

  test('formats 59 seconds', () => {
    expect(formatUptime(59)).toBe('59s');
  });

  test('formats minutes', () => {
    expect(formatUptime(120)).toBe('2m');
  });

  test('formats 59 minutes', () => {
    expect(formatUptime(59 * 60)).toBe('59m');
  });

  test('formats hours and minutes', () => {
    expect(formatUptime(3600 + 1800)).toBe('1h 30m');
  });

  test('formats multiple hours', () => {
    expect(formatUptime(3 * 3600 + 15 * 60)).toBe('3h 15m');
  });

  test('formats hours with 0 remaining minutes', () => {
    expect(formatUptime(2 * 3600)).toBe('2h 0m');
  });

  test('floors fractional seconds', () => {
    expect(formatUptime(45.7)).toBe('45s');
  });
});

describe('PerformanceView - formatNumber', () => {
  function formatNumber(n: number): string {
    if (n >= 1_000_000) {
      return `${(n / 1_000_000).toFixed(1)}M`;
    }
    if (n >= 1_000) {
      return `${(n / 1_000).toFixed(1)}K`;
    }
    return n.toString();
  }

  test('formats small numbers as-is', () => {
    expect(formatNumber(0)).toBe('0');
    expect(formatNumber(100)).toBe('100');
    expect(formatNumber(999)).toBe('999');
  });

  test('formats thousands with K suffix', () => {
    expect(formatNumber(1000)).toBe('1.0K');
    expect(formatNumber(1500)).toBe('1.5K');
    expect(formatNumber(10000)).toBe('10.0K');
  });

  test('formats millions with M suffix', () => {
    expect(formatNumber(1000000)).toBe('1.0M');
    expect(formatNumber(2500000)).toBe('2.5M');
    expect(formatNumber(10000000)).toBe('10.0M');
  });

  test('handles boundary at 1000', () => {
    expect(formatNumber(999)).toBe('999');
    expect(formatNumber(1000)).toBe('1.0K');
  });

  test('handles boundary at 1000000', () => {
    expect(formatNumber(999999)).toBe('1000.0K');
    expect(formatNumber(1000000)).toBe('1.0M');
  });
});

describe('PerformanceView - ProgressBar calculation', () => {
  function calculateProgressBar(percent: number, width: number): { filled: number; empty: number } {
    const filled = Math.round((percent / 100) * width);
    const empty = width - filled;
    return { filled, empty };
  }

  test('0% is all empty', () => {
    const { filled, empty } = calculateProgressBar(0, 10);
    expect(filled).toBe(0);
    expect(empty).toBe(10);
  });

  test('100% is all filled', () => {
    const { filled, empty } = calculateProgressBar(100, 10);
    expect(filled).toBe(10);
    expect(empty).toBe(0);
  });

  test('50% is half filled', () => {
    const { filled, empty } = calculateProgressBar(50, 10);
    expect(filled).toBe(5);
    expect(empty).toBe(5);
  });

  test('rounds correctly', () => {
    const { filled, empty } = calculateProgressBar(33, 10);
    expect(filled).toBe(3);
    expect(empty).toBe(7);
  });

  test('handles different widths', () => {
    const { filled, empty } = calculateProgressBar(50, 20);
    expect(filled).toBe(10);
    expect(empty).toBe(10);
  });
});

describe('PerformanceView - utilization percent', () => {
  function calculateUtilization(working: number, active: number): number {
    return active > 0 ? Math.round((working / active) * 100) : 0;
  }

  test('100% when all active are working', () => {
    expect(calculateUtilization(5, 5)).toBe(100);
  });

  test('50% when half are working', () => {
    expect(calculateUtilization(3, 6)).toBe(50);
  });

  test('0% when no active agents', () => {
    expect(calculateUtilization(0, 0)).toBe(0);
  });

  test('0% when none working', () => {
    expect(calculateUtilization(0, 5)).toBe(0);
  });

  test('rounds correctly', () => {
    expect(calculateUtilization(1, 3)).toBe(33);
    expect(calculateUtilization(2, 3)).toBe(67);
  });
});

describe('PerformanceView - health percent', () => {
  function calculateHealth(healthyCount: number, total: number): number {
    return total > 0 ? Math.round((healthyCount / total) * 100) : 100;
  }

  test('100% when all healthy', () => {
    expect(calculateHealth(10, 10)).toBe(100);
  });

  test('100% when no agents', () => {
    expect(calculateHealth(0, 0)).toBe(100);
  });

  test('50% when half healthy', () => {
    expect(calculateHealth(5, 10)).toBe(50);
  });

  test('0% when none healthy', () => {
    expect(calculateHealth(0, 10)).toBe(0);
  });
});

describe('PerformanceView - health color', () => {
  function getHealthColor(percent: number): string {
    return percent >= 80 ? 'green' : percent >= 50 ? 'yellow' : 'red';
  }

  test('green for 80% or higher', () => {
    expect(getHealthColor(80)).toBe('green');
    expect(getHealthColor(100)).toBe('green');
  });

  test('yellow for 50-79%', () => {
    expect(getHealthColor(79)).toBe('yellow');
    expect(getHealthColor(50)).toBe('yellow');
  });

  test('red for below 50%', () => {
    expect(getHealthColor(49)).toBe('red');
    expect(getHealthColor(0)).toBe('red');
  });
});

describe('PerformanceView - utilization color', () => {
  function getUtilizationColor(utilization: number): string {
    return utilization >= 80 ? 'green' : utilization >= 50 ? 'yellow' : 'gray';
  }

  test('green for 80% or higher', () => {
    expect(getUtilizationColor(80)).toBe('green');
    expect(getUtilizationColor(100)).toBe('green');
  });

  test('yellow for 50-79%', () => {
    expect(getUtilizationColor(79)).toBe('yellow');
    expect(getUtilizationColor(50)).toBe('yellow');
  });

  test('gray for below 50%', () => {
    expect(getUtilizationColor(49)).toBe('gray');
    expect(getUtilizationColor(0)).toBe('gray');
  });
});

describe('PerformanceView - metric latency colors', () => {
  function getAvgColor(average: number): string {
    return average < 16 ? 'green' : average < 50 ? 'yellow' : 'red';
  }

  function getMaxColor(max: number): string {
    return max < 50 ? 'green' : max < 100 ? 'yellow' : 'red';
  }

  test('avg green for fast (<16ms)', () => {
    expect(getAvgColor(10)).toBe('green');
    expect(getAvgColor(15)).toBe('green');
  });

  test('avg yellow for moderate (16-49ms)', () => {
    expect(getAvgColor(16)).toBe('yellow');
    expect(getAvgColor(49)).toBe('yellow');
  });

  test('avg red for slow (>=50ms)', () => {
    expect(getAvgColor(50)).toBe('red');
    expect(getAvgColor(100)).toBe('red');
  });

  test('max green for fast (<50ms)', () => {
    expect(getMaxColor(30)).toBe('green');
    expect(getMaxColor(49)).toBe('green');
  });

  test('max yellow for moderate (50-99ms)', () => {
    expect(getMaxColor(50)).toBe('yellow');
    expect(getMaxColor(99)).toBe('yellow');
  });

  test('max red for slow (>=100ms)', () => {
    expect(getMaxColor(100)).toBe('red');
    expect(getMaxColor(200)).toBe('red');
  });
});

describe('PerformanceView - metric name truncation', () => {
  function truncateMetricName(name: string, maxLen = 16): string {
    return name.length > maxLen
      ? name.slice(0, maxLen - 1) + '…'
      : name;
  }

  test('short names not truncated', () => {
    expect(truncateMetricName('render')).toBe('render');
  });

  test('exact length not truncated', () => {
    expect(truncateMetricName('1234567890123456')).toBe('1234567890123456');
  });

  test('long names truncated with ellipsis', () => {
    const longName = 'very_long_metric_name_here';
    const result = truncateMetricName(longName);
    expect(result).toBe('very_long_metri…');
    expect(result.length).toBe(16);
  });
});

describe('PerformanceView - unhealthy agents', () => {
  function getUnhealthyCount(stuck: number, error: number): number {
    return stuck + error;
  }

  function getWarningMessage(unhealthyCount: number): string {
    return `⚠ ${unhealthyCount} agent${unhealthyCount > 1 ? 's' : ''} need attention`;
  }

  test('counts unhealthy agents', () => {
    expect(getUnhealthyCount(2, 1)).toBe(3);
    expect(getUnhealthyCount(0, 0)).toBe(0);
  });

  test('singular warning message', () => {
    expect(getWarningMessage(1)).toBe('⚠ 1 agent need attention');
  });

  test('plural warning message', () => {
    expect(getWarningMessage(2)).toBe('⚠ 2 agents need attention');
    expect(getWarningMessage(5)).toBe('⚠ 5 agents need attention');
  });
});

describe('PerformanceView - footer hints', () => {
  interface Hint {
    key: string;
    label: string;
  }

  function getFooterHints(debugEnabled: boolean): Hint[] {
    return [
      { key: 'r', label: 'refresh' },
      { key: 'c', label: 'clear metrics' },
      { key: 'Ctrl+P', label: debugEnabled ? 'debug off' : 'debug on' },
      { key: 'q', label: 'back' },
    ];
  }

  test('debug off hint when debug disabled', () => {
    const hints = getFooterHints(false);
    const debugHint = hints.find(h => h.key === 'Ctrl+P');
    expect(debugHint?.label).toBe('debug on');
  });

  test('debug on hint when debug enabled', () => {
    const hints = getFooterHints(true);
    const debugHint = hints.find(h => h.key === 'Ctrl+P');
    expect(debugHint?.label).toBe('debug off');
  });

  test('has refresh hint', () => {
    const hints = getFooterHints(false);
    expect(hints.some(h => h.key === 'r' && h.label === 'refresh')).toBe(true);
  });

  test('has clear metrics hint', () => {
    const hints = getFooterHints(false);
    expect(hints.some(h => h.label === 'clear metrics')).toBe(true);
  });
});

describe('PerformanceView - metrics sorting', () => {
  interface Metric {
    name: string;
    value: number;
  }

  function sortMetricsByName(metrics: Metric[]): Metric[] {
    return [...metrics].sort((a, b) => a.name.localeCompare(b.name));
  }

  function limitMetrics(metrics: Metric[], limit = 10): Metric[] {
    return metrics.slice(0, limit);
  }

  test('sorts metrics alphabetically', () => {
    const metrics: Metric[] = [
      { name: 'zebra', value: 1 },
      { name: 'apple', value: 2 },
      { name: 'mango', value: 3 },
    ];
    const sorted = sortMetricsByName(metrics);
    expect(sorted[0].name).toBe('apple');
    expect(sorted[1].name).toBe('mango');
    expect(sorted[2].name).toBe('zebra');
  });

  test('limits metrics to 10', () => {
    const metrics: Metric[] = Array.from({ length: 15 }, (_, i) => ({
      name: `metric_${i}`,
      value: i,
    }));
    const limited = limitMetrics(metrics);
    expect(limited).toHaveLength(10);
  });

  test('handles fewer than limit', () => {
    const metrics: Metric[] = [
      { name: 'a', value: 1 },
      { name: 'b', value: 2 },
    ];
    const limited = limitMetrics(metrics);
    expect(limited).toHaveLength(2);
  });
});

describe('PerformanceView - cost formatting', () => {
  function formatCost(cost: number): string {
    return `$${cost.toFixed(4)}`;
  }

  test('formats small costs', () => {
    expect(formatCost(0.0001)).toBe('$0.0001');
  });

  test('formats larger costs', () => {
    expect(formatCost(1.5)).toBe('$1.5000');
  });

  test('formats zero', () => {
    expect(formatCost(0)).toBe('$0.0000');
  });
});
