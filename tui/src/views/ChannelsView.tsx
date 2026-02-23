/**
 * ChannelsView - Channel list and message history component
 * Refactored from 475 lines to ~120 lines (#1590)
 *
 * Components extracted to ./channels/:
 * - ChannelRow: Single channel row in the list
 * - ChannelHistoryView: Message history and compose view
 */

import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { Box, Text } from 'ink';
import { useChannelsWithUnread, useDisableInput, useListNavigation } from '../hooks';
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
  const [viewMode, setViewMode] = useState<'list' | 'history'>('list');
  const { setBreadcrumbs, clearBreadcrumbs } = useNavigation();
  const { setFocus } = useFocus();

  // Track if we should start in compose mode when entering history view
  const [startCompose, setStartCompose] = useState(false);

  // Ref for accessing current channels in handlers
  const channelsRef = useRef(channels);
  const selectedIndexRef = useRef(0);
  useEffect(() => { channelsRef.current = channels; }, [channels]);

  // Enter channel handler
  const handleEnterChannel = useCallback(() => {
    const channel = channelsRef.current?.[selectedIndexRef.current];
    if (channel) {
      setFocus('view');
      setViewMode('history');
    }
  }, [setFocus]);

  // Compose handler - enter channel in compose mode (#1316)
  const handleCompose = useCallback(() => {
    const channel = channelsRef.current?.[selectedIndexRef.current];
    if (channel) {
      setStartCompose(true);
      setFocus('view');
      setViewMode('history');
    }
  }, [setFocus]);

  // #1737: Use useListNavigation hook for keyboard navigation
  const customKeys = useMemo(() => ({
    m: handleCompose,
  }), [handleCompose]);

  const { selectedIndex } = useListNavigation({
    items: channels ?? [],
    disabled: disableInput || viewMode !== 'list',
    onSelect: handleEnterChannel,
    customKeys,
  });

  // Keep ref in sync
  useEffect(() => { selectedIndexRef.current = selectedIndex; }, [selectedIndex]);

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
