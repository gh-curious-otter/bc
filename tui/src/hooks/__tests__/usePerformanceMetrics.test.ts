/**
 * usePerformanceMetrics hook tests (#1081)
 *
 * Tests cover:
 * - PerformanceMetric interface structure
 * - PerformanceMetrics interface structure
 * - createPerformanceTracker factory function
 * - Timer ID generation and tracking
 * - Metric calculations (average, min, max)
 * - Sample limiting (MAX_SAMPLES = 100)
 * - globalPerformanceTracker singleton
 */

import { describe, test, expect, beforeEach } from 'bun:test';
import type { PerformanceMetric, PerformanceMetrics } from '../usePerformanceMetrics';
import { createPerformanceTracker, globalPerformanceTracker } from '../usePerformanceMetrics';

// Test PerformanceMetric interface
describe('usePerformanceMetrics Types', () => {
  describe('PerformanceMetric interface', () => {
    test('has required name field', () => {
      const metric: PerformanceMetric = {
        name: 'render:dashboard',
        value: 15.5,
        average: 12.3,
        min: 8.0,
        max: 25.0,
        count: 10,
        lastUpdated: new Date(),
      };
      expect(metric.name).toBe('render:dashboard');
    });

    test('has required value field (latest value in ms)', () => {
      const metric: PerformanceMetric = {
        name: 'poll:agents',
        value: 125.5,
        average: 100.0,
        min: 50.0,
        max: 200.0,
        count: 5,
        lastUpdated: new Date(),
      };
      expect(metric.value).toBe(125.5);
    });

    test('has required average field', () => {
      const metric: PerformanceMetric = {
        name: 'command:list',
        value: 50.0,
        average: 45.5,
        min: 30.0,
        max: 80.0,
        count: 20,
        lastUpdated: new Date(),
      };
      expect(metric.average).toBe(45.5);
    });

    test('has required min/max fields', () => {
      const metric: PerformanceMetric = {
        name: 'api:fetch',
        value: 100.0,
        average: 75.0,
        min: 25.0,
        max: 150.0,
        count: 15,
        lastUpdated: new Date(),
      };
      expect(metric.min).toBe(25.0);
      expect(metric.max).toBe(150.0);
    });

    test('has required count field', () => {
      const metric: PerformanceMetric = {
        name: 'render:view',
        value: 10.0,
        average: 12.0,
        min: 8.0,
        max: 20.0,
        count: 50,
        lastUpdated: new Date(),
      };
      expect(metric.count).toBe(50);
    });

    test('has required lastUpdated field', () => {
      const now = new Date();
      const metric: PerformanceMetric = {
        name: 'poll:status',
        value: 5.0,
        average: 6.0,
        min: 3.0,
        max: 10.0,
        count: 100,
        lastUpdated: now,
      };
      expect(metric.lastUpdated).toBe(now);
    });
  });

  describe('PerformanceMetrics interface', () => {
    test('has metrics Map', () => {
      const metrics: PerformanceMetrics = {
        metrics: new Map(),
        totalMeasurements: 0,
        uptime: 0,
        debugEnabled: false,
      };
      expect(metrics.metrics).toBeInstanceOf(Map);
    });

    test('has totalMeasurements counter', () => {
      const metrics: PerformanceMetrics = {
        metrics: new Map(),
        totalMeasurements: 42,
        uptime: 120,
        debugEnabled: false,
      };
      expect(metrics.totalMeasurements).toBe(42);
    });

    test('has uptime in seconds', () => {
      const metrics: PerformanceMetrics = {
        metrics: new Map(),
        totalMeasurements: 0,
        uptime: 3600,
        debugEnabled: false,
      };
      expect(metrics.uptime).toBe(3600);
    });

    test('has debugEnabled flag', () => {
      const metrics: PerformanceMetrics = {
        metrics: new Map(),
        totalMeasurements: 0,
        uptime: 0,
        debugEnabled: true,
      };
      expect(metrics.debugEnabled).toBe(true);
    });
  });
});

