/**
 * ChannelsView - Channel list and message history component
 * Refactored from 475 lines to ~120 lines (#1590)
 *
 * Components extracted to ./channels/:
 * - ChannelRow: Single channel row in the list
 * - ChannelHistoryView: Message history and compose view
 */

import React, { useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { useChannelsWithUnread, useDisableInput } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { PulseText } from '../components/AnimatedText';
import { ChannelRow, ChannelHistoryView } from '../components/channels';

// #1594: Using empty interface for future extensibility, props removed
// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface ChannelsViewProps {}

/**
 * ChannelsView - Main channel list component
 *
 * Features:
 * - List all channels with unread counts
 * - Keyboard navigation (j/k, Enter)
 * - Enter channel to view history
 * - Press 'm' to jump to compose
 */
export function ChannelsView(_props: ChannelsViewProps = {}): React.ReactElement {
  // #1594: Use context instead of prop drilling
  const { isDisabled: disableInput } = useDisableInput();
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
        startInComposeMode={startCompose}
        onBack={() => {
          setViewMode('list');
          setStartCompose(false);
        }}
      />
    );
  }

  // #1483 fix: Remove width="100%" to avoid layout overflow at 80 columns
  // Ink's layout calculates width incorrectly when width="100%" + padding + border
  // Let flexbox handle width naturally through flexGrow
  // #1461 fix: Removed inline hints - global footer shows view-specific hints
  return (
    <Box flexDirection="column" flexGrow={1}>
      <Text bold>Channels</Text>
      <Box marginTop={1} flexDirection="column" flexGrow={1} borderStyle="single" borderColor="gray" paddingX={1}>
        {channels?.map((channel, index) => (
          <ChannelRow
            key={channel.name}
            channel={channel}
            selected={index === selectedIndex}
            unreadCount={channel.unread}
          />
        ))}
        {(!channels || channels.length === 0) && (
          <Box flexDirection="column">
            <Text dimColor>No channels yet.</Text>
            <Text dimColor>Create one with: bc channel create &lt;name&gt;</Text>
          </Box>
        )}
      </Box>
    </Box>
  );
}

export default ChannelsView;
