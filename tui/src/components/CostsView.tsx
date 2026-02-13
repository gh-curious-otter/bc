/**
 * CostsView - Cost dashboard component
 */

import React from 'react';
import { Box, Text, useStdout } from 'ink';
import { useCosts } from '../hooks';

interface CostsViewProps {
  /** Disable input handling (useful for testing) */
  disableInput?: boolean;
}

export function CostsView({ disableInput: _disableInput = false }: CostsViewProps): React.ReactElement {
  const { stdout } = useStdout();
  const terminalWidth = stdout?.columns ?? 80;

  const { data: costs, loading, error } = useCosts();

  if (loading) {
    return (
      <Box flexDirection="column" width={terminalWidth}>
        <Text bold>Costs</Text>
        <Text dimColor>Loading cost data...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Box flexDirection="column" width={terminalWidth}>
        <Text bold>Costs</Text>
        <Text color="red">Error: {error}</Text>
      </Box>
    );
  }

  if (!costs) {
    return (
      <Box flexDirection="column" width={terminalWidth}>
        <Text bold>Costs</Text>
        <Text dimColor>No cost data available</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" width={terminalWidth} padding={1}>
      <Text bold>Cost Dashboard</Text>

      {/* Summary */}
      <Box marginTop={1} flexDirection="column" borderStyle="single" borderColor="gray" paddingX={1} width={terminalWidth - 2}>
        <Text bold color="cyan">Summary</Text>
        <Box>
          <Text>Total Cost: </Text>
          <Text color="yellow" bold>${costs.total_cost.toFixed(4)}</Text>
        </Box>
        <Box>
          <Text>Input Tokens: </Text>
          <Text>{costs.total_input_tokens.toLocaleString()}</Text>
        </Box>
        <Box>
          <Text>Output Tokens: </Text>
          <Text>{costs.total_output_tokens.toLocaleString()}</Text>
        </Box>
      </Box>

      {/* By Agent */}
      <Box marginTop={1} flexDirection="column" borderStyle="single" borderColor="gray" paddingX={1} width={terminalWidth - 2}>
        <Text bold color="cyan">By Agent</Text>
        {Object.entries(costs.by_agent).length === 0 ? (
          <Text dimColor>No agent costs recorded</Text>
        ) : (
          Object.entries(costs.by_agent)
            .sort(([, a], [, b]) => b - a)
            .slice(0, 10)
            .map(([agent, cost]) => (
              <Box key={agent}>
                <Text color="green">{agent.padEnd(20)}</Text>
                <Text>${cost.toFixed(4)}</Text>
              </Box>
            ))
        )}
      </Box>

      {/* By Model */}
      <Box marginTop={1} flexDirection="column" borderStyle="single" borderColor="gray" paddingX={1} width={terminalWidth - 2}>
        <Text bold color="cyan">By Model</Text>
        {Object.entries(costs.by_model).length === 0 ? (
          <Text dimColor>No model costs recorded</Text>
        ) : (
          Object.entries(costs.by_model)
            .sort(([, a], [, b]) => b - a)
            .map(([model, cost]) => (
              <Box key={model}>
                <Text color="magenta">{model.padEnd(20)}</Text>
                <Text>${cost.toFixed(4)}</Text>
              </Box>
            ))
        )}
      </Box>

      {/* By Team */}
      {Object.keys(costs.by_team).length > 0 && (
        <Box marginTop={1} flexDirection="column" borderStyle="single" borderColor="gray" paddingX={1} width={terminalWidth - 2}>
          <Text bold color="cyan">By Team</Text>
          {Object.entries(costs.by_team)
            .sort(([, a], [, b]) => b - a)
            .map(([team, cost]) => (
              <Box key={team}>
                <Text color="blue">{team.padEnd(20)}</Text>
                <Text>${cost.toFixed(4)}</Text>
              </Box>
            ))}
        </Box>
      )}
    </Box>
  );
}

export default CostsView;