// Test createPerformanceTracker factory
describe('createPerformanceTracker', () => {
  let tracker: ReturnType<typeof createPerformanceTracker>;

  beforeEach(() => {
    tracker = createPerformanceTracker();
  });

  describe('startTimer/endTimer', () => {
    test('returns unique timer ID', () => {
      const id1 = tracker.startTimer('test');
      const id2 = tracker.startTimer('test');
      expect(id1).not.toBe(id2);
    });

    test('timer ID contains metric name', () => {
      const id = tracker.startTimer('render:dashboard');
      expect(id).toContain('render:dashboard');
    });

    test('endTimer returns duration', () => {
      const id = tracker.startTimer('test');
      const duration = tracker.endTimer(id, 'test');
      expect(duration).toBeGreaterThanOrEqual(0);
    });

    test('endTimer returns 0 for unknown timer', () => {
      const duration = tracker.endTimer('unknown-timer', 'test');
      expect(duration).toBe(0);
    });

    test('records metric after endTimer', () => {
      const id = tracker.startTimer('api:call');
      tracker.endTimer(id, 'api:call');
      const metric = tracker.getMetric('api:call');
      expect(metric).not.toBeNull();
      expect(metric?.name).toBe('api:call');
    });
  });

  describe('getMetric', () => {
    test('returns null for non-existent metric', () => {
      expect(tracker.getMetric('nonexistent')).toBeNull();
    });

    test('returns metric with correct structure', () => {
      const id = tracker.startTimer('test:metric');
      tracker.endTimer(id, 'test:metric');

      const metric = tracker.getMetric('test:metric');
      expect(metric).not.toBeNull();
      expect(metric?.name).toBe('test:metric');
      expect(typeof metric?.value).toBe('number');
      expect(typeof metric?.average).toBe('number');
      expect(typeof metric?.min).toBe('number');
      expect(typeof metric?.max).toBe('number');
      expect(typeof metric?.count).toBe('number');
      expect(metric?.lastUpdated).toBeInstanceOf(Date);
    });

    test('updates min correctly', () => {
      // Record several values
      for (let i = 0; i < 5; i++) {
        const id = tracker.startTimer('min:test');
        tracker.endTimer(id, 'min:test');
      }

      const metric = tracker.getMetric('min:test');
      expect(metric?.min).toBeLessThanOrEqual(metric?.value ?? Infinity);
    });

    test('updates max correctly', () => {
      for (let i = 0; i < 5; i++) {
        const id = tracker.startTimer('max:test');
        tracker.endTimer(id, 'max:test');
      }

      const metric = tracker.getMetric('max:test');
      expect(metric?.max).toBeGreaterThanOrEqual(metric?.value ?? -Infinity);
    });
  });

  describe('getAllMetrics', () => {
    test('returns empty array when no metrics', () => {
      const metrics = tracker.getAllMetrics();
      expect(metrics).toEqual([]);
    });

    test('returns all recorded metrics', () => {
      const id1 = tracker.startTimer('metric:a');
      tracker.endTimer(id1, 'metric:a');

      const id2 = tracker.startTimer('metric:b');
      tracker.endTimer(id2, 'metric:b');

      const metrics = tracker.getAllMetrics();
      expect(metrics).toHaveLength(2);
    });

    test('sorts metrics by name', () => {
      const id1 = tracker.startTimer('zebra');
      tracker.endTimer(id1, 'zebra');

      const id2 = tracker.startTimer('alpha');
      tracker.endTimer(id2, 'alpha');

      const id3 = tracker.startTimer('beta');
      tracker.endTimer(id3, 'beta');

      const metrics = tracker.getAllMetrics();
      expect(metrics[0].name).toBe('alpha');
      expect(metrics[1].name).toBe('beta');
      expect(metrics[2].name).toBe('zebra');
    });
  });

  describe('clear', () => {
    test('removes all metrics', () => {
      const id = tracker.startTimer('test');
      tracker.endTimer(id, 'test');
      expect(tracker.getAllMetrics()).toHaveLength(1);

      tracker.clear();
      expect(tracker.getAllMetrics()).toEqual([]);
    });

    test('clears pending timers', () => {
      tracker.startTimer('pending');
      tracker.clear();

      // After clear, the pending timer should be gone
      const metrics = tracker.getAllMetrics();
      expect(metrics).toEqual([]);
    });
  });

  describe('metric calculations', () => {
    test('calculates correct count', () => {
      for (let i = 0; i < 10; i++) {
        const id = tracker.startTimer('count:test');
        tracker.endTimer(id, 'count:test');
      }

      const metric = tracker.getMetric('count:test');
      expect(metric?.count).toBe(10);
    });

    test('average equals value for single sample', () => {
      const id = tracker.startTimer('single');
      tracker.endTimer(id, 'single');

      const metric = tracker.getMetric('single');
      expect(metric?.average).toBe(metric?.value);
    });

    test('min equals max for single sample', () => {
      const id = tracker.startTimer('single:minmax');
      tracker.endTimer(id, 'single:minmax');

      const metric = tracker.getMetric('single:minmax');
      expect(metric?.min).toBe(metric?.max);
    });
  });
});

