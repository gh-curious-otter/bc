/**
 * Breadcrumb - Always shows current view name + active filter
 */

import React from 'react';
import { Box, Text } from 'ink';
import { useNavigation } from './NavigationContext';
import { useFilter } from '../hooks/useFilter';

export function Breadcrumb(): React.ReactElement {
  const { breadcrumbs, currentView, getTabByView } = useNavigation();
  const { query, isActive: filterActive } = useFilter();

  const currentTab = getTabByView(currentView);
  const basePath = currentTab?.label ?? currentView;

  return (
    <Box>
      <Text dimColor>{'> '}</Text>
      <Text color="cyan" bold>{basePath}</Text>
      {breadcrumbs.map((item, index) => (
        <React.Fragment key={index}>
          <Text dimColor> {'>'} </Text>
          <Text color={index === breadcrumbs.length - 1 ? 'white' : 'cyan'}>
            {item.label}
          </Text>
        </React.Fragment>
      ))}
      {filterActive && (
        <>
          <Text dimColor>{'  '}</Text>
          <Text color="yellow">/{query}</Text>
        </>
      )}
    </Box>
  );
}

export default Breadcrumb;
