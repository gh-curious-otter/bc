/**
 * App - Main TUI application component
 */

import React, { useState, useMemo } from 'react';
import { Box, Text, useStdout, useInput } from 'ink';
import {
  NavigationProvider,
  useNavigation,
  useKeyboardNavigation,
  Drawer,
  Breadcrumb,
  FocusProvider,
  type View,
} from './navigation';
import { ThemeProvider, useTheme, type ThemeMode } from './theme';
import { UnreadProvider, useKeybindingHints, useResponsiveLayout, HintsProvider, useHintsContext } from './hooks';
import { ConfigProvider, useThemeConfig } from './config';
import { Dashboard } from './views/Dashboard';
import { AgentsView } from './views/AgentsView';
import { CommandsView } from './views/CommandsView';
import { RolesView } from './views/RolesView';
import { ChannelsView } from './components/ChannelsView';
import { CostsView } from './components/CostsView';
import { LogsView } from './views/LogsView';
import { WorktreesView } from './views/WorktreesView';
import { WorkspaceSelectorView } from './views/WorkspaceSelectorView';
import { FilesView } from './views/FilesView';
import { DemonsView } from './views/DemonsView';
import { ProcessesView } from './views/ProcessesView';
import { MemoryView } from './views/MemoryView';
import { RoutingView } from './views/RoutingView';
import { CommandPalette } from './components/CommandPalette';
import { type BcCommand } from './types/commands';

interface AppProps {
  /** Disable input handling (useful for testing) */
  disableInput?: boolean;
  /** Initial view to display */
  initialView?: View;
}

export function App({
  disableInput = false,
  initialView = 'dashboard',
}: AppProps): React.ReactElement {
  return (
    <ConfigProvider>
      <AppWithTheme disableInput={disableInput} initialView={initialView} />
    </ConfigProvider>
  );
}

interface AppWithThemeProps {
  disableInput: boolean;
  initialView: View;
}

/**
 * AppWithTheme - Wraps the app with theme provider initialized from config
 * Must be inside ConfigProvider to use useThemeConfig
 */
function AppWithTheme({
  disableInput,
  initialView,
}: AppWithThemeProps): React.ReactElement {
  // Get theme config from workspace configuration
  const themeConfig = useThemeConfig();

  // Convert config theme/mode to ThemeMode for ThemeProvider
  // If a named theme is set (matrix, synthwave, etc), use 'auto' mode to let theme system handle it
  const effectiveMode: ThemeMode = themeConfig.theme !== 'dark' && themeConfig.theme !== 'light'
    ? 'auto'
    : (themeConfig.mode as ThemeMode);

  return (
    <ThemeProvider config={{ mode: effectiveMode }}>
      <NavigationProvider initialView={initialView}>
        <FocusProvider>
          <UnreadProvider>
            <HintsProvider>
              <AppContent disableInput={disableInput} themeConfig={themeConfig} />
            </HintsProvider>
          </UnreadProvider>
        </FocusProvider>
      </NavigationProvider>
    </ThemeProvider>
  );
}

interface AppContentProps {
  disableInput: boolean;
  themeConfig: ReturnType<typeof useThemeConfig>;
}

