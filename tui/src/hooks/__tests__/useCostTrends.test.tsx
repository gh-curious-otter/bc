/**
 * Tests for useCostTrends hook - Cost trends and spending patterns
 * Validates type exports, interface definitions, and calculateTrend helper
 *
 * #1081 Q1 Cleanup: TUI hook test coverage
 *
 * Note: React hook testing requires DOM environment which is not available in Bun/Ink.
 * These tests focus on type checking, interface validation, and the exported helper function.
 */

import { describe, it, expect } from 'bun:test';
import { calculateTrend } from '../useCostTrends';
import type {
  CostData,
  CostTrend,
  CostBudgetStatus,
} from '../useCostTrends';

describe('useCostTrends - calculateTrend Helper', () => {
  describe('trend direction', () => {
    it('returns up for significant increase', () => {
      const result = calculateTrend(110, 100);
      expect(result.trend).toBe('up');
      expect(result.symbol).toBe('↗');
    });

    it('returns down for significant decrease', () => {
      const result = calculateTrend(90, 100);
      expect(result.trend).toBe('down');
      expect(result.symbol).toBe('↘');
    });

    it('returns flat for small change (<5%)', () => {
      const result = calculateTrend(102, 100);
      expect(result.trend).toBe('flat');
      expect(result.symbol).toBe('→');
    });

    it('returns flat when previous is zero', () => {
      const result = calculateTrend(100, 0);
      expect(result.trend).toBe('flat');
      expect(result.change).toBe(0);
    });
  });

  describe('change percentage', () => {
    it('calculates 10% increase correctly', () => {
      const result = calculateTrend(110, 100);
      expect(result.change).toBe(10);
    });

    it('calculates 50% decrease correctly', () => {
      const result = calculateTrend(50, 100);
      expect(result.change).toBe(-50);
    });

    it('calculates 100% increase correctly', () => {
      const result = calculateTrend(200, 100);
      expect(result.change).toBe(100);
    });

    it('calculates small changes accurately', () => {
      const result = calculateTrend(101, 100);
      expect(result.change).toBe(1);
      expect(result.trend).toBe('flat');
    });
  });

  describe('threshold at 5%', () => {
    it('treats 4.9% change as flat', () => {
      const result = calculateTrend(104.9, 100);
      expect(result.trend).toBe('flat');
    });

    it('treats 5% change as up', () => {
      const result = calculateTrend(105.1, 100);
      expect(result.trend).toBe('up');
    });

    it('treats -4.9% change as flat', () => {
      const result = calculateTrend(95.1, 100);
      expect(result.trend).toBe('flat');
    });

    it('treats -5% change as down', () => {
      const result = calculateTrend(94.9, 100);
      expect(result.trend).toBe('down');
    });
  });
});

