/**
 * TabBar - Navigation tab bar component
 */

import React from 'react';
import { Box, Text } from 'ink';
import { useNavigation } from './NavigationContext';

export interface TabBarProps {
  /** Show app title before tabs */
  showTitle?: boolean;
  /** App title text */
  title?: string;
}

export function TabBar({
  showTitle = true,
  title = 'bc',
}: TabBarProps): React.ReactElement {
  const { currentView, tabs, canGoBack } = useNavigation();

  return (
    <Box flexShrink={0}>
      {showTitle && (
        <>
          <Text bold color="cyan">
            {title}{' '}
          </Text>
          <Text dimColor>|</Text>
        </>
      )}
      {tabs.map((tab, index) => (
        <React.Fragment key={tab.view}>
          <Text> </Text>
          <Text
            bold={currentView === tab.view}
            color={currentView === tab.view ? 'green' : undefined}
            dimColor={currentView !== tab.view}
          >
            [{tab.key}] {tab.label}
          </Text>
          {index < tabs.length - 1 && <Text dimColor> </Text>}
        </React.Fragment>
      ))}
      {canGoBack && (
        <>
          <Text dimColor> | </Text>
          <Text dimColor>[←] Back</Text>
        </>
      )}
    </Box>
  );
}

export default TabBar;
