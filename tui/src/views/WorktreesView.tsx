/**
 * WorktreesView - Git worktree management view
 * Issue #868 - Worktrees tab
 */

import { useState } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { useWorktrees } from '../hooks';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';
import { Footer } from '../components/Footer';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { ErrorDisplay } from '../components/ErrorDisplay';
import type { Worktree, WorktreeStatus } from '../types';

export interface WorktreesViewProps {
  onBack?: () => void;
}

/**
 * Get status badge state from worktree status
 */
function getStatusState(status: WorktreeStatus): string {
  switch (status) {
    case 'OK': return 'working';
    case 'MISSING': return 'error';
    case 'ORPHANED': return 'idle';
    default: return 'idle';
  }
}

/**
 * Get status color
 */
function getStatusColor(status: WorktreeStatus): string {
  switch (status) {
    case 'OK': return 'green';
    case 'MISSING': return 'red';
    case 'ORPHANED': return 'yellow';
    default: return 'gray';
  }
}

/**
 * WorktreesView - Display and manage agent worktrees
 */
export function WorktreesView({ onBack }: WorktreesViewProps) {
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;
  const {
    data: worktrees,
    loading,
    error,
    total,
    active,
    orphaned,
    missing,
    refresh,
    prune,
  } = useWorktrees();

  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showConfirm, setShowConfirm] = useState(false);
  const [pruneResult, setPruneResult] = useState<string | null>(null);
  const [pruning, setPruning] = useState(false);

  // Sort worktrees: OK first, then MISSING, then ORPHANED
  const sortedWorktrees = [...(worktrees ?? [])].sort((a, b) => {
    const order: Record<WorktreeStatus, number> = { OK: 0, MISSING: 1, ORPHANED: 2 };
    return order[a.status] - order[b.status];
  });

  // Keyboard navigation
  useInput((input, key) => {
    if (showConfirm) {
      if (input === 'y' || input === 'Y') {
        setShowConfirm(false);
        setPruning(true);
        prune(false)
          .then((result) => {
            setPruneResult(result || 'Pruned orphaned worktrees');
            setTimeout(() => setPruneResult(null), 5000);
          })
          .catch((err: unknown) => {
            const msg = err instanceof Error ? err.message : String(err);
            setPruneResult(`Error: ${msg}`);
            setTimeout(() => setPruneResult(null), 5000);
          })
          .finally(() => setPruning(false));
      } else if (input === 'n' || input === 'N' || key.escape) {
        setShowConfirm(false);
      }
      return;
    }

    // Navigation
    if (key.downArrow || input === 'j') {
      setSelectedIndex((i) => Math.min(i + 1, sortedWorktrees.length - 1));
    }
    if (key.upArrow || input === 'k') {
      setSelectedIndex((i) => Math.max(i - 1, 0));
    }
    if (input === 'g') {
      setSelectedIndex(0);
    }
    if (input === 'G') {
      setSelectedIndex(Math.max(0, sortedWorktrees.length - 1));
    }

    // Actions
    if (input === 'p' && orphaned > 0) {
      setShowConfirm(true);
    }
    if (input === 'r') {
      void refresh();
    }
    if (input === 'q' || key.escape) {
      onBack?.();
    }
  });

  if (error) {
    return <ErrorDisplay error={error} onRetry={() => { void refresh(); }} />;
  }

  if (loading && !worktrees) {
    return <LoadingIndicator message="Loading worktrees..." />;
  }

  // Responsive column widths
  const agentWidth = Math.min(16, Math.floor((terminalWidth - 20) * 0.25));
  const pathWidth = Math.min(40, Math.floor((terminalWidth - 20) * 0.55));

  const selectedWorktree = sortedWorktrees[selectedIndex];

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="blue">Worktrees</Text>
        <Text> · </Text>
        <Text color="green">{active} active</Text>
        {orphaned > 0 && (
          <>
            <Text> · </Text>
            <Text color="yellow">{orphaned} orphaned</Text>
          </>
        )}
        {missing > 0 && (
          <>
            <Text> · </Text>
            <Text color="red">{missing} missing</Text>
          </>
        )}
        {loading && <Text color="cyan"> (refreshing...)</Text>}
      </Box>

      {/* Prune Confirmation */}
      {showConfirm && (
        <Box
          borderStyle="double"
          borderColor="yellow"
          padding={1}
          marginBottom={1}
        >
          <Text bold color="yellow">
            Prune {orphaned} orphaned worktree{orphaned > 1 ? 's' : ''}?
          </Text>
          <Text> (y/n)</Text>
        </Box>
      )}

      {/* Prune Result */}
      {pruneResult && (
        <Box marginBottom={1}>
          <Text color={pruneResult.startsWith('Error') ? 'red' : 'green'}>
            {pruneResult}
          </Text>
        </Box>
      )}

      {/* Pruning Indicator */}
      {pruning && (
        <Box marginBottom={1}>
          <Text color="yellow">Pruning worktrees...</Text>
        </Box>
      )}

      {/* Worktree List */}
      <Panel title={`All Worktrees (${total})`}>
        {sortedWorktrees.length === 0 ? (
          <Text dimColor>No worktrees found</Text>
        ) : (
          <Box flexDirection="column">
            {/* Header row */}
            <Box marginBottom={1}>
              <Box width={3}><Text> </Text></Box>
              <Box width={agentWidth}><Text bold dimColor>AGENT</Text></Box>
              <Box width={pathWidth}><Text bold dimColor>PATH</Text></Box>
              <Box width={10}><Text bold dimColor>STATUS</Text></Box>
            </Box>

            {/* Worktree rows */}
            {sortedWorktrees.map((wt, index) => (
              <WorktreeRow
                key={wt.agent}
                worktree={wt}
                selected={index === selectedIndex}
                agentWidth={agentWidth}
                pathWidth={pathWidth}
              />
            ))}
          </Box>
        )}
      </Panel>

      {/* Selected Worktree Details */}
      {selectedWorktree && (
        <Box marginTop={1} flexDirection="column">
          <Text bold color="cyan">Details</Text>
          <Box marginLeft={1} flexDirection="column">
            <Text>
              <Text dimColor>Agent: </Text>
              <Text>{selectedWorktree.agent}</Text>
            </Text>
            <Text>
              <Text dimColor>Path: </Text>
              <Text>{selectedWorktree.path}</Text>
            </Text>
            <Text>
              <Text dimColor>Status: </Text>
              <Text color={getStatusColor(selectedWorktree.status)}>
                {selectedWorktree.status}
              </Text>
            </Text>
            {selectedWorktree.branch && (
              <Text>
                <Text dimColor>Branch: </Text>
                <Text color="magenta">{selectedWorktree.branch}</Text>
              </Text>
            )}
          </Box>
        </Box>
      )}

      {/* Footer */}
      <Footer
        hints={[
          { key: 'j/k', label: 'navigate' },
          ...(orphaned > 0 ? [{ key: 'p', label: 'prune orphans' }] : []),
          { key: 'r', label: 'refresh' },
          { key: 'q', label: 'back' },
        ]}
      />
    </Box>
  );
}

interface WorktreeRowProps {
  worktree: Worktree;
  selected: boolean;
  agentWidth: number;
  pathWidth: number;
}

function WorktreeRow({ worktree, selected, agentWidth, pathWidth }: WorktreeRowProps) {
  const truncatedAgent = worktree.agent.length > agentWidth - 2
    ? worktree.agent.slice(0, agentWidth - 3) + '…'
    : worktree.agent;

  const truncatedPath = worktree.path.length > pathWidth - 2
    ? '…' + worktree.path.slice(-(pathWidth - 3))
    : worktree.path;

  return (
    <Box>
      <Box width={3}>
        <Text color={selected ? 'cyan' : undefined}>
          {selected ? '▸ ' : '  '}
        </Text>
      </Box>
      <Box width={agentWidth}>
        <Text color={selected ? 'cyan' : undefined} bold={selected}>
          {truncatedAgent}
        </Text>
      </Box>
      <Box width={pathWidth}>
        <Text dimColor>{truncatedPath}</Text>
      </Box>
      <Box width={10}>
        <StatusBadge state={getStatusState(worktree.status)} />
      </Box>
    </Box>
  );
}

export default WorktreesView;