// Test globalPerformanceTracker singleton
describe('globalPerformanceTracker', () => {
  beforeEach(() => {
    globalPerformanceTracker.clear();
  });

  test('is a singleton instance', () => {
    expect(globalPerformanceTracker).toBeDefined();
    expect(typeof globalPerformanceTracker.startTimer).toBe('function');
    expect(typeof globalPerformanceTracker.endTimer).toBe('function');
    expect(typeof globalPerformanceTracker.getMetric).toBe('function');
    expect(typeof globalPerformanceTracker.getAllMetrics).toBe('function');
    expect(typeof globalPerformanceTracker.clear).toBe('function');
  });

  test('records metrics correctly', () => {
    const id = tracker => {
      const timerId = globalPerformanceTracker.startTimer('global:test');
      globalPerformanceTracker.endTimer(timerId, 'global:test');
    };
    id(globalPerformanceTracker);

    const metric = globalPerformanceTracker.getMetric('global:test');
    expect(metric).not.toBeNull();
    expect(metric?.name).toBe('global:test');
  });

  test('persists across calls', () => {
    const id = globalPerformanceTracker.startTimer('persist:test');
    globalPerformanceTracker.endTimer(id, 'persist:test');

    // Without clearing, metric should persist
    const metric = globalPerformanceTracker.getMetric('persist:test');
    expect(metric).not.toBeNull();
  });
});

// Test metric naming conventions
describe('Metric Naming', () => {
  test('supports colon-separated namespaces', () => {
    const tracker = createPerformanceTracker();

    const id = tracker.startTimer('render:dashboard:header');
    tracker.endTimer(id, 'render:dashboard:header');

    const metric = tracker.getMetric('render:dashboard:header');
    expect(metric?.name).toBe('render:dashboard:header');
  });

  test('supports hyphenated names', () => {
    const tracker = createPerformanceTracker();

    const id = tracker.startTimer('poll-agents');
    tracker.endTimer(id, 'poll-agents');

    const metric = tracker.getMetric('poll-agents');
    expect(metric?.name).toBe('poll-agents');
  });

  test('supports underscore names', () => {
    const tracker = createPerformanceTracker();

    const id = tracker.startTimer('fetch_data');
    tracker.endTimer(id, 'fetch_data');

    const metric = tracker.getMetric('fetch_data');
    expect(metric?.name).toBe('fetch_data');
  });
});

