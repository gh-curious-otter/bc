/**
 * LogsView - Event logs tab with filtering and search (#866)
 *
 * #1720: Migrated to useListNavigation for consolidated keyboard patterns
 */

import React, { useMemo, useCallback, useEffect, useReducer, useState } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { useTheme } from '../theme';
import {
  useLogs,
  getSeverityColor,
  getSeverityIcon,
  useDebounce,
  useListNavigation,
} from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { ErrorDisplay } from '../components/ErrorDisplay';
import { Footer } from '../components/Footer';
import { DATA_LIMITS } from '../constants';
import { POLL_INTERVALS } from '../constants/timings';
import type { LogSeverity } from '../hooks/useLogs';
import type { LogEntry } from '../types';

type LogsViewProps = Record<string, never>;

type TimeFilter = '1h' | '6h' | '24h' | 'all';

// #1601: Consolidated UI state with useReducer
// #1720: Navigation moved to useListNavigation, reducer handles view-specific state
interface UIState {
  showDetail: boolean;
  agentFilter: string | null;
  timeFilter: TimeFilter;
}

type UIAction =
  | { type: 'SHOW_DETAIL' }
  | { type: 'HIDE_DETAIL' }
  | { type: 'SET_AGENT_FILTER'; agent: string | null }
  | { type: 'SET_TIME_FILTER'; time: TimeFilter }
  | { type: 'CLEAR_FILTERS' };

const initialUIState: UIState = {
  showDetail: false,
  agentFilter: null,
  timeFilter: 'all',
};

function uiReducer(state: UIState, action: UIAction): UIState {
  switch (action.type) {
    case 'SHOW_DETAIL':
      return { ...state, showDetail: true };
    case 'HIDE_DETAIL':
      return { ...state, showDetail: false };
    case 'SET_AGENT_FILTER':
      return { ...state, agentFilter: action.agent };
    case 'SET_TIME_FILTER':
      return { ...state, timeFilter: action.time };
    case 'CLEAR_FILTERS':
      return { ...state, agentFilter: null, timeFilter: 'all' };
    default:
      return state;
  }
}

/**
 * Format timestamp for display
 * #973 fix: Show date for logs from previous days
 */
function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    const now = new Date();
    const isToday = date.toDateString() === now.toDateString();

    if (isToday) {
      // Today: show time only (HH:MM:SS)
      return date.toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
      });
    } else {
      // Previous days: show MM/DD HH:MM
      const month = String(date.getMonth() + 1).padStart(2, '0');
      const day = String(date.getDate()).padStart(2, '0');
      const hours = String(date.getHours()).padStart(2, '0');
      const mins = String(date.getMinutes()).padStart(2, '0');
      return `${month}/${day} ${hours}:${mins}`;
    }
  } catch {
    return timestamp.slice(0, 8);
  }
}

/**
 * Filter logs by time range
 */
function filterByTime(logs: LogEntry[], timeFilter: TimeFilter): LogEntry[] {
  if (timeFilter === 'all') return logs;

  const now = Date.now();
  const hours = timeFilter === '1h' ? 1 : timeFilter === '6h' ? 6 : 24;
  const cutoff = now - hours * 60 * 60 * 1000;

  return logs.filter((log) => {
    try {
      return new Date(log.ts).getTime() >= cutoff;
    } catch {
      return true;
    }
  });
}

/**
 * Abbreviate log type for compact display (#1364)
 * agent.report → report, channel.message → msg, etc.
 */
function abbreviateType(type: string | undefined): string {
  if (!type) return '';
  // Extract action from type (after last dot)
  const parts = type.split('.');
  const action = parts[parts.length - 1];

  // Common abbreviations
  const abbreviations: Record<string, string> = {
    message: 'msg',
    report: 'report',
    working: 'work',
    error: 'error',
    warning: 'warn',
    stuck: 'stuck',
    done: 'done',
    idle: 'idle',
    starting: 'start',
    stopping: 'stop',
  };

  return abbreviations[action] ?? action;
}

