/**
 * Drawer - Vertical navigation drawer component
 *
 * A 14-character fixed-width left panel with:
 * - j/k for vim-style navigation
 * - Enter to select
 * - Number keys for quick jump
 * - Active view indicator (triangular marker)
 *
 * Issue #1289: TUI revamp with drawer layout
 */

import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import { useNavigation, type TabConfig } from './NavigationContext';
import { useFocus } from './FocusContext';

/** Fixed width for drawer panel */
const DRAWER_WIDTH = 14;
/** Shrunk width for narrow terminals */
const DRAWER_SHRUNK_WIDTH = 10;

export interface DrawerProps {
  /** Title displayed at top of drawer */
  title?: string;
  /** Disable keyboard handling */
  disabled?: boolean;
  /** Shrink drawer for narrow terminals */
  shrunk?: boolean;
}

export function Drawer({
  title = 'bc v2',
  disabled = false,
  shrunk = false,
}: DrawerProps): React.ReactElement {
  const { currentView, tabs, navigate } = useNavigation();
  const { isFocused } = useFocus();
  const [highlightedIndex, setHighlightedIndex] = useState(() =>
    tabs.findIndex(t => t.view === currentView)
  );

  // Determine width based on shrunk state
  const width = shrunk ? DRAWER_SHRUNK_WIDTH : DRAWER_WIDTH;

  // Handle keyboard navigation within drawer
  useInput(
    (input, key) => {
      // Don't handle keys when in input mode
      if (isFocused('input')) {
        return;
      }

      // j or down arrow: move highlight down
      if (input === 'j' || key.downArrow) {
        setHighlightedIndex(prev => Math.min(prev + 1, tabs.length - 1));
        return;
      }

      // k or up arrow: move highlight up
      if (input === 'k' || key.upArrow) {
        setHighlightedIndex(prev => Math.max(prev - 1, 0));
        return;
      }

      // g: jump to first
      if (input === 'g') {
        setHighlightedIndex(0);
        return;
      }

      // G: jump to last
      if (input === 'G') {
        setHighlightedIndex(tabs.length - 1);
        return;
      }

      // Enter: select highlighted item
      if (key.return) {
        navigate(tabs[highlightedIndex].view);
        return;
      }
    },
    { isActive: !disabled && tabs.length > 0 }
  );

  // Sync highlight with current view when navigation happens externally
  React.useEffect(() => {
    const idx = tabs.findIndex(t => t.view === currentView);
    if (idx >= 0 && idx !== highlightedIndex) {
      setHighlightedIndex(idx);
    }
  }, [currentView, tabs, highlightedIndex]);

  return (
    <Box
      flexDirection="column"
      width={width}
      borderStyle="single"
      borderRight
      borderTop={false}
      borderBottom={false}
      borderLeft={false}
      paddingRight={1}
    >
      {/* Drawer title */}
      <Box marginBottom={1}>
        <Text bold color="cyan">{shrunk ? 'bc' : title}</Text>
      </Box>
      <Box>
        <Text dimColor>{'─'.repeat(width - 2)}</Text>
      </Box>

      {/* Navigation items */}
      <Box flexDirection="column" marginTop={1}>
        {tabs.map((tab, index) => (
          <DrawerItem
            key={tab.view}
            tab={tab}
            isActive={currentView === tab.view}
            isHighlighted={index === highlightedIndex}
            shrunk={shrunk}
          />
        ))}
      </Box>
    </Box>
  );
}

interface DrawerItemProps {
  tab: TabConfig;
  isActive: boolean;
  isHighlighted: boolean;
  shrunk?: boolean;
}

function DrawerItem({ tab, isActive, isHighlighted, shrunk = false }: DrawerItemProps): React.ReactElement {
  // Use triangular marker for active view
  const marker = isActive ? '▸' : ' ';

  // Short label for compact display, truncate more when shrunk
  const fullLabel = tab.shortLabel ?? tab.label;
  const maxLen = shrunk ? 6 : 12;
  const label = fullLabel.length > maxLen
    ? fullLabel.substring(0, maxLen - 1) + '…'
    : fullLabel;

  // Determine text styling
  const textColor = isActive ? 'green' : isHighlighted ? 'yellow' : undefined;
  const isBold = isActive || isHighlighted;
  const isDim = !isActive && !isHighlighted;

  return (
    <Box>
      <Text color={isActive ? 'green' : undefined}>{marker}</Text>
      <Text
        bold={isBold}
        color={textColor}
        dimColor={isDim}
      >
        {label}
      </Text>
    </Box>
  );
}

export default Drawer;
