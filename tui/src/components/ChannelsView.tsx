/**
 * ChannelsView - Channel list and message history component
 */

import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import { useChannels, useChannelHistory } from '../hooks';
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
      <Box marginTop={1} flexDirection="column">
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
  const visibleMessages = 12;

  const messageList = messages ?? [];
  const messageCount = messageList.length;

  // Scroll to bottom when messages change
  React.useEffect(() => {
    if (messages) {
      const newOffset = Math.max(0, messages.length - visibleMessages);
      setScrollOffset(newOffset);
    }
  }, [messages, visibleMessages]);

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
        // Message navigation
        if ((key.upArrow || input === 'k') && scrollOffset > 0) {
          setScrollOffset((o) => Math.max(0, o - 1));
        }
        if ((key.downArrow || input === 'j') && scrollOffset < messageCount - visibleMessages) {
          setScrollOffset((o) => Math.min(messageCount - visibleMessages, o + 1));
        }
        // 'm' to compose message
        if (input === 'm') {
          setInputMode(true);
        }
      }
    },
    { isActive: !disableInput }
  );

  // Memoized visible slice
  const visibleSlice = React.useMemo(
    () => messageList.slice(scrollOffset, scrollOffset + visibleMessages),
    [messageList, scrollOffset]
  );
  const canScrollUp = scrollOffset > 0;
  const canScrollDown = scrollOffset < messageCount - visibleMessages;

  return (
    <Box flexDirection="column" height={undefined}>
      <Box>
        <Text bold color="cyan">#{channel.name}</Text>
        <Text dimColor> - {channel.members.length} members</Text>
      </Box>
      <Text dimColor>ESC to go back, m to compose message, ↑/↓ or j/k to scroll</Text>

      {/* Messages container with scrolling */}
      <Box marginTop={1} flexDirection="column" flexGrow={1}>
        {loading && !messages && <Text dimColor>Loading messages...</Text>}
        {error && <Text color="red">Error: {error}</Text>}

        {canScrollUp && (
          <Box>
            <Text dimColor>↑ {scrollOffset} more above</Text>
          </Box>
        )}

        {visibleSlice.map((msg, index) => (
          <Box key={`${msg.time}-${index}`}>
            <Box width={8}>
              <Text dimColor>{formatTimestamp(msg.time)}</Text>
            </Box>
            <Box width={14}>
              <Text color="yellow" bold>{msg.sender.slice(0, 13)}</Text>
            </Box>
            <Box flexGrow={1}>
              <Text wrap="truncate">{msg.message}</Text>
            </Box>
          </Box>
        ))}

        {canScrollDown && (
          <Box>
            <Text dimColor>↓ {messageCount - scrollOffset - visibleMessages} more below</Text>
          </Box>
        )}

        {messageCount === 0 && <Text dimColor>No messages yet</Text>}
      </Box>

      {/* Input area - anchored at bottom */}
      <Box marginTop={0} borderStyle="single" borderColor={inputMode ? 'cyan' : 'gray'} paddingX={1}>
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
    </Box>
  );
}

/**
 * Format timestamp for display
 * Shows time only if today, otherwise shows date
 */
function formatTimestamp(isoString: string): string {
  try {
    const date = new Date(isoString);
    const now = new Date();
    const isToday = date.toDateString() === now.toDateString();

    if (isToday) {
      return date.toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      });
    }

    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
    });
  } catch {
    return '??:??';
  }
}

export default ChannelsView;
