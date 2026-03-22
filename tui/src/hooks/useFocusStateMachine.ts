/**
 * useFocusStateMachine - Centralized focus state management
 *
 * Issue #1825: Refactor focus management into state machine
 *
 * Problems solved:
 * - ESC goes to wrong view (returnFocus race conditions)
 * - Keybinds trigger while typing in input
 * - Focus stuck after exiting composition
 * - j/k navigation state unclear
 *
 * States:
 * - main: Global navigation enabled (q, Tab, j/k list nav)
 * - input: Only ESC and text input allowed (search, compose)
 * - detail: Nested view, ESC returns to parent (drill-down views)
 * - modal: Overlay, ESC closes modal
 */

import { useState, useCallback, useMemo } from 'react';

/** Focus states */
export type FocusState = 'main' | 'input' | 'detail' | 'modal';

/** State transition event */
export type FocusTransition =
  | 'ENTER_INPUT' // User starts typing (search, compose)
  | 'EXIT_INPUT' // User finishes typing (Enter, ESC)
  | 'OPEN_DETAIL' // User drills into detail view
  | 'CLOSE_DETAIL' // User exits detail view (ESC)
  | 'OPEN_MODAL' // Modal opens
  | 'CLOSE_MODAL' // Modal closes (ESC, action)
  | 'GO_HOME'; // Return to main (e.g., breadcrumb home)

/** Key categories for canHandle checks */
export type KeyCategory =
  | 'global_nav' // Tab, ?, M, I (view switching)
  | 'global_quit' // q (quit application)
  | 'list_nav' // j, k, g, G (list navigation)
  | 'selection' // Enter (select/confirm)
  | 'escape' // ESC (back/cancel/close)
  | 'text_input' // Any text character
  | 'refresh'; // r, Ctrl+R (refresh)

/** Valid state transitions */
const TRANSITIONS: Record<FocusState, Partial<Record<FocusTransition, FocusState>>> = {
  main: {
    ENTER_INPUT: 'input',
    OPEN_DETAIL: 'detail',
    OPEN_MODAL: 'modal',
  },
  input: {
    EXIT_INPUT: 'main', // Goes back to previous (could be main or detail)
  },
  detail: {
    ENTER_INPUT: 'input',
    CLOSE_DETAIL: 'main', // Goes back to parent
    OPEN_DETAIL: 'detail', // Nested detail (stays in detail, updates stack)
    OPEN_MODAL: 'modal',
    GO_HOME: 'main',
  },
  modal: {
    CLOSE_MODAL: 'main', // Goes back to previous (could be main or detail)
    ENTER_INPUT: 'input', // Modal with input field
  },
};

/** Which keys are allowed in each state */
const KEY_PERMISSIONS: Record<FocusState, Set<KeyCategory>> = {
  main: new Set(['global_nav', 'global_quit', 'list_nav', 'selection', 'escape', 'refresh']),
  input: new Set([
    'text_input',
    'escape', // To exit input mode
    'selection', // Enter to submit
  ]),
  detail: new Set([
    'global_nav',
    'list_nav',
    'selection',
    'escape',
    'refresh',
    // Note: no 'global_quit' - q doesn't quit from detail view
  ]),
  modal: new Set([
    'selection', // Confirm action
    'escape', // Close modal
    'list_nav', // Navigate options
  ]),
};

export interface FocusStateMachineResult {
  /** Current focus state */
  state: FocusState;

  /** Transition to a new state */
  transition: (event: FocusTransition) => void;

  /** Check if a key category can be handled in current state */
  canHandle: (category: KeyCategory) => boolean;

  /** Check if currently in a specific state */
  isState: (s: FocusState) => boolean;

  /** State history stack (for debugging) */
  history: FocusState[];

  /** Previous state before current (for returnFocus pattern) */
  previousState: FocusState | null;
}

/**
 * Hook for managing focus state machine
 *
 * @param initialState - Starting state (defaults to 'main')
 * @returns State machine interface
 *
 * @example
 * const { state, transition, canHandle } = useFocusStateMachine();
 *
 * // Enter input mode when user starts typing
 * const handleSearchStart = () => transition('ENTER_INPUT');
 *
 * // Check if global quit should be handled
 * if (canHandle('global_quit')) {
 *   process.exit(0);
 * }
 */
export function useFocusStateMachine(initialState: FocusState = 'main'): FocusStateMachineResult {
  const [state, setState] = useState<FocusState>(initialState);
  const [history, setHistory] = useState<FocusState[]>([initialState]);

  const previousState = useMemo(
    () => (history.length >= 2 ? history[history.length - 2] : null),
    [history]
  );

  const transition = useCallback((event: FocusTransition) => {
    setState((currentState) => {
      const validTransitions = TRANSITIONS[currentState];
      const nextState = validTransitions[event];

      if (nextState === undefined) {
        // Invalid transition - log for debugging but don't change state
        if (process.env.NODE_ENV === 'development') {
          console.warn(`[FocusStateMachine] Invalid transition: ${currentState} + ${event}`);
        }
        return currentState;
      }

      // Special handling for EXIT_INPUT and CLOSE_MODAL - return to previous state
      if (event === 'EXIT_INPUT' || event === 'CLOSE_MODAL') {
        setHistory((prev) => {
          // Pop current state, return to previous
          if (prev.length >= 2) {
            const newHistory = prev.slice(0, -1);
            // Actually set state to previous
            const returnTo = newHistory[newHistory.length - 1];
            setState(returnTo);
            return newHistory;
          }
          return prev;
        });
        return currentState; // Will be overwritten by setHistory callback
      }

      // Normal transition - push new state to history
      setHistory((prev) => [...prev, nextState]);
      return nextState;
    });
  }, []);

  const canHandle = useCallback(
    (category: KeyCategory): boolean => {
      return KEY_PERMISSIONS[state].has(category);
    },
    [state]
  );

  const isState = useCallback((s: FocusState): boolean => state === s, [state]);

  return {
    state,
    transition,
    canHandle,
    isState,
    history,
    previousState,
  };
}

/**
 * Map a key press to a key category
 *
 * @param input - The input character
 * @param key - The key object from useInput
 * @returns The key category, or null if not categorized
 */
export function categorizeKey(
  input: string,
  key: { escape?: boolean; return?: boolean; tab?: boolean; ctrl?: boolean; shift?: boolean }
): KeyCategory | null {
  // ESC key
  if (key.escape) {
    return 'escape';
  }

  // Enter/Return key
  if (key.return) {
    return 'selection';
  }

  // Tab key (view switching)
  if (key.tab) {
    return 'global_nav';
  }

  // Ctrl+R (refresh)
  if (key.ctrl && input === 'r') {
    return 'refresh';
  }

  // Global navigation shortcuts
  if (input === '?' || input === 'M' || input === 'I') {
    return 'global_nav';
  }

  // Quit
  if (input === 'q') {
    return 'global_quit';
  }

  // List navigation
  if (['j', 'k', 'g', 'G'].includes(input)) {
    return 'list_nav';
  }

  // Refresh
  if (input === 'r') {
    return 'refresh';
  }

  // Any other printable character is text input
  if (input && input.length === 1 && !key.ctrl) {
    return 'text_input';
  }

  return null;
}

export default useFocusStateMachine;
