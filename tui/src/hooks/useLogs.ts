/**
 * useLogs hook - Fetch and poll event logs for live activity feed
 */

import { useState, useEffect, useCallback, useMemo } from 'react';
import type { LogEntry, BcResult } from '../types';
import { getLogs } from '../services/bc';

/** Log severity level derived from event type */
export type LogSeverity = 'info' | 'warn' | 'error';

export interface UseLogsOptions {
  /** Polling interval in ms (default: 3000) */
  pollInterval?: number;
  /** Whether to poll automatically (default: true) */
  autoPoll?: boolean;
  /** Number of log entries to fetch (default: 50) */
  tail?: number;
  /** Filter by agent name */
  agent?: string;
  /** Filter by event type */
  eventType?: string;
  /** Filter by severity */
  severity?: LogSeverity;
}

export interface UseLogsResult extends BcResult<LogEntry[]> {
  /** Manually refresh logs */
  refresh: () => Promise<void>;
  /** Filter logs by severity */
  filterBySeverity: (severity: LogSeverity | null) => void;
  /** Current severity filter */
  severityFilter: LogSeverity | null;
}

/**
 * Determine severity from event type
 */
function getSeverity(eventType: string): LogSeverity {
  const lowerType = eventType.toLowerCase();
  if (lowerType.includes('error') || lowerType.includes('fail')) {
    return 'error';
  }
  if (lowerType.includes('warn') || lowerType.includes('stuck')) {
    return 'warn';
  }
  return 'info';
}

/**
 * Hook to fetch and optionally poll event logs
 * @param options - Configuration options
 * @returns Log entries with loading state and filtering
 */
export function useLogs(options: UseLogsOptions = {}): UseLogsResult {
  const {
    pollInterval = 3000,
    autoPoll = true,
    tail = 50,
    agent,
    eventType,
  } = options;

  const [rawData, setRawData] = useState<LogEntry[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [severityFilter, setSeverityFilter] = useState<LogSeverity | null>(null);

  const fetchLogs = useCallback(async () => {
    try {
      const logs = await getLogs(tail, agent, eventType);
      setRawData(logs);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch logs');
    } finally {
      setLoading(false);
    }
  }, [tail, agent, eventType]);

  // Apply severity filter
  const data = useMemo(() => {
    if (!rawData) return null;
    if (!severityFilter) return rawData;
    return rawData.filter((entry) => getSeverity(entry.type) === severityFilter);
  }, [rawData, severityFilter]);

  // Initial fetch
  useEffect(() => {
    void fetchLogs();
  }, [fetchLogs]);

  // Polling
  useEffect(() => {
    if (!autoPoll) return;

    const interval = setInterval(() => {
      void fetchLogs();
    }, pollInterval);
    return () => {
      clearInterval(interval);
    };
  }, [autoPoll, pollInterval, fetchLogs]);

  const filterBySeverity = useCallback((severity: LogSeverity | null) => {
    setSeverityFilter(severity);
  }, []);

  return {
    data,
    error,
    loading,
    refresh: fetchLogs,
    filterBySeverity,
    severityFilter,
  };
}

/**
 * Get severity color for log entry
 */
export function getSeverityColor(type: string): string {
  const severity = getSeverity(type);
  switch (severity) {
    case 'error':
      return 'red';
    case 'warn':
      return 'yellow';
    default:
      return 'gray';
  }
}
