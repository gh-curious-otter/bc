import { useEffect, useRef, useState, useCallback } from 'react';

export function usePolling<T>(
  fetcher: () => Promise<T>,
  intervalMs: number = 5000,
): { data: T | null; loading: boolean; error: string | null; refresh: () => void } {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const timer = useRef<ReturnType<typeof setInterval>>();
  const abortRef = useRef<AbortController>();

  const doFetch = useCallback(async () => {
    // Abort any in-flight request before starting a new one
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    try {
      const result = await fetcher();
      if (!controller.signal.aborted) {
        setData(result);
        setError(null);
      }
    } catch (err) {
      if (controller.signal.aborted) return; // Silently ignore aborted requests
      setError(err instanceof Error ? err.message : 'Fetch failed');
    } finally {
      if (!controller.signal.aborted) {
        setLoading(false);
      }
    }
  }, [fetcher]);

  useEffect(() => {
    void doFetch();
    timer.current = setInterval(() => void doFetch(), intervalMs);
    return () => {
      clearInterval(timer.current);
      abortRef.current?.abort(); // Abort in-flight request on unmount
    };
  }, [doFetch, intervalMs]);

  return { data, loading, error, refresh: doFetch };
}
