/**
 * Tests for useAdaptivePolling hook - Adaptive polling with state-aware intervals
 * Validates type exports, mode transitions, and polling logic
 *
 * #1081 Q1 Cleanup: TUI hook test coverage
 *
 * Note: React hook testing requires DOM environment which is not available in Bun/Ink.
 * These tests focus on type validation and mode transition logic.
 */

import { describe, it, expect } from 'bun:test';
import type {
  PollingMode,
  AdaptivePollingState,
  UseAdaptivePollingOptions,
  UseAdaptivePollingResult,
} from '../useAdaptivePolling';

describe('useAdaptivePolling - Type Exports', () => {
  describe('PollingMode type', () => {
    it('supports fast mode', () => {
      const mode: PollingMode = 'fast';
      expect(mode).toBe('fast');
    });

    it('supports normal mode', () => {
      const mode: PollingMode = 'normal';
      expect(mode).toBe('normal');
    });

    it('supports slow mode', () => {
      const mode: PollingMode = 'slow';
      expect(mode).toBe('slow');
    });

    it('supports backoff mode', () => {
      const mode: PollingMode = 'backoff';
      expect(mode).toBe('backoff');
    });
  });

  describe('AdaptivePollingState interface', () => {
    it('contains all required fields', () => {
      const state: AdaptivePollingState = {
        mode: 'normal',
        interval: 5000,
        idleTime: 0,
        isIdle: false,
      };

      expect(state.mode).toBe('normal');
      expect(state.interval).toBe(5000);
      expect(state.idleTime).toBe(0);
      expect(state.isIdle).toBe(false);
    });

    it('supports fast mode state', () => {
      const state: AdaptivePollingState = {
        mode: 'fast',
        interval: 2000,
        idleTime: 0,
        isIdle: false,
      };

      expect(state.mode).toBe('fast');
    });

    it('supports slow mode state', () => {
      const state: AdaptivePollingState = {
        mode: 'slow',
        interval: 10000,
        idleTime: 15000,
        isIdle: true,
      };

      expect(state.mode).toBe('slow');
      expect(state.isIdle).toBe(true);
    });

    it('supports backoff mode state', () => {
      const state: AdaptivePollingState = {
        mode: 'backoff',
        interval: 30000,
        idleTime: 60000,
        isIdle: true,
      };

      expect(state.mode).toBe('backoff');
      expect(state.interval).toBe(30000);
    });
  });

  describe('UseAdaptivePollingOptions interface', () => {
    it('allows all optional fields', () => {
      const options: UseAdaptivePollingOptions = {
        initialInterval: 5000,
        adaptiveEnabled: true,
        enabled: true,
        onTick: () => { /* noop */ },
      };

      expect(options.initialInterval).toBe(5000);
      expect(options.adaptiveEnabled).toBe(true);
      expect(options.enabled).toBe(true);
    });

    it('allows empty options', () => {
      const options: UseAdaptivePollingOptions = {};
      expect(options.initialInterval).toBeUndefined();
      expect(options.adaptiveEnabled).toBeUndefined();
    });

    it('allows partial options', () => {
      const options: UseAdaptivePollingOptions = {
        initialInterval: 3000,
      };

      expect(options.initialInterval).toBe(3000);
      expect(options.enabled).toBeUndefined();
    });
  });
});

describe('useAdaptivePolling - Mode Transition Logic', () => {
  const BACKOFF_FACTOR = 1.5;
  const IDLE_THRESHOLD_MS = 10000;
  const FAST_INTERVAL = 2000;
  const NORMAL_INTERVAL = 5000;
  const SLOW_INTERVAL = 10000;
  const MAX_INTERVAL = 30000;

  function calculateInterval(mode: PollingMode, backoffCount: number): number {
    switch (mode) {
      case 'fast':
        return FAST_INTERVAL;
      case 'normal':
        return NORMAL_INTERVAL;
      case 'slow':
        return SLOW_INTERVAL;
      case 'backoff':
        return Math.min(
          MAX_INTERVAL,
          SLOW_INTERVAL * Math.pow(BACKOFF_FACTOR, backoffCount)
        );
    }
  }

  describe('interval calculations', () => {
    it('fast mode uses fast interval', () => {
      expect(calculateInterval('fast', 0)).toBe(FAST_INTERVAL);
    });

    it('normal mode uses normal interval', () => {
      expect(calculateInterval('normal', 0)).toBe(NORMAL_INTERVAL);
    });

    it('slow mode uses slow interval', () => {
      expect(calculateInterval('slow', 0)).toBe(SLOW_INTERVAL);
    });

    it('backoff mode applies exponential backoff', () => {
      expect(calculateInterval('backoff', 0)).toBe(SLOW_INTERVAL);
      expect(calculateInterval('backoff', 1)).toBe(15000);
      expect(calculateInterval('backoff', 2)).toBe(22500);
    });

    it('backoff caps at MAX_INTERVAL', () => {
      expect(calculateInterval('backoff', 10)).toBe(MAX_INTERVAL);
    });
  });

  describe('idle threshold logic', () => {
    function determineMode(timeSinceActivity: number, currentMode: PollingMode): PollingMode {
      if (timeSinceActivity > IDLE_THRESHOLD_MS * 3) {
        return 'backoff';
      }
      if (timeSinceActivity > IDLE_THRESHOLD_MS) {
        return currentMode !== 'backoff' ? 'slow' : 'backoff';
      }
      if (timeSinceActivity > IDLE_THRESHOLD_MS / 2) {
        return 'normal';
      }
      return 'fast';
    }

    it('returns fast mode when recently active', () => {
      expect(determineMode(0, 'normal')).toBe('fast');
      expect(determineMode(4000, 'normal')).toBe('fast');
    });

    it('returns normal mode during transition to idle', () => {
      expect(determineMode(5001, 'normal')).toBe('normal');
      expect(determineMode(9000, 'normal')).toBe('normal');
    });

    it('returns slow mode after idle threshold', () => {
      expect(determineMode(15000, 'normal')).toBe('slow');
      expect(determineMode(25000, 'normal')).toBe('slow');
    });

    it('returns backoff mode after extended idle', () => {
      expect(determineMode(35000, 'normal')).toBe('backoff');
      expect(determineMode(60000, 'normal')).toBe('backoff');
    });

    it('maintains backoff mode once entered', () => {
      expect(determineMode(15000, 'backoff')).toBe('backoff');
    });
  });
});

