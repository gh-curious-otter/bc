/**
 * usePerformanceMetrics - TUI Performance Metrics Hook
 * Issue #965: Track render cycles, command latency, and poll times
 */

import { useState, useCallback, useRef } from 'react';

export interface PerformanceMetric {
  /** Metric name */
  name: string;
  /** Latest value in ms */
  value: number;
  /** Average value in ms */
  average: number;
  /** Minimum value in ms */
  min: number;
  /** Maximum value in ms */
  max: number;
  /** Number of samples */
  count: number;
  /** Timestamp of last update */
  lastUpdated: Date;
}

export interface PerformanceMetrics {
  /** All tracked metrics */
  metrics: Map<string, PerformanceMetric>;
  /** Total number of measurements */
  totalMeasurements: number;
  /** Time since metrics started */
  uptime: number;
  /** Whether debug mode is enabled */
  debugEnabled: boolean;
}

interface MetricData {
  values: number[];
  min: number;
  max: number;
  sum: number;
  lastUpdated: Date;
}

const MAX_SAMPLES = 100; // Keep last 100 samples for average calculation

/**
 * Hook for tracking TUI performance metrics
 * Provides timing functions for render cycles, commands, and polls
 */
export function usePerformanceMetrics() {
  const [metrics, setMetrics] = useState<Map<string, PerformanceMetric>>(new Map());
  const [totalMeasurements, setTotalMeasurements] = useState(0);
  const [debugEnabled, setDebugEnabled] = useState(() => {
    // Check for debug mode via env var
    return process.env.BC_TUI_DEBUG === '1' || process.env.BC_TUI_DEBUG === 'true';
  });

  const startTimeRef = useRef(Date.now());
  const dataRef = useRef<Map<string, MetricData>>(new Map());
  const pendingTimersRef = useRef<Map<string, number>>(new Map());

  /**
   * Start timing an operation
   * @param name - Unique name for this metric (e.g., "poll:agents", "render:dashboard")
   * @returns Timer ID for use with endTimer
   */
  const startTimer = useCallback((name: string): string => {
    const timerId = `${name}-${String(Date.now())}-${Math.random().toString(36).slice(2, 9)}`;
    pendingTimersRef.current.set(timerId, performance.now());
    return timerId;
  }, []);

  /**
   * End timing an operation and record the metric
   * @param timerId - Timer ID from startTimer
   * @param name - Metric name (must match startTimer name for consistency)
   */
  const endTimer = useCallback((timerId: string, name: string): number => {
    const startTime = pendingTimersRef.current.get(timerId);
    if (startTime === undefined) {
      return 0;
    }

    const duration = performance.now() - startTime;
    pendingTimersRef.current.delete(timerId);

    // Update internal data
    let data = dataRef.current.get(name);
    if (!data) {
      data = { values: [], min: Infinity, max: -Infinity, sum: 0, lastUpdated: new Date() };
      dataRef.current.set(name, data);
    }

    // Add new value, maintain max samples
    data.values.push(duration);
    if (data.values.length > MAX_SAMPLES) {
      const removed = data.values.shift();
      if (removed !== undefined) {
        data.sum -= removed;
      }
    }
    data.sum += duration;
    data.min = Math.min(data.min, duration);
    data.max = Math.max(data.max, duration);
    data.lastUpdated = new Date();

    // Update metrics state
    const metric: PerformanceMetric = {
      name,
      value: duration,
      average: data.sum / data.values.length,
      min: data.min,
      max: data.max,
      count: data.values.length,
      lastUpdated: data.lastUpdated,
    };

    setMetrics((prev) => new Map(prev).set(name, metric));
    setTotalMeasurements((prev) => prev + 1);

    return duration;
  }, []);

  /**
   * Record a metric value directly (for external timing)
   * @param name - Metric name
   * @param value - Duration in ms
   */
  const recordMetric = useCallback((name: string, value: number): void => {
    let data = dataRef.current.get(name);
    if (!data) {
      data = { values: [], min: Infinity, max: -Infinity, sum: 0, lastUpdated: new Date() };
      dataRef.current.set(name, data);
    }

    data.values.push(value);
    if (data.values.length > MAX_SAMPLES) {
      const removed = data.values.shift();
      if (removed !== undefined) {
        data.sum -= removed;
      }
    }
    data.sum += value;
    data.min = Math.min(data.min, value);
    data.max = Math.max(data.max, value);
    data.lastUpdated = new Date();

    const metric: PerformanceMetric = {
      name,
      value,
      average: data.sum / data.values.length,
      min: data.min,
      max: data.max,
      count: data.values.length,
      lastUpdated: data.lastUpdated,
    };

    setMetrics((prev) => new Map(prev).set(name, metric));
    setTotalMeasurements((prev) => prev + 1);
  }, []);

  /**
   * Time an async operation
   * @param name - Metric name
   * @param fn - Async function to time
   * @returns Result of the function
   */
  const timeAsync = useCallback(async <T>(name: string, fn: () => Promise<T>): Promise<T> => {
    const timerId = startTimer(name);
    try {
      return await fn();
    } finally {
      endTimer(timerId, name);
    }
  }, [startTimer, endTimer]);

  /**
   * Time a sync operation
   * @param name - Metric name
   * @param fn - Sync function to time
   * @returns Result of the function
   */
  const timeSync = useCallback(<T>(name: string, fn: () => T): T => {
    const timerId = startTimer(name);
    try {
      return fn();
    } finally {
      endTimer(timerId, name);
    }
  }, [startTimer, endTimer]);

  /**
   * Get a specific metric by name
   */
  const getMetric = useCallback((name: string): PerformanceMetric | undefined => {
    return metrics.get(name);
  }, [metrics]);

  /**
   * Get all metrics as an array, sorted by name
   */
  const getAllMetrics = useCallback((): PerformanceMetric[] => {
    return Array.from(metrics.values()).sort((a, b) => a.name.localeCompare(b.name));
  }, [metrics]);

  /**
   * Clear all metrics
   */
  const clearMetrics = useCallback((): void => {
    setMetrics(new Map());
    setTotalMeasurements(0);
    dataRef.current.clear();
    startTimeRef.current = Date.now();
  }, []);

  /**
   * Toggle debug mode
   */
  const toggleDebug = useCallback((): void => {
    setDebugEnabled((prev) => !prev);
  }, []);

  /**
   * Get uptime in seconds
   */
  const getUptime = useCallback((): number => {
    return (Date.now() - startTimeRef.current) / 1000;
  }, []);

  return {
    // State
    metrics,
    totalMeasurements,
    debugEnabled,

    // Timing functions
    startTimer,
    endTimer,
    recordMetric,
    timeAsync,
    timeSync,

    // Accessors
    getMetric,
    getAllMetrics,
    getUptime,

    // Control
    clearMetrics,
    toggleDebug,
  };
}

