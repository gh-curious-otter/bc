/**
 * Tests for useChannels hook - Channel data fetching and polling
 * Validates state management, polling behavior, and error handling
 */

import { renderHook, act } from '@testing-library/react';
import { useChannels, useChannelHistory } from '../useChannels';
import * as bcService from '../../services/bc';

jest.mock('../../services/bc');

const mockBcService = bcService as jest.Mocked<typeof bcService>;

describe('useChannels - Fetching channels', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it('initializes with loading state', () => {
    mockBcService.getChannels.mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    const { result } = renderHook(() => useChannels());
    expect(result.current.loading).toBe(true);
    expect(result.current.data).toBe(null);
    expect(result.current.error).toBe(null);
  });

  it('fetches and sets channel data', async () => {
    const channelsData = {
      channels: [
        { name: 'eng', members: ['eng-01', 'eng-02'] },
        { name: 'leads', members: ['tl-01', 'tl-02'] },
      ],
    };
    mockBcService.getChannels.mockResolvedValue(channelsData);

    const { result } = renderHook(() => useChannels());

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.loading).toBe(false);
    expect(result.current.data).toEqual(channelsData.channels);
    expect(result.current.error).toBe(null);
  });

  it('handles fetch errors gracefully', async () => {
    const errorMessage = 'Network error';
    mockBcService.getChannels.mockRejectedValue(new Error(errorMessage));

    const { result } = renderHook(() => useChannels());

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.loading).toBe(false);
    expect(result.current.data).toBe(null);
    expect(result.current.error).toBe(errorMessage);
  });

  it('polls channels at specified interval', async () => {
    mockBcService.getChannels.mockResolvedValue({
      channels: [{ name: 'eng', members: [] }],
    });

    const { result } = renderHook(() => useChannels({ pollInterval: 1000, autoPoll: true }));

    await act(async () => {
      jest.advanceTimersByTime(1000);
    });

    expect(mockBcService.getChannels).toHaveBeenCalledTimes(2); // Initial + first poll
  });

  it('stops polling when autoPoll is false', async () => {
    mockBcService.getChannels.mockResolvedValue({
      channels: [{ name: 'eng', members: [] }],
    });

    renderHook(() => useChannels({ autoPoll: false }));

    await act(async () => {
      jest.advanceTimersByTime(5000);
    });

    expect(mockBcService.getChannels).toHaveBeenCalledTimes(1); // Only initial fetch
  });

  it('provides manual refresh function', async () => {
    mockBcService.getChannels.mockResolvedValue({
      channels: [{ name: 'eng', members: [] }],
    });

    const { result } = renderHook(() => useChannels({ autoPoll: false }));

    await act(async () => {
      await result.current.refresh();
    });

    expect(mockBcService.getChannels).toHaveBeenCalledTimes(2); // Initial + refresh
  });
});

