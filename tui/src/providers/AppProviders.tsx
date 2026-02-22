/**
 * AppProviders - Consolidated provider tree for the TUI
 * Issue #1608: Simplify context tree architecture
 *
 * Composes all application providers into a single component to:
 * - Reduce visual nesting complexity in app.tsx
 * - Make provider dependencies explicit
 * - Simplify testing setup
 */

import React from 'react';
import { ConfigProvider } from '../config';
import { ThemeProvider, type ThemeMode } from '../theme';
import { NavigationProvider, FocusProvider, type View } from '../navigation';
import { UnreadProvider, HintsProvider, DisableInputProvider } from '../hooks';

interface AppProvidersProps {
  children: React.ReactNode;
  /** Initial view for navigation */
  initialView?: View;
  /** Disable input handling (for testing) */
  disableInput?: boolean;
  /** Theme mode override */
  themeMode?: ThemeMode;
}

/**
 * RootProviders - Config and theme providers (no dependencies)
 * These form the foundation that other providers may depend on.
 */
function RootProviders({
  children,
  themeMode = 'auto',
}: {
  children: React.ReactNode;
  themeMode?: ThemeMode;
}): React.ReactElement {
  return (
    <ConfigProvider>
      <ThemeProvider config={{ mode: themeMode }}>
        {children}
      </ThemeProvider>
    </ConfigProvider>
  );
}

/**
 * NavigationProviders - Navigation and focus management
 * Handles view routing and keyboard focus areas.
 */
function NavigationProviders({
  children,
  initialView = 'dashboard',
}: {
  children: React.ReactNode;
  initialView?: View;
}): React.ReactElement {
  return (
    <NavigationProvider initialView={initialView}>
      <FocusProvider>
        {children}
      </FocusProvider>
    </NavigationProvider>
  );
}

/**
 * UIStateProviders - UI state management
 * Handles unread counts, keyboard hints, and input disable state.
 */
function UIStateProviders({
  children,
  disableInput = false,
}: {
  children: React.ReactNode;
  disableInput?: boolean;
}): React.ReactElement {
  return (
    <UnreadProvider>
      <HintsProvider>
        <DisableInputProvider disabled={disableInput}>
          {children}
        </DisableInputProvider>
      </HintsProvider>
    </UnreadProvider>
  );
}

/**
 * AppProviders - Composes all providers into a single component
 *
 * Provider hierarchy:
 * 1. RootProviders (Config + Theme)
 * 2. NavigationProviders (Navigation + Focus)
 * 3. UIStateProviders (Unread + Hints + DisableInput)
 *
 * @example
 * ```tsx
 * <AppProviders initialView="dashboard" disableInput={false}>
 *   <AppContent />
 * </AppProviders>
 * ```
 */
export function AppProviders({
  children,
  initialView = 'dashboard',
  disableInput = false,
  themeMode = 'auto',
}: AppProvidersProps): React.ReactElement {
  return (
    <RootProviders themeMode={themeMode}>
      <NavigationProviders initialView={initialView}>
        <UIStateProviders disableInput={disableInput}>
          {children}
        </UIStateProviders>
      </NavigationProviders>
    </RootProviders>
  );
}

// Export individual provider groups for testing
export { RootProviders, NavigationProviders, UIStateProviders };

export default AppProviders;
