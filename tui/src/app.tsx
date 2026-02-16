/**
 * App - Main TUI application component
 */

import React from 'react';
import { Box, Text, useStdout } from 'ink';
import {
  NavigationProvider,
  useNavigation,
  useKeyboardNavigation,
  TabBar,
  Breadcrumb,
  FocusProvider,
  type View,
} from './navigation';
import { ThemeProvider, useTheme, type ThemeMode } from './theme';
import { UnreadProvider } from './hooks';
import { Dashboard } from './views/Dashboard';
import { AgentsView } from './views/AgentsView';
import { CommandsView } from './views/CommandsView';
import { RolesView } from './views/RolesView';
import { ChannelsView } from './components/ChannelsView';
import { CostsView } from './components/CostsView';
import { LogsView } from './views/LogsView';
import { WorktreesView } from './views/WorktreesView';

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
        <FocusProvider>
          <UnreadProvider>
            <AppContent disableInput={disableInput} />
          </UnreadProvider>
        </FocusProvider>
      </NavigationProvider>
    </ThemeProvider>
  );
}

interface AppContentProps {
  disableInput: boolean;
}

function AppContent({ disableInput }: AppContentProps): React.ReactElement {
  const { currentView } = useNavigation();
  const { stdout } = useStdout();

  // Handle global keyboard navigation
  useKeyboardNavigation({ disabled: disableInput });

  // Get terminal dimensions - constrain to actual terminal height
  const terminalHeight = stdout.rows;
  const terminalWidth = stdout.columns;

  return (
    <Box flexDirection="column" padding={1} width={terminalWidth} height={terminalHeight}>
      {/* Header with tab bar */}
      <TabBar />

      {/* Breadcrumb navigation (shows path when navigated deep) */}
      <Breadcrumb />

      {/* Main content area - grows to fill available space */}
      <Box flexDirection="column" marginTop={1} flexGrow={1}>
        <ViewContent view={currentView} disableInput={disableInput} />
      </Box>

      {/* Footer with navigation hints - anchored to bottom */}
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
      return <Dashboard />;
    case 'agents':
      return <AgentsView />;
    case 'channels':
      return <ChannelsView disableInput={disableInput} />;
    case 'costs':
      return <CostsView disableInput={disableInput} />;
    case 'commands':
      return <CommandsView disableInput={disableInput} />;
    case 'roles':
      return <RolesView disableInput={disableInput} />;
    case 'logs':
      return <LogsView />;
    case 'worktrees':
      return <WorktreesView />;
    case 'help':
      return <HelpView />;
    default:
      return <Text>Unknown view</Text>;
  }
}

function HelpView(): React.ReactElement {
  const { theme, isDark } = useTheme();
  return (
    <Box flexDirection="column" paddingX={1}>
      <Text bold color="cyan">KEYBOARD SHORTCUTS</Text>
      <Text dimColor>{'─'.repeat(50)}</Text>

      {/* Global shortcuts */}
      <Box marginTop={1} flexDirection="column">
        <Text bold>GLOBAL</Text>
        <Box flexDirection="column" marginLeft={2}>
          <ShortcutRow keys="1-8" desc="Switch tabs" />
          <ShortcutRow keys="?" desc="Toggle help" />
          <ShortcutRow keys="ESC" desc="Go back / Home" />
          <ShortcutRow keys="Backspace" desc="History back" />
          <ShortcutRow keys="q" desc="Quit" />
          <ShortcutRow keys="Tab" desc="Next tab" />
          <ShortcutRow keys="Shift+Tab" desc="Previous tab" />
        </Box>
      </Box>

      {/* Navigation */}
      <Box marginTop={1} flexDirection="column">
        <Text bold>NAVIGATION</Text>
        <Box flexDirection="column" marginLeft={2}>
          <ShortcutRow keys="j / ↓" desc="Move down" />
          <ShortcutRow keys="k / ↑" desc="Move up" />
          <ShortcutRow keys="g" desc="Go to top" />
          <ShortcutRow keys="G" desc="Go to bottom" />
          <ShortcutRow keys="Enter" desc="Select / Drill down" />
        </Box>
      </Box>

      {/* Agents */}
      <Box marginTop={1} flexDirection="column">
        <Text bold>AGENTS</Text>
        <Box flexDirection="column" marginLeft={2}>
          <ShortcutRow keys="Enter" desc="View agent details" />
          <ShortcutRow keys="a" desc="Attach to session" />
          <ShortcutRow keys="p" desc="Peek agent output" />
          <ShortcutRow keys="s" desc="Start new agent" />
          <ShortcutRow keys="/" desc="Search agents" />
        </Box>
      </Box>

      {/* Channels */}
      <Box marginTop={1} flexDirection="column">
        <Text bold>CHANNELS</Text>
        <Box flexDirection="column" marginLeft={2}>
          <ShortcutRow keys="Enter" desc="View channel messages" />
          <ShortcutRow keys="m" desc="Compose message" />
          <ShortcutRow keys="j/k" desc="Scroll messages" />
          <ShortcutRow keys="c" desc="Clear message draft" />
        </Box>
      </Box>

      {/* Commands */}
      <Box marginTop={1} flexDirection="column">
        <Text bold>COMMANDS</Text>
        <Box flexDirection="column" marginLeft={2}>
          <ShortcutRow keys="/" desc="Search commands" />
          <ShortcutRow keys="Enter" desc="Copy command" />
          <ShortcutRow keys="f" desc="Toggle favorite" />
        </Box>
      </Box>

      {/* Theme */}
      <Box marginTop={1} flexDirection="column">
        <Text bold>THEME</Text>
        <Box marginLeft={2}>
          <Text dimColor>
            Current: {theme.name} ({isDark ? 'dark' : 'light'} mode)
          </Text>
        </Box>
      </Box>
    </Box>
  );
}

function ShortcutRow({ keys, desc }: { keys: string; desc: string }): React.ReactElement {
  return (
    <Box>
      <Box width={14}>
        <Text color="yellow">{keys}</Text>
      </Box>
      <Text>{desc}</Text>
    </Box>
  );
}

// Footer with hints and theme indicator - anchored to bottom
function Footer(): React.ReactElement {
  const { theme } = useTheme();
  return (
    <Box marginTop={1} justifyContent="space-between">
      <Text dimColor>Press [?] for help, [q] to quit</Text>
      <Text dimColor>Theme: {theme.name}</Text>
    </Box>
  );
}
