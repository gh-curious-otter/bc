/**
 * Tests for useActivityData hook - Activity timeline aggregation
 * Validates parseLogToActivity and aggregateActivity helper functions
 *
 * #1081 Q1 Cleanup: TUI hook test coverage
 *
 * Note: React hook testing requires DOM environment which is not available in Bun/Ink.
 * These tests focus on utility functions that can be tested without hooks.
 */

import { describe, it, expect } from 'bun:test';
import {
  parseLogToActivity,
  aggregateActivity,
  type ActivityEvent,
  type ActivityPeriod,
} from '../useActivityData';
import type { LogEntry } from '../../types';

describe('useActivityData - parseLogToActivity', () => {
  describe('state change events', () => {
    it('parses state.working events', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:00:00Z',
        agent: 'eng-01',
        type: 'state.working',
        message: 'Agent started working',
      };

      const result = parseLogToActivity(log);
      expect(result).not.toBeNull();
      expect(result?.agent).toBe('eng-01');
      expect(result?.action).toBe('working');
    });

    it('parses state.idle events', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:00:00Z',
        agent: 'eng-02',
        type: 'state.idle',
        message: 'Agent idle',
      };

      const result = parseLogToActivity(log);
      expect(result?.agent).toBe('eng-02');
      expect(result?.action).toBe('idle');
    });

    it('parses state.done events', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:00:00Z',
        agent: 'eng-03',
        type: 'state.done',
        message: 'Task completed',
      };

      const result = parseLogToActivity(log);
      expect(result?.action).toBe('done');
    });

    it('parses state.stuck events', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:00:00Z',
        agent: 'eng-01',
        type: 'state.stuck',
        message: 'Agent stuck',
      };

      const result = parseLogToActivity(log);
      expect(result?.action).toBe('stuck');
    });

    it('parses state.error events', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:00:00Z',
        agent: 'eng-01',
        type: 'state.error',
        message: 'Error occurred',
      };

      const result = parseLogToActivity(log);
      expect(result?.action).toBe('error');
    });
  });

  describe('agent lifecycle events', () => {
    it('parses agent.started events', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:00:00Z',
        agent: 'eng-01',
        type: 'agent.started',
        message: 'Agent started',
      };

      const result = parseLogToActivity(log);
      expect(result?.action).toBe('started');
    });

    it('parses agent.stopped events', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:00:00Z',
        agent: 'eng-01',
        type: 'agent.stopped',
        message: 'Agent stopped',
      };

      const result = parseLogToActivity(log);
      expect(result?.action).toBe('stopped');
    });
  });

  describe('non-state events', () => {
    it('filters out non-state log types', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:00:00Z',
        agent: 'eng-01',
        type: 'message.sent',
        message: 'Message sent',
      };

      const result = parseLogToActivity(log);
      expect(result).toBeNull();
    });

    it('filters out command events', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:00:00Z',
        agent: 'eng-01',
        type: 'command.executed',
        message: 'Command ran',
      };

      const result = parseLogToActivity(log);
      expect(result).toBeNull();
    });
  });

  describe('timestamp parsing', () => {
    it('parses ISO timestamp correctly', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:30:45Z',
        agent: 'eng-01',
        type: 'state.working',
        message: 'Working',
      };

      const result = parseLogToActivity(log);
      expect(result?.timestamp.toISOString()).toBe('2024-02-20T10:30:45.000Z');
    });

    it('handles different timezone formats', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:30:45+00:00',
        agent: 'eng-01',
        type: 'state.working',
        message: 'Working',
      };

      const result = parseLogToActivity(log);
      expect(result?.timestamp).toBeDefined();
    });
  });

  describe('cost handling', () => {
    it('defaults cost to 0', () => {
      const log: LogEntry = {
        ts: '2024-02-20T10:00:00Z',
        agent: 'eng-01',
        type: 'state.working',
        message: 'Working',
      };

      const result = parseLogToActivity(log);
      expect(result?.cost).toBe(0);
    });
  });
});

