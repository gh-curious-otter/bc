/**
 * CostsView - Cost dashboard component
 * Issue #1346: Borderless compact layout for 80x24 terminals
 */

import React from 'react';
import { Box, Text } from 'ink';
import { Panel } from './Panel';
import { useCosts } from '../hooks';
import { useResponsiveLayout } from '../hooks/useResponsiveLayout';

interface CostsViewProps {
  /** Disable input handling (useful for testing) */
  disableInput?: boolean;
}

export function CostsView({ disableInput: _disableInput = false }: CostsViewProps): React.ReactElement {
  const { isCompact, isMinimal, isMD } = useResponsiveLayout();
  // #1365: Extend borderless to 100-120 cols (isMD) to prevent box fragmentation
  const isNarrow = isCompact || isMinimal || isMD;

  const { data: costs, loading, error } = useCosts();

  if (loading) {
    return (
      <Box flexDirection="column">
        <Text bold>Costs</Text>
        <Text dimColor>Loading cost data...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Box flexDirection="column">
        <Text bold>Costs</Text>
        <Text color="red">Error: {error}</Text>
      </Box>
    );
  }

  if (!costs) {
    return (
      <Box flexDirection="column">
        <Text bold>Costs</Text>
        <Text dimColor>No cost data available</Text>
      </Box>
    );
  }

  // #1346: Compact borderless layout for narrow terminals (<100 cols)
  if (isNarrow) {
    const agentEntries = Object.entries(costs.by_agent ?? {})
      .sort(([, a], [, b]) => b - a)
      .slice(0, 5);
    const modelEntries = Object.entries(costs.by_model ?? {})
      .sort(([, a], [, b]) => b - a)
      .slice(0, 3);

    return (
      <Box flexDirection="column" padding={1}>
        <Text bold>Cost Dashboard</Text>

        {/* Inline Summary */}
        <Box marginTop={1}>
          <Text dimColor>Total: </Text>
          <Text color="yellow" bold>${costs.total_cost.toFixed(4)}</Text>
          {costs.total_input_tokens > 0 && (
            <>
              <Text> · </Text>
              <Text dimColor>{costs.total_input_tokens.toLocaleString()} in</Text>
              <Text> / </Text>
              <Text dimColor>{costs.total_output_tokens.toLocaleString()} out</Text>
            </>
          )}
        </Box>

        {/* Inline By Agent */}
        {agentEntries.length > 0 && (
          <Box flexDirection="column" marginTop={1}>
            <Text bold dimColor>By Agent:</Text>
            {agentEntries.map(([agent, cost]) => (
              <Text key={agent}>
                <Text color="green">{agent.length > 12 ? agent.slice(0, 11) + '…' : agent}</Text>
                <Text dimColor>: </Text>
                <Text>${cost.toFixed(4)}</Text>
              </Text>
            ))}
            {Object.keys(costs.by_agent ?? {}).length > 5 && (
              <Text dimColor>+{Object.keys(costs.by_agent ?? {}).length - 5} more</Text>
            )}
          </Box>
        )}

        {/* Inline By Model */}
        {modelEntries.length > 0 && (
          <Box flexDirection="column" marginTop={1}>
            <Text bold dimColor>By Model:</Text>
            {modelEntries.map(([model, cost]) => (
              <Text key={model}>
                <Text color="magenta">{model.length > 15 ? model.slice(0, 14) + '…' : model}</Text>
                <Text dimColor>: </Text>
                <Text>${cost.toFixed(4)}</Text>
              </Text>
            ))}
          </Box>
        )}
      </Box>
    );
  }

  // Standard bordered Panel layout for wider terminals
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold>Cost Dashboard</Text>

      {/* Summary */}
      <Panel title="Summary">
        <Box>
          <Text>Total Cost: </Text>
          <Text color="yellow" bold>${costs.total_cost.toFixed(4)}</Text>
        </Box>
        <Box>
          <Text>Input Tokens: </Text>
          {costs.total_input_tokens === 0 && costs.total_cost > 0 ? (
            <Text dimColor>N/A (manual entry)</Text>
          ) : (
            <Text>{costs.total_input_tokens.toLocaleString()}</Text>
          )}
        </Box>
        <Box>
          <Text>Output Tokens: </Text>
          {costs.total_output_tokens === 0 && costs.total_cost > 0 ? (
            <Text dimColor>N/A (manual entry)</Text>
          ) : (
            <Text>{costs.total_output_tokens.toLocaleString()}</Text>
          )}
        </Box>
      </Panel>

      {/* By Agent */}
      <Panel title="By Agent">
        {Object.entries(costs.by_agent ?? {}).length === 0 ? (
          <Text dimColor>No agent costs recorded</Text>
        ) : (
          Object.entries(costs.by_agent ?? {})
            .sort(([, a], [, b]) => b - a)
            .slice(0, 10)
            .map(([agent, cost]) => (
              <Box key={agent}>
                <Text color="green">{agent.padEnd(20)}</Text>
                <Text>${cost.toFixed(4)}</Text>
              </Box>
            ))
        )}
      </Panel>

      {/* By Model */}
      <Panel title="By Model">
        {Object.entries(costs.by_model ?? {}).length === 0 ? (
          <Text dimColor>No model costs recorded</Text>
        ) : (
          Object.entries(costs.by_model ?? {})
            .sort(([, a], [, b]) => b - a)
            .map(([model, cost]) => (
              <Box key={model}>
                <Text color="magenta">{model.padEnd(20)}</Text>
                <Text>${cost.toFixed(4)}</Text>
              </Box>
            ))
        )}
      </Panel>

      {/* By Team */}
      {Object.keys(costs.by_team ?? {}).length > 0 && (
        <Panel title="By Team">
          {Object.entries(costs.by_team ?? {})
            .sort(([, a], [, b]) => b - a)
            .map(([team, cost]) => (
              <Box key={team}>
                <Text color="blue">{team.padEnd(20)}</Text>
                <Text>${cost.toFixed(4)}</Text>
              </Box>
            ))}
        </Panel>
      )}
    </Box>
  );
}

export default CostsView;
