/**
 * ChannelHistoryView - Message history and compose view for a channel
 * Extracted from ChannelsView.tsx (#1590)
 */

import React, { useState, useEffect, useMemo, useRef } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { useChannelHistory, useUnread } from '../../hooks';
import { useFocus } from '../../navigation/FocusContext';
import { ChatMessage } from '../ChatMessage';
import type { Channel } from '../../types';

/** Duration in ms to show send errors before auto-clearing */
const SEND_ERROR_DISPLAY_DURATION = 3000;

/**
 * Calculate input box height based on message length
 * Height expands from 3 (min) to 10 (max) lines based on content
 */
function calculateInputHeight(messageLength: number, terminalWidth: number): number {
  const MIN_HEIGHT = 3;
  const MAX_HEIGHT = 10;
  // Account for border (2) + prompt "> " (2) + cursor (1)
  const availableWidth = Math.max(terminalWidth - 5, 20);
  const lines = Math.ceil(messageLength / availableWidth) + 1;
  return Math.min(MAX_HEIGHT, Math.max(MIN_HEIGHT, lines));
}

export interface ChannelHistoryViewProps {
  channel: Channel;
  disableInput?: boolean;
  onBack?: () => void;
  /** Start in compose mode immediately (#1316) */
  startInComposeMode?: boolean;
}

/**
 * ChannelHistoryView - Display message history and handle message composition
 *
 * Features:
 * - Message display with scroll support
 * - Compose mode with @ mention autocomplete
 * - Draft preservation on ESC
 * - Dynamic layout based on terminal size
 */
