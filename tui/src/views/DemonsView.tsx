/**
 * DemonsView - Scheduled tasks list view
 * Issue #554 - Demons list view
 */

import React, { useState, useEffect, useMemo, useRef } from 'react';
import { Box, Text } from 'ink';
import { useDemons, useDebounce, useDisableInput, useListNavigation } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { StatusBadge } from '../components/StatusBadge';
import { HeaderBar } from '../components/HeaderBar';
import { ViewWrapper } from '../components/ViewWrapper';
import type { Demon } from '../types';

/** Duration in ms to show action errors before auto-clearing */
const ERROR_DISPLAY_DURATION = 3000;

// #1594: Using empty interface for future extensibility, props removed
// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface DemonsViewProps {}

/**
 * Format cron schedule to human-readable string
 */
function formatSchedule(schedule: string): string {
  // Common patterns
  if (schedule === '* * * * *') return 'every minute';
  if (schedule === '0 * * * *') return 'every hour';
  if (schedule.startsWith('*/')) {
    const match = schedule.match(/^\*\/(\d+) \* \* \* \*$/);
    if (match) return `every ${match[1]} min`;
  }
  if (schedule.match(/^0 \d+ \* \* \*$/)) {
    const hour = schedule.split(' ')[1];
    return `daily at ${hour}:00`;
  }
  return schedule;
}

/**
 * Format relative time for last/next run
 * Issue #1362: Fix invalid date calculation (739667d ago bug)
 */
