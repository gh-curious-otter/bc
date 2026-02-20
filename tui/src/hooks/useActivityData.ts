/**
 * useActivityData - Hook for fetching and aggregating agent activity data
 * Issue #1047: Activity timeline and cost trend tracking
 *
 * Aggregates agent activity logs into time periods for display in timeline view.
 */

import { useState, useEffect, useCallback } from 'react';
import type { LogEntry } from '../types';
import { getLogs } from '../services/bc';
import { usePerformanceConfig } from '../config';

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
  eventCount: number;
}

interface UseActivityDataOptions {
  hours?: number; // How many hours back to load (default: 24)
  limit?: number; // Max events to load (default: 100)
}

/**
 * Convert log entries to activity events
 */
function logsToEvents(logs: LogEntry[]): ActivityEvent[] {
  return logs.map((log) => ({
    timestamp: new Date(log.ts),
    agent: log.agent || 'system',
    action: log.type,
    duration: undefined,
    cost: undefined,
  }));
}

/**
 * Filter events by time range (hours back from now)
 */
function filterByHours(events: ActivityEvent[], hours: number): ActivityEvent[] {
  const cutoff = Date.now() - hours * 60 * 60 * 1000;
  return events.filter((e) => e.timestamp.getTime() >= cutoff);
}

/**
 * Aggregate activity events into time periods
 */
function aggregateActivity(events: ActivityEvent[], periodMinutes = 15): ActivityPeriod[] {
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
        eventCount: 0,
      });
    }

    const period = periods.get(key);
    if (period) {
      if (!period.agents.includes(event.agent)) {
        period.agents.push(event.agent);
      }
      period.totalCost += event.cost ?? 0;
      period.eventCount += 1;
    }
  });

  // Sort by time, most recent first
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

  const perfConfig = usePerformanceConfig();
  const pollInterval = perfConfig.poll_interval_logs;

  const fetchActivityData = useCallback(async () => {
    try {
      // Fetch logs from bc CLI
      const logs = await getLogs(limit);
      const events = logsToEvents(logs);
      const filtered = filterByHours(events, hours);
      const aggregated = aggregateActivity(filtered, 15);
      setActivities(aggregated);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load activity data');
    } finally {
      setLoading(false);
    }
  }, [hours, limit]);

  // Initial fetch
  useEffect(() => {
    void fetchActivityData();
  }, [fetchActivityData]);

  // Polling
  useEffect(() => {
    const interval = setInterval(() => {
      void fetchActivityData();
    }, pollInterval);
    return () => { clearInterval(interval); };
  }, [fetchActivityData, pollInterval]);

  return { activities, loading, error, refresh: fetchActivityData };
}

export default useActivityData;
