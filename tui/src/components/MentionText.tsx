import React from 'react';
import { Text } from 'ink';

export interface MentionTextProps {
  text: string;
  currentUser?: string;
}

/**
 * Token types for markdown/mention parsing
 */
type TokenType = 'text' | 'bold' | 'italic' | 'code' | 'mention' | 'broadcast' | 'self-mention';

interface Token {
  type: TokenType;
  content: string;
  index: number;
}

/**
 * Parse text into tokens for markdown and mentions
 * #972 fix: Added markdown rendering support
 *
 * Supports:
 * - **bold** or __bold__
 * - *italic* or _italic_
 * - `code`
 * - @mentions
 */
function parseFormattedText(text: string, currentUser?: string): Token[] {
  const tokens: Token[] = [];

  // Combined pattern for all formatting
  // Order matters: longer patterns first to avoid partial matches
  const pattern = /(\*\*[^*]+\*\*|__[^_]+__|`[^`]+`|\*[^*]+\*|_[^_]+_|@\w+[-\w]*)/g;

  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = pattern.exec(text)) !== null) {
    // Add plain text before this match
    if (match.index > lastIndex) {
      tokens.push({
        type: 'text',
        content: text.slice(lastIndex, match.index),
        index: lastIndex,
      });
    }

    const matched = match[0];
    let type: TokenType = 'text';
    let content = matched;

    if (matched.startsWith('**') && matched.endsWith('**')) {
      type = 'bold';
      content = matched.slice(2, -2);
    } else if (matched.startsWith('__') && matched.endsWith('__')) {
      type = 'bold';
      content = matched.slice(2, -2);
    } else if (matched.startsWith('`') && matched.endsWith('`')) {
      type = 'code';
      content = matched.slice(1, -1);
    } else if ((matched.startsWith('*') && matched.endsWith('*')) ||
               (matched.startsWith('_') && matched.endsWith('_'))) {
      type = 'italic';
      content = matched.slice(1, -1);
    } else if (matched.startsWith('@')) {
      const username = matched.slice(1);
      if (username === 'all' || username === 'everyone') {
        type = 'broadcast';
      } else if (currentUser && username === currentUser) {
        type = 'self-mention';
      } else {
        type = 'mention';
      }
      content = matched;
    }

    tokens.push({ type, content, index: match.index });
    lastIndex = match.index + matched.length;
  }

  // Add remaining text
  if (lastIndex < text.length) {
    tokens.push({
      type: 'text',
      content: text.slice(lastIndex),
      index: lastIndex,
    });
  }

  return tokens;
}

/**
 * Text component that renders markdown and @mentions
 * #972 fix: Added markdown rendering support
 *
 * Markdown:
 * - **bold** or __bold__: Bold text
 * - *italic* or _italic_: Dim text
 * - `code`: Magenta colored text
 *
 * Mentions:
 * - @username: Cyan color
 * - @currentUser: Bold cyan inverse (self-mention)
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

  const tokens = parseFormattedText(text, currentUser);
  const parts: React.ReactNode[] = [];

  for (const token of tokens) {
    const key = `token-${String(token.index)}`;

    switch (token.type) {
      case 'bold':
        parts.push(
          <Text key={key} bold>
            {token.content}
          </Text>
        );
        break;
      case 'italic':
        parts.push(
          <Text key={key} dimColor>
            {token.content}
          </Text>
        );
        break;
      case 'code':
        parts.push(
          <Text key={key} color="magenta">
            {token.content}
          </Text>
        );
        break;
      case 'broadcast':
        parts.push(
          <Text key={key} color="yellow" bold>
            {token.content}
          </Text>
        );
        break;
      case 'self-mention':
        parts.push(
          <Text key={key} color="cyan" bold inverse>
            {token.content}
          </Text>
        );
        break;
      case 'mention':
        parts.push(
          <Text key={key} color="cyan">
            {token.content}
          </Text>
        );
        break;
      default:
        // Plain text - add as string without wrapping
        parts.push(token.content);
    }
  }

  return <Text wrap="wrap">{parts}</Text>;
};

export default MentionText;
