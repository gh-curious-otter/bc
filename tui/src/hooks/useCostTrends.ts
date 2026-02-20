/**
 * useCostTrends - Hook for analyzing cost trends and spending patterns
 * Issue #1047: Activity timeline and cost trend tracking
 *
 * Calculates cost trends, burn rates, and projections for dashboard and cost view.
 */

import { useState, useEffect, useCallback } from 'react';
import type { CostSummary } from '../types';
import { getCostSummary } from '../services/bc';
import { usePerformanceConfig } from '../config';

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
 * Get days remaining in the period
 */
function getDaysRemaining(period: 'day' | 'week' | 'month'): number {
  const now = new Date();
  if (period === 'day') {
    return 1;
  } else if (period === 'week') {
    const dayOfWeek = now.getDay();
    return 7 - dayOfWeek;
  } else {
    const lastDay = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
    return lastDay - now.getDate() + 1;
  }
}

/**
 * Get days elapsed in the period
 */
function getDaysElapsed(period: 'day' | 'week' | 'month'): number {
  const now = new Date();
  if (period === 'day') {
    return 1;
  } else if (period === 'week') {
    return now.getDay() || 7; // Sunday = 0 -> 7
  } else {
    return now.getDate();
  }
}

/**
 * Calculate budget status from cost summary
 */
function calculateBudgetStatus(
  costData: CostSummary,
  budget: number,
  period: 'day' | 'week' | 'month'
): CostBudgetStatus {
  const spent = costData.total_cost;
  const percentUsed = budget > 0 ? Math.round((spent / budget) * 100) : 0;
  const daysRemaining = getDaysRemaining(period);
  const daysElapsed = getDaysElapsed(period);
  const burnRate = daysElapsed > 0 ? spent / daysElapsed : 0;
  const totalDays = daysElapsed + daysRemaining;
  const projectedTotal = burnRate * totalDays;

  // Determine status based on projected spend
  let status: 'normal' | 'warning' | 'critical' = 'normal';
  if (percentUsed >= 90 || projectedTotal > budget * 1.2) {
    status = 'critical';
  } else if (percentUsed >= 70 || projectedTotal > budget) {
    status = 'warning';
  }

  return {
    spent,
    budget,
    percentUsed,
    daysRemaining,
    burnRate,
    projectedTotal,
    status,
  };
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

  const perfConfig = usePerformanceConfig();
  const pollInterval = perfConfig.poll_interval_costs;

  const fetchCostTrends = useCallback(async () => {
    try {
      // Fetch cost data from bc CLI
      const costData = await getCostSummary();

      // Calculate budget status
      const status = calculateBudgetStatus(costData, budget, period);
      setBudgetStatus(status);

      // Build cost trends by agent (if available)
      const agentTrends: CostTrend[] = [];
      if (costData.by_agent) {
        for (const [agent, cost] of Object.entries(costData.by_agent)) {
          agentTrends.push({
            period: agent,
            startDate: new Date(),
            endDate: new Date(),
            totalCost: cost,
            previousPeriodCost: 0,
            percentChange: 0,
            trend: 'flat',
            trendSymbol: '→',
          });
        }
      }
      setTrends(agentTrends);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load cost trends');
    } finally {
      setLoading(false);
    }
  }, [budget, period]);

  // Initial fetch
  useEffect(() => {
    void fetchCostTrends();
  }, [fetchCostTrends]);

  // Polling
  useEffect(() => {
    const interval = setInterval(() => {
      void fetchCostTrends();
    }, pollInterval);
    return () => { clearInterval(interval); };
  }, [fetchCostTrends, pollInterval]);

  return { trends, budgetStatus, loading, error, refresh: fetchCostTrends };
}

export default useCostTrends;
