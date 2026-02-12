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
  reactions?: Array<{ type: ReactionType; count: number; isOwn?: boolean }>;
  isRead?: boolean;
  isSelected?: boolean;
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
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;

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
  if (sender.startsWith('tech-lead') || sender.includes('fox') || sender.includes('eagle')) {
    return 'cyan';
  }
  if (sender.startsWith('eng-') || sender.includes('falcon')) return 'green';
  return 'white';
};

/**
 * Dense chat message component with futuristic design
 *
 * Features:
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
}) => {
  const time = formatRelativeTime(timestamp);
  const senderColor = getRoleColor(sender);

  return (
    <Box
      flexDirection="column"
      paddingX={1}
      borderStyle={isSelected ? 'single' : undefined}
      borderColor={isSelected ? 'cyan' : undefined}
    >
      {/* Header: time | sender | read status */}
      <Box>
        <Text color="gray" dimColor>
          {time}
        </Text>
        <Text> </Text>
        <Text color={senderColor} bold>
          {sender}
        </Text>
        {!isRead && (
          <Text color="blue"> ●</Text>
        )}
      </Box>

      {/* Message body with @mentions */}
      <Box paddingLeft={2}>
        <MentionText text={message} currentUser={currentUser} />
      </Box>

      {/* Reactions */}
      {reactions.length > 0 && (
        <Box paddingLeft={2}>
          <ReactionBar reactions={reactions} />
        </Box>
      )}
    </Box>
  );
};

export default ChatMessage;
