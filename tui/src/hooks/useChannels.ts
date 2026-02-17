/**
 * useChannels hook - Fetch and manage channel data
 * Issue #1004: Performance configuration tunables (Phase 5)
 *
 * Poll interval is configurable via workspace config [performance] section.
 */

import { useState, useEffect, useCallback } from 'react';
import type { Channel, ChannelMessage, BcResult } from '../types';
import { getChannels, getChannelHistory, sendChannelMessage } from '../services/bc';
import { usePerformanceConfig } from '../config';

export interface UseChannelsOptions {
  /** Polling interval in ms (default: from config) */
  pollInterval?: number;
  /** Whether to poll automatically (default: true) */
  autoPoll?: boolean;
}

export interface UseChannelsResult extends BcResult<Channel[]> {
  /** Manually refresh channels */
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch and poll channel list
 */
export function useChannels(options: UseChannelsOptions = {}): UseChannelsResult {
  // Get configurable poll interval from workspace config
  const perfConfig = usePerformanceConfig();
  const defaultPollInterval = perfConfig.poll_interval_channels;

  const { pollInterval = defaultPollInterval, autoPoll = true } = options;

  const [data, setData] = useState<Channel[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchChannels = useCallback(async () => {
    try {
      const response = await getChannels();
      setData(response.channels);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch channels');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchChannels();
  }, [fetchChannels]);

  useEffect(() => {
    if (!autoPoll) return;
    const interval = setInterval(() => { void fetchChannels(); }, pollInterval);
    return () => { clearInterval(interval); };
  }, [autoPoll, pollInterval, fetchChannels]);

  return { data, error, loading, refresh: fetchChannels };
}

export interface UseChannelHistoryOptions {
  /** Max messages to fetch (default: 50) */
  limit?: number;
  /** Polling interval in ms (default: 2000) */
  pollInterval?: number;
  /** Whether to poll automatically (default: true) */
  autoPoll?: boolean;
}

export interface UseChannelHistoryResult extends BcResult<ChannelMessage[]> {
  /** Channel name */
  channel: string;
  /** Send a message to this channel */
  send: (message: string) => Promise<void>;
  /** Manually refresh history */
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch and poll channel message history
 */
export function useChannelHistory(
  channelName: string,
  options: UseChannelHistoryOptions = {}
): UseChannelHistoryResult {
  const { limit = 50, pollInterval = 2000, autoPoll = true } = options;

  const [data, setData] = useState<ChannelMessage[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchHistory = useCallback(async () => {
    try {
      const response = await getChannelHistory(channelName, limit);
      setData(response.messages);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch history');
    } finally {
      setLoading(false);
    }
  }, [channelName, limit]);

  const send = useCallback(
    async (message: string) => {
      await sendChannelMessage(channelName, message);
      // Refresh after sending
      await fetchHistory();
    },
    [channelName, fetchHistory]
  );

  useEffect(() => {
    void fetchHistory();
  }, [fetchHistory]);

  useEffect(() => {
    if (!autoPoll) return;
    const interval = setInterval(() => { void fetchHistory(); }, pollInterval);
    return () => { clearInterval(interval); };
  }, [autoPoll, pollInterval, fetchHistory]);

  return {
    data,
    error,
    loading,
    channel: channelName,
    send,
    refresh: fetchHistory,
  };
}

/**
 * Hook to get unread message count for a channel
 * Tracks last read timestamp and counts newer messages
 */
export function useUnreadCount(
  channelName: string,
  options?: UseChannelHistoryOptions
): { unread: number; markRead: () => void; loading: boolean } {
  const { data: messages, loading } = useChannelHistory(channelName, options);
  const [lastReadTime, setLastReadTime] = useState<Date>(new Date());

  const unread =
    messages?.filter((msg) => new Date(msg.time) > lastReadTime).length ?? 0;

  const markRead = useCallback(() => {
    setLastReadTime(new Date());
  }, []);

  return { unread, markRead, loading };
}

/**
 * Hook to get all channels with their unread counts
 */
export function useChannelsWithUnread(options?: UseChannelsOptions): {
  channels: (Channel & { unread: number })[] | null;
  loading: boolean;
  error: string | null;
} {
  const { data: channels, loading: channelsLoading, error } = useChannels(options);
  const [channelsWithUnread, setChannelsWithUnread] = useState<
    (Channel & { unread: number })[] | null
  >(null);

  useEffect(() => {
    if (!channels) return;

    // For now, set unread to 0 - actual implementation would track per-channel
    // This is a placeholder until we have proper unread tracking
    const withUnread = channels.map((ch) => ({
      ...ch,
      unread: 0,
    }));
    setChannelsWithUnread(withUnread);
  }, [channels]);

  return {
    channels: channelsWithUnread,
    loading: channelsLoading,
    error,
  };
}
