/**
 * useAgents hook - Fetch and poll agent status
 */

import { useState, useEffect, useCallback } from 'react';
import type { Agent, BcResult } from '../types';
import { getStatus } from '../services/bc';

export interface UseAgentsOptions {
  /** Polling interval in ms (default: 2000) */
  pollInterval?: number;
  /** Whether to poll automatically (default: true) */
  autoPoll?: boolean;
}

export interface UseAgentsResult extends BcResult<Agent[]> {
  /** Total number of agents */
  total: number;
  /** Number of active (non-stopped) agents */
  active: number;
  /** Number of working agents */
  working: number;
  /** Workspace name */
  workspace: string;
  /** Manually refresh agents */
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch and optionally poll agent status
 * @param options - Configuration options
 * @returns Agent list with metadata and loading state
 */
export function useAgents(options: UseAgentsOptions = {}): UseAgentsResult {
  const { pollInterval = 2000, autoPoll = true } = options;

  const [data, setData] = useState<Agent[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [active, setActive] = useState(0);
  const [working, setWorking] = useState(0);
  const [workspace, setWorkspace] = useState('');

  const fetchAgents = useCallback(async () => {
    try {
      const status = await getStatus();
      setData(status.agents);
      setTotal(status.total);
      setActive(status.active);
      setWorking(status.working);
      setWorkspace(status.workspace);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch agents');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial fetch
  useEffect(() => {
    fetchAgents();
  }, [fetchAgents]);

  // Polling
  useEffect(() => {
    if (!autoPoll) return;

    const interval = setInterval(fetchAgents, pollInterval);
    return () => clearInterval(interval);
  }, [autoPoll, pollInterval, fetchAgents]);

  return {
    data,
    error,
    loading,
    total,
    active,
    working,
    workspace,
    refresh: fetchAgents,
  };
}

/**
 * Get agents filtered by state
 */
export function useAgentsByState(
  state: string,
  options?: UseAgentsOptions
): BcResult<Agent[]> & { refresh: () => Promise<void> } {
  const result = useAgents(options);

  const filteredData = result.data?.filter((agent) => agent.state === state) ?? null;

  return {
    data: filteredData,
    error: result.error,
    loading: result.loading,
    refresh: result.refresh,
  };
}

/**
 * Get agents filtered by role
 */
export function useAgentsByRole(
  role: string,
  options?: UseAgentsOptions
): BcResult<Agent[]> & { refresh: () => Promise<void> } {
  const result = useAgents(options);

  const filteredData = result.data?.filter((agent) => agent.role === role) ?? null;

  return {
    data: filteredData,
    error: result.error,
    loading: result.loading,
    refresh: result.refresh,
  };
}

/**
 * Get a single agent by name
 */
export function useAgent(
  name: string,
  options?: UseAgentsOptions
): BcResult<Agent> & { refresh: () => Promise<void> } {
  const result = useAgents(options);

  const agent = result.data?.find((a) => a.name === name) ?? null;

  return {
    data: agent,
    error: result.error,
    loading: result.loading,
    refresh: result.refresh,
  };
}
