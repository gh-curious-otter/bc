/**
 * ConfigContext - Provides workspace performance configuration to TUI components
 * Issue #1004: Phase 5 - Performance configuration tunables
 *
 * Fetches performance config from `bc config show performance --json` and makes
 * it available to all hooks via React context.
 */

import React, { createContext, useContext, useState, useEffect } from 'react';
import type { PerformanceConfig } from '../types';
import { execBcJson } from '../services/bc';

// Default performance config values (matches Go defaults)
const DEFAULT_PERFORMANCE_CONFIG: PerformanceConfig = {
  poll_interval_agents: 2000,
  poll_interval_channels: 3000,
  poll_interval_costs: 5000,
  poll_interval_status: 2000,
  poll_interval_logs: 3000,
  poll_interval_teams: 10000,
  poll_interval_demons: 5000,
  cache_ttl_tmux: 2000,
  cache_ttl_commands: 5000,
  adaptive_fast_interval: 1000,
  adaptive_normal_interval: 2000,
  adaptive_slow_interval: 4000,
  adaptive_max_interval: 8000,
};

interface ConfigContextValue {
  /** Performance configuration from workspace config */
  performance: PerformanceConfig;
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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = async () => {
    try {
      // Fetch performance section from workspace config
      const response = await execBcJson<PerformanceConfig>(['config', 'show', 'performance']);

      // Merge with defaults to handle missing values
      setPerformance({
        ...DEFAULT_PERFORMANCE_CONFIG,
        ...response,
      });
      setError(null);
    } catch (err) {
      // On error, use defaults - config may not exist yet
      setPerformance(DEFAULT_PERFORMANCE_CONFIG);
      setError(err instanceof Error ? err.message : 'Failed to load config');
    } finally {
      setLoading(false);
    }
  };

  // Fetch config on mount
  useEffect(() => {
    void fetchConfig();
  }, []);

  const value: ConfigContextValue = {
    performance,
    loading,
    error,
    refresh: fetchConfig,
  };

  return (
    <ConfigContext.Provider value={value}>
      {children}
    </ConfigContext.Provider>
  );
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

export { DEFAULT_PERFORMANCE_CONFIG };
export default ConfigProvider;
