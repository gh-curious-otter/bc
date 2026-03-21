/**
 * useAgentDetails hook - Fetch agent-specific details including costs and activity
 * Issue #1048: Agent details expansion with memory, work history, and metrics
 */

import { useState, useEffect, useCallback } from 'react';
import type { LogEntry, CostSummary } from '../types';
import { getLogs, getCostSummary } from '../services/bc';

/** Agent cost breakdown */
export interface AgentCostDetails {
  totalCost: number;
  inputTokens: number;
  outputTokens: number;
}

/** Agent activity event */
export interface AgentActivity {
  timestamp: string;
  type: string;
  message: string;
}

/** Agent details result */
export interface AgentDetailsResult {
  /** Agent cost breakdown */
  cost: AgentCostDetails | null;
  /** Recent activity/logs for this agent */
  activity: AgentActivity[];
  /** Loading state */
  loading: boolean;
  /** Error message if any */
  error: string | null;
  /** Refresh data */
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch agent-specific details (costs, activity)
 * @param agentName - Name of the agent
 * @returns Agent details with costs and activity
 */
export function useAgentDetails(agentName: string): AgentDetailsResult {
  const [cost, setCost] = useState<AgentCostDetails | null>(null);
  const [activity, setActivity] = useState<AgentActivity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchDetails = useCallback(async () => {
    try {
      // Fetch costs and logs in parallel
      const [costSummary, logs] = await Promise.all([
        getCostSummary(),
        getLogs(20, agentName), // Last 20 events for this agent
      ]);

      // Extract agent-specific cost from summary
      const agentCost = extractAgentCost(costSummary, agentName);
      setCost(agentCost);

      // Transform logs to activity
      const agentActivity = transformLogsToActivity(logs);
      setActivity(agentActivity);

      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch agent details');
    } finally {
      setLoading(false);
    }
  }, [agentName]);

  useEffect(() => {
    void fetchDetails();
  }, [fetchDetails]);

  // Poll for updates every 10 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      void fetchDetails();
    }, 10000);
    return () => { clearInterval(interval); };
  }, [fetchDetails]);

  return {
    cost,
    activity,
    loading,
    error,
    refresh: fetchDetails,
  };
}

/**
 * Extract agent-specific cost from cost summary
 */
function extractAgentCost(summary: CostSummary, agentName: string): AgentCostDetails | null {
  // Try agent_costs array first (preferred format)
  const agentCosts = summary.agent_costs;
  if (agentCosts && Array.isArray(agentCosts)) {
    const agentData = agentCosts.find(c => c.agent === agentName);
    if (agentData) {
      return {
        totalCost: agentData.total_cost || 0,
        inputTokens: agentData.input_tokens || 0,
        outputTokens: agentData.output_tokens || 0,
      };
    }
  }

  // Fall back to by_agent record (simple number format)
  const byAgent = summary.by_agent;
  if (byAgent && typeof byAgent === 'object') {
    const agentCost = byAgent[agentName];
    if (typeof agentCost === 'number') {
      return {
        totalCost: agentCost,
        inputTokens: 0,
        outputTokens: 0,
      };
    }
  }

  return null;
}

/**
 * Transform log entries to activity events
 */
function transformLogsToActivity(logs: LogEntry[]): AgentActivity[] {
  return logs.map(log => ({
    timestamp: log.ts, // LogEntry uses 'ts' not 'timestamp'
    type: log.type ?? 'event',
    message: log.message ?? '',
  }));
}

export default useAgentDetails;