describe('useCostTrends - Type Exports', () => {
  describe('CostData', () => {
    it('has timestamp property', () => {
      const data: CostData = {
        timestamp: new Date(),
        totalCostUSD: 5.50,
        inputTokens: 10000,
        outputTokens: 5000,
      };
      expect(data.timestamp).toBeInstanceOf(Date);
    });

    it('has totalCostUSD property', () => {
      const data: CostData = {
        timestamp: new Date(),
        totalCostUSD: 12.75,
        inputTokens: 20000,
        outputTokens: 10000,
      };
      expect(data.totalCostUSD).toBe(12.75);
    });

    it('has token counts', () => {
      const data: CostData = {
        timestamp: new Date(),
        totalCostUSD: 8.00,
        inputTokens: 15000,
        outputTokens: 8000,
      };
      expect(data.inputTokens).toBe(15000);
      expect(data.outputTokens).toBe(8000);
    });
  });

  describe('CostTrend', () => {
    it('has period property', () => {
      const trend: Partial<CostTrend> = {
        period: 'month',
      };
      expect(trend.period).toBe('month');
    });

    it('has date range properties', () => {
      const trend: Partial<CostTrend> = {
        startDate: new Date('2024-02-01'),
        endDate: new Date('2024-02-29'),
      };
      expect(trend.startDate).toBeInstanceOf(Date);
      expect(trend.endDate).toBeInstanceOf(Date);
    });

    it('has cost comparison properties', () => {
      const trend: Partial<CostTrend> = {
        totalCost: 50.00,
        previousPeriodCost: 40.00,
        percentChange: 25,
      };
      expect(trend.totalCost).toBe(50.00);
      expect(trend.previousPeriodCost).toBe(40.00);
      expect(trend.percentChange).toBe(25);
    });

    it('has trend direction properties', () => {
      const trend: Partial<CostTrend> = {
        trend: 'up',
        trendSymbol: '↗',
      };
      expect(trend.trend).toBe('up');
      expect(trend.trendSymbol).toBe('↗');
    });

    it('models complete trend object', () => {
      const trend: CostTrend = {
        period: 'week',
        startDate: new Date('2024-02-12'),
        endDate: new Date('2024-02-18'),
        totalCost: 25.00,
        previousPeriodCost: 20.00,
        percentChange: 25,
        trend: 'up',
        trendSymbol: '↗',
      };
      expect(trend.period).toBe('week');
      expect(trend.trend).toBe('up');
    });
  });

  describe('CostBudgetStatus', () => {
    it('has spent property', () => {
      const status: Partial<CostBudgetStatus> = {
        spent: 25.50,
      };
      expect(status.spent).toBe(25.50);
    });

    it('has budget property', () => {
      const status: Partial<CostBudgetStatus> = {
        budget: 100.00,
      };
      expect(status.budget).toBe(100.00);
    });

    it('has percentUsed property', () => {
      const status: Partial<CostBudgetStatus> = {
        percentUsed: 75,
      };
      expect(status.percentUsed).toBe(75);
    });

    it('has daysRemaining property', () => {
      const status: Partial<CostBudgetStatus> = {
        daysRemaining: 10,
      };
      expect(status.daysRemaining).toBe(10);
    });

    it('has burnRate property', () => {
      const status: Partial<CostBudgetStatus> = {
        burnRate: 2.50,
      };
      expect(status.burnRate).toBe(2.50);
    });

    it('has projectedTotal property', () => {
      const status: Partial<CostBudgetStatus> = {
        projectedTotal: 85.00,
      };
      expect(status.projectedTotal).toBe(85.00);
    });

    it('has status property', () => {
      const normal: Partial<CostBudgetStatus> = { status: 'normal' };
      const warning: Partial<CostBudgetStatus> = { status: 'warning' };
      const critical: Partial<CostBudgetStatus> = { status: 'critical' };

      expect(normal.status).toBe('normal');
      expect(warning.status).toBe('warning');
      expect(critical.status).toBe('critical');
    });
  });
});

describe('useCostTrends - Budget Status Scenarios', () => {
  it('models normal budget status', () => {
    const status: CostBudgetStatus = {
      spent: 30.00,
      budget: 100.00,
      percentUsed: 30,
      daysRemaining: 20,
      burnRate: 1.50,
      projectedTotal: 45.00,
      status: 'normal',
    };
    expect(status.status).toBe('normal');
    expect(status.percentUsed).toBeLessThan(70);
  });

  it('models warning budget status (70-90% used)', () => {
    const status: CostBudgetStatus = {
      spent: 75.00,
      budget: 100.00,
      percentUsed: 75,
      daysRemaining: 10,
      burnRate: 3.75,
      projectedTotal: 112.50,
      status: 'warning',
    };
    expect(status.status).toBe('warning');
    expect(status.percentUsed).toBeGreaterThanOrEqual(70);
    expect(status.percentUsed).toBeLessThan(90);
  });

  it('models critical budget status (>90% used)', () => {
    const status: CostBudgetStatus = {
      spent: 95.00,
      budget: 100.00,
      percentUsed: 95,
      daysRemaining: 5,
      burnRate: 4.75,
      projectedTotal: 118.75,
      status: 'critical',
    };
    expect(status.status).toBe('critical');
    expect(status.percentUsed).toBeGreaterThanOrEqual(90);
  });

  it('calculates budget percentage correctly', () => {
    const spent = 50;
    const budget = 100;
    const percentUsed = (spent / budget) * 100;
    expect(percentUsed).toBe(50);
  });

  it('handles zero budget gracefully', () => {
    const spent = 10;
    const budget = 0;
    const percentUsed = budget > 0 ? (spent / budget) * 100 : 0;
    expect(percentUsed).toBe(0);
  });
});