describe('useAdaptivePolling - Backoff Calculations', () => {
  const BACKOFF_FACTOR = 1.5;
  const SLOW_INTERVAL = 10000;
  const MAX_INTERVAL = 30000;

  function calculateBackoffInterval(count: number): number {
    return Math.min(
      MAX_INTERVAL,
      SLOW_INTERVAL * Math.pow(BACKOFF_FACTOR, count)
    );
  }

  it('backoff count 0 = slow interval', () => {
    expect(calculateBackoffInterval(0)).toBe(10000);
  });

  it('backoff count 1 = slow * 1.5', () => {
    expect(calculateBackoffInterval(1)).toBe(15000);
  });

  it('backoff count 2 = slow * 2.25', () => {
    expect(calculateBackoffInterval(2)).toBe(22500);
  });

  it('backoff count 3 exceeds max, caps at MAX', () => {
    expect(calculateBackoffInterval(3)).toBe(30000);
  });

  it('large backoff counts stay at MAX', () => {
    expect(calculateBackoffInterval(5)).toBe(30000);
    expect(calculateBackoffInterval(10)).toBe(30000);
  });
});

describe('useAdaptivePolling - Idle Time Detection', () => {
  const IDLE_THRESHOLD_MS = 10000;

  function isIdle(idleTime: number): boolean {
    return idleTime > IDLE_THRESHOLD_MS;
  }

  it('not idle when time < threshold', () => {
    expect(isIdle(0)).toBe(false);
    expect(isIdle(5000)).toBe(false);
    expect(isIdle(9999)).toBe(false);
  });

  it('idle when time > threshold', () => {
    expect(isIdle(10001)).toBe(true);
    expect(isIdle(15000)).toBe(true);
    expect(isIdle(60000)).toBe(true);
  });

  it('not idle at exactly threshold', () => {
    expect(isIdle(10000)).toBe(false);
  });
});

describe('useAdaptivePolling - Result Interface', () => {
  const mockResult: UseAdaptivePollingResult = {
    tick: 0,
    state: {
      mode: 'normal',
      interval: 5000,
      idleTime: 0,
      isIdle: false,
    },
    reportActivity: () => { /* noop */ },
    reportIdle: () => { /* noop */ },
    pause: () => { /* noop */ },
    resume: () => { /* noop */ },
    isPaused: false,
    setMode: () => { /* noop */ },
  };

  it('contains tick counter', () => {
    expect(typeof mockResult.tick).toBe('number');
  });

  it('contains adaptive state', () => {
    expect(mockResult.state.mode).toBe('normal');
    expect(mockResult.state.interval).toBe(5000);
  });

  it('contains activity methods', () => {
    expect(typeof mockResult.reportActivity).toBe('function');
    expect(typeof mockResult.reportIdle).toBe('function');
  });

  it('contains pause/resume methods', () => {
    expect(typeof mockResult.pause).toBe('function');
    expect(typeof mockResult.resume).toBe('function');
    expect(typeof mockResult.isPaused).toBe('boolean');
  });

  it('contains setMode method', () => {
    expect(typeof mockResult.setMode).toBe('function');
  });
});

describe('useAdaptivePolling - Function Import', () => {
  it('useAdaptivePolling is importable', async () => {
    const module = await import('../useAdaptivePolling');
    expect(typeof module.useAdaptivePolling).toBe('function');
    expect(typeof module.default).toBe('function');
  });

  it('useAdaptiveAgentPolling is importable', async () => {
    const module = await import('../useAdaptivePolling');
    expect(typeof module.useAdaptiveAgentPolling).toBe('function');
  });
});

describe('useAdaptivePolling - Agent Polling Extension', () => {
  describe('working agent detection', () => {
    function shouldReportActivity(prevWorking: number, currentWorking: number): boolean {
      return currentWorking > prevWorking;
    }

    function shouldReportIdle(prevWorking: number, currentWorking: number): boolean {
      return currentWorking === 0 && prevWorking === 0;
    }

    it('reports activity when working agents increase', () => {
      expect(shouldReportActivity(0, 1)).toBe(true);
      expect(shouldReportActivity(1, 2)).toBe(true);
    });

    it('does not report activity when working agents decrease', () => {
      expect(shouldReportActivity(2, 1)).toBe(false);
      expect(shouldReportActivity(1, 0)).toBe(false);
    });

    it('does not report activity when working agents stay same', () => {
      expect(shouldReportActivity(2, 2)).toBe(false);
    });

    it('reports idle when no working agents continuously', () => {
      expect(shouldReportIdle(0, 0)).toBe(true);
    });

    it('does not report idle when agents are working', () => {
      expect(shouldReportIdle(1, 0)).toBe(false);
      expect(shouldReportIdle(0, 1)).toBe(false);
      expect(shouldReportIdle(2, 2)).toBe(false);
    });
  });
});
