/**
 * useAdaptivePolling - Adaptive polling with state-aware intervals
 * Issue #979: Optimize agent polling with adaptive intervals
 * Issue #1004: Performance configuration tunables (Phase 5)
 *
 * Reduces CPU usage by slowing down polling when agents are idle
 * and speeding up when activity is detected.
 *
 * Interval values are configurable via workspace config [performance] section.
 */

import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { usePerformanceConfig } from '../config';

const BACKOFF_FACTOR = 1.5; // Exponential backoff multiplier
const IDLE_THRESHOLD_MS = 10000; // 10s without changes = idle state

export type PollingMode = 'fast' | 'normal' | 'slow' | 'backoff';

export interface AdaptivePollingState {
  /** Current polling mode */
  mode: PollingMode;
  /** Current interval in ms */
  interval: number;
  /** Time since last activity */
  idleTime: number;
  /** Whether in idle state */
  isIdle: boolean;
}

export interface UseAdaptivePollingOptions {
  /** Initial polling interval (default: NORMAL_INTERVAL) */
  initialInterval?: number;
  /** Enable adaptive behavior (default: true) */
  adaptiveEnabled?: boolean;
  /** Enable polling (default: true) */
  enabled?: boolean;
  /** Callback when tick occurs */
  onTick?: () => void;
}

export interface UseAdaptivePollingResult {
  /** Current tick count */
  tick: number;
  /** Current adaptive state */
  state: AdaptivePollingState;
  /** Report activity (resets idle timer, speeds up polling) */
  reportActivity: () => void;
  /** Report idle state change */
  reportIdle: () => void;
  /** Pause polling */
  pause: () => void;
  /** Resume polling */
  resume: () => void;
  /** Whether polling is paused */
  isPaused: boolean;
  /** Force a specific mode temporarily */
  setMode: (mode: PollingMode) => void;
}

/**
 * Adaptive polling hook that adjusts intervals based on activity
 *
 * Modes:
 * - fast: configurable interval - when agents are actively working
 * - normal: configurable interval - default state
 * - slow: configurable interval - when agents have been idle for a while
 * - backoff: up to max interval - exponential backoff during extended quiet
 *
 * Intervals are configured via workspace [performance] section.
 */
