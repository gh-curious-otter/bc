/**
 * App - Main TUI application component
 */

import React, { useState, useMemo } from 'react';
import { Box, Text, useStdout, useInput } from 'ink';
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
  const { stdout } = useStdout();
  const [scrollOffset, setScrollOffset] = useState(0);

  // All help sections as an array of renderable items
  const helpSections = useMemo(() => [
    { type: 'header' as const },
    { type: 'section' as const, title: 'Global', shortcuts: [
      { keys: '1-8', desc: 'Switch tabs' },
      { keys: '?', desc: 'Toggle help' },
      { keys: 'ESC', desc: 'Go back / Home' },
      { keys: 'Tab', desc: 'Next tab' },
      { keys: 'Shift+Tab', desc: 'Previous tab' },
      { keys: 'Ctrl+R', desc: 'Refresh current view' },
      { keys: 'q', desc: 'Quit' },
    ]},
    { type: 'section' as const, title: 'Navigation', shortcuts: [
      { keys: 'j / ↓', desc: 'Move down' },
      { keys: 'k / ↑', desc: 'Move up' },
      { keys: 'g', desc: 'Jump to top' },
      { keys: 'G', desc: 'Jump to bottom' },
      { keys: 'Enter', desc: 'Select / Drill down' },
    ]},
    { type: 'section' as const, title: 'Agents', shortcuts: [
      { keys: 'Enter', desc: 'Attach to agent session' },
      { keys: 'p', desc: 'Peek agent output' },
      { keys: 'x', desc: 'Stop agent' },
      { keys: 'X', desc: 'Kill agent (force)' },
      { keys: 'R', desc: 'Restart agent' },
    ]},
    { type: 'section' as const, title: 'Channels', shortcuts: [
      { keys: 'Enter', desc: 'View channel history' },
      { keys: 'm', desc: 'Compose message' },
      { keys: 'j/k', desc: 'Scroll messages' },
      { keys: 'c', desc: 'Clear draft' },
    ]},
    { type: 'section' as const, title: 'Costs', shortcuts: [
      { keys: '1/2/3', desc: 'Switch agent/model/team tabs' },
      { keys: 'b', desc: 'Set budget' },
      { keys: 'e', desc: 'Export to CSV' },
      { keys: 'r', desc: 'Refresh data' },
    ]},
    { type: 'section' as const, title: 'Commands', shortcuts: [
      { keys: '/', desc: 'Search commands' },
      { keys: 'f', desc: 'Toggle favorite' },
      { keys: 'Enter', desc: 'Copy command' },
    ]},
    { type: 'footer' as const },
  ], []);

  // Calculate total lines needed
  const totalLines = helpSections.reduce((acc, section) => {
    if (section.type === 'header') return acc + 2;
    if (section.type === 'footer') return acc + 3;
    return acc + 1 + section.shortcuts.length + 1; // title + shortcuts + margin
  }, 0);

  // Available height for content (reserve 4 lines for header/footer/hints)
  const availableHeight = Math.max(10, (stdout.rows || 24) - 6);
  const needsScroll = totalLines > availableHeight;
  const maxScroll = Math.max(0, totalLines - availableHeight);

  // Handle scroll with j/k
  useInput((input, key) => {
    if (needsScroll) {
      if (input === 'j' || key.downArrow) {
        setScrollOffset(prev => Math.min(prev + 1, maxScroll));
      }
      if (input === 'k' || key.upArrow) {
        setScrollOffset(prev => Math.max(prev - 1, 0));
      }
      if (input === 'g') {
        setScrollOffset(0);
      }
      if (input === 'G') {
        setScrollOffset(maxScroll);
      }
    }
  });

  // Build visible content
  let currentLine = 0;
  const visibleContent: React.ReactNode[] = [];

  for (const section of helpSections) {
    if (section.type === 'header') {
      if (currentLine >= scrollOffset && currentLine < scrollOffset + availableHeight) {
        visibleContent.push(
          <Text key="title" bold color="cyan">KEYBOARD SHORTCUTS</Text>,
          <Text key="divider" dimColor>{'─'.repeat(40)}</Text>
        );
      }
      currentLine += 2;
    } else if (section.type === 'footer') {
      if (currentLine >= scrollOffset && currentLine < scrollOffset + availableHeight) {
        visibleContent.push(
          <Box key="footer" marginTop={1} flexDirection="column">
            <Text dimColor>{'─'.repeat(40)}</Text>
            <Text dimColor>
              Theme: {theme.name} ({isDark ? 'dark' : 'light'} mode)
            </Text>
          </Box>
        );
      }
      currentLine += 3;
    } else {
      // Section with shortcuts
      const sectionLines = 1 + section.shortcuts.length + 1;
      if (currentLine + sectionLines > scrollOffset && currentLine < scrollOffset + availableHeight) {
        const startIdx = Math.max(0, scrollOffset - currentLine);
        const endIdx = Math.min(sectionLines, scrollOffset + availableHeight - currentLine);

        const sectionContent: React.ReactNode[] = [];
        if (startIdx === 0) {
          sectionContent.push(<Text key={`${section.title}-title`} bold>{section.title}</Text>);
        }

        section.shortcuts.forEach((shortcut, idx) => {
          const lineIdx = idx + 1;
          if (lineIdx >= startIdx && lineIdx < endIdx) {
            sectionContent.push(
              <ShortcutRow key={`${section.title}-${shortcut.keys}`} keys={shortcut.keys} desc={shortcut.desc} />
            );
          }
        });

        if (sectionContent.length > 0) {
          visibleContent.push(
            <Box key={section.title} marginTop={currentLine > scrollOffset ? 1 : 0} flexDirection="column">
              {sectionContent}
            </Box>
          );
        }
      }
      currentLine += sectionLines;
    }
  }

  return (
    <Box flexDirection="column" height={availableHeight + 2}>
      {needsScroll && scrollOffset > 0 && (
        <Text dimColor>↑ Scroll up (k)</Text>
      )}
      <Box flexDirection="column" flexGrow={1} overflow="hidden">
        {visibleContent}
      </Box>
      {needsScroll && scrollOffset < maxScroll && (
        <Text dimColor>↓ Scroll down (j) — {Math.round((scrollOffset / maxScroll) * 100)}%</Text>
      )}
      {needsScroll && (
        <Text dimColor>Use j/k to scroll, g/G for top/bottom</Text>
      )}
    </Box>
  );
}

/** Helper component for shortcut rows */
function ShortcutRow({ keys, desc }: { keys: string; desc: string }): React.ReactElement {
  return (
    <Text>
      <Text color="yellow">{keys.padEnd(12)}</Text>
      <Text>{desc}</Text>
    </Text>
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
