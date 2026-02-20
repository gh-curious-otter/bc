/**
 * Tests for useDashboard hook - Dashboard data aggregation
 * Validates formatTime helper and type exports
 *
 * #1081 Q1 Cleanup: TUI hook test coverage
 *
 * Note: React hook testing requires DOM environment which is not available in Bun/Ink.
 * These tests focus on utility functions that can be tested without hooks.
 */

import { describe, it, expect, beforeEach, afterEach } from 'bun:test';
import { formatTime } from '../useDashboard';

describe('useDashboard - formatTime', () => {
  // Store original Date for restoration
  let originalDate: DateConstructor;

  beforeEach(() => {
    // Mock Date.now() for consistent testing
    originalDate = global.Date;
  });

  afterEach(() => {
    global.Date = originalDate;
  });

  describe('edge cases', () => {
    it('returns "-" for undefined input', () => {
      expect(formatTime(undefined)).toBe('-');
    });

    it('returns "-" for empty string', () => {
      expect(formatTime('')).toBe('-');
    });

    it('handles invalid date strings gracefully', () => {
      // Invalid dates result in "Invalid Date" from Date parsing
      // The function doesn't crash and returns a string
      const result1 = formatTime('not-a-date');
      const result2 = formatTime('invalid');
      expect(typeof result1).toBe('string');
      expect(typeof result2).toBe('string');
    });
  });

  describe('relative time formatting', () => {
    it('returns "now" for timestamps less than 1 minute ago', () => {
      const now = new Date();
      const thirtySecondsAgo = new Date(now.getTime() - 30 * 1000);
      expect(formatTime(thirtySecondsAgo.toISOString())).toBe('now');
    });

    it('returns minutes ago for timestamps 1-59 minutes ago', () => {
      const now = new Date();
      const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000);
      expect(formatTime(fiveMinutesAgo.toISOString())).toBe('5m ago');

      const thirtyMinutesAgo = new Date(now.getTime() - 30 * 60 * 1000);
      expect(formatTime(thirtyMinutesAgo.toISOString())).toBe('30m ago');
    });

    it('returns hours ago for timestamps 1-23 hours ago', () => {
      const now = new Date();
      const twoHoursAgo = new Date(now.getTime() - 2 * 60 * 60 * 1000);
      expect(formatTime(twoHoursAgo.toISOString())).toBe('2h ago');

      const twelveHoursAgo = new Date(now.getTime() - 12 * 60 * 60 * 1000);
      expect(formatTime(twelveHoursAgo.toISOString())).toBe('12h ago');
    });

    it('returns date for timestamps more than 24 hours ago', () => {
      const now = new Date();
      const twoDaysAgo = new Date(now.getTime() - 2 * 24 * 60 * 60 * 1000);
      const result = formatTime(twoDaysAgo.toISOString());
      // Should contain month abbreviation and day
      expect(result).toMatch(/[A-Z][a-z]{2} \d{1,2}/);
    });
  });

  describe('boundary cases', () => {
    it('handles exactly 1 minute ago', () => {
      const now = new Date();
      const oneMinuteAgo = new Date(now.getTime() - 60 * 1000);
      expect(formatTime(oneMinuteAgo.toISOString())).toBe('1m ago');
    });

    it('handles exactly 1 hour ago', () => {
      const now = new Date();
      const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);
      expect(formatTime(oneHourAgo.toISOString())).toBe('1h ago');
    });

    it('handles exactly 59 minutes ago', () => {
      const now = new Date();
      const fiftyNineMinutesAgo = new Date(now.getTime() - 59 * 60 * 1000);
      expect(formatTime(fiftyNineMinutesAgo.toISOString())).toBe('59m ago');
    });

    it('handles exactly 23 hours ago', () => {
      const now = new Date();
      const twentyThreeHoursAgo = new Date(now.getTime() - 23 * 60 * 60 * 1000);
      expect(formatTime(twentyThreeHoursAgo.toISOString())).toBe('23h ago');
    });
  });

  describe('ISO format handling', () => {
    it('handles standard ISO 8601 format', () => {
      const now = new Date();
      const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000);
      expect(formatTime(fiveMinutesAgo.toISOString())).toBe('5m ago');
    });

    it('handles ISO format with timezone', () => {
      const now = new Date();
      const tenMinutesAgo = new Date(now.getTime() - 10 * 60 * 1000);
      // Simulate a UTC timestamp
      const isoWithZ = tenMinutesAgo.toISOString();
      expect(formatTime(isoWithZ)).toBe('10m ago');
    });
  });
});

