import React, { memo } from 'react';
import { Text } from 'ink';

export type ReactionType = 'ack' | 'plus' | 'check' | 'thumbsup' | 'heart';

export interface ReactionProps {
  type: ReactionType;
  count?: number;
  isOwn?: boolean;
}

const reactionEmoji: Record<ReactionType, string> = {
  ack: '✓',
  plus: '➕',
  check: '✅',
  thumbsup: '👍',
  heart: '❤️',
};

const reactionColors: Record<ReactionType, string> = {
  ack: 'green',
  plus: 'green',
  check: 'green',
  thumbsup: 'yellow',
  heart: 'red',
};

/**
 * Reaction component for channel messages
 * Displays emoji with optional count
 *
 * Memoized for performance - Issue #1003 Phase 3 optimization.
 */
export const Reaction = memo(function Reaction({
  type,
  count = 1,
  isOwn = false,
}: ReactionProps) {
  const emoji = reactionEmoji[type] || '❓';
  const color = isOwn ? 'cyan' : reactionColors[type] || 'white';

  return (
    <Text color={color}>
      {emoji}
      {count > 1 && <Text dimColor> {count}</Text>}
    </Text>
  );
});

export interface ReactionBarProps {
  reactions: { type: ReactionType; count: number; isOwn?: boolean }[];
}

/**
 * Bar of reactions for a message
 *
 * Memoized for performance - Issue #1003 Phase 3 optimization.
 */
export const ReactionBar = memo(function ReactionBar({ reactions }: ReactionBarProps) {
  if (reactions.length === 0) return null;

  return (
    <Text>
      {reactions.map((r, i) => (
        <React.Fragment key={r.type}>
          {i > 0 && ' '}
          <Reaction type={r.type} count={r.count} isOwn={r.isOwn} />
        </React.Fragment>
      ))}
    </Text>
  );
});

export default Reaction;
