/**
 * FocusContext - Manages focus state across the TUI
 */

import React, {
  createContext,
  useContext,
  useState,
  useCallback,
  useMemo,
} from 'react';

export type FocusArea = 'main' | 'detail' | 'input' | 'modal' | 'view' | 'command' | 'filter';

interface FocusContextValue {
  focusedArea: FocusArea;
  setFocus: (area: FocusArea) => void;
  isFocused: (area: FocusArea) => boolean;
  previousArea: FocusArea | null;
  returnFocus: () => void;
  cycleFocus: () => void;
}

const FocusContext = createContext<FocusContextValue | null>(null);

const FOCUS_ORDER: FocusArea[] = ['main', 'detail'];

interface FocusProviderProps {
  children: React.ReactNode;
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
    setPreviousArea((prev) => {
      if (prev) {
        setFocusedArea(prev);
      }
      return null;
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

export function useIsFocused(area: FocusArea): boolean {
  const { isFocused } = useFocus();
  return isFocused(area);
}

export default FocusContext;
