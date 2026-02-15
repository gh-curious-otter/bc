/**
 * useCosts hook - Fetch and poll cost data
 */

import { useState, useEffect, useCallback } from 'react';
import type { CostSummary, BcResult } from '../types';
import { getCostSummary } from '../services/bc';

export interface UseCostsOptions {
  /** Polling interval in ms (default: 5000) */
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
  const { pollInterval = 5000, autoPoll = true } = options;

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
    const interval = setInterval(fetchCosts, pollInterval);
    return () => clearInterval(interval);
  }, [autoPoll, pollInterval, fetchCosts]);

  return {
    data,
    error,
    loading,
    refresh: fetchCosts,
  };
}
