import React, { useState, useEffect, useCallback } from 'react';
import { Box, Text, useInput as inkUseInput } from 'ink';
import type { Agent } from '../types';
import { execBc } from '../services/bc';
import { StatusBadge } from '../components/StatusBadge';
import { useFocus } from '../navigation/FocusContext';
import { useAgentDetails } from '../hooks/useAgentDetails';
import { MetricCard } from '../components/MetricCard';

// Safe wrapper for useInput that handles test environments
const useSafeInput = (handler: Parameters<typeof inkUseInput>[0]) => {
  try {
    inkUseInput(handler);
  } catch {
    // Silently fail in test environments
  }
};

/**
 * Normalize task status by replacing cooking metaphors with clearer terms.
 * Issue #970 - Replace cooking terminology from Claude Code status line.
 */
function normalizeTask(task: string | undefined): string {
  if (!task) return '(no task)';
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

interface AgentDetailViewProps {
  agent: Agent;
  onBack?: () => void;
}

export const AgentDetailView: React.FC<AgentDetailViewProps> = ({
  agent,
  onBack,
}) => {
  const [outputLines, setOutputLines] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [inputMode, setInputMode] = useState(false);
  const [messageBuffer, setMessageBuffer] = useState('');
  const [sendStatus, setSendStatus] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'output' | 'details' | 'metrics'>('output');
  const { setFocus } = useFocus();

  // Fetch agent-specific details (costs, activity)
  const { cost, activity } = useAgentDetails(agent.name);

  /**
   * Synchronize focus state with input mode
   *
   * When user enters input mode (presses 'i' or 'm'), we set focus to 'input' area.
   * This prevents global keybinds (q, 1-9, ESC) from triggering during message typing.
   *
   * When user exits input mode (presses Enter or Escape), we set focus to 'view'
   * to keep global navigation disabled while in agent detail view. This ensures that
   * ESC navigates back to agent list (via onBack) rather than to Dashboard.
   */
  useEffect(() => {
    if (inputMode) {
      setFocus('input');
    } else {
      // Keep focus on 'view' to prevent global ESC from going to Dashboard
      setFocus('view');
    }
  }, [inputMode, setFocus]);

  const fetchAgentOutput = useCallback(async () => {
    try {
      const output = await execBc(['agent', 'peek', agent.name, '--lines', '50']);
      const lines = output.split('\n').filter(line => line.trim());
      setOutputLines(lines);
      setError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch agent output';
      setError(message);
    }
  }, [agent.name]);

  useEffect(() => {
    setLoading(true);
    void fetchAgentOutput().finally(() => { setLoading(false); });
  }, [fetchAgentOutput]);

  useEffect(() => {
    const interval = setInterval(() => {
      void fetchAgentOutput();
    }, 2000);
    return () => { clearInterval(interval); };
  }, [fetchAgentOutput]);

  const sendMessage = useCallback(async (message: string) => {
    if (!message.trim()) return;
    try {
      setSendStatus(`Sending to ${agent.name}...`);
      await execBc(['agent', 'send', agent.name, message]);
      setSendStatus(`Sent to ${agent.name}`);
      setMessageBuffer('');
      setTimeout(() => { setSendStatus(null); }, 2000);
      await fetchAgentOutput();
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to send';
      setSendStatus(`Error: ${errorMsg}`);
      setTimeout(() => { setSendStatus(null); }, 3000);
    }
  }, [agent.name, fetchAgentOutput]);

  // Use safe input wrapper that handles test environments gracefully
  useSafeInput((input, key) => {
    if (inputMode) {
      if (key.return) {
        void sendMessage(messageBuffer);
        setInputMode(false);
      } else if (key.escape) {
        setMessageBuffer('');
        setInputMode(false);
      } else if (key.backspace || key.delete) {
        setMessageBuffer(messageBuffer.slice(0, -1));
      } else if (input && !key.ctrl && !key.meta) {
        setMessageBuffer(messageBuffer + input);
      }
    } else {
      if (input === 'i' || input === 'm') {
        setInputMode(true);
      } else if (input === 'q' || key.escape) {
        onBack?.();
      } else if (input === 'r') {
        void fetchAgentOutput();
      } else if (input === '1') {
        setActiveTab('output');
      } else if (input === '2') {
        setActiveTab('details');
      } else if (input === '3') {
        setActiveTab('metrics');
      } else if (key.tab) {
        // Cycle through tabs
        setActiveTab(prev => {
          if (prev === 'output') return 'details';
          if (prev === 'details') return 'metrics';
          return 'output';
        });
      }
    }
  });

  const outputHeight = Math.max(10, 24 - 8);

  return (
    <Box flexDirection="column" width="100%" height="100%">
      {/* Header */}
      <Box flexDirection="row" marginBottom={1} paddingX={1}>
        <Box flexDirection="column" flexGrow={1}>
          <Box>
            <Text bold color="cyan">
              {agent.name}
            </Text>
            <Text dimColor> | Role: {agent.role}</Text>
          </Box>
          <Box>
            <Text>State: </Text>
            <StatusBadge state={agent.state} />
            <Text dimColor> | Task: {normalizeTask(agent.task)}</Text>
          </Box>
        </Box>
      </Box>

      {/* Tab Bar */}
      <Box paddingX={1} marginBottom={1}>
        <TabButton label="Output" tabKey="1" active={activeTab === 'output'} />
        <Text> </Text>
        <TabButton label="Details" tabKey="2" active={activeTab === 'details'} />
        <Text> </Text>
        <TabButton label="Metrics" tabKey="3" active={activeTab === 'metrics'} />
      </Box>

      {/* Tab Content */}
      <Box flexDirection="column" flexGrow={1}>
        {activeTab === 'output' && (
          <>
            <Box
              flexDirection="column"
              flexGrow={1}
              marginBottom={1}
              paddingX={1}
              borderStyle="single"
              borderColor="gray"
              height={outputHeight}
            >
              {loading && outputLines.length === 0 ? (
                <Text color="yellow">Loading agent output...</Text>
              ) : error ? (
                <Text color="red">Error: {error}</Text>
              ) : outputLines.length === 0 ? (
                <Text dimColor>No output yet. Agent may be idle.</Text>
              ) : (
                outputLines.slice(-outputHeight + 2).map((line, idx) => (
                  <Text key={idx} dimColor wrap="truncate">
                    {line}
                  </Text>
                ))
              )}
            </Box>

            <Box
              flexDirection="column"
              height={4}
              marginBottom={1}
              paddingX={1}
              borderStyle="single"
              borderColor={inputMode ? 'cyan' : 'gray'}
            >
              {inputMode ? (
                <Box>
                  <Text color="cyan">{"> "}</Text>
                  <Text>{messageBuffer}</Text>
                  <Text color="cyan">|</Text>
                </Box>
              ) : (
                <Text dimColor>Press i or m to send message</Text>
              )}
              {sendStatus && (
                <Box marginTop={1}>
                  <Text color="green">
                    {sendStatus}
                  </Text>
                </Box>
              )}
            </Box>
          </>
        )}

        {activeTab === 'details' && (
          <Box flexDirection="column" paddingX={1}>
            <DetailRow label="ID" value={agent.id} />
            <DetailRow label="Name" value={agent.name} />
            <DetailRow label="Role" value={<Text color="cyan">{agent.role}</Text>} />
            <DetailRow
              label="State"
              value={<StatusBadge state={agent.state} />}
            />
            <DetailRow label="Session" value={agent.session} />
            {agent.tool && <DetailRow label="Tool" value={agent.tool} />}

            <Box marginY={1}>
              <Text bold color="white">Task</Text>
            </Box>
            <Box paddingLeft={2}>
              <Text wrap="wrap">{normalizeTask(agent.task)}</Text>
            </Box>

            <Box marginY={1}>
              <Text bold color="white">Paths</Text>
            </Box>
            <DetailRow label="Workspace" value={agent.workspace} />
            <DetailRow label="Worktree" value={agent.worktree_dir} />
            <DetailRow label="Memory" value={agent.memory_dir} />

            <Box marginY={1}>
              <Text bold color="white">Timestamps</Text>
            </Box>
            <DetailRow label="Started" value={formatDate(agent.started_at)} />
            <DetailRow label="Updated" value={formatDate(agent.updated_at)} />
          </Box>
        )}

        {activeTab === 'metrics' && (
          <Box flexDirection="column" paddingX={1}>
            {/* Cost Metrics */}
            <Box marginBottom={1}>
              <Text bold color="white">Cost Breakdown</Text>
            </Box>
            <Box flexDirection="row" marginBottom={1}>
              <MetricCard
                label="Total Cost"
                value={cost ? `$${cost.totalCost.toFixed(4)}` : '$0.00'}
                color="green"
              />
              <MetricCard
                label="Input Tokens"
                value={cost ? formatNumber(cost.inputTokens) : '0'}
                color="cyan"
              />
              <MetricCard
                label="Output Tokens"
                value={cost ? formatNumber(cost.outputTokens) : '0'}
                color="cyan"
              />
            </Box>

            {/* Activity Timeline */}
            <Box marginY={1}>
              <Text bold color="white">Recent Activity</Text>
            </Box>
            <Box flexDirection="column" paddingX={1} borderStyle="single" borderColor="gray" minHeight={6}>
              {activity.length === 0 ? (
                <Text dimColor>No recent activity</Text>
              ) : (
                activity.slice(0, 8).map((event, idx) => (
                  <Box key={idx}>
                    <Text dimColor wrap="truncate">{formatTime(event.timestamp)}</Text>
                    <Text color="cyan" wrap="truncate"> [{event.type.split('.').pop()}] </Text>
                    <Text wrap="truncate">{truncateMessage(event.message, 40)}</Text>
                  </Box>
                ))
              )}
            </Box>

            {/* Performance Summary */}
            <Box marginY={1}>
              <Text bold color="white">Session Info</Text>
            </Box>
            <DetailRow label="Uptime" value={formatUptime(agent.started_at)} />
            <DetailRow label="Last Update" value={formatDate(agent.updated_at)} />
            <DetailRow label="Events" value={String(activity.length)} />
          </Box>
        )}
      </Box>

      {/* Footer with keybindings */}
      <Box marginTop={1} paddingX={1}>
        <Text dimColor>
          {inputMode
            ? 'Enter: send | Esc: cancel'
            : '1-3: tabs | Tab: cycle | i: message | r: refresh | q/ESC: back'}
        </Text>
        {loading && <Text color="gray"> (refreshing...)</Text>}
      </Box>
    </Box>
  );
};

// Helper component for detail rows
interface DetailRowProps {
  label: string;
  value: string | React.ReactElement;
}

function DetailRow({ label, value }: DetailRowProps): React.ReactElement {
  return (
    <Box>
      <Text bold>{label}:</Text>
      <Box marginLeft={1} flexShrink={1}>
        <Text wrap="truncate">{value}</Text>
      </Box>
    </Box>
  );
}

// Tab button component
interface TabButtonProps {
  label: string;
  tabKey: string;
  active: boolean;
}

function TabButton({ label, tabKey, active }: TabButtonProps): React.ReactElement {
  return (
    <Box>
      <Text color={active ? 'cyan' : 'gray'} bold={active}>
        [{tabKey}]{label}
      </Text>
    </Box>
  );
}

// Format date for display
function formatDate(dateString: string | undefined): string {
  if (!dateString) return '-';
  try {
    const date = new Date(dateString);
    return date.toLocaleString();
  } catch {
    return dateString;
  }
}

// Format time for activity display (HH:MM:SS)
function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
  } catch {
    return timestamp;
  }
}

// Format large numbers with K/M suffixes
function formatNumber(num: number): string {
  if (num >= 1000000) {
    return `${(num / 1000000).toFixed(1)}M`;
  }
  if (num >= 1000) {
    return `${(num / 1000).toFixed(1)}K`;
  }
  return String(num);
}

// Truncate message to max length
function truncateMessage(message: string, maxLen: number): string {
  if (message.length <= maxLen) return message;
  return message.slice(0, maxLen - 3) + '...';
}

// Format uptime from started_at timestamp
function formatUptime(startedAt: string | undefined): string {
  if (!startedAt) return '-';
  try {
    const started = new Date(startedAt);
    const now = new Date();
    const diffMs = now.getTime() - started.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const mins = diffMins % 60;

    if (diffHours > 0) {
      return `${String(diffHours)}h ${String(mins)}m`;
    }
    return `${String(mins)}m`;
  } catch {
    return '-';
  }
}

export default AgentDetailView;
