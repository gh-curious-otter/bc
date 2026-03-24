/**
 * ChannelsView - Channel list and message history component
 * Refactored from 475 lines to ~120 lines (#1590)
 *
 * Components extracted to ./channels/:
 * - ChannelRow: Single channel row in the list
 * - ChannelHistoryView: Message history and compose view
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../theme';
import { useChannelsWithUnread, useDisableInput, useListNavigation } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { ErrorDisplay } from '../components/ErrorDisplay';
import { HeaderBar } from '../components/HeaderBar';
import { Footer } from '../components/Footer';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { ChannelRow, ChannelHistoryView } from '../components/channels';
import type { Channel } from '../types';

// #1737: Type for channel with unread info from useChannelsWithUnread
type ChannelWithUnread = Channel & { unread: number; messageCount: number };

// #1594: Using type alias for future extensibility, props removed
type ChannelsViewProps = Record<string, never>;

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
  const { theme } = useTheme();
  const { isDisabled: disableInput } = useDisableInput();
  // #1129: Use useChannelsWithUnread for proper unread message tracking
  const {
    channels,
    loading: channelsLoading,
    error: channelsError,
    refresh,
  } = useChannelsWithUnread();
  const [viewMode, setViewMode] = useState<'list' | 'history'>('list');
  const { setBreadcrumbs, clearBreadcrumbs } = useNavigation();
  const { setFocus } = useFocus();

  // Track if we should start in compose mode when entering history view
  const [startCompose, setStartCompose] = useState(false);

  // Channel list for navigation
  const channelList = useMemo(() => channels ?? [], [channels]);

  // #1737: Handle channel selection (Enter key)
  const handleSelect = useCallback(
    (_channel: ChannelWithUnread) => {
      setFocus('view');
      setViewMode('history');
    },
    [setFocus]
  );

  // #1737: Custom key handlers
  const customKeys = useMemo(
    () => ({
      // 'm' to compose - enter channel and start compose mode (#1316)
      m: () => {
        if (channelList.length > 0) {
          setStartCompose(true);
          setFocus('view');
          setViewMode('history');
        }
      },
      r: () => {
        void refresh();
      },
    }),
    [channelList.length, setFocus, refresh]
  );

  // #1737: Use useListNavigation for keyboard handling
  const { selectedIndex, selectedItem: selectedChannel } = useListNavigation({
    items: channelList,
    onSelect: handleSelect,
    disabled: disableInput || viewMode !== 'list',
    customKeys,
  });

  // Update breadcrumbs and focus when view mode changes
  useEffect(() => {
    if (viewMode === 'history' && selectedChannel) {
      setBreadcrumbs([{ label: `#${selectedChannel.name}` }]);
    } else {
      clearBreadcrumbs();
      // Restore focus to 'main' when returning to list view
      // This must happen AFTER global ESC handler has checked focus
      setFocus('main');
    }
  }, [viewMode, selectedChannel, setBreadcrumbs, clearBreadcrumbs, setFocus]);

  if (channelsLoading && channelList.length === 0) {
    return <LoadingIndicator message="Loading channels..." />;
  }

  if (channelsError) {
    return (
      <ErrorDisplay
        error={channelsError}
        onRetry={() => {
          void refresh();
        }}
      />
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

  // #1890: Redesigned with HeaderBar, table layout, Footer
  return (
    <Box flexDirection="column" width="100%" overflow="hidden">
      <HeaderBar
        title="Channels"
        count={channelList.length}
        loading={channelsLoading}
        color={theme.colors.primary}
      />

      {/* Channel table */}
      <Box flexDirection="column" marginBottom={1}>
        {/* Column headers */}
        <Box paddingX={1}>
          <Box width={24}>
            <Text bold dimColor>
              CHANNEL
            </Text>
          </Box>
          <Box width={12}>
            <Text bold dimColor>
              UNREAD
            </Text>
          </Box>
          <Box width={10}>
            <Text bold dimColor>
              MEMBERS
            </Text>
          </Box>
          <Box flexGrow={1}>
            <Text bold dimColor>
              DESCRIPTION
            </Text>
          </Box>
        </Box>

        {/* Channel rows */}
        {channelList.length === 0 ? (
          <Box paddingX={1} marginTop={1} flexDirection="column">
            <Text dimColor>No channels yet.</Text>
            <Text dimColor>Create one with: bc channel create &lt;name&gt;</Text>
          </Box>
        ) : (
          channelList.map((channel, index) => (
            <ChannelRow
              key={channel.name}
              channel={channel}
              selected={index === selectedIndex}
              unreadCount={channel.unread}
            />
          ))
        )}
      </Box>

      {/* Footer */}
      <Footer
        hints={[
          { key: 'j/k', label: 'nav' },
          { key: 'g/G', label: 'top/bottom' },
          { key: 'Enter', label: 'open' },
          { key: 'm', label: 'compose' },
          { key: 'r', label: 'refresh' },
        ]}
      />
    </Box>
  );
}

export default ChannelsView;