export const LogsView: React.FC<LogsViewProps> = () => {
  const { theme } = useTheme();
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;

  const {
    data: logs,
    loading,
    error,
    refresh,
    filterBySeverity,
    severityFilter,
  } = useLogs({
    tail: DATA_LIMITS.LOG_TAIL,
    autoPoll: true,
    pollInterval: POLL_INTERVALS.LOGS,
  });

  // #1601: UI state consolidated with useReducer
  // #1720: Navigation state moved to useListNavigation, search kept separate
  const [ui, dispatch] = useReducer(uiReducer, initialUIState);
  const { showDetail, agentFilter, timeFilter } = ui;
  const { setFocus } = useFocus();

  // Search state - kept separate for debounce integration
  const [searchQuery, setSearchQuery] = useState('');
  const [searchMode, setSearchMode] = useState(false);

  // Debounce search query for filtering (issue #1602)
  const debouncedSearchQuery = useDebounce(searchQuery, 300);

  // Get unique agents for filter
  const agents = useMemo(() => {
    if (!logs) return [];
    const agentSet = new Set(logs.map((log) => log.agent ?? '').filter(Boolean));
    return Array.from(agentSet).sort();
  }, [logs]);

  // Apply all filters
  const filteredLogs = useMemo(() => {
    if (!logs) return [];

    let result = logs;

    // Time filter
    result = filterByTime(result, timeFilter);

    // Agent filter
    if (agentFilter) {
      result = result.filter((log) => log.agent === agentFilter);
    }

    // Search filter (using debounced query for performance - issue #1602)
    if (debouncedSearchQuery) {
      const query = debouncedSearchQuery.toLowerCase();
      result = result.filter(
        (log) =>
          (log.message ?? '').toLowerCase().includes(query) ||
          (log.agent ?? '').toLowerCase().includes(query) ||
          (log.type ?? '').toLowerCase().includes(query)
      );
    }

    return result;
  }, [logs, timeFilter, agentFilter, debouncedSearchQuery]);

  // Cycle through severity filters
  const cycleSeverity = useCallback(() => {
    const severities: (LogSeverity | null)[] = [null, 'info', 'warn', 'error'];
    const currentIdx = severities.indexOf(severityFilter);
    const nextIdx = (currentIdx + 1) % severities.length;
    filterBySeverity(severities[nextIdx]);
  }, [severityFilter, filterBySeverity]);

  // Cycle through time filters
  const cycleTimeFilter = useCallback(() => {
    const times: TimeFilter[] = ['all', '1h', '6h', '24h'];
    const currentIdx = times.indexOf(timeFilter);
    const nextIdx = (currentIdx + 1) % times.length;
    dispatch({ type: 'SET_TIME_FILTER', time: times[nextIdx] });
  }, [timeFilter]);

  // Cycle through agent filters
  const cycleAgentFilter = useCallback(() => {
    if (agents.length === 0) return;
    if (agentFilter === null) {
      dispatch({ type: 'SET_AGENT_FILTER', agent: agents[0] });
    } else {
      const currentIdx = agents.indexOf(agentFilter);
      if (currentIdx === agents.length - 1) {
        dispatch({ type: 'SET_AGENT_FILTER', agent: null });
      } else {
        dispatch({ type: 'SET_AGENT_FILTER', agent: agents[currentIdx + 1] });
      }
    }
  }, [agentFilter, agents]);

  // Clear all filters
  const clearAllFilters = useCallback(() => {
    dispatch({ type: 'CLEAR_FILTERS' });
    filterBySeverity(null);
    setSearchQuery('');
  }, [filterBySeverity]);

  // Custom key handlers for view-specific actions (#1720)
  const customKeys = useMemo(
    () => ({
      s: cycleSeverity,
      a: cycleAgentFilter,
      t: cycleTimeFilter,
      c: clearAllFilters,
      r: () => {
        void refresh();
      },
      '/': () => {
        setSearchMode(true);
      },
    }),
    [cycleSeverity, cycleAgentFilter, cycleTimeFilter, clearAllFilters, refresh]
  );

  // #1720: useListNavigation for consolidated keyboard patterns
  const {
    selectedIndex,
    selectedItem: selectedLog,
    setSelectedIndex,
  } = useListNavigation({
    items: filteredLogs,
    onSelect: () => {
      dispatch({ type: 'SHOW_DETAIL' });
    },
    disabled: showDetail || searchMode,
    customKeys,
  });

  // Manage focus state for nested view navigation
  // When in search mode, set focus='input' to allow typing special chars (#1692)
  useEffect(() => {
    if (showDetail) {
      setFocus('view');
    } else if (searchMode) {
      setFocus('input');
    } else {
      setFocus('main');
    }
  }, [showDetail, searchMode, setFocus]);

  // Reset selection when filters change
  useEffect(() => {
    setSelectedIndex(0);
  }, [timeFilter, agentFilter, debouncedSearchQuery, setSelectedIndex]);

  // Keyboard handling for search mode and detail view
  useInput(
    (input, key) => {
      if (searchMode) {
        // Search mode input
        if (key.return || key.escape) {
          setSearchMode(false);
        } else if (key.backspace || key.delete) {
          setSearchQuery((prev) => prev.slice(0, -1));
        } else if (input && !key.ctrl && !key.meta) {
          setSearchQuery((prev) => prev + input);
        }
        return;
      }

      if (showDetail) {
        // Detail view - any key returns to list
        if (key.escape || input === 'q' || key.return) {
          dispatch({ type: 'HIDE_DETAIL' });
        }
      }
    },
    { isActive: searchMode || showDetail }
  );

  // Show detail view
  if (showDetail && selectedLog) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text bold color={theme.colors.primary}>
          Log Details
        </Text>
        <Box
          marginTop={1}
          flexDirection="column"
          borderStyle="single"
          borderColor={theme.colors.textMuted}
          padding={1}
        >
          <Box>
            <Text bold>Timestamp: </Text>
            <Text>{selectedLog.ts}</Text>
          </Box>
          <Box>
            <Text bold>Agent: </Text>
            <Text color={theme.colors.primary}>{selectedLog.agent ?? 'unknown'}</Text>
          </Box>
          <Box>
            <Text bold>Type: </Text>
            <Text color={getSeverityColor(selectedLog.type ?? '')}>
              {getSeverityIcon(selectedLog.type ?? '')} {selectedLog.type ?? 'unknown'}
            </Text>
          </Box>
          <Box marginTop={1} flexDirection="column">
            <Text bold>Message:</Text>
            <Box paddingLeft={2} marginTop={1}>
              <Text wrap="wrap">{selectedLog.message ?? '(no message)'}</Text>
            </Box>
          </Box>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Press any key to return</Text>
        </Box>
      </Box>
    );
  }

  // Search mode overlay
  if (searchMode) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text bold>Search Logs</Text>
        <Box marginTop={1} borderStyle="single" borderColor={theme.colors.primary} paddingX={1}>
          <Text color={theme.colors.primary}>{'> '}</Text>
          <Text>{searchQuery}</Text>
          <Text color={theme.colors.primary}>|</Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Enter to confirm, Esc to cancel</Text>
        </Box>
      </Box>
    );
  }

  if (loading && !logs) {
    return (
      <Box padding={1}>
        <Text color={theme.colors.primary}>Loading logs...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <ErrorDisplay
        error={error}
        onRetry={() => {
          void refresh();
        }}
      />
    );
  }

  // Calculate column widths based on terminal width
  // #973 fix: Increased from 10 to 12 to fit date format (MM/DD HH:MM)
  const timeWidth = 12;
  const agentWidth = Math.min(12, Math.floor((terminalWidth - 40) * 0.2));
  const typeWidth = 10;
  // -12 accounts for: selection indicator (2) + spacing (10)
  const messageWidth = terminalWidth - timeWidth - agentWidth - typeWidth - 12;

  // Visible rows - dynamic based on terminal height (#80x24 support)
  // Account for: app overhead (6) + header (1) + filters (1) + table border (2) + footer (1)
  const terminalHeight = stdout.rows;
  const viewOverhead = 11;
  const visibleRows = Math.max(5, Math.min(15, terminalHeight - viewOverhead));
  const startIdx = Math.max(0, selectedIndex - Math.floor(visibleRows / 2));
  const visibleLogs = filteredLogs.slice(startIdx, startIdx + visibleRows);

  return (
    <Box flexDirection="column" overflow="hidden">
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color={theme.colors.accent}>
          Logs
        </Text>
        <Text dimColor> ({filteredLogs.length} entries)</Text>
        {loading && <Text color={theme.colors.textMuted}> (refreshing...)</Text>}
      </Box>

      {/* Filters */}
      <Box marginBottom={1}>
        <Text dimColor>Filters: </Text>
        <Text color={severityFilter ? theme.colors.primary : theme.colors.textMuted}>
          [s] {severityFilter ?? 'All'}
        </Text>
        <Text> </Text>
        <Text color={agentFilter ? theme.colors.primary : theme.colors.textMuted}>
          [a] {agentFilter ?? 'All agents'}
        </Text>
        <Text> </Text>
        <Text color={timeFilter !== 'all' ? theme.colors.primary : theme.colors.textMuted}>
          [t] {timeFilter === 'all' ? 'All time' : `Last ${timeFilter}`}
        </Text>
        {searchQuery && (
          <>
            <Text> </Text>
            <Text color={theme.colors.primary}>[/] &quot;{searchQuery}&quot;</Text>
          </>
        )}
      </Box>

      {/* Log table */}
      <Box flexDirection="column" borderStyle="single" borderColor={theme.colors.textMuted}>
        {/* Table header */}
        <Box>
          <Text>{'  '}</Text>
          <Text bold color={theme.colors.textMuted}>
            {'TIME'.padEnd(timeWidth)}
            {'AGENT'.padEnd(agentWidth)}
            {'TYPE'.padEnd(typeWidth)}
            {'MESSAGE'}
          </Text>
        </Box>

        {/* Table rows */}
        {visibleLogs.map((log, idx) => {
          const actualIdx = startIdx + idx;
          const isSelected = actualIdx === selectedIndex;
          const logType = log.type ?? '';
          const severityColor = getSeverityColor(logType);
          const severityIcon = getSeverityIcon(logType);

          return (
            <Box key={`${log.ts}-${String(idx)}`}>
              <Text color={isSelected ? theme.colors.primary : undefined}>
                {isSelected ? '▸ ' : '  '}
              </Text>
              <Text color={isSelected ? theme.colors.primary : undefined}>
                {formatTime(log.ts).padEnd(timeWidth)}
              </Text>
              <Text color={isSelected ? theme.colors.primary : theme.colors.primary}>
                {(log.agent ?? '').slice(0, agentWidth - 1).padEnd(agentWidth)}
              </Text>
              <Text color={isSelected ? theme.colors.primary : severityColor}>
                {severityIcon}{' '}
                {abbreviateType(logType)
                  .slice(0, typeWidth - 3)
                  .padEnd(typeWidth - 2)}
              </Text>
              <Text color={isSelected ? theme.colors.primary : undefined} wrap="truncate">
                {(log.message ?? '').slice(0, messageWidth)}
              </Text>
            </Box>
          );
        })}

        {filteredLogs.length === 0 && (
          <Box padding={1}>
            <Text dimColor>No logs match filters</Text>
          </Box>
        )}
      </Box>

      {/* Footer - view-specific hints only, global hints (Tab/q/?) in app footer */}
      <Footer
        hints={[
          { key: 'j/k', label: 'nav' },
          { key: 'g/G', label: 'top/bottom' },
          { key: 'Enter', label: 'details' },
          { key: '/', label: 'search' },
          { key: 's', label: 'severity' },
          { key: 'a', label: 'agent' },
          { key: 't', label: 'time' },
          { key: 'c', label: 'clear' },
          { key: 'r', label: 'refresh' },
        ]}
      />
    </Box>
  );
};

export default LogsView;
