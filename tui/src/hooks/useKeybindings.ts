/**
 * useKeybindings - 3-tier keybinding system
 *
 * Provides a hierarchical keybinding architecture:
 * - Tier 1: Global (always active) - view switching, quit, help
 * - Tier 2: View-local (when in specific view) - navigation, refresh
 * - Tier 3: Context (modal/input mode) - cancel, confirm
 *
 * Issue #1327: TUI Keybinding System
 */

import { useMemo } from 'react';
import type { View } from '../navigation';

/** Keybinding definition */
export interface Keybinding {
  /** Key(s) that trigger this binding */
  keys: string | string[];
  /** Human-readable description */
  description: string;
  /** Action to perform */
  action: () => void;
  /** Show in status bar hints */
  showInHints?: boolean;
  /** Hint label (short form for status bar) */
  hintLabel?: string;
}

/** Tier 1: Global keybindings (always active unless in input mode) */
export interface GlobalBindings {
  /** Number keys 1-9, 0 for view switching */
  viewSwitch: Record<string, View>;
  /** Uppercase letters for direct view access */
  viewShortcuts: Record<string, View>;
  /** Tab/Shift+Tab for view cycling */
  tabNavigation: boolean;
  /** ? for help overlay */
  helpToggle: boolean;
  /** q for quit */
  quit: boolean;
  /** Ctrl+R for global refresh */
  globalRefresh: boolean;
  /** ESC for back/close */
  escape: boolean;
}

/** Tier 2: View-local keybindings */
export interface ViewBindings {
  /** j/k for up/down navigation */
  listNavigation: boolean;
  /** g/G for top/bottom */
  jumpNavigation: boolean;
  /** Enter for select */
  select: boolean;
  /** r for view-local refresh */
  localRefresh: boolean;
  /** Custom view-specific bindings */
  custom: Keybinding[];
}

/** Tier 3: Context keybindings (modal/input mode) */
export interface ContextBindings {
  /** ESC to cancel/close */
  cancel: boolean;
  /** Enter to confirm */
  confirm: boolean;
  /** Custom context bindings */
  custom: Keybinding[];
}

/** Full keybinding configuration */
export interface KeybindingConfig {
  global: Partial<GlobalBindings>;
  view: Partial<ViewBindings>;
  context: Partial<ContextBindings>;
}

/** Default global view shortcuts (uppercase = view navigation) */
export const DEFAULT_VIEW_SHORTCUTS: Record<string, View> = {
  'M': 'memory',
  'R': 'routing',
};

/** Default number key mappings */
export const DEFAULT_VIEW_NUMBERS: Record<string, View> = {
  '1': 'dashboard',
  '2': 'agents',
  '3': 'channels',
  '4': 'costs',
  '5': 'commands',
  '6': 'roles',
  '7': 'logs',
  '8': 'worktrees',
  '9': 'workspaces',
  '0': 'demons',
  '-': 'processes',
};

/** Status bar hint for a keybinding */
export interface KeyHint {
  key: string;
  label: string;
  priority: number; // Lower = show first
}

/** Get status bar hints for current context */
export function getStatusBarHints(
  view: View,
  context: 'normal' | 'input' | 'modal' = 'normal',
  customHints: KeyHint[] = []
): KeyHint[] {
  const hints: KeyHint[] = [];

  if (context === 'input') {
    // Input mode hints
    hints.push(
      { key: 'Enter', label: 'send', priority: 1 },
      { key: 'Esc', label: 'cancel', priority: 2 },
    );
  } else if (context === 'modal') {
    // Modal hints
    hints.push(
      { key: 'Enter', label: 'confirm', priority: 1 },
      { key: 'Esc', label: 'close', priority: 2 },
    );
  } else {
    // Normal mode - view-specific hints
    switch (view) {
      case 'dashboard':
        hints.push(
          { key: 'r', label: 'refresh', priority: 1 },
          { key: 'j/k', label: 'navigate', priority: 2 },
        );
        break;
      case 'agents':
        hints.push(
          { key: 'Enter', label: 'attach', priority: 1 },
          { key: 'p', label: 'peek', priority: 2 },
          { key: 'x', label: 'stop', priority: 3 },
          { key: 'j/k', label: 'navigate', priority: 4 },
        );
        break;
      case 'channels':
        hints.push(
          { key: 'm', label: 'compose', priority: 1 },
          { key: 'Enter', label: 'view', priority: 2 },
          { key: 'j/k', label: 'navigate', priority: 3 },
        );
        break;
      case 'costs':
        hints.push(
          { key: '1/2/3', label: 'tabs', priority: 1 },
          { key: 'b', label: 'budget', priority: 2 },
          { key: 'e', label: 'export', priority: 3 },
        );
        break;
      case 'memory':
        hints.push(
          { key: '/', label: 'search', priority: 1 },
          { key: 'Enter', label: 'view', priority: 2 },
          { key: 'j/k', label: 'navigate', priority: 3 },
        );
        break;
      case 'routing':
        hints.push(
          { key: 'Enter', label: 'details', priority: 1 },
          { key: 'j/k', label: 'navigate', priority: 2 },
        );
        break;
      default:
        hints.push(
          { key: 'j/k', label: 'navigate', priority: 1 },
          { key: 'Enter', label: 'select', priority: 2 },
        );
    }

    // Add global hints at end
    // #1462: Show 'back' for non-dashboard views, 'quit' only on dashboard
    hints.push(
      { key: '?', label: 'help', priority: 10 },
      { key: 'q', label: view === 'dashboard' ? 'quit' : 'back', priority: 11 },
    );
  }

  // Add custom hints and sort by priority
  return [...hints, ...customHints].sort((a, b) => a.priority - b.priority);
}

/** Format hints for display in status bar */
export function formatHintsForStatusBar(hints: KeyHint[], maxWidth = 80): string {
  const parts: string[] = [];
  let currentWidth = 0;

  for (const hint of hints) {
    const part = `[${hint.key}] ${hint.label}`;
    const partWidth = part.length + 2; // +2 for spacing

    if (currentWidth + partWidth > maxWidth) {
      break; // Don't overflow
    }

    parts.push(part);
    currentWidth += partWidth;
  }

  return parts.join('  ');
}

/**
 * Hook to get keybinding hints for current view
 */
export function useKeybindingHints(
  view: View,
  context: 'normal' | 'input' | 'modal' = 'normal',
  customHints: KeyHint[] = []
): {
  hints: KeyHint[];
  formatted: string;
} {
  const hints = useMemo(
    () => getStatusBarHints(view, context, customHints),
    [view, context, customHints]
  );

  const formatted = useMemo(
    () => formatHintsForStatusBar(hints),
    [hints]
  );

  return { hints, formatted };
}

/**
 * Check if a key matches a keybinding
 */
export function matchesKey(input: string, binding: string | string[]): boolean {
  if (Array.isArray(binding)) {
    return binding.includes(input);
  }
  return input === binding;
}

/**
 * Get the view for a key press (number or uppercase letter)
 */
export function getViewForKey(key: string): View | undefined {
  // Check number keys first
  if (key in DEFAULT_VIEW_NUMBERS) {
    return DEFAULT_VIEW_NUMBERS[key];
  }
  // Check uppercase shortcuts
  if (key in DEFAULT_VIEW_SHORTCUTS) {
    return DEFAULT_VIEW_SHORTCUTS[key];
  }
  // Check ? for help
  if (key === '?') {
    return 'help';
  }
  return undefined;
}

export default useKeybindingHints;
