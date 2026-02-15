/**
 * FocusContext - Manages focus state across the TUI
 *
 * Provides a way for components to register as focusable and
 * manage keyboard focus between different areas of the UI.
 */

import React, {
  createContext,
  useContext,
  useState,
  useCallback,
  useMemo,
} from 'react';

export type FocusArea = 'sidebar' | 'main' | 'detail' | 'input' | 'modal';

interface FocusContextValue {
  /** Currently focused area */
  focusedArea: FocusArea;
  /** Set focus to a specific area */
  setFocus: (area: FocusArea) => void;
  /** Check if an area is focused */
  isFocused: (area: FocusArea) => boolean;
  /** Previous focused area (for returning focus) */
  previousArea: FocusArea | null;
  /** Return focus to previous area */
  returnFocus: () => void;
  /** Cycle focus to next area */
  cycleFocus: () => void;
}

const FocusContext = createContext<FocusContextValue | null>(null);

const FOCUS_ORDER: FocusArea[] = ['sidebar', 'main', 'detail'];

interface FocusProviderProps {
  children: React.ReactNode;
  /** Initial focused area (defaults to 'main') */
  initialFocus?: FocusArea;
}

export function FocusProvider({
  children,
  initialFocus = 'main',
}: FocusProviderProps): React.ReactElement {
  const [focusedArea, setFocusedArea] = useState<FocusArea>(initialFocus);
  const [previousArea, setPreviousArea] = useState<FocusArea | null>(null);

  const setFocus = useCallback(
    (area: FocusArea) => {
      setPreviousArea(focusedArea);
      setFocusedArea(area);
    },
    [focusedArea]
  );

  const isFocused = useCallback(
    (area: FocusArea): boolean => focusedArea === area,
    [focusedArea]
  );

  const returnFocus = useCallback(() => {
    // Restore the previous focus area.
    // NOTE: We don't include 'previousArea' in dependencies because that creates
    // stale closure issues. Instead, this callback always reads the current
    // state when called via React's closure mechanism.
    setPreviousArea((prev) => {
      if (prev) {
        setFocusedArea(prev);
      }
      return null; // Clear previous area after restoring
    });
  }, []);

  const cycleFocus = useCallback(() => {
    const currentIndex = FOCUS_ORDER.indexOf(focusedArea);
    const nextIndex = (currentIndex + 1) % FOCUS_ORDER.length;
    setPreviousArea(focusedArea);
    setFocusedArea(FOCUS_ORDER[nextIndex]);
  }, [focusedArea]);

  const value = useMemo(
    () => ({
      focusedArea,
      setFocus,
      isFocused,
      previousArea,
      returnFocus,
      cycleFocus,
    }),
    [focusedArea, setFocus, isFocused, previousArea, returnFocus, cycleFocus]
  );

  return (
    <FocusContext.Provider value={value}>{children}</FocusContext.Provider>
  );
}

export function useFocus(): FocusContextValue {
  const context = useContext(FocusContext);
  if (!context) {
    throw new Error('useFocus must be used within a FocusProvider');
  }
  return context;
}

/**
 * Hook that checks if the specified area is focused
 */
export function useIsFocused(area: FocusArea): boolean {
  const { isFocused } = useFocus();
  return isFocused(area);
}

export default FocusContext;
