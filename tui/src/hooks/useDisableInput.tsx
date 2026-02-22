/**
 * useDisableInput - Context for disabling input handling
 * Issue #1594: Remove prop drilling by using React Context
 *
 * Provides a centralized way to disable input handling across the app,
 * eliminating the need to pass disableInput prop through multiple layers.
 */

import React, { createContext, useContext, useMemo } from 'react';

interface DisableInputContextValue {
  /** Whether input handling is disabled */
  isDisabled: boolean;
}

const DisableInputContext = createContext<DisableInputContextValue>({
  isDisabled: false,
});

export interface DisableInputProviderProps {
  /** Whether to disable input handling */
  disabled?: boolean;
  children: React.ReactNode;
}

/**
 * Provider component for disable input context
 */
export function DisableInputProvider({
  disabled = false,
  children,
}: DisableInputProviderProps): React.ReactElement {
  const value = useMemo(() => ({
    isDisabled: disabled,
  }), [disabled]);

  return (
    <DisableInputContext.Provider value={value}>
      {children}
    </DisableInputContext.Provider>
  );
}

/**
 * Hook to access the disable input state
 *
 * @returns Object with isDisabled boolean
 *
 * @example
 * function MyComponent() {
 *   const { isDisabled } = useDisableInput();
 *   useInput((input, key) => {
 *     // handle input
 *   }, { isActive: !isDisabled });
 * }
 */
export function useDisableInput(): DisableInputContextValue {
  return useContext(DisableInputContext);
}
