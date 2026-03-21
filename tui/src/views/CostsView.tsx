/**
 * CostsView - Cost dashboard with horizontal bars and agent drill-down
 * Issue #1882: Cost dashboard design with ccusage integration
 * Issue #1346: Borderless compact layout for 80x24 terminals
 * Issue #1816: Add keybinding hints
 */

import React, { useState, useCallback, useEffect, useMemo } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { ErrorDisplay } from '../components/ErrorDisplay';
import { Footer } from '../components/Footer';
import { Spinner } from '../components/LoadingIndicator';
import { useCosts, useDisableInput, useListNavigation, useLoadingTimeout } from '../hooks';
import { useFocus, useIsOverlayActive } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { CostsViewCompact, CostsViewWide, AgentCostDetail, type SortMode, type AgentEntry } from './costs';

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface CostsViewProps {}

export function CostsView(_props: CostsViewProps = {}): React.ReactElement {
  const { isDisabled: disableInput } = useDisableInput();
  const { setFocus } = useFocus();
  const overlayActive = useIsOverlayActive();
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
  }, { isActive: showDetail && !disableInput && !overlayActive });

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

export default CostsView;
