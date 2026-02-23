/**
 * TopTabBar - Horizontal navigation tab bar component
 *
 * Issue #1755: Alternative to side drawer navigation
 *
 * Display modes based on terminal width:
 * - Full (>=140 cols): [1] Dashboard [2] Agents ...
 * - Short (100-139 cols): [1] Dash [2] Agt ...
 * - Compact (80-99 cols): [1] [2] [3] ... with group dropdowns
 * - Minimal (<80 cols): Shows only current tab + arrows
 */

import React, { useMemo } from 'react';
import { Box, Text, useStdout } from 'ink';
import { useNavigation, type TabConfig } from './NavigationContext';

/** Terminal width thresholds for display modes */
const WIDTH_THRESHOLDS = {
  FULL: 140,
  SHORT: 100,
  COMPACT: 80,
};

/** Display mode for tab labels */
type DisplayMode = 'full' | 'short' | 'compact' | 'minimal';

/** Tab sections for grouped navigation */
interface TabSection {
  name: string;
  color: string;
  views: string[];
}

const TAB_SECTIONS: TabSection[] = [
  { name: 'Work', color: 'cyan', views: ['dashboard', 'agents', 'channels', 'files', 'commands'] },
  { name: 'Monitor', color: 'yellow', views: ['logs', 'costs', 'processes', 'demons'] },
  { name: 'System', color: 'magenta', views: ['roles', 'worktrees', 'workspaces', 'memory', 'routing'] },
];

function getDisplayMode(width: number): DisplayMode {
  if (width >= WIDTH_THRESHOLDS.FULL) return 'full';
  if (width >= WIDTH_THRESHOLDS.SHORT) return 'short';
  if (width >= WIDTH_THRESHOLDS.COMPACT) return 'compact';
  return 'minimal';
}

export interface TopTabBarProps {
  /** Show app title before tabs */
  showTitle?: boolean;
  /** App title text */
  title?: string;
  /** Override terminal width (for testing) */
  terminalWidth?: number;
  /** Show section group indicators */
  showSections?: boolean;
}

export function TopTabBar({
  showTitle = true,
  title = 'bc',
  terminalWidth: overrideWidth,
  showSections = true,
}: TopTabBarProps): React.ReactElement {
  const { currentView, tabs, canGoBack, canGoForward } = useNavigation();
  const { stdout } = useStdout();

  const terminalWidth = overrideWidth ?? stdout.columns;
  const displayMode = useMemo(() => getDisplayMode(terminalWidth), [terminalWidth]);

  // Get tab label based on display mode
  const getTabLabel = (tab: TabConfig): string => {
    switch (displayMode) {
      case 'full':
        return tab.label;
      case 'short':
        return tab.shortLabel ?? tab.label.slice(0, 4);
      case 'compact':
      case 'minimal':
        return '';
    }
  };

  // Minimal mode: just current tab with navigation arrows
  if (displayMode === 'minimal') {
    const currentTab = tabs.find(t => t.view === currentView);
    return (
      <Box flexShrink={0}>
        {showTitle && (
          <Text bold color="cyan">{title}</Text>
        )}
        <Text> </Text>
        {canGoBack && <Text color="gray">{'<'} </Text>}
        <Text bold color="green">
          [{currentTab?.key ?? '?'}] {currentTab?.shortLabel ?? currentTab?.label ?? currentView}
        </Text>
        {canGoForward && <Text color="gray"> {'>'}</Text>}
        <Text dimColor> | Tab: switch</Text>
      </Box>
    );
  }

  // Compact mode: grouped tabs
  if (displayMode === 'compact' && showSections) {
    return (
      <Box flexShrink={0} flexWrap="wrap">
        {showTitle && (
          <>
            <Text bold color="cyan">{title}</Text>
            <Text dimColor> |</Text>
          </>
        )}
        {TAB_SECTIONS.map((section) => {
          const sectionTabs = tabs.filter(t => section.views.includes(t.view));
          const hasActive = sectionTabs.some(t => t.view === currentView);

          return (
            <Box key={section.name} marginLeft={1}>
              <Text
                color={hasActive ? section.color : undefined}
                dimColor={!hasActive}
                bold={hasActive}
              >
                {section.name}:
              </Text>
              {sectionTabs.map((tab) => {
                const isActive = currentView === tab.view;
                return (
                  <Text
                    key={tab.view}
                    bold={isActive}
                    color={isActive ? 'green' : undefined}
                    dimColor={!isActive}
                  >
                    {' '}[{tab.key}]
                  </Text>
                );
              })}
            </Box>
          );
        })}
        {/* Help tab */}
        <Text dimColor> [?]</Text>
      </Box>
    );
  }

  // Short/Full mode: linear tabs
  return (
    <Box flexShrink={0} flexWrap="nowrap">
      {showTitle && (
        <>
          <Text bold color="cyan">{title}</Text>
          <Text dimColor> |</Text>
        </>
      )}
      {tabs.filter(t => t.view !== 'help').map((tab) => {
        const isActive = currentView === tab.view;
        const label = getTabLabel(tab);

        // Find section for color
        const section = TAB_SECTIONS.find(s => s.views.includes(tab.view));
        const sectionColor = section?.color;

        return (
          <React.Fragment key={tab.view}>
            <Text> </Text>
            <Text
              bold={isActive}
              color={isActive ? 'green' : sectionColor}
              dimColor={!isActive && !sectionColor}
            >
              [{tab.key}]{label ? ` ${label}` : ''}
            </Text>
          </React.Fragment>
        );
      })}
      <Text dimColor> | [?] Help</Text>
    </Box>
  );
}

export default TopTabBar;