function formatRelativeTime(timestamp?: string): string {
  if (!timestamp || timestamp === '0' || timestamp === '') return '-';
  try {
    const date = new Date(timestamp);
    // Validate the parsed date - NaN check and sanity check for epoch/invalid dates
    if (isNaN(date.getTime())) return '-';
    // If date is before year 2000, it's likely invalid (epoch or parse error)
    if (date.getFullYear() < 2000) return '-';

    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(Math.abs(diffMs) / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    const prefix = diffMs < 0 ? 'in ' : '';
    const suffix = diffMs >= 0 ? ' ago' : '';

    if (diffMins < 1) return 'now';
    if (diffMins < 60) return `${prefix}${String(diffMins)}m${suffix}`;
    if (diffHours < 24) return `${prefix}${String(diffHours)}h${suffix}`;
    // Cap days at 365 to avoid absurd values
    if (diffDays > 365) return diffMs < 0 ? '>1y' : '>1y ago';
    return `${prefix}${String(diffDays)}d${suffix}`;
  } catch {
    return '-';
  }
}

/**
 * DemonsView - Display list of scheduled tasks (demons)
 *
 * Features:
 * - List all configured demons
 * - Show schedule, status, run history
 * - Keyboard navigation (j/k, e/d to enable/disable, r to run)
 */
export function DemonsView(_props: DemonsViewProps = {}): React.ReactElement {
  // #1594: Use context instead of prop drilling
  const { isDisabled: disableInput } = useDisableInput();
  const { data: demons, loading, error, enabled, refresh, enable, disable, run } = useDemons();
  const { setFocus } = useFocus();
  const [actionError, setActionError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  // Debounce search query for filtering (issue #1602)
  const debouncedSearchQuery = useDebounce(searchQuery, 300);

  // Filter demons by search query (using debounced query for performance)
  const filteredDemons = useMemo(() => {
    const list = demons ?? [];
    if (!debouncedSearchQuery) return list;
    const query = debouncedSearchQuery.toLowerCase();
    return list.filter(
      (demon) =>
        demon.name.toLowerCase().includes(query) ||
        demon.command.toLowerCase().includes(query) ||
        (demon.description?.toLowerCase().includes(query) ?? false)
    );
  }, [demons, debouncedSearchQuery]);

  // Refs to avoid stale closures in customKeys callbacks
  const filteredDemonsRef = useRef(filteredDemons);
  const selectedIndexRef = useRef(0);
  useEffect(() => { filteredDemonsRef.current = filteredDemons; }, [filteredDemons]);

  // #1731: Use useListNavigation hook for keyboard navigation
  // Stable customKeys that use refs to access current state
  const customKeys = useMemo(() => ({
    e: () => {
      const demon = filteredDemonsRef.current[selectedIndexRef.current] as Demon | undefined;
      if (demon !== undefined) {
        enable(demon.name).catch((err: unknown) => {
          const message = err instanceof Error ? err.message : String(err);
          setActionError(`Enable failed: ${message}`);
        });
      }
    },
    D: () => {
      const demon = filteredDemonsRef.current[selectedIndexRef.current] as Demon | undefined;
      if (demon !== undefined) {
        disable(demon.name).catch((err: unknown) => {
          const message = err instanceof Error ? err.message : String(err);
          setActionError(`Disable failed: ${message}`);
        });
      }
    },
    x: () => {
      const demon = filteredDemonsRef.current[selectedIndexRef.current] as Demon | undefined;
      if (demon !== undefined) {
        run(demon.name).catch((err: unknown) => {
          const message = err instanceof Error ? err.message : String(err);
          setActionError(`Run failed: ${message}`);
        });
      }
    },
    r: () => { void refresh(); },
  }), [enable, disable, run, refresh]);

  const { selectedIndex, search } = useListNavigation<Demon>({
    items: filteredDemons,
    disabled: disableInput,
    enableSearch: true,
    onSearchChange: setSearchQuery,
    customKeys,
  });

  // Keep selectedIndexRef in sync
  useEffect(() => { selectedIndexRef.current = selectedIndex; }, [selectedIndex]);

  // Manage focus state for search mode (#1692)
  useEffect(() => {
    if (search.isActive) {
      setFocus('input');
    } else {
      setFocus('main');
    }
  }, [search.isActive, setFocus]);

  // Auto-clear action errors after a delay
  useEffect(() => {
    if (!actionError) return;
    const timer = setTimeout(() => { setActionError(null); }, ERROR_DISPLAY_DURATION);
    return () => { clearTimeout(timer); };
  }, [actionError]);

  // Search mode overlay
  if (search.isActive) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text bold>Search Demons</Text>
        <Box marginTop={1} borderStyle="single" borderColor="cyan" paddingX={1}>
          <Text color="cyan">{'> '}</Text>
          <Text>{search.query}</Text>
          <Text color="cyan">|</Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Enter to confirm, Esc to cancel</Text>
        </Box>
      </Box>
    );
  }

  // Build hints array dynamically
  const hints = [
    { key: 'j/k', label: 'nav' },
    { key: 'g/G', label: 'top/bottom' },
    { key: '/', label: 'search' },
    ...(search.query ? [{ key: 'c', label: 'clear' }] : []),
    { key: 'e', label: 'enable' },
    { key: 'D', label: 'disable' },
    { key: 'x', label: 'run' },
    { key: 'r', label: 'refresh' },
    { key: 'q/ESC', label: 'back' },
  ];

  // Build subtitle with enabled count and search
  const subtitleParts: string[] = [`${String(enabled)} enabled`];
  if (search.query) {
    subtitleParts.push(`[/] "${search.query}"`);
  }

  return (
    <ViewWrapper
      loading={loading && !demons}
      loadingMessage="Loading demons..."
      error={error}
      onRetry={() => { void refresh(); }}
      hints={hints}
    >
      {/* Header with count (#1446) */}
      <HeaderBar
        title="Demons"
        count={filteredDemons.length}
        loading={loading && (demons?.length ?? 0) > 0}
        subtitle={subtitleParts.join(' · ')}
        color="yellow"
      />
      {/* Action error feedback */}
      {actionError && (
        <Box marginBottom={1}>
          <Text color="red">{actionError}</Text>
        </Box>
      )}

      {/* Demon list */}
      {filteredDemons.length > 0 ? (
        <Box flexDirection="column">
          {/* Header row - total width: 3+14+13+9+7+10+10 = 66 (fits 80-col) */}
          <Box marginBottom={1}>
            <Box width={3}><Text> </Text></Box>
            <Box width={14}><Text bold dimColor>NAME</Text></Box>
            <Box width={13}><Text bold dimColor>SCHEDULE</Text></Box>
            <Box width={9}><Text bold dimColor>STATUS</Text></Box>
            <Box width={7}><Text bold dimColor>RUNS</Text></Box>
            <Box width={10}><Text bold dimColor>LAST</Text></Box>
            <Box width={10}><Text bold dimColor>NEXT</Text></Box>
          </Box>

          {/* Demon rows */}
          {filteredDemons.map((demon, index) => (
            <DemonRow
              key={demon.name}
              demon={demon}
              selected={index === selectedIndex}
            />
          ))}
        </Box>
      ) : (
        <Box flexDirection="column" paddingY={2}>
          <Text dimColor>{search.query ? 'No demons match search' : 'No demons configured'}</Text>
          {!search.query && <Text dimColor>Create one with: bc demon create {'<name>'} --schedule {'\'<cron>\''} --cmd {'\'<command>\''}</Text>}
        </Box>
      )}

      {/* Selected demon details */}
      {filteredDemons.length > 0 && filteredDemons[selectedIndex] && (
        <Box marginTop={1} borderStyle="single" borderColor="gray" padding={1} flexDirection="column">
          <Text bold>{filteredDemons[selectedIndex].name}</Text>
          <Box marginTop={1}>
            <Text dimColor>Command: </Text>
            <Text>{filteredDemons[selectedIndex].command}</Text>
          </Box>
          {filteredDemons[selectedIndex].description && (
            <Box>
              <Text dimColor>Description: </Text>
              <Text>{filteredDemons[selectedIndex].description}</Text>
            </Box>
          )}
          {filteredDemons[selectedIndex].owner && (
            <Box>
              <Text dimColor>Owner: </Text>
              <Text>{filteredDemons[selectedIndex].owner}</Text>
            </Box>
          )}
        </Box>
      )}
    </ViewWrapper>
  );
}

interface DemonRowProps {
  demon: Demon;
  selected: boolean;
}

function DemonRow({ demon, selected }: DemonRowProps): React.ReactElement {
  const statusText = demon.enabled ? 'enabled' : 'disabled';

  return (
    <Box>
      <Box width={3}>
        <Text color={selected ? 'cyan' : undefined}>
          {selected ? '▸ ' : '  '}
        </Text>
      </Box>
      <Box width={14}>
        <Text color={selected ? 'cyan' : undefined} bold={selected}>
          {demon.name.length > 12 ? demon.name.slice(0, 11) + '…' : demon.name}
        </Text>
      </Box>
      <Box width={13}>
        <Text dimColor>{formatSchedule(demon.schedule).slice(0, 11)}</Text>
      </Box>
      <Box width={9}>
        <StatusBadge state={statusText} showIcon={false} />
      </Box>
      <Box width={7}>
        <Text>{demon.run_count}</Text>
      </Box>
      <Box width={10}>
        <Text dimColor>{formatRelativeTime(demon.last_run)}</Text>
      </Box>
      <Box width={10}>
        <Text color={demon.enabled ? 'yellow' : 'gray'}>
          {formatRelativeTime(demon.next_run)}
        </Text>
      </Box>
    </Box>
  );
}

export default DemonsView;
