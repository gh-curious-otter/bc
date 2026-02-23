import { memo } from 'react';
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
  /** Maximum lines to display before truncating (default: unlimited, set 0 for no limit) */
  maxLines?: number;
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
 * Get role color for sender name styling
 * CLI directive: Improve name theming with consistent color scheme
 */
const getRoleColor = (sender: string): string => {
  // Root agent - special magenta
  if (sender === 'root') return 'magenta';
  // Tech leads - cyan
  if (sender.startsWith('tech-lead') || sender.startsWith('tl-')) return 'cyan';
  // Engineers - green
  if (sender.startsWith('eng-')) return 'green';
  // Managers and PMs - yellow
  if (sender.startsWith('mgr-') || sender.startsWith('pm-')) return 'yellow';
  // UX team - blue
  if (sender.startsWith('ux-')) return 'blue';
  // QA - red
  if (sender.startsWith('qa-')) return 'red';
  // CLI/system messages - gray
  if (sender === 'cli' || sender === 'system') return 'gray';
  // Default - white
  return 'white';
};

/**
 * Get role prefix emoji for visual distinction
 */
const getRolePrefix = (sender: string): string => {
  if (sender === 'root') return '⚙ ';
  if (sender.startsWith('tl-') || sender.startsWith('tech-lead')) return '🔧 ';
  if (sender.startsWith('eng-')) return '💻 ';
  if (sender.startsWith('mgr-')) return '📋 ';
  if (sender.startsWith('pm-')) return '📊 ';
  if (sender.startsWith('ux-')) return '🎨 ';
  if (sender.startsWith('qa-')) return '🧪 ';
  if (sender === 'cli') return '⌨ ';
  return '';
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
}) {
  const time = formatRelativeTime(timestamp);
  const senderColor = getRoleColor(sender);
  const rolePrefix = getRolePrefix(sender);
  const isOwnMessage = currentUser !== undefined && sender === currentUser;

  // #1463: Truncate long messages only if maxLines > 0
  // #1718: Changed default to 0 (no limit) to show full message content
  const lines = message.split('\n');
  const isTruncated = maxLines > 0 && lines.length > maxLines;
  const displayMessage = isTruncated
    ? lines.slice(0, maxLines).join('\n')
    : message;

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
