/**
 * LogsView - Event logs tab with filtering and search (#866)
 */

import React, { useState, useMemo, useCallback, useEffect } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { useLogs, getSeverityColor, getSeverityIcon } from '../hooks/useLogs';
import { useFocus } from '../navigation/FocusContext';
import { PulseText } from '../components/AnimatedText';
import type { LogSeverity } from '../hooks/useLogs';
import type { LogEntry } from '../types';

interface LogsViewProps {
  onBack?: () => void;
}

type TimeFilter = '1h' | '6h' | '24h' | 'all';

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
function abbreviateType(type: string): string {
  // Extract action from type (after last dot)
  const parts = type.split('.');
  const action = parts[parts.length - 1];

  // Common abbreviations
  const abbreviations: Record<string, string> = {
    'message': 'msg',
    'report': 'report',
    'working': 'work',
    'error': 'error',
    'warning': 'warn',
    'stuck': 'stuck',
    'done': 'done',
    'idle': 'idle',
    'starting': 'start',
    'stopping': 'stop',
  };

  return abbreviations[action] ?? action;
}

export const LogsView: React.FC<LogsViewProps> = ({ onBack }) => {
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;

  const { data: logs, loading, error, refresh, filterBySeverity, severityFilter } = useLogs({
    tail: 100,
    autoPoll: true,
    pollInterval: 5000,
  });

  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showDetail, setShowDetail] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchMode, setSearchMode] = useState(false);
  const [agentFilter, setAgentFilter] = useState<string | null>(null);
  const [timeFilter, setTimeFilter] = useState<TimeFilter>('all');
  const { setFocus } = useFocus();

  // Manage focus state for nested view navigation
  useEffect(() => {
    if (showDetail) {
      setFocus('view');
    } else {
      setFocus('main');
    }
  }, [showDetail, setFocus]);

  // Get unique agents for filter
  const agents = useMemo(() => {
    if (!logs) return [];
    const agentSet = new Set(logs.map((log) => log.agent));
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

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (log) =>
          log.message.toLowerCase().includes(query) ||
          log.agent.toLowerCase().includes(query) ||
          log.type.toLowerCase().includes(query)
      );
    }

    return result;
  }, [logs, timeFilter, agentFilter, searchQuery]);

  const selectedLog = filteredLogs[selectedIndex] as LogEntry | undefined;

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
    setTimeFilter(times[nextIdx]);
    setSelectedIndex(0);
  }, [timeFilter]);

  // Cycle through agent filters
  const cycleAgentFilter = useCallback(() => {
    if (agents.length === 0) return;
    if (agentFilter === null) {
      setAgentFilter(agents[0]);
    } else {
      const currentIdx = agents.indexOf(agentFilter);
      if (currentIdx === agents.length - 1) {
        setAgentFilter(null);
      } else {
        setAgentFilter(agents[currentIdx + 1]);
      }
    }
    setSelectedIndex(0);
  }, [agentFilter, agents]);

  // Keyboard navigation
  useInput((input, key) => {
    if (searchMode) {
      // Search mode input
      if (key.return || key.escape) {
        setSearchMode(false);
      } else if (key.backspace || key.delete) {
        setSearchQuery(searchQuery.slice(0, -1));
      } else if (input && !key.ctrl && !key.meta) {
        setSearchQuery(searchQuery + input);
      }
      return;
    }

    if (showDetail) {
      // Detail view - any key returns to list
      if (key.escape || input === 'q' || key.return) {
        setShowDetail(false);
      }
      return;
    }

    // List view navigation
    if (key.upArrow || input === 'k') {
      setSelectedIndex((i) => Math.max(0, i - 1));
    } else if (key.downArrow || input === 'j') {
      setSelectedIndex((i) => Math.min(filteredLogs.length - 1, i + 1));
    } else if (input === 'g') {
      setSelectedIndex(0);
    } else if (input === 'G') {
      setSelectedIndex(Math.max(0, filteredLogs.length - 1));
    } else if (key.return) {
      if (selectedLog) {
        setShowDetail(true);
      }
    } else if (input === '/') {
      setSearchMode(true);
    } else if (input === 's') {
      cycleSeverity();
    } else if (input === 'a') {
      cycleAgentFilter();
    } else if (input === 't') {
      cycleTimeFilter();
    } else if (input === 'c') {
      // Clear all filters
      setSearchQuery('');
      setAgentFilter(null);
      setTimeFilter('all');
      filterBySeverity(null);
      setSelectedIndex(0);
    } else if (input === 'r') {
      void refresh();
    } else if (input === 'q' || key.escape) {
      onBack?.();
    }
  });

  // Show detail view
  if (showDetail && selectedLog) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text bold color="cyan">Log Details</Text>
        <Box marginTop={1} flexDirection="column" borderStyle="single" borderColor="gray" padding={1}>
          <Box>
            <Text bold>Timestamp: </Text>
            <Text>{selectedLog.ts}</Text>
          </Box>
          <Box>
            <Text bold>Agent: </Text>
            <Text color="cyan">{selectedLog.agent}</Text>
          </Box>
          <Box>
            <Text bold>Type: </Text>
            <Text color={getSeverityColor(selectedLog.type)}>{getSeverityIcon(selectedLog.type)} {selectedLog.type}</Text>
          </Box>
          <Box marginTop={1} flexDirection="column">
            <Text bold>Message:</Text>
            <Box paddingLeft={2} marginTop={1}>
              <Text wrap="wrap">{selectedLog.message}</Text>
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
        <Box marginTop={1} borderStyle="single" borderColor="cyan" paddingX={1}>
          <Text color="cyan">{'> '}</Text>
          <Text>{searchQuery}</Text>
          <Text color="cyan">|</Text>
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
        <PulseText color="cyan">Loading logs...</PulseText>
      </Box>
    );
  }

  if (error) {
    return (
      <Box padding={1}>
        <Text color="red">Error: {error}</Text>
      </Box>
    );
  }

  // Calculate column widths based on terminal width
  // #973 fix: Increased from 10 to 12 to fit date format (MM/DD HH:MM)
  const timeWidth = 12;
  const agentWidth = Math.min(12, Math.floor((terminalWidth - 40) * 0.2));
  const typeWidth = 10;
  const messageWidth = terminalWidth - timeWidth - agentWidth - typeWidth - 10;

  // Visible rows - dynamic based on terminal height (#80x24 support)
  // Account for: app overhead (6) + header (1) + filters (1) + table border (2) + footer (1)
  const terminalHeight = stdout.rows;
  const viewOverhead = 11;
  const visibleRows = Math.max(5, Math.min(15, terminalHeight - viewOverhead));
  const startIdx = Math.max(0, selectedIndex - Math.floor(visibleRows / 2));
  const visibleLogs = filteredLogs.slice(startIdx, startIdx + visibleRows);

  return (
    <Box flexDirection="column">
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="magenta">Logs</Text>
        <Text dimColor> ({filteredLogs.length} entries)</Text>
        {loading && <PulseText color="gray"> (refreshing...)</PulseText>}
      </Box>

      {/* Filters */}
      <Box marginBottom={1}>
        <Text dimColor>Filters: </Text>
        <Text color={severityFilter ? 'cyan' : 'gray'}>
          [s] {severityFilter ?? 'All'}
        </Text>
        <Text> </Text>
        <Text color={agentFilter ? 'cyan' : 'gray'}>
          [a] {agentFilter ?? 'All agents'}
        </Text>
        <Text> </Text>
        <Text color={timeFilter !== 'all' ? 'cyan' : 'gray'}>
          [t] {timeFilter === 'all' ? 'All time' : `Last ${timeFilter}`}
        </Text>
        {searchQuery && (
          <>
            <Text> </Text>
            <Text color="cyan">[/] &quot;{searchQuery}&quot;</Text>
          </>
        )}
      </Box>

      {/* Log table */}
      <Box flexDirection="column" borderStyle="single" borderColor="gray">
        {/* Table header */}
        <Box>
          <Text bold color="gray">
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
          const severityColor = getSeverityColor(log.type);
          const severityIcon = getSeverityIcon(log.type);

          return (
            <Box key={`${log.ts}-${String(idx)}`}>
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : undefined}
              >
                {formatTime(log.ts).padEnd(timeWidth)}
              </Text>
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : 'cyan'}
              >
                {log.agent.slice(0, agentWidth - 1).padEnd(agentWidth)}
              </Text>
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : severityColor}
              >
                {severityIcon} {abbreviateType(log.type).slice(0, typeWidth - 3).padEnd(typeWidth - 2)}
              </Text>
              <Text
                backgroundColor={isSelected ? 'blue' : undefined}
                color={isSelected ? 'white' : undefined}
                wrap="truncate"
              >
                {log.message.slice(0, messageWidth)}
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

      {/* Footer */}
      <Box marginTop={1}>
        <Text dimColor>
          j/k: nav | g/G: top/bottom | Enter: details | /: search | s: severity | a: agent | t: time | c: clear | r: refresh | q/ESC: back
        </Text>
      </Box>
    </Box>
  );
};

export default LogsView;
