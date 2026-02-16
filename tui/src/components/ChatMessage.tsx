import React from 'react';
import { Box, Text } from 'ink';
import { MentionText } from './MentionText';
import { ReactionBar } from './Reaction';
import type { ReactionType } from './Reaction';

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

const getRoleColor = (sender: string): string => {
  if (sender === 'root') return 'magenta';
  if (sender.startsWith('tech-lead') || sender.startsWith('tl-') || sender.includes('fox') || sender.includes('eagle')) {
    return 'cyan';
  }
  if (sender.startsWith('eng-') || sender.includes('falcon')) return 'green';
  if (sender.startsWith('mgr-') || sender.startsWith('pm-')) return 'yellow';
  if (sender.startsWith('ux-')) return 'blue';
  return 'white';
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
 */
export const ChatMessage: React.FC<ChatMessageProps> = ({
  sender,
  message,
  timestamp,
  currentUser,
  reactions = [],
  isRead = true,
  isSelected = false,
  maxBubbleWidth = 60,
}) => {
  const time = formatRelativeTime(timestamp);
  const senderColor = getRoleColor(sender);
  const isOwnMessage = currentUser !== undefined && sender === currentUser;

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
        {/* Message bubble */}
        <Box
          flexDirection="column"
          borderStyle={isSelected ? 'double' : 'round'}
          borderColor={isSelected ? 'yellow' : bubbleBorderColor}
          paddingX={1}
          width={maxBubbleWidth}
        >
          {/* Header: sender | time | read status */}
          <Box justifyContent="space-between">
            <Box>
              <Text color={senderColor} bold>
                {sender}
              </Text>
              {isOwnMessage && (
                <Text dimColor> (you)</Text>
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

          {/* Message body with @mentions - #915 fix: use flexGrow+minHeight instead of width */}
          <Box flexGrow={1} minHeight={1}>
            <MentionText text={message} currentUser={currentUser} />
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
};

export default ChatMessage;
