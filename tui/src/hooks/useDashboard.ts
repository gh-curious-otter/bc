import { useState, useEffect, useMemo, useCallback } from 'react';

// Types matching bc CLI --json output
interface Agent {
  name: string;
  role: string;
  state: string;
  task: string;
  startedAt: string;
  updatedAt: string;
}

interface Channel {
  name: string;
  members: string[];
  description?: string;
}

interface StatusResponse {
  workspace: string;
  total: number;
  active: number;
  working: number;
  agents: Agent[];
}

interface CostSummary {
  totalCostUSD: number;
  recordCount: number;
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
}

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
  totalCostUSD: number;
}

/**
 * useDashboard - Aggregates data from multiple bc CLI commands
 * Hook for Dashboard view (Issue #543)
 *
 * Note: This is a skeleton that will be connected to the service layer
 * once eng-02's PRs (#537, #538, #539) are merged.
 */
export function useDashboard() {
  // Placeholder state - will be replaced with actual hooks from eng-02
  const [agents, setAgents] = useState<UseDataResult<Agent[]>>({
    data: null,
    isLoading: true,
    error: null,
  });

  const [channels, setChannels] = useState<UseDataResult<Channel[]>>({
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

  // Fetch data on mount
  useEffect(() => {
    // TODO: Replace with actual service calls when available
    // For now, simulate loading completion with empty data
    const timer = setTimeout(() => {
      setAgents({ data: [], isLoading: false, error: null });
      setChannels({ data: [], isLoading: false, error: null });
      setCost({
        data: { totalCostUSD: 0, recordCount: 0, inputTokens: 0, outputTokens: 0, totalTokens: 0 },
        isLoading: false,
        error: null,
      });
    }, 100);

    return () => clearTimeout(timer);
  }, []);

  // Compute summary from data
  const summary = useMemo<DashboardSummary>(() => ({
    workspaceName,
    total: agents.data?.length ?? 0,
    active: agents.data?.filter((a) => a.state !== 'stopped').length ?? 0,
    working: agents.data?.filter((a) => a.state === 'working').length ?? 0,
    totalCostUSD: cost.data?.totalCostUSD ?? 0,
  }), [workspaceName, agents.data, cost.data]);

  const refresh = useCallback(() => {
    // TODO: Implement refresh when service layer is available
    setAgents((prev) => ({ ...prev, isLoading: true }));
    setChannels((prev) => ({ ...prev, isLoading: true }));
    setCost((prev) => ({ ...prev, isLoading: true }));

    // Simulate refresh
    setTimeout(() => {
      setAgents({ data: [], isLoading: false, error: null });
      setChannels({ data: [], isLoading: false, error: null });
      setCost({
        data: { totalCostUSD: 0, recordCount: 0, inputTokens: 0, outputTokens: 0, totalTokens: 0 },
        isLoading: false,
        error: null,
      });
    }, 500);
  }, []);

  const isLoading = agents.isLoading || channels.isLoading || cost.isLoading;
  const error = agents.error || channels.error || cost.error;

  return {
    summary,
    agents,
    channels,
    cost,
    isLoading,
    error,
    refresh,
  };
}

export default useDashboard;
