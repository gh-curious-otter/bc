import React, { useState, useCallback, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { useAgents } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { useResponsiveLayout } from '../hooks/useResponsiveLayout';
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

/** State counts for header summary */
interface StateCounts {
  working: number;
  idle: number;
  stuck: number;
  error: number;
  stopped: number;
}

/** Count agents by state for header summary */
function countAgentStates(agents: Agent[]): StateCounts {
  const counts: StateCounts = { working: 0, idle: 0, stuck: 0, error: 0, stopped: 0 };
  for (const agent of agents) {
    if (agent.state === 'working' || agent.state === 'starting') {
      counts.working++;
    } else if (agent.state === 'idle' || agent.state === 'done') {
      counts.idle++;
    } else if (agent.state === 'stuck') {
      counts.stuck++;
    } else if (agent.state === 'error') {
      counts.error++;
    } else {
      // stopped or other states
      counts.stopped++;
    }
  }
  return counts;
}

/** Role group with agents and stats (#1346) */
interface RoleGroup {
  role: string;
  agents: Agent[];
  working: number;
  idle: number;
  stuck: number;
}

/** Group agents by role for grouped view (#1346) */
function groupAgentsByRole(agents: Agent[]): RoleGroup[] {
  const groups = new Map<string, Agent[]>();

  for (const agent of agents) {
    const role = agent.role;
    const existing = groups.get(role) ?? [];
    existing.push(agent);
    groups.set(role, existing);
  }

  // Convert to array and calculate stats
  const result: RoleGroup[] = [];
  for (const [role, roleAgents] of groups) {
    const counts = countAgentStates(roleAgents);
    result.push({
      role,
      agents: roleAgents,
      working: counts.working,
      idle: counts.idle,
      stuck: counts.stuck,
    });
  }

  // Sort by role name (engineers first, then alphabetically)
  return result.sort((a, b) => {
    if (a.role === 'engineer') return -1;
    if (b.role === 'engineer') return 1;
    return a.role.localeCompare(b.role);
  });
}

/**
 * Normalize task status by replacing cooking metaphors with clearer terms.
 * Issue #970 - Replace cooking terminology from Claude Code status line.
 */
function normalizeTask(task: string | undefined): string {
  if (!task) return '-';
  // #1364 Issue 3: Normalize cooking/quirky terms to clear status verbs
  const replacements: [string, string][] = [
    ['Sautéed', 'Working'],
    ['Sauteed', 'Working'], // ASCII fallback
    ['Brewed', 'Done'],
    ['Cooked', 'Processed'],
    ['Cogitated', 'Thinking'],
    ['Marinated', 'Idle'],
    ['Frolicking', 'Active'],
    ['Grooving', 'Active'],
  ];
  for (const [old, replacement] of replacements) {
    if (task.includes(old)) {
      return task.replace(old, replacement);
    }
  }
  return task;
}

/**
 * Abbreviate role names for compact display (#1364)
 * product-manager → PM, tech-lead → TL, engineer → Eng
 */
function abbreviateRole(role: string): string {
  const abbreviations: Record<string, string> = {
    'product-manager': 'PM',
    'tech-lead': 'TL',
    'engineer': 'Eng',
    'manager': 'Mgr',
    'root': 'Root',
  };
  return abbreviations[role] ?? role;
}

/** Available agent actions */
type AgentAction = 'start' | 'stop' | 'kill' | 'restart' | 'attach';

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
  const { isCompact, isMinimal } = useResponsiveLayout();
  const isNarrow = isCompact || isMinimal; // #1346: Borderless at <100 cols
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showDetail, setShowDetail] = useState(false);
  const [confirmAction, setConfirmAction] = useState<AgentAction | null>(null);
  const [actionState, setActionState] = useState<ActionState | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchMode, setSearchMode] = useState(false);
  const [peekOutput, setPeekOutput] = useState<string[] | null>(null);
  const [peekLoading, setPeekLoading] = useState(false);
  // #1346: Grouped view mode and collapsed roles
  const [groupedView, setGroupedView] = useState(true);
  const [collapsedRoles, setCollapsedRoles] = useState<Set<string>>(new Set());

  // Filter agents by search query
  const agentList = React.useMemo(() => {
    const list = agents ?? [];
    if (!searchQuery) return list;
    const query = searchQuery.toLowerCase();
    return list.filter(
      (agent) =>
        agent.name.toLowerCase().includes(query) ||
        agent.role.toLowerCase().includes(query) ||
        agent.state.toLowerCase().includes(query)
    );
  }, [agents, searchQuery]);

  // Calculate state counts for header summary (#1331)
  const stateCounts = React.useMemo(() => countAgentStates(agentList), [agentList]);

  // #1346: Group agents by role for grouped view
  const roleGroups = React.useMemo(() => groupAgentsByRole(agentList), [agentList]);

  /** Item types for grouped view navigation (#1346) */
  type GroupedItem = { type: 'header'; role: string; group: RoleGroup } | { type: 'agent'; agent: Agent; role: string };

  // #1346: Build flat list of visible items for navigation in grouped view
  const visibleItems = React.useMemo((): GroupedItem[] => {
    if (!groupedView) {
      // Return agents wrapped as GroupedItem for consistent typing
      return agentList.map((agent) => ({ type: 'agent' as const, agent, role: agent.role }));
    }

    const items: GroupedItem[] = [];
    for (const group of roleGroups) {
      items.push({ type: 'header', role: group.role, group });
      if (!collapsedRoles.has(group.role)) {
        for (const agent of group.agents) {
          items.push({ type: 'agent', agent, role: group.role });
        }
      }
    }
    return items;
  }, [groupedView, roleGroups, collapsedRoles, agentList]);

  // Get selected agent from visible items
  const selectedAgent = React.useMemo((): Agent | undefined => {
    if (selectedIndex < 0 || selectedIndex >= visibleItems.length) return undefined;
    const item = visibleItems[selectedIndex];
    if (item.type === 'agent') {
      return item.agent;
    }
    return undefined;
  }, [visibleItems, selectedIndex]);
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
  const executeAction = useCallback(async (action: AgentAction, agentName: string, role?: string) => {
    try {
      switch (action) {
        case 'start':
          // Start requires role - use the agent's existing role
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

  // Peek agent output (#1331)
  const peekAgent = useCallback(async (agentName: string) => {
    setPeekLoading(true);
    try {
      const output = await execBc(['agent', 'peek', agentName, '--lines', '8']);
      const lines = output.split('\n').filter((line: string) => line.trim());
      setPeekOutput(lines.slice(-6)); // Show last 6 lines
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

    if (showDetail) {
      // Detail view handles its own keybinds via AgentDetailView
      return;
    }

    // Confirmation mode
    if (confirmAction && selectedAgent) {
      if (input === 'y' || input === 'Y') {
        // Pass role for start action
        void executeAction(confirmAction, selectedAgent.name, selectedAgent.role);
        setConfirmAction(null);
      } else if (input === 'n' || input === 'N' || key.escape) {
        setConfirmAction(null);
      }
      return;
    }

    // List view navigation
    const maxIndex = groupedView ? visibleItems.length - 1 : agentList.length - 1;
    if (key.upArrow || input === 'k') {
      setSelectedIndex((i) => Math.max(0, i - 1));
    } else if (key.downArrow || input === 'j') {
      setSelectedIndex((i) => Math.min(maxIndex, i + 1));
    } else if (input === 'G') {
      setSelectedIndex(Math.max(0, maxIndex));
    } else if (input === 'v') {
      // #1346: Toggle grouped view mode
      setGroupedView((v) => !v);
      setSelectedIndex(0);
    } else if (key.return || input === 'a') {
      // #1346: Handle Enter on role header (toggle collapse)
      if (selectedIndex >= 0 && selectedIndex < visibleItems.length) {
        const item = visibleItems[selectedIndex];
        if (item.type === 'header') {
          // Toggle collapse for this role
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
      // View agent details / attach
      if (selectedAgent) {
        setShowDetail(true);
      }
    } else if (input === 'x' && selectedAgent && selectedAgent.state !== 'stopped') {
      // Stop running agent (with confirmation)
      setConfirmAction('stop');
    } else if (input === 'X' && selectedAgent) {
      // Kill agent (with confirmation)
      setConfirmAction('kill');
    } else if (input === 'R' && selectedAgent) {
      // Restart agent (with confirmation) - also works as "start" for stopped agents
      setConfirmAction('restart');
    } else if (input === 'p' && selectedAgent) {
      // Peek agent output (#1331)
      if (peekOutput) {
        setPeekOutput(null); // Toggle off if already showing
      } else {
        void peekAgent(selectedAgent.name);
      }
    } else if (input === '/') {
      // Enter search mode
      setSearchMode(true);
    } else if (input === 'c' && searchQuery) {
      // Clear search
      setSearchQuery('');
      setSelectedIndex(0);
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
        <Text>{abbreviateRole(agent.role)}</Text>
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

  // Search mode overlay
  // #1346: Borderless at narrow widths
  if (searchMode) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text bold>Search Agents</Text>
        <Box marginTop={1} borderStyle={isNarrow ? undefined : 'single'} borderColor="cyan" paddingX={1}>
          <Text color="cyan">{'> '}</Text>
          <Text>{searchQuery}</Text>
          <Text color="cyan">|</Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Enter to confirm, Esc to cancel</Text>
        </Box>
      </Box>
    );
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
      {/* Header with state summary (#1331) */}
      <Box marginBottom={1}>
        <Text bold color="green">
          Agents ({agentList.length})
        </Text>
        {/* State counts summary */}
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

      {/* Peek output panel (#1331, #1346: Borderless at <100 cols) */}
      {peekOutput && selectedAgent && (
        <Box
          marginBottom={1}
          paddingX={isNarrow ? 0 : 1}
          borderStyle={isNarrow ? undefined : 'single'}
          borderColor="cyan"
          flexDirection="column"
        >
          <Box marginBottom={1}>
            <Text bold color="cyan">Peek: {selectedAgent.name}</Text>
            <Text dimColor> (press p to close)</Text>
          </Box>
          {peekLoading ? (
            <Text dimColor>Loading...</Text>
          ) : (
            peekOutput.map((line, idx) => (
              <Text key={idx} wrap="truncate" dimColor>{line}</Text>
            ))
          )}
        </Box>
      )}

      {/* Confirmation dialog (#1346: Borderless at <100 cols) */}
      {confirmAction && selectedAgent && (
        <Box marginBottom={1} paddingX={isNarrow ? 0 : 1} borderStyle={isNarrow ? undefined : 'round'} borderColor="yellow">
          <Text color="yellow">
            {confirmAction === 'start' && `Start agent "${selectedAgent.name}" as ${selectedAgent.role}?`}
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

      {/* Agents View - Grouped or Table (#1346) */}
      {groupedView ? (
        <Box flexDirection="column">
          {visibleItems.map((item, idx) => {
            const isSelected = idx === selectedIndex;

            if (item.type === 'header') {
              // Role header row
              const isCollapsed = collapsedRoles.has(item.role);
              return (
                <Box key={`header-${item.role}`}>
                  <Text color={isSelected ? 'cyan' : 'white'} bold>
                    {isSelected ? '▸ ' : '  '}
                    {isCollapsed ? '▶' : '▼'}{' '}
                  </Text>
                  <Text bold color={isSelected ? 'cyan' : 'white'}>
                    {item.role.toUpperCase()} ({item.group.agents.length})
                  </Text>
                  {item.group.working > 0 && (
                    <Text color="blue"> ● {item.group.working}</Text>
                  )}
                  {item.group.stuck > 0 && (
                    <Text color="yellow"> ⚠ {item.group.stuck}</Text>
                  )}
                </Box>
              );
            }

            // Agent row (type === 'agent')
            return (
              <Box key={`agent-${item.agent.name}`} marginLeft={2}>
                <Text color={isSelected ? 'cyan' : undefined}>
                  {isSelected ? '▸ ' : '  '}
                </Text>
                <Text color={isSelected ? 'cyan' : undefined}>
                  {item.agent.name.length > 12 ? item.agent.name.slice(0, 11) + '…' : item.agent.name.padEnd(12)}
                </Text>
                <Text> </Text>
                <StatusBadge state={item.agent.state} />
                <Text> </Text>
                <Text wrap="truncate" dimColor>
                  {normalizeTask(item.agent.task).slice(0, 30)}
                </Text>
              </Box>
            );
          })}
        </Box>
      ) : (
        <Table
          data={agentList}
          columns={columns}
          selectedIndex={selectedIndex}
        />
      )}

      {/* Inline action bar for selected agent (#1331 - updated keybindings) */}
      {selectedAgent && !confirmAction && (
        <Box marginTop={1} paddingX={1}>
          <Text dimColor>Actions: </Text>
          <Text color="cyan">[p]</Text>
          <Text dimColor> peek </Text>
          {selectedAgent.state !== 'stopped' && selectedAgent.state !== 'error' && (
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
          <Text color="green">[R]</Text>
          <Text dimColor> restart </Text>
          <Text color="cyan">[Enter]</Text>
          <Text dimColor> attach</Text>
        </Box>
      )}

      {/* Footer with keybindings (#1331, #1346) */}
      <Box marginTop={1}>
        <Text color="gray">
          j/k: nav | v: {groupedView ? 'flat' : 'grouped'} | /: search{searchQuery ? ' | c: clear' : ''} | p: peek | Enter: {groupedView ? 'expand/attach' : 'attach'} | r: refresh | q: back
        </Text>
      </Box>
    </Box>
  );
};

export default AgentsView;
