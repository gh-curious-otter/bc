/**
 * useWorktrees hook - Fetch and manage worktree data
 * Issue #868 - Worktrees tab
 */

import { useState, useEffect, useCallback } from 'react';
import type { Worktree, WorktreeListResponse, BcResult } from '../types';
import { execBc } from '../services/bc';

export interface UseWorktreesOptions {
  /** Polling interval in ms (default: 10000) */
  pollInterval?: number;
  /** Whether to poll automatically (default: true) */
  autoPoll?: boolean;
}

export interface UseWorktreesResult extends BcResult<Worktree[]> {
  /** Total worktree count */
  total: number;
  /** Count of active (OK) worktrees */
  active: number;
  /** Count of orphaned worktrees */
  orphaned: number;
  /** Count of missing worktrees */
  missing: number;
  /** Manually refresh worktree data */
  refresh: () => Promise<void>;
  /** Prune orphaned worktrees */
  prune: (dryRun?: boolean) => Promise<string>;
}

/**
 * Hook to fetch and optionally poll worktree data
 */
export function useWorktrees(options: UseWorktreesOptions = {}): UseWorktreesResult {
  const { pollInterval = 10000, autoPoll = true } = options;

  const [data, setData] = useState<Worktree[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchWorktrees = useCallback(async () => {
    try {
      const output = await execBc(['worktree', 'list']);
      const worktrees: WorktreeListResponse = JSON.parse(output);
      setData(worktrees);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch worktrees');
    } finally {
      setLoading(false);
    }
  }, []);

  const prune = useCallback(async (dryRun = true): Promise<string> => {
    try {
      const args = ['worktree', 'prune'];
      if (!dryRun) {
        args.push('--force');
      }
      const output = await execBc(args);
      // Refresh after pruning
      if (!dryRun) {
        await fetchWorktrees();
      }
      return output;
    } catch (err) {
      throw new Error(err instanceof Error ? err.message : 'Failed to prune worktrees');
    }
  }, [fetchWorktrees]);

  // Initial fetch
  useEffect(() => {
    void fetchWorktrees();
  }, [fetchWorktrees]);

  // Polling
  useEffect(() => {
    if (!autoPoll) return;
    const interval = setInterval(() => { void fetchWorktrees(); }, pollInterval);
    return () => { clearInterval(interval); };
  }, [autoPoll, pollInterval, fetchWorktrees]);

  // Compute counts
  const total = data?.length ?? 0;
  const active = data?.filter(w => w.status === 'OK').length ?? 0;
  const orphaned = data?.filter(w => w.status === 'ORPHANED').length ?? 0;
  const missing = data?.filter(w => w.status === 'MISSING').length ?? 0;

  return {
    data,
    error,
    loading,
    total,
    active,
    orphaned,
    missing,
    refresh: fetchWorktrees,
    prune,
  };
}
