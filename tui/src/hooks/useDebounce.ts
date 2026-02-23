/**
 * useDebounce - Debounce hook for expensive operations
 * Issue #1602: Add debounce to expensive input operations
 *
 * Provides utilities to debounce values and callbacks:
 * - useDebounce: Debounce a changing value
 * - useDebouncedCallback: Debounce a callback function
 * - useDebouncedSearch: Specialized hook for search input patterns
 */

import { useState, useEffect, useRef, useCallback, useMemo } from 'react';

/** Default debounce delay in milliseconds */
export const DEFAULT_DEBOUNCE_MS = 300;

/**
 * Debounce a value - returns the value after the specified delay
 * Useful for search inputs where you want to wait for user to stop typing
 *
 * @param value - The value to debounce
 * @param delay - Delay in milliseconds (default: 300)
 * @returns The debounced value
 *
 * @example
 * ```tsx
 * const [searchQuery, setSearchQuery] = useState('');
 * const debouncedQuery = useDebounce(searchQuery, 300);
 *
 * useEffect(() => {
 *   // Only runs 300ms after user stops typing
 *   performSearch(debouncedQuery);
 * }, [debouncedQuery]);
 * ```
 */
export function useDebounce<T>(value: T, delay: number = DEFAULT_DEBOUNCE_MS): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    // Set up timeout to update debounced value after delay
    const timer = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    // Clear timeout if value changes or component unmounts
    return () => {
      clearTimeout(timer);
    };
  }, [value, delay]);

  return debouncedValue;
}

/**
 * Options for useDebouncedCallback
 */
export interface UseDebouncedCallbackOptions {
  /** Delay in milliseconds (default: 300) */
  delay?: number;
  /** Maximum wait time before forcing execution (default: undefined = no max) */
  maxWait?: number;
  /** Call immediately on first invocation (default: false) */
  leading?: boolean;
  /** Call on trailing edge of timeout (default: true) */
  trailing?: boolean;
}

/**
 * Result from useDebouncedCallback
 */
export interface UseDebouncedCallbackResult<T extends (...args: unknown[]) => unknown> {
  /** The debounced callback function */
  callback: T;
  /** Cancel any pending debounced call */
  cancel: () => void;
  /** Flush any pending debounced call immediately */
  flush: () => void;
  /** Whether a call is currently pending */
  isPending: boolean;
}

/**
 * Debounce a callback function
 * More flexible than useDebounce for cases where you need to debounce actions
 *
 * @param callback - The callback function to debounce
 * @param options - Debounce options
 * @returns Object with debounced callback and control functions
 *
 * @example
 * ```tsx
 * const { callback: debouncedSearch, cancel } = useDebouncedCallback(
 *   (query: string) => {
 *     performSearch(query);
 *   },
 *   { delay: 300 }
 * );
 *
 * // Call debouncedSearch on each keystroke
 * // Actual search only runs 300ms after last call
 * ```
 */
