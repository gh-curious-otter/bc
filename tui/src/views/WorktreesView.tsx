/**
 * WorktreesView - Git worktree management tab (#868)
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { getWorktrees, pruneWorktrees } from '../services/bc';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { useFocus } from '../navigation/FocusContext';
import type { Worktree } from '../types';

interface WorktreesViewProps {
  onBack?: () => void;
}

/**
 * Format path for display - show relative path
 */
function formatPath(fullPath: string): string {
  // Extract .bc/worktrees/... portion
  const match = fullPath.match(/\.bc\/worktrees\/.+$/);
  return match ? match[0] : fullPath;
}

export const WorktreesView: React.FC<WorktreesViewProps> = ({ onBack }) => {
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;

  const [worktrees, setWorktrees] = useState<Worktree[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showDetail, setShowDetail] = useState(false);
  const [showPruneConfirm, setShowPruneConfirm] = useState(false);
  const [pruneResult, setPruneResult] = useState<string | null>(null);
  const [showOrphanedOnly, setShowOrphanedOnly] = useState(false);
  const { setFocus } = useFocus();

  // Manage focus state for nested view navigation
  useEffect(() => {
    if (showDetail) {
      setFocus('view');
    } else {
      setFocus('main');
    }
  }, [showDetail, setFocus]);

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

  const selectedWorktree = filteredWorktrees[selectedIndex] as Worktree | undefined;
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

  // Keyboard navigation
  useInput((input, key) => {
    if (showPruneConfirm) {
      if (input === 'y' || input === 'Y') {
        void handlePrune().catch(() => { /* error handled in handlePrune */ });
      } else if (input === 'n' || input === 'N' || key.escape) {
        setShowPruneConfirm(false);
      }
      return;
    }

    if (showDetail) {
      if (key.escape || input === 'q' || key.return) {
        setShowDetail(false);
      }
      return;
    }

    // List navigation
    if (key.upArrow || input === 'k') {
      setSelectedIndex((i) => Math.max(0, i - 1));
    } else if (key.downArrow || input === 'j') {
      setSelectedIndex((i) => Math.min(filteredWorktrees.length - 1, i + 1));
    } else if (input === 'g') {
      setSelectedIndex(0);
    } else if (input === 'G') {
      setSelectedIndex(Math.max(0, filteredWorktrees.length - 1));
    } else if (key.return) {
      if (selectedWorktree) {
        setShowDetail(true);
      }
    } else if (input === 'p' && hasOrphans) {
      setShowPruneConfirm(true);
    } else if (input === 'o') {
      setShowOrphanedOnly(!showOrphanedOnly);
      setSelectedIndex(0);
    } else if (input === 'r') {
      void fetchWorktrees();
    } else if (input === 'q' || key.escape) {
      onBack?.();
    }
  });

  // Prune confirmation dialog
  if (showPruneConfirm) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text bold color="yellow">Confirm Prune</Text>
        <Box marginTop={1} flexDirection="column" borderStyle="single" borderColor="yellow" padding={1}>
          <Text>This will remove {orphanedWorktrees.length} orphaned worktree(s):</Text>
          <Box marginTop={1} flexDirection="column">
            {orphanedWorktrees.slice(0, 5).map((wt) => (
              <Text key={wt.path} color="red">- {wt.agent}: {formatPath(wt.path)}</Text>
            ))}
            {orphanedWorktrees.length > 5 && (
              <Text dimColor>... and {orphanedWorktrees.length - 5} more</Text>
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
    return (
      <Box padding={1}>
        <Text color="red">Error: {error}</Text>
      </Box>
    );
  }

  // Calculate column widths
  const agentWidth = 15;
  const statusWidth = 10;
  const pathWidth = Math.min(50, terminalWidth - agentWidth - statusWidth - 10);

  return (
    <Box flexDirection="column">
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="blue">Worktrees</Text>
        <Text dimColor> ({activeWorktrees.length} active</Text>
        {orphanedWorktrees.length > 0 && (
          <Text color="yellow">, {orphanedWorktrees.length} orphaned</Text>
        )}
        <Text dimColor>)</Text>
        {loading && <Text color="gray"> (refreshing...)</Text>}
      </Box>

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
          <Text bold color="gray">
            {'AGENT'.padEnd(agentWidth)}
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
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : 'cyan'}
              >
                {wt.agent.slice(0, agentWidth - 1).padEnd(agentWidth)}
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
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : 'yellow'}
              >
                {(wt.agent || '(orphan)').slice(0, agentWidth - 1).padEnd(agentWidth)}
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
          <Box padding={1}>
            <Text dimColor>No worktrees found</Text>
          </Box>
        )}
      </Box>

      {/* Footer */}
      <Box marginTop={1}>
        <Text dimColor>
          j/k: nav | g/G: top/bottom | Enter: details | o: {showOrphanedOnly ? 'show all' : 'orphans only'}
          {hasOrphans ? ' | p: prune' : ''} | r: refresh | q/ESC: back
        </Text>
      </Box>
    </Box>
  );
};

export default WorktreesView;
