import React from 'react';
import { Box, Text } from 'ink';
import type { Agent } from '../../types';

/** Available agent actions */
export type AgentAction = 'start' | 'stop' | 'kill' | 'restart' | 'attach';

export interface AgentConfirmDialogProps {
  action: AgentAction;
  agent: Agent;
  isNarrow: boolean;
}

/**
 * AgentConfirmDialog - Confirmation dialog for agent actions
 * Prompts user to confirm destructive actions like stop/kill.
 * Extracted from AgentsView (#1592).
 */
export function AgentConfirmDialog({
  action,
  agent,
  isNarrow,
}: AgentConfirmDialogProps): React.ReactElement {
  const getMessage = (): string => {
    switch (action) {
      case 'start':
        return `Start agent "${agent.name}" as ${agent.role}?`;
      case 'stop':
        return `Stop agent "${agent.name}"?`;
      case 'kill':
        return `Kill agent "${agent.name}"? (force terminate)`;
      case 'restart':
        return `Restart agent "${agent.name}"?`;
      default:
        return `${action} agent "${agent.name}"?`;
    }
  };

  return (
    <Box
      marginBottom={1}
      paddingX={isNarrow ? 0 : 1}
      borderStyle={isNarrow ? undefined : 'round'}
      borderColor="yellow"
    >
      <Text color="yellow">{getMessage()} </Text>
      <Text color="green">[y]es</Text>
      <Text color="gray"> / </Text>
      <Text color="red">[n]o</Text>
    </Box>
  );
}

export default AgentConfirmDialog;
