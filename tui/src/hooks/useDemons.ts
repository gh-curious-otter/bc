/**
 * useDemons hook - Fetch and poll demon status
 */

import { useState, useEffect, useCallback } from 'react';
import type { Demon, BcResult } from '../types';
import { getDemons, getDemonLogs, enableDemon, disableDemon, runDemon } from '../services/bc';
import type { DemonRunLog } from '../types';

export interface UseDemonsOptions {
  /** Polling interval in ms (default: 5000) */
  pollInterval?: number;
  /** Whether to poll automatically (default: true) */
  autoPoll?: boolean;
}

export interface UseDemonsResult extends BcResult<Demon[]> {
  /** Total number of demons */
  total: number;
  /** Number of enabled demons */
  enabled: number;
  /** Manually refresh demons */
  refresh: () => Promise<void>;
  /** Enable a demon */
  enable: (name: string) => Promise<void>;
  /** Disable a demon */
  disable: (name: string) => Promise<void>;
  /** Run a demon manually */
  run: (name: string) => Promise<void>;
}

/**
 * Hook to fetch and optionally poll demon status
 * @param options - Configuration options
 * @returns Demon list with metadata and loading state
 */
export function useDemons(options: UseDemonsOptions = {}): UseDemonsResult {
  const { pollInterval = 5000, autoPoll = true } = options;

  const [data, setData] = useState<Demon[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [enabled, setEnabled] = useState(0);

  const fetchDemons = useCallback(async () => {
    try {
      const demons = await getDemons();
      setData(demons);
      setTotal(demons.length);
      setEnabled(demons.filter((d) => d.enabled).length);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch demons');
    } finally {
      setLoading(false);
    }
  }, []);

  const handleEnable = useCallback(async (name: string) => {
    await enableDemon(name);
    await fetchDemons();
  }, [fetchDemons]);

  const handleDisable = useCallback(async (name: string) => {
    await disableDemon(name);
    await fetchDemons();
  }, [fetchDemons]);

  const handleRun = useCallback(async (name: string) => {
    await runDemon(name);
    await fetchDemons();
  }, [fetchDemons]);

  // Initial fetch
  useEffect(() => {
    fetchDemons();
  }, [fetchDemons]);

  // Polling
  useEffect(() => {
    if (!autoPoll) return;

    const interval = setInterval(fetchDemons, pollInterval);
    return () => clearInterval(interval);
  }, [autoPoll, pollInterval, fetchDemons]);

  return {
    data,
    error,
    loading,
    total,
    enabled,
    refresh: fetchDemons,
    enable: handleEnable,
    disable: handleDisable,
    run: handleRun,
  };
}

export interface UseDemonLogsOptions {
  /** Number of recent entries to fetch (default: 10) */
  limit?: number;
  /** Polling interval in ms (default: 5000) */
  pollInterval?: number;
  /** Whether to poll automatically (default: false) */
  autoPoll?: boolean;
}

export interface UseDemonLogsResult extends BcResult<DemonRunLog[]> {
  /** Manually refresh logs */
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch run logs for a specific demon
 */
export function useDemonLogs(
  name: string,
  options: UseDemonLogsOptions = {}
): UseDemonLogsResult {
  const { limit = 10, pollInterval = 5000, autoPoll = false } = options;

  const [data, setData] = useState<DemonRunLog[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchLogs = useCallback(async () => {
    try {
      const logs = await getDemonLogs(name, limit);
      setData(logs);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch demon logs');
    } finally {
      setLoading(false);
    }
  }, [name, limit]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  useEffect(() => {
    if (!autoPoll) return;

    const interval = setInterval(fetchLogs, pollInterval);
    return () => clearInterval(interval);
  }, [autoPoll, pollInterval, fetchLogs]);

  return {
    data,
    error,
    loading,
    refresh: fetchLogs,
  };
}
