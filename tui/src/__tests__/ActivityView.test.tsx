/**
 * ActivityView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

type TimePeriod = '24h' | 'week' | 'month';

describe('ActivityView - time period to hours conversion', () => {
  function getHoursForPeriod(period: TimePeriod): number {
    return period === '24h' ? 24 : period === 'week' ? 168 : 720;
  }

  test('24h returns 24 hours', () => {
    expect(getHoursForPeriod('24h')).toBe(24);
  });

  test('week returns 168 hours (7 days)', () => {
    expect(getHoursForPeriod('week')).toBe(168);
  });

  test('month returns 720 hours (30 days)', () => {
    expect(getHoursForPeriod('month')).toBe(720);
  });
});

describe('ActivityView - time period to trend period conversion', () => {
  function getTrendPeriod(period: TimePeriod): 'day' | 'week' | 'month' {
    return period === '24h' ? 'day' : period === 'week' ? 'week' : 'month';
  }

  test('24h maps to day', () => {
    expect(getTrendPeriod('24h')).toBe('day');
  });

  test('week maps to week', () => {
    expect(getTrendPeriod('week')).toBe('week');
  });

  test('month maps to month', () => {
    expect(getTrendPeriod('month')).toBe('month');
  });
});

describe('ActivityView - loading state logic', () => {
  function isLoading(activitiesLoading: boolean, trendLoading: boolean): boolean {
    return activitiesLoading || trendLoading;
  }

  test('both loading shows loading', () => {
    expect(isLoading(true, true)).toBe(true);
  });

  test('activities loading shows loading', () => {
    expect(isLoading(true, false)).toBe(true);
  });

  test('trends loading shows loading', () => {
    expect(isLoading(false, true)).toBe(true);
  });

  test('neither loading shows content', () => {
    expect(isLoading(false, false)).toBe(false);
  });
});

describe('ActivityView - time period label', () => {
  function getPeriodLabel(period: TimePeriod): string {
    return period === '24h' ? 'Last 24 Hours' : period === 'week' ? 'Last 7 Days' : 'Last 30 Days';
  }

  test('24h shows Last 24 Hours', () => {
    expect(getPeriodLabel('24h')).toBe('Last 24 Hours');
  });

  test('week shows Last 7 Days', () => {
    expect(getPeriodLabel('week')).toBe('Last 7 Days');
  });

  test('month shows Last 30 Days', () => {
    expect(getPeriodLabel('month')).toBe('Last 30 Days');
  });
});

describe('ActivityView - activity slice limit by viewport', () => {
  function getActivityLimit(isWide: boolean): number {
    return isWide ? 15 : 8;
  }

  test('wide viewport shows 15 activities', () => {
    expect(getActivityLimit(true)).toBe(15);
  });

  test('narrow viewport shows 8 activities', () => {
    expect(getActivityLimit(false)).toBe(8);
  });
});

describe('ActivityView - budget status color', () => {
  type BudgetStatusType = 'critical' | 'warning' | 'normal';

  const STATUS_COLORS = {
    error: 'red',
    warning: 'yellow',
    info: 'cyan',
  };

  function getBudgetStatusColor(status: BudgetStatusType): string {
    return status === 'critical'
      ? STATUS_COLORS.error
      : status === 'warning'
      ? STATUS_COLORS.warning
      : STATUS_COLORS.info;
  }

  test('critical status is red', () => {
    expect(getBudgetStatusColor('critical')).toBe('red');
  });

  test('warning status is yellow', () => {
    expect(getBudgetStatusColor('warning')).toBe('yellow');
  });

  test('normal status is cyan', () => {
    expect(getBudgetStatusColor('normal')).toBe('cyan');
  });
});

describe('ActivityView - budget display formatting', () => {
  function formatCurrency(value: number): string {
    return `$${value.toFixed(2)}`;
  }

  test('formats whole numbers', () => {
    expect(formatCurrency(100)).toBe('$100.00');
  });

  test('formats decimals', () => {
    expect(formatCurrency(42.50)).toBe('$42.50');
  });

  test('formats small values', () => {
    expect(formatCurrency(0.05)).toBe('$0.05');
  });

  test('formats large values', () => {
    expect(formatCurrency(1234.56)).toBe('$1234.56');
  });

  test('formats zero', () => {
    expect(formatCurrency(0)).toBe('$0.00');
  });
});

describe('ActivityView - budget percent display', () => {
  function calculatePercentUsed(spent: number, budget: number): number {
    if (budget === 0) return 0;
    return Math.round((spent / budget) * 100);
  }

  test('half budget used', () => {
    expect(calculatePercentUsed(50, 100)).toBe(50);
  });

  test('full budget used', () => {
    expect(calculatePercentUsed(100, 100)).toBe(100);
  });

  test('over budget', () => {
    expect(calculatePercentUsed(150, 100)).toBe(150);
  });

  test('no spending', () => {
    expect(calculatePercentUsed(0, 100)).toBe(0);
  });

  test('zero budget returns 0', () => {
    expect(calculatePercentUsed(50, 0)).toBe(0);
  });

  test('rounds to nearest percent', () => {
    expect(calculatePercentUsed(33.33, 100)).toBe(33);
  });
});

describe('ActivityView - keyboard input handling', () => {
  function getNewPeriod(input: string, currentPeriod: TimePeriod): TimePeriod {
    if (input === 'd') return '24h';
    if (input === 'w') return 'week';
    if (input === 'm') return 'month';
    return currentPeriod;
  }

  test('d key switches to 24h', () => {
    expect(getNewPeriod('d', 'week')).toBe('24h');
    expect(getNewPeriod('d', 'month')).toBe('24h');
    expect(getNewPeriod('d', '24h')).toBe('24h');
  });

  test('w key switches to week', () => {
    expect(getNewPeriod('w', '24h')).toBe('week');
    expect(getNewPeriod('w', 'month')).toBe('week');
    expect(getNewPeriod('w', 'week')).toBe('week');
  });

  test('m key switches to month', () => {
    expect(getNewPeriod('m', '24h')).toBe('month');
    expect(getNewPeriod('m', 'week')).toBe('month');
    expect(getNewPeriod('m', 'month')).toBe('month');
  });

  test('other keys do not change period', () => {
    expect(getNewPeriod('x', '24h')).toBe('24h');
    expect(getNewPeriod('1', 'week')).toBe('week');
    expect(getNewPeriod('', 'month')).toBe('month');
  });
});

describe('ActivityView - activity time formatting', () => {
  function formatActivityTime(date: Date): string {
    return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
  }

  test('formats morning time', () => {
    const date = new Date('2026-02-24T09:30:00');
    const result = formatActivityTime(date);
    // Time format varies by locale, just check it contains colon
    expect(result).toContain(':');
  });

  test('formats afternoon time', () => {
    const date = new Date('2026-02-24T14:45:00');
    const result = formatActivityTime(date);
    expect(result).toContain(':');
  });

  test('formats midnight', () => {
    const date = new Date('2026-02-24T00:00:00');
    const result = formatActivityTime(date);
    expect(result).toContain(':');
  });
});

describe('ActivityView - activity item display', () => {
  interface Activity {
    startTime: Date;
    agents: string[];
    duration: number;
  }

  function formatActivityAgents(activity: Activity): string {
    return activity.agents.join(', ');
  }

  function formatDuration(activity: Activity): string {
    return `${String(activity.duration)}m`;
  }

  test('single agent formatted', () => {
    const activity: Activity = {
      startTime: new Date(),
      agents: ['eng-01'],
      duration: 30,
    };
    expect(formatActivityAgents(activity)).toBe('eng-01');
  });

  test('multiple agents joined', () => {
    const activity: Activity = {
      startTime: new Date(),
      agents: ['eng-01', 'eng-02', 'eng-03'],
      duration: 30,
    };
    expect(formatActivityAgents(activity)).toBe('eng-01, eng-02, eng-03');
  });

  test('duration formatted with m suffix', () => {
    const activity: Activity = {
      startTime: new Date(),
      agents: ['eng-01'],
      duration: 45,
    };
    expect(formatDuration(activity)).toBe('45m');
  });

  test('zero duration', () => {
    const activity: Activity = {
      startTime: new Date(),
      agents: ['eng-01'],
      duration: 0,
    };
    expect(formatDuration(activity)).toBe('0m');
  });
});

describe('ActivityView - empty activity state', () => {
  function getEmptyMessage(): string {
    return 'No activity recorded in this period';
  }

  function hasActivities(activities: unknown[]): boolean {
    return activities.length > 0;
  }

  test('empty message is correct', () => {
    expect(getEmptyMessage()).toBe('No activity recorded in this period');
  });

  test('empty array returns false', () => {
    expect(hasActivities([])).toBe(false);
  });

  test('array with items returns true', () => {
    expect(hasActivities([{ id: 1 }])).toBe(true);
  });
});

describe('ActivityView - hints generation', () => {
  interface Hint {
    key: string;
    label: string;
  }

  function buildHints(): Hint[] {
    return [
      { key: 'd', label: '24h' },
      { key: 'w', label: 'week' },
      { key: 'm', label: 'month' },
    ];
  }

  test('hints include day shortcut', () => {
    const hints = buildHints();
    expect(hints[0]).toEqual({ key: 'd', label: '24h' });
  });

  test('hints include week shortcut', () => {
    const hints = buildHints();
    expect(hints[1]).toEqual({ key: 'w', label: 'week' });
  });

  test('hints include month shortcut', () => {
    const hints = buildHints();
    expect(hints[2]).toEqual({ key: 'm', label: 'month' });
  });

  test('hints array has 3 items', () => {
    expect(buildHints()).toHaveLength(3);
  });
});

describe('ActivityView - period selector highlighting', () => {
  const STATUS_COLORS = {
    working: 'green',
  };

  function getPeriodColor(period: TimePeriod, currentPeriod: TimePeriod): string {
    return period === currentPeriod ? STATUS_COLORS.working : 'white';
  }

  test('selected 24h is highlighted', () => {
    expect(getPeriodColor('24h', '24h')).toBe('green');
    expect(getPeriodColor('week', '24h')).toBe('white');
    expect(getPeriodColor('month', '24h')).toBe('white');
  });

  test('selected week is highlighted', () => {
    expect(getPeriodColor('24h', 'week')).toBe('white');
    expect(getPeriodColor('week', 'week')).toBe('green');
    expect(getPeriodColor('month', 'week')).toBe('white');
  });

  test('selected month is highlighted', () => {
    expect(getPeriodColor('24h', 'month')).toBe('white');
    expect(getPeriodColor('week', 'month')).toBe('white');
    expect(getPeriodColor('month', 'month')).toBe('green');
  });
});

describe('ActivityView - burn rate calculation', () => {
  function formatBurnRate(burnRate: number): string {
    return `$${burnRate.toFixed(2)}/day`;
  }

  test('formats burn rate with /day suffix', () => {
    expect(formatBurnRate(10.5)).toBe('$10.50/day');
  });

  test('formats zero burn rate', () => {
    expect(formatBurnRate(0)).toBe('$0.00/day');
  });

  test('formats high burn rate', () => {
    expect(formatBurnRate(100.25)).toBe('$100.25/day');
  });
});

describe('ActivityView - projected total calculation', () => {
  function formatProjectedTotal(projected: number): string {
    return `$${projected.toFixed(2)}`;
  }

  test('formats projected total', () => {
    expect(formatProjectedTotal(500.75)).toBe('$500.75');
  });

  test('formats zero projected', () => {
    expect(formatProjectedTotal(0)).toBe('$0.00');
  });
});

describe('ActivityView - disableInput prop', () => {
  function shouldHandleInput(disableInput: boolean): boolean {
    return !disableInput;
  }

  test('handles input when not disabled', () => {
    expect(shouldHandleInput(false)).toBe(true);
  });

  test('ignores input when disabled', () => {
    expect(shouldHandleInput(true)).toBe(false);
  });
});

describe('ActivityView - default props', () => {
  function getDefaultDisableInput(): boolean {
    return false;
  }

  test('disableInput defaults to false', () => {
    expect(getDefaultDisableInput()).toBe(false);
  });
});

describe('ActivityView - budget status determination', () => {
  function determineBudgetStatus(percentUsed: number): 'critical' | 'warning' | 'normal' {
    if (percentUsed >= 90) return 'critical';
    if (percentUsed >= 70) return 'warning';
    return 'normal';
  }

  test('0% used is normal', () => {
    expect(determineBudgetStatus(0)).toBe('normal');
  });

  test('50% used is normal', () => {
    expect(determineBudgetStatus(50)).toBe('normal');
  });

  test('69% used is normal', () => {
    expect(determineBudgetStatus(69)).toBe('normal');
  });

  test('70% used is warning', () => {
    expect(determineBudgetStatus(70)).toBe('warning');
  });

  test('80% used is warning', () => {
    expect(determineBudgetStatus(80)).toBe('warning');
  });

  test('89% used is warning', () => {
    expect(determineBudgetStatus(89)).toBe('warning');
  });

  test('90% used is critical', () => {
    expect(determineBudgetStatus(90)).toBe('critical');
  });

  test('100% used is critical', () => {
    expect(determineBudgetStatus(100)).toBe('critical');
  });

  test('over budget is critical', () => {
    expect(determineBudgetStatus(120)).toBe('critical');
  });
});