/**
 * Create a global performance metrics instance for use outside React
 */
export function createPerformanceTracker() {
  const data = new Map<string, MetricData>();
  const pendingTimers = new Map<string, number>();

  return {
    startTimer(name: string): string {
      const timerId = `${name}-${String(Date.now())}-${Math.random().toString(36).slice(2, 9)}`;
      pendingTimers.set(timerId, performance.now());
      return timerId;
    },

    endTimer(timerId: string, name: string): number {
      const startTime = pendingTimers.get(timerId);
      if (startTime === undefined) return 0;

      const duration = performance.now() - startTime;
      pendingTimers.delete(timerId);

      let metricData = data.get(name);
      if (!metricData) {
        metricData = { values: [], min: Infinity, max: -Infinity, sum: 0, lastUpdated: new Date() };
        data.set(name, metricData);
      }

      metricData.values.push(duration);
      if (metricData.values.length > MAX_SAMPLES) {
        const removed = metricData.values.shift();
        if (removed !== undefined) {
          metricData.sum -= removed;
        }
      }
      metricData.sum += duration;
      metricData.min = Math.min(metricData.min, duration);
      metricData.max = Math.max(metricData.max, duration);
      metricData.lastUpdated = new Date();

      return duration;
    },

    getMetric(name: string): PerformanceMetric | null {
      const metricData = data.get(name);
      if (!metricData || metricData.values.length === 0) return null;

      return {
        name,
        value: metricData.values[metricData.values.length - 1],
        average: metricData.sum / metricData.values.length,
        min: metricData.min,
        max: metricData.max,
        count: metricData.values.length,
        lastUpdated: metricData.lastUpdated,
      };
    },

    getAllMetrics(): PerformanceMetric[] {
      const metrics: PerformanceMetric[] = [];
      for (const [name, metricData] of data) {
        if (metricData.values.length > 0) {
          metrics.push({
            name,
            value: metricData.values[metricData.values.length - 1],
            average: metricData.sum / metricData.values.length,
            min: metricData.min,
            max: metricData.max,
            count: metricData.values.length,
            lastUpdated: metricData.lastUpdated,
          });
        }
      }
      return metrics.sort((a, b) => a.name.localeCompare(b.name));
    },

    clear(): void {
      data.clear();
      pendingTimers.clear();
    },
  };
}

// Global tracker for use in service layer
export const globalPerformanceTracker = createPerformanceTracker();
