import { memo } from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../../theme';
import { InlineProgressBar } from '../../components/ProgressBar';
import { Panel } from '../../components/Panel';
import { Footer } from '../../components/Footer';
import { getCostIndicator, type CostStatus } from '../../theme/StatusColors';
import { getColorForName } from '../../constants/colors';
import { type useCosts } from '../../hooks';
import { type AgentEntry, type SortMode } from './types';

interface CostsViewWideProps {
  costs: NonNullable<ReturnType<typeof useCosts>['data']>;
  agentEntries: AgentEntry[];
  selectedIndex: number;
  isSelected: (index: number) => boolean;
  sortMode: SortMode;
  hints: { key: string; label: string }[];
  terminalWidth: number;
}

const CostsViewWide = memo(function CostsViewWide({
  costs,
  agentEntries,
  selectedIndex,
  isSelected,
  sortMode,
  hints,
  terminalWidth,
}: CostsViewWideProps) {
  const { theme } = useTheme();
  const agentCount = agentEntries.length;
  const maxCost = agentEntries.length > 0 ? agentEntries[0].cost : 1;

  // More room at 120+ width
  const maxVisible = 15;
  const needsScroll = agentCount > maxVisible;
  let startIdx = 0;
  if (needsScroll) {
    startIdx = Math.max(0, Math.min(selectedIndex - Math.floor(maxVisible / 2), agentCount - maxVisible));
  }
  const visibleAgents = needsScroll ? agentEntries.slice(startIdx, startIdx + maxVisible) : agentEntries;
  const hiddenBelow = needsScroll ? agentCount - (startIdx + maxVisible) : 0;

  const barWidth = Math.max(20, terminalWidth - 35);

  const burnRate = costs.burn_rate ?? 0;
  const projected = costs.projected_total ?? 0;
  const cacheHit = costs.cache_hit_rate ?? 0;
  const billingSpent = costs.billing_window_spent ?? costs.total_cost;

  const costStatus: CostStatus = burnRate > 50 ? 'critical' : burnRate > 20 ? 'warning' : 'normal';
  const { symbol: costSymbol } = getCostIndicator(costStatus);

  const modelEntries = Object.entries(costs.by_model ?? {}).sort(([, a], [, b]) => b - a);

  const sortLabel = sortMode === 'cost' ? 'by cost' : sortMode === 'name' ? 'by name' : 'by %';

  return (
    <Box flexDirection="column" paddingX={1} overflow="hidden">
      {/* Header */}
      <Box>
        <Text bold>Costs</Text>
        <Text dimColor> ({agentCount})</Text>
        <Text>  </Text>
        <Text color={theme.colors.warning} bold>${costs.total_cost.toFixed(2)}</Text>
        <Text dimColor> total</Text>
        <Box flexGrow={1} />
        {burnRate > 0 && (
          <Text dimColor>
            {costSymbol} ${burnRate.toFixed(2)}/hr → ${projected.toFixed(2)}
          </Text>
        )}
      </Box>

      {/* Agent list */}
      <Box flexDirection="column" marginTop={1}>
        {visibleAgents.map((agent, visIdx) => {
          const actualIdx = startIdx + visIdx;
          const selected = isSelected(actualIdx);
          const nameColor = getColorForName(agent.name);
          const displayName = agent.name.length > 12 ? agent.name.slice(0, 11) + '…' : agent.name.padEnd(12);
          const costStr = `$${agent.cost.toFixed(2)}`.padStart(8);
          const pctStr = `${String(agent.percent)}%`.padStart(4);

          return (
            <Box key={agent.name}>
              <Text color={selected ? theme.colors.primary : undefined} bold={selected}>
                {selected ? '▸ ' : '  '}
              </Text>
              <Text color={selected ? theme.colors.primary : nameColor} bold={selected} wrap="truncate">
                {displayName}
              </Text>
              <Text> </Text>
              <Text color={selected ? theme.colors.primary : theme.colors.warning}>{costStr}</Text>
              <Text> </Text>
              <InlineProgressBar
                value={agent.cost}
                max={maxCost}
                width={barWidth}
              />
              <Text dimColor>{pctStr}</Text>
            </Box>
          );
        })}
        {hiddenBelow > 0 && (
          <Text dimColor>  ↓ {hiddenBelow} more</Text>
        )}
      </Box>

      {/* Side-by-side Models + Billing panels */}
      <Box marginTop={1}>
        <Panel title="Models" width="50%">
          {modelEntries.map(([model, cost]) => {
            const pct = costs.total_cost > 0 ? Math.round((cost / costs.total_cost) * 100) : 0;
            const maxModelCost = modelEntries.length > 0 ? modelEntries[0][1] : 1;
            return (
              <Box key={model}>
                <Text color={theme.colors.accent} wrap="truncate">{model.padEnd(10)}</Text>
                <Text> ${cost.toFixed(2).padStart(7)}</Text>
                <Text> </Text>
                <InlineProgressBar value={cost} max={maxModelCost} width={13} />
                <Text dimColor> {String(pct).padStart(3)}%</Text>
              </Box>
            );
          })}
          {modelEntries.length === 0 && <Text dimColor>No model data</Text>}
        </Panel>
        <Panel title="Billing" width="50%">
          <Box>
            <Text>Spent     </Text>
            <Text color={theme.colors.warning}>${billingSpent.toFixed(2)}</Text>
          </Box>
          {burnRate > 0 && (
            <Box>
              <Text>Rate      </Text>
              <Text>${burnRate.toFixed(2)}/hr</Text>
            </Box>
          )}
          {projected > 0 && (
            <Box>
              <Text>Projected </Text>
              <Text>${projected.toFixed(2)}</Text>
            </Box>
          )}
          {cacheHit > 0 && (
            <Box>
              <Text>Cache     </Text>
              <Text color={theme.colors.success}>{cacheHit.toFixed(1)}% hit</Text>
            </Box>
          )}
        </Panel>
      </Box>

      {/* Sort indicator + footer */}
      <Box>
        <Box flexGrow={1} />
        <Text dimColor>({sortLabel})</Text>
      </Box>

      <Footer hints={[...hints, { key: 'm', label: 'models' }]} />
    </Box>
  );
});

export { CostsViewWide };
export type { CostsViewWideProps };
