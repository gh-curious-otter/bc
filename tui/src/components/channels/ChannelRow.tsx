/**
 * ChannelRow - Single channel row in the channel list
 * Extracted from ChannelsView.tsx (#1590)
 */

import React from 'react';
import { Box, Text } from 'ink';
import type { Channel } from '../../types';

export interface ChannelRowProps {
  channel: Channel;
  selected: boolean;
  unreadCount: number;
}

/**
 * ChannelRow - Renders a single channel in the list
 *
 * Features:
 * - Selection indicator (▸)
 * - Unread message badge (● or "N new")
 * - Member count suffix
 * - Color highlighting for selected/unread
 */
export function ChannelRow({ channel, selected, unreadCount }: ChannelRowProps): React.ReactElement {
  // #981 fix: Build name row as single truncated text to ensure visibility at 80 cols
  // #1129: Highlight channels with unread messages
  // #1364 Issue 2: Clarify channel numbers (unread vs members)
  // Priority: name > unread indicator > member count > description
  const namePrefix = selected ? '▸ ' : '  ';
  const channelName = `#${channel.name}`;

  // Format member count with 'm' suffix to distinguish from unread (#1364)
  const memberInfo = ` ${String(channel.members.length)}m`;

  // Format unread badge with 'new' label to clarify meaning (#1364)
  // "●" for 1 unread, "N new" for multiple
  const unreadBadge = unreadCount > 0
    ? unreadCount === 1
      ? ' ●'
      : ` ${unreadCount > 99 ? '99+' : String(unreadCount)} new`
    : '';

  // Build single text line to avoid nested Text truncation issues on narrow terminals
  // Issue #981: Nested Text elements break rendering at 80x24 width
  const nameLineText = `${namePrefix}${channelName}${unreadBadge}${memberInfo}`;

  // Determine text color: cyan if selected, yellow if has unread, default otherwise
  const textColor = selected ? 'cyan' : unreadCount > 0 ? 'yellow' : undefined;

  // #1171 fix: Remove explicit width="100%" to avoid nested width calculation issues at 80x24
  // #1528 fix: Add flexGrow={1} to ensure Box takes available width for wrap="truncate" to work
  // The parent Box already has width="100%", inner Box needs flexGrow to claim its share
  return (
    <Box flexDirection="column" flexGrow={1}>
      {/* Name row: single Text for proper truncation at narrow widths */}
      <Text
        wrap="truncate"
        color={textColor}
        bold={selected || unreadCount > 0}
      >
        {nameLineText}
      </Text>
      {channel.description && (
        <Text dimColor wrap="truncate">{channel.description}</Text>
      )}
    </Box>
  );
}

export default ChannelRow;
