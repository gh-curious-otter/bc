/**
 * useCostTrends - Hook for analyzing cost trends and spending patterns
 * Issue #1047: Activity timeline and cost trend tracking
 *
 * Calculates cost trends, burn rates, and projections for dashboard and cost view.
 */

import { useState, useEffect, useCallback } from 'react';
import { getCostSummary } from '../services/bc';

export interface CostData {
  timestamp: Date;
  totalCostUSD: number;
  inputTokens: number;
  outputTokens: number;
}

export interface CostTrend {
  period: string; // "day", "week", "month"
  startDate: Date;
  endDate: Date;
  totalCost: number;
  previousPeriodCost: number;
  percentChange: number; // 0-100
  trend: 'up' | 'down' | 'flat';
  trendSymbol: '↗' | '↘' | '→';
}

export interface CostBudgetStatus {
  spent: number;
  budget: number;
  percentUsed: number;
  daysRemaining: number;
  burnRate: number; // $ per day
  projectedTotal: number; // Projected end-of-period cost
  status: 'normal' | 'warning' | 'critical';
}

interface UseCostTrendsOptions {
  budget?: number; // Monthly budget in USD
  period?: 'day' | 'week' | 'month'; // Trend period
}

/**
 * Calculate trend direction and percentage change
 */
export function calculateTrend(current: number, previous: number): { trend: 'up' | 'down' | 'flat'; symbol: '↗' | '↘' | '→'; change: number } {
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

/**
 * Get days remaining in the current period
 */
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

/**
 * Get days elapsed in current period
 */
function getDaysElapsed(period: 'day' | 'week' | 'month'): number {
  const now = new Date();
  switch (period) {
    case 'day':
      return 1;
    case 'week':
      return now.getDay() || 7; // Sunday = 7
    case 'month':
      return now.getDate();
  }
}

/**
 * Hook to analyze cost trends and budget status
 */
export function useCostTrends(options: UseCostTrendsOptions = {}) {
  const { budget = 10.0, period = 'month' } = options;
  const [trends, setTrends] = useState<CostTrend[]>([]);
  const [budgetStatus, setBudgetStatus] = useState<CostBudgetStatus>({
    spent: 0,
    budget,
    percentUsed: 0,
    daysRemaining: 30,
    burnRate: 0,
    projectedTotal: 0,
    status: 'normal',
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchCostTrends = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Fetch cost data from bc CLI
      const costData = await getCostSummary();

      const spent = costData.total_cost ?? 0;
      const daysElapsed = getDaysElapsed(period);
      const daysRemaining = getDaysRemaining(period);
      const totalDays = daysElapsed + daysRemaining;

      // Calculate burn rate ($ per day)
      const burnRate = daysElapsed > 0 ? spent / daysElapsed : 0;

      // Project total spend for the period
      const projectedTotal = burnRate * totalDays;

      // Calculate budget usage percentage
      const percentUsed = budget > 0 ? Math.round((spent / budget) * 100) : 0;

      // Determine status based on projected spend vs budget
      let status: 'normal' | 'warning' | 'critical' = 'normal';
      if (projectedTotal > budget * 1.2 || percentUsed > 90) {
        status = 'critical';
      } else if (projectedTotal > budget * 0.9 || percentUsed > 70) {
        status = 'warning';
      }

      setBudgetStatus({
        spent,
        budget,
        percentUsed,
        daysRemaining,
        burnRate,
        projectedTotal,
        status,
      });

      // TODO: Store historical data to calculate trends over time
      setTrends([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load cost trends');
    } finally {
      setLoading(false);
    }
  }, [budget, period]);

  useEffect(() => {
    void fetchCostTrends();
    const interval = setInterval(() => {
      void fetchCostTrends();
    }, 60000); // Refresh every 60 seconds
    return () => clearInterval(interval);
  }, [fetchCostTrends]);

  return { trends, budgetStatus, loading, error, refresh: fetchCostTrends };
}

export default useCostTrends;
