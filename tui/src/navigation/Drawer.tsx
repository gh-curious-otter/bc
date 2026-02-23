/**
 * Drawer - Vertical navigation drawer component
 *
 * Issue #1345: Visual overhaul with grouped sections
 * - Sections: WORKSPACE, MONITORING, SYSTEM
 * - Visual indicators: ● selected, ○ unselected
 * - Header with branding
 * - Footer with help shortcut
 *
 * Issue #1289: TUI revamp with drawer layout
 */

import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import { useNavigation, type TabConfig } from './NavigationContext';
import { useFocus } from './FocusContext';

/** Fixed width for drawer panel */
const DRAWER_WIDTH = 16;
/** Shrunk width for narrow terminals (80-99 cols) */
const DRAWER_SHRUNK_WIDTH = 8;
/** Width threshold to use short labels (LG+ mode gets full labels at 14-char width) */
const DRAWER_SHORT_LABEL_THRESHOLD = 14;

/** Section definitions for grouped navigation */
interface DrawerSection {
  title: string;
  views: string[];
}

const DRAWER_SECTIONS: DrawerSection[] = [
  {
    title: 'WORKSPACE',
    views: ['dashboard', 'agents', 'channels', 'files', 'commands'],
  },
  {
    title: 'MONITOR',
    views: ['logs', 'costs', 'processes', 'demons', 'performance'],
  },
  {
    title: 'SYSTEM',
    views: ['roles', 'worktrees', 'workspaces', 'memory', 'routing'],
  },
];

export interface DrawerProps {
  /** Title displayed at top of drawer */
  title?: string;
  /** Disable keyboard handling */
  disabled?: boolean;
  /** Shrink drawer for narrow terminals */
  shrunk?: boolean;
  /** Version string for footer */
  version?: string;
  /** Actual drawer width from responsive layout (#1364) */
  width?: number;
}

export function Drawer({
  title = 'bc',
  disabled = false,
  shrunk = false,
  version = 'v2',
  width: propWidth,
}: DrawerProps): React.ReactElement {
  const { currentView, tabs, navigate } = useNavigation();
  const { isFocused } = useFocus();

  // Filter out help tab - it goes in footer
  const mainTabs = tabs.filter(t => t.view !== 'help');
  const helpTab = tabs.find(t => t.view === 'help');

  const [highlightedIndex, setHighlightedIndex] = useState(() =>
    mainTabs.findIndex(t => t.view === currentView)
  );

  // Determine width based on shrunk state or prop (#1364)
  const width = propWidth ?? (shrunk ? DRAWER_SHRUNK_WIDTH : DRAWER_WIDTH);
  // Use short labels when width is constrained (#1364)
  const useShortLabel = width < DRAWER_SHORT_LABEL_THRESHOLD;

  // Handle keyboard navigation within drawer
  useInput(
    (input, key) => {
      // Don't handle keys when in input mode or view mode (#1680)
      // When focus is 'view', a nested view (like ChannelHistoryView) handles j/k
      if (isFocused('input') || isFocused('view')) {
        return;
      }

      // j or down arrow: move highlight down
      if (input === 'j' || key.downArrow) {
        setHighlightedIndex(prev => Math.min(prev + 1, mainTabs.length - 1));
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
        setHighlightedIndex(mainTabs.length - 1);
        return;
      }

      // Enter: select highlighted item
      if (key.return) {
        navigate(mainTabs[highlightedIndex].view);
        return;
      }
    },
    { isActive: !disabled && mainTabs.length > 0 }
  );

  // Sync highlight with current view when navigation happens externally
  React.useEffect(() => {
    const idx = mainTabs.findIndex(t => t.view === currentView);
    if (idx >= 0 && idx !== highlightedIndex) {
      setHighlightedIndex(idx);
    }
  }, [currentView, mainTabs, highlightedIndex]);

  // Shrunk mode: minimal display
  if (shrunk) {
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
        <Text bold color="cyan">{title}</Text>
        <Box marginTop={1} flexDirection="column">
          {mainTabs.map((tab, index) => (
            <DrawerItemShrunk
              key={tab.view}
              tab={tab}
              isActive={currentView === tab.view}
              isHighlighted={index === highlightedIndex}
            />
          ))}
        </Box>
      </Box>
    );
  }

  // Full drawer with sections
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
      {/* Header */}
      <Box>
        <Text bold color="cyan">{title}</Text>
        <Text dimColor> {version}</Text>
      </Box>
      <Text dimColor>{'━'.repeat(width - 2)}</Text>

      {/* Grouped sections */}
      <Box flexDirection="column" marginTop={1} flexGrow={1}>
        {DRAWER_SECTIONS.map((section) => {
          const sectionTabs = mainTabs.filter(t => section.views.includes(t.view));
          if (sectionTabs.length === 0) return null;

          return (
            <Box key={section.title} flexDirection="column" marginBottom={1}>
              {/* Section header - #1501 fix: truncate to prevent overflow */}
              <Text dimColor bold wrap="truncate">{section.title}</Text>
              {/* Section items */}
              {sectionTabs.map(tab => {
                const globalIndex = mainTabs.findIndex(t => t.view === tab.view);
                return (
                  <DrawerItem
                    key={tab.view}
                    tab={tab}
                    isActive={currentView === tab.view}
                    isHighlighted={globalIndex === highlightedIndex}
                    useShortLabel={useShortLabel}
                  />
                );
              })}
            </Box>
          );
        })}
      </Box>

      {/* Footer separator */}
      <Text dimColor>{'─'.repeat(width - 2)}</Text>

      {/* Help in footer */}
      {helpTab && (
        <Box marginTop={1}>
          <Text color={currentView === 'help' ? 'green' : undefined}>
            {currentView === 'help' ? '●' : '○'}
          </Text>
          <Text
            color={currentView === 'help' ? 'green' : undefined}
            dimColor={currentView !== 'help'}
          >
            {' '}Help
          </Text>
          <Text dimColor> ?</Text>
        </Box>
      )}
    </Box>
  );
}

