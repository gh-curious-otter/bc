/**
 * RootProvider - Combined root-level context provider
 *
 * Issue #1608: Simplify context tree architecture
 *
 * Consolidates related providers:
 * - ConfigProvider: Workspace configuration
 * - ThemeProvider: Theme styling
 *
 * This reduces provider nesting depth and groups related contexts.
 */

import React, { useMemo, type ReactNode } from 'react';
import { ConfigProvider, useThemeConfig } from '../config';
import { ThemeProvider, type ThemeMode } from '../theme';

export interface RootProviderProps {
  children: ReactNode;
}

/**
 * Inner component that has access to ConfigContext
 * Configures ThemeProvider based on workspace config
 */
function RootProviderInner({ children }: RootProviderProps): React.ReactElement {
  const themeConfig = useThemeConfig();

  // Convert config theme/mode to ThemeMode for ThemeProvider
  const effectiveMode: ThemeMode =
    themeConfig.mode === 'auto'
      ? 'dark' // Default to dark in auto mode (terminal UIs typically dark)
      : themeConfig.mode;

  const themeProviderConfig = useMemo(() => ({ mode: effectiveMode }), [effectiveMode]);

  return (
    <ThemeProvider config={themeProviderConfig}>
      {children}
    </ThemeProvider>
  );
}

/**
 * RootProvider - Root-level provider combining Config + Theme
 *
 * Usage:
 * ```tsx
 * <RootProvider>
 *   <NavigationProvider>
 *     <App />
 *   </NavigationProvider>
 * </RootProvider>
 * ```
 */
export function RootProvider({ children }: RootProviderProps): React.ReactElement {
  return (
    <ConfigProvider>
      <RootProviderInner>
        {children}
      </RootProviderInner>
    </ConfigProvider>
  );
}

export default RootProvider;
