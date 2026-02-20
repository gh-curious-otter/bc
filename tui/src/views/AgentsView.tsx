import React, { useState, useCallback, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { useAgents } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { Table } from '../components/Table';
import type { Column } from '../components/Table';
import { StatusBadge } from '../components/StatusBadge';
import { PulseText } from '../components/AnimatedText';
import { AgentDetailView } from './AgentDetailView';
import { execBc } from '../services/bc';
import type { Agent } from '../types';

interface AgentsViewProps {
  onBack?: () => void;
}

/** Action feedback display duration in ms */
const ACTION_FEEDBACK_DURATION = 2500;

/**
 * Normalize task status by replacing cooking metaphors with clearer terms.
 * Issue #970 - Replace cooking terminology from Claude Code status line.
 */
function normalizeTask(task: string | undefined): string {
  if (!task) return '-';
  const replacements: [string, string][] = [
    ['Sautéed', 'Working'],
    ['Sauteed', 'Working'], // ASCII fallback
    ['Cooked', 'Processed'],
    ['Cogitated', 'Thinking'],
    ['Marinated', 'Idle'],
    ['Frolicking', 'Active'],
  ];
  for (const [old, replacement] of replacements) {
    if (task.includes(old)) {
      return task.replace(old, replacement);
    }
  }
  return task;
}

/** Available agent actions */
type AgentAction = 'stop' | 'kill' | 'restart' | 'attach';

interface ActionState {
  action: AgentAction | null;
  target: string | null;
  status: 'pending' | 'success' | 'error';
  message: string;
}

export const AgentsView: React.FC<AgentsViewProps> = ({
  onBack,
}) => {
  const { data: agents, loading, error, refresh } = useAgents();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showDetail, setShowDetail] = useState(false);
  const [confirmAction, setConfirmAction] = useState<AgentAction | null>(null);
  const [actionState, setActionState] = useState<ActionState | null>(null);
  const agentList = agents ?? [];
  const selectedAgent = agentList[selectedIndex] as typeof agentList[number] | undefined;
  const { setFocus } = useFocus();

  // Manage focus state for nested view navigation
  // When showing detail view, set focus='view' to prevent global ESC from firing
  // This fixes ESC hierarchy: agent detail → ESC → agent list (not Dashboard)
  useEffect(() => {
    if (showDetail) {
      setFocus('view');
    } else {
      // Restore focus to 'main' when returning to list view
      setFocus('main');
    }
  }, [showDetail, setFocus]);

  // Clear action feedback after delay
  const showActionFeedback = useCallback((action: AgentAction, target: string, status: 'success' | 'error', message: string) => {
    setActionState({ action, target, status, message });
    setTimeout(() => { setActionState(null); }, ACTION_FEEDBACK_DURATION);
  }, []);

  // Execute agent action
  const executeAction = useCallback(async (action: AgentAction, agentName: string) => {
    try {
      switch (action) {
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

  // Keyboard navigation
  useInput((input, key) => {
    if (showDetail) {
      // Detail view handles its own keybinds via AgentDetailView
      return;
    }

    // Confirmation mode
    if (confirmAction && selectedAgent) {
      if (input === 'y' || input === 'Y') {
        void executeAction(confirmAction, selectedAgent.name);
        setConfirmAction(null);
      } else if (input === 'n' || input === 'N' || key.escape) {
        setConfirmAction(null);
      }
      return;
    }

    // List view navigation
    if (key.upArrow || input === 'k') {
      setSelectedIndex((i) => Math.max(0, i - 1));
    } else if (key.downArrow || input === 'j') {
      setSelectedIndex((i) => Math.min(agentList.length - 1, i + 1));
    } else if (input === 'g') {
      setSelectedIndex(0);
    } else if (input === 'G') {
      setSelectedIndex(Math.max(0, agentList.length - 1));
    } else if (key.return || input === 'a') {
      // View agent details / attach
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- defensive check for empty list
      if (selectedAgent) {
        setShowDetail(true);
      }
    } else if (input === 'x' && selectedAgent) {
      // Stop agent (with confirmation)
      setConfirmAction('stop');
    } else if (input === 'X' && selectedAgent) {
      // Kill agent (with confirmation)
      setConfirmAction('kill');
    } else if (input === 'R' && selectedAgent) {
      // Restart agent (with confirmation)
      setConfirmAction('restart');
    } else if (input === 'r') {
      void refresh();
    } else if (input === 'q' || key.escape) {
      onBack?.();
    }
  });

  // If showing detail view, render AgentDetailView instead
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- defensive check for empty list
  if (showDetail && selectedAgent) {
    return (
      <AgentDetailView
        agent={selectedAgent}
        onBack={() => { setShowDetail(false); }}
      />
    );
  }

  // Column widths: 14+10+10+32 = 66 (fits 80-col terminal)
  const columns: Column<Agent>[] = [
    {
      key: 'name',
      header: 'Name',
      width: 14,
      render: (agent) => (
        <Text>{agent.name.length > 12 ? agent.name.slice(0, 11) + '…' : agent.name}</Text>
      ),
    },
    {
      key: 'role',
      header: 'Role',
      width: 10,
      render: (agent) => (
        <Text>{agent.role.length > 8 ? agent.role.slice(0, 7) + '…' : agent.role}</Text>
      ),
    },
    {
      key: 'state',
      header: 'State',
      width: 10,
      render: (agent) => <StatusBadge state={agent.state} />,
    },
    {
      key: 'task',
      header: 'Task',
      width: 32,
      render: (agent) => (
        <Text wrap="truncate">
          {normalizeTask(agent.task).slice(0, 30)}
        </Text>
      ),
    },
  ];

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
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="green">
          Agents ({agentList.length})
        </Text>
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

      {/* Confirmation dialog */}
      {confirmAction && selectedAgent && (
        <Box marginBottom={1} paddingX={1} borderStyle="round" borderColor="yellow">
          <Text color="yellow">
            {confirmAction === 'stop' && `Stop agent "${selectedAgent.name}"?`}
            {confirmAction === 'kill' && `Kill agent "${selectedAgent.name}"? (force terminate)`}
            {confirmAction === 'restart' && `Restart agent "${selectedAgent.name}"?`}
            {' '}
          </Text>
          <Text color="green">[y]es</Text>
          <Text color="gray"> / </Text>
          <Text color="red">[n]o</Text>
        </Box>
      )}

      {/* Agents Table */}
      <Table
        data={agentList}
        columns={columns}
        selectedIndex={selectedIndex}
      />

      {/* Inline action bar for selected agent */}
      {selectedAgent && !confirmAction && (
        <Box marginTop={1} paddingX={1}>
          <Text dimColor>Actions: </Text>
          {selectedAgent.state === 'working' && (
            <>
              <Text color="yellow">[x]</Text>
              <Text dimColor> stop </Text>
            </>
          )}
          {selectedAgent.state !== 'stopped' && (
            <>
              <Text color="red">[X]</Text>
              <Text dimColor> kill </Text>
            </>
          )}
          {selectedAgent.state === 'stopped' || selectedAgent.state === 'error' ? (
            <>
              <Text color="green">[R]</Text>
              <Text dimColor> restart </Text>
            </>
          ) : null}
          <Text color="cyan">[a/Enter]</Text>
          <Text dimColor> details</Text>
        </Box>
      )}

      {/* Footer with keybindings */}
      <Box marginTop={1}>
        <Text color="gray">
          j/k: navigate | g/G: top/bottom | a/Enter: details | x: stop | X: kill | R: restart | r: refresh | q/ESC: back
        </Text>
      </Box>
    </Box>
  );
};

export default AgentsView;
