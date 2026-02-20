/**
 * useCostTrends hook tests
 * Issue #1047 - Activity timeline and cost trend tracking
 */

import { describe, it, expect } from 'bun:test';
import { calculateTrend } from '../../hooks/useCostTrends';

describe('calculateTrend', () => {
  it('returns flat trend when previous is zero', () => {
    const result = calculateTrend(100, 0);
    expect(result.trend).toBe('flat');
    expect(result.symbol).toBe('→');
    expect(result.change).toBe(0);
  });

  it('returns up trend for significant increase', () => {
    const result = calculateTrend(150, 100);
    expect(result.trend).toBe('up');
    expect(result.symbol).toBe('↗');
    expect(result.change).toBe(50);
  });

  it('returns down trend for significant decrease', () => {
    const result = calculateTrend(50, 100);
    expect(result.trend).toBe('down');
    expect(result.symbol).toBe('↘');
    expect(result.change).toBe(-50);
  });

  it('returns flat trend for small changes (< 5%)', () => {
    const result = calculateTrend(102, 100);
    expect(result.trend).toBe('flat');
    expect(result.symbol).toBe('→');
  });

  it('handles edge case of exactly 5% change', () => {
    const result = calculateTrend(105, 100);
    // 5% is the threshold, so >= 5% should be up
    expect(result.trend).toBe('up');
  });

  it('calculates correct percentage change', () => {
    const result = calculateTrend(200, 100);
    expect(result.change).toBe(100); // 100% increase
  });

  it('handles decimal values', () => {
    const result = calculateTrend(5.5, 5.0);
    expect(result.trend).toBe('up');
    expect(result.change).toBe(10); // 10% increase
  });
});

describe('budget status calculation logic', () => {
  function getDaysRemaining(period: 'day' | 'week' | 'month'): number {
    const now = new Date();
    switch (period) {
      case 'day':
        return 1;
      case 'week': {
        const dayOfWeek = now.getDay();
        return 7 - dayOfWeek;
      }
      case 'month': {
        const lastDay = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
        return lastDay - now.getDate();
      }
    }
  }

  function getDaysElapsed(period: 'day' | 'week' | 'month'): number {
    const now = new Date();
    switch (period) {
      case 'day':
        return 1;
      case 'week':
        return now.getDay() || 7;
      case 'month':
        return now.getDate();
    }
  }

  it('day period has 1 day remaining', () => {
    expect(getDaysRemaining('day')).toBe(1);
  });

  it('week period has <= 7 days remaining', () => {
    const remaining = getDaysRemaining('week');
    expect(remaining).toBeGreaterThanOrEqual(0);
    expect(remaining).toBeLessThanOrEqual(7);
  });

  it('month period has reasonable days remaining', () => {
    const remaining = getDaysRemaining('month');
    expect(remaining).toBeGreaterThanOrEqual(0);
    expect(remaining).toBeLessThanOrEqual(31);
  });

  it('day period has 1 day elapsed', () => {
    expect(getDaysElapsed('day')).toBe(1);
  });

  it('week period has 1-7 days elapsed', () => {
    const elapsed = getDaysElapsed('week');
    expect(elapsed).toBeGreaterThanOrEqual(1);
    expect(elapsed).toBeLessThanOrEqual(7);
  });

  it('month period has >= 1 days elapsed', () => {
    const elapsed = getDaysElapsed('month');
    expect(elapsed).toBeGreaterThanOrEqual(1);
    expect(elapsed).toBeLessThanOrEqual(31);
  });
});

describe('budget status thresholds', () => {
  function getStatus(spent: number, budget: number, projectedTotal: number): 'normal' | 'warning' | 'critical' {
    const percentUsed = budget > 0 ? Math.round((spent / budget) * 100) : 0;

    if (projectedTotal > budget * 1.2 || percentUsed > 90) {
      return 'critical';
    } else if (projectedTotal > budget * 0.9 || percentUsed > 70) {
      return 'warning';
    }
    return 'normal';
  }

  it('returns normal for low spend', () => {
    expect(getStatus(2, 10, 4)).toBe('normal');
  });

  it('returns warning when projected > 90% of budget', () => {
    expect(getStatus(5, 10, 9.5)).toBe('warning');
  });

  it('returns warning when spent > 70% of budget', () => {
    expect(getStatus(7.5, 10, 7.5)).toBe('warning');
  });

  it('returns critical when projected > 120% of budget', () => {
    expect(getStatus(5, 10, 13)).toBe('critical');
  });

  it('returns critical when spent > 90% of budget', () => {
    expect(getStatus(9.5, 10, 9.5)).toBe('critical');
  });
});
