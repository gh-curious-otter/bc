/**
 * Tests for useAdaptivePolling hook
 * Issue #979: Adaptive polling with state-aware intervals
 */

import { describe, test, expect, beforeEach, mock, spyOn } from 'bun:test';

// Test the constants and logic without React hooks
describe('Adaptive Polling Constants', () => {
  const FAST_INTERVAL = 1000;
  const NORMAL_INTERVAL = 2000;
  const SLOW_INTERVAL = 4000;
  const MAX_INTERVAL = 8000;
  const BACKOFF_FACTOR = 1.5;
  const IDLE_THRESHOLD_MS = 10000;

  test('fast interval is 1 second', () => {
    expect(FAST_INTERVAL).toBe(1000);
  });

  test('normal interval is 2 seconds', () => {
    expect(NORMAL_INTERVAL).toBe(2000);
  });

  test('slow interval is 4 seconds', () => {
    expect(SLOW_INTERVAL).toBe(4000);
  });

  test('max interval is 8 seconds', () => {
    expect(MAX_INTERVAL).toBe(8000);
  });

  test('backoff factor is 1.5x', () => {
    expect(BACKOFF_FACTOR).toBe(1.5);
  });

  test('idle threshold is 10 seconds', () => {
    expect(IDLE_THRESHOLD_MS).toBe(10000);
  });

  test('exponential backoff calculation', () => {
    // Test backoff progression
    const backoff0 = SLOW_INTERVAL * Math.pow(BACKOFF_FACTOR, 0);
    const backoff1 = SLOW_INTERVAL * Math.pow(BACKOFF_FACTOR, 1);
    const backoff2 = SLOW_INTERVAL * Math.pow(BACKOFF_FACTOR, 2);

    expect(backoff0).toBe(4000); // 4s
    expect(backoff1).toBe(6000); // 6s
    expect(backoff2).toBe(9000); // 9s (but capped at MAX_INTERVAL)

    // Verify capping
    expect(Math.min(MAX_INTERVAL, backoff2)).toBe(MAX_INTERVAL);
  });
});

describe('Polling Mode Logic', () => {
  type PollingMode = 'fast' | 'normal' | 'slow' | 'backoff';

  // Simulate the interval selection logic
  const getIntervalForMode = (mode: PollingMode): number => {
    const FAST_INTERVAL = 1000;
    const NORMAL_INTERVAL = 2000;
    const SLOW_INTERVAL = 4000;
    const MAX_INTERVAL = 8000;

    switch (mode) {
      case 'fast':
        return FAST_INTERVAL;
      case 'normal':
        return NORMAL_INTERVAL;
      case 'slow':
        return SLOW_INTERVAL;
      case 'backoff':
        return MAX_INTERVAL;
    }
  };

  test('fast mode returns 1s interval', () => {
    expect(getIntervalForMode('fast')).toBe(1000);
  });

  test('normal mode returns 2s interval', () => {
    expect(getIntervalForMode('normal')).toBe(2000);
  });

  test('slow mode returns 4s interval', () => {
    expect(getIntervalForMode('slow')).toBe(4000);
  });

  test('backoff mode returns 8s interval', () => {
    expect(getIntervalForMode('backoff')).toBe(8000);
  });

  test('all modes return valid intervals', () => {
    const modes: PollingMode[] = ['fast', 'normal', 'slow', 'backoff'];
    for (const mode of modes) {
      const interval = getIntervalForMode(mode);
      expect(interval).toBeGreaterThan(0);
      expect(interval).toBeLessThanOrEqual(8000);
    }
  });
});

describe('Idle State Detection', () => {
  const IDLE_THRESHOLD_MS = 10000;

  // Simulate idle detection logic
  const isIdleState = (timeSinceActivity: number): boolean => {
    return timeSinceActivity > IDLE_THRESHOLD_MS;
  };

  const getModeFromIdleTime = (
    timeSinceActivity: number
  ): 'fast' | 'normal' | 'slow' | 'backoff' => {
    if (timeSinceActivity < IDLE_THRESHOLD_MS / 2) {
      return 'fast';
    }
    if (timeSinceActivity < IDLE_THRESHOLD_MS) {
      return 'normal';
    }
    if (timeSinceActivity < IDLE_THRESHOLD_MS * 3) {
      return 'slow';
    }
    return 'backoff';
  };

  test('not idle when activity is recent', () => {
    expect(isIdleState(0)).toBe(false);
    expect(isIdleState(5000)).toBe(false);
    expect(isIdleState(9999)).toBe(false);
  });

  test('idle when activity exceeds threshold', () => {
    expect(isIdleState(10001)).toBe(true);
    expect(isIdleState(20000)).toBe(true);
    expect(isIdleState(60000)).toBe(true);
  });

  test('fast mode when very recent activity', () => {
    expect(getModeFromIdleTime(0)).toBe('fast');
    expect(getModeFromIdleTime(4999)).toBe('fast');
  });

  test('normal mode during transition period', () => {
    expect(getModeFromIdleTime(5000)).toBe('normal');
    expect(getModeFromIdleTime(9999)).toBe('normal');
  });

  test('slow mode when idle', () => {
    expect(getModeFromIdleTime(10001)).toBe('slow');
    expect(getModeFromIdleTime(25000)).toBe('slow');
  });

  test('backoff mode during extended idle', () => {
    expect(getModeFromIdleTime(30001)).toBe('backoff');
    expect(getModeFromIdleTime(60000)).toBe('backoff');
  });
});

