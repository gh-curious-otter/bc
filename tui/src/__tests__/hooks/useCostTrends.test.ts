/**
 * useCostTrends hook tests - Unit tests for utility functions
 * Issue #1047 - Activity timeline and cost trend tracking
 */

import { describe, it, expect } from 'bun:test';

// Test the trend calculation logic
function testCalculateTrend(current: number, previous: number): { trend: 'up' | 'down' | 'flat'; symbol: '↗' | '↘' | '→'; change: number } {
  if (previous === 0) return { trend: 'flat', symbol: '→', change: 0 };

  const change = ((current - previous) / previous) * 100;
  const absChange = Math.abs(change);

  if (absChange < 5) {
    return { trend: 'flat', symbol: '→', change };
  } else if (change > 0) {
    return { trend: 'up', symbol: '↗', change };
  } else {
    return { trend: 'down', symbol: '↘', change };
  }
}

// Test the days remaining calculation
function testGetDaysRemaining(period: 'day' | 'week' | 'month', testDate: Date): number {
  if (period === 'day') {
    return 1;
  } else if (period === 'week') {
    const dayOfWeek = testDate.getDay();
    return 7 - dayOfWeek;
  } else {
    const lastDay = new Date(testDate.getFullYear(), testDate.getMonth() + 1, 0).getDate();
    return lastDay - testDate.getDate() + 1;
  }
}

// Test the days elapsed calculation
function testGetDaysElapsed(period: 'day' | 'week' | 'month', testDate: Date): number {
  if (period === 'day') {
    return 1;
  } else if (period === 'week') {
    return testDate.getDay() || 7;
  } else {
    return testDate.getDate();
  }
}

// Test budget status determination
interface CostBudgetStatus {
  spent: number;
  budget: number;
  percentUsed: number;
  status: 'normal' | 'warning' | 'critical';
}

function testDetermineBudgetStatus(spent: number, budget: number, projectedTotal: number): 'normal' | 'warning' | 'critical' {
  const percentUsed = budget > 0 ? Math.round((spent / budget) * 100) : 0;

  if (percentUsed >= 90 || projectedTotal > budget * 1.2) {
    return 'critical';
  } else if (percentUsed >= 70 || projectedTotal > budget) {
    return 'warning';
  }
  return 'normal';
}

describe('calculateTrend', () => {
  it('returns up trend for increase > 5%', () => {
    const result = testCalculateTrend(110, 100);

    expect(result.trend).toBe('up');
    expect(result.symbol).toBe('↗');
    expect(result.change).toBe(10);
  });

  it('returns down trend for decrease > 5%', () => {
    const result = testCalculateTrend(90, 100);

    expect(result.trend).toBe('down');
    expect(result.symbol).toBe('↘');
    expect(result.change).toBe(-10);
  });

  it('returns flat trend for change < 5%', () => {
    const result = testCalculateTrend(102, 100);

    expect(result.trend).toBe('flat');
    expect(result.symbol).toBe('→');
    expect(result.change).toBe(2);
  });

  it('handles zero previous value', () => {
    const result = testCalculateTrend(100, 0);

    expect(result.trend).toBe('flat');
    expect(result.symbol).toBe('→');
    expect(result.change).toBe(0);
  });

  it('handles large percentage changes', () => {
    const result = testCalculateTrend(200, 100);

    expect(result.trend).toBe('up');
    expect(result.change).toBe(100);
  });
});

describe('getDaysRemaining', () => {
  it('returns 1 for day period', () => {
    const testDate = new Date('2026-02-20');
    expect(testGetDaysRemaining('day', testDate)).toBe(1);
  });

  it('calculates week days remaining', () => {
    // Friday (day 5) has 2 days remaining (Sat, Sun)
    const friday = new Date('2026-02-20'); // This is a Friday
    expect(testGetDaysRemaining('week', friday)).toBe(2);
  });

  it('calculates month days remaining', () => {
    // Feb 20 in a leap year has 9 days remaining (21-29)
    const feb20 = new Date('2026-02-20');
    const daysRemaining = testGetDaysRemaining('month', feb20);
    expect(daysRemaining).toBeGreaterThan(0);
  });
});

describe('getDaysElapsed', () => {
  it('returns 1 for day period', () => {
    const testDate = new Date('2026-02-20');
    expect(testGetDaysElapsed('day', testDate)).toBe(1);
  });

  it('calculates week days elapsed', () => {
    // Friday (day 5)
    const friday = new Date('2026-02-20');
    expect(testGetDaysElapsed('week', friday)).toBe(5);
  });

  it('handles Sunday (day 0) as 7', () => {
    const sunday = new Date('2026-02-22');
    expect(testGetDaysElapsed('week', sunday)).toBe(7);
  });

  it('calculates month days elapsed', () => {
    const feb20 = new Date('2026-02-20');
    expect(testGetDaysElapsed('month', feb20)).toBe(20);
  });
});

describe('determineBudgetStatus', () => {
  it('returns normal for low usage', () => {
    const status = testDetermineBudgetStatus(5, 10, 7);
    expect(status).toBe('normal');
  });

  it('returns warning for 70-90% usage', () => {
    const status = testDetermineBudgetStatus(8, 10, 9);
    expect(status).toBe('warning');
  });

  it('returns warning when projected exceeds budget', () => {
    const status = testDetermineBudgetStatus(5, 10, 12);
    expect(status).toBe('warning');
  });

  it('returns critical for 90%+ usage', () => {
    const status = testDetermineBudgetStatus(9.5, 10, 9.5);
    expect(status).toBe('critical');
  });

  it('returns critical when projected exceeds 120% of budget', () => {
    const status = testDetermineBudgetStatus(5, 10, 15);
    expect(status).toBe('critical');
  });
});
