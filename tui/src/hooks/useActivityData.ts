/**
 * useActivityData - Hook for fetching and aggregating agent activity data
 * Issue #1047: Activity timeline and cost trend tracking
 *
 * Aggregates agent activity logs into time periods for display in timeline view.
 */

import { useState, useEffect, useCallback } from 'react';
import { getLogs } from '../services/bc';
import type { LogEntry } from '../types';

export interface ActivityEvent {
  timestamp: Date;
  agent: string;
  action: string;
  duration?: number;
  cost?: number;
}

export interface ActivityPeriod {
  startTime: Date;
  endTime: Date;
  agents: string[];
  action: string;
  duration: number; // in minutes
  totalCost: number;
}

interface UseActivityDataOptions {
  hours?: number; // How many hours back to load (default: 24)
  limit?: number; // Max events to load (default: 100)
}

/**
 * Parse log entries to extract activity events
 */
export function parseLogToActivity(log: LogEntry): ActivityEvent | null {
  try {
    const timestamp = new Date(log.ts);
    // Filter to only state-change events (working, idle, done, etc.)
    const stateTypes = ['state.working', 'state.idle', 'state.done', 'state.stuck', 'state.error', 'agent.started', 'agent.stopped'];
    const isStateEvent = stateTypes.some((t) => log.type.toLowerCase().includes(t.toLowerCase()));

    if (!isStateEvent && !log.type.toLowerCase().includes('state')) {
      return null;
    }

    return {
      timestamp,
      agent: log.agent,
      action: log.type.split('.').pop() ?? log.type,
      cost: 0,
    };
  } catch {
    return null;
  }
}

/**
 * Aggregate activity events into time periods
 */
export function aggregateActivity(events: ActivityEvent[], periodMinutes = 15): ActivityPeriod[] {
  if (events.length === 0) return [];

  const periods = new Map<number, ActivityPeriod>();

  events.forEach((event) => {
    const periodStart = Math.floor(event.timestamp.getTime() / (periodMinutes * 60 * 1000)) * (periodMinutes * 60 * 1000);
    const key = periodStart;

    if (!periods.has(key)) {
      periods.set(key, {
        startTime: new Date(periodStart),
        endTime: new Date(periodStart + periodMinutes * 60 * 1000),
        agents: [],
        action: event.action,
        duration: periodMinutes,
        totalCost: 0,
      });
    }

    const period = periods.get(key);
    if (period && !period.agents.includes(event.agent)) {
      period.agents.push(event.agent);
    }
    if (period) {
      period.totalCost += event.cost ?? 0;
    }
  });

  // Sort by time descending (most recent first)
  return Array.from(periods.values()).sort((a, b) => b.startTime.getTime() - a.startTime.getTime());
}

/**
 * Hook to fetch and aggregate agent activity data
 */
export function useActivityData(options: UseActivityDataOptions = {}) {
  const { hours = 24, limit = 100 } = options;
  const [activities, setActivities] = useState<ActivityPeriod[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchActivityData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Fetch logs from bc CLI
      const logs = await getLogs(limit);

      // Filter logs by time range
      const cutoffTime = Date.now() - hours * 60 * 60 * 1000;
      const recentLogs = logs.filter((log) => {
        const logTime = new Date(log.ts).getTime();
        return logTime >= cutoffTime;
      });

      // Parse logs into activity events
      const events: ActivityEvent[] = recentLogs
        .map(parseLogToActivity)
        .filter((e): e is ActivityEvent => e !== null);

      // Aggregate into time periods
      const aggregated = aggregateActivity(events, 15);
      setActivities(aggregated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load activity data');
    } finally {
      setLoading(false);
    }
  }, [hours, limit]);

  useEffect(() => {
    void fetchActivityData();
    const interval = setInterval(() => {
      void fetchActivityData();
    }, 30000); // Refresh every 30 seconds
    return () => { clearInterval(interval); };
  }, [fetchActivityData]);

  return { activities, loading, error, refresh: fetchActivityData };
}

export default useActivityData;
