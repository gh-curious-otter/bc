import React, { useState, useEffect, useCallback } from 'react';
import { Box, Text, useInput as inkUseInput } from 'ink';
import type { Agent } from '../types';
import { execBc } from '../services/bc';
import { StatusBadge } from '../components/StatusBadge';

// Safe wrapper for useInput that handles test environments
const useSafeInput = (handler: Parameters<typeof inkUseInput>[0]) => {
  try {
    inkUseInput(handler);
  } catch {
    // Silently fail in test environments
  }
};

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

  const fetchAgentOutput = useCallback(async () => {
    try {
      const output = await execBc(['agent', 'peek', agent.name, '--tail', '50']);
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
      }
    }
  });

  const outputHeight = Math.max(10, 24 - 8);

  return (
    <Box flexDirection="column" width="100%" height="100%">
      <Box flexDirection="row" marginBottom={1} paddingX={1} height={3}>
        <Box flexDirection="column" flexGrow={1}>
          <Box>
            <Text bold color="cyan">
              {agent.name}
            </Text>
            <Text dimColor> | Role: {agent.role || 'none'}</Text>
          </Box>
          <Box marginTop={1}>
            <Text>State: </Text>
            <StatusBadge state={agent.state} />
            <Text dimColor> | Task: {agent.task || 'none'}</Text>
          </Box>
        </Box>
      </Box>

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
          outputLines.map((line, idx) => (
            <Text key={idx} dimColor>
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
            <Text color="cyan">L</Text>
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

      <Box height={1}>
        <Text dimColor>
          {inputMode
            ? 'Enter: send | Esc: cancel'
            : 'i/m: message | r: refresh | q: back'}
        </Text>
        {loading && <Text color="gray"> (refreshing...)</Text>}
      </Box>

      {/* Details */}
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
          <Text wrap="wrap">{agent.task || '(no task)'}</Text>
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

      {/* Footer with keybindings */}
      <Box marginY={1}>
        <Text color="gray">r: refresh | q: back</Text>
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
      <Box marginLeft={1}>
        <Text>{value}</Text>
      </Box>
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

export default AgentDetailView;
