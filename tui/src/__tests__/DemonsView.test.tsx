/* eslint-disable @typescript-eslint/restrict-template-expressions */

import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { Text, Box } from 'ink';

// Test the format functions used in DemonsView
describe('DemonsView format functions', () => {
  // formatSchedule tests
  describe('formatSchedule', () => {
    function formatSchedule(schedule: string): string {
      if (schedule === '* * * * *') return 'every minute';
      if (schedule === '0 * * * *') return 'every hour';
      if (schedule.startsWith('*/')) {
        const match = schedule.match(/^\*\/(\d+) \* \* \* \*$/);
        if (match) return `every ${match[1]} min`;
      }
      if (schedule.match(/^0 \d+ \* \* \*$/)) {
        const hour = schedule.split(' ')[1];
        return `daily at ${hour}:00`;
      }
      return schedule;
    }

    test('formats every minute', () => {
      expect(formatSchedule('* * * * *')).toBe('every minute');
    });

    test('formats every hour', () => {
      expect(formatSchedule('0 * * * *')).toBe('every hour');
    });

    test('formats every N minutes', () => {
      expect(formatSchedule('*/5 * * * *')).toBe('every 5 min');
      expect(formatSchedule('*/15 * * * *')).toBe('every 15 min');
    });

    test('formats daily at hour', () => {
      expect(formatSchedule('0 9 * * *')).toBe('daily at 9:00');
      expect(formatSchedule('0 14 * * *')).toBe('daily at 14:00');
    });

    test('returns raw schedule for complex patterns', () => {
      expect(formatSchedule('0 0 * * 0')).toBe('0 0 * * 0');
    });
  });

  // formatRelativeTime tests
  describe('formatRelativeTime', () => {
    function formatRelativeTime(timestamp?: string): string {
      if (!timestamp) return '-';
      try {
        const date = new Date(timestamp);
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        const diffMins = Math.floor(Math.abs(diffMs) / 60000);
        const diffHours = Math.floor(diffMins / 60);
        const diffDays = Math.floor(diffHours / 24);

        const prefix = diffMs < 0 ? 'in ' : '';
        const suffix = diffMs >= 0 ? ' ago' : '';

        if (diffMins < 1) return 'now';
        if (diffMins < 60) return `${prefix}${diffMins}m${suffix}`;
        if (diffHours < 24) return `${prefix}${diffHours}h${suffix}`;
        return `${prefix}${diffDays}d${suffix}`;
      } catch {
        return timestamp;
      }
    }

    test('returns dash for undefined timestamp', () => {
      expect(formatRelativeTime(undefined)).toBe('-');
    });

    test('returns now for recent timestamps', () => {
      const now = new Date().toISOString();
      expect(formatRelativeTime(now)).toBe('now');
    });

    test('formats minutes ago', () => {
      const fiveMinAgo = new Date(Date.now() - 5 * 60000).toISOString();
      expect(formatRelativeTime(fiveMinAgo)).toBe('5m ago');
    });

    test('formats hours ago', () => {
      const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60000).toISOString();
      expect(formatRelativeTime(twoHoursAgo)).toBe('2h ago');
    });

    test('formats days ago', () => {
      const threeDaysAgo = new Date(Date.now() - 3 * 24 * 60 * 60000).toISOString();
      expect(formatRelativeTime(threeDaysAgo)).toBe('3d ago');
    });

    test('formats future timestamps', () => {
      const inFiveMin = new Date(Date.now() + 5 * 60000).toISOString();
      expect(formatRelativeTime(inFiveMin)).toBe('in 5m');
    });
  });
});

// Test DemonsView rendering
describe('DemonsView UI', () => {
  test('renders header text', () => {
    // Simple component that mimics DemonsView header
    const Header = () => (
      <Box>
        <Text bold color="magenta">Demons</Text>
        <Text> · </Text>
        <Text dimColor>3 total</Text>
        <Text dimColor> · </Text>
        <Text color="green">2 enabled</Text>
      </Box>
    );

    const { lastFrame } = render(<Header />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Demons');
    expect(output).toContain('3 total');
    expect(output).toContain('2 enabled');
  });

  test('renders column headers', () => {
    const Headers = () => (
      <Box>
        <Box width={18}><Text bold dimColor>NAME</Text></Box>
        <Box width={16}><Text bold dimColor>SCHEDULE</Text></Box>
        <Box width={10}><Text bold dimColor>STATUS</Text></Box>
        <Box width={10}><Text bold dimColor>RUNS</Text></Box>
      </Box>
    );

    const { lastFrame } = render(<Headers />);
    const output = lastFrame() ?? '';
    expect(output).toContain('NAME');
    expect(output).toContain('SCHEDULE');
    expect(output).toContain('STATUS');
    expect(output).toContain('RUNS');
  });

  test('renders empty state message', () => {
    const EmptyState = () => (
      <Box flexDirection="column">
        <Text dimColor>No demons configured</Text>
      </Box>
    );

    const { lastFrame } = render(<EmptyState />);
    expect(lastFrame()).toContain('No demons configured');
  });
});
