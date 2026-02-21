/**
 * ChannelsView - Channel list and message history component
 */

import React, { useState, useEffect, useMemo, useRef } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { useChannelsWithUnread, useChannelHistory, useUnread, useMentionAutocomplete } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { PulseText } from './AnimatedText';
import { ChatMessage } from './ChatMessage';
import { MentionAutocomplete } from './MentionAutocomplete';
import type { Channel } from '../types';
import type { DetailItem } from './DetailPane';

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

interface ChannelsViewProps {
  /** Disable input handling (useful for testing) */
  disableInput?: boolean;
  /** Callback when a channel is selected (for detail pane) */
  onSelectItem?: (item: DetailItem | null) => void;
}

/** Channel with unread count from useChannelsWithUnread hook */
interface ChannelWithUnread extends Channel {
  unread: number;
}

/**
 * Convert a channel to DetailItem for the detail pane (#1418)
 */
function channelToDetailItem(channel: ChannelWithUnread): DetailItem {
  const extraCount = channel.members.length - 3;
  return {
    title: `#${channel.name}`,
    type: 'channel',
    description: channel.description ?? 'No description',
    fields: [
      { label: 'Members', value: String(channel.members.length), color: 'cyan' },
      { label: 'Unread', value: String(channel.unread), color: channel.unread > 0 ? 'yellow' : undefined },
      ...channel.members.slice(0, 3).map((member, idx) => ({
        label: idx === 0 ? 'Active' : '',
        value: member,
      })),
      ...(extraCount > 0 ? [{ label: '', value: `+${String(extraCount)} more` }] : []),
    ],
  };
}

