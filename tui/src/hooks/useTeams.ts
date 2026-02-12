/**
 * useTeams hook - Fetch and manage teams data
 * Issue #556 - Teams view
 */

import { useState, useEffect, useCallback } from 'react';
import type { Team, BcResult } from '../types';
import { getTeams, addTeamMember, removeTeamMember } from '../services/bc.js';

export interface UseTeamsOptions {
  /** Polling interval in ms (default: 10000) */
  pollInterval?: number;
  /** Whether to poll automatically (default: true) */
  autoPoll?: boolean;
}

export interface UseTeamsResult extends BcResult<Team[]> {
  /** Manually refresh team data */
  refresh: () => Promise<void>;
  /** Add member to a team */
  addMember: (teamName: string, agentName: string) => Promise<void>;
  /** Remove member from a team */
  removeMember: (teamName: string, agentName: string) => Promise<void>;
}

/**
 * Hook to fetch and manage teams data
 */
export function useTeams(options: UseTeamsOptions = {}): UseTeamsResult {
  const { pollInterval = 10000, autoPoll = true } = options;

  const [data, setData] = useState<Team[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchTeams = useCallback(async () => {
    try {
      const response = await getTeams();
      setData(response.teams || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch teams');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial fetch
  useEffect(() => {
    fetchTeams();
  }, [fetchTeams]);

  // Polling
  useEffect(() => {
    if (!autoPoll) return;
    const interval = setInterval(fetchTeams, pollInterval);
    return () => clearInterval(interval);
  }, [autoPoll, pollInterval, fetchTeams]);

  const addMember = useCallback(
    async (teamName: string, agentName: string) => {
      try {
        await addTeamMember(teamName, agentName);
        // Refresh data after adding member
        await fetchTeams();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to add member');
        throw err;
      }
    },
    [fetchTeams]
  );

  const removeMember = useCallback(
    async (teamName: string, agentName: string) => {
      try {
        await removeTeamMember(teamName, agentName);
        // Refresh data after removing member
        await fetchTeams();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to remove member');
        throw err;
      }
    },
    [fetchTeams]
  );

  return {
    data,
    error,
    loading,
    refresh: fetchTeams,
    addMember,
    removeMember,
  };
}

export default useTeams;
