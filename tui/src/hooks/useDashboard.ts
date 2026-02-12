import { useState, useEffect, useMemo, useCallback } from 'react';
import { getStatus, getChannels, getCostSummary } from '../services/bc.js';
import type { StatusResponse, ChannelsResponse, CostSummary, Agent } from '../types';

interface UseDataResult<T> {
  data: T | null;
  isLoading: boolean;
  error: Error | null;
}

interface DashboardSummary {
  workspaceName: string;
  total: number;
  active: number;
  working: number;
  idle: number;
  stuck: number;
  error: number;
  totalCostUSD: number;
  inputTokens: number;
  outputTokens: number;
}

interface AgentStats {
  byState: Record<string, number>;
  byRole: Record<string, number>;
}

interface DashboardAgent {
  name: string;
  role: string;
  state: string;
  task: string;
  uptime: string;
  startedAt: string;
  updatedAt: string;
  [key: string]: unknown;
}

interface DashboardChannel {
  name: string;
  members: string[];
  description?: string;
}

/**
 * useDashboard - Aggregates data from multiple bc CLI commands
 * Hook for Dashboard view (Issues #543, #544)
 *
 * Integrates with bc CLI via service layer for real-time workspace data.
 */
export function useDashboard() {
  const [agents, setAgents] = useState<UseDataResult<DashboardAgent[]>>({
    data: null,
    isLoading: true,
    error: null,
  });

  const [channels, setChannels] = useState<UseDataResult<DashboardChannel[]>>({
    data: null,
    isLoading: true,
    error: null,
  });

  const [cost, setCost] = useState<UseDataResult<CostSummary>>({
    data: null,
    isLoading: true,
    error: null,
  });

  const [workspaceName, setWorkspaceName] = useState('bc');
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);

  // Fetch all data
  const fetchData = useCallback(async () => {
    setAgents((prev) => ({ ...prev, isLoading: true }));
    setChannels((prev) => ({ ...prev, isLoading: true }));
    setCost((prev) => ({ ...prev, isLoading: true }));

    // Fetch status (agents)
    try {
      const statusResponse: StatusResponse = await getStatus();
      setWorkspaceName(statusResponse.workspace || 'bc');
      setAgents({
        data: statusResponse.agents.map((a: Agent) => ({
          name: a.name,
          role: a.role,
          state: a.state,
          task: a.task || '',
          uptime: '',
          startedAt: formatTime(a.started_at),
          updatedAt: formatTime(a.updated_at),
        })),
        isLoading: false,
        error: null,
      });
    } catch (err) {
      setAgents({
        data: [],
        isLoading: false,
        error: err instanceof Error ? err : new Error('Failed to fetch agents'),
      });
    }

    // Fetch channels
    try {
      const channelsResponse: ChannelsResponse = await getChannels();
      setChannels({
        data: channelsResponse.channels.map((c) => ({
          name: c.name,
          members: c.members,
        })),
        isLoading: false,
        error: null,
      });
    } catch (err) {
      setChannels({
        data: [],
        isLoading: false,
        error: err instanceof Error ? err : new Error('Failed to fetch channels'),
      });
    }

    // Fetch costs
    try {
      const costResponse: CostSummary = await getCostSummary();
      setCost({
        data: costResponse,
        isLoading: false,
        error: null,
      });
    } catch (err) {
      setCost({
        data: null,
        isLoading: false,
        error: err instanceof Error ? err : new Error('Failed to fetch costs'),
      });
    }

    setLastRefresh(new Date());
  }, []);

  // Initial fetch on mount
  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Auto-refresh every 30 seconds (optimized for performance - Issue #559)
  // Users can manually refresh with 'r' key for immediate updates
  useEffect(() => {
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, [fetchData]);

  // Compute agent stats breakdown
  const agentStats = useMemo<AgentStats>(() => {
    const agentList = agents.data ?? [];
    const byState: Record<string, number> = {};
    const byRole: Record<string, number> = {};

    for (const agent of agentList) {
      byState[agent.state] = (byState[agent.state] || 0) + 1;
      byRole[agent.role] = (byRole[agent.role] || 0) + 1;
    }

    return { byState, byRole };
  }, [agents.data]);

  // Compute summary from data
  const summary = useMemo<DashboardSummary>(() => {
    const agentList = agents.data ?? [];
    return {
      workspaceName,
      total: agentList.length,
      active: agentList.filter((a) => a.state !== 'stopped' && a.state !== 'idle').length,
      working: agentList.filter((a) => a.state === 'working').length,
      idle: agentList.filter((a) => a.state === 'idle').length,
      stuck: agentList.filter((a) => a.state === 'stuck').length,
      error: agentList.filter((a) => a.state === 'error').length,
      totalCostUSD: cost.data?.total_cost ?? 0,
      inputTokens: cost.data?.total_input_tokens ?? 0,
      outputTokens: cost.data?.total_output_tokens ?? 0,
    };
  }, [workspaceName, agents.data, cost.data]);

  const isLoading = agents.isLoading || channels.isLoading || cost.isLoading;
  const error = agents.error || channels.error || cost.error;

  return {
    summary,
    agents,
    channels,
    cost,
    agentStats,
    isLoading,
    error,
    refresh: fetchData,
    lastRefresh,
  };
}

/**
 * Format ISO timestamp to relative time string
 */
function formatTime(isoString: string | undefined): string {
  if (!isoString) return '-';
  try {
    const date = new Date(isoString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return 'now';
    if (diffMins < 60) return `${diffMins}m ago`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;

    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  } catch {
    return '-';
  }
}

export default useDashboard;
