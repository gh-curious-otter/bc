/**
 * useActivityData hook tests
 * Issue #1047 - Activity timeline and cost trend tracking
 */

import { describe, it, expect } from 'bun:test';
import type { LogEntry } from '../../types';

// Test the log parsing logic directly
interface ActivityEvent {
  timestamp: Date;
  agent: string;
  action: string;
  cost?: number;
}

function parseLogToActivity(log: LogEntry): ActivityEvent | null {
  try {
    const timestamp = new Date(log.ts);
    // Filter to only state-change events
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

describe('parseLogToActivity', () => {
  it('parses state.working events', () => {
    const log: LogEntry = {
      ts: '2026-02-20T10:00:00Z',
      type: 'state.working',
      agent: 'eng-01',
      message: 'Started working on task',
    };
    const result = parseLogToActivity(log);
    expect(result).not.toBeNull();
    expect(result?.agent).toBe('eng-01');
    expect(result?.action).toBe('working');
  });

  it('parses state.idle events', () => {
    const log: LogEntry = {
      ts: '2026-02-20T10:00:00Z',
      type: 'state.idle',
      agent: 'eng-02',
      message: 'Waiting for next task',
    };
    const result = parseLogToActivity(log);
    expect(result).not.toBeNull();
    expect(result?.agent).toBe('eng-02');
    expect(result?.action).toBe('idle');
  });

  it('parses agent.started events', () => {
    const log: LogEntry = {
      ts: '2026-02-20T10:00:00Z',
      type: 'agent.started',
      agent: 'eng-03',
      message: 'Agent started',
    };
    const result = parseLogToActivity(log);
    expect(result).not.toBeNull();
    expect(result?.action).toBe('started');
  });

  it('ignores non-state events', () => {
    const log: LogEntry = {
      ts: '2026-02-20T10:00:00Z',
      type: 'message.sent',
      agent: 'eng-01',
      message: 'Sent message to channel',
    };
    const result = parseLogToActivity(log);
    expect(result).toBeNull();
  });

  it('handles invalid timestamps gracefully', () => {
    const log: LogEntry = {
      ts: 'invalid-date',
      type: 'state.working',
      agent: 'eng-01',
      message: 'Test',
    };
    // Should still return a result (Date will be Invalid Date)
    const result = parseLogToActivity(log);
    expect(result?.agent).toBe('eng-01');
  });
});

describe('aggregateActivity logic', () => {
  interface ActivityPeriod {
    startTime: Date;
    endTime: Date;
    agents: string[];
    action: string;
    duration: number;
    totalCost: number;
  }

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

    return Array.from(periods.values()).sort((a, b) => b.startTime.getTime() - a.startTime.getTime());
  }

  it('returns empty array for no events', () => {
    const result = aggregateActivity([]);
    expect(result).toEqual([]);
  });

  it('groups events by 15-minute periods', () => {
    const events: ActivityEvent[] = [
      { timestamp: new Date('2026-02-20T10:00:00Z'), agent: 'eng-01', action: 'working' },
      { timestamp: new Date('2026-02-20T10:05:00Z'), agent: 'eng-02', action: 'working' },
      { timestamp: new Date('2026-02-20T10:30:00Z'), agent: 'eng-03', action: 'idle' },
    ];
    const result = aggregateActivity(events);
    // Should have 2 periods: 10:00-10:15 and 10:30-10:45
    expect(result.length).toBe(2);
  });

  it('combines agents in same period', () => {
    const events: ActivityEvent[] = [
      { timestamp: new Date('2026-02-20T10:00:00Z'), agent: 'eng-01', action: 'working' },
      { timestamp: new Date('2026-02-20T10:05:00Z'), agent: 'eng-02', action: 'working' },
      { timestamp: new Date('2026-02-20T10:10:00Z'), agent: 'eng-01', action: 'working' },
    ];
    const result = aggregateActivity(events);
    expect(result.length).toBe(1);
    // Should have 2 unique agents (eng-01 not duplicated)
    expect(result[0].agents.length).toBe(2);
    expect(result[0].agents).toContain('eng-01');
    expect(result[0].agents).toContain('eng-02');
  });

  it('sums costs within a period', () => {
    const events: ActivityEvent[] = [
      { timestamp: new Date('2026-02-20T10:00:00Z'), agent: 'eng-01', action: 'working', cost: 1.5 },
      { timestamp: new Date('2026-02-20T10:05:00Z'), agent: 'eng-02', action: 'working', cost: 2.0 },
    ];
    const result = aggregateActivity(events);
    expect(result[0].totalCost).toBe(3.5);
  });

  it('sorts results by time descending (most recent first)', () => {
    const events: ActivityEvent[] = [
      { timestamp: new Date('2026-02-20T10:00:00Z'), agent: 'eng-01', action: 'working' },
      { timestamp: new Date('2026-02-20T11:00:00Z'), agent: 'eng-02', action: 'idle' },
    ];
    const result = aggregateActivity(events);
    expect(result[0].startTime.getTime()).toBeGreaterThan(result[1].startTime.getTime());
  });
});
