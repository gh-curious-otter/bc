/**
 * CostsView - Cost dashboard with horizontal bars and agent drill-down
 * Issue #1882: Cost dashboard design with ccusage integration
 * Issue #1346: Borderless compact layout for 80x24 terminals
 * Issue #1816: Add keybinding hints
 */

import React, { useState, useCallback, useEffect, useMemo, memo } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { InlineProgressBar } from '../components/ProgressBar';
import { Panel } from '../components/Panel';
import { ErrorDisplay } from '../components/ErrorDisplay';
import { Footer } from '../components/Footer';
import { Spinner } from '../components/LoadingIndicator';
import { useCosts, useDisableInput, useListNavigation, useLoadingTimeout } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { getCostIndicator, type CostStatus } from '../theme/StatusColors';
import { getColorForName } from '../constants/colors';

type SortMode = 'cost' | 'name' | 'percent';

interface AgentEntry {
  name: string;
  cost: number;
  percent: number;
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface CostsViewProps {}

export function CostsView(_props: CostsViewProps = {}): React.ReactElement {
  const { isDisabled: disableInput } = useDisableInput();
  const { setFocus } = useFocus();
  const { setBreadcrumbs, clearBreadcrumbs } = useNavigation();
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;
  const isWide = terminalWidth >= 120;

  const { data: costs, loading, error, refresh } = useCosts();

  const [showDetail, setShowDetail] = useState(false);
  const [sortMode, setSortMode] = useState<SortMode>('cost');

  // Build sorted agent entries
  const agentEntries = useMemo<AgentEntry[]>(() => {
    if (!costs?.by_agent) return [];
    const total = costs.total_cost || 1;
    const entries = Object.entries(costs.by_agent).map(([name, cost]) => ({
      name,
      cost,
      percent: Math.round((cost / total) * 100),
    }));

    switch (sortMode) {
      case 'name':
        return entries.sort((a, b) => a.name.localeCompare(b.name));
      case 'percent':
      case 'cost':
      default:
        return entries.sort((a, b) => b.cost - a.cost);
    }
  }, [costs, sortMode]);

  // List navigation
  const handleSelect = useCallback(() => {
    setShowDetail(true);
  }, []);

  const handleCycleSort = useCallback(() => {
    setSortMode((prev) => {
      if (prev === 'cost') return 'name';
      if (prev === 'name') return 'percent';
      return 'cost';
    });
  }, []);

  const handleRefresh = useCallback(() => {
    void refresh();
  }, [refresh]);

  const { selectedIndex, isSelected } = useListNavigation({
    items: agentEntries,
    onSelect: handleSelect,
    disabled: disableInput || showDetail,
    customKeys: {
      s: handleCycleSort,
      r: handleRefresh,
    },
  });

  // Focus/breadcrumb management
  useEffect(() => {
    if (showDetail && agentEntries[selectedIndex]) {
      setFocus('view');
      setBreadcrumbs([{ label: agentEntries[selectedIndex].name }]);
    } else {
      setFocus('main');
      clearBreadcrumbs();
    }
  }, [showDetail, selectedIndex, agentEntries, setFocus, setBreadcrumbs, clearBreadcrumbs]);

  // Detail view input handling
  useInput((input, key) => {
    if (showDetail) {
      if (key.escape || input === 'q') {
        setShowDetail(false);
      }
      if (input === 'r') {
        handleRefresh();
      }
    }
  }, { isActive: showDetail && !disableInput });

  // Keybinding hints
  const mainHints = [
    { key: 'j/k', label: 'nav' },
    { key: 'Enter', label: 'detail' },
    { key: 's', label: 'sort' },
    { key: 'r', label: 'refresh' },
  ];

  const detailHints = [
    { key: 'Esc/q', label: 'back' },
    { key: 'r', label: 'refresh' },
  ];

  // #1898: Track loading duration for timeout messages
  const loadingElapsed = useLoadingTimeout(loading && !costs);

  // #1898: Skeleton state during initial load (no data yet)
  if (loading && !costs) {
    // After 10s: timeout message with retry
    if (loadingElapsed >= 10) {
      return (
        <Box flexDirection="column" paddingX={1}>
          <Box>
            <Text bold>Costs</Text>
            <Text>  </Text>
            <Text color="yellow">⚠ Data unavailable</Text>
          </Box>
          <Box flexDirection="column" marginTop={1}>
            <Text dimColor>Cost data could not be loaded.</Text>
            <Text dimColor>This usually means ccusage is slow or not installed.</Text>
            <Text dimColor>Press [r] to retry.</Text>
          </Box>
          <Footer hints={[{ key: 'r', label: 'refresh' }]} />
        </Box>
      );
    }

    // Skeleton with spinner and placeholder rows
    const loadingMsg = loadingElapsed >= 5
      ? 'Taking longer than expected...'
      : 'Fetching cost analytics...';

    return (
      <Box flexDirection="column" paddingX={1}>
        <Box>
          <Text bold>Costs</Text>
          <Text dimColor> (—)</Text>
          <Box flexGrow={1} />
          <Spinner />
          <Text> {loadingMsg}</Text>
        </Box>
        <Box flexDirection="column" marginTop={1}>
          <Text dimColor>  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─</Text>
          <Text dimColor>  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─</Text>
          <Text dimColor>  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─</Text>
        </Box>
        <Footer hints={mainHints} />
      </Box>
    );
  }

  if (error && !costs) {
    return <ErrorDisplay error={error} onRetry={handleRefresh} />;
  }

  if (!costs) {
    return (
      <Box flexDirection="column" paddingX={1}>
        <Text bold>Costs</Text>
        <Text dimColor>No cost data available</Text>
        <Footer hints={mainHints} />
      </Box>
    );
  }

  // Detail sub-view
  if (showDetail && agentEntries[selectedIndex]) {
    const agent = agentEntries[selectedIndex];
    return (
      <AgentCostDetail
        agent={agent}
        costs={costs}
        hints={detailHints}
      />
    );
  }

  // Main view
  if (isWide) {
    return (
      <CostsViewWide
        costs={costs}
        agentEntries={agentEntries}
        selectedIndex={selectedIndex}
        isSelected={isSelected}
        sortMode={sortMode}
        hints={mainHints}
        terminalWidth={terminalWidth}
      />
    );
  }

  return (
    <CostsViewCompact
      costs={costs}
      agentEntries={agentEntries}
      selectedIndex={selectedIndex}
      isSelected={isSelected}
      sortMode={sortMode}
      hints={mainHints}
      terminalWidth={terminalWidth}
    />
  );
}

// ============================================================================
// Compact layout (80x24)
// ============================================================================

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
          const pctStr = `${agent.percent}%`.padStart(4);

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

// ============================================================================
// Wide layout (120x30)
// ============================================================================

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
          const displayName = agent.name.length > 12 ? agent.name.slice(0, 11) + '…' : agent.name.padEnd(12);
          const costStr = `$${agent.cost.toFixed(2)}`.padStart(8);
          const pctStr = `${agent.percent}%`.padStart(4);

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

      {/* Side-by-side Models + Billing panels */}
      <Box marginTop={1}>
        <Panel title="Models" width="50%">
          {modelEntries.map(([model, cost]) => {
            const pct = costs.total_cost > 0 ? Math.round((cost / costs.total_cost) * 100) : 0;
            const maxModelCost = modelEntries.length > 0 ? modelEntries[0][1] : 1;
            return (
              <Box key={model}>
                <Text color="magenta" wrap="truncate">{model.padEnd(10)}</Text>
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
            <Text color="yellow">${billingSpent.toFixed(2)}</Text>
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
              <Text color="green">{cacheHit.toFixed(1)}% hit</Text>
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

// ============================================================================
// Agent cost detail sub-view
// ============================================================================

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
        <Text color="yellow" bold>${agent.cost.toFixed(2)}</Text>
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
                <Text color="magenta">{displayModel}</Text>
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

export default CostsView;
