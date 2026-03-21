/**
 * Details tab for AgentDetailView
 * Shows agent metadata: identity, task, paths, timestamps
 */

import React from 'react';
import { Box, Text } from 'ink';
import type { Agent } from '../../types';
import { StatusBadge } from '../../components/StatusBadge';
import { DetailRow, normalizeTask, formatDate } from './types';

interface AgentDetailsTabProps {
  agent: Agent;
}

export function AgentDetailsTab({ agent }: AgentDetailsTabProps): React.ReactElement {
  return (
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
  );
}