describe('useActivityData - aggregateActivity', () => {
  const createEvent = (timestamp: string, agent: string, action: string): ActivityEvent => ({
    timestamp: new Date(timestamp),
    agent,
    action,
    cost: 0,
  });

  describe('empty input', () => {
    it('returns empty array for empty events', () => {
      const result = aggregateActivity([]);
      expect(result).toEqual([]);
    });
  });

  describe('single event', () => {
    it('creates single period for one event', () => {
      const events = [createEvent('2024-02-20T10:05:00Z', 'eng-01', 'working')];
      const result = aggregateActivity(events, 15);

      expect(result).toHaveLength(1);
      expect(result[0].agents).toContain('eng-01');
      expect(result[0].duration).toBe(15);
    });
  });

  describe('multiple events same period', () => {
    it('groups events in same 15-minute window', () => {
      const events = [
        createEvent('2024-02-20T10:05:00Z', 'eng-01', 'working'),
        createEvent('2024-02-20T10:10:00Z', 'eng-02', 'working'),
        createEvent('2024-02-20T10:12:00Z', 'eng-03', 'idle'),
      ];
      const result = aggregateActivity(events, 15);

      expect(result).toHaveLength(1);
      expect(result[0].agents).toHaveLength(3);
      expect(result[0].agents).toContain('eng-01');
      expect(result[0].agents).toContain('eng-02');
      expect(result[0].agents).toContain('eng-03');
    });

    it('deduplicates same agent in period', () => {
      const events = [
        createEvent('2024-02-20T10:05:00Z', 'eng-01', 'working'),
        createEvent('2024-02-20T10:10:00Z', 'eng-01', 'idle'),
      ];
      const result = aggregateActivity(events, 15);

      expect(result).toHaveLength(1);
      expect(result[0].agents).toHaveLength(1);
      expect(result[0].agents).toContain('eng-01');
    });
  });

  describe('multiple periods', () => {
    it('separates events in different periods', () => {
      const events = [
        createEvent('2024-02-20T10:05:00Z', 'eng-01', 'working'),
        createEvent('2024-02-20T10:20:00Z', 'eng-02', 'working'), // Different 15-min window
      ];
      const result = aggregateActivity(events, 15);

      expect(result).toHaveLength(2);
    });

    it('sorts periods by time descending (most recent first)', () => {
      const events = [
        createEvent('2024-02-20T10:05:00Z', 'eng-01', 'working'),
        createEvent('2024-02-20T10:35:00Z', 'eng-02', 'working'),
        createEvent('2024-02-20T10:50:00Z', 'eng-03', 'working'),
      ];
      const result = aggregateActivity(events, 15);

      expect(result).toHaveLength(3);
      // Most recent first
      expect(result[0].agents).toContain('eng-03');
      expect(result[2].agents).toContain('eng-01');
    });
  });

  describe('period boundaries', () => {
    it('calculates period boundaries correctly', () => {
      const events = [createEvent('2024-02-20T10:07:30Z', 'eng-01', 'working')];
      const result = aggregateActivity(events, 15);

      expect(result).toHaveLength(1);
      // Should be in 10:00-10:15 period
      expect(result[0].startTime.getMinutes()).toBe(0);
      expect(result[0].endTime.getMinutes()).toBe(15);
    });
  });

  describe('cost aggregation', () => {
    it('sums costs within period', () => {
      const events: ActivityEvent[] = [
        { timestamp: new Date('2024-02-20T10:05:00Z'), agent: 'eng-01', action: 'working', cost: 0.5 },
        { timestamp: new Date('2024-02-20T10:10:00Z'), agent: 'eng-02', action: 'working', cost: 0.3 },
      ];
      const result = aggregateActivity(events, 15);

      expect(result).toHaveLength(1);
      expect(result[0].totalCost).toBe(0.8);
    });

    it('handles undefined costs', () => {
      const events: ActivityEvent[] = [
        { timestamp: new Date('2024-02-20T10:05:00Z'), agent: 'eng-01', action: 'working' },
      ];
      const result = aggregateActivity(events, 15);

      expect(result).toHaveLength(1);
      expect(result[0].totalCost).toBe(0);
    });
  });

  describe('different period sizes', () => {
    it('handles 30-minute periods', () => {
      const events = [
        createEvent('2024-02-20T10:05:00Z', 'eng-01', 'working'),
        createEvent('2024-02-20T10:25:00Z', 'eng-02', 'working'), // Same 30-min window
      ];
      const result = aggregateActivity(events, 30);

      expect(result).toHaveLength(1);
      expect(result[0].agents).toHaveLength(2);
      expect(result[0].duration).toBe(30);
    });

    it('handles 5-minute periods', () => {
      const events = [
        createEvent('2024-02-20T10:01:00Z', 'eng-01', 'working'),
        createEvent('2024-02-20T10:06:00Z', 'eng-02', 'working'), // Different 5-min window
      ];
      const result = aggregateActivity(events, 5);

      expect(result).toHaveLength(2);
    });
  });
});

describe('useActivityData - Type Exports', () => {
  it('exports ActivityEvent interface', () => {
    const event: ActivityEvent = {
      timestamp: new Date(),
      agent: 'eng-01',
      action: 'working',
      duration: 60,
      cost: 0.5,
    };

    expect(event.agent).toBe('eng-01');
    expect(event.action).toBe('working');
  });

  it('exports ActivityPeriod interface', () => {
    const period: ActivityPeriod = {
      startTime: new Date(),
      endTime: new Date(),
      agents: ['eng-01', 'eng-02'],
      action: 'working',
      duration: 15,
      totalCost: 1.5,
    };

    expect(period.agents).toHaveLength(2);
    expect(period.duration).toBe(15);
    expect(period.totalCost).toBe(1.5);
  });

  it('useActivityData function is importable', async () => {
    const module = await import('../useActivityData');
    expect(typeof module.useActivityData).toBe('function');
    expect(typeof module.default).toBe('function');
  });

  it('helper functions are importable', async () => {
    const module = await import('../useActivityData');
    expect(typeof module.parseLogToActivity).toBe('function');
    expect(typeof module.aggregateActivity).toBe('function');
  });
});
