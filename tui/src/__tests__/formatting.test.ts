/**
 * formatting.ts unit tests
 * Tests shared formatting utility functions
 */

import { describe, expect, test } from 'bun:test';
import {
  formatRelativeTime,
  formatDuration,
  truncate,
  formatNumber,
  formatBytes,
  formatCost,
  capitalize,
  toTitleCase,
} from '../utils/formatting';

describe('formatting - formatRelativeTime', () => {
  test('returns "now" for very recent timestamps', () => {
    const now = new Date();
    expect(formatRelativeTime(now)).toBe('now');
  });

  test('formats minutes ago', () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000);
    expect(formatRelativeTime(fiveMinutesAgo)).toBe('5m ago');
  });

  test('formats 1 minute ago', () => {
    const oneMinuteAgo = new Date(Date.now() - 1 * 60 * 1000);
    expect(formatRelativeTime(oneMinuteAgo)).toBe('1m ago');
  });

  test('formats 59 minutes ago', () => {
    const time = new Date(Date.now() - 59 * 60 * 1000);
    expect(formatRelativeTime(time)).toBe('59m ago');
  });

  test('formats hours ago', () => {
    const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000);
    expect(formatRelativeTime(twoHoursAgo)).toBe('2h ago');
  });

  test('formats 23 hours ago', () => {
    const time = new Date(Date.now() - 23 * 60 * 60 * 1000);
    expect(formatRelativeTime(time)).toBe('23h ago');
  });

  test('formats days ago', () => {
    const threeDaysAgo = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000);
    expect(formatRelativeTime(threeDaysAgo)).toBe('3d ago');
  });

  test('formats 6 days ago', () => {
    const sixDaysAgo = new Date(Date.now() - 6 * 24 * 60 * 60 * 1000);
    expect(formatRelativeTime(sixDaysAgo)).toBe('6d ago');
  });

  test('formats older dates as date string', () => {
    const twoWeeksAgo = new Date(Date.now() - 14 * 24 * 60 * 60 * 1000);
    const result = formatRelativeTime(twoWeeksAgo);
    // Should be "Mon DD" format
    expect(result).toMatch(/^[A-Z][a-z]{2} \d{1,2}$/);
  });

  test('handles ISO string input', () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000);
    expect(formatRelativeTime(fiveMinutesAgo.toISOString())).toBe('5m ago');
  });

  test('handles invalid date string', () => {
    // Invalid dates return the toLocaleDateString output which is 'Invalid Date'
    expect(formatRelativeTime('invalid-date')).toBe('Invalid Date');
  });
});

describe('formatting - formatDuration', () => {
  test('formats milliseconds', () => {
    expect(formatDuration(500)).toBe('500ms');
  });

  test('formats 0 milliseconds', () => {
    expect(formatDuration(0)).toBe('0ms');
  });

  test('formats less than 1 second', () => {
    expect(formatDuration(999)).toBe('999ms');
  });

  test('formats 1 second', () => {
    expect(formatDuration(1000)).toBe('1s');
  });

  test('formats seconds only', () => {
    expect(formatDuration(30 * 1000)).toBe('30s');
  });

  test('formats minutes and seconds', () => {
    expect(formatDuration(90 * 1000)).toBe('1m 30s');
  });

  test('formats exact minutes', () => {
    expect(formatDuration(5 * 60 * 1000)).toBe('5m');
  });

  test('formats 59 minutes 59 seconds', () => {
    expect(formatDuration((59 * 60 + 59) * 1000)).toBe('59m 59s');
  });

  test('formats hours and minutes', () => {
    expect(formatDuration((2 * 60 * 60 + 30 * 60) * 1000)).toBe('2h 30m');
  });

  test('formats exact hours', () => {
    expect(formatDuration(3 * 60 * 60 * 1000)).toBe('3h');
  });

  test('formats many hours', () => {
    expect(formatDuration(25 * 60 * 60 * 1000)).toBe('25h');
  });

  test('rounds milliseconds', () => {
    expect(formatDuration(500.7)).toBe('501ms');
  });
});

describe('formatting - truncate', () => {
  test('returns empty string for null', () => {
    expect(truncate(null, 10)).toBe('');
  });

  test('returns empty string for undefined', () => {
    expect(truncate(undefined, 10)).toBe('');
  });

  test('returns unchanged string when shorter than maxLength', () => {
    expect(truncate('hello', 10)).toBe('hello');
  });

  test('returns unchanged string when equal to maxLength', () => {
    expect(truncate('hello', 5)).toBe('hello');
  });

  test('truncates with ellipsis when longer', () => {
    expect(truncate('hello world', 8)).toBe('hello...');
  });

  test('handles very short maxLength', () => {
    expect(truncate('hello world', 4)).toBe('h...');
  });

  test('handles empty string', () => {
    expect(truncate('', 10)).toBe('');
  });

  test('handles single character', () => {
    expect(truncate('a', 10)).toBe('a');
  });

  test('truncates long strings correctly', () => {
    const longString = 'a'.repeat(100);
    const result = truncate(longString, 20);
    expect(result.length).toBe(20);
    expect(result).toBe('aaaaaaaaaaaaaaaaa...');
  });
});

