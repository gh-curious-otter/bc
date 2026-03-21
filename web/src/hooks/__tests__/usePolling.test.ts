import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePolling } from '../usePolling';

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

describe('usePolling', () => {
  it('returns data from fetcher', async () => {
    const fetcher = vi.fn().mockResolvedValue(['a', 'b']);
    const { result } = renderHook(() => usePolling(fetcher, 5000));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(result.current.data).toEqual(['a', 'b']);
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
    expect(result.current.timedOut).toBe(false);
  });

  it('sets error on fetch failure', async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error('network down'));
    const { result } = renderHook(() => usePolling(fetcher, 5000));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(result.current.error).toBe('network down');
    expect(result.current.loading).toBe(false);
  });

  it('clears interval on unmount', async () => {
    const fetcher = vi.fn().mockResolvedValue('ok');
    const { result, unmount } = renderHook(() => usePolling(fetcher, 5000));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(result.current.data).toBe('ok');
    const callCount = fetcher.mock.calls.length;
    unmount();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(15000);
    });

    expect(fetcher.mock.calls.length).toBe(callCount);
  });

  it('polls at the given interval', async () => {
    const fetcher = vi.fn().mockResolvedValue('data');
    renderHook(() => usePolling(fetcher, 1000));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(fetcher).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    expect(fetcher.mock.calls.length).toBeGreaterThanOrEqual(2);
  });

  it('sets timedOut after 10 seconds without data', async () => {
    const fetcher = vi.fn().mockImplementation(() => new Promise(() => {})); // never resolves
    const { result } = renderHook(() => usePolling(fetcher, 5000));

    expect(result.current.timedOut).toBe(false);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });

    expect(result.current.timedOut).toBe(true);
  });

  it('clears timedOut when data arrives', async () => {
    let resolve: (v: string) => void;
    const fetcher = vi.fn().mockImplementation(() => new Promise<string>((r) => { resolve = r; }));
    const { result } = renderHook(() => usePolling(fetcher, 5000));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });

    expect(result.current.timedOut).toBe(true);

    await act(async () => {
      resolve!('data');
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(result.current.timedOut).toBe(false);
    expect(result.current.data).toBe('data');
  });
});
