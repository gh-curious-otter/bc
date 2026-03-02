import { memo } from 'react';
import { Box, Text } from 'ink';
import { MentionText } from './MentionText';
import { ReactionBar } from './Reaction';
import type { ReactionType } from './Reaction';
import { getColorForName, getEmojiForName } from '../constants/colors.js';

export interface ChatMessageProps {
  sender: string;
  message: string;
  timestamp: string;
  currentUser?: string;
  reactions?: { type: ReactionType; count: number; isOwn?: boolean }[];
  isRead?: boolean;
  isSelected?: boolean;
  /** Maximum width for message bubbles (default: 60) */
  maxBubbleWidth?: number;
  /** Maximum lines to display before truncating (default: unlimited, set 0 for no limit) */
  maxLines?: number;
  /** #1899: Compact mode — no bubble borders, flat layout for narrow terminals */
  compact?: boolean;
}

const formatRelativeTime = (timestamp: string): string => {
  try {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'now';
    if (diffMins < 60) return `${String(diffMins)}m ago`;
    if (diffHours < 24) return `${String(diffHours)}h ago`;
    if (diffDays < 7) return `${String(diffDays)}d ago`;

    // For older messages, show date
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
    });
  } catch {
    return timestamp;
  }
};


/**
 * Chat message component with bubble styling
 *
 * Features:
 * - Message bubbles with visual distinction
 * - Own messages aligned right, others aligned left
 * - Colored sender by role
 * - Time in compact format
 * - @mention highlighting
 * - Reaction display
 * - Read receipts
 *
 * Memoized for performance - Issue #1003 Phase 3 optimization.
 */
export const ChatMessage = memo<ChatMessageProps>(function ChatMessage({
  sender,
  message,
  timestamp,
  currentUser,
  reactions = [],
  isRead = true,
  isSelected = false,
  maxBubbleWidth = 60,
  maxLines = 0, // #1718: Default to no truncation for full message visibility
  compact = false, // #1899: Flat layout for narrow terminals
}) {
  const time = formatRelativeTime(timestamp);
  const senderColor = getColorForName(sender);
  const rolePrefix = getEmojiForName(sender);
  const isOwnMessage = currentUser !== undefined && sender === currentUser;

  // #1463: Truncate long messages only if maxLines > 0
  // #1718: Changed default to 0 (no limit) to show full message content
  const lines = message.split('\n');
  const isTruncated = maxLines > 0 && lines.length > maxLines;
  const displayMessage = isTruncated
    ? lines.slice(0, maxLines).join('\n')
    : message;

  // #1899: Compact mode — no bubble borders, sender + time on one line, message below
  // Used at narrow terminals (<100 cols) to avoid border corruption and wasted space
  if (compact) {
    return (
      <Box flexDirection="column" width="100%" paddingX={1} marginBottom={1}>
        <Box>
          <Text color={senderColor} bold>{rolePrefix}{sender}</Text>
          {isOwnMessage && <Text color="cyan" dimColor> (you)</Text>}
          <Text dimColor>  {time}</Text>
          {!isRead && <Text color="blue"> ●</Text>}
        </Box>
        <Box paddingLeft={2} flexDirection="column">
          <MentionText text={displayMessage} currentUser={currentUser} />
          {isTruncated && (
            <Text dimColor>... ({lines.length - maxLines} more lines)</Text>
          )}
        </Box>
        {reactions.length > 0 && (
          <Box paddingLeft={2}>
            <ReactionBar reactions={reactions} />
          </Box>
        )}
      </Box>
    );
  }

  // Bubble styling based on ownership
  const bubbleBorderColor = isOwnMessage ? 'cyan' : 'gray';
  const bubbleAlignment = isOwnMessage ? 'flex-end' : 'flex-start';

  return (
    <Box
      flexDirection="column"
      width="100%"
      marginY={0}
    >
      {/* Message container with alignment */}
      <Box
        justifyContent={bubbleAlignment}
        width="100%"
      >
        {/* Message bubble - #1589 fix: Add overflow="hidden" to prevent text bleeding */}
        <Box
          flexDirection="column"
          borderStyle={isSelected ? 'double' : 'round'}
          borderColor={isSelected ? 'yellow' : bubbleBorderColor}
          paddingX={1}
          width={maxBubbleWidth}
          overflow="hidden"
        >
          {/* Header: sender | time | read status - #1589 fix: Add overflow="hidden" */}
          <Box justifyContent="space-between" overflow="hidden">
            <Box>
              <Text color={senderColor} bold>
                {rolePrefix}{sender}
              </Text>
              {isOwnMessage && (
                <Text color="cyan" dimColor> (you)</Text>
              )}
            </Box>
            <Box>
              <Text color="gray" dimColor>
                {time}
              </Text>
              {!isRead && (
                <Text color="blue"> ●</Text>
              )}
            </Box>
          </Box>

          {/* Message body with @mentions
              CLI directive: Fix long message rendering - ensure text wraps properly
              Use width constraint to force text wrapping within bubble
              #1589 fix: Add overflow="hidden" to prevent text bleeding artifacts */}
          <Box flexDirection="column" flexGrow={1} minHeight={1} width={maxBubbleWidth - 4} overflow="hidden">
            <MentionText text={displayMessage} currentUser={currentUser} />
            {/* #1463: Show truncation indicator for long messages */}
            {isTruncated && (
              <Text dimColor>... ({lines.length - maxLines} more lines)</Text>
            )}
          </Box>

          {/* Reactions */}
          {reactions.length > 0 && (
            <Box marginTop={0}>
              <ReactionBar reactions={reactions} />
            </Box>
          )}
        </Box>
      </Box>
    </Box>
  );
});

export default ChatMessage;
