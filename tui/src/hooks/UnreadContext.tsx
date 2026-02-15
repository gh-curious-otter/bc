/**
 * UnreadContext - Global unread message tracking
 *
 * Tracks when user last viewed each channel to calculate unread counts.
 * Persists data to ~/.bc/tui-unread.json for cross-session tracking.
 */

import React, { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';
import type { ReactNode } from 'react';
import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';

// Storage file location
const STORAGE_DIR = join(homedir(), '.bc');
const STORAGE_FILE = join(STORAGE_DIR, 'tui-unread.json');

interface UnreadData {
  /** Last viewed timestamp per channel (ISO string) */
  lastViewed: Record<string, string>;
  /** Last known message count per channel */
  lastMessageCount: Record<string, number>;
}

/**
 * Load unread tracking data from disk
 */
function loadUnreadData(): UnreadData {
  try {
    if (existsSync(STORAGE_FILE)) {
      const content = readFileSync(STORAGE_FILE, 'utf-8');
      return JSON.parse(content);
    }
  } catch {
    // Ignore read errors, start fresh
  }
  return {
    lastViewed: {},
    lastMessageCount: {},
  };
}

/**
 * Save unread tracking data to disk
 */
function saveUnreadData(data: UnreadData): void {
  try {
    if (!existsSync(STORAGE_DIR)) {
      mkdirSync(STORAGE_DIR, { recursive: true });
    }
    writeFileSync(STORAGE_FILE, JSON.stringify(data, null, 2));
  } catch {
    // Ignore write errors
  }
}

interface UnreadContextValue {
  /** Get unread count for a channel */
  getUnread: (channel: string, currentMessageCount: number) => number;
  /** Mark a channel as viewed with current message count */
  markViewed: (channel: string, messageCount: number) => void;
  /** Get last viewed time for a channel */
  getLastViewed: (channel: string) => Date | null;
}

const UnreadContext = createContext<UnreadContextValue | null>(null);

export interface UnreadProviderProps {
  children: ReactNode;
}

export function UnreadProvider({ children }: UnreadProviderProps): React.ReactElement {
  const [data, setData] = useState<UnreadData>(loadUnreadData);
  const dataRef = useRef(data);

  // Keep ref in sync for callbacks
  useEffect(() => {
    dataRef.current = data;
  }, [data]);

  // Save to disk when data changes
  useEffect(() => {
    saveUnreadData(data);
  }, [data]);

  const getUnread = useCallback((channel: string, currentMessageCount: number): number => {
    const lastCount = dataRef.current.lastMessageCount[channel];
    if (lastCount === undefined) {
      // Never viewed - all messages are "unread" but cap at a reasonable number
      return Math.min(currentMessageCount, 99);
    }
    // Unread = current - last viewed count
    return Math.max(0, currentMessageCount - lastCount);
  }, []);

  const markViewed = useCallback((channel: string, messageCount: number) => {
    setData((prev) => ({
      ...prev,
      lastViewed: {
        ...prev.lastViewed,
        [channel]: new Date().toISOString(),
      },
      lastMessageCount: {
        ...prev.lastMessageCount,
        [channel]: messageCount,
      },
    }));
  }, []);

  const getLastViewed = useCallback((channel: string): Date | null => {
    const timestamp = dataRef.current.lastViewed[channel];
    return timestamp ? new Date(timestamp) : null;
  }, []);

  const value: UnreadContextValue = {
    getUnread,
    markViewed,
    getLastViewed,
  };

  return (
    <UnreadContext.Provider value={value}>{children}</UnreadContext.Provider>
  );
}

/**
 * Hook to access unread tracking
 * @throws Error if used outside UnreadProvider
 */
export function useUnread(): UnreadContextValue {
  const context = useContext(UnreadContext);
  if (!context) {
    throw new Error('useUnread must be used within an UnreadProvider');
  }
  return context;
}
