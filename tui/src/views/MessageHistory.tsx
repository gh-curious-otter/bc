import { useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import type { ChannelMessage } from '../types';
import { useChannelHistory } from '../hooks';

interface MessageHistoryProps {
  channelName: string;
  maxMessages?: number;
  onBack?: () => void;
}

/**
 * MessageHistory - Scrollable message history view for a channel
 * Issue #549 - Message history view
 */
export function MessageHistory({
  channelName,
  maxMessages = 50,
  onBack,
}: MessageHistoryProps) {
  const { data: messages, loading: isLoading, error } = useChannelHistory(channelName, {
    limit: maxMessages,
    pollInterval: 5000,
  });
  const [scrollOffset, setScrollOffset] = useState(0);
  const visibleMessages = 15;

  // Scroll to bottom when messages change
  useEffect(() => {
    if (messages) {
      const newOffset = Math.max(0, messages.length - visibleMessages);
      setScrollOffset(newOffset);
    }
  }, [messages]);

  const messageList = messages ?? [];
  const messageCount = messageList.length;

  // Keyboard navigation
  useInput((input, key) => {
    if (key.upArrow && scrollOffset > 0) {
      setScrollOffset((o) => Math.max(0, o - 1));
    }
    if (key.downArrow && scrollOffset < messageCount - visibleMessages) {
      setScrollOffset((o) => Math.min(messageCount - visibleMessages, o + 1));
    }
    if (key.pageUp) {
      setScrollOffset((o) => Math.max(0, o - visibleMessages));
    }
    if (key.pageDown) {
      setScrollOffset((o) =>
        Math.min(messageCount - visibleMessages, o + visibleMessages)
      );
    }
    if (input === 'q' || key.escape) {
      onBack?.();
    }
    if (input === 'g') {
      // Go to top
      setScrollOffset(0);
    }
    if (input === 'G') {
      // Go to bottom
      setScrollOffset(Math.max(0, messageCount - visibleMessages));
    }
  });

  if (isLoading && messageCount === 0) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="cyan">Loading #{channelName} history...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="red">Error: {error}</Text>
        <Text dimColor>Press 'q' to go back</Text>
      </Box>
    );
  }

  const visibleSlice = messageList.slice(scrollOffset, scrollOffset + visibleMessages);
  const canScrollUp = scrollOffset > 0;
  const canScrollDown = scrollOffset < messageCount - visibleMessages;

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="cyan">
          #{channelName}
        </Text>
        <Text dimColor> ({messageCount} messages)</Text>
        {isLoading && <Text color="yellow"> ↻</Text>}
      </Box>

      {/* Scroll indicator (top) */}
      {canScrollUp && (
        <Box>
          <Text dimColor>↑ {scrollOffset} more messages above</Text>
        </Box>
      )}

      {/* Messages */}
      <Box
        flexDirection="column"
        borderStyle="single"
        borderColor="gray"
        paddingX={1}
        height={visibleMessages + 2}
      >
        {visibleSlice.length === 0 ? (
          <Text dimColor>No messages in this channel</Text>
        ) : (
          visibleSlice.map((msg, idx) => (
            <MessageItem
              key={`${msg.time}-${idx}`}
              message={msg}
              isFirst={scrollOffset + idx === 0}
            />
          ))
        )}
      </Box>

      {/* Scroll indicator (bottom) */}
      {canScrollDown && (
        <Box>
          <Text dimColor>
            ↓ {messageCount - scrollOffset - visibleMessages} more messages below
          </Text>
        </Box>
      )}

      {/* Footer */}
      <Box marginTop={1}>
        <Text dimColor>
          [↑/↓] scroll [PgUp/PgDn] page [g/G] top/bottom [q] back
        </Text>
      </Box>
    </Box>
  );
}

interface MessageItemProps {
  message: ChannelMessage;
  isFirst?: boolean;
}

function MessageItem({ message }: MessageItemProps) {
  const timeStr = formatTimestamp(message.time);
  const senderColor = getSenderColor(message.sender);

  return (
    <Box>
      <Box width={8}>
        <Text dimColor>{timeStr}</Text>
      </Box>
      <Box width={15}>
        <Text color={senderColor} bold>
          {truncate(message.sender, 14)}
        </Text>
      </Box>
      <Box flexGrow={1}>
        <Text wrap="truncate">{message.message}</Text>
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

/**
 * Get consistent color for a sender name
 */
function getSenderColor(sender: string): string {
  const colors = ['blue', 'green', 'yellow', 'magenta', 'cyan'];
  let hash = 0;
  for (let i = 0; i < sender.length; i++) {
    hash = sender.charCodeAt(i) + ((hash << 5) - hash);
  }
  return colors[Math.abs(hash) % colors.length];
}

/**
 * Truncate string to max length
 */
function truncate(str: string, maxLen: number): string {
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen - 1) + '…';
}

export default MessageHistory;
