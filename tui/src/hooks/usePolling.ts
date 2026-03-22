/**
 * usePolling - Enhanced polling hooks for real-time updates
 * Issue #551: Real-time polling with incremental fetch
 * Issue #1004: Performance configuration tunables (Phase 5)
 *
 * Poll intervals are configurable via workspace config [performance] section.
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import type { ChannelMessage, Agent, BcResult } from '../types';
import { getChannelHistory, getStatus } from '../services/bc';
import { usePerformanceConfig } from '../config';

export interface UsePollingOptions {
  /** Polling interval in ms (default: from config) */
  interval?: number;
  /** Whether to poll automatically (default: true) */
  enabled?: boolean;
  /** Callback when new data arrives */
  onUpdate?: () => void;
}

export interface UseMessagePollingOptions extends UsePollingOptions {
  /** Channel to poll */
  channel: string;
  /** Max messages to fetch per poll (default: 50) */
  limit?: number;
  /** Callback when new messages arrive */
  onNewMessages?: (messages: ChannelMessage[]) => void;
}

export interface UseMessagePollingResult extends BcResult<ChannelMessage[]> {
  /** Number of new messages since last check */
  newCount: number;
  /** Timestamp of last poll */
  lastPoll: Date | null;
  /** Pause polling */
  pause: () => void;
  /** Resume polling */
  resume: () => void;
  /** Whether polling is active */
  isPolling: boolean;
  /** Manually trigger a poll */
  poll: () => Promise<void>;
}

/**
 * Hook for polling channel messages with incremental detection
 * Tracks new messages since last poll and provides callbacks
 */
