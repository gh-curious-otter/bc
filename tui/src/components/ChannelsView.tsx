/**
 * ChannelsView - Channel list and message history component
 */

import React, { useState, useEffect, useMemo } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { useChannels, useChannelHistory, useUnread } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { ChatMessage } from './ChatMessage';
import type { Channel } from '../types';

/** Duration in ms to show send errors before auto-clearing */
const SEND_ERROR_DISPLAY_DURATION = 3000;

/**
 * Calculate input box height based on message length
 * Height expands from 3 (min) to 10 (max) lines based on content
 */
function calculateInputHeight(messageLength: number, terminalWidth: number): number {
  const MIN_HEIGHT = 3;
  const MAX_HEIGHT = 10;
  // Account for border (2) + prompt "> " (2) + cursor (1)
  const availableWidth = Math.max(terminalWidth - 5, 20);
  const lines = Math.ceil(messageLength / availableWidth) + 1;
  return Math.min(MAX_HEIGHT, Math.max(MIN_HEIGHT, lines));
}

interface ChannelsViewProps {
  /** Disable input handling (useful for testing) */
  disableInput?: boolean;
}

export function ChannelsView({ disableInput = false }: ChannelsViewProps): React.ReactElement {
  const { data: channels, loading: channelsLoading, error: channelsError } = useChannels();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [viewMode, setViewMode] = useState<'list' | 'history'>('list');
  const { setBreadcrumbs, clearBreadcrumbs } = useNavigation();
  const { getLastViewed } = useUnread();
  const { setFocus } = useFocus();

  const selectedChannel = channels?.[selectedIndex];

  // Update breadcrumbs and focus when view mode changes
  useEffect(() => {
    if (viewMode === 'history' && selectedChannel) {
      setBreadcrumbs([{ label: `#${selectedChannel.name}` }]);
    } else {
      clearBreadcrumbs();
      // Restore focus to 'main' when returning to list view
      // This must happen AFTER global ESC handler has checked focus
      setFocus('main');
    }
  }, [viewMode, selectedChannel, setBreadcrumbs, clearBreadcrumbs, setFocus]);

  // Calculate unread status for each channel (new = never viewed)
  const getChannelUnread = (channelName: string): number => {
    const lastViewed = getLastViewed(channelName);
    // If never viewed, show as "new" (1 indicates new)
    return lastViewed === null ? 1 : 0;
  };

  useInput(
    (input, key) => {
      if (viewMode === 'list') {
        // Navigate channel list
        if ((key.upArrow || input === 'k') && selectedIndex > 0) {
          setSelectedIndex(selectedIndex - 1);
        }
        if ((key.downArrow || input === 'j') && channels && selectedIndex < channels.length - 1) {
          setSelectedIndex(selectedIndex + 1);
        }
        // Vim-style top/bottom navigation
        if (input === 'g') {
          setSelectedIndex(0);
        }
        if (input === 'G' && channels) {
          setSelectedIndex(channels.length - 1);
        }
        // Enter channel - set focus to 'view' BEFORE changing mode to prevent race
        if (key.return && selectedChannel) {
          setFocus('view');
          setViewMode('history');
        }
      }
      // Note: ESC in history mode is handled by ChannelHistoryView's onBack callback
    },
    { isActive: !disableInput }
  );

  if (channelsLoading) {
    return (
      <Box flexDirection="column">
        <Text bold>Channels</Text>
        <Text dimColor>Loading channels...</Text>
      </Box>
    );
  }

  if (channelsError) {
    return (
      <Box flexDirection="column">
        <Text bold>Channels</Text>
        <Text color="red">Error: {channelsError}</Text>
      </Box>
    );
  }

  if (viewMode === 'history' && selectedChannel) {
    return (
      <ChannelHistoryView
        key={selectedChannel.name}
        channel={selectedChannel}
        disableInput={disableInput}
        onBack={() => { setViewMode('list'); }}
      />
    );
  }

  return (
    <Box flexDirection="column" width="100%">
      <Text bold>Channels</Text>
      <Text dimColor>↑/↓ navigate, Enter to view messages, ESC to go back</Text>
      <Box marginTop={1} flexDirection="column" width="100%" borderStyle="single" borderColor="gray" paddingX={2}>
        {channels?.map((channel, index) => (
          <ChannelRow
            key={channel.name}
            channel={channel}
            selected={index === selectedIndex}
            unreadCount={getChannelUnread(channel.name)}
          />
        ))}
        {(!channels || channels.length === 0) && (
          <Text dimColor>No channels found</Text>
        )}
      </Box>
    </Box>
  );
}