describe('useChannelHistory - Message history fetching', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it('fetches channel message history', async () => {
    const historyData = {
      messages: [
        { sender: 'eng-01', text: 'Hello', timestamp: 1000 },
        { sender: 'tl-01', text: 'Hi there', timestamp: 1100 },
      ],
    };
    mockBcService.getChannelHistory.mockResolvedValue(historyData);

    const { result } = renderHook(() => useChannelHistory('eng'));

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data).toEqual(historyData.messages);
    expect(result.current.channel).toBe('eng');
  });

  it('sends message to channel', async () => {
    const historyData = { messages: [] };
    mockBcService.getChannelHistory.mockResolvedValue(historyData);
    mockBcService.sendChannelMessage.mockResolvedValue(undefined);

    const { result } = renderHook(() => useChannelHistory('eng'));

    await act(async () => {
      await result.current.send('Test message');
    });

    expect(mockBcService.sendChannelMessage).toHaveBeenCalledWith('eng', 'Test message');
  });

  it('handles send errors gracefully', async () => {
    mockBcService.getChannelHistory.mockResolvedValue({ messages: [] });
    mockBcService.sendChannelMessage.mockRejectedValue(new Error('Send failed'));

    const { result } = renderHook(() => useChannelHistory('eng'));

    await expect(
      act(async () => {
        await result.current.send('Test');
      })
    ).rejects.toThrow();
  });

  it('polls history at specified interval', async () => {
    mockBcService.getChannelHistory.mockResolvedValue({ messages: [] });

    renderHook(() => useChannelHistory('eng', { pollInterval: 2000, autoPoll: true }));

    await act(async () => {
      jest.advanceTimersByTime(2000);
    });

    expect(mockBcService.getChannelHistory).toHaveBeenCalledTimes(2); // Initial + first poll
  });

  it('stops polling when autoPoll is false', async () => {
    mockBcService.getChannelHistory.mockResolvedValue({ messages: [] });

    renderHook(() => useChannelHistory('eng', { autoPoll: false }));

    await act(async () => {
      jest.advanceTimersByTime(5000);
    });

    expect(mockBcService.getChannelHistory).toHaveBeenCalledTimes(1); // Only initial fetch
  });

  it('refreshes history manually', async () => {
    mockBcService.getChannelHistory.mockResolvedValue({ messages: [] });

    const { result } = renderHook(() => useChannelHistory('eng', { autoPoll: false }));

    await act(async () => {
      await result.current.refresh();
    });

    expect(mockBcService.getChannelHistory).toHaveBeenCalledTimes(2); // Initial + refresh
  });

  it('handles missing message data gracefully', async () => {
    mockBcService.getChannelHistory.mockRejectedValue(new Error('Channel not found'));

    const { result } = renderHook(() => useChannelHistory('nonexistent'));

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.error).toBe('Channel not found');
    expect(result.current.data).toBe(null);
  });
});

describe('useChannels - Polling behavior', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it('respects custom poll interval', async () => {
    mockBcService.getChannels.mockResolvedValue({
      channels: [],
    });

    renderHook(() => useChannels({ pollInterval: 5000 }));

    await act(async () => {
      jest.advanceTimersByTime(5000);
    });

    expect(mockBcService.getChannels).toHaveBeenCalledTimes(2); // Initial + 1 poll

    await act(async () => {
      jest.advanceTimersByTime(4999);
    });

    expect(mockBcService.getChannels).toHaveBeenCalledTimes(2); // No additional poll yet
  });

  it('handles rapid polling without duplicates', async () => {
    mockBcService.getChannels.mockResolvedValue({
      channels: [{ name: 'eng', members: [] }],
    });

    const { result } = renderHook(() => useChannels({ pollInterval: 100, autoPoll: true }));

    await act(async () => {
      jest.advanceTimersByTime(500);
    });

    // Should have 1 initial + 5 polls = 6 calls
    expect(mockBcService.getChannels).toHaveBeenCalledTimes(6);
  });
});

describe('useChannelHistory - Edge cases', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it('handles empty channel history', async () => {
    mockBcService.getChannelHistory.mockResolvedValue({ messages: [] });

    const { result } = renderHook(() => useChannelHistory('empty-channel'));

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data).toEqual([]);
    expect(result.current.loading).toBe(false);
  });

  it('handles large message lists efficiently', async () => {
    const largeMessageSet = Array.from({ length: 1000 }, (_, i) => ({
      sender: `agent-${i % 10}`,
      text: `Message ${i}`,
      timestamp: 1000 + i,
    }));

    mockBcService.getChannelHistory.mockResolvedValue({ messages: largeMessageSet });

    const { result } = renderHook(() => useChannelHistory('busy-channel'));

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data).toHaveLength(1000);
  });

  it('handles special characters in messages', async () => {
    const specialMessages = {
      messages: [
        { sender: 'eng-01', text: 'Hello "world"', timestamp: 1000 },
        { sender: 'eng-02', text: "It's working!", timestamp: 1100 },
        { sender: 'eng-03', text: 'Line\\nbreak', timestamp: 1200 },
      ],
    };

    mockBcService.getChannelHistory.mockResolvedValue(specialMessages);

    const { result } = renderHook(() => useChannelHistory('text-channel'));

    await act(async () => {
      jest.runAllTimers();
    });

    expect(result.current.data).toEqual(specialMessages.messages);
  });
});
