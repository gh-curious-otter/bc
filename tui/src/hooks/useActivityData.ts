/**
 * useActivityData - Hook for fetching and aggregating agent activity data
 * Issue #1047: Activity timeline and cost trend tracking
 *
 * Aggregates agent activity logs into time periods for display in timeline view.
 */

import { useState, useEffect } from 'react';

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
 * Aggregate activity events into time periods
 */
function aggregateActivity(events: ActivityEvent[], periodMinutes: number = 15): ActivityPeriod[] {
  if (events.length === 0) return [];

  const periods: Map<number, ActivityPeriod> = new Map();

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

    const period = periods.get(key)!;
    if (!period.agents.includes(event.agent)) {
      period.agents.push(event.agent);
    }
    period.totalCost += event.cost || 0;
  });

  return Array.from(periods.values()).sort((a, b) => a.startTime.getTime() - b.startTime.getTime());
}

/**
 * Hook to fetch and aggregate agent activity data
 */
export function useActivityData(options: UseActivityDataOptions = {}) {
  const { hours = 24, limit = 100 } = options;
  const [activities, setActivities] = useState<ActivityPeriod[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchActivityData = async () => {
    setLoading(true);
    setError(null);
    try {
      // In a real implementation, this would query bc logs API
      // For now, return empty data structure
      const events: ActivityEvent[] = [];
      const aggregated = aggregateActivity(events, 15);
      setActivities(aggregated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load activity data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchActivityData();
    const interval = setInterval(() => {
      void fetchActivityData();
    }, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, [hours, limit]);

  return { activities, loading, error, refresh: fetchActivityData };
}

export default useActivityData;