export function useMessagePolling(options: UseMessagePollingOptions): UseMessagePollingResult {
  // Get configurable poll interval from workspace config
  const perfConfig = usePerformanceConfig();
  const defaultInterval = perfConfig.poll_interval_channels;

  const {
    channel,
    interval = defaultInterval,
    enabled = true,
    limit = 50,
    onNewMessages,
    onUpdate,
  } = options;

  const [data, setData] = useState<ChannelMessage[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [newCount, setNewCount] = useState(0);
  const [lastPoll, setLastPoll] = useState<Date | null>(null);
  const [isPolling, setIsPolling] = useState(enabled);

  // Track last seen message time for incremental detection
  const lastSeenTimeRef = useRef<string | null>(null);
  const isFirstFetchRef = useRef(true);

  const fetchMessages = useCallback(async () => {
    try {
      const response = await getChannelHistory(channel, limit);
      const messages = response.messages;

      // Detect new messages (those newer than last seen)
      let newMessages: ChannelMessage[] = [];
      if (!isFirstFetchRef.current && lastSeenTimeRef.current && messages.length > 0) {
        const lastSeenTime = new Date(lastSeenTimeRef.current);
        newMessages = messages.filter((msg) => new Date(msg.time) > lastSeenTime);
      }

      // Update last seen time to newest message
      if (messages.length > 0) {
        const newestTime = messages.reduce((max, msg) => {
          const msgTime = new Date(msg.time);
          return msgTime > max ? msgTime : max;
        }, new Date(0));
        lastSeenTimeRef.current = newestTime.toISOString();
      }

      setData(messages);
      setNewCount(newMessages.length);
      setLastPoll(new Date());
      setError(null);
      isFirstFetchRef.current = false;

      // Callbacks
      if (newMessages.length > 0 && onNewMessages) {
        onNewMessages(newMessages);
      }
      if (onUpdate) {
        onUpdate();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch messages');
    } finally {
      setLoading(false);
    }
  }, [channel, limit, onNewMessages, onUpdate]);

  const pause = useCallback(() => {
    setIsPolling(false);
  }, []);
  const resume = useCallback(() => {
    setIsPolling(true);
  }, []);

  // Initial fetch
  useEffect(() => {
    isFirstFetchRef.current = true;
    lastSeenTimeRef.current = null;
    void fetchMessages();
  }, [channel, fetchMessages]); // Reset on channel change

  // Polling interval
  useEffect(() => {
    if (!isPolling) return;
    const timer = setInterval(() => {
      void fetchMessages();
    }, interval);
    return () => {
      clearInterval(timer);
    };
  }, [isPolling, interval, fetchMessages]);

  return {
    data,
    error,
    loading,
    newCount,
    lastPoll,
    pause,
    resume,
    isPolling,
    poll: fetchMessages,
  };
}

export interface UseAgentPollingOptions extends UsePollingOptions {
  /** Callback when agent states change */
  onStateChange?: (agents: Agent[], changes: AgentChange[]) => void;
}

export interface AgentChange {
  agent: string;
  field: 'state' | 'task' | 'tool';
  oldValue: string | undefined;
  newValue: string | undefined;
}

export interface UseAgentPollingResult extends BcResult<Agent[]> {
  /** Agents that changed since last poll */
  changes: AgentChange[];
  /** Workspace name */
  workspace: string;
  /** Agent counts */
  counts: { total: number; active: number; working: number };
  /** Pause polling */
  pause: () => void;
  /** Resume polling */
  resume: () => void;
  /** Whether polling is active */
  isPolling: boolean;
  /** Manually trigger a poll */
  poll: () => Promise<void>;
}

/**
 * Hook for polling agent status with change detection
 * Tracks state changes and provides callbacks
 */
export function useAgentPolling(options: UseAgentPollingOptions = {}): UseAgentPollingResult {
  // Get configurable poll interval from workspace config
  const perfConfig = usePerformanceConfig();
  const defaultInterval = perfConfig.poll_interval_agents;

  const { interval = defaultInterval, enabled = true, onStateChange, onUpdate } = options;

  const [data, setData] = useState<Agent[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [changes, setChanges] = useState<AgentChange[]>([]);
  const [workspace, setWorkspace] = useState('');
  const [counts, setCounts] = useState({ total: 0, active: 0, working: 0 });
  const [isPolling, setIsPolling] = useState(enabled);

  // Track previous agent states for change detection
  const prevAgentsRef = useRef<Map<string, Agent>>(new Map());

  const fetchAgents = useCallback(async () => {
    try {
      const status = await getStatus();
      const agents = status.agents;

      // Detect changes
      const detectedChanges: AgentChange[] = [];
      for (const agent of agents) {
        const prev = prevAgentsRef.current.get(agent.name);
        if (prev) {
          if (prev.state !== agent.state) {
            detectedChanges.push({
              agent: agent.name,
              field: 'state',
              oldValue: prev.state,
              newValue: agent.state,
            });
          }
          if (prev.task !== agent.task) {
            detectedChanges.push({
              agent: agent.name,
              field: 'task',
              oldValue: prev.task,
              newValue: agent.task,
            });
          }
          if (prev.tool !== agent.tool) {
            detectedChanges.push({
              agent: agent.name,
              field: 'tool',
              oldValue: prev.tool,
              newValue: agent.tool,
            });
          }
        }
      }

      // Update previous state map
      prevAgentsRef.current = new Map(agents.map((a) => [a.name, a]));

      setData(agents);
      setChanges(detectedChanges);
      setWorkspace(status.workspace);
      setCounts({
        total: status.total,
        active: status.active,
        working: status.working,
      });
      setError(null);

      // Callbacks
      if (detectedChanges.length > 0 && onStateChange) {
        onStateChange(agents, detectedChanges);
      }
      if (onUpdate) {
        onUpdate();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch agents');
    } finally {
      setLoading(false);
    }
  }, [onStateChange, onUpdate]);

  const pause = useCallback(() => {
    setIsPolling(false);
  }, []);
  const resume = useCallback(() => {
    setIsPolling(true);
  }, []);

  // Initial fetch
  useEffect(() => {
    prevAgentsRef.current = new Map();
    void fetchAgents();
  }, [fetchAgents]);

  // Polling interval
  useEffect(() => {
    if (!isPolling) return;
    const timer = setInterval(() => {
      void fetchAgents();
    }, interval);
    return () => {
      clearInterval(timer);
    };
  }, [isPolling, interval, fetchAgents]);

  return {
    data,
    error,
    loading,
    changes,
    workspace,
    counts,
    pause,
    resume,
    isPolling,
    poll: fetchAgents,
  };
}

/**
 * Hook for coordinated polling across multiple data sources
 * Useful for dashboard that needs agents, channels, and costs
 */
export interface UseCoordinatedPollingOptions {
  /** Base interval in ms (default: 2000) */
  interval?: number;
  /** Whether polling is enabled (default: true) */
  enabled?: boolean;
}

export function useCoordinatedPolling(options: UseCoordinatedPollingOptions = {}) {
  // Get configurable poll interval from workspace config
  const perfConfig = usePerformanceConfig();
  const defaultInterval = perfConfig.poll_interval_status;

  const { interval = defaultInterval, enabled = true } = options;
  const [tick, setTick] = useState(0);
  const [isPaused, setIsPaused] = useState(!enabled);

  useEffect(() => {
    if (isPaused) return;
    const timer = setInterval(() => {
      setTick((t) => t + 1);
    }, interval);
    return () => {
      clearInterval(timer);
    };
  }, [isPaused, interval]);

  const pause = useCallback(() => {
    setIsPaused(true);
  }, []);
  const resume = useCallback(() => {
    setIsPaused(false);
  }, []);
  const trigger = useCallback(() => {
    setTick((t) => t + 1);
  }, []);

  return {
    /** Current tick count (changes each interval) */
    tick,
    /** Pause all coordinated polling */
    pause,
    /** Resume coordinated polling */
    resume,
    /** Manually trigger an update */
    trigger,
    /** Whether polling is paused */
    isPaused,
  };
}