export function ChannelsView({ disableInput = false, onSelectItem }: ChannelsViewProps): React.ReactElement {
  // #1129: Use useChannelsWithUnread for proper unread message tracking
  const { channels, loading: channelsLoading, error: channelsError } = useChannelsWithUnread();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [viewMode, setViewMode] = useState<'list' | 'history'>('list');
  const { setBreadcrumbs, clearBreadcrumbs } = useNavigation();
  const { setFocus } = useFocus();

  // Update breadcrumbs and focus when view mode changes
  useEffect(() => {
    const channel = channels?.[selectedIndex];
    if (viewMode === 'history' && channel) {
      setBreadcrumbs([{ label: `#${channel.name}` }]);
    } else {
      clearBreadcrumbs();
      // Restore focus to 'main' when returning to list view
      // This must happen AFTER global ESC handler has checked focus
      setFocus('main');
    }
  }, [viewMode, channels, selectedIndex, setBreadcrumbs, clearBreadcrumbs, setFocus]);

  // Track if we should start in compose mode when entering history view
  const [startCompose, setStartCompose] = useState(false);

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
        // Vim-style top/bottom navigation
        if (input === 'g') {
          setSelectedIndex(0);
        }
        if (input === 'G' && channels) {
          setSelectedIndex(channels.length - 1);
        }
        // Enter channel - get current channel inside callback to avoid stale closure
        // This fixes #1064: Enter key not working when channels load after initial render
        const currentChannel = channels?.[selectedIndex];
        if (key.return && currentChannel) {
          setFocus('view');
          setViewMode('history');
        }
        // 'm' to compose - enter channel and start compose mode (#1316)
        if (input === 'm' && currentChannel) {
          setStartCompose(true);
          setFocus('view');
          setViewMode('history');
        }
      }
      // Note: ESC in history mode is handled by ChannelHistoryView's onBack callback
    },
    { isActive: !disableInput }
  );

  // Get currently selected channel for rendering
  const selectedChannel = channels?.[selectedIndex];

  // #1418: Update detail pane when selected channel changes
  useEffect(() => {
    if (selectedChannel && viewMode === 'list') {
      onSelectItem?.(channelToDetailItem(selectedChannel));
    } else if (!selectedChannel) {
      onSelectItem?.(null);
    }
  }, [selectedChannel, viewMode, onSelectItem]);

  if (channelsLoading) {
    return (
      <Box flexDirection="column">
        <Text bold>Channels</Text>
        <PulseText dimColor>Loading channels...</PulseText>
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
    return (
      <ChannelHistoryView
        key={selectedChannel.name}
        channel={selectedChannel}
        disableInput={disableInput}
        startInComposeMode={startCompose}
        onBack={() => {
          setViewMode('list');
          setStartCompose(false);
        }}
      />
    );
  }

  return (
    <Box flexDirection="column" width="100%">
      <Text bold>Channels</Text>
      <Text dimColor>↑/↓ navigate, Enter to view messages, ESC to go back</Text>
      <Box marginTop={1} flexDirection="column" width="100%" borderStyle="single" borderColor="gray" paddingX={2}>
        {channels?.map((channel, index) => (
          <ChannelRow
            key={channel.name}
            channel={channel}
            selected={index === selectedIndex}
            unreadCount={channel.unread}
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
  unreadCount: number;
}

function ChannelRow({ channel, selected, unreadCount }: ChannelRowProps): React.ReactElement {
  // #981 fix: Build name row as single truncated text to ensure visibility at 80 cols
  // #1129: Highlight channels with unread messages
  // #1364 Issue 2: Clarify channel numbers (unread vs members)
  // Priority: name > unread indicator > member count > description
  const namePrefix = selected ? '▸ ' : '  ';
  const channelName = `#${channel.name}`;

  // Format member count with 'm' suffix to distinguish from unread (#1364)
  const memberInfo = ` ${String(channel.members.length)}m`;

  // Format unread badge with 'new' label to clarify meaning (#1364)
  // "●" for 1 unread, "N new" for multiple
  const unreadBadge = unreadCount > 0
    ? unreadCount === 1
      ? ' ●'
      : ` ${unreadCount > 99 ? '99+' : String(unreadCount)} new`
    : '';

  // Build single text line to avoid nested Text truncation issues on narrow terminals
  // Issue #981: Nested Text elements break rendering at 80x24 width
  const nameLineText = `${namePrefix}${channelName}${unreadBadge}${memberInfo}`;

  // Determine text color: cyan if selected, yellow if has unread, default otherwise
  const textColor = selected ? 'cyan' : unreadCount > 0 ? 'yellow' : undefined;

  // #1171 fix: Remove explicit width="100%" to avoid nested width calculation issues at 80x24
  // The parent Box already has width="100%", inner Box inherits flexbox width naturally
  return (
    <Box flexDirection="column">
      {/* Name row: single Text for proper truncation at narrow widths */}
      <Text
        wrap="truncate"
        color={textColor}
        bold={selected || unreadCount > 0}
      >
        {nameLineText}
      </Text>
      {channel.description && (
        <Text dimColor wrap="truncate">{channel.description}</Text>
      )}
    </Box>
  );
}

interface ChannelHistoryViewProps {
  channel: Channel;
  disableInput?: boolean;
  onBack?: () => void;
  /** Start in compose mode immediately (#1316) */
  startInComposeMode?: boolean;
}

function ChannelHistoryView({
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

  // Mention autocomplete for @mentions
  const autocomplete = useMentionAutocomplete({
    input: messageBuffer,
    cursorPosition: messageBuffer.length,
  });

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

  // Dynamic bubble width: 80% of terminal width, min 50, max 140
  const maxBubbleWidth = Math.min(140, Math.max(50, Math.floor(terminalWidth * 0.8)));

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

        // Handle autocomplete navigation when active
        if (autocomplete.isActive) {
          if (key.upArrow) {
            autocomplete.moveUp();
            return;
          }
          if (key.downArrow) {
            autocomplete.moveDown();
            return;
          }
          // Complete mention with Tab or Enter
          if (key.tab || key.return) {
            const completed = autocomplete.complete();
            setMessageBuffer(completed);
            return;
          }
          if (key.escape) {
            // Close autocomplete, stay in input mode
            autocomplete.reset();
            return;
          }
        }

        if (key.return) {
          if (messageBuffer.trim()) {
            send(messageBuffer.trim()).catch((err: unknown) => {
              const message = err instanceof Error ? err.message : String(err);
              setSendError(`Send failed: ${message}`);
            });
            setMessageBuffer('');
            autocomplete.reset();
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
    <Box flexDirection="column" width="100%" height="100%">
      {/* Header section - fixed height */}
      <Box flexDirection="column" height={3} marginBottom={1}>
        <Box>
          <Text bold color="cyan">#{channel.name}</Text>
          <Text dimColor> - {channel.members.length} members</Text>
        </Box>
        <Text dimColor>ESC: back  m: compose  ↑/↓ or j/k: scroll</Text>
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
        {loading && <PulseText dimColor>Loading messages...</PulseText>}
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

      {/* Mention autocomplete dropdown */}
      {inputMode && (
        <MentionAutocomplete
          suggestions={autocomplete.suggestions}
          selectedIndex={autocomplete.selectedIndex}
          visible={autocomplete.isActive}
          query={autocomplete.query}
        />
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

      {/* Footer - anchored at bottom */}
      <Box height={1}>
        <Text dimColor>
          {inputMode && autocomplete.isActive
            ? '↑/↓: select  Tab: complete  Esc: close'
            : `ESC: ${inputMode ? 'save draft' : 'back'}  m: compose  @: mention  ↑/↓: scroll  Enter: send`}
        </Text>
      </Box>
    </Box>
  );
}

export default ChannelsView;