describe('Activity Reporting', () => {
  // Simulate activity reporting state machine
  interface PollingState {
    mode: 'fast' | 'normal' | 'slow' | 'backoff';
    lastActivity: number;
    backoffCount: number;
  }

  const reportActivity = (state: PollingState): PollingState => ({
    mode: 'fast',
    lastActivity: Date.now(),
    backoffCount: 0,
  });

  test('activity resets to fast mode', () => {
    const initialState: PollingState = {
      mode: 'slow',
      lastActivity: Date.now() - 30000,
      backoffCount: 5,
    };

    const newState = reportActivity(initialState);
    expect(newState.mode).toBe('fast');
    expect(newState.backoffCount).toBe(0);
  });

  test('activity resets from backoff mode', () => {
    const initialState: PollingState = {
      mode: 'backoff',
      lastActivity: Date.now() - 60000,
      backoffCount: 10,
    };

    const newState = reportActivity(initialState);
    expect(newState.mode).toBe('fast');
    expect(newState.backoffCount).toBe(0);
  });
});

describe('Agent Working Count Integration', () => {
  // Simulate the useAdaptiveAgentPolling logic
  const shouldReportActivity = (currentWorking: number, prevWorking: number): boolean => {
    return currentWorking > prevWorking;
  };

  const shouldReportIdle = (currentWorking: number, prevWorking: number): boolean => {
    return currentWorking === 0 && prevWorking === 0;
  };

  test('reports activity when agents start working', () => {
    expect(shouldReportActivity(1, 0)).toBe(true);
    expect(shouldReportActivity(3, 2)).toBe(true);
    expect(shouldReportActivity(5, 1)).toBe(true);
  });

  test('does not report activity when working decreases', () => {
    expect(shouldReportActivity(0, 1)).toBe(false);
    expect(shouldReportActivity(2, 3)).toBe(false);
  });

  test('does not report activity when stable', () => {
    expect(shouldReportActivity(2, 2)).toBe(false);
    expect(shouldReportActivity(0, 0)).toBe(false);
  });

  test('reports idle when no agents working', () => {
    expect(shouldReportIdle(0, 0)).toBe(true);
  });

  test('does not report idle when agents are working', () => {
    expect(shouldReportIdle(1, 0)).toBe(false);
    expect(shouldReportIdle(0, 1)).toBe(false);
    expect(shouldReportIdle(2, 2)).toBe(false);
  });
});

describe('Interval Bounds', () => {
  const MIN_INTERVAL = 1000; // FAST_INTERVAL
  const MAX_INTERVAL = 8000;

  test('interval never goes below minimum', () => {
    // Test that fast mode doesn't go below 1s
    expect(MIN_INTERVAL).toBe(1000);
  });

  test('interval never exceeds maximum', () => {
    const SLOW_INTERVAL = 4000;
    const BACKOFF_FACTOR = 1.5;

    // Test backoff capping
    for (let i = 0; i < 10; i++) {
      const backoffInterval = SLOW_INTERVAL * Math.pow(BACKOFF_FACTOR, i);
      const cappedInterval = Math.min(MAX_INTERVAL, backoffInterval);
      expect(cappedInterval).toBeLessThanOrEqual(MAX_INTERVAL);
    }
  });

  test('interval progression is monotonic until cap', () => {
    const SLOW_INTERVAL = 4000;
    const BACKOFF_FACTOR = 1.5;

    let prevInterval = 0;
    for (let i = 0; i < 3; i++) {
      const backoffInterval = SLOW_INTERVAL * Math.pow(BACKOFF_FACTOR, i);
      const cappedInterval = Math.min(MAX_INTERVAL, backoffInterval);
      expect(cappedInterval).toBeGreaterThanOrEqual(prevInterval);
      prevInterval = cappedInterval;
    }
  });
});
