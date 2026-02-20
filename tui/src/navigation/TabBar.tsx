/**
 * TabBar - Responsive navigation tab bar component
 *
 * Display modes based on terminal width:
 * - Full (>=120 cols): [1] Dashboard [2] Agents ... (~140 cols needed)
 * - Short (100-119 cols): [1] Dash [2] Agt ... (~105 cols needed)
 * - Minimal (<100 cols): [1] [2] [3] ... (~55 cols needed, fits 80x24)
 *
 * Issue #1109: Fixed 80x24 display by using minimal mode at <100 cols
 */

import React, { useMemo } from 'react';
import { Box, Text, useStdout } from 'ink';
import { useNavigation } from './NavigationContext';

/** Terminal width thresholds for display modes - aligned with BREAKPOINTS in useResponsiveLayout
 * 12 tabs with full labels need ~140 cols
 * 12 tabs with short labels need ~105 cols
 * 12 tabs minimal (just numbers) need ~55 cols
 */
const FULL_WIDTH_THRESHOLD = 120;  // BREAKPOINTS.MEDIUM - full labels
const SHORT_WIDTH_THRESHOLD = 100; // BREAKPOINTS.COMPACT - short labels (at 80 cols, use minimal)

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
