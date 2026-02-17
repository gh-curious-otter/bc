/**
 * useCosts hook - Fetch and poll cost data
 * Issue #1004: Performance configuration tunables (Phase 5)
 *
 * Poll interval is configurable via workspace config [performance] section.
 */

import { useState, useEffect, useCallback } from 'react';
import type { CostSummary, BcResult } from '../types';
import { getCostSummary } from '../services/bc';
import { usePerformanceConfig } from '../config';

export interface UseCostsOptions {
  /** Polling interval in ms (default: from config) */
  pollInterval?: number;
  /** Whether to poll automatically (default: true) */
  autoPoll?: boolean;
}

export interface UseCostsResult extends BcResult<CostSummary> {
  /** Manually refresh cost data */
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch and optionally poll cost data
 */
export function useCosts(options: UseCostsOptions = {}): UseCostsResult {
  // Get configurable poll interval from workspace config
  const perfConfig = usePerformanceConfig();
  const defaultPollInterval = perfConfig.poll_interval_costs;

  const { pollInterval = defaultPollInterval, autoPoll = true } = options;

  const [data, setData] = useState<CostSummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchCosts = useCallback(async () => {
    try {
      const summary = await getCostSummary();
      setData(summary);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch costs');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial fetch
  useEffect(() => {
    void fetchCosts();
  }, [fetchCosts]);

  // Polling
  useEffect(() => {
    if (!autoPoll) return;
    const interval = setInterval(() => { void fetchCosts(); }, pollInterval);
    return () => { clearInterval(interval); };
  }, [autoPoll, pollInterval, fetchCosts]);

  return {
    data,
    error,
    loading,
    refresh: fetchCosts,
  };
}
