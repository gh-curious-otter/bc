/**
 * Navigation Context - Global navigation state management
 */

import React, { createContext, useContext, useState, useCallback, useMemo } from 'react';
import type { ReactNode } from 'react';

// View types for navigation
export type View = 'dashboard' | 'agents' | 'channels' | 'files' | 'costs' | 'help' | 'commands' | 'roles' | 'logs' | 'worktrees' | 'workspaces' | 'demons' | 'processes' | 'memory' | 'routing';

// Tab configuration
export interface TabConfig {
  key: string;
  view: View;
  label: string;
  /** Short label for narrow terminals (80-119 cols) */
  shortLabel?: string;
  shortcut?: string;
}

// Tab order matches DRAWER_SECTIONS visual grouping (#1526)
// WORKSPACE: dashboard, agents, channels, files, commands
// MONITOR: logs, costs, processes, demons
// SYSTEM: roles, worktrees, workspaces, memory, routing
export const DEFAULT_TABS: TabConfig[] = [
  // WORKSPACE section
  { key: '1', view: 'dashboard', label: 'Dashboard', shortLabel: 'Dash', shortcut: '1' },
  { key: '2', view: 'agents', label: 'Agents', shortLabel: 'Agt', shortcut: '2' },
  { key: '3', view: 'channels', label: 'Channels', shortLabel: 'Chan', shortcut: '3' },
  { key: '4', view: 'files', label: 'Files', shortLabel: 'File', shortcut: '4' },
  { key: '5', view: 'commands', label: 'Commands', shortLabel: 'Cmd', shortcut: '5' },
  // MONITOR section
  { key: '6', view: 'logs', label: 'Logs', shortLabel: 'Log', shortcut: '6' },
  { key: '7', view: 'costs', label: 'Costs', shortLabel: 'Cost', shortcut: '7' },
  { key: '8', view: 'processes', label: 'Processes', shortLabel: 'Proc', shortcut: '8' },
  { key: '9', view: 'demons', label: 'Demons', shortLabel: 'Dmn', shortcut: '9' },
  // SYSTEM section
  { key: '0', view: 'roles', label: 'Roles', shortLabel: 'Role', shortcut: '0' },
  { key: '-', view: 'worktrees', label: 'Worktrees', shortLabel: 'Tree', shortcut: '-' },
  { key: '=', view: 'workspaces', label: 'Workspaces', shortLabel: 'Wksp', shortcut: '=' },
  { key: 'M', view: 'memory', label: 'Memory', shortLabel: 'Mem', shortcut: 'M' },
  { key: 'R', view: 'routing', label: 'Routing', shortLabel: 'Rte', shortcut: 'R' },
  { key: '?', view: 'help', label: 'Help', shortLabel: '?', shortcut: '?' },
];

// Breadcrumb item for showing navigation path
export interface BreadcrumbItem {
  label: string;
  view?: View;
}

// Navigation state
export interface NavigationState {
  currentView: View;
  previousView: View | null;
  history: View[];
  historyIndex: number;
  breadcrumbs: BreadcrumbItem[];
}

// Navigation context value
export interface NavigationContextValue {
  // State
  currentView: View;
  previousView: View | null;
  tabs: TabConfig[];
  canGoBack: boolean;
  canGoForward: boolean;
  breadcrumbs: BreadcrumbItem[];

  // Actions
  navigate: (view: View) => void;
  goBack: () => void;
  goForward: () => void;
  goHome: () => void;
  nextTab: () => void;
  prevTab: () => void;
  setBreadcrumbs: (items: BreadcrumbItem[]) => void;
  clearBreadcrumbs: () => void;

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
    breadcrumbs: [],
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
        breadcrumbs: [], // Clear breadcrumbs on navigation
      };
    });
  }, []);

  const goBack = useCallback(() => {
    setState((prev) => {
      if (prev.historyIndex <= 0) return prev;

      const newIndex = prev.historyIndex - 1;

      return {
        ...prev,
        currentView: prev.history[newIndex],
        previousView: prev.currentView,
        historyIndex: newIndex,
      };
    });
  }, []);

  const goForward = useCallback(() => {
    setState((prev) => {
      if (prev.historyIndex >= prev.history.length - 1) return prev;

      const newIndex = prev.historyIndex + 1;

      return {
        ...prev,
        currentView: prev.history[newIndex],
        previousView: prev.currentView,
        historyIndex: newIndex,
      };
    });
  }, []);

  const goHome = useCallback(() => {
    navigate('dashboard');
  }, [navigate]);

  // Get main tabs (exclude help '?')
  const mainTabs = useMemo(() => tabs.filter(t => t.key !== '?'), [tabs]);

  const nextTab = useCallback(() => {
    const currentIndex = mainTabs.findIndex(t => t.view === state.currentView);
    if (currentIndex === -1) {
      // If not on a main tab, go to first
      navigate(mainTabs[0]?.view ?? 'dashboard');
    } else {
      // Wrap around to first tab after last
      const nextIndex = (currentIndex + 1) % mainTabs.length;
      navigate(mainTabs[nextIndex]?.view ?? 'dashboard');
    }
  }, [mainTabs, state.currentView, navigate]);

  const prevTab = useCallback(() => {
    const currentIndex = mainTabs.findIndex(t => t.view === state.currentView);
    if (currentIndex === -1) {
      // If not on a main tab, go to last
      navigate(mainTabs[mainTabs.length - 1]?.view ?? 'dashboard');
    } else {
      // Wrap around to last tab before first
      const prevIndex = (currentIndex - 1 + mainTabs.length) % mainTabs.length;
      navigate(mainTabs[prevIndex]?.view ?? 'dashboard');
    }
  }, [mainTabs, state.currentView, navigate]);

  const setBreadcrumbs = useCallback((items: BreadcrumbItem[]) => {
    setState((prev) => ({ ...prev, breadcrumbs: items }));
  }, []);

  const clearBreadcrumbs = useCallback(() => {
    setState((prev) => ({ ...prev, breadcrumbs: [] }));
  }, []);

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
      breadcrumbs: state.breadcrumbs,
      navigate,
      goBack,
      goForward,
      goHome,
      nextTab,
      prevTab,
      setBreadcrumbs,
      clearBreadcrumbs,
      isActive,
      getTabByKey,
      getTabByView,
    }),
    [state, tabs, navigate, goBack, goForward, goHome, nextTab, prevTab, setBreadcrumbs, clearBreadcrumbs, isActive, getTabByKey, getTabByView]
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
