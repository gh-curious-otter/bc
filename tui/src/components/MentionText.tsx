import React from 'react';
import { Text } from 'ink';

export interface MentionTextProps {
  text: string;
  currentUser?: string;
}

/**
 * Text component that highlights @mentions
 *
 * - @username: Cyan color
 * - @currentUser: Bold cyan (self-mention)
 * - @all/@everyone: Yellow (broadcast)
 */
export const MentionText: React.FC<MentionTextProps> = ({
  text,
  currentUser,
}) => {
  // Handle empty, missing, or whitespace-only text
  if (!text || text.trim().length === 0) {
    return <Text dimColor>(empty)</Text>;
  }

  // Pattern to match @mentions
  const mentionPattern = /@(\w+[-\w]*)/g;
  const parts: React.ReactNode[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = mentionPattern.exec(text)) !== null) {
    // Add text before the mention
    if (match.index > lastIndex) {
      parts.push(
        <Text key={`text-${String(lastIndex)}`}>
          {text.slice(lastIndex, match.index)}
        </Text>
      );
    }

    const mention = match[0];
    const username = match[1];

    // Determine mention type and styling
    const isSelfMention = currentUser && username === currentUser;
    const isBroadcast = username === 'all' || username === 'everyone';

    if (isBroadcast) {
      parts.push(
        <Text key={`mention-${String(match.index)}`} color="yellow" bold>
          {mention}
        </Text>
      );
    } else if (isSelfMention) {
      parts.push(
        <Text key={`mention-${String(match.index)}`} color="cyan" bold inverse>
          {mention}
        </Text>
      );
    } else {
      parts.push(
        <Text key={`mention-${String(match.index)}`} color="cyan">
          {mention}
        </Text>
      );
    }

    lastIndex = match.index + mention.length;
  }

  // Add remaining text
  if (lastIndex < text.length) {
    parts.push(
      <Text key={`text-${String(lastIndex)}`}>
        {text.slice(lastIndex)}
      </Text>
    );
  }

  return <Text wrap="wrap">{parts}</Text>;
};

export default MentionText;
