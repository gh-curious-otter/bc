/**
 * ChannelsView - Channel list and message history component
 */

import React, { useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { useChannels, useChannelHistory } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import type { Channel } from '../types';

interface ChannelsViewProps {
  /** Disable input handling (useful for testing) */
  disableInput?: boolean;
}

export function ChannelsView({ disableInput = false }: ChannelsViewProps): React.ReactElement {
  const { data: channels, loading: channelsLoading, error: channelsError } = useChannels();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [viewMode, setViewMode] = useState<'list' | 'history'>('list');

  const selectedChannel = channels?.[selectedIndex];

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
        // Enter channel
        if (key.return && selectedChannel) {
          setViewMode('history');
        }
      } else {
        // Back to list
        if (key.escape) {
          setViewMode('list');
        }
      }
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
    return <ChannelHistoryView channel={selectedChannel} disableInput={disableInput} />;
  }

  return (
    <Box flexDirection="column">
      <Text bold>Channels</Text>
      <Text dimColor>↑/↓ navigate, Enter to view messages, ESC to go back</Text>
      <Box marginTop={1} flexDirection="column" borderStyle="single" borderColor="gray" paddingX={1}>
        {channels?.map((channel, index) => (
          <ChannelRow
            key={channel.name}
            channel={channel}
            selected={index === selectedIndex}
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
}

function ChannelRow({ channel, selected }: ChannelRowProps): React.ReactElement {
  return (
    <Box>
      <Text color={selected ? 'cyan' : undefined} bold={selected}>
        {selected ? '▸ ' : '  '}
        #{channel.name}
      </Text>
      <Text dimColor> ({channel.members.length} members)</Text>
    </Box>
  );
}

interface ChannelHistoryViewProps {
  channel: Channel;
  disableInput?: boolean;
}

function ChannelHistoryView({
  channel,
  disableInput = false,
}: ChannelHistoryViewProps): React.ReactElement {
  const { data: messages, loading, error, send } = useChannelHistory(channel.name, {
    limit: 50,
  });
  const [inputMode, setInputMode] = useState(false);
  const [messageBuffer, setMessageBuffer] = useState('');
  const [scrollOffset, setScrollOffset] = useState(0);
  const { setFocus, returnFocus } = useFocus();

  /**
   * Synchronize focus state with input mode
   *
   * When user enters input mode (presses 'm'), we set focus to 'input' area.
   * This prevents global keybinds (q, 1-9, ESC) from triggering during message typing.
   *
   * When user exits input mode (presses Enter or Escape), we restore focus to the
   * previous area, which re-enables global navigation keybinds.
   *
   * The useKeyboardNavigation hook checks isFocused('input') before handling global
   * keybinds, so focus state acts as the guard that disables/enables them.
   *
   * This fixes issue #653: "After typing a message in a channel, the keybinds to
   * q, 1,2,3... are not re-enabled"
   */
  useEffect(() => {
    if (inputMode) {
      setFocus('input');
    } else {
      returnFocus();
    }
  }, [inputMode, setFocus, returnFocus]);

  useInput(
    (input, key) => {
      if (inputMode) {
        if (key.return) {
          if (messageBuffer.trim()) {
            send(messageBuffer.trim()).catch(() => {
              // Error handled by hook
            });
            setMessageBuffer('');
          }
          setInputMode(false);
        } else if (key.escape) {
          setMessageBuffer('');
          setInputMode(false);
        } else if (key.backspace || key.delete) {
          setMessageBuffer(messageBuffer.slice(0, -1));
        } else if (input && !key.ctrl && !key.meta) {
          setMessageBuffer(messageBuffer + input);
        }
      } else {
        // 'm' to compose message
        if (input === 'm') {
          setInputMode(true);
        }
        // 'j' to scroll down, 'k' to scroll up
        if (input === 'j' && messages) {
          setScrollOffset(Math.max(0, scrollOffset - 1));
        }
        if (input === 'k' && messages) {
          setScrollOffset(Math.min(Math.max(0, messages.length - 10), scrollOffset + 1));
        }
      }
    },
    { isActive: !disableInput }
  );

  const displayMessages = messages ? messages.slice(Math.max(0, messages.length - 10 - scrollOffset), messages.length - scrollOffset) : [];
  const hasMoreAbove = scrollOffset > 0;
  const hasMoreBelow = messages && messages.length > 10 && scrollOffset < messages.length - 10;

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

      {/* Message area - flex grow to fill available space */}
      <Box marginBottom={1} flexDirection="column" flexGrow={1}>
        {loading && <Text dimColor>Loading messages...</Text>}
        {error && <Text color="red">Error: {error}</Text>}
        {!loading && !error && (
          <>
            {hasMoreAbove && <Text dimColor>↑ more messages above</Text>}
            {displayMessages.map((msg, index) => (
              <Box key={index}>
                <Text color="yellow">{msg.sender}</Text>
                <Text dimColor> ({formatMessageTime(msg.time)}) </Text>
                <Text>: </Text>
                <Text>{msg.message}</Text>
              </Box>
            ))}
            {hasMoreBelow && <Text dimColor>↓ more messages below</Text>}
            {messages?.length === 0 && <Text dimColor>No messages yet</Text>}
          </>
        )}
      </Box>

      {/* Input area - fixed height with proper separation */}
      <Box height={3} flexDirection="column" marginBottom={1} borderStyle="single" borderColor={inputMode ? 'cyan' : 'gray'} paddingX={1}>
        {inputMode ? (
          <Text>
            <Text color="cyan">{'> '}</Text>
            {messageBuffer}
            <Text color="cyan">▌</Text>
          </Text>
        ) : (
          <Text dimColor>Press m to compose message</Text>
        )}
      </Box>

      {/* Footer - anchored at bottom */}
      <Box height={1}>
        <Text dimColor>ESC: back  m: compose  j/k: scroll  [?] help  Theme: dark</Text>
      </Box>
    </Box>
  );
}

/**
 * Format message timestamp for display
 */
function formatMessageTime(time: string): string {
  try {
    const date = new Date(time);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return 'now';
    if (diffMins < 60) return `${String(diffMins)}m ago`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${String(diffHours)}h ago`;

    return date.toLocaleTimeString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return time;
  }
}

export default ChannelsView;
