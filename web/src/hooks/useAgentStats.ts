import { useCallback } from 'react';
import { api } from '../api/client';
import type { AgentStatsRecord } from '../api/client';
import { usePolling } from './usePolling';

/**
 * Polls Docker resource stats for an agent every `intervalMs` milliseconds.
 * Returns null data (not an error) when the endpoint returns 404 or empty,
 * which indicates the agent runs on a non-Docker runtime (e.g. tmux).
 */
export function useAgentStats(agentName: string, intervalMs = 10000) {
  const fetcher = useCallback(async (): Promise<AgentStatsRecord[]> => {
    try {
      const records = await api.getAgentStats(agentName, 1);
      return records;
    } catch {
      // 404 or error means stats not available for this runtime
      return [];
    }
  }, [agentName]);

  return usePolling<AgentStatsRecord[]>(fetcher, intervalMs);
}
