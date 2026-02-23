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

import React, { useState, useCallback, useEffect, useMemo, useReducer } from 'react';
import { Box, Text, useInput } from 'ink';
import { useAgents, useDebounce } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
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

// #1601: Consolidated UI state with useReducer
interface UIState {
  selectedIndex: number;
  showDetail: boolean;
  confirmAction: AgentAction | null;
  searchQuery: string;
  searchMode: boolean;
  peekOutput: string[] | null;
  peekLoading: boolean;
  groupedView: boolean;
  collapsedRoles: Set<string>;
}

type UIAction =
  | { type: 'SET_SELECTED_INDEX'; index: number }
  | { type: 'NAVIGATE_UP'; maxIndex: number }
  | { type: 'NAVIGATE_DOWN'; maxIndex: number }
  | { type: 'NAVIGATE_TO_END'; maxIndex: number }
  | { type: 'SHOW_DETAIL' }
  | { type: 'HIDE_DETAIL' }
  | { type: 'SET_CONFIRM_ACTION'; action: AgentAction | null }
  | { type: 'ENTER_SEARCH_MODE' }
  | { type: 'EXIT_SEARCH_MODE' }
  | { type: 'SET_SEARCH_QUERY'; query: string }
  | { type: 'APPEND_SEARCH_CHAR'; char: string }
  | { type: 'BACKSPACE_SEARCH' }
  | { type: 'CLEAR_SEARCH' }
  | { type: 'SET_PEEK_OUTPUT'; output: string[] | null }
  | { type: 'SET_PEEK_LOADING'; loading: boolean }
  | { type: 'TOGGLE_GROUPED_VIEW' }
  | { type: 'TOGGLE_ROLE_COLLAPSE'; role: string };

const initialUIState: UIState = {
  selectedIndex: 0,
  showDetail: false,
  confirmAction: null,
  searchQuery: '',
  searchMode: false,
  peekOutput: null,
  peekLoading: false,
  groupedView: true,
  collapsedRoles: new Set(),
};

function uiReducer(state: UIState, action: UIAction): UIState {
  switch (action.type) {
    case 'SET_SELECTED_INDEX':
      return { ...state, selectedIndex: action.index };
    case 'NAVIGATE_UP':
      return { ...state, selectedIndex: Math.max(0, state.selectedIndex - 1) };
    case 'NAVIGATE_DOWN':
      return { ...state, selectedIndex: Math.min(action.maxIndex, state.selectedIndex + 1) };
    case 'NAVIGATE_TO_END':
      return { ...state, selectedIndex: Math.max(0, action.maxIndex) };
    case 'SHOW_DETAIL':
      return { ...state, showDetail: true };
    case 'HIDE_DETAIL':
      return { ...state, showDetail: false };
    case 'SET_CONFIRM_ACTION':
      return { ...state, confirmAction: action.action };
    case 'ENTER_SEARCH_MODE':
      return { ...state, searchMode: true };
    case 'EXIT_SEARCH_MODE':
      return { ...state, searchMode: false };
    case 'SET_SEARCH_QUERY':
      return { ...state, searchQuery: action.query };
    case 'APPEND_SEARCH_CHAR':
      return { ...state, searchQuery: state.searchQuery + action.char };
    case 'BACKSPACE_SEARCH':
      return { ...state, searchQuery: state.searchQuery.slice(0, -1) };
    case 'CLEAR_SEARCH':
      return { ...state, searchQuery: '', selectedIndex: 0 };
    case 'SET_PEEK_OUTPUT':
      return { ...state, peekOutput: action.output };
    case 'SET_PEEK_LOADING':
      return { ...state, peekLoading: action.loading };
    case 'TOGGLE_GROUPED_VIEW':
      return { ...state, groupedView: !state.groupedView, selectedIndex: 0 };
    case 'TOGGLE_ROLE_COLLAPSE': {
      const next = new Set(state.collapsedRoles);
      if (next.has(action.role)) {
        next.delete(action.role);
      } else {
        next.add(action.role);
      }
      return { ...state, collapsedRoles: next };
    }
    default:
      return state;
  }
}

