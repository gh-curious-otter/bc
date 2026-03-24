import { useCallback } from "react";
import { api } from "../api/client";
import type { AgentStatsRecord } from "../api/client";
import { usePolling } from "./usePolling";

export function useAgentStats(agentName: string) {
  const fetcher = useCallback(async () => {
    return api.getAgentDockerStats(agentName);
  }, [agentName]);

  const {
    data: stats,
    loading,
    error,
  } = usePolling<AgentStatsRecord[]>(fetcher, 10000);

  return { stats, loading, error };
}