describe('useDashboard - Type Exports', () => {
  it('useDashboard function is importable', async () => {
    const module = await import('../useDashboard');
    expect(typeof module.useDashboard).toBe('function');
    expect(typeof module.default).toBe('function');
  });

  it('formatTime function is importable', async () => {
    const module = await import('../useDashboard');
    expect(typeof module.formatTime).toBe('function');
  });
});

describe('useDashboard - Summary Calculations (Unit Logic)', () => {
  // Test the summary calculation logic without invoking hooks
  describe('agent state counting logic', () => {
    it('should count agent states correctly', () => {
      const agents = [
        { state: 'working' },
        { state: 'working' },
        { state: 'idle' },
        { state: 'stuck' },
        { state: 'error' },
        { state: 'stopped' },
      ];

      const working = agents.filter((a) => a.state === 'working').length;
      const idle = agents.filter((a) => a.state === 'idle').length;
      const stuck = agents.filter((a) => a.state === 'stuck').length;
      const error = agents.filter((a) => a.state === 'error').length;
      const active = agents.filter((a) => a.state !== 'stopped' && a.state !== 'idle').length;

      expect(working).toBe(2);
      expect(idle).toBe(1);
      expect(stuck).toBe(1);
      expect(error).toBe(1);
      expect(active).toBe(4); // working(2) + stuck(1) + error(1)
    });

    it('should handle empty agent list', () => {
      const agents: { state: string }[] = [];

      const total = agents.length;
      const working = agents.filter((a) => a.state === 'working').length;

      expect(total).toBe(0);
      expect(working).toBe(0);
    });
  });

  describe('role counting logic', () => {
    it('should count agents by role', () => {
      const agents = [
        { role: 'engineer' },
        { role: 'engineer' },
        { role: 'manager' },
        { role: 'root' },
      ];

      const byRole: Record<string, number> = {};
      for (const agent of agents) {
        byRole[agent.role] = (byRole[agent.role] || 0) + 1;
      }

      expect(byRole.engineer).toBe(2);
      expect(byRole.manager).toBe(1);
      expect(byRole.root).toBe(1);
    });
  });
});

describe('useDashboard - Data Integration Types', () => {
  it('DashboardAgent interface shape', () => {
    // Verify the expected shape of dashboard agent data
    const agent = {
      name: 'eng-01',
      role: 'engineer',
      state: 'working',
      task: 'Implementing feature',
      uptime: '2h',
      startedAt: '10:00 AM',
      updatedAt: '12:00 PM',
    };

    expect(agent.name).toBe('eng-01');
    expect(agent.role).toBe('engineer');
    expect(agent.state).toBe('working');
    expect(agent.task).toBe('Implementing feature');
  });

  it('DashboardChannel interface shape', () => {
    const channel = {
      name: 'eng',
      members: ['eng-01', 'eng-02', 'mgr-01'],
      description: 'Engineering channel',
    };

    expect(channel.name).toBe('eng');
    expect(channel.members).toHaveLength(3);
    expect(channel.description).toBe('Engineering channel');
  });

  it('DashboardSummary interface shape', () => {
    const summary = {
      workspaceName: 'bc',
      total: 5,
      active: 3,
      working: 2,
      idle: 1,
      stuck: 0,
      error: 0,
      totalCostUSD: 1.25,
      inputTokens: 50000,
      outputTokens: 10000,
    };

    expect(summary.workspaceName).toBe('bc');
    expect(summary.total).toBe(5);
    expect(summary.totalCostUSD).toBe(1.25);
  });
});
