/**
 * useStatus hook - Workspace status and summary
 * Issue #1004: Performance configuration tunables (Phase 5)
 *
 * Poll interval is configurable via workspace config [performance] section.
 */

import { useState, useEffect, useCallback } from 'react';
import type { StatusResponse, BcResult } from '../types';
import { getStatus } from '../services/bc';
import { usePerformanceConfig } from '../config';

export interface UseStatusOptions {
  /** Polling interval in ms (default: from config) */
  pollInterval?: number;
  /** Whether to poll automatically (default: true) */
  autoPoll?: boolean;
}

export interface WorkspaceStatus {
  /** Workspace name */
  workspace: string;
  /** Total agent count */
  total: number;
  /** Active (non-stopped) agent count */
  active: number;
  /** Working agent count */
  working: number;
  /** Idle agent count */
  idle: number;
  /** Done agent count */
  done: number;
  /** Stuck agent count */
  stuck: number;
  /** Error agent count */
  error: number;
  /** Stopped agent count */
  stopped: number;
}

export interface UseStatusResult extends BcResult<WorkspaceStatus> {
  /** Raw status response */
  rawResponse: StatusResponse | null;
  /** Manually refresh status */
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch and poll workspace status summary
 * @param options - Configuration options
 * @returns Workspace status with agent counts by state
 */
export function useStatus(options: UseStatusOptions = {}): UseStatusResult {
  // Get configurable poll interval from workspace config
  const perfConfig = usePerformanceConfig();
  const defaultPollInterval = perfConfig.poll_interval_status;

  const { pollInterval = defaultPollInterval, autoPoll = true } = options;

  const [rawResponse, setRawResponse] = useState<StatusResponse | null>(null);
  const [data, setData] = useState<WorkspaceStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchStatus = useCallback(async () => {
    try {
      const status = await getStatus();
      setRawResponse(status);

      // Calculate counts by state
      const counts = {
        idle: 0,
        working: 0,
        done: 0,
        stuck: 0,
        error: 0,
        stopped: 0,
      };

      for (const agent of status.agents) {
        if (agent.state in counts) {
          counts[agent.state as keyof typeof counts]++;
        }
      }

      setData({
        workspace: status.workspace,
        total: status.total,
        active: status.active,
        ...counts,
      });
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch status');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial fetch
  useEffect(() => {
    void fetchStatus();
  }, [fetchStatus]);

  // Polling
  useEffect(() => {
    if (!autoPoll) return;

    const interval = setInterval(() => { void fetchStatus(); }, pollInterval);
    return () => { clearInterval(interval); };
  }, [autoPoll, pollInterval, fetchStatus]);

  return {
    data,
    error,
    loading,
    rawResponse,
    refresh: fetchStatus,
  };
}

/**
 * Hook to check if workspace is healthy
 * Returns true if no agents are stuck or in error state
 */
export function useWorkspaceHealth(options?: UseStatusOptions): {
  healthy: boolean;
  stuckCount: number;
  errorCount: number;
  loading: boolean;
} {
  const { data, loading } = useStatus(options);

  return {
    healthy: data ? data.stuck === 0 && data.error === 0 : true,
    stuckCount: data?.stuck ?? 0,
    errorCount: data?.error ?? 0,
    loading,
  };
}

/**
 * Hook to get workspace utilization (working/active ratio)
 */
export function useUtilization(options?: UseStatusOptions): {
  utilization: number;
  loading: boolean;
} {
  const { data, loading } = useStatus(options);

  const utilization =
    data && data.active > 0 ? (data.working / data.active) * 100 : 0;

  return {
    utilization: Math.round(utilization),
    loading,
  };
}