export function ChannelHistoryView({
  channel,
  disableInput = false,
  onBack,
  startInComposeMode = false,
}: ChannelHistoryViewProps): React.ReactElement {
  const { data: messages, loading, error, send } = useChannelHistory(channel.name, {
    limit: 50,
  });
  const [inputMode, setInputMode] = useState(startInComposeMode);
  // Skip first keystroke when entering via 'm' from list view (#1337/#1339)
  const skipFirstInput = useRef(startInComposeMode);
  const [messageBuffer, setMessageBuffer] = useState('');
  const [scrollOffset, setScrollOffset] = useState(0);
  const [sendError, setSendError] = useState<string | null>(null);
  const { setFocus } = useFocus();

  // Auto-clear send errors after a delay
  useEffect(() => {
    if (!sendError) return;
    const timer = setTimeout(() => { setSendError(null); }, SEND_ERROR_DISPLAY_DURATION);
    return () => { clearTimeout(timer); };
  }, [sendError]);
  const { stdout } = useStdout();
  const { markViewed } = useUnread();

  // Mark channel as viewed when messages load
  useEffect(() => {
    if (messages && messages.length >= 0) {
      markViewed(channel.name, messages.length);
    }
  }, [channel.name, messages, markViewed]);

  // Calculate dynamic input height based on message length
  const terminalWidth = stdout.columns;
  const terminalHeight = stdout.rows;
  const inputHeight = useMemo(
    () => calculateInputHeight(messageBuffer.length, terminalWidth),
    [messageBuffer.length, terminalWidth]
  );

  // Dynamic layout based on terminal size (#976)
  // CLI directive: Fix messages appearing behind input field
  // Layout breakdown: header(3+1margin) + input(inputHeight+1margin) + footer(1) + borders(4) + safety(2)
  const layoutOverhead = 4 + inputHeight + 1 + 1 + 4 + 2; // = 12 + inputHeight
  const messageAreaHeight = Math.max(8, terminalHeight - layoutOverhead);

  // Dynamic bubble width: 80% of available width, min 40, max 140
  // #1681 fix: Account for container overhead (8 cols: view border/padding + bubble border/padding)
  const containerOverhead = 8;
  const maxBubbleWidth = Math.min(140, Math.max(40, Math.floor((terminalWidth - containerOverhead) * 0.8)));

  // Dynamic message count: ~4 lines per message bubble
  const maxMessages = Math.max(3, Math.floor(messageAreaHeight / 4));

  /**
   * Synchronize focus state with input mode
   *
   * When user enters input mode (presses 'm'), we set focus to 'input' area.
   * This prevents global keybinds (q, 1-9, ESC) from triggering during message typing.
   *
   * When user exits input mode (presses Enter or Escape), we set focus to 'view'
   * to keep global navigation disabled while in channel history view. This ensures that
   * ESC navigates back to channel list (via onBack) rather than to Dashboard.
   *
   * This fixes issue #653: "After typing a message in a channel, the keybinds to
   * q, 1,2,3... are not re-enabled"
   * This also fixes issue #884: "ESC from channel history goes to Dashboard instead
   * of Channels list"
   */
  useEffect(() => {
    if (inputMode) {
      setFocus('input');
    } else {
      // Keep focus on 'view' to prevent global ESC from going to Dashboard
      setFocus('view');
    }
  }, [inputMode, setFocus]);

  useInput(
    (input, key) => {
      if (inputMode) {
        // Skip the first keystroke when entering via 'm' from list view (#1337/#1339)
        // The 'm' that triggered compose mode shouldn't appear in input
        if (skipFirstInput.current) {
          skipFirstInput.current = false;
          return;
        }

        if (key.return) {
          if (messageBuffer.trim()) {
            send(messageBuffer.trim()).catch((err: unknown) => {
              const message = err instanceof Error ? err.message : String(err);
              setSendError(`Send failed: ${message}`);
            });
            setMessageBuffer('');
          }
          setInputMode(false);
        } else if (key.escape) {
          // Draft save: preserve message on Esc for later editing
          // Only clear if message was empty (cancel vs save draft)
          setInputMode(false);
        } else if (key.backspace || key.delete) {
          setMessageBuffer(messageBuffer.slice(0, -1));
        } else if (input && !key.ctrl && !key.meta) {
          setMessageBuffer(messageBuffer + input);
        }
      } else {
        // ESC to go back to channel list
        // Note: Don't call returnFocus() here - focus must stay 'view' until
        // after global ESC handler runs, otherwise goHome() will fire
        if (key.escape) {
          onBack?.();
        }
        // 'm' to compose message
        if (input === 'm') {
          setInputMode(true);
        }
        // 'c' to clear draft
        if (input === 'c' && messageBuffer) {
          setMessageBuffer('');
        }
        // j/k and arrow keys to scroll
        // Note: Uses dynamic maxMessages calculated from terminal height (#976)
        // CLI directive: Add arrow key support for scrolling
        if ((input === 'j' || key.downArrow) && messages) {
          setScrollOffset(Math.max(0, scrollOffset - 1));
        }
        if ((input === 'k' || key.upArrow) && messages) {
          setScrollOffset(Math.min(Math.max(0, messages.length - maxMessages), scrollOffset + 1));
        }
      }
    },
    { isActive: !disableInput }
  );

  // #976 fix: Dynamic message display based on terminal height
  // maxMessages is calculated above based on available messageAreaHeight
  const displayMessages = messages ? messages.slice(Math.max(0, messages.length - maxMessages - scrollOffset), messages.length - scrollOffset) : [];
  const hasMoreAbove = scrollOffset > 0;
  const hasMoreBelow = messages && messages.length > maxMessages && scrollOffset < messages.length - maxMessages;

  return (
    // #1425 fix: Use flexGrow instead of height="100%" to prevent layout overflow
    <Box flexDirection="column" width="100%" flexGrow={1} overflow="hidden">
      {/* Header section - #1461 fix: Removed duplicate hints (shown in footer) */}
      <Box flexDirection="column" height={2} marginBottom={1}>
        <Box>
          <Text bold color="cyan">#{channel.name}</Text>
          <Text dimColor> - {channel.members.length} members</Text>
        </Box>
      </Box>

      {/* Message area - dynamic height adjusts as input expands */}
      <Box
        marginBottom={1}
        flexDirection="column"
        height={messageAreaHeight}
        borderStyle="single"
        borderColor="gray"
        paddingX={1}
        overflow="hidden"
      >
        {loading && <Text dimColor>Loading messages...</Text>}
        {error && <Text color="red">Error: {error}</Text>}
        {!loading && !error && (
          <>
            {hasMoreAbove && <Text dimColor>↑ more messages above</Text>}
            {displayMessages.map((msg, index) => (
              <ChatMessage
                key={`${msg.time}-${String(index)}`}
                sender={msg.sender}
                message={msg.message}
                timestamp={msg.time}
                currentUser={process.env.BC_AGENT_ID}
                maxBubbleWidth={maxBubbleWidth}
              />
            ))}
            {hasMoreBelow && <Text dimColor>↓ more messages below</Text>}
            {messages?.length === 0 && <Text dimColor>No messages yet</Text>}
          </>
        )}
      </Box>

      {/* Send error feedback */}
      {sendError && (
        <Box marginBottom={1}>
          <Text color="red">{sendError}</Text>
        </Box>
      )}

      {/* Input area - auto-expands based on message length (3-10 lines) */}
      <Box height={inputHeight} flexDirection="column" marginBottom={1} borderStyle="single" borderColor={inputMode ? 'cyan' : (messageBuffer ? 'yellow' : 'gray')} paddingX={1}>
        {inputMode ? (
          <Text>
            <Text color="cyan">{'> '}</Text>
            {messageBuffer}
            <Text color="cyan">▌</Text>
          </Text>
        ) : messageBuffer ? (
          <Text>
            <Text color="yellow">[Draft] </Text>
            <Text dimColor>{messageBuffer.length > 40 ? messageBuffer.slice(0, 40) + '...' : messageBuffer}</Text>
            <Text dimColor> (press m to edit)</Text>
          </Text>
        ) : (
          <Text dimColor>Press m to compose message</Text>
        )}
      </Box>

      {/* #1461 fix: Removed duplicate footer - global footer shows navigation hints */}
    </Box>
  );
}

export default ChannelHistoryView;
