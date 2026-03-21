import { memo } from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../../theme';
import { InlineProgressBar } from '../../components/ProgressBar';
import { Footer } from '../../components/Footer';
import { getColorForName } from '../../constants/colors';
import { type useCosts } from '../../hooks';
import { type AgentEntry } from './types';

interface AgentCostDetailProps {
  agent: AgentEntry;
  costs: NonNullable<ReturnType<typeof useCosts>['data']>;
  hints: { key: string; label: string }[];
}

const AgentCostDetail = memo(function AgentCostDetail({
  agent,
  costs,
  hints,
}: AgentCostDetailProps) {
  const { theme } = useTheme();
  const totalTokensIn = costs.total_input_tokens;
  const totalTokensOut = costs.total_output_tokens;

  // Estimate per-agent tokens proportionally (until per-agent endpoint exists)
  const ratio = costs.total_cost > 0 ? agent.cost / costs.total_cost : 0;
  const estTokensIn = Math.round(totalTokensIn * ratio);
  const estTokensOut = Math.round(totalTokensOut * ratio);

  // Estimate per-agent model breakdown proportionally
  const modelEntries = Object.entries(costs.by_model ?? {}).sort(([, a], [, b]) => b - a);
  const agentModelCosts = modelEntries.map(([model, cost]) => ({
    model,
    cost: cost * ratio,
    percent: costs.total_cost > 0 ? Math.round((cost / costs.total_cost) * 100) : 0,
  }));
  const maxModelCost = agentModelCosts.length > 0 ? agentModelCosts[0].cost : 1;

  return (
    <Box flexDirection="column" paddingX={1} overflow="hidden">
      {/* Header */}
      <Box>
        <Text dimColor>◀ </Text>
        <Text bold color={getColorForName(agent.name)}>{agent.name}</Text>
        <Box flexGrow={1} />
        <Text color={theme.colors.warning} bold>${agent.cost.toFixed(2)}</Text>
      </Box>

      {/* Stats */}
      <Box flexDirection="column" marginTop={1} marginLeft={2}>
        <Box>
          <Text dimColor>{'Tokens     '.padEnd(12)}</Text>
          <Text>{estTokensIn.toLocaleString()} in / {estTokensOut.toLocaleString()} out</Text>
        </Box>
        <Box>
          <Text dimColor>{'API calls  '.padEnd(12)}</Text>
          <Text dimColor>—</Text>
        </Box>
        <Box>
          <Text dimColor>{'% of total '.padEnd(12)}</Text>
          <Text>{agent.percent}%</Text>
        </Box>
      </Box>

      {/* Model Breakdown */}
      {agentModelCosts.length > 0 && (
        <Box flexDirection="column" marginTop={1} marginLeft={2}>
          <Text bold dimColor>Model Breakdown</Text>
          {agentModelCosts.map(({ model, cost, percent }) => {
            const displayModel = model.length > 10 ? model.slice(0, 9) + '…' : model.padEnd(10);
            return (
              <Box key={model}>
                <Text color={theme.colors.accent}>{displayModel}</Text>
                <Text> ${cost.toFixed(2).padStart(7)}</Text>
                <Text> </Text>
                <InlineProgressBar value={cost} max={maxModelCost} width={34} />
                <Text dimColor> {String(percent).padStart(3)}%</Text>
              </Box>
            );
          })}
        </Box>
      )}

      <Box flexGrow={1} />
      <Footer hints={hints} />
    </Box>
  );
});

export { AgentCostDetail };
export type { AgentCostDetailProps };
