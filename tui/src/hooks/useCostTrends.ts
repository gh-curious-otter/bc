/**
 * useCostTrends - Hook for analyzing cost trends and spending patterns
 * Issue #1047: Activity timeline and cost trend tracking
 *
 * Calculates cost trends, burn rates, and projections for dashboard and cost view.
 */

import { useState, useEffect } from 'react';

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
 * (Used in future: trend visualization and cost analysis)
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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCostTrends = async () => {
    setLoading(true);
    setError(null);
    try {
      // In real implementation, query bc cost API
      // For now, return empty structure
      setTrends([]);
      setBudgetStatus({
        spent: 0,
        budget,
        percentUsed: 0,
        daysRemaining: 30,
        burnRate: 0,
        projectedTotal: 0,
        status: 'normal',
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load cost trends');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchCostTrends();
    const interval = setInterval(() => {
      void fetchCostTrends();
    }, 60000); // Refresh every 60 seconds
    return () => clearInterval(interval);
  }, [budget, period]);

  return { trends, budgetStatus, loading, error, refresh: fetchCostTrends };
}

export default useCostTrends;
