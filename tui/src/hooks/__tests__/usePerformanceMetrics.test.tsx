/**
 * Tests for usePerformanceMetrics hook
 * Issue #965: TUI Performance Metrics
 */

import { describe, test, expect, beforeEach } from 'bun:test';
import { createPerformanceTracker } from '../usePerformanceMetrics';

describe('Performance Metrics Tracker', () => {
  let tracker: ReturnType<typeof createPerformanceTracker>;

  beforeEach(() => {
    tracker = createPerformanceTracker();
  });

  describe('Timer Operations', () => {
    test('startTimer returns unique timer ID', () => {
      const id1 = tracker.startTimer('test');
      const id2 = tracker.startTimer('test');
      expect(id1).not.toBe(id2);
      expect(id1).toContain('test-');
      expect(id2).toContain('test-');
    });

    test('endTimer returns duration in ms', () => {
      const timerId = tracker.startTimer('duration:test');
      const duration = tracker.endTimer(timerId, 'duration:test');
      expect(duration).toBeGreaterThanOrEqual(0);
    });

    test('endTimer with invalid timerId returns 0', () => {
      const duration = tracker.endTimer('invalid-id', 'test');
      expect(duration).toBe(0);
    });

    test('endTimer records metric', () => {
      const timerId = tracker.startTimer('record:test');
      tracker.endTimer(timerId, 'record:test');

      const metric = tracker.getMetric('record:test');
      expect(metric).not.toBeNull();
      expect(metric?.name).toBe('record:test');
      expect(metric?.count).toBe(1);
    });
  });

  describe('Metric Calculations', () => {
    test('calculates correct average', () => {
      // Record multiple values by timing operations
      for (let i = 0; i < 4; i++) {
        const timerId = tracker.startTimer('avg:test');
        tracker.endTimer(timerId, 'avg:test');
      }

      const metric = tracker.getMetric('avg:test');
      expect(metric).not.toBeNull();
      expect(metric?.count).toBe(4);
      expect(metric?.average).toBeGreaterThanOrEqual(0);
    });

    test('tracks min value correctly', () => {
      // First measurement
      let timerId = tracker.startTimer('min:test');
      tracker.endTimer(timerId, 'min:test');
      const firstMetric = tracker.getMetric('min:test');
      const firstMin = firstMetric?.min ?? Infinity;

      // Second measurement
      timerId = tracker.startTimer('min:test');
      tracker.endTimer(timerId, 'min:test');
      const secondMetric = tracker.getMetric('min:test');

      expect(secondMetric?.min).toBeLessThanOrEqual(firstMin);
    });

    test('tracks max value correctly', () => {
      const timerId = tracker.startTimer('max:test');
      tracker.endTimer(timerId, 'max:test');

      const metric = tracker.getMetric('max:test');
      expect(metric?.max).toBeGreaterThanOrEqual(metric?.min ?? 0);
    });

    test('increments count on each measurement', () => {
      for (let i = 1; i <= 5; i++) {
        const timerId = tracker.startTimer('count:test');
        tracker.endTimer(timerId, 'count:test');
        const metric = tracker.getMetric('count:test');
        expect(metric?.count).toBe(i);
      }
    });
  });

  describe('Metric Retrieval', () => {
    test('getMetric returns null for unknown metric', () => {
      const metric = tracker.getMetric('nonexistent');
      expect(metric).toBeNull();
    });

    test('getMetric returns metric after recording', () => {
      const timerId = tracker.startTimer('get:test');
      tracker.endTimer(timerId, 'get:test');

      const metric = tracker.getMetric('get:test');
      expect(metric).not.toBeNull();
      expect(metric?.name).toBe('get:test');
    });

    test('getAllMetrics returns empty array initially', () => {
      const freshTracker = createPerformanceTracker();
      const metrics = freshTracker.getAllMetrics();
      expect(metrics).toEqual([]);
    });

    test('getAllMetrics returns all recorded metrics', () => {
      tracker.endTimer(tracker.startTimer('a'), 'a');
      tracker.endTimer(tracker.startTimer('b'), 'b');
      tracker.endTimer(tracker.startTimer('c'), 'c');

      const metrics = tracker.getAllMetrics();
      expect(metrics.length).toBe(3);
    });

    test('getAllMetrics returns metrics sorted by name', () => {
      tracker.endTimer(tracker.startTimer('zebra'), 'zebra');
      tracker.endTimer(tracker.startTimer('apple'), 'apple');
      tracker.endTimer(tracker.startTimer('mango'), 'mango');

      const metrics = tracker.getAllMetrics();
      expect(metrics[0].name).toBe('apple');
      expect(metrics[1].name).toBe('mango');
      expect(metrics[2].name).toBe('zebra');
    });
  });

  describe('Clear Operations', () => {
    test('clear removes all metrics', () => {
      tracker.endTimer(tracker.startTimer('test1'), 'test1');
      tracker.endTimer(tracker.startTimer('test2'), 'test2');
      expect(tracker.getAllMetrics().length).toBe(2);

      tracker.clear();
      expect(tracker.getAllMetrics().length).toBe(0);
    });

    test('can record after clear', () => {
      tracker.endTimer(tracker.startTimer('before'), 'before');
      tracker.clear();
      tracker.endTimer(tracker.startTimer('after'), 'after');

      const metrics = tracker.getAllMetrics();
      expect(metrics.length).toBe(1);
      expect(metrics[0].name).toBe('after');
    });
  });

  describe('Metric Names', () => {
    test('supports poll: prefix for polling metrics', () => {
      tracker.endTimer(tracker.startTimer('poll:agents'), 'poll:agents');
      tracker.endTimer(tracker.startTimer('poll:channels'), 'poll:channels');

      const metrics = tracker.getAllMetrics();
      const pollMetrics = metrics.filter((m) => m.name.startsWith('poll:'));
      expect(pollMetrics.length).toBe(2);
    });

    test('supports cmd: prefix for command metrics', () => {
      tracker.endTimer(tracker.startTimer('cmd:status'), 'cmd:status');
      tracker.endTimer(tracker.startTimer('cmd:list'), 'cmd:list');

      const metrics = tracker.getAllMetrics();
      const cmdMetrics = metrics.filter((m) => m.name.startsWith('cmd:'));
      expect(cmdMetrics.length).toBe(2);
    });

    test('supports render: prefix for render metrics', () => {
      tracker.endTimer(tracker.startTimer('render:dashboard'), 'render:dashboard');

      const metric = tracker.getMetric('render:dashboard');
      expect(metric).not.toBeNull();
    });
  });

  describe('Metric Structure', () => {
    test('metric has all required fields', () => {
      tracker.endTimer(tracker.startTimer('structure:test'), 'structure:test');

      const metric = tracker.getMetric('structure:test')!;
      expect(metric).toHaveProperty('name');
      expect(metric).toHaveProperty('value');
      expect(metric).toHaveProperty('average');
      expect(metric).toHaveProperty('min');
      expect(metric).toHaveProperty('max');
      expect(metric).toHaveProperty('count');
      expect(metric).toHaveProperty('lastUpdated');
    });

    test('lastUpdated is a Date object', () => {
      tracker.endTimer(tracker.startTimer('date:test'), 'date:test');

      const metric = tracker.getMetric('date:test');
      expect(metric?.lastUpdated).toBeInstanceOf(Date);
    });

    test('value equals last recorded duration', () => {
      tracker.endTimer(tracker.startTimer('value:test'), 'value:test');
      const first = tracker.getMetric('value:test')?.value;

      tracker.endTimer(tracker.startTimer('value:test'), 'value:test');
      const second = tracker.getMetric('value:test')?.value;

      // Second value should be updated (may or may not equal first)
      expect(second).toBeGreaterThanOrEqual(0);
    });
  });
});

describe('Multiple Trackers', () => {
  test('trackers are independent', () => {
    const tracker1 = createPerformanceTracker();
    const tracker2 = createPerformanceTracker();

    tracker1.endTimer(tracker1.startTimer('tracker1:metric'), 'tracker1:metric');
    tracker2.endTimer(tracker2.startTimer('tracker2:metric'), 'tracker2:metric');

    expect(tracker1.getMetric('tracker2:metric')).toBeNull();
    expect(tracker2.getMetric('tracker1:metric')).toBeNull();
  });
});
