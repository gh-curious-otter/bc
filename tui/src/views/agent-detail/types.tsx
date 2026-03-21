/**
 * Shared types, constants, and helper components for AgentDetailView
 */

import React from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../../theme';

// Tab type for agent detail view
export type AgentTab = 'output' | 'live' | 'details' | 'metrics';

// Consolidated state for the agent detail reducer
export interface AgentDetailState {
  outputLines: string[];
  loading: boolean;
  error: string | null;
  inputMode: boolean;
  messageBuffer: string;
  sendStatus: string | null;
  activeTab: AgentTab;
  liveLines: string[];
  scrollOffset: number;
  isFollowing: boolean;
}

// Discriminated union for reducer actions
export type AgentDetailAction =
  | { type: 'SET_OUTPUT'; lines: string[] }
  | { type: 'SET_LOADING'; loading: boolean }
  | { type: 'SET_ERROR'; error: string | null }
  | { type: 'SET_TAB'; tab: AgentTab }
  | { type: 'TOGGLE_INPUT_MODE'; enabled: boolean }
  | { type: 'SET_MESSAGE_BUFFER'; buffer: string }
  | { type: 'SET_SEND_STATUS'; status: string | null }
  | { type: 'SET_LIVE_LINES'; lines: string[]; scrollOffset?: number }
  | { type: 'SET_SCROLL_OFFSET'; offset: number }
  | { type: 'SET_IS_FOLLOWING'; following: boolean }
  | { type: 'RESET_INPUT' };

// Consistent label column width
export const LABEL_WIDTH = 12;

/**
 * Normalize task status by replacing cooking metaphors with clearer terms.
 * Issue #970 - Replace cooking terminology from Claude Code status line.
 */
export function normalizeTask(task: string | undefined): string {
  if (!task) return '(no task)';
  const replacements: [string, string][] = [
    ['Saut\u00e9ed', 'Working'],
    ['Sauteed', 'Working'], // ASCII fallback
    ['Cooked', 'Processed'],
    ['Cogitated', 'Thinking'],
    ['Marinated', 'Idle'],
    ['Frolicking', 'Active'],
  ];
  for (const [old, replacement] of replacements) {
    if (task.includes(old)) {
      return task.replace(old, replacement);
    }
  }
  return task;
}

// Format date for display
export function formatDate(dateString: string | undefined): string {
  if (!dateString) return '-';
  try {
    const date = new Date(dateString);
    return date.toLocaleString();
  } catch {
    return dateString;
  }
}

// Format time for activity display (HH:MM:SS)
export function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
  } catch {
    return timestamp;
  }
}

// Format large numbers with K/M suffixes
export function formatNumber(num: number): string {
  if (num >= 1000000) {
    return `${(num / 1000000).toFixed(1)}M`;
  }
  if (num >= 1000) {
    return `${(num / 1000).toFixed(1)}K`;
  }
  return String(num);
}

// Truncate message to max length
export function truncateMessage(message: string, maxLen: number): string {
  if (message.length <= maxLen) return message;
  return message.slice(0, maxLen - 3) + '...';
}

// Format uptime from started_at timestamp
export function formatUptime(startedAt: string | undefined): string {
  if (!startedAt) return '-';
  try {
    const started = new Date(startedAt);
    const now = new Date();
    const diffMs = now.getTime() - started.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const mins = diffMins % 60;

    if (diffHours > 0) {
      return `${String(diffHours)}h ${String(mins)}m`;
    }
    return `${String(mins)}m`;
  } catch {
    return '-';
  }
}

// Helper component for detail rows with consistent alignment
// #1161: Fixed-width labels for proper column alignment
interface DetailRowProps {
  label: string;
  value: string | React.ReactElement;
  labelWidth?: number;
}

export function DetailRow({ label, value, labelWidth = LABEL_WIDTH }: DetailRowProps): React.ReactElement {
  const { theme } = useTheme();
  const paddedLabel = label.padEnd(labelWidth);
  return (
    <Box>
      <Text bold color={theme.colors.textMuted}>{paddedLabel}</Text>
      <Box marginLeft={1} flexShrink={1}>
        {typeof value === 'string' ? (
          <Text wrap="truncate">{value}</Text>
        ) : (
          value
        )}
      </Box>
    </Box>
  );
}

// Tab button component
interface TabButtonProps {
  label: string;
  tabKey: string;
  active: boolean;
}

export function TabButton({ label, tabKey, active }: TabButtonProps): React.ReactElement {
  const { theme } = useTheme();
  return (
    <Box>
      <Text color={active ? theme.colors.primary : theme.colors.textMuted} bold={active}>
        [{tabKey}]{label}
      </Text>
    </Box>
  );
}
