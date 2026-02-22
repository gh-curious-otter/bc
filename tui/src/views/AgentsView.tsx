/**
 * AgentsView - Agent list and management view
 * Refactored from 614 lines to ~250 lines (#1592)
 *
 * Components extracted to ./agents/:
 * - AgentCard, AgentGroupHeader, AgentList
 * - AgentActions, AgentPeekPanel, AgentConfirmDialog
 * - AgentSearchOverlay
 *
 * Logic extracted to hooks/:
 * - useAgentGroups
 */

import React, { useState, useCallback, useEffect, useMemo } from 'react';
import { Box, Text, useInput } from 'ink';
import { useAgents } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { useResponsiveLayout } from '../hooks/useResponsiveLayout';
import { useAgentGroups } from '../hooks/useAgentGroups';
import { PulseText } from '../components/AnimatedText';
import { AgentDetailView } from './AgentDetailView';
import { execBc } from '../services/bc';
import type { Agent } from '../types';

// Import extracted components
import {
  AgentList,
  AgentActions,
  AgentPeekPanel,
  AgentConfirmDialog,
  AgentSearchOverlay,
  type AgentAction,
} from './agents';

// eslint-disable-next-line @typescript-eslint/no-empty-interface -- AgentsView has no props currently
interface AgentsViewProps {}

/** Action feedback display duration in ms */
const ACTION_FEEDBACK_DURATION = 2500;

interface ActionState {
  action: AgentAction | null;
  target: string | null;
  status: 'pending' | 'success' | 'error';
  message: string;
}

