/**
 * useProcesses - Hook for managing process data
 * Issue #555: Processes view
 */

import { useState, useEffect, useCallback } from 'react';
import type { Process, BcResult } from '../types';
import { getProcesses, getProcessLogs } from '../services/bc';

export interface UseProcessesOptions {
  /** Polling interval in ms (default: 3000) */
  interval?: number;
  /** Whether to poll automatically (default: true) */
  enabled?: boolean;
  /** Callback when process list updates */
  onUpdate?: () => void;
}

export interface UseProcessesResult extends BcResult<Process[]> {
  /** Refresh process list */
  refresh: () => Promise<void>;
  /** Whether polling is active */
  isPolling: boolean;
  /** Pause polling */
  pause: () => void;
  /** Resume polling */
  resume: () => void;
}

/**
 * Hook for fetching and polling process list
 */
export function useProcesses(options: UseProcessesOptions = {}): UseProcessesResult {
  const { interval = 3000, enabled = true, onUpdate } = options;

  const [data, setData] = useState<Process[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [isPolling, setIsPolling] = useState(enabled);

  const fetchProcesses = useCallback(async () => {
    try {
      const response = await getProcesses();
      setData(response.processes);
      setError(null);
      if (onUpdate) {
        onUpdate();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch processes');
    } finally {
      setLoading(false);
    }
  }, [onUpdate]);

  const pause = useCallback(() => { setIsPolling(false); }, []);
  const resume = useCallback(() => { setIsPolling(true); }, []);

  // Initial fetch
  useEffect(() => {
    void fetchProcesses();
  }, []);

  // Polling interval
  useEffect(() => {
    if (!isPolling) return;
    const timer = setInterval(fetchProcesses, interval);
    return () => clearInterval(timer);
  }, [isPolling, interval, fetchProcesses]);

  return {
    data,
    error,
    loading,
    refresh: fetchProcesses,
    isPolling,
    pause,
    resume,
  };
}

export interface UseProcessLogsOptions {
  /** Process name */
  name: string;
  /** Number of lines to fetch (default: 100) */
  lines?: number;
  /** Polling interval in ms (default: 2000) */
  interval?: number;
  /** Whether to poll automatically (default: true) */
  enabled?: boolean;
}

export interface UseProcessLogsResult extends BcResult<string[]> {
  /** Refresh logs */
  refresh: () => Promise<void>;
  /** Whether polling is active */
  isPolling: boolean;
  /** Pause polling */
  pause: () => void;
  /** Resume polling */
  resume: () => void;
}

/**
 * Hook for fetching and polling process logs
 */
export function useProcessLogs(options: UseProcessLogsOptions): UseProcessLogsResult {
  const { name, lines = 100, interval = 2000, enabled = true } = options;

  const [data, setData] = useState<string[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [isPolling, setIsPolling] = useState(enabled);

  const fetchLogs = useCallback(async () => {
    try {
      const logLines = await getProcessLogs(name, lines);
      setData(logLines);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch logs');
    } finally {
      setLoading(false);
    }
  }, [name, lines]);

  const pause = useCallback(() => setIsPolling(false), []);
  const resume = useCallback(() => setIsPolling(true), []);

  // Initial fetch and reset on name change
  useEffect(() => {
    setLoading(true);
    setData(null);
    fetchLogs();
  }, [name]);

  // Polling interval
  useEffect(() => {
    if (!isPolling) return;
    const timer = setInterval(fetchLogs, interval);
    return () => clearInterval(timer);
  }, [isPolling, interval, fetchLogs]);

  return {
    data,
    error,
    loading,
    refresh: fetchLogs,
    isPolling,
    pause,
    resume,
  };
}
