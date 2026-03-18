import { useEffect, useRef, useState, useCallback } from 'react';

/**
 * usePolling - fetch data on interval until WebSocket replaces it.
 * Falls back gracefully when API is unavailable.
 */
export function usePolling<T>(
  fetcher: () => Promise<T>,
  intervalMs: number = 5000,
): { data: T | null; loading: boolean; error: string | null; refresh: () => void } {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const timer = useRef<ReturnType<typeof setInterval>>();

  const doFetch = useCallback(async () => {
    try {
      const result = await fetcher();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Fetch failed');
    } finally {
      setLoading(false);
    }
  }, [fetcher]);

  useEffect(() => {
    void doFetch();
    timer.current = setInterval(() => void doFetch(), intervalMs);
    return () => clearInterval(timer.current);
  }, [doFetch, intervalMs]);

  return { data, loading, error, refresh: doFetch };
}
