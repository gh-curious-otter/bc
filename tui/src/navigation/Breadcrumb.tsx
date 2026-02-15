/**
 * Breadcrumb - Navigation breadcrumb component
 * Shows current location path in the navigation hierarchy
 */

import React from 'react';
import { Box, Text } from 'ink';
import { useNavigation } from './NavigationContext';

export function Breadcrumb(): React.ReactElement | null {
  const { breadcrumbs, currentView, getTabByView } = useNavigation();

  // Don't show breadcrumb if empty or only has one item
  if (breadcrumbs.length === 0) {
    return null;
  }

  // Get current tab label as base
  const currentTab = getTabByView(currentView);
  const basePath = currentTab?.label ?? currentView;

  return (
    <Box>
      <Text dimColor>{'> '}</Text>
      <Text color="cyan">{basePath}</Text>
      {breadcrumbs.map((item, index) => (
        <React.Fragment key={index}>
          <Text dimColor> {'>'} </Text>
          <Text color={index === breadcrumbs.length - 1 ? 'white' : 'cyan'}>
            {item.label}
          </Text>
        </React.Fragment>
      ))}
    </Box>
  );
}

export default Breadcrumb;
