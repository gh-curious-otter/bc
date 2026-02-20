/**
 * ActivityFeed - Real-time log stream widget
 * Issue #796 - Live activity feed with severity filtering
 */

import React, { memo, useMemo } from 'react';
import { Box, Text, useStdout } from 'ink';
import { Panel } from './Panel';
import { useLogs, getSeverityColor } from '../hooks';
import type { LogSeverity } from '../hooks';
import type { LogEntry } from '../types';

export interface ActivityFeedProps {
  /** Maximum number of entries to display */
  maxEntries?: number;
  /** Filter by severity level */
  severityFilter?: LogSeverity | null;
  /** Panel width */
  width?: number | string;
  /** Panel height */
  height?: number | string;
  /** Show compact view (no timestamps) */
  compact?: boolean;
  /** Show filter hints in title */
  showFilterHints?: boolean;
}

/**
 * Format timestamp for display
 */
function formatTime(ts: string): string {
  try {
    const date = new Date(ts);
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    });
  } catch {
    return '--:--:--';
  }
}

/**
 * Format event type for display (shorter labels)
 */
function formatEventType(type: string): string {
  // Convert dots to simpler format
  const parts = type.split('.');
  if (parts.length > 1) {
    return parts[parts.length - 1];
  }
  return type;
}

/**
 * Truncate message to fit in compact display
 */
function truncateMessage(msg: string, maxLen: number): string {
  if (msg.length <= maxLen) return msg;
  return msg.slice(0, maxLen - 3) + '...';
}

/**
 * ActivityFeed component - Real-time log stream
 * #1196: Responsive message truncation based on terminal width
 */
export function ActivityFeed({
  maxEntries = 8,
  severityFilter = null,
  width,
  height,
  compact = false,
  showFilterHints = true,
}: ActivityFeedProps): React.ReactElement {
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;

  const { data: logs, loading, severityFilter: currentFilter } = useLogs({
    tail: 50,
    pollInterval: 3000,
  });

  // #1196: Calculate responsive message length based on terminal width
  // Layout: [timestamp 9] [agent 11] [event 13] [message]
  // Compact: [agent 11] [event 13] [message]
  const maxMsgLen = useMemo(() => {
    const timestampWidth = compact ? 0 : 9;
    const agentWidth = 11; // padEnd(10) + space
    const eventWidth = 13; // padEnd(12) + space
    const panelOverhead = 6; // borders, padding, margin
    const available = terminalWidth - timestampWidth - agentWidth - eventWidth - panelOverhead;
    // Clamp between 20 and 100
    return Math.max(20, Math.min(100, available));
  }, [terminalWidth, compact]);

  // Apply local severity filter or use hook filter
  const activeFilter = severityFilter ?? currentFilter;

  // Filter and limit entries
  const displayLogs = useMemo(() => {
    if (!logs) return [];
    let filtered = logs;
    if (activeFilter) {
      filtered = logs.filter((entry) => {
        const type = entry.type.toLowerCase();
        switch (activeFilter) {
          case 'error':
            return type.includes('error') || type.includes('fail');
          case 'warn':
            return type.includes('warn') || type.includes('stuck');
          default:
            return !type.includes('error') && !type.includes('fail') && !type.includes('warn') && !type.includes('stuck');
        }
      });
    }
    // Show most recent first, then limit
    return filtered.slice(-maxEntries).reverse();
  }, [logs, activeFilter, maxEntries]);

  // Build title with optional filter hints
  const title = useMemo(() => {
    let t = 'Activity';
    if (activeFilter) {
      t += ` [${activeFilter}]`;
    }
    if (showFilterHints) {
      t += ' (i/w/e/*)';
    }
    return t;
  }, [activeFilter, showFilterHints]);

  if (loading && !logs) {
    return (
      <Panel title="Activity" width={width} height={height}>
        <Text dimColor>Loading activity...</Text>
      </Panel>
    );
  }

  return (
    <Panel title={title} width={width} height={height}>
      {displayLogs.length === 0 ? (
        <Text dimColor>No activity</Text>
      ) : (
        <Box flexDirection="column">
          {displayLogs.map((entry, idx) => (
            <ActivityEntry key={`${entry.ts}-${String(idx)}`} entry={entry} compact={compact} maxMsgLen={maxMsgLen} />
          ))}
        </Box>
      )}
    </Panel>
  );
}

/**
 * Individual activity entry - memoized for performance
 * #1196: Now accepts maxMsgLen prop for responsive truncation
 */
interface ActivityEntryProps {
  entry: LogEntry;
  compact?: boolean;
  maxMsgLen: number;
}

const ActivityEntry = memo(function ActivityEntry({
  entry,
  compact = false,
  maxMsgLen,
}: ActivityEntryProps): React.ReactElement {
  const severityColor = getSeverityColor(entry.type);
  const eventLabel = formatEventType(entry.type);

  return (
    <Box>
      {!compact && (
        <Text dimColor>{formatTime(entry.ts)} </Text>
      )}
      <Text color="cyan">{entry.agent.padEnd(10)} </Text>
      <Text color={severityColor}>{eventLabel.padEnd(12)} </Text>
      <Text>{truncateMessage(entry.message, maxMsgLen)}</Text>
    </Box>
  );
});

export default ActivityFeed;