interface DrawerItemProps {
  tab: TabConfig;
  isActive: boolean;
  isHighlighted: boolean;
  /** Use short label for constrained widths (#1364) */
  useShortLabel?: boolean;
}

function DrawerItem({ tab, isActive, isHighlighted, useShortLabel = false }: DrawerItemProps): React.ReactElement {
  // Issue #1467: Visual indicators with highlight arrow
  // ▶ highlighted (yellow), ● selected (green), ○ inactive (dim)
  const indicator = isHighlighted ? '▶' : isActive ? '●' : '○';

  // Determine text styling
  const textColor = isActive ? 'green' : isHighlighted ? 'yellow' : undefined;
  const isBold = isActive || isHighlighted;
  const isDim = !isActive && !isHighlighted;

  // Use shortLabel when width is constrained (#1364)
  const label = useShortLabel && tab.shortLabel ? tab.shortLabel : tab.label;

  // #1501 fix: Use wrap="truncate" to prevent text overflow and rendering artifacts
  return (
    <Box>
      <Text color={isHighlighted ? 'yellow' : isActive ? 'green' : undefined}>{indicator}</Text>
      <Text
        bold={isBold}
        color={textColor}
        dimColor={isDim}
        wrap="truncate"
      >
        {' '}{label}
      </Text>
    </Box>
  );
}

interface DrawerItemShrunkProps {
  tab: TabConfig;
  isActive: boolean;
  isHighlighted: boolean;
}

function DrawerItemShrunk({ tab, isActive, isHighlighted }: DrawerItemShrunkProps): React.ReactElement {
  // Issue #1467: Use arrow indicator for highlighted items
  const indicator = isHighlighted ? '▶' : isActive ? '●' : '○';
  const textColor = isActive ? 'green' : isHighlighted ? 'yellow' : undefined;

  // Use first letter as minimal label (number shortcuts removed)
  const label = tab.label.charAt(0);

  return (
    <Box>
      <Text color={isHighlighted ? 'yellow' : isActive ? 'green' : undefined}>{indicator}</Text>
      <Text color={textColor} dimColor={!isActive && !isHighlighted}>{label}</Text>
    </Box>
  );
}

export default Drawer;
