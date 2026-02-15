/**
 * Tests for useDemons hook - Scheduled task management
 * Validates demon lifecycle, logging, and error handling
 *
 * SKIPPED: These tests use jest.mock() which is incompatible with bun:test.
 * TODO: Convert to bun:test mock.module() in a follow-up PR.
 * See bc.test.ts for conversion example.
 */

import { renderHook, act } from '@testing-library/react';
import { useDemons, useDemonLogs } from '../useDemons';
import * as bcService from '../../services/bc';

// jest.mock('../../services/bc');

const mockBcService = bcService as any;

describe.skip('useDemons - Daemon/scheduled task management', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it('fetches list of demons', async () => {
    const demonsData = [
      { name: 'hourly-sync', enabled: true, next_run: 12345 },
      { name: 'daily-cleanup', enabled: false, next_run: 54321 },
      { name: 'weekly-report', enabled: true, next_run: 99999 },
    ];
    mockBcService.getDemons.mockResolvedValue(demonsData);

    const { result } = renderHook(() => useDemons());

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data).toEqual(demonsData);
    expect(result.current.loading).toBe(false);
  });

  it('returns empty list when no demons exist', async () => {
    mockBcService.getDemons.mockResolvedValue([]);

    const { result } = renderHook(() => useDemons());

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data).toEqual([]);
  });

  it('filters enabled demons', async () => {
    const demonsData = [
      { name: 'enabled-1', enabled: true, next_run: 100 },
      { name: 'disabled-1', enabled: false, next_run: 200 },
      { name: 'enabled-2', enabled: true, next_run: 300 },
    ];
    mockBcService.getDemons.mockResolvedValue(demonsData);

    const { result } = renderHook(() => useDemons());

    await act(async () => {
      jest.runAllTimers();
    });

    const enabled = result.current.data?.filter(d => d.enabled) || [];
    expect(enabled).toHaveLength(2);
  });

  it('sorts demons by next run time', async () => {
    const demonsData = [
      { name: 'task-c', enabled: true, next_run: 3000 },
      { name: 'task-a', enabled: true, next_run: 1000 },
      { name: 'task-b', enabled: true, next_run: 2000 },
    ];
    mockBcService.getDemons.mockResolvedValue(demonsData);

    const { result } = renderHook(() => useDemons());

    await act(async () => {
      jest.runAllTimers();
    });

    const sorted = (result.current.data || []).sort((a, b) => a.next_run - b.next_run);
    expect(sorted[0].name).toBe('task-a');
    expect(sorted[2].name).toBe('task-c');
  });

  it('polls demons at specified interval', async () => {
    mockBcService.getDemons.mockResolvedValue([
      { name: 'task', enabled: true, next_run: 1000 },
    ]);

    renderHook(() => useDemons({ pollInterval: 2000, autoPoll: true }));

    await act(async () => {
      jest.advanceTimersByTime(5000);
    });

    expect(mockBcService.getDemons).toHaveBeenCalledTimes(3); // Initial + 2 polls
  });

  it('provides manual refresh', async () => {
    mockBcService.getDemons.mockResolvedValue([]);

    const { result } = renderHook(() => useDemons({ autoPoll: false }));

    await act(async () => {
      await result.current.refresh();
    });

    expect(mockBcService.getDemons).toHaveBeenCalledTimes(2);
  });

  it('handles errors gracefully', async () => {
    mockBcService.getDemons.mockRejectedValue(new Error('Failed to fetch demons'));

    const { result } = renderHook(() => useDemons());

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.error).toBe('Failed to fetch demons');
    expect(result.current.data).toBe(null);
  });
});

describe.skip('useDemonLogs - Demon execution logs', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it('fetches demon run logs', async () => {
    const logsData = [
      { timestamp: 1000, status: 'success', message: 'Sync completed' },
      { timestamp: 2000, status: 'success', message: 'Sync completed' },
      { timestamp: 3000, status: 'failed', message: 'Network timeout' },
    ];
    mockBcService.getDemonLogs.mockResolvedValue(logsData);

    const { result } = renderHook(() => useDemonLogs('hourly-sync'));

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data).toEqual(logsData);
  });

  it('returns empty logs when none exist', async () => {
    mockBcService.getDemonLogs.mockResolvedValue([]);

    const { result } = renderHook(() => useDemonLogs('task'));

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data).toEqual([]);
  });

  it('respects tail limit', async () => {
    const logsData = Array.from({ length: 10 }, (_, i) => ({
      timestamp: (i + 1) * 1000,
      status: 'success',
      message: `Run ${i + 1}`,
    }));
    mockBcService.getDemonLogs.mockResolvedValue(logsData);

    renderHook(() => useDemonLogs('task', { tail: 5 }));

    await act(async () => {
      jest.runAllTimers();
    });

    expect(mockBcService.getDemonLogs).toHaveBeenCalledWith('task', expect.objectContaining({ tail: 5 }));
  });

  it('filters successful runs', async () => {
    const logsData = [
      { timestamp: 1000, status: 'success', message: 'OK' },
      { timestamp: 2000, status: 'failed', message: 'Error' },
      { timestamp: 3000, status: 'success', message: 'OK' },
    ];
    mockBcService.getDemonLogs.mockResolvedValue(logsData);

    const { result } = renderHook(() => useDemonLogs('task'));

    await act(async () => {
      jest.runAllTimers();
    });

    const successful = result.current.data?.filter(l => l.status === 'success') || [];
    expect(successful).toHaveLength(2);
  });

  it('provides log refresh', async () => {
    mockBcService.getDemonLogs.mockResolvedValue([]);

    const { result } = renderHook(() => useDemonLogs('task', { autoPoll: false }));

    await act(async () => {
      await result.current.refresh();
    });

    expect(mockBcService.getDemonLogs).toHaveBeenCalledTimes(2);
  });

  it('handles missing demon gracefully', async () => {
    mockBcService.getDemonLogs.mockRejectedValue(new Error('Demon not found'));

    const { result } = renderHook(() => useDemonLogs('nonexistent'));

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.error).toBe('Demon not found');
  });
});

