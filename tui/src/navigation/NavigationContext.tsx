/**
 * Navigation Context - Global navigation state management
 */

import React, { createContext, useContext, useState, useCallback, useMemo } from 'react';
import type { ReactNode } from 'react';

// View types for navigation
export type View = 'dashboard' | 'agents' | 'channels' | 'costs' | 'help' | 'commands';

// Tab configuration
export interface TabConfig {
  key: string;
  view: View;
  label: string;
  shortcut?: string;
}

export const DEFAULT_TABS: TabConfig[] = [
  { key: '1', view: 'dashboard', label: 'Dashboard', shortcut: '1' },
  { key: '2', view: 'agents', label: 'Agents', shortcut: '2' },
  { key: '3', view: 'channels', label: 'Channels', shortcut: '3' },
  { key: '4', view: 'costs', label: 'Costs', shortcut: '4' },
  { key: '5', view: 'commands', label: 'Commands', shortcut: '5' },
  { key: '?', view: 'help', label: 'Help', shortcut: '?' },
];

// Navigation state
export interface NavigationState {
  currentView: View;
  previousView: View | null;
  history: View[];
  historyIndex: number;
}

// Navigation context value
export interface NavigationContextValue {
  // State
  currentView: View;
  previousView: View | null;
  tabs: TabConfig[];
  canGoBack: boolean;
  canGoForward: boolean;

  // Actions
  navigate: (view: View) => void;
  goBack: () => void;
  goForward: () => void;
  goHome: () => void;

  // Utilities
  isActive: (view: View) => boolean;
  getTabByKey: (key: string) => TabConfig | undefined;
  getTabByView: (view: View) => TabConfig | undefined;
}

const NavigationContext = createContext<NavigationContextValue | null>(null);

export interface NavigationProviderProps {
  children: ReactNode;
  initialView?: View;
  tabs?: TabConfig[];
}

export function NavigationProvider({
  children,
  initialView = 'dashboard',
  tabs = DEFAULT_TABS,
}: NavigationProviderProps): React.ReactElement {
  const [state, setState] = useState<NavigationState>({
    currentView: initialView,
    previousView: null,
    history: [initialView],
    historyIndex: 0,
  });

  const navigate = useCallback((view: View) => {
    setState((prev) => {
      // Don't navigate to the same view
      if (prev.currentView === view) return prev;

      // Truncate forward history when navigating
      const newHistory = [...prev.history.slice(0, prev.historyIndex + 1), view];

      return {
        currentView: view,
        previousView: prev.currentView,
        history: newHistory,
        historyIndex: newHistory.length - 1,
      };
    });
  }, []);

  const goBack = useCallback(() => {
    setState((prev) => {
      if (prev.historyIndex <= 0) return prev;

      const newIndex = prev.historyIndex - 1;
      const newView = prev.history[newIndex];
      if (!newView) return prev;

      return {
        ...prev,
        currentView: newView,
        previousView: prev.currentView,
        historyIndex: newIndex,
      };
    });
  }, []);

  const goForward = useCallback(() => {
    setState((prev) => {
      if (prev.historyIndex >= prev.history.length - 1) return prev;

      const newIndex = prev.historyIndex + 1;
      const newView = prev.history[newIndex];
      if (!newView) return prev;

      return {
        ...prev,
        currentView: newView,
        previousView: prev.currentView,
        historyIndex: newIndex,
      };
    });
  }, []);

  const goHome = useCallback(() => {
    navigate('dashboard');
  }, [navigate]);

  const isActive = useCallback(
    (view: View) => state.currentView === view,
    [state.currentView]
  );

  const getTabByKey = useCallback(
    (key: string) => tabs.find((t) => t.key === key),
    [tabs]
  );

  const getTabByView = useCallback(
    (view: View) => tabs.find((t) => t.view === view),
    [tabs]
  );

  const value = useMemo<NavigationContextValue>(
    () => ({
      currentView: state.currentView,
      previousView: state.previousView,
      tabs,
      canGoBack: state.historyIndex > 0,
      canGoForward: state.historyIndex < state.history.length - 1,
      navigate,
      goBack,
      goForward,
      goHome,
      isActive,
      getTabByKey,
      getTabByView,
    }),
    [state, tabs, navigate, goBack, goForward, goHome, isActive, getTabByKey, getTabByView]
  );

  return (
    <NavigationContext.Provider value={value}>{children}</NavigationContext.Provider>
  );
}

/**
 * Hook to access navigation context
 * @throws Error if used outside NavigationProvider
 */
export function useNavigation(): NavigationContextValue {
  const context = useContext(NavigationContext);
  if (!context) {
    throw new Error('useNavigation must be used within a NavigationProvider');
  }
  return context;
}

/**
 * Hook to check if a specific view is active
 */
export function useIsActiveView(view: View): boolean {
  const { isActive } = useNavigation();
  return isActive(view);
}
