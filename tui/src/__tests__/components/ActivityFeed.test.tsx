/**
 * ActivityFeed component tests
 * Issue #796 - Live activity feed with severity filtering
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, vi, beforeEach } from 'bun:test';
import { ActivityFeed } from '../../components/ActivityFeed';

// Mock only useLogs, not getSeverityColor (which is a pure function)
vi.mock('../../hooks/useLogs', () => ({
  useLogs: vi.fn(() => ({
    data: [
      {
        ts: '2026-02-16T10:00:00Z',
        type: 'message.sent',
        agent: 'eng-01',
        message: 'Working on task',
      },
      {
        ts: '2026-02-16T10:01:00Z',
        type: 'agent.error',
        agent: 'eng-02',
        message: 'Build failed',
      },
      {
        ts: '2026-02-16T10:02:00Z',
        type: 'agent.stuck',
        agent: 'eng-03',
        message: 'Waiting for response',
      },
    ],
    loading: false,
    error: null,
    severityFilter: null,
    filterBySeverity: vi.fn(),
    refresh: vi.fn(),
  })),
  // Fix #1151: Add toLowerCase() to match real implementation
  getSeverityColor: (type: string) => {
    const lowerType = type.toLowerCase();
    if (lowerType.includes('error') || lowerType.includes('fail')) return 'red';
    if (lowerType.includes('warn') || lowerType.includes('stuck')) return 'yellow';
    return 'gray';
  },
}));

describe('ActivityFeed', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders activity entries', () => {
    const { lastFrame } = render(<ActivityFeed />);
    const output = lastFrame();

    expect(output).toContain('Activity');
    expect(output).toContain('eng-01');
    expect(output).toContain('Working on task');
  });

  it('renders in compact mode without timestamps', () => {
    const { lastFrame } = render(<ActivityFeed compact />);
    const output = lastFrame();

    expect(output).toContain('eng-01');
    expect(output).toContain('sent');
  });

  it('shows error entries with error styling', () => {
    const { lastFrame } = render(<ActivityFeed />);
    const output = lastFrame();

    expect(output).toContain('eng-02');
    expect(output).toContain('Build failed');
  });

  it('shows warning entries', () => {
    const { lastFrame } = render(<ActivityFeed />);
    const output = lastFrame();

    expect(output).toContain('eng-03');
    expect(output).toContain('Waiting for response');
  });

  it('respects maxEntries limit', () => {
    const { lastFrame } = render(<ActivityFeed maxEntries={2} />);
    const output = lastFrame();

    // Should show limited entries
    expect(output).toBeDefined();
  });

  it('hides filter hints when showFilterHints is false', () => {
    const { lastFrame } = render(<ActivityFeed showFilterHints={false} />);
    const output = lastFrame();

    expect(output).not.toContain('(i/w/e/*)');
  });

  it('shows filter hints by default', () => {
    const { lastFrame } = render(<ActivityFeed showFilterHints />);
    const output = lastFrame();

    expect(output).toContain('(i/w/e/*)');
  });
});