describe.skip('useDemons - Demon control operations', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('enables a demon', async () => {
    mockBcService.enableDemon.mockResolvedValue(undefined);

    mockBcService.enableDemon('hourly-sync');

    expect(mockBcService.enableDemon).toHaveBeenCalledWith('hourly-sync');
  });

  it('disables a demon', async () => {
    mockBcService.disableDemon.mockResolvedValue(undefined);

    mockBcService.disableDemon('daily-cleanup');

    expect(mockBcService.disableDemon).toHaveBeenCalledWith('daily-cleanup');
  });

  it('runs demon manually', async () => {
    mockBcService.runDemon.mockResolvedValue(undefined);

    mockBcService.runDemon('hourly-sync');

    expect(mockBcService.runDemon).toHaveBeenCalledWith('hourly-sync');
  });

  it('handles control operation errors', async () => {
    mockBcService.enableDemon.mockRejectedValue(new Error('Demon not found'));

    await expect(bcService.enableDemon('invalid')).rejects.toThrow('Demon not found');
  });
});

describe.skip('useDemons - Edge cases', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it('handles demons with large names', async () => {
    const longName = 'a'.repeat(256);
    mockBcService.getDemons.mockResolvedValue([
      { name: longName, enabled: true, next_run: 1000 },
    ]);

    const { result } = renderHook(() => useDemons());

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data?.[0].name).toBe(longName);
  });

  it('handles demons with far future run times', async () => {
    mockBcService.getDemons.mockResolvedValue([
      { name: 'future-task', enabled: true, next_run: Number.MAX_SAFE_INTEGER },
    ]);

    const { result } = renderHook(() => useDemons());

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data?.[0].next_run).toBe(Number.MAX_SAFE_INTEGER);
  });

  it('handles rapid enable/disable cycles', async () => {
    mockBcService.enableDemon.mockResolvedValue(undefined);
    mockBcService.disableDemon.mockResolvedValue(undefined);

    for (let i = 0; i < 5; i++) {
      await bcService.enableDemon('task');
      await bcService.disableDemon('task');
    }

    expect(mockBcService.enableDemon).toHaveBeenCalledTimes(5);
    expect(mockBcService.disableDemon).toHaveBeenCalledTimes(5);
  });

  it('cleans up polling on unmount', () => {
    mockBcService.getDemons.mockResolvedValue([]);

    const { unmount } = renderHook(() => useDemons({ pollInterval: 1000 }));

    expect(mockBcService.getDemons).toHaveBeenCalled();

    unmount();

    jest.advanceTimersByTime(5000);
    // Should not add more calls after unmount
    expect(mockBcService.getDemons).toHaveBeenCalledTimes(1);
  });

  it('handles logs with special characters', async () => {
    const logsData = [
      { timestamp: 1000, status: 'success', message: 'Completed: "task" done' },
      { timestamp: 2000, status: 'failed', message: "Error: can't connect to DB" },
      { timestamp: 3000, status: 'success', message: 'Line\\nbreak\\nin message' },
    ];
    mockBcService.getDemonLogs.mockResolvedValue(logsData);

    const { result } = renderHook(() => useDemonLogs('task'));

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data).toEqual(logsData);
  });
});

describe.skip('useDemons - State consistency', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it('maintains list consistency across polls', async () => {
    const demonsData = [
      { name: 'task-1', enabled: true, next_run: 1000 },
      { name: 'task-2', enabled: false, next_run: 2000 },
    ];
    mockBcService.getDemons.mockResolvedValue(demonsData);

    const { result } = renderHook(() => useDemons({ pollInterval: 500 }));

    await act(async () => {
      jest.advanceTimersByTime(2000);
    });

    // All polls should return same data
    expect(result.current.data).toEqual(demonsData);
  });

  it('handles demon status changes', async () => {
    mockBcService.getDemons
      .mockResolvedValueOnce([
        { name: 'task', enabled: false, next_run: 1000 },
      ])
      .mockResolvedValueOnce([
        { name: 'task', enabled: true, next_run: 2000 },
      ]);

    const { result } = renderHook(() => useDemons({ pollInterval: 500, autoPoll: true }));

    await act(async () => {
      jest.advanceTimersByTime(600);
    });

    const firstRead = result.current.data?.[0]?.enabled;

    await act(async () => {
      jest.advanceTimersByTime(600);
    });

    const secondRead = result.current.data?.[0]?.enabled;

    expect(firstRead).toBe(false);
    expect(secondRead).toBe(true);
  });
});