interface ChannelRowProps {
  channel: Channel;
  selected: boolean;
  unreadCount: number;
}

function ChannelRow({ channel, selected, unreadCount }: ChannelRowProps): React.ReactElement {
  // #981 fix: Build name row as single truncated text to ensure visibility at 80 cols
  // Priority: name > unread indicator > member count > description
  const namePrefix = selected ? '▸ ' : '  ';
  const channelName = `#${channel.name}`;
  const memberInfo = ` (${String(channel.members.length)})`;
  const unreadInfo = unreadCount > 0 ? ` [${unreadCount > 99 ? '99+' : String(unreadCount)} new]` : '';

  return (
    <Box width="100%" flexDirection="column">
      {/* Name row: combined into single Text for proper truncation */}
      <Text wrap="truncate">
        <Text color={selected ? 'cyan' : undefined} bold={selected}>
          {namePrefix}{channelName}
        </Text>
        {unreadCount > 0 && (
          <Text color="yellow" bold>{unreadInfo}</Text>
        )}
        <Text dimColor>{memberInfo}</Text>
      </Text>
      {channel.description && (
        <Text dimColor wrap="truncate">{channel.description}</Text>
      )}
    </Box>
  );
}

interface ChannelHistoryViewProps {
  channel: Channel;
  disableInput?: boolean;
  onBack?: () => void;
}

