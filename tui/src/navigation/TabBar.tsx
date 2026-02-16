/**
 * TabBar - Responsive navigation tab bar component
 *
 * Display modes based on terminal width:
 * - Full (>=120 cols): [1] Dashboard [2] Agents ...
 * - Short (80-119 cols): [1] Dash [2] Agt ...
 * - Minimal (<80 cols): [1] [2] [3] ...
 */

import React, { useMemo } from 'react';
import { Box, Text, useStdout } from 'ink';
import { useNavigation } from './NavigationContext';

/** Terminal width thresholds for display modes */
const FULL_WIDTH_THRESHOLD = 120;
const SHORT_WIDTH_THRESHOLD = 80;

/** Display mode for tab labels */
type DisplayMode = 'full' | 'short' | 'minimal';

/**
 * Determine display mode based on terminal width
 */
function getDisplayMode(width: number): DisplayMode {
  if (width >= FULL_WIDTH_THRESHOLD) return 'full';
  if (width >= SHORT_WIDTH_THRESHOLD) return 'short';
  return 'minimal';
}

export interface TabBarProps {
  /** Show app title before tabs */
  showTitle?: boolean;
  /** App title text */
  title?: string;
  /** Override terminal width (for testing) */
  terminalWidth?: number;
}

export function TabBar({
  showTitle = true,
  title = 'bc',
  terminalWidth: overrideWidth,
}: TabBarProps): React.ReactElement {
  const { currentView, tabs, canGoBack } = useNavigation();
  const { stdout } = useStdout();

  // Use override width for testing, otherwise use actual terminal width
  const terminalWidth = overrideWidth ?? stdout.columns;
  const displayMode = useMemo(() => getDisplayMode(terminalWidth), [terminalWidth]);

  /**
   * Get tab label based on display mode
   */
  const getTabLabel = (tab: { key: string; label: string; shortLabel?: string }): string => {
    switch (displayMode) {
      case 'full':
        return tab.label;
      case 'short':
        return tab.shortLabel ?? tab.label;
      case 'minimal':
        return ''; // Just show key in minimal mode
    }
  };

  return (
    <Box flexShrink={0}>
      {showTitle && (
        <>
          <Text bold color="cyan">
            {title}
          </Text>
          <Text dimColor> |</Text>
        </>
      )}
      {tabs.map((tab) => {
        const isActive = currentView === tab.view;
        const label = getTabLabel(tab);

        return (
          <React.Fragment key={tab.view}>
            <Text> </Text>
            <Text
              bold={isActive}
              color={isActive ? 'green' : undefined}
              dimColor={!isActive}
            >
              [{tab.key}]{label ? ` ${label}` : ''}
            </Text>
          </React.Fragment>
        );
      })}
      {canGoBack && (
        <>
          <Text dimColor> |</Text>
          <Text dimColor> [←]{displayMode !== 'minimal' ? ' Back' : ''}</Text>
        </>
      )}
    </Box>
  );
}

export default TabBar;