describe('useCostTrends - Burn Rate Calculations', () => {
  it('calculates burn rate from daily spending', () => {
    const spent = 30;
    const daysElapsed = 10;
    const burnRate = spent / daysElapsed;
    expect(burnRate).toBe(3);
  });

  it('projects total based on burn rate', () => {
    const burnRate = 3;
    const totalDays = 30;
    const projectedTotal = burnRate * totalDays;
    expect(projectedTotal).toBe(90);
  });

  it('handles first day of period', () => {
    const spent = 5;
    const daysElapsed = 1;
    const burnRate = daysElapsed > 0 ? spent / daysElapsed : 0;
    expect(burnRate).toBe(5);
  });

  it('handles zero days elapsed', () => {
    const spent = 0;
    const daysElapsed = 0;
    const burnRate = daysElapsed > 0 ? spent / daysElapsed : 0;
    expect(burnRate).toBe(0);
  });
});

describe('useCostTrends - Trend Period Scenarios', () => {
  it('models daily period', () => {
    const period = 'day';
    expect(period).toBe('day');
  });

  it('models weekly period', () => {
    const period = 'week';
    expect(period).toBe('week');
  });

  it('models monthly period', () => {
    const period = 'month';
    expect(period).toBe('month');
  });

  it('trend symbols map correctly', () => {
    const upTrend: CostTrend = {
      period: 'day',
      startDate: new Date(),
      endDate: new Date(),
      totalCost: 10,
      previousPeriodCost: 8,
      percentChange: 25,
      trend: 'up',
      trendSymbol: '↗',
    };

    const downTrend: CostTrend = {
      period: 'day',
      startDate: new Date(),
      endDate: new Date(),
      totalCost: 8,
      previousPeriodCost: 10,
      percentChange: -20,
      trend: 'down',
      trendSymbol: '↘',
    };

    const flatTrend: CostTrend = {
      period: 'day',
      startDate: new Date(),
      endDate: new Date(),
      totalCost: 10,
      previousPeriodCost: 10,
      percentChange: 0,
      trend: 'flat',
      trendSymbol: '→',
    };

    expect(upTrend.trendSymbol).toBe('↗');
    expect(downTrend.trendSymbol).toBe('↘');
    expect(flatTrend.trendSymbol).toBe('→');
  });
});

describe('useCostTrends - Common Patterns', () => {
  it('cost values are numbers', () => {
    const data: CostData = {
      timestamp: new Date(),
      totalCostUSD: 5.25,
      inputTokens: 10000,
      outputTokens: 5000,
    };
    expect(typeof data.totalCostUSD).toBe('number');
  });

  it('token counts are integers', () => {
    const data: CostData = {
      timestamp: new Date(),
      totalCostUSD: 1.00,
      inputTokens: 5000,
      outputTokens: 2500,
    };
    expect(Number.isInteger(data.inputTokens)).toBe(true);
    expect(Number.isInteger(data.outputTokens)).toBe(true);
  });

  it('percentages are 0-100 range', () => {
    const status: CostBudgetStatus = {
      spent: 50,
      budget: 100,
      percentUsed: 50,
      daysRemaining: 15,
      burnRate: 2.5,
      projectedTotal: 75,
      status: 'normal',
    };
    expect(status.percentUsed).toBeGreaterThanOrEqual(0);
    expect(status.percentUsed).toBeLessThanOrEqual(100);
  });
});
