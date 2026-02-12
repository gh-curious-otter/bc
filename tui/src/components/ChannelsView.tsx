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
    limit: 20,
  });
  const [inputMode, setInputMode] = useState(false);
  const [messageBuffer, setMessageBuffer] = useState('');
  const [scrollOffset, setScrollOffset] = useState(0);

  const messageList = messages ?? [];
  // Calculate how many messages we can display (approximate height)
  const visibleCount = Math.max(5, Math.min(messageList.length, 10));
  const displayedMessages = messageList.slice(
    Math.max(0, messageList.length - visibleCount - scrollOffset),
    Math.max(visibleCount, messageList.length - scrollOffset)
  );

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
        // 'j' to scroll down (newer messages)
        if (input === 'j' && scrollOffset > 0) {
          setScrollOffset(scrollOffset - 1);
        }
        // 'k' to scroll up (older messages)
        if (input === 'k' && scrollOffset < messageList.length - visibleCount) {
          setScrollOffset(scrollOffset + 1);
        }
        // Arrow keys for scrolling
        if (key.downArrow && scrollOffset > 0) {
          setScrollOffset(scrollOffset - 1);
        }
        if (key.upArrow && scrollOffset < messageList.length - visibleCount) {
          setScrollOffset(scrollOffset + 1);
        }
      }
    },
    { isActive: !disableInput }
  );

  const formatTime = (isoString: string): string => {
    try {
      const date = new Date(isoString);
      return date.toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
      });
    } catch {
      return '-';
    }
  };

  return (
    <Box flexDirection="column" height="100%">
      <Box>
        <Text bold color="cyan">#{channel.name}</Text>
        <Text dimColor> - {channel.members.length} members</Text>
      </Box>
      <Text dimColor>m: compose, j/k or ↑↓: scroll, ESC: go back</Text>

      {/* Messages area - flexible height that grows */}
      <Box marginTop={1} flexDirection="column" flexGrow={1} overflow="hidden">
        {loading && <Text dimColor>Loading messages...</Text>}
        {error && <Text color="red">Error: {error}</Text>}
        {!loading && !error && displayedMessages.length === 0 && messageList.length === 0 && (
          <Text dimColor>No messages yet</Text>
        )}
        {!loading && !error && displayedMessages.map((msg, index) => (
          <Box key={index}>
            <Text dimColor>[{formatTime(msg.time)}] </Text>
            <Text color="yellow">{msg.sender}</Text>
            <Text dimColor>: </Text>
            <Text>{msg.message}</Text>
          </Box>
        ))}
        {!loading && !error && messageList.length > visibleCount && (
          <Text dimColor>
            (showing {displayedMessages.length} of {messageList.length} messages)
          </Text>
        )}
      </Box>

      {/* Input area - fixed at bottom */}
      <Box marginTop={1} borderStyle="single" borderColor={inputMode ? 'cyan' : 'gray'} paddingX={1}>
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