export function useAdaptivePolling(
  options: UseAdaptivePollingOptions = {}
): UseAdaptivePollingResult {
  // Get configurable intervals from workspace config
  const perfConfig = usePerformanceConfig();
  const FAST_INTERVAL = perfConfig.adaptive_fast_interval;
  const NORMAL_INTERVAL = perfConfig.adaptive_normal_interval;
  const SLOW_INTERVAL = perfConfig.adaptive_slow_interval;
  const MAX_INTERVAL = perfConfig.adaptive_max_interval;

  const {
    initialInterval = NORMAL_INTERVAL,
    adaptiveEnabled = true,
    enabled = true,
    onTick,
  } = options;

  const [tick, setTick] = useState(0);
  const [mode, setMode] = useState<PollingMode>('normal');
  const [interval, setIntervalMs] = useState(initialInterval);
  const [isPaused, setIsPaused] = useState(!enabled);

  // Track activity timing
  const lastActivityRef = useRef<number>(Date.now());
  const backoffCountRef = useRef<number>(0);
  const idleCheckRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Calculate idle time
  const [idleTime, setIdleTime] = useState(0);
  const isIdle = idleTime > IDLE_THRESHOLD_MS;

  // Report activity - speeds up polling
  const reportActivity = useCallback(() => {
    if (!adaptiveEnabled) return;

    lastActivityRef.current = Date.now();
    backoffCountRef.current = 0;
    setIdleTime(0);

    // Immediately switch to fast mode
    setMode('fast');
    setIntervalMs(FAST_INTERVAL);
  }, [adaptiveEnabled, FAST_INTERVAL]);

  // Report idle - allows backoff
  const reportIdle = useCallback(() => {
    if (!adaptiveEnabled) return;

    // Check if we should transition to slower polling
    const timeSinceActivity = Date.now() - lastActivityRef.current;

    if (timeSinceActivity > IDLE_THRESHOLD_MS) {
      // In idle state - use slow or backoff mode
      if (mode !== 'backoff') {
        setMode('slow');
        setIntervalMs(SLOW_INTERVAL);
      }

      // Check for extended idle - apply backoff
      if (timeSinceActivity > IDLE_THRESHOLD_MS * 3) {
        const newInterval = Math.min(
          MAX_INTERVAL,
          SLOW_INTERVAL * Math.pow(BACKOFF_FACTOR, backoffCountRef.current)
        );
        setMode('backoff');
        setIntervalMs(newInterval);
        backoffCountRef.current += 1;
      }
    } else if (timeSinceActivity > IDLE_THRESHOLD_MS / 2) {
      // Transitioning to idle - use normal
      setMode('normal');
      setIntervalMs(NORMAL_INTERVAL);
    }
  }, [adaptiveEnabled, mode, MAX_INTERVAL, NORMAL_INTERVAL, SLOW_INTERVAL]);

  // Update idle time periodically
  useEffect(() => {
    if (!adaptiveEnabled) return;

    idleCheckRef.current = setInterval(() => {
      const timeSinceActivity = Date.now() - lastActivityRef.current;
      setIdleTime(timeSinceActivity);

      // Auto-trigger idle check
      if (timeSinceActivity > IDLE_THRESHOLD_MS / 2) {
        reportIdle();
      }
    }, 1000);

    return () => {
      if (idleCheckRef.current) {
        clearInterval(idleCheckRef.current);
      }
    };
  }, [adaptiveEnabled, reportIdle]);

  // Main polling interval
  useEffect(() => {
    if (isPaused) return;

    const timer = setInterval(() => {
      setTick((t) => t + 1);
      onTick?.();
    }, interval);

    return () => {
      clearInterval(timer);
    };
  }, [isPaused, interval, onTick]);

  const pause = useCallback(() => {
    setIsPaused(true);
  }, []);

  const resume = useCallback(() => {
    setIsPaused(false);
    // Reset to normal on resume
    setMode('normal');
    setIntervalMs(NORMAL_INTERVAL);
    lastActivityRef.current = Date.now();
    backoffCountRef.current = 0;
  }, [NORMAL_INTERVAL]);

  const setModeManual = useCallback(
    (newMode: PollingMode) => {
      setMode(newMode);
      switch (newMode) {
        case 'fast':
          setIntervalMs(FAST_INTERVAL);
          break;
        case 'normal':
          setIntervalMs(NORMAL_INTERVAL);
          break;
        case 'slow':
          setIntervalMs(SLOW_INTERVAL);
          break;
        case 'backoff':
          setIntervalMs(MAX_INTERVAL);
          break;
      }
    },
    [FAST_INTERVAL, MAX_INTERVAL, NORMAL_INTERVAL, SLOW_INTERVAL]
  );

  const state = useMemo<AdaptivePollingState>(
    () => ({
      mode,
      interval,
      idleTime,
      isIdle,
    }),
    [mode, interval, idleTime, isIdle]
  );

  return {
    tick,
    state,
    reportActivity,
    reportIdle,
    pause,
    resume,
    isPaused,
    setMode: setModeManual,
  };
}

/**
 * Hook to create adaptive polling integrated with agent state
 * Automatically detects activity based on agent state changes
 */
export function useAdaptiveAgentPolling(
  options: UseAdaptivePollingOptions & {
    /** Current agent counts */
    agentCounts?: { working: number; active: number };
  } = {}
): UseAdaptivePollingResult {
  const { agentCounts, ...pollingOptions } = options;
  const result = useAdaptivePolling(pollingOptions);
  const prevWorkingRef = useRef(agentCounts?.working ?? 0);

  // Automatically report activity when working agents increase
  useEffect(() => {
    const working = agentCounts?.working ?? 0;
    const prevWorking = prevWorkingRef.current;

    if (working > prevWorking) {
      // Agent started working - speed up
      result.reportActivity();
    } else if (working === 0 && prevWorking === 0) {
      // No working agents - report idle
      result.reportIdle();
    }

    prevWorkingRef.current = working;
  }, [agentCounts?.working, result]);

  return result;
}

export default useAdaptivePolling;