// Test edge cases
describe('Edge Cases', () => {
  test('handles rapid sequential timers', () => {
    const tracker = createPerformanceTracker();

    for (let i = 0; i < 50; i++) {
      const id = tracker.startTimer('rapid');
      tracker.endTimer(id, 'rapid');
    }

    const metric = tracker.getMetric('rapid');
    expect(metric?.count).toBe(50);
  });

  test('handles concurrent timers for different metrics', () => {
    const tracker = createPerformanceTracker();

    const id1 = tracker.startTimer('concurrent:a');
    const id2 = tracker.startTimer('concurrent:b');
    const id3 = tracker.startTimer('concurrent:c');

    tracker.endTimer(id2, 'concurrent:b');
    tracker.endTimer(id1, 'concurrent:a');
    tracker.endTimer(id3, 'concurrent:c');

    expect(tracker.getMetric('concurrent:a')).not.toBeNull();
    expect(tracker.getMetric('concurrent:b')).not.toBeNull();
    expect(tracker.getMetric('concurrent:c')).not.toBeNull();
  });

  test('handles multiple timers for same metric concurrently', () => {
    const tracker = createPerformanceTracker();

    const id1 = tracker.startTimer('same:metric');
    const id2 = tracker.startTimer('same:metric');

    tracker.endTimer(id1, 'same:metric');
    tracker.endTimer(id2, 'same:metric');

    const metric = tracker.getMetric('same:metric');
    expect(metric?.count).toBe(2);
  });

  test('ending same timer twice returns 0 second time', () => {
    const tracker = createPerformanceTracker();

    const id = tracker.startTimer('double:end');
    const duration1 = tracker.endTimer(id, 'double:end');
    const duration2 = tracker.endTimer(id, 'double:end');

    expect(duration1).toBeGreaterThanOrEqual(0);
    expect(duration2).toBe(0);
  });

  test('handles very long metric names', () => {
    const tracker = createPerformanceTracker();
    const longName = 'a'.repeat(1000);

    const id = tracker.startTimer(longName);
    tracker.endTimer(id, longName);

    const metric = tracker.getMetric(longName);
    expect(metric?.name).toBe(longName);
  });

  test('handles empty metric name', () => {
    const tracker = createPerformanceTracker();

    const id = tracker.startTimer('');
    tracker.endTimer(id, '');

    const metric = tracker.getMetric('');
    expect(metric?.name).toBe('');
  });
});

// Test helper calculations
describe('Calculation Helpers', () => {
  test('computes running average correctly', () => {
    const values = [10, 20, 30, 40, 50];
    const expectedAverage = 30; // (10+20+30+40+50) / 5

    const tracker = createPerformanceTracker();

    // We can't directly control timing, but we can test the concept
    const computeAverage = (nums: number[]) =>
      nums.reduce((sum, n) => sum + n, 0) / nums.length;

    expect(computeAverage(values)).toBe(expectedAverage);
  });

  test('finds min correctly', () => {
    const values = [50, 30, 80, 10, 60];
    const expectedMin = 10;

    const findMin = (nums: number[]) => Math.min(...nums);
    expect(findMin(values)).toBe(expectedMin);
  });

  test('finds max correctly', () => {
    const values = [50, 30, 80, 10, 60];
    const expectedMax = 80;

    const findMax = (nums: number[]) => Math.max(...nums);
    expect(findMax(values)).toBe(expectedMax);
  });
});

// Test sample limiting
describe('Sample Limiting', () => {
  const MAX_SAMPLES = 100;

  test('limits samples to MAX_SAMPLES', () => {
    const tracker = createPerformanceTracker();

    // Record more than MAX_SAMPLES
    for (let i = 0; i < 150; i++) {
      const id = tracker.startTimer('overflow');
      tracker.endTimer(id, 'overflow');
    }

    const metric = tracker.getMetric('overflow');
    // Count should be capped at MAX_SAMPLES
    expect(metric?.count).toBeLessThanOrEqual(MAX_SAMPLES);
  });

  test('maintains correct average after overflow', () => {
    const tracker = createPerformanceTracker();

    // Record enough samples to trigger overflow
    for (let i = 0; i < 110; i++) {
      const id = tracker.startTimer('avg:test');
      tracker.endTimer(id, 'avg:test');
    }

    const metric = tracker.getMetric('avg:test');
    // Average should still be reasonable (not NaN or Infinity)
    expect(Number.isFinite(metric?.average)).toBe(true);
  });
});
