/**
 * PerformanceContext - Global performance metrics provider
 * Issue #965: Provide performance metrics across TUI components
 */

import React, { createContext, useContext, useMemo, type ReactNode } from 'react';
import { usePerformanceMetrics, type PerformanceMetric } from './usePerformanceMetrics';

interface PerformanceContextValue {
  /** All tracked metrics */
  metrics: Map<string, PerformanceMetric>;
  /** Total number of measurements */
  totalMeasurements: number;
  /** Whether debug mode is enabled */
  debugEnabled: boolean;
  /** Start timing an operation */
  startTimer: (name: string) => string;
  /** End timing an operation */
  endTimer: (timerId: string, name: string) => number;
  /** Record a metric value directly */
  recordMetric: (name: string, value: number) => void;
  /** Time an async operation */
  timeAsync: <T>(name: string, fn: () => Promise<T>) => Promise<T>;
  /** Time a sync operation */
  timeSync: <T>(name: string, fn: () => T) => T;
  /** Get a specific metric */
  getMetric: (name: string) => PerformanceMetric | undefined;
  /** Get all metrics as array */
  getAllMetrics: () => PerformanceMetric[];
  /** Get uptime in seconds */
  getUptime: () => number;
  /** Clear all metrics */
  clearMetrics: () => void;
  /** Toggle debug mode */
  toggleDebug: () => void;
}

const PerformanceContext = createContext<PerformanceContextValue | null>(null);

interface PerformanceProviderProps {
  children: ReactNode;
}

/**
 * Performance metrics provider component
 * Wrap your app with this to enable performance tracking
 */
export function PerformanceProvider({ children }: PerformanceProviderProps): React.ReactElement {
  const perf = usePerformanceMetrics();

  const value = useMemo<PerformanceContextValue>(() => ({
    metrics: perf.metrics,
    totalMeasurements: perf.totalMeasurements,
    debugEnabled: perf.debugEnabled,
    startTimer: perf.startTimer,
    endTimer: perf.endTimer,
    recordMetric: perf.recordMetric,
    timeAsync: perf.timeAsync,
    timeSync: perf.timeSync,
    getMetric: perf.getMetric,
    getAllMetrics: perf.getAllMetrics,
    getUptime: perf.getUptime,
    clearMetrics: perf.clearMetrics,
    toggleDebug: perf.toggleDebug,
  }), [perf]);

  return (
    <PerformanceContext.Provider value={value}>
      {children}
    </PerformanceContext.Provider>
  );
}

/**
 * Hook to access performance metrics context
 * Must be used within PerformanceProvider
 */
export function usePerformance(): PerformanceContextValue {
  const context = useContext(PerformanceContext);
  if (!context) {
    throw new Error('usePerformance must be used within PerformanceProvider');
  }
  return context;
}

/**
 * Hook to conditionally access performance metrics
 * Returns null if not within PerformanceProvider (safe for optional usage)
 */
export function usePerformanceOptional(): PerformanceContextValue | null {
  return useContext(PerformanceContext);
}
