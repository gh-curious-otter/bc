import React, { useState, useEffect, useCallback } from 'react';
import { Box, Text, useInput as inkUseInput } from 'ink';
import { spawnSync } from 'child_process';
import type { Agent } from '../types';
import { execBc } from '../services/bc';
import { StatusBadge } from '../components/StatusBadge';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { useFocus } from '../navigation/FocusContext';
import { useAgentDetails } from '../hooks/useAgentDetails';
import { MetricCard } from '../components/MetricCard';
import { hasAnsiCodes, isPeekHeader } from '../utils';

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

/**
 * Colorize output line based on content patterns.
 * #1161: Apply semantic colors to agent output for better readability.
 * #1844: Pass through lines that already contain ANSI escape codes from log streaming.
 *
 * Patterns: errors (red), warnings (yellow), success (green), info (cyan)
 */
function colorizeOutputLine(line: string): React.ReactElement {
  // #1844: If line already has ANSI codes from log streaming, render as-is.
  // Ink 4.x renders embedded ANSI escape sequences in Text content.
  if (hasAnsiCodes(line)) {
    return <Text>{line}</Text>;
  }

  const trimmed = line.trim().toLowerCase();

  // Error patterns
  if (
    trimmed.includes('error') ||
    trimmed.includes('failed') ||
    trimmed.includes('exception') ||
    trimmed.startsWith('✗') ||
    trimmed.startsWith('x ')
  ) {
    return <Text color="red">{line}</Text>;
  }

  // Warning patterns
  if (
    trimmed.includes('warning') ||
    trimmed.includes('warn') ||
    trimmed.includes('deprecated') ||
    trimmed.startsWith('⚠')
  ) {
    return <Text color="yellow">{line}</Text>;
  }

  // Success patterns
  if (
    trimmed.includes('success') ||
    trimmed.includes('passed') ||
    trimmed.includes('complete') ||
    trimmed.startsWith('✓') ||
    trimmed.startsWith('✔')
  ) {
    return <Text color="green">{line}</Text>;
  }

  // Tool/command patterns (cyan for actions)
  if (
    trimmed.startsWith('>') ||
    trimmed.startsWith('$') ||
    trimmed.includes('running') ||
    trimmed.includes('executing')
  ) {
    return <Text color="cyan">{line}</Text>;
  }

  // File paths (dim white)
  if (trimmed.match(/^[./~].*\.(tsx?|jsx?|go|py|md|json)$/)) {
    return <Text color="white">{line}</Text>;
  }

  // Default: standard white text (not dimmed)
  return <Text>{line}</Text>;
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
  const [activeTab, setActiveTab] = useState<'output' | 'live' | 'details' | 'metrics'>('output');
  const [liveLines, setLiveLines] = useState<string[]>([]);
  const [scrollOffset, setScrollOffset] = useState(0);
  const [isFollowing, setIsFollowing] = useState(true); // Auto-scroll to bottom
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
      // #1844: Strip peek headers and empty lines from output
      const lines = output.split('\n').filter(line => line.trim() && !isPeekHeader(line));
      setOutputLines(lines);
      setError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch agent output';
      setError(message);
    }
  }, [agent.name]);

  // Live mode fetches more lines and refreshes faster
  const fetchLiveOutput = useCallback(async () => {
    try {
      const output = await execBc(['agent', 'peek', agent.name, '--lines', '200']);
      // #1844: Strip peek headers and empty lines from output
      const lines = output.split('\n').filter(line => line.trim() && !isPeekHeader(line));
      setLiveLines(prevLines => {
        // Auto-scroll to bottom if following and new content arrived
        if (isFollowing && lines.length > prevLines.length) {
          const newOffset = Math.max(0, lines.length - 20);
          setScrollOffset(newOffset);
        }
        return lines;
      });
      setError(null);
    } catch (err) {
      // Silently fail for live mode - don't disrupt the experience
    }
  }, [agent.name, isFollowing]);

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

  // Live mode: faster polling (500ms) when tab is active
  useEffect(() => {
    if (activeTab === 'live') {
      void fetchLiveOutput();
      const interval = setInterval(() => {
        void fetchLiveOutput();
      }, 500);
      return () => { clearInterval(interval); };
    }
    return undefined;
  }, [activeTab, fetchLiveOutput]);

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
      } else if (input === 'a') {
        // #1691: Attach to agent's tmux session directly
        // Suspend TUI and give control to tmux attach
        const bcBin = process.env.BC_BIN ?? 'bc';
        spawnSync(bcBin, ['agent', 'attach', agent.name], {
          stdio: 'inherit',
        });
        // After detach, refresh output
        void fetchAgentOutput();
      } else if (input === 'q' || key.escape) {
        onBack?.();
      } else if (input === 'r') {
        void fetchAgentOutput();
      } else if (input === '1') {
        setActiveTab('output');
      } else if (input === '2') {
        setActiveTab('live');
        setScrollOffset(0); // Reset scroll when switching to live
      } else if (input === '3') {
        setActiveTab('details');
      } else if (input === '4') {
        setActiveTab('metrics');
      } else if (activeTab === 'live' && (input === 'j' || key.downArrow)) {
        // Scroll down in live view
        const maxOffset = Math.max(0, liveLines.length - 20);
        setScrollOffset(prev => {
          const newOffset = Math.min(prev + 1, maxOffset);
          // Re-enable following if at bottom
          if (newOffset >= maxOffset) {
            setIsFollowing(true);
          }
          return newOffset;
        });
      } else if (activeTab === 'live' && (input === 'k' || key.upArrow)) {
        // Scroll up in live view - disable following
        setIsFollowing(false);
        setScrollOffset(prev => Math.max(0, prev - 1));
      } else if (activeTab === 'live' && input === 'g') {
        // Jump to top - disable following
        setIsFollowing(false);
        setScrollOffset(0);
      } else if (activeTab === 'live' && input === 'G') {
        // Jump to bottom - re-enable following
        setIsFollowing(true);
        setScrollOffset(Math.max(0, liveLines.length - 20));
      } else if (activeTab === 'live' && input === 'f') {
        // Toggle follow mode
        setIsFollowing(prev => !prev);
        if (!isFollowing) {
          setScrollOffset(Math.max(0, liveLines.length - 20));
        }
      }
      // Note: Tab removed to allow global view navigation (#1520)
      // Use 1/2/3/4 to switch tabs within this view
    }
  });

  // #1161: Use full available height, don't cap artificially
  const outputHeight = 20;

  return (
    <Box flexDirection="column" width="100%" height="100%" overflow="hidden">
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
            <Text dimColor wrap="truncate"> | Task: {normalizeTask(agent.task)}</Text>
          </Box>
        </Box>
      </Box>

      {/* Tab Bar */}
      <Box paddingX={1} marginBottom={1}>
        <TabButton label="Output" tabKey="1" active={activeTab === 'output'} />
        <Text> </Text>
        <TabButton label="Live" tabKey="2" active={activeTab === 'live'} />
        <Text> </Text>
        <TabButton label="Details" tabKey="3" active={activeTab === 'details'} />
        <Text> </Text>
        <TabButton label="Metrics" tabKey="4" active={activeTab === 'metrics'} />
      </Box>

      {/* Tab Content */}
      <Box flexDirection="column" flexGrow={1}>
        {activeTab === 'output' && (
          <>
            {/* #1161: Output box with bottom-aligned content and preserved colors */}
            <Box
              flexDirection="column"
              flexGrow={1}
              marginBottom={1}
              paddingX={1}
              borderStyle="single"
              borderColor="gray"
              height={outputHeight}
              justifyContent="flex-end"
            >
              {loading && outputLines.length === 0 ? (
                <LoadingIndicator message="Loading agent output..." />
              ) : error ? (
                <Text color="red">Error: {error}</Text>
              ) : outputLines.length === 0 ? (
                <Text dimColor>No output yet. Agent may be idle.</Text>
              ) : (
                outputLines.slice(-outputHeight + 2).map((line, idx) => (
                  <Text key={idx} wrap="truncate">
                    {colorizeOutputLine(line)}
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

        {activeTab === 'live' && (
          <Box
            flexDirection="column"
            flexGrow={1}
            marginBottom={1}
            paddingX={1}
            borderStyle="single"
            borderColor="cyan"
          >
            <Box marginBottom={1}>
              <Text color="cyan" bold>LIVE OUTPUT</Text>
              <Text dimColor> - 500ms refresh | </Text>
              {isFollowing ? (
                <Text color="green">FOLLOWING</Text>
              ) : (
                <Text color="yellow">PAUSED</Text>
              )}
              <Text dimColor> | f: toggle follow</Text>
            </Box>
            <Box flexDirection="column" height={outputHeight + 2} overflow="hidden">
              {liveLines.length === 0 ? (
                <Text dimColor>Waiting for output...</Text>
              ) : (
                liveLines.slice(scrollOffset, scrollOffset + outputHeight).map((line, idx) => (
                  <Text key={idx} wrap="truncate">
                    {colorizeOutputLine(line)}
                  </Text>
                ))
              )}
            </Box>
            {liveLines.length > outputHeight && (
              <Box marginTop={1}>
                <Text dimColor>
                  Lines {scrollOffset + 1}-{Math.min(scrollOffset + outputHeight, liveLines.length)} of {liveLines.length}
                  {scrollOffset === 0 && ' (following)'}
                </Text>
              </Box>
            )}
          </Box>
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
            {agent.log_file && <DetailRow label="Log File" value={agent.log_file} />}

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
        <Text dimColor wrap="truncate">
          {inputMode
            ? 'Enter: send | Esc: cancel'
            : activeTab === 'live'
              ? '1-4: tabs | j/k: scroll | g/G: top/bottom | f: follow | a: attach | q/ESC: back'
              : '1-4: tabs | i: message | a: attach | r: refresh | q/ESC: back'}
        </Text>
        {loading && <Text color="gray"> (refreshing...)</Text>}
      </Box>
    </Box>
  );
};

// Helper component for detail rows with consistent alignment
// #1161: Fixed-width labels for proper column alignment
interface DetailRowProps {
  label: string;
  value: string | React.ReactElement;
  labelWidth?: number;
}

const LABEL_WIDTH = 12; // Consistent label column width

function DetailRow({ label, value, labelWidth = LABEL_WIDTH }: DetailRowProps): React.ReactElement {
  // Pad label to fixed width for alignment
  const paddedLabel = label.padEnd(labelWidth);
  return (
    <Box>
      <Text bold color="gray">{paddedLabel}</Text>
      <Box marginLeft={1} flexShrink={1}>
        {typeof value === 'string' ? (
          <Text wrap="truncate">{value}</Text>
        ) : (
          value
        )}
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
