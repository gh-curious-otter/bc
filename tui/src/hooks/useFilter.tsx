/**
 * useFilter - Global filter state for k9s-style /filter
 */

import React, { createContext, useContext, useState, useCallback, useMemo } from 'react';

interface FilterContextValue {
  query: string;
  isActive: boolean;
  setFilter: (query: string) => void;
  clearFilter: () => void;
}

const FilterContext = createContext<FilterContextValue | null>(null);

interface FilterProviderProps {
  children: React.ReactNode;
}

export function FilterProvider({ children }: FilterProviderProps): React.ReactElement {
  const [query, setQuery] = useState('');

  const setFilter = useCallback((q: string) => {
    setQuery(q);
  }, []);

  const clearFilter = useCallback(() => {
    setQuery('');
  }, []);

  const value = useMemo(
    () => ({
      query,
      isActive: query.length > 0,
      setFilter,
      clearFilter,
    }),
    [query, setFilter, clearFilter]
  );

  return <FilterContext.Provider value={value}>{children}</FilterContext.Provider>;
}

export function useFilter(): FilterContextValue {
  const context = useContext(FilterContext);
  if (!context) {
    throw new Error('useFilter must be used within a FilterProvider');
  }
  return context;
}