export function useDebouncedCallback<T extends (...args: unknown[]) => unknown>(
  callback: T,
  options: UseDebouncedCallbackOptions = {}
): UseDebouncedCallbackResult<T> {
  const {
    delay = DEFAULT_DEBOUNCE_MS,
    maxWait,
    leading = false,
    trailing = true,
  } = options;

  const callbackRef = useRef(callback);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const maxTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastArgsRef = useRef<Parameters<T> | null>(null);
  const lastCallTimeRef = useRef<number>(0);
  const [isPending, setIsPending] = useState(false);

  // Keep callback ref up to date
  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  // Cancel any pending timers
  const cancel = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    if (maxTimerRef.current) {
      clearTimeout(maxTimerRef.current);
      maxTimerRef.current = null;
    }
    lastArgsRef.current = null;
    setIsPending(false);
  }, []);

  // Flush pending call immediately
  const flush = useCallback(() => {
    if (lastArgsRef.current && trailing) {
      callbackRef.current(...lastArgsRef.current);
    }
    cancel();
  }, [cancel, trailing]);

  // Create the debounced callback
  const debouncedCallback = useCallback(
    (...args: Parameters<T>) => {
      const now = Date.now();
      lastArgsRef.current = args;
      lastCallTimeRef.current = now;

      // Clear existing timer
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }

      // Handle leading edge call
      if (leading && !isPending) {
        callbackRef.current(...args);
      }

      setIsPending(true);

      // Set up trailing edge timer
      timerRef.current = setTimeout(() => {
        if (trailing && lastArgsRef.current) {
          callbackRef.current(...lastArgsRef.current);
        }
        cancel();
      }, delay);

      // Set up maxWait timer if specified
      if (maxWait && !maxTimerRef.current) {
        maxTimerRef.current = setTimeout(() => {
          if (lastArgsRef.current) {
            callbackRef.current(...lastArgsRef.current);
          }
          cancel();
        }, maxWait);
      }
    },
    [delay, maxWait, leading, trailing, isPending, cancel]
  ) as T;

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      cancel();
    };
  }, [cancel]);

  return useMemo(
    () => ({
      callback: debouncedCallback,
      cancel,
      flush,
      isPending,
    }),
    [debouncedCallback, cancel, flush, isPending]
  );
}

/**
 * Options for useDebouncedSearch
 */
export interface UseDebouncedSearchOptions {
  /** Initial search query (default: '') */
  initialQuery?: string;
  /** Debounce delay in milliseconds (default: 300) */
  delay?: number;
  /** Minimum query length to trigger search (default: 0) */
  minLength?: number;
  /** Called when debounced query changes */
  onSearch?: (query: string) => void;
}

/**
 * Result from useDebouncedSearch
 */
export interface UseDebouncedSearchResult {
  /** Current input value (immediate) */
  query: string;
  /** Debounced query value (for filtering) */
  debouncedQuery: string;
  /** Set the search query */
  setQuery: (query: string) => void;
  /** Clear the search */
  clear: () => void;
  /** Whether search is being debounced */
  isDebouncing: boolean;
}

/**
 * Specialized hook for search input patterns
 * Combines state management with debouncing for search UX
 *
 * @param options - Search options
 * @returns Search state and controls
 *
 * @example
 * ```tsx
 * const { query, debouncedQuery, setQuery, clear, isDebouncing } = useDebouncedSearch({
 *   delay: 300,
 *   minLength: 2,
 *   onSearch: (q) => console.log('Searching:', q),
 * });
 *
 * // Use query for display, debouncedQuery for filtering
 * const filtered = items.filter(item =>
 *   debouncedQuery.length >= 2 && item.name.includes(debouncedQuery)
 * );
 * ```
 */
export function useDebouncedSearch(
  options: UseDebouncedSearchOptions = {}
): UseDebouncedSearchResult {
  const {
    initialQuery = '',
    delay = DEFAULT_DEBOUNCE_MS,
    minLength = 0,
    onSearch,
  } = options;

  const [query, setQueryState] = useState(initialQuery);
  const debouncedQuery = useDebounce(query, delay);
  const prevDebouncedRef = useRef(debouncedQuery);

  // Track if we're currently debouncing
  const isDebouncing = query !== debouncedQuery;

  // Call onSearch when debounced query changes
  useEffect(() => {
    if (prevDebouncedRef.current !== debouncedQuery) {
      prevDebouncedRef.current = debouncedQuery;
      if (debouncedQuery.length >= minLength && onSearch) {
        onSearch(debouncedQuery);
      }
    }
  }, [debouncedQuery, minLength, onSearch]);

  const setQuery = useCallback((newQuery: string) => {
    setQueryState(newQuery);
  }, []);

  const clear = useCallback(() => {
    setQueryState('');
  }, []);

  return useMemo(
    () => ({
      query,
      debouncedQuery,
      setQuery,
      clear,
      isDebouncing,
    }),
    [query, debouncedQuery, setQuery, clear, isDebouncing]
  );
}

export default useDebounce;