function AppContent({ disableInput, themeConfig }: AppContentProps): React.ReactElement {
  const { currentView, navigate } = useNavigation();
  const { stdout } = useStdout();
  const { setThemeName } = useTheme();
  const [showCommandPalette, setShowCommandPalette] = useState(false);

  // #1326: Use centralized responsive layout system
  const layout = useResponsiveLayout();

  // Apply configured theme on mount or when config changes
  React.useEffect(() => {
    // Only set theme if it's a named theme (not dark/light which are handled by mode)
    if (themeConfig.theme !== 'dark' && themeConfig.theme !== 'light') {
      setThemeName(themeConfig.theme as Parameters<typeof setThemeName>[0]);
    }
  }, [themeConfig.theme, setThemeName]);

  // Handle command palette selection - navigate to view if applicable
  const handleCommandSelect = (command: BcCommand): void => {
    // Map command names to views
    const viewMap: Record<string, View> = {
      'agent status': 'agents',
      'agent list': 'agents',
      'agent peek': 'agents',
      'channel list': 'channels',
      'channel history': 'channels',
      'cost show': 'costs',
      'logs': 'logs',
      'status': 'dashboard',
      'stats': 'dashboard',
      'process list': 'processes',
      'demon list': 'demons',
      'role list': 'roles',
      'memory list': 'memory',
      'memory show': 'memory',
      'memory search': 'memory',
      'help': 'help',
    };

    const targetView = viewMap[command.name] as View | undefined;
    if (targetView !== undefined) {
      navigate(targetView);
    }
    setShowCommandPalette(false);
  };

  // Handle global keyboard navigation
  useKeyboardNavigation({
    disabled: disableInput || showCommandPalette,
    onCommandPalette: () => { setShowCommandPalette(true); },
  });

  // Get terminal dimensions
  const terminalHeight = stdout.rows;
  const terminalWidth = stdout.columns;

  return (
    <Box flexDirection="column" padding={1} width={terminalWidth} height={terminalHeight}>
      {/* Main layout: drawer + content + detail pane */}
      <Box flexDirection="row" flexGrow={1}>
        {/* Left drawer navigation - controlled by responsive layout (#1326) */}
        {layout.drawer.visible && (
          <Drawer
            disabled={disableInput || showCommandPalette}
            shrunk={layout.drawer.shrunk}
            width={layout.drawer.width}
          />
        )}

        {/* Center content area */}
        <Box flexDirection="column" flexGrow={1} paddingLeft={layout.drawer.visible ? 1 : 0}>
          {/* Breadcrumb navigation (shows path when navigated deep) */}
          <Breadcrumb />

          {/* Main content area */}
          <Box flexDirection="column" flexGrow={1}>
            <ViewContent
              view={currentView}
              disableInput={disableInput}
            />
          </Box>
        </Box>
      </Box>

      {/* Footer with navigation hints - anchored to bottom */}
      <Footer />

      {/* Command palette overlay */}
      {showCommandPalette && (
        <Box position="absolute" marginTop={2} marginLeft={16}>
          <CommandPalette
            isOpen={showCommandPalette}
            onClose={() => { setShowCommandPalette(false); }}
            onSelect={handleCommandSelect}
          />
        </Box>
      )}
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
    case 'files':
      return <FilesView />;
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
    case 'workspaces':
      return <WorkspaceSelectorView />;
    case 'demons':
      return <DemonsView disableInput={disableInput} />;
    case 'processes':
      return <ProcessesView />;
    case 'memory':
      return <MemoryView disableInput={disableInput} />;
    case 'routing':
      return <RoutingView disableInput={disableInput} />;
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
      { keys: 'Tab', desc: 'Next view' },
      { keys: 'Shift+Tab', desc: 'Previous view' },
      { keys: 'M', desc: 'Memory view' },
      { keys: 'R', desc: 'Routing view' },
      { keys: '?', desc: 'Toggle help' },
      { keys: 'ESC', desc: 'Go back / Home' },
      { keys: 'Ctrl+R', desc: 'Refresh current view' },
      { keys: 'q', desc: 'Quit' },
    ]},
    { type: 'section' as const, title: 'Navigation (Drawer & Lists)', shortcuts: [
      { keys: 'j / ↓', desc: 'Move down in drawer/list' },
      { keys: 'k / ↑', desc: 'Move up in drawer/list' },
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
    { type: 'section' as const, title: 'Memory', shortcuts: [
      { keys: 'j/k', desc: 'Navigate agents' },
      { keys: 'Enter', desc: 'View details' },
      { keys: '/', desc: 'Search memories' },
      { keys: '1/2', desc: 'Switch exp/learnings' },
      { keys: 'c', desc: 'Clear memory' },
    ]},
    { type: 'section' as const, title: 'Routing', shortcuts: [
      { keys: 'j/k', desc: 'Navigate rules' },
      { keys: 'Enter', desc: 'View details' },
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

/**
 * Footer with dynamic hints and theme indicator - anchored to bottom
 *
 * Issue #1461: Combines view-specific hints (from ViewWrapper via context)
 * with universal hints (Tab, ?, q). This eliminates duplicate hint bars.
 */
function Footer(): React.ReactElement {
  const { theme } = useTheme();
  const { currentView } = useNavigation();
  const { viewHints } = useHintsContext();
  const { hints: universalHints } = useKeybindingHints(currentView, 'normal');

  // Combine view-specific hints with universal hints
  // View hints come first, then universal hints (Tab, ?, q)
  const allHints = [...viewHints, ...universalHints];

  // Format hints for display
  const formatted = allHints
    .map((h) => `[${h.key}] ${h.label}`)
    .join('  ');

  return (
    <Box marginTop={1} justifyContent="space-between">
      <Text dimColor>{formatted}</Text>
      <Text dimColor>Theme: {theme.name}</Text>
    </Box>
  );
}
