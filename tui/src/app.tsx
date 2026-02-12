/**
 * App - Main TUI application component
 */

import React from 'react';
import { Box, Text } from 'ink';
import {
  NavigationProvider,
  useNavigation,
  useKeyboardNavigation,
  TabBar,
  type View,
} from './navigation';
import { ThemeProvider, useTheme, type ThemeMode } from './theme';
import { ChannelsView } from './components/ChannelsView';
import { CostsView } from './components/CostsView';

interface AppProps {
  /** Disable input handling (useful for testing) */
  disableInput?: boolean;
  /** Initial view to display */
  initialView?: View;
  /** Theme mode (dark/light/auto) */
  themeMode?: ThemeMode;
}

export function App({
  disableInput = false,
  initialView = 'dashboard',
  themeMode = 'auto',
}: AppProps): React.ReactElement {
  return (
    <ThemeProvider config={{ mode: themeMode }}>
      <NavigationProvider initialView={initialView}>
        <AppContent disableInput={disableInput} />
      </NavigationProvider>
    </ThemeProvider>
  );
}

interface AppContentProps {
  disableInput: boolean;
}

function AppContent({ disableInput }: AppContentProps): React.ReactElement {
  const { currentView } = useNavigation();

  // Handle global keyboard navigation
  useKeyboardNavigation({ disabled: disableInput });

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header with tab bar */}
      <TabBar />

      {/* Main content area */}
      <Box flexDirection="column" marginTop={1}>
        <ViewContent view={currentView} disableInput={disableInput} />
      </Box>

      {/* Footer with navigation hints */}
      <Footer />
    </Box>
  );
}

interface ViewContentProps {
  view: View;
  disableInput: boolean;
}

// Main content router
function ViewContent({ view, disableInput }: ViewContentProps): React.ReactElement {
  switch (view) {
    case 'dashboard':
      return <DashboardView />;
    case 'agents':
      return <AgentsView />;
    case 'channels':
      return <ChannelsView disableInput={disableInput} />;
    case 'costs':
      return <CostsView disableInput={disableInput} />;
    case 'help':
      return <HelpView />;
    default:
      return <Text>Unknown view</Text>;
  }
}

// Placeholder views - will be implemented in Phase 2
function DashboardView(): React.ReactElement {
  return (
    <Box flexDirection="column">
      <Text bold>Dashboard</Text>
      <Text dimColor>Loading workspace status...</Text>
    </Box>
  );
}

function AgentsView(): React.ReactElement {
  return (
    <Box flexDirection="column">
      <Text bold>Agents</Text>
      <Text dimColor>Loading agents...</Text>
    </Box>
  );
}

function HelpView(): React.ReactElement {
  const { theme, isDark } = useTheme();
  return (
    <Box flexDirection="column">
      <Text bold>Keyboard Shortcuts</Text>
      <Box marginTop={1} flexDirection="column">
        <Text>
          <Text color="yellow">1-4</Text>       Switch tabs
        </Text>
        <Text>
          <Text color="yellow">?</Text>         Show help
        </Text>
        <Text>
          <Text color="yellow">ESC</Text>       Go back / Home
        </Text>
        <Text>
          <Text color="yellow">Backspace</Text> Go back in history
        </Text>
        <Text>
          <Text color="yellow">q</Text>         Quit
        </Text>
      </Box>
      <Box marginTop={1} flexDirection="column">
        <Text bold>View-specific shortcuts:</Text>
        <Text dimColor>Check each view for additional shortcuts</Text>
      </Box>
      <Box marginTop={1} flexDirection="column">
        <Text bold>Theme:</Text>
        <Text dimColor>
          Current: {theme.name} ({isDark ? 'dark' : 'light'} mode)
        </Text>
      </Box>
    </Box>
  );
}

// Footer with hints and theme indicator
function Footer(): React.ReactElement {
  const { theme } = useTheme();
  return (
    <Box marginTop={1} justifyContent="space-between">
      <Text dimColor>Press [?] for help, [q] to quit</Text>
      <Text dimColor>Theme: {theme.name}</Text>
    </Box>
  );
}
