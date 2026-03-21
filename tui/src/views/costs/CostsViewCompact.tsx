import { memo } from 'react';
import { Box, Text } from 'ink';
import { InlineProgressBar } from '../../components/ProgressBar';
import { Footer } from '../../components/Footer';
import { getCostIndicator, type CostStatus } from '../../theme/StatusColors';
import { getColorForName } from '../../constants/colors';
import { type useCosts } from '../../hooks';
import { type AgentEntry, type SortMode } from './types';

interface CostsViewCompactProps {
  costs: NonNullable<ReturnType<typeof useCosts>['data']>;
  agentEntries: AgentEntry[];
  selectedIndex: number;
  isSelected: (index: number) => boolean;
  sortMode: SortMode;
  hints: { key: string; label: string }[];
  terminalWidth: number;
}

const CostsViewCompact = memo(function CostsViewCompact({
  costs,
  agentEntries,
  selectedIndex,
  isSelected,
  sortMode,
  hints,
  terminalWidth,
}: CostsViewCompactProps) {
  const agentCount = agentEntries.length;
  const maxCost = agentEntries.length > 0 ? agentEntries[0].cost : 1;

  // Scrolling: at 80x24, content budget is ~16 lines
  // Header(1) + margin(1) + agents + margin(1) + model(1) + stats(1) = agents + 5
  // So max visible agents = 11
  const maxVisible = 11;
  const needsScroll = agentCount > maxVisible;
  let startIdx = 0;
  if (needsScroll) {
    startIdx = Math.max(0, Math.min(selectedIndex - Math.floor(maxVisible / 2), agentCount - maxVisible));
  }
  const visibleAgents = needsScroll ? agentEntries.slice(startIdx, startIdx + maxVisible) : agentEntries;
  const hiddenBelow = needsScroll ? agentCount - (startIdx + maxVisible) : 0;

  // Bar width: name(12) + cost(8) + space(1) + bar + space(1) + pct(4) = 26 + bar
  const barWidth = Math.max(10, terminalWidth - 30);

  const modelEntries = Object.entries(costs.by_model ?? {}).sort(([, a], [, b]) => b - a);

  // Burn rate / projection
  const burnRate = costs.burn_rate ?? 0;
  const projected = costs.projected_total ?? 0;
  const cacheHit = costs.cache_hit_rate ?? 0;
  const billingSpent = costs.billing_window_spent ?? costs.total_cost;

  // Cost status
  const costStatus: CostStatus = burnRate > 50 ? 'critical' : burnRate > 20 ? 'warning' : 'normal';
  const { symbol: costSymbol } = getCostIndicator(costStatus);

  // Sort indicator
  const sortLabel = sortMode === 'cost' ? 'by cost' : sortMode === 'name' ? 'by name' : 'by %';

  return (
    <Box flexDirection="column" paddingX={1} overflow="hidden">
      {/* Header */}
      <Box>
        <Text bold>Costs</Text>
        <Text dimColor> ({agentCount})</Text>
        <Text>  </Text>
        <Text color="yellow" bold>${costs.total_cost.toFixed(2)}</Text>
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
          const displayName = agent.name.length > 10 ? agent.name.slice(0, 9) + '…' : agent.name.padEnd(10);
          const costStr = `$${agent.cost.toFixed(2)}`.padStart(7);
          const pctStr = `${String(agent.percent)}%`.padStart(4);

          return (
            <Box key={agent.name}>
              <Text color={selected ? 'cyan' : undefined} bold={selected}>
                {selected ? '▸ ' : '  '}
              </Text>
              <Text color={selected ? 'cyan' : nameColor} bold={selected} wrap="truncate">
                {displayName}
              </Text>
              <Text> </Text>
              <Text color={selected ? 'cyan' : 'yellow'}>{costStr}</Text>
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

      {/* Model summary */}
      {modelEntries.length > 0 && (
        <Box marginTop={1}>
          {modelEntries.map(([model, cost]) => {
            const pct = costs.total_cost > 0 ? Math.round((cost / costs.total_cost) * 100) : 0;
            const shortName = model.length > 10 ? model.slice(0, 9) + '…' : model;
            return (
              <Box key={model} marginRight={2}>
                <Text color="magenta">{shortName}</Text>
                <Text> ${cost.toFixed(2)}</Text>
                <Text dimColor> ({pct}%)</Text>
              </Box>
            );
          })}
        </Box>
      )}

      {/* Stats line */}
      <Box>
        {cacheHit > 0 && (
          <>
            <Text dimColor>Cache {cacheHit.toFixed(1)}% hit</Text>
            <Text dimColor> │ </Text>
          </>
        )}
        <Text dimColor>Billing ${billingSpent.toFixed(2)} spent</Text>
        <Box flexGrow={1} />
        <Text dimColor>({sortLabel})</Text>
      </Box>

      <Footer hints={hints} />
    </Box>
  );
});

export { CostsViewCompact };
export type { CostsViewCompactProps };
