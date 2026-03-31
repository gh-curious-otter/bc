/**
 * ConfigContext - Provides workspace configuration to TUI components
 * Issue #1004: Phase 5 - Performance configuration tunables
 * Issue #1022: Phase 3 - Advanced theming system
 *
 * Fetches configuration from `bc config show` commands and makes
 * it available to all hooks via React context.
 */

import React, { createContext, useContext, useState, useEffect, useCallback, useMemo } from 'react';
import type { PerformanceConfig, TUIConfig } from '../types';
import { execBcJson } from '../services/bc';

// Default performance config values (matches Go defaults)
const DEFAULT_PERFORMANCE_CONFIG: PerformanceConfig = {
  poll_interval_agents: 2000,
  poll_interval_channels: 3000,
  poll_interval_costs: 5000,
  poll_interval_status: 2000,
  poll_interval_logs: 3000,
  poll_interval_demons: 5000,
  poll_interval_dashboard: 30000, // Dashboard aggregates data, uses slower interval
  cache_ttl_tmux: 2000,
  cache_ttl_commands: 5000,
  adaptive_fast_interval: 1000,
  adaptive_normal_interval: 2000,
  adaptive_slow_interval: 4000,
  adaptive_max_interval: 8000,
};

// Default TUI config values (matches Go defaults)
const DEFAULT_TUI_CONFIG: TUIConfig = {
  theme: 'dark',
  mode: 'auto',
};

interface ConfigContextValue {
  /** Performance configuration from workspace config */
  performance: PerformanceConfig;
  /** TUI theme configuration from workspace config */
  tui: TUIConfig;
  /** Whether config is still loading */
  loading: boolean;
  /** Error if config fetch failed */
  error: string | null;
  /** Refresh config from workspace */
  refresh: () => Promise<void>;
}

const ConfigContext = createContext<ConfigContextValue | null>(null);

interface ConfigProviderProps {
  children: React.ReactNode;
}

/**
 * ConfigProvider - Provides workspace configuration context to children
 */
export function ConfigProvider({ children }: ConfigProviderProps): React.ReactElement {
  const [performance, setPerformance] = useState<PerformanceConfig>(DEFAULT_PERFORMANCE_CONFIG);
  const [tui, setTUI] = useState<TUIConfig>(DEFAULT_TUI_CONFIG);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    try {
      // Fetch performance section from workspace config
      const performanceResponse = await execBcJson<PerformanceConfig>([
        'config',
        'show',
        'performance',
      ]);

      // Merge with defaults to handle missing values
      setPerformance({
        ...DEFAULT_PERFORMANCE_CONFIG,
        ...performanceResponse,
      });

      // Fetch TUI theme configuration
      const tuiResponse = await execBcJson<TUIConfig>(['config', 'show', 'tui']);

      // Merge with defaults to handle missing values
      setTUI({
        ...DEFAULT_TUI_CONFIG,
        ...tuiResponse,
      });

      setError(null);
    } catch (err) {
      // On error, use defaults - config may not exist yet
      setPerformance(DEFAULT_PERFORMANCE_CONFIG);
      setTUI(DEFAULT_TUI_CONFIG);
      setError(err instanceof Error ? err.message : 'Failed to load config');
    } finally {
      setLoading(false);
    }
  }, []);

  // Fetch config on mount
  useEffect(() => {
    void fetchConfig();
  }, [fetchConfig]);

  const value = useMemo<ConfigContextValue>(
    () => ({
      performance,
      tui,
      loading,
      error,
      refresh: fetchConfig,
    }),
    [performance, tui, loading, error, fetchConfig]
  );

  return <ConfigContext.Provider value={value}>{children}</ConfigContext.Provider>;
}

/**
 * useConfig - Hook to access workspace configuration
 * @throws Error if used outside of ConfigProvider
 */
export function useConfig(): ConfigContextValue {
  const context = useContext(ConfigContext);
  if (!context) {
    throw new Error('useConfig must be used within a ConfigProvider');
  }
  return context;
}

/**
 * usePerformanceConfig - Hook to access performance configuration only
 * Returns the performance config object directly
 */
export function usePerformanceConfig(): PerformanceConfig {
  const { performance } = useConfig();
  return performance;
}

/**
 * useThemeConfig - Hook to access TUI theme configuration only
 * Returns the theme config object directly (theme name and mode)
 */
export function useThemeConfig(): TUIConfig {
  const { tui } = useConfig();
  return tui;
}

export { DEFAULT_PERFORMANCE_CONFIG, DEFAULT_TUI_CONFIG };
export default ConfigProvider;
