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
});
