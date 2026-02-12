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

/**
 * Format timestamp for display
 */
function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    const now = new Date();
    const isToday = date.toDateString() === now.toDateString();

    if (isToday) {
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
    return date.toLocaleDateString([], { month: 'short', day: 'numeric' }) + ' ' +
           date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } catch {
    return '';
  }
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

  const visibleCount = 12;
  const totalMessages = messages?.length ?? 0;
  const maxOffset = Math.max(0, totalMessages - visibleCount);

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
        // j/k or arrow keys to scroll messages
        if ((input === 'j' || key.downArrow) && scrollOffset < maxOffset) {
          setScrollOffset(scrollOffset + 1);
        }
        if ((input === 'k' || key.upArrow) && scrollOffset > 0) {
          setScrollOffset(scrollOffset - 1);
        }
        // g/G to jump to start/end
        if (input === 'g') {
          setScrollOffset(0);
        }
        if (input === 'G') {
          setScrollOffset(maxOffset);
        }
      }
    },
    { isActive: !disableInput }
  );

  // Auto-scroll to bottom when new messages arrive
  React.useEffect(() => {
    if (messages && messages.length > 0) {
      setScrollOffset(maxOffset);
    }
  }, [messages?.length, maxOffset]);

  const visibleMessages = messages?.slice(scrollOffset, scrollOffset + visibleCount) ?? [];

  return (
    <Box flexDirection="column" height="100%">
      {/* Header */}
      <Box>
        <Text bold color="cyan">#{channel.name}</Text>
        <Text dimColor> - {channel.members.length} members</Text>
      </Box>
      <Text dimColor>ESC back | j/k scroll | g/G start/end | m compose</Text>

      {/* Message area - takes remaining space */}
      <Box marginTop={1} flexDirection="column" flexGrow={1}>
        {loading && <Text dimColor>Loading messages...</Text>}
        {error && <Text color="red">Error: {error}</Text>}
        {visibleMessages.map((msg, index) => (
          <Box key={scrollOffset + index}>
            <Text dimColor>{formatTime(msg.time)} </Text>
            <Text color="yellow">{msg.sender}</Text>
            <Text dimColor>: </Text>
            <Text>{msg.message}</Text>
          </Box>
        ))}
        {messages?.length === 0 && <Text dimColor>No messages yet</Text>}
        {totalMessages > visibleCount && (
          <Text dimColor>
            [{scrollOffset + 1}-{Math.min(scrollOffset + visibleCount, totalMessages)} of {totalMessages}]
          </Text>
        )}
      </Box>

      {/* Input area - anchored at bottom */}
      <Box borderStyle="single" borderColor={inputMode ? 'cyan' : 'gray'} paddingX={1}>
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

export default ChannelsView;
