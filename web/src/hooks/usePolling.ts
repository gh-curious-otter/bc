import { useEffect, useRef, useState, useCallback } from "react";

const TIMEOUT_MS = 10000;

export function usePolling<T>(
  fetcher: () => Promise<T>,
  intervalMs: number = 5000,
): {
  data: T | null;
  loading: boolean;
  error: string | null;
  refresh: () => void;
  timedOut: boolean;
} {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [timedOut, setTimedOut] = useState(false);
  const timer = useRef<ReturnType<typeof setInterval>>();
  const timeoutTimer = useRef<ReturnType<typeof setTimeout>>();
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
        setTimedOut(false);
        clearTimeout(timeoutTimer.current);
      }
    } catch (err) {
      if (controller.signal.aborted) return; // Silently ignore aborted requests
      setError(err instanceof Error ? err.message : "Fetch failed");
    } finally {
      if (!controller.signal.aborted) {
        setLoading(false);
      }
    }
  }, [fetcher]);

  useEffect(() => {
    setTimedOut(false);
    timeoutTimer.current = setTimeout(() => {
      setTimedOut(true);
    }, TIMEOUT_MS);

    void doFetch();
    timer.current = setInterval(() => void doFetch(), intervalMs);
    return () => {
      clearInterval(timer.current);
      clearTimeout(timeoutTimer.current);
      abortRef.current?.abort(); // Abort in-flight request on unmount
    };
  }, [doFetch, intervalMs]);

  const refresh = useCallback(() => {
    setTimedOut(false);
    setLoading(true);
    clearTimeout(timeoutTimer.current);
    timeoutTimer.current = setTimeout(() => {
      setTimedOut(true);
    }, TIMEOUT_MS);
    void doFetch();
  }, [doFetch]);

  return { data, loading, error, refresh, timedOut };
}
