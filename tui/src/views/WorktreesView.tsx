/**
 * WorktreesView - Git worktree management tab (#868)
 * Issue #1736: Migrated to useListNavigation hook
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { getWorktrees, pruneWorktrees } from '../services/bc';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { ErrorDisplay } from '../components/ErrorDisplay';
import { HeaderBar } from '../components/HeaderBar';
import { Footer } from '../components/Footer';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { useListNavigation } from '../hooks';
import { DISPLAY_LIMITS } from '../constants';
import type { Worktree } from '../types';

/**
 * Format path for display - show relative path
 */
function formatPath(fullPath: string): string {
  // Extract .bc/worktrees/... portion
  const match = fullPath.match(/\.bc\/worktrees\/.+$/);
  return match ? match[0] : fullPath;
}

export const WorktreesView: React.FC = () => {
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;

  const [worktrees, setWorktrees] = useState<Worktree[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDetail, setShowDetail] = useState(false);
  const [showPruneConfirm, setShowPruneConfirm] = useState(false);
  const [pruneResult, setPruneResult] = useState<string | null>(null);
  const [showOrphanedOnly, setShowOrphanedOnly] = useState(false);
  const { setFocus } = useFocus();
  const { setBreadcrumbs, clearBreadcrumbs } = useNavigation();

  const fetchWorktrees = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getWorktrees();
      setWorktrees(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch worktrees');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchWorktrees();
  }, [fetchWorktrees]);

  // Filter worktrees based on view mode
  const filteredWorktrees = useMemo(() => {
    if (!showOrphanedOnly) return worktrees;
    return worktrees.filter((wt) => wt.status === 'ORPHANED');
  }, [worktrees, showOrphanedOnly]);

  // Separate active and orphaned for display
  const activeWorktrees = useMemo(
    () => filteredWorktrees.filter((wt) => wt.status === 'OK'),
    [filteredWorktrees]
  );
  const orphanedWorktrees = useMemo(
    () => filteredWorktrees.filter((wt) => wt.status !== 'OK'),
    [filteredWorktrees]
  );

  const hasOrphans = orphanedWorktrees.length > 0;

  // Handle prune action
  const handlePrune = useCallback(async () => {
    try {
      const result = await pruneWorktrees(true);
      setPruneResult(result || 'Pruned successfully');
      setShowPruneConfirm(false);
      // Refresh after prune
      await fetchWorktrees();
      // Clear result after delay
      setTimeout(() => { setPruneResult(null); }, 3000);
    } catch (err) {
      setPruneResult(`Error: ${err instanceof Error ? err.message : 'Failed to prune'}`);
    }
  }, [fetchWorktrees]);

  // #1736: Use useListNavigation hook for vim-style navigation
  const {
    selectedIndex,
    selectedItem: selectedWorktree,
    setSelectedIndex,
  } = useListNavigation({
    items: filteredWorktrees,
    onSelect: () => { setShowDetail(true); },
    customKeys: {
      'o': () => {
        setShowOrphanedOnly(!showOrphanedOnly);
        setSelectedIndex(0);
      },
      'p': () => { if (hasOrphans) setShowPruneConfirm(true); },
      'r': () => { void fetchWorktrees(); },
    },
    // Disable navigation when in modal states
    isActive: !showDetail && !showPruneConfirm,
  });

  // Manage focus state and breadcrumbs for nested view navigation (#1604)
  useEffect(() => {
    if (showDetail && selectedWorktree) {
      setFocus('view');
      setBreadcrumbs([{ label: selectedWorktree.agent }]);
    } else {
      setFocus('main');
      clearBreadcrumbs();
    }
  }, [showDetail, selectedWorktree, setFocus, setBreadcrumbs, clearBreadcrumbs]);

  // Handle prune confirmation dialog input
  useInput((input, key) => {
    if (showPruneConfirm) {
      if (input === 'y' || input === 'Y') {
        void handlePrune().catch(() => { /* error handled in handlePrune */ });
      } else if (input === 'n' || input === 'N' || key.escape) {
        setShowPruneConfirm(false);
      }
    } else if (showDetail) {
      if (key.escape || input === 'q' || key.return) {
        setShowDetail(false);
      }
    }
  }, { isActive: showPruneConfirm || showDetail });

  // Prune confirmation dialog
  if (showPruneConfirm) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text bold color="yellow">Confirm Prune</Text>
        <Box marginTop={1} flexDirection="column" borderStyle="single" borderColor="yellow" padding={1}>
          <Text>This will remove {orphanedWorktrees.length} orphaned worktree(s):</Text>
          <Box marginTop={1} flexDirection="column">
            {orphanedWorktrees.slice(0, DISPLAY_LIMITS.ORPHANED_WORKTREES).map((wt) => (
              <Text key={wt.path} color="red">- {wt.agent}: {formatPath(wt.path)}</Text>
            ))}
            {orphanedWorktrees.length > DISPLAY_LIMITS.ORPHANED_WORKTREES && (
              <Text dimColor>... and {orphanedWorktrees.length - DISPLAY_LIMITS.ORPHANED_WORKTREES} more</Text>
            )}
          </Box>
          <Box marginTop={1}>
            <Text bold>Proceed? (y/n)</Text>
          </Box>
        </Box>
      </Box>
    );
  }

  // Detail view
  if (showDetail && selectedWorktree) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text bold color="cyan">Worktree Details</Text>
        <Box marginTop={1} flexDirection="column" borderStyle="single" borderColor="gray" padding={1}>
          <Box>
            <Text bold>Agent: </Text>
            <Text color="cyan">{selectedWorktree.agent}</Text>
          </Box>
          <Box>
            <Text bold>Path: </Text>
            <Text>{selectedWorktree.path}</Text>
          </Box>
          <Box>
            <Text bold>Status: </Text>
            <Text color={selectedWorktree.status === 'OK' ? 'green' : 'red'}>
              {selectedWorktree.status}
            </Text>
          </Box>
          {selectedWorktree.branch && (
            <Box>
              <Text bold>Branch: </Text>
              <Text color="magenta">{selectedWorktree.branch}</Text>
            </Box>
          )}
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Press any key to return</Text>
        </Box>
      </Box>
    );
  }

  if (loading && worktrees.length === 0) {
    return <LoadingIndicator message="Loading worktrees..." />;
  }

  if (error) {
    return <ErrorDisplay error={error} onRetry={() => { void fetchWorktrees(); }} />;
  }

  // Calculate column widths
  const agentWidth = 15;
  const statusWidth = 10;
  const pathWidth = Math.min(50, terminalWidth - agentWidth - statusWidth - 10);

  // Build subtitle with stats
  const worktreeSubtitle = orphanedWorktrees.length > 0
    ? `${String(orphanedWorktrees.length)} orphaned`
    : undefined;

  return (
    <Box flexDirection="column" overflow="hidden">
      {/* Header - using shared HeaderBar component (#1419) */}
      <HeaderBar
        title="Worktrees"
        count={activeWorktrees.length}
        loading={loading}
        color="blue"
        subtitle={worktreeSubtitle}
      />

      {/* Filter indicator */}
      {showOrphanedOnly && (
        <Box marginBottom={1}>
          <Text color="yellow">[Showing orphaned only]</Text>
        </Box>
      )}

      {/* Prune result */}
      {pruneResult && (
        <Box marginBottom={1}>
          <Text color={pruneResult.startsWith('Error') ? 'red' : 'green'}>
            {pruneResult}
          </Text>
        </Box>
      )}

      {/* Worktree table */}
      <Box flexDirection="column" borderStyle="single" borderColor="gray">
        {/* Header */}
        <Box>
          <Text>{'  '}</Text>
          <Text bold color="gray">
            {'AGENT'.padEnd(agentWidth - 2)}
            {'STATUS'.padEnd(statusWidth)}
            {'PATH'}
          </Text>
        </Box>

        {/* Active worktrees */}
        {activeWorktrees.map((wt) => {
          const actualIdx = filteredWorktrees.indexOf(wt);
          const isSelected = actualIdx === selectedIndex;

          return (
            <Box key={wt.path}>
              <Text color={isSelected ? 'cyan' : undefined}>
                {isSelected ? '▸ ' : '  '}
              </Text>
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : 'cyan'}
              >
                {wt.agent.slice(0, agentWidth - 3).padEnd(agentWidth - 2)}
              </Text>
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : 'green'}
              >
                {wt.status.padEnd(statusWidth)}
              </Text>
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : undefined}
                wrap="truncate"
              >
                {formatPath(wt.path).slice(0, pathWidth)}
              </Text>
            </Box>
          );
        })}

        {/* Separator if both types exist */}
        {activeWorktrees.length > 0 && orphanedWorktrees.length > 0 && !showOrphanedOnly && (
          <Box>
            <Text dimColor>{'─'.repeat(terminalWidth - 4)}</Text>
          </Box>
        )}

        {/* Orphaned worktrees */}
        {orphanedWorktrees.map((wt) => {
          const actualIdx = filteredWorktrees.indexOf(wt);
          const isSelected = actualIdx === selectedIndex;

          return (
            <Box key={wt.path}>
              <Text color={isSelected ? 'cyan' : undefined}>
                {isSelected ? '▸ ' : '  '}
              </Text>
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : 'yellow'}
              >
                {(wt.agent || '(orphan)').slice(0, agentWidth - 3).padEnd(agentWidth - 2)}
              </Text>
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : 'red'}
              >
                {wt.status.padEnd(statusWidth)}
              </Text>
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : undefined}
                wrap="truncate"
              >
                {formatPath(wt.path).slice(0, pathWidth)}
              </Text>
            </Box>
          );
        })}

        {filteredWorktrees.length === 0 && (
          <Box padding={1} flexDirection="column">
            <Text dimColor>No worktrees found.</Text>
            <Text dimColor>Worktrees are created automatically when agents start.</Text>
          </Box>
        )}
      </Box>

      {/* Footer */}
      <Footer hints={[
        { key: 'j/k', label: 'nav' },
        { key: 'g/G', label: 'top/bottom' },
        { key: 'Enter', label: 'details' },
        { key: 'o', label: showOrphanedOnly ? 'show all' : 'orphans only' },
        ...(hasOrphans ? [{ key: 'p', label: 'prune' }] : []),
        { key: 'r', label: 'refresh' },
      ]} />
    </Box>
  );
};

export default WorktreesView;