function ChannelHistoryView({
  channel,
  disableInput = false,
  onBack,
}: ChannelHistoryViewProps): React.ReactElement {
  const { data: messages, loading, error, send } = useChannelHistory(channel.name, {
    limit: 50,
  });
  const [inputMode, setInputMode] = useState(false);
  const [messageBuffer, setMessageBuffer] = useState('');
  const [scrollOffset, setScrollOffset] = useState(0);
  const [sendError, setSendError] = useState<string | null>(null);
  const { setFocus } = useFocus();

  // Auto-clear send errors after a delay
  useEffect(() => {
    if (!sendError) return;
    const timer = setTimeout(() => { setSendError(null); }, SEND_ERROR_DISPLAY_DURATION);
    return () => { clearTimeout(timer); };
  }, [sendError]);
  const { stdout } = useStdout();
  const { markViewed } = useUnread();

  // Mark channel as viewed when messages load
  useEffect(() => {
    if (messages && messages.length >= 0) {
      markViewed(channel.name, messages.length);
    }
  }, [channel.name, messages, markViewed]);

  // Calculate dynamic input height based on message length
  const terminalWidth = stdout.columns;
  const inputHeight = useMemo(
    () => calculateInputHeight(messageBuffer.length, terminalWidth),
    [messageBuffer.length, terminalWidth]
  );
  // Message area adjusts as input expands (base 14 + extra from input growth)
  const messageAreaHeight = 14 + (3 - inputHeight);

  /**
   * Synchronize focus state with input mode
   *
   * When user enters input mode (presses 'm'), we set focus to 'input' area.
   * This prevents global keybinds (q, 1-9, ESC) from triggering during message typing.
   *
   * When user exits input mode (presses Enter or Escape), we set focus to 'view'
   * to keep global navigation disabled while in channel history view. This ensures that
   * ESC navigates back to channel list (via onBack) rather than to Dashboard.
   *
   * This fixes issue #653: "After typing a message in a channel, the keybinds to
   * q, 1,2,3... are not re-enabled"
   * This also fixes issue #884: "ESC from channel history goes to Dashboard instead
   * of Channels list"
   */
  useEffect(() => {
    if (inputMode) {
      setFocus('input');
    } else {
      // Keep focus on 'view' to prevent global ESC from going to Dashboard
      setFocus('view');
    }
  }, [inputMode, setFocus]);

  useInput(
    (input, key) => {
      if (inputMode) {
        if (key.return) {
          if (messageBuffer.trim()) {
            send(messageBuffer.trim()).catch((err: unknown) => {
              const message = err instanceof Error ? err.message : String(err);
              setSendError(`Send failed: ${message}`);
            });
            setMessageBuffer('');
          }
          setInputMode(false);
        } else if (key.escape) {
          // Draft save: preserve message on Esc for later editing
          // Only clear if message was empty (cancel vs save draft)
          setInputMode(false);
        } else if (key.backspace || key.delete) {
          setMessageBuffer(messageBuffer.slice(0, -1));
        } else if (input && !key.ctrl && !key.meta) {
          setMessageBuffer(messageBuffer + input);
        }
      } else {
        // ESC to go back to channel list
        // Note: Don't call returnFocus() here - focus must stay 'view' until
        // after global ESC handler runs, otherwise goHome() will fire
        if (key.escape) {
          onBack?.();
        }
        // 'm' to compose message
        if (input === 'm') {
          setInputMode(true);
        }
        // 'c' to clear draft
        if (input === 'c' && messageBuffer) {
          setMessageBuffer('');
        }
        // 'j' to scroll down, 'k' to scroll up
        // Note: maxMessages=3 for display, scroll bounds use same constant
        if (input === 'j' && messages) {
          setScrollOffset(Math.max(0, scrollOffset - 1));
        }
        if (input === 'k' && messages) {
          setScrollOffset(Math.min(Math.max(0, messages.length - 3), scrollOffset + 1));
        }
      }
    },
    { isActive: !disableInput }
  );

  // #915 fix: Reduce max messages from 10 to 3 to fit in available space
  // Layout math: 14 lines available / 4 lines per message = ~3 messages max
  const maxMessages = 3;
  const displayMessages = messages ? messages.slice(Math.max(0, messages.length - maxMessages - scrollOffset), messages.length - scrollOffset) : [];
  const hasMoreAbove = scrollOffset > 0;
  const hasMoreBelow = messages && messages.length > maxMessages && scrollOffset < messages.length - maxMessages;

  return (
    <Box flexDirection="column" width="100%" height="100%">
      {/* Header section - fixed height */}
      <Box flexDirection="column" height={3} marginBottom={1}>
        <Box>
          <Text bold color="cyan">#{channel.name}</Text>
          <Text dimColor> - {channel.members.length} members</Text>
        </Box>
        <Text dimColor>ESC to go back, m to compose, j/k to scroll</Text>
      </Box>

      {/* Message area - dynamic height adjusts as input expands */}
      <Box
        marginBottom={1}
        flexDirection="column"
        height={messageAreaHeight}
        borderStyle="single"
        borderColor="gray"
        paddingX={1}
        overflow="hidden"
      >
        {loading && <Text dimColor>Loading messages...</Text>}
        {error && <Text color="red">Error: {error}</Text>}
        {!loading && !error && (
          <>
            {hasMoreAbove && <Text dimColor>↑ more messages above</Text>}
            {displayMessages.map((msg, index) => (
              <ChatMessage
                key={`${msg.time}-${String(index)}`}
                sender={msg.sender}
                message={msg.message}
                timestamp={msg.time}
                currentUser={process.env.BC_AGENT_ID}
              />
            ))}
            {hasMoreBelow && <Text dimColor>↓ more messages below</Text>}
            {messages?.length === 0 && <Text dimColor>No messages yet</Text>}
          </>
        )}
      </Box>

      {/* Send error feedback */}
      {sendError && (
        <Box marginBottom={1}>
          <Text color="red">{sendError}</Text>
        </Box>
      )}

      {/* Input area - auto-expands based on message length (3-10 lines) */}
      <Box height={inputHeight} flexDirection="column" marginBottom={1} borderStyle="single" borderColor={inputMode ? 'cyan' : (messageBuffer ? 'yellow' : 'gray')} paddingX={1}>
        {inputMode ? (
          <Text>
            <Text color="cyan">{'> '}</Text>
            {messageBuffer}
            <Text color="cyan">▌</Text>
          </Text>
        ) : messageBuffer ? (
          <Text>
            <Text color="yellow">[Draft] </Text>
            <Text dimColor>{messageBuffer.length > 40 ? messageBuffer.slice(0, 40) + '...' : messageBuffer}</Text>
            <Text dimColor> (press m to edit)</Text>
          </Text>
        ) : (
          <Text dimColor>Press m to compose message</Text>
        )}
      </Box>

      {/* Footer - anchored at bottom */}
      <Box height={1}>
        <Text dimColor>ESC: {inputMode ? 'save draft' : 'back'}  m: compose  j/k: scroll  Enter: send</Text>
      </Box>
    </Box>
  );
}

export default ChannelsView;
