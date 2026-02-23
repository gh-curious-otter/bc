/**
 * useSystemHealth - Hook for fetching system health/observability data
 * Issue #1759: Performance/Monitor tab
 */

import { useState, useEffect, useCallback, useMemo } from 'react';
import { getAgentHealth, getStats, getCostSummary } from '../services/bc';
import type { AgentHealth, StatsResponse, CostSummary } from '../types';

export interface SystemHealthData {
  health: AgentHealth[];
  stats: StatsResponse | null;
  costs: CostSummary | null;
}

export interface SystemHealthSummary {
  totalAgents: number;
  healthyAgents: number;
  degradedAgents: number;
  stuckAgents: number;
  errorAgents: number;
  totalCost: number;
  costToday: number;
}

export interface UseSystemHealthOptions {
  /** Auto-refresh interval in ms (default: 30000) */
  pollInterval?: number;
  /** Whether to auto-poll (default: true) */
  autoPoll?: boolean;
}

export interface UseSystemHealthResult {
  data: SystemHealthData;
  summary: SystemHealthSummary;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  lastRefresh: Date | null;
}

export function useSystemHealth(options: UseSystemHealthOptions = {}): UseSystemHealthResult {
  const { pollInterval = 30000, autoPoll = true } = options;

  const [health, setHealth] = useState<AgentHealth[]>([]);
  const [stats, setStats] = useState<StatsResponse | null>(null);
  const [costs, setCosts] = useState<CostSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const [healthData, statsData, costData] = await Promise.all([
        getAgentHealth(),
        getStats(),
        getCostSummary(),
      ]);

      setHealth(healthData);
      setStats(statsData);
      setCosts(costData);
      setLastRefresh(new Date());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch performance data');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial load
  useEffect(() => {
    void refresh();
  }, [refresh]);

  // Auto-poll
  useEffect(() => {
    if (!autoPoll) return;

    const interval = setInterval(() => {
      void refresh();
    }, pollInterval);

    return () => { clearInterval(interval); };
  }, [autoPoll, pollInterval, refresh]);

  // Compute summary
  const summary = useMemo<SystemHealthSummary>(() => {
    const healthyCount = health.filter(h => h.status === 'healthy').length;
    const degradedCount = health.filter(h => h.status === 'degraded').length;
    const stuckCount = health.filter(h => h.status === 'stuck').length;
    const errorCount = health.filter(h => h.status === 'error').length;

    return {
      totalAgents: health.length,
      healthyAgents: healthyCount,
      degradedAgents: degradedCount,
      stuckAgents: stuckCount,
      errorAgents: errorCount,
      totalCost: costs?.total_cost ?? 0,
      costToday: 0, // TODO: Add daily cost tracking
    };
  }, [health, costs]);

  return {
    data: { health, stats, costs },
    summary,
    loading,
    error,
    refresh,
    lastRefresh,
  };
}

export default useSystemHealth;
