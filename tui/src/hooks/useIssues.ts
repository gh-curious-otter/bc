/**
 * useIssues hook - Fetch and manage GitHub issues
 * Issue #1754 - Issues View
 */

import { useState, useEffect, useCallback, useMemo } from 'react';
import type { BcResult } from '../types';
import { getIssues, getIssue, closeIssue, assignIssue, type GHIssue } from '../services/bc';

export interface UseIssuesOptions {
  /** Polling interval in ms (default: 30000) */
  pollInterval?: number;
  /** Whether to poll automatically (default: true) */
  autoPoll?: boolean;
  /** Filter by labels (comma-separated) */
  labels?: string;
  /** Filter by assignee */
  assignee?: string;
  /** Issue state filter (default: 'open') */
  state?: 'open' | 'closed' | 'all';
}

export interface UseIssuesResult extends BcResult<GHIssue[]> {
  /** Manually refresh issue data */
  refresh: () => Promise<void>;
  /** Close an issue */
  close: (number: number, reason?: 'completed' | 'not_planned', comment?: string) => Promise<void>;
  /** Assign an issue */
  assign: (number: number, assignee: string) => Promise<void>;
  /** Get filtered issue counts */
  counts: {
    total: number;
    open: number;
    closed: number;
    byLabel: Record<string, number>;
  };
}

/**
 * Hook to fetch and manage GitHub issues
 */
export function useIssues(options: UseIssuesOptions = {}): UseIssuesResult {
  const {
    pollInterval = 30000,
    autoPoll = true,
    labels,
    assignee,
    state = 'open',
  } = options;

  const [data, setData] = useState<GHIssue[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchIssues = useCallback(async () => {
    try {
      const issues = await getIssues(labels, assignee, state);
      setData(issues);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch issues');
    } finally {
      setLoading(false);
    }
  }, [labels, assignee, state]);

  // Initial fetch
  useEffect(() => {
    void fetchIssues();
  }, [fetchIssues]);

  // Polling
  useEffect(() => {
    if (!autoPoll) return;
    const interval = setInterval(() => { void fetchIssues(); }, pollInterval);
    return () => { clearInterval(interval); };
  }, [autoPoll, pollInterval, fetchIssues]);

  // Close an issue
  const close = useCallback(
    async (number: number, reason: 'completed' | 'not_planned' = 'completed', comment?: string) => {
      try {
        await closeIssue(number, reason, comment);
        await fetchIssues();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to close issue');
        throw err;
      }
    },
    [fetchIssues]
  );

  // Assign an issue
  const assign = useCallback(
    async (number: number, assigneeUser: string) => {
      try {
        await assignIssue(number, assigneeUser);
        await fetchIssues();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to assign issue');
        throw err;
      }
    },
    [fetchIssues]
  );

  // Compute counts
  const counts = useMemo(() => {
    const issues = data ?? [];
    const byLabel: Record<string, number> = {};

    for (const issue of issues) {
      for (const label of issue.labels) {
        byLabel[label.name] = (byLabel[label.name] ?? 0) + 1;
      }
    }

    return {
      total: issues.length,
      open: issues.filter(i => i.state === 'OPEN').length,
      closed: issues.filter(i => i.state === 'CLOSED').length,
      byLabel,
    };
  }, [data]);

  return {
    data,
    error,
    loading,
    refresh: fetchIssues,
    close,
    assign,
    counts,
  };
}

export interface UseIssueDetailOptions {
  /** Issue number */
  number: number;
  /** Include comments (default: true) */
  includeComments?: boolean;
}

export interface UseIssueDetailResult extends BcResult<GHIssue> {
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch a single issue with details
 */
export function useIssueDetail(options: UseIssueDetailOptions): UseIssueDetailResult {
  const { number, includeComments = true } = options;

  const [data, setData] = useState<GHIssue | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchIssue = useCallback(async () => {
    try {
      setLoading(true);
      const issue = await getIssue(number, includeComments);
      setData(issue);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch issue');
    } finally {
      setLoading(false);
    }
  }, [number, includeComments]);

  useEffect(() => {
    void fetchIssue();
  }, [fetchIssue]);

  return {
    data,
    error,
    loading,
    refresh: fetchIssue,
  };
}

export default useIssues;
