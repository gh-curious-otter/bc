/**
 * useAgents hook - Fetch and poll agent status
 * Issue #1004: Performance configuration tunables (Phase 5)
 *
 * Includes debounce for working→idle transitions to prevent flickering.
 * When an agent transitions from 'working' to 'idle', the display state
 * remains 'working' for a debounce period before showing 'idle'.
 *
 * Poll interval is configurable via workspace config [performance] section.
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import type { Agent, AgentState, BcResult } from '../types';
import { getStatus } from '../services/bc';
import { usePerformanceConfig } from '../config';

/** Debounce period for working→idle transition (in ms) */
const WORKING_TO_IDLE_DEBOUNCE_MS = 5000;

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
  // Get configurable poll interval from workspace config
  const perfConfig = usePerformanceConfig();
  const defaultPollInterval = perfConfig.poll_interval_agents;

  const { pollInterval = defaultPollInterval, autoPoll = true } = options;

  const [data, setData] = useState<Agent[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [active, setActive] = useState(0);
  const [working, setWorking] = useState(0);
  const [workspace, setWorkspace] = useState('');

  // Track last time each agent was in "working" state (for debounce)
  const lastWorkingTimeRef = useRef<Record<string, number>>({});
  // Track previous state for each agent to detect transitions
  const prevStateRef = useRef<Record<string, AgentState>>({});

  /**
   * Apply debounce to working→idle transitions.
   * If an agent just transitioned from working to idle within the debounce period,
   * keep showing "working" to prevent flickering.
   *
   * #1427: Always return new objects to ensure React detects state changes
   * (e.g., task field updates while state remains 'working')
   */
  const applyStateDebounce = useCallback((agents: Agent[]): Agent[] => {
    const now = Date.now();

    return agents.map((agent) => {
      const prevState = prevStateRef.current[agent.name];
      const lastWorkingTime = lastWorkingTimeRef.current[agent.name] ?? 0;

      // Update tracking: record when agent was last "working"
      if (agent.state === 'working') {
        lastWorkingTimeRef.current[agent.name] = now;
        prevStateRef.current[agent.name] = agent.state;
        // #1427: Always return a new object to trigger React re-render
        return { ...agent };
      }

      // Detect working→idle transition
      if (prevState === 'working' && agent.state === 'idle') {
        const timeSinceWorking = now - lastWorkingTime;

        // If within debounce period, keep showing "working"
        if (timeSinceWorking < WORKING_TO_IDLE_DEBOUNCE_MS) {
          return { ...agent, state: 'working' as AgentState };
        }
      }

      // Update previous state tracking
      prevStateRef.current[agent.name] = agent.state;
      // #1427: Always return a new object to trigger React re-render
      return { ...agent };
    });
  }, []);

  const fetchAgents = useCallback(async () => {
    try {
      const status = await getStatus();
      // Apply debounce to agent states before setting data
      const debouncedAgents = applyStateDebounce(status.agents);
      setData(debouncedAgents);
      setTotal(status.total);
      setActive(status.active);
      // Recalculate working count based on debounced states
      const workingCount = debouncedAgents.filter((a) => a.state === 'working').length;
      setWorking(workingCount);
      setWorkspace(status.workspace);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch agents');
    } finally {
      setLoading(false);
    }
  }, [applyStateDebounce]);

  // Initial fetch
  useEffect(() => {
    void fetchAgents();
  }, [fetchAgents]);

  // Polling
  useEffect(() => {
    if (!autoPoll) return;

    const interval = setInterval(() => { void fetchAgents(); }, pollInterval);
    return () => { clearInterval(interval); };
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
