import React from 'react';
import { Text } from 'ink';
import type { AgentState } from '../types';

interface StatusBadgeProps {
  state: AgentState;
}

const stateColors: Record<AgentState, string> = {
  idle: 'gray',
  starting: 'yellow',
  working: 'blue',
  done: 'green',
  stuck: 'red',
  error: 'red',
  stopped: 'gray',
};

const stateSymbols: Record<AgentState, string> = {
  idle: '○',
  starting: '◐',
  working: '●',
  done: '✓',
  stuck: '!',
  error: '✗',
  stopped: '◌',
};

export const StatusBadge: React.FC<StatusBadgeProps> = ({ state }) => {
  const color = stateColors[state] || 'white';
  const symbol = stateSymbols[state] || '?';

  return (
    <Text color={color}>
      {symbol} {state}
    </Text>
  );
};

export default StatusBadge;
