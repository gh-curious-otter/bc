import React from 'react';
import { Box, Text, useInput } from 'ink';
import { useAgent } from '../hooks';
import { StatusBadge } from '../components/StatusBadge';
// Agent type used by useAgent hook

interface AgentDetailViewProps {
  agentName: string;
  onBack?: () => void;
}

const formatDate = (dateStr: string): string => {
  try {
    const date = new Date(dateStr);
    return date.toLocaleString();
  } catch {
    return dateStr;
  }
};

const DetailRow: React.FC<{ label: string; value: React.ReactNode }> = ({
  label,
  value,
}) => (
  <Box>
    <Box width={15}>
      <Text color="gray">{label}:</Text>
    </Box>
    <Box flexGrow={1}>
      {typeof value === 'string' ? <Text>{value}</Text> : value}
    </Box>
  </Box>
);

export const AgentDetailView: React.FC<AgentDetailViewProps> = ({
  agentName,
  onBack,
}) => {
  const { data: agent, loading, error, refresh } = useAgent(agentName);

  // Keyboard navigation
  useInput((input, key) => {
    if (input === 'r') {
      refresh();
    } else if (input === 'q' || key.escape) {
      onBack?.();
    }
  });

  if (loading && !agent) {
    return (
      <Box padding={1}>
        <Text color="yellow">Loading agent details...</Text>
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

  if (!agent) {
    return (
      <Box padding={1}>
        <Text color="red">Agent not found: {agentName}</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      {/* Header */}
      <Box marginBottom={1} borderStyle="single" paddingX={1}>
        <Text bold color="green">
          Agent: {agent.name}
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

export default AgentDetailView;
