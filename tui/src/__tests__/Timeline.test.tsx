/**
 * Timeline Tests
 * Issue #1046: Data visualization components
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { Timeline, AgentTimeline, TimelineLegend } from '../components/Timeline';
import type { TimelineSegment } from '../components/Timeline';

describe('Timeline', () => {
  describe('empty state', () => {
    it('renders empty message when no segments', () => {
      const { lastFrame } = render(<Timeline segments={[]} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('no data');
    });

    it('renders with label when no segments', () => {
      const { lastFrame } = render(<Timeline segments={[]} label="Test" />);
      const output = lastFrame() ?? '';
      expect(output).toContain('Test');
      expect(output).toContain('no data');
    });
  });

  describe('rendering', () => {
    const baseTime = Date.now();
    const segments: TimelineSegment[] = [
      { start: baseTime, end: baseTime + 1000, status: 'working' },
      { start: baseTime + 1000, end: baseTime + 2000, status: 'idle' },
    ];

    it('renders timeline with segments', () => {
      const { lastFrame } = render(<Timeline segments={segments} width={20} />);
      const output = lastFrame() ?? '';
      // Should contain working (█) and idle (░) characters
      expect(output).toContain('█');
      expect(output).toContain('░');
    });

    it('shows time labels by default', () => {
      const { lastFrame } = render(<Timeline segments={segments} width={20} />);
      const output = lastFrame() ?? '';
      // Should have time format HH:MM
      expect(output).toMatch(/\d{2}:\d{2}/);
    });

    it('hides time labels when showTimeLabels=false', () => {
      const { lastFrame } = render(
        <Timeline segments={segments} width={20} showTimeLabels={false} />
      );
      const output = lastFrame() ?? '';
      // Output should be a single line (no time labels below)
      const lines = output.split('\n').filter((l) => l.trim() !== '');
      expect(lines.length).toBe(1);
    });

    it('shows label when provided', () => {
      const { lastFrame } = render(
        <Timeline segments={segments} label="Activity" width={20} />
      );
      const output = lastFrame() ?? '';
      expect(output).toContain('Activity:');
    });
  });

  describe('status colors', () => {
    const baseTime = Date.now();

    it('renders working status with filled chars', () => {
      const segments: TimelineSegment[] = [
        { start: baseTime, end: baseTime + 1000, status: 'working' },
      ];
      const { lastFrame } = render(<Timeline segments={segments} width={10} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('█');
    });

    it('renders stuck status with half-filled chars', () => {
      const segments: TimelineSegment[] = [
        { start: baseTime, end: baseTime + 1000, status: 'stuck' },
      ];
      const { lastFrame } = render(<Timeline segments={segments} width={10} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('▓');
    });

    it('renders error status with light chars', () => {
      const segments: TimelineSegment[] = [
        { start: baseTime, end: baseTime + 1000, status: 'error' },
      ];
      const { lastFrame } = render(<Timeline segments={segments} width={10} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('▒');
    });

    it('renders done status correctly', () => {
      const segments: TimelineSegment[] = [
        { start: baseTime, end: baseTime + 1000, status: 'done' },
      ];
      const { lastFrame } = render(<Timeline segments={segments} width={10} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('▄');
    });
  });

  describe('width handling', () => {
    const baseTime = Date.now();
    const segments: TimelineSegment[] = [
      { start: baseTime, end: baseTime + 1000, status: 'working' },
    ];

    it('respects custom width', () => {
      const { lastFrame } = render(
        <Timeline segments={segments} width={30} showTimeLabels={false} />
      );
      const output = lastFrame() ?? '';
      // The line should be approximately 30 chars (may have label)
      const mainLine = output.split('\n')[0] ?? '';
      expect(mainLine.length).toBeGreaterThanOrEqual(20);
    });

    it('uses default width of 40', () => {
      const { lastFrame } = render(
        <Timeline segments={segments} showTimeLabels={false} />
      );
      const output = lastFrame() ?? '';
      const mainLine = output.split('\n')[0] ?? '';
      expect(mainLine.length).toBeGreaterThanOrEqual(30);
    });
  });
});

describe('AgentTimeline', () => {
  const baseTime = Date.now();

  it('renders agent label', () => {
    const { lastFrame } = render(
      <AgentTimeline agent="eng-01" segments={[]} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('eng-01');
  });

  it('shows timeline bars with separators', () => {
    const segments: TimelineSegment[] = [
      { start: baseTime, end: baseTime + 1000, status: 'working' },
    ];
    const { lastFrame } = render(
      <AgentTimeline agent="eng-01" segments={segments} width={20} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('|');
  });

  it('pads agent name to consistent width', () => {
    const { lastFrame } = render(
      <AgentTimeline agent="a" segments={[]} />
    );
    const output = lastFrame() ?? '';
    // Agent name should be padded
    expect(output).toContain('a');
    const firstPart = output.split('|')[0] ?? '';
    expect(firstPart.length).toBeGreaterThanOrEqual(10);
  });

  it('handles empty segments gracefully', () => {
    const { lastFrame } = render(
      <AgentTimeline agent="eng-02" segments={[]} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('eng-02');
    expect(output).toContain('░');
  });
});

describe('TimelineLegend', () => {
  it('shows working status', () => {
    const { lastFrame } = render(<TimelineLegend />);
    const output = lastFrame() ?? '';
    expect(output).toContain('working');
    expect(output).toContain('█');
  });

  it('shows idle status', () => {
    const { lastFrame } = render(<TimelineLegend />);
    const output = lastFrame() ?? '';
    expect(output).toContain('idle');
    expect(output).toContain('░');
  });

  it('shows stuck status', () => {
    const { lastFrame } = render(<TimelineLegend />);
    const output = lastFrame() ?? '';
    expect(output).toContain('stuck');
  });

  it('shows done status', () => {
    const { lastFrame } = render(<TimelineLegend />);
    const output = lastFrame() ?? '';
    expect(output).toContain('done');
  });

  it('includes legend label', () => {
    const { lastFrame } = render(<TimelineLegend />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Legend:');
  });
});
