/**
 * HintsContext - Centralized hint management for TUI footer
 *
 * Issue #1461: Fix duplicate keyboard hints
 *
 * Views provide their specific hints via this context.
 * The global footer combines view hints with universal hints.
 * This eliminates the need for ViewWrapper to render its own footer.
 */

import React, { createContext, useContext, useState, useCallback, useMemo, type ReactNode } from 'react';
import type { HintItem } from '../components/Footer';

interface HintsContextValue {
  /** Current view-specific hints */
  viewHints: HintItem[];
  /** Set view-specific hints (called by views/ViewWrapper) */
  setViewHints: (hints: HintItem[]) => void;
  /** Clear view hints (called when view unmounts) */
  clearViewHints: () => void;
}

const HintsContext = createContext<HintsContextValue | undefined>(undefined);

export interface HintsProviderProps {
  children: ReactNode;
}

/**
 * HintsProvider - Provides centralized hint management
 */
export function HintsProvider({ children }: HintsProviderProps): React.ReactElement {
  const [viewHints, setViewHintsState] = useState<HintItem[]>([]);

  const setViewHints = useCallback((hints: HintItem[]) => {
    setViewHintsState(hints);
  }, []);

  const clearViewHints = useCallback(() => {
    setViewHintsState([]);
  }, []);

  const value = useMemo(
    () => ({ viewHints, setViewHints, clearViewHints }),
    [viewHints, setViewHints, clearViewHints]
  );

  return (
    <HintsContext.Provider value={value}>
      {children}
    </HintsContext.Provider>
  );
}

/** Default noop context for when provider is not available (e.g., tests) */
const defaultContext: HintsContextValue = {
  viewHints: [],
  // eslint-disable-next-line @typescript-eslint/no-empty-function
  setViewHints: () => {},
  // eslint-disable-next-line @typescript-eslint/no-empty-function
  clearViewHints: () => {},
};

/**
 * useHintsContext - Access the hints context
 *
 * Returns a noop context if provider is not available (for testing).
 */
export function useHintsContext(): HintsContextValue {
  const context = useContext(HintsContext);
  // Return default noop context if provider not available (e.g., in tests)
  return context ?? defaultContext;
}

/**
 * useViewHints - Hook for views to provide their hints
 *
 * @example
 * ```tsx
 * function MyView() {
 *   useViewHints([
 *     { key: 'j/k', label: 'navigate' },
 *     { key: 'Enter', label: 'select' },
 *   ]);
 *   return <Box>...</Box>;
 * }
 * ```
 */
export function useViewHints(hints: HintItem[]): void {
  const { setViewHints, clearViewHints } = useHintsContext();

  React.useEffect(() => {
    setViewHints(hints);
    return () => {
      clearViewHints();
    };
  }, [hints, setViewHints, clearViewHints]);
}

export default HintsProvider;
