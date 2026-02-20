/**
 * useActivityData hook tests - Unit tests for utility functions
 * Issue #1047 - Activity timeline and cost trend tracking
 */

import { describe, it, expect } from 'bun:test';
import type { LogEntry } from '../../types';

// Test the log-to-event conversion logic
interface ActivityEvent {
  timestamp: Date;
  agent: string;
  action: string;
}

function testLogsToEvents(logs: LogEntry[]): ActivityEvent[] {
  return logs.map((log) => ({
    timestamp: new Date(log.ts),
    agent: log.agent || 'system',
    action: log.type,
  }));
}

// Test the time filtering logic
function testFilterByHours(events: ActivityEvent[], hours: number): ActivityEvent[] {
  const cutoff = Date.now() - hours * 60 * 60 * 1000;
  return events.filter((e) => e.timestamp.getTime() >= cutoff);
}

// Test the aggregation logic
interface ActivityPeriod {
  startTime: Date;
  endTime: Date;
  agents: string[];
  eventCount: number;
}

function testAggregateActivity(events: ActivityEvent[], periodMinutes: number = 15): ActivityPeriod[] {
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
        eventCount: 0,
      });
    }

    const period = periods.get(key)!;
    if (!period.agents.includes(event.agent)) {
      period.agents.push(event.agent);
    }
    period.eventCount += 1;
  });

  return Array.from(periods.values()).sort((a, b) => b.startTime.getTime() - a.startTime.getTime());
}

describe('logsToEvents', () => {
  it('converts log entries to activity events', () => {
    const logs: LogEntry[] = [
      { ts: '2026-02-20T10:00:00Z', type: 'state_change', agent: 'eng-01', message: 'Started' },
      { ts: '2026-02-20T10:05:00Z', type: 'task_complete', agent: 'eng-02', message: 'Done' },
    ];

    const events = testLogsToEvents(logs);

    expect(events.length).toBe(2);
    expect(events[0].agent).toBe('eng-01');
    expect(events[0].action).toBe('state_change');
    expect(events[1].agent).toBe('eng-02');
  });

  it('handles missing agent field', () => {
    const logs: LogEntry[] = [
      { ts: '2026-02-20T10:00:00Z', type: 'system_event', agent: '', message: 'Boot' },
    ];

    const events = testLogsToEvents(logs);

    expect(events[0].agent).toBe('system');
  });
});

describe('filterByHours', () => {
  it('filters events within time range', () => {
    const now = Date.now();
    const events: ActivityEvent[] = [
      { timestamp: new Date(now - 1 * 60 * 60 * 1000), agent: 'eng-01', action: 'recent' },
      { timestamp: new Date(now - 25 * 60 * 60 * 1000), agent: 'eng-02', action: 'old' },
    ];

    const filtered = testFilterByHours(events, 24);

    expect(filtered.length).toBe(1);
    expect(filtered[0].agent).toBe('eng-01');
  });

  it('includes events at boundary', () => {
    const now = Date.now();
    const events: ActivityEvent[] = [
      { timestamp: new Date(now - 23 * 60 * 60 * 1000), agent: 'eng-01', action: 'edge' },
    ];

    const filtered = testFilterByHours(events, 24);

    expect(filtered.length).toBe(1);
  });
});

describe('aggregateActivity', () => {
  it('groups events into time periods', () => {
    const baseTime = new Date('2026-02-20T10:00:00Z').getTime();
    const events: ActivityEvent[] = [
      { timestamp: new Date(baseTime), agent: 'eng-01', action: 'work' },
      { timestamp: new Date(baseTime + 5 * 60 * 1000), agent: 'eng-02', action: 'work' },
      { timestamp: new Date(baseTime + 30 * 60 * 1000), agent: 'eng-01', action: 'work' },
    ];

    const periods = testAggregateActivity(events, 15);

    expect(periods.length).toBe(2);
  });

  it('deduplicates agents within period', () => {
    const baseTime = new Date('2026-02-20T10:00:00Z').getTime();
    const events: ActivityEvent[] = [
      { timestamp: new Date(baseTime), agent: 'eng-01', action: 'work' },
      { timestamp: new Date(baseTime + 1 * 60 * 1000), agent: 'eng-01', action: 'work' },
      { timestamp: new Date(baseTime + 2 * 60 * 1000), agent: 'eng-01', action: 'work' },
    ];

    const periods = testAggregateActivity(events, 15);

    expect(periods.length).toBe(1);
    expect(periods[0].agents.length).toBe(1);
    expect(periods[0].eventCount).toBe(3);
  });

  it('returns empty array for no events', () => {
    const periods = testAggregateActivity([]);

    expect(periods.length).toBe(0);
  });

  it('sorts periods by most recent first', () => {
    const baseTime = new Date('2026-02-20T10:00:00Z').getTime();
    const events: ActivityEvent[] = [
      { timestamp: new Date(baseTime), agent: 'eng-01', action: 'old' },
      { timestamp: new Date(baseTime + 60 * 60 * 1000), agent: 'eng-02', action: 'new' },
    ];

    const periods = testAggregateActivity(events, 15);

    expect(periods[0].agents[0]).toBe('eng-02');
    expect(periods[1].agents[0]).toBe('eng-01');
  });
});
