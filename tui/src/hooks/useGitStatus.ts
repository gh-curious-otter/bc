/**
 * useGitStatus hook - Git status for file explorer
 *
 * RFC 002: File Explorer for TUI
 * Provides git status information for files in a worktree.
 *
 * Features:
 * - Parses `git status --porcelain` output
 * - Returns file-level modification status
 * - Supports caching with configurable TTL
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { execSync } from 'child_process';

/** Git file status */
export type GitFileStatus = 'modified' | 'added' | 'deleted' | 'renamed' | 'untracked' | 'ignored';

/** Git status entry for a file */
export interface GitStatusEntry {
  path: string;
  status: GitFileStatus;
  staged: boolean;
}

/** Summary of git status */
export interface GitStatusSummary {
  modified: number;
  added: number;
  deleted: number;
  untracked: number;
  total: number;
}

export interface UseGitStatusOptions {
  /** Working directory path */
  workingDir: string;
  /** Auto-refresh interval in ms (0 to disable) */
  refreshInterval?: number;
  /** Cache TTL in ms */
  cacheTTL?: number;
}

export interface UseGitStatusResult {
  /** Map of file path to status */
  statusMap: Map<string, GitStatusEntry>;
  /** Summary counts */
  summary: GitStatusSummary;
  /** Loading state */
  loading: boolean;
  /** Error message if any */
  error: string | null;
  /** Get status for a specific file */
  getStatus: (filePath: string) => GitStatusEntry | null;
  /** Check if a file is modified */
  isModified: (filePath: string) => boolean;
  /** Manually refresh status */
  refresh: () => void;
}

/**
 * Parse git status --porcelain output
 * Format: XY PATH or XY ORIG_PATH -> NEW_PATH
 */
function parseGitStatusLine(line: string): GitStatusEntry | null {
  if (line.length < 3) return null;

  const indexStatus = line[0];
  const workTreeStatus = line[1];
  let path = line.slice(3);

  // Handle renamed files (R PATH -> NEWPATH)
  if (path.includes(' -> ')) {
    path = path.split(' -> ')[1];
  }

  // Determine status based on index and worktree status
  let status: GitFileStatus;
  let staged = false;

  // Untracked
  if (indexStatus === '?' && workTreeStatus === '?') {
    status = 'untracked';
  }
  // Ignored
  else if (indexStatus === '!' && workTreeStatus === '!') {
    status = 'ignored';
  }
  // Added (staged)
  else if (indexStatus === 'A') {
    status = 'added';
    staged = true;
  }
  // Deleted
  else if (indexStatus === 'D' || workTreeStatus === 'D') {
    status = 'deleted';
    staged = indexStatus === 'D';
  }
  // Renamed
  else if (indexStatus === 'R') {
    status = 'renamed';
    staged = true;
  }
  // Modified
  else if (indexStatus === 'M' || workTreeStatus === 'M') {
    status = 'modified';
    staged = indexStatus === 'M';
  }
  // Default to modified for any other status
  else {
    status = 'modified';
  }

  return { path, status, staged };
}

/**
 * Execute git status and parse output
 */
function getGitStatus(workingDir: string): {
  entries: GitStatusEntry[];
  error: string | null;
} {
  try {
    const output = execSync('git status --porcelain', {
      cwd: workingDir,
      encoding: 'utf-8',
      timeout: 5000,
    });

    const entries: GitStatusEntry[] = [];
    const lines = output.split('\n').filter((line) => line.length > 0);

    for (const line of lines) {
      const entry = parseGitStatusLine(line);
      if (entry) {
        entries.push(entry);
      }
    }

    return { entries, error: null };
  } catch (err) {
    return {
      entries: [],
      error: err instanceof Error ? err.message : 'Failed to get git status',
    };
  }
}

/**
 * Hook to get git status for files in a worktree
 */
export function useGitStatus(options: UseGitStatusOptions): UseGitStatusResult {
  const { workingDir, refreshInterval = 5000, cacheTTL = 2000 } = options;

  const [statusMap, setStatusMap] = useState<Map<string, GitStatusEntry>>(
    new Map()
  );
  const [summary, setSummary] = useState<GitStatusSummary>({
    modified: 0,
    added: 0,
    deleted: 0,
    untracked: 0,
    total: 0,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Cache tracking
  const lastFetchRef = useRef<number>(0);
  const cacheRef = useRef<Map<string, GitStatusEntry>>(new Map());

  /**
   * Fetch git status and update state
   */
  const fetchStatus = useCallback(() => {
    if (!workingDir) {
      setStatusMap(new Map());
      setSummary({ modified: 0, added: 0, deleted: 0, untracked: 0, total: 0 });
      setLoading(false);
      return;
    }

    // Check cache
    const now = Date.now();
    if (now - lastFetchRef.current < cacheTTL && cacheRef.current.size > 0) {
      return;
    }

    setLoading(true);
    const { entries, error: gitError } = getGitStatus(workingDir);

    if (gitError) {
      setError(gitError);
      setLoading(false);
      return;
    }

    // Build status map
    const newMap = new Map<string, GitStatusEntry>();
    let modified = 0;
    let added = 0;
    let deleted = 0;
    let untracked = 0;

    for (const entry of entries) {
      newMap.set(entry.path, entry);
      switch (entry.status) {
        case 'modified':
          modified++;
          break;
        case 'added':
          added++;
          break;
        case 'deleted':
          deleted++;
          break;
        case 'untracked':
          untracked++;
          break;
      }
    }

    // Update state
    setStatusMap(newMap);
    setSummary({
      modified,
      added,
      deleted,
      untracked,
      total: entries.length,
    });
    setError(null);
    setLoading(false);

    // Update cache
    lastFetchRef.current = now;
    cacheRef.current = newMap;
  }, [workingDir, cacheTTL]);

  // Fetch on mount and when workingDir changes
  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  // Set up auto-refresh
  useEffect(() => {
    if (refreshInterval <= 0) return;

    const interval = setInterval(fetchStatus, refreshInterval);
    return () => { clearInterval(interval); };
  }, [fetchStatus, refreshInterval]);

  /**
   * Get status for a specific file
   */
  const getStatus = useCallback(
    (filePath: string): GitStatusEntry | null => {
      // Try exact match first
      if (statusMap.has(filePath)) {
        return statusMap.get(filePath) ?? null;
      }

      // Try relative path
      const relativePath = filePath.replace(workingDir + '/', '');
      if (statusMap.has(relativePath)) {
        return statusMap.get(relativePath) ?? null;
      }

      return null;
    },
    [statusMap, workingDir]
  );

  /**
   * Check if a file is modified (any non-clean status)
   */
  const isModified = useCallback(
    (filePath: string): boolean => {
      return getStatus(filePath) !== null;
    },
    [getStatus]
  );

  /**
   * Manual refresh
   */
  const refresh = useCallback(() => {
    lastFetchRef.current = 0; // Invalidate cache
    fetchStatus();
  }, [fetchStatus]);

  return {
    statusMap,
    summary,
    loading,
    error,
    getStatus,
    isModified,
    refresh,
  };
}

export default useGitStatus;