export const AgentsView: React.FC<AgentsViewProps> = () => {
  const { data: agents, loading, error, refresh } = useAgents();
  const { isCompact, isMinimal } = useResponsiveLayout();
  const isNarrow = isCompact || isMinimal;

  // #1601: UI state consolidated with useReducer
  const [ui, dispatch] = useReducer(uiReducer, initialUIState);
  const {
    selectedIndex, showDetail, confirmAction, searchQuery, searchMode,
    peekOutput, peekLoading, groupedView, collapsedRoles,
  } = ui;

  // Action feedback state - kept separate as it's timer-managed
  const [actionState, setActionState] = useState<ActionState | null>(null);

  // Debounce search query for filtering (issue #1602)
  const debouncedSearchQuery = useDebounce(searchQuery, 300);

  // Use extracted hook for grouping logic (using debounced query for performance)
  const { agentList, stateCounts, visibleItems } = useAgentGroups(
    agents ?? [],
    debouncedSearchQuery,
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
  const { setBreadcrumbs, clearBreadcrumbs } = useNavigation();

  // Manage focus state and breadcrumbs for nested view navigation (#1604)
  // When in search mode, set focus='input' to allow typing special chars (#1692)
  useEffect(() => {
    if (showDetail && selectedAgent) {
      setFocus('view');
      setBreadcrumbs([{ label: selectedAgent.name }]);
    } else if (searchMode) {
      setFocus('input');
      clearBreadcrumbs();
    } else {
      setFocus('main');
      clearBreadcrumbs();
    }
  }, [showDetail, selectedAgent, searchMode, setFocus, setBreadcrumbs, clearBreadcrumbs]);

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
    dispatch({ type: 'SET_PEEK_LOADING', loading: true });
    try {
      const output = await execBc(['agent', 'peek', agentName, '--lines', '8']);
      const lines = output.split('\n').filter((line: string) => line.trim());
      dispatch({ type: 'SET_PEEK_OUTPUT', output: lines.slice(-6) });
    } catch {
      dispatch({ type: 'SET_PEEK_OUTPUT', output: ['(No output available)'] });
    } finally {
      dispatch({ type: 'SET_PEEK_LOADING', loading: false });
    }
  }, []);

  // Keyboard navigation
  useInput((input, key) => {
    // Search mode input handling
    if (searchMode) {
      if (key.return || key.escape) {
        dispatch({ type: 'EXIT_SEARCH_MODE' });
      } else if (key.backspace || key.delete) {
        dispatch({ type: 'BACKSPACE_SEARCH' });
      } else if (input && !key.ctrl && !key.meta) {
        dispatch({ type: 'APPEND_SEARCH_CHAR', char: input });
      }
      return;
    }

    if (showDetail) return;

    // Confirmation mode
    if (confirmAction && selectedAgent) {
      if (input === 'y' || input === 'Y') {
        void executeAction(confirmAction, selectedAgent.name, selectedAgent.role);
        dispatch({ type: 'SET_CONFIRM_ACTION', action: null });
      } else if (input === 'n' || input === 'N' || key.escape) {
        dispatch({ type: 'SET_CONFIRM_ACTION', action: null });
      }
      return;
    }

    // List view navigation
    const maxIndex = visibleItems.length - 1;
    if (key.upArrow || input === 'k') {
      dispatch({ type: 'NAVIGATE_UP', maxIndex });
    } else if (key.downArrow || input === 'j') {
      dispatch({ type: 'NAVIGATE_DOWN', maxIndex });
    } else if (input === 'G') {
      dispatch({ type: 'NAVIGATE_TO_END', maxIndex });
    } else if (input === 'v') {
      dispatch({ type: 'TOGGLE_GROUPED_VIEW' });
    } else if (key.return || input === 'a') {
      if (selectedIndex >= 0 && selectedIndex < visibleItems.length) {
        const item = visibleItems[selectedIndex];
        if (item.type === 'header') {
          dispatch({ type: 'TOGGLE_ROLE_COLLAPSE', role: item.role });
          return;
        }
      }
      if (selectedAgent) {
        dispatch({ type: 'SHOW_DETAIL' });
      }
    } else if (input === 'x' && selectedAgent && selectedAgent.state !== 'stopped') {
      dispatch({ type: 'SET_CONFIRM_ACTION', action: 'stop' });
    } else if (input === 'X' && selectedAgent) {
      dispatch({ type: 'SET_CONFIRM_ACTION', action: 'kill' });
    } else if (input === 'R' && selectedAgent) {
      dispatch({ type: 'SET_CONFIRM_ACTION', action: 'restart' });
    } else if (input === 'p' && selectedAgent) {
      if (peekOutput) {
        dispatch({ type: 'SET_PEEK_OUTPUT', output: null });
      } else {
        void peekAgent(selectedAgent.name);
      }
    } else if (input === '/') {
      dispatch({ type: 'ENTER_SEARCH_MODE' });
    } else if (input === 'c' && searchQuery) {
      dispatch({ type: 'CLEAR_SEARCH' });
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
        onBack={() => { dispatch({ type: 'HIDE_DETAIL' }); }}
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