describe('formatting - formatNumber', () => {
  test('formats zero', () => {
    expect(formatNumber(0)).toBe('0');
  });

  test('formats small numbers without commas', () => {
    expect(formatNumber(100)).toBe('100');
    expect(formatNumber(999)).toBe('999');
  });

  test('formats thousands with commas', () => {
    expect(formatNumber(1000)).toBe('1,000');
    expect(formatNumber(1234)).toBe('1,234');
  });

  test('formats millions with commas', () => {
    expect(formatNumber(1000000)).toBe('1,000,000');
    expect(formatNumber(1234567)).toBe('1,234,567');
  });

  test('formats negative numbers', () => {
    expect(formatNumber(-1234)).toBe('-1,234');
  });

  test('formats decimals', () => {
    const result = formatNumber(1234.56);
    expect(result).toContain('1,234');
  });
});

describe('formatting - formatBytes', () => {
  test('formats 0 bytes', () => {
    expect(formatBytes(0)).toBe('0 B');
  });

  test('formats bytes', () => {
    expect(formatBytes(500)).toBe('500 B');
  });

  test('formats kilobytes', () => {
    expect(formatBytes(1024)).toBe('1.0 KB');
    expect(formatBytes(1536)).toBe('1.5 KB');
  });

  test('formats megabytes', () => {
    expect(formatBytes(1024 * 1024)).toBe('1.0 MB');
    expect(formatBytes(1.5 * 1024 * 1024)).toBe('1.5 MB');
  });

  test('formats gigabytes', () => {
    expect(formatBytes(1024 * 1024 * 1024)).toBe('1.0 GB');
    expect(formatBytes(2.5 * 1024 * 1024 * 1024)).toBe('2.5 GB');
  });

  test('formats terabytes', () => {
    expect(formatBytes(1024 * 1024 * 1024 * 1024)).toBe('1.0 TB');
  });

  test('formats fractional kilobytes', () => {
    expect(formatBytes(1500)).toBe('1.5 KB');
  });

  test('handles edge case at 1023 bytes', () => {
    expect(formatBytes(1023)).toBe('1023 B');
  });
});

describe('formatting - formatCost', () => {
  test('formats zero', () => {
    expect(formatCost(0)).toBe('<$0.01');
  });

  test('formats very small costs', () => {
    expect(formatCost(0.001)).toBe('<$0.01');
    expect(formatCost(0.009)).toBe('<$0.01');
  });

  test('formats one cent', () => {
    expect(formatCost(0.01)).toBe('$0.01');
  });

  test('formats dollars', () => {
    expect(formatCost(1.0)).toBe('$1.00');
    expect(formatCost(1.5)).toBe('$1.50');
  });

  test('formats cents with precision', () => {
    expect(formatCost(0.99)).toBe('$0.99');
  });

  test('formats larger amounts', () => {
    expect(formatCost(123.45)).toBe('$123.45');
  });

  test('rounds to two decimal places', () => {
    expect(formatCost(1.234)).toBe('$1.23');
    expect(formatCost(1.235)).toBe('$1.24');
  });
});

describe('formatting - capitalize', () => {
  test('capitalizes first letter', () => {
    expect(capitalize('hello')).toBe('Hello');
  });

  test('handles already capitalized', () => {
    expect(capitalize('Hello')).toBe('Hello');
  });

  test('handles empty string', () => {
    expect(capitalize('')).toBe('');
  });

  test('handles single character', () => {
    expect(capitalize('a')).toBe('A');
  });

  test('handles single uppercase character', () => {
    expect(capitalize('A')).toBe('A');
  });

  test('only capitalizes first letter', () => {
    expect(capitalize('hELLO')).toBe('HELLO');
  });

  test('handles numbers', () => {
    expect(capitalize('123abc')).toBe('123abc');
  });
});

describe('formatting - toTitleCase', () => {
  test('converts space-separated words', () => {
    expect(toTitleCase('hello world')).toBe('Hello World');
  });

  test('converts underscore-separated words', () => {
    expect(toTitleCase('hello_world')).toBe('Hello World');
  });

  test('converts dash-separated words', () => {
    expect(toTitleCase('hello-world')).toBe('Hello World');
  });

  test('handles mixed separators', () => {
    expect(toTitleCase('hello_world-test case')).toBe('Hello World Test Case');
  });

  test('handles single word', () => {
    expect(toTitleCase('hello')).toBe('Hello');
  });

  test('handles empty string', () => {
    expect(toTitleCase('')).toBe('');
  });

  test('handles all caps', () => {
    expect(toTitleCase('HELLO WORLD')).toBe('HELLO WORLD');
  });

  test('handles multiple spaces', () => {
    expect(toTitleCase('hello  world')).toBe('Hello World');
  });
});