export const AgentsView: React.FC<AgentsViewProps> = () => {
  const { data: agents, loading, error, refresh } = useAgents();
  const { isCompact, isMinimal } = useResponsiveLayout();
  const isNarrow = isCompact || isMinimal;
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showDetail, setShowDetail] = useState(false);
  const [confirmAction, setConfirmAction] = useState<AgentAction | null>(null);
  const [actionState, setActionState] = useState<ActionState | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchMode, setSearchMode] = useState(false);
  const [peekOutput, setPeekOutput] = useState<string[] | null>(null);
  const [peekLoading, setPeekLoading] = useState(false);
  const [groupedView, setGroupedView] = useState(true);
  const [collapsedRoles, setCollapsedRoles] = useState<Set<string>>(new Set());

  // Use extracted hook for grouping logic
  const { agentList, stateCounts, visibleItems } = useAgentGroups(
    agents ?? [],
    searchQuery,
    groupedView,
    collapsedRoles
  );

  // Get selected agent from visible items
  const selectedAgent = useMemo((): Agent | undefined => {
    if (selectedIndex < 0 || selectedIndex >= visibleItems.length) return undefined;
    const item = visibleItems[selectedIndex];
    if (item.type === 'agent') {
      return item.agent;
    }
    return undefined;
  }, [visibleItems, selectedIndex]);

  const { setFocus } = useFocus();

  // Manage focus state for nested view navigation
  useEffect(() => {
    if (showDetail) {
      setFocus('view');
    } else {
      setFocus('main');
    }
  }, [showDetail, setFocus]);

  // Clear action feedback after delay
  const showActionFeedback = useCallback((action: AgentAction, target: string, status: 'success' | 'error', message: string) => {
    setActionState({ action, target, status, message });
    setTimeout(() => { setActionState(null); }, ACTION_FEEDBACK_DURATION);
  }, []);

  // Execute agent action
  const executeAction = useCallback(async (action: AgentAction, agentName: string, role?: string) => {
    try {
      switch (action) {
        case 'start':
          await execBc(['agent', 'start', agentName, '--role', role ?? 'engineer']);
          showActionFeedback(action, agentName, 'success', `Started ${agentName}`);
          break;
        case 'stop':
          await execBc(['agent', 'stop', agentName]);
          showActionFeedback(action, agentName, 'success', `Stopped ${agentName}`);
          break;
        case 'kill':
          await execBc(['agent', 'kill', agentName]);
          showActionFeedback(action, agentName, 'success', `Killed ${agentName}`);
          break;
        case 'restart':
          await execBc(['agent', 'restart', agentName]);
          showActionFeedback(action, agentName, 'success', `Restarted ${agentName}`);
          break;
        case 'attach':
          await execBc(['agent', 'attach', agentName]);
          showActionFeedback(action, agentName, 'success', `Attached to ${agentName}`);
          break;
      }
      void refresh();
    } catch (err) {
      const message = err instanceof Error ? err.message : `Failed to ${action} ${agentName}`;
      showActionFeedback(action, agentName, 'error', message);
    }
  }, [refresh, showActionFeedback]);

  // Peek agent output
  const peekAgent = useCallback(async (agentName: string) => {
    setPeekLoading(true);
    try {
      const output = await execBc(['agent', 'peek', agentName, '--lines', '8']);
      const lines = output.split('\n').filter((line: string) => line.trim());
      setPeekOutput(lines.slice(-6));
    } catch {
      setPeekOutput(['(No output available)']);
    } finally {
      setPeekLoading(false);
    }
  }, []);

  // Keyboard navigation
  useInput((input, key) => {
    // Search mode input handling
    if (searchMode) {
      if (key.return || key.escape) {
        setSearchMode(false);
      } else if (key.backspace || key.delete) {
        setSearchQuery(searchQuery.slice(0, -1));
      } else if (input && !key.ctrl && !key.meta) {
        setSearchQuery(searchQuery + input);
      }
      return;
    }

    if (showDetail) return;

    // Confirmation mode
    if (confirmAction && selectedAgent) {
      if (input === 'y' || input === 'Y') {
        void executeAction(confirmAction, selectedAgent.name, selectedAgent.role);
        setConfirmAction(null);
      } else if (input === 'n' || input === 'N' || key.escape) {
        setConfirmAction(null);
      }
      return;
    }

    // List view navigation
    const maxIndex = visibleItems.length - 1;
    if (key.upArrow || input === 'k') {
      setSelectedIndex((i) => Math.max(0, i - 1));
    } else if (key.downArrow || input === 'j') {
      setSelectedIndex((i) => Math.min(maxIndex, i + 1));
    } else if (input === 'G') {
      setSelectedIndex(Math.max(0, maxIndex));
    } else if (input === 'v') {
      setGroupedView((v) => !v);
      setSelectedIndex(0);
    } else if (key.return || input === 'a') {
      if (selectedIndex >= 0 && selectedIndex < visibleItems.length) {
        const item = visibleItems[selectedIndex];
        if (item.type === 'header') {
          setCollapsedRoles((prev) => {
            const next = new Set(prev);
            if (next.has(item.role)) {
              next.delete(item.role);
            } else {
              next.add(item.role);
            }
            return next;
          });
          return;
        }
      }
      if (selectedAgent) {
        setShowDetail(true);
      }
    } else if (input === 'x' && selectedAgent && selectedAgent.state !== 'stopped') {
      setConfirmAction('stop');
    } else if (input === 'X' && selectedAgent) {
      setConfirmAction('kill');
    } else if (input === 'R' && selectedAgent) {
      setConfirmAction('restart');
    } else if (input === 'p' && selectedAgent) {
      if (peekOutput) {
        setPeekOutput(null);
      } else {
        void peekAgent(selectedAgent.name);
      }
    } else if (input === '/') {
      setSearchMode(true);
    } else if (input === 'c' && searchQuery) {
      setSearchQuery('');
      setSelectedIndex(0);
    } else if (input === 'r') {
      void refresh();
    }
  });

  // Detail view
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- defensive check
  if (showDetail && selectedAgent) {
    return (
      <AgentDetailView
        agent={selectedAgent}
        onBack={() => { setShowDetail(false); }}
      />
    );
  }

  // Search mode overlay
  if (searchMode) {
    return <AgentSearchOverlay searchQuery={searchQuery} isNarrow={isNarrow} />;
  }

  if (loading && agentList.length === 0) {
    return (
      <Box padding={1}>
        <PulseText color="cyan">Loading agents...</PulseText>
      </Box>
    );
  }

  if (error) {
    return (
      <Box padding={1}>
        <Text color="red">Error: {error}</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      {/* Header with state summary */}
      <Box marginBottom={1}>
        <Text bold color="green">Agents ({agentList.length})</Text>
        {stateCounts.working > 0 && (
          <Text color="blue"> ● {stateCounts.working} working</Text>
        )}
        {stateCounts.stuck > 0 && (
          <Text color="yellow"> ⚠ {stateCounts.stuck} stuck</Text>
        )}
        {stateCounts.error > 0 && (
          <Text color="red"> ✗ {stateCounts.error} error</Text>
        )}
        {searchQuery && (
          <Text color="cyan"> [/] &quot;{searchQuery}&quot;</Text>
        )}
        {loading && <PulseText color="gray"> (refreshing...)</PulseText>}
      </Box>

      {/* Action feedback */}
      {actionState && (
        <Box marginBottom={1}>
          <Text color={actionState.status === 'success' ? 'green' : 'red'}>
            {actionState.status === 'success' ? '✓' : '✗'} {actionState.message}
          </Text>
        </Box>
      )}

      {/* Peek output panel */}
      {peekOutput && selectedAgent && (
        <AgentPeekPanel
          agent={selectedAgent}
          output={peekOutput}
          loading={peekLoading}
          isNarrow={isNarrow}
        />
      )}

      {/* Confirmation dialog */}
      {confirmAction && selectedAgent && (
        <AgentConfirmDialog
          action={confirmAction}
          agent={selectedAgent}
          isNarrow={isNarrow}
        />
      )}

      {/* Agent list */}
      <AgentList
        items={visibleItems}
        agents={agentList}
        selectedIndex={selectedIndex}
        groupedView={groupedView}
        collapsedRoles={collapsedRoles}
      />

      {/* Actions bar */}
      {selectedAgent && !confirmAction && (
        <AgentActions agent={selectedAgent} />
      )}

      {/* Footer */}
      <Box marginTop={1}>
        <Text color="gray">
          j/k: nav | v: {groupedView ? 'flat' : 'grouped'} | /: search{searchQuery ? ' | c: clear' : ''} | p: peek | Enter: {groupedView ? 'expand/attach' : 'attach'} | r: refresh | q: back
        </Text>
      </Box>
    </Box>
  );
};

export default AgentsView;
