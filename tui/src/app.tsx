/**
 * App - Main TUI application component
 */

import React, { useState, useMemo, useCallback, memo } from 'react';
import { Box, Text, useStdout } from 'ink';
import {
  NavigationProvider,
  useNavigation,
  useKeyboardNavigation,
  Drawer,
  Breadcrumb,
  FocusProvider,
  type View,
} from './navigation';
import { useTheme } from './theme';
import { UnreadProvider, useKeybindingHints, useResponsiveLayout, HintsProvider, useHintsContext, DisableInputProvider, useDisableInput } from './hooks';
import { useThemeConfig } from './config';
import { RootProvider } from './providers';
import { Dashboard } from './views/Dashboard';
import { AgentsView } from './views/AgentsView';
import { CommandsView } from './views/CommandsView';
import { RolesView } from './views/RolesView';
import { ChannelsView } from './views/ChannelsView';
import { CostsView } from './views/CostsView';
import { LogsView } from './views/LogsView';
import { WorktreesView } from './views/WorktreesView';
import { WorkspaceSelectorView } from './views/WorkspaceSelectorView';
import { FilesView } from './views/FilesView';
import { DemonsView } from './views/DemonsView';
import { ProcessesView } from './views/ProcessesView';
import { MemoryView } from './views/MemoryView';
import { RoutingView } from './views/RoutingView';
import { HelpView } from './views/HelpView';
import { CommandPalette } from './components/CommandPalette';
import { ViewErrorBoundary } from './components/ErrorBoundary';
import { type BcCommand } from './types/commands';

interface AppProps {
  /** Disable input handling (useful for testing) */
  disableInput?: boolean;
  /** Initial view to display */
  initialView?: View;
}

/**
 * App - Main entry point with simplified provider tree (#1608)
 *
 * Provider hierarchy:
 * - RootProvider: Config + Theme (combined)
 * - NavigationProvider: View routing
 * - FocusProvider: Keyboard focus management
 * - UnreadProvider: Unread message tracking
 * - HintsProvider: Footer keyboard hints
 * - DisableInputProvider: Input control for modals/tests
 */
export function App({
  disableInput = false,
  initialView = 'dashboard',
}: AppProps): React.ReactElement {
  return (
    <RootProvider>
      <AppWithFeatureProviders disableInput={disableInput} initialView={initialView} />
    </RootProvider>
  );
}

interface AppWithFeatureProvidersProps {
  disableInput: boolean;
  initialView: View;
}

/**
 * AppWithFeatureProviders - Feature-level providers
 * Inside RootProvider, wraps with navigation and UI state providers
 */
function AppWithFeatureProviders({
  disableInput,
  initialView,
}: AppWithFeatureProvidersProps): React.ReactElement {
  // Get theme config from workspace configuration (provided by RootProvider)
  const themeConfig = useThemeConfig();

  return (
    <NavigationProvider initialView={initialView}>
      <FocusProvider>
        <UnreadProvider>
          <HintsProvider>
            {/* #1594: DisableInputProvider eliminates prop drilling */}
            <DisableInputProvider disabled={disableInput}>
              <AppContent themeConfig={themeConfig} />
            </DisableInputProvider>
          </HintsProvider>
        </UnreadProvider>
      </FocusProvider>
    </NavigationProvider>
  );
}

interface AppContentProps {
  themeConfig: ReturnType<typeof useThemeConfig>;
}

function AppContent({ themeConfig }: AppContentProps): React.ReactElement {
  const { currentView, navigate } = useNavigation();
  const { stdout } = useStdout();
  const { setThemeName } = useTheme();
  // #1594: Use context instead of prop drilling
  const { isDisabled: disableInput } = useDisableInput();
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
  // #1596: Memoized to prevent unnecessary re-renders
  const handleCommandSelect = useCallback((command: BcCommand): void => {
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
  }, [navigate]);

  // #1596: Memoized callbacks for command palette state
  const openCommandPalette = useCallback(() => {
    setShowCommandPalette(true);
  }, []);

  const closeCommandPalette = useCallback(() => {
    setShowCommandPalette(false);
  }, []);

  // Handle global keyboard navigation
  useKeyboardNavigation({
    disabled: disableInput || showCommandPalette,
    onCommandPalette: openCommandPalette,
  });

  // Get terminal dimensions
  const terminalHeight = stdout.rows;
  const terminalWidth = stdout.columns;

  // #1611 fix: Calculate responsive margins for command palette overlay
  // Center the palette horizontally with minimum margin of 4
  const commandPaletteMarginLeft = Math.max(4, Math.floor((terminalWidth - 60) / 2));

  return (
    <Box flexDirection="column" padding={1} width={terminalWidth} height={terminalHeight} overflow="hidden">
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

        {/* Center content area - #1611 fix: Add overflow="hidden" to prevent content overflow */}
        <Box flexDirection="column" flexGrow={1} paddingLeft={layout.drawer.visible ? 1 : 0} overflow="hidden">
          {/* Breadcrumb navigation (shows path when navigated deep) */}
          <Breadcrumb />

          {/* Main content area - wrapped with error boundary (#1585) */}
          <Box flexDirection="column" flexGrow={1}>
            <ViewErrorBoundary viewName={currentView}>
              {/* #1594: Views use useDisableInput hook instead of props */}
              <ViewContent view={currentView} />
            </ViewErrorBoundary>
          </Box>
        </Box>
      </Box>

      {/* Footer with navigation hints - anchored to bottom */}
      <Footer />

      {/* Command palette overlay - #1611 fix: Use responsive margins */}
      {showCommandPalette && (
        <Box position="absolute" marginTop={2} marginLeft={commandPaletteMarginLeft}>
          <CommandPalette
            isOpen={showCommandPalette}
            onClose={closeCommandPalette}
            onSelect={handleCommandSelect}
          />
        </Box>
      )}
    </Box>
  );
}

interface ViewContentProps {
  view: View;
}

// Main content router - #1596: Memoized, #1594: views use context
const ViewContent = memo(function ViewContent({ view }: ViewContentProps): React.ReactElement {
  switch (view) {
    case 'dashboard':
      return <Dashboard />;
    case 'agents':
      return <AgentsView />;
    case 'channels':
      return <ChannelsView />;
    case 'files':
      return <FilesView />;
    case 'costs':
      return <CostsView />;
    case 'commands':
      return <CommandsView />;
    case 'roles':
      return <RolesView />;
    case 'logs':
      return <LogsView />;
    case 'worktrees':
      return <WorktreesView />;
    case 'workspaces':
      return <WorkspaceSelectorView />;
    case 'demons':
      return <DemonsView />;
    case 'processes':
      return <ProcessesView />;
    case 'memory':
      return <MemoryView />;
    case 'routing':
      return <RoutingView />;
    case 'help':
      return <HelpView />;
    default:
      return <Text>Unknown view</Text>;
  }
});

/**
 * Footer with dynamic hints and theme indicator - anchored to bottom
 *
 * Issue #1461: Combines view-specific hints (from ViewWrapper via context)
 * with universal hints (Tab, ?, q). This eliminates duplicate hint bars.
 * #1596: Memoized to prevent unnecessary re-renders
 */
const Footer = memo(function Footer(): React.ReactElement {
  const { theme } = useTheme();
  const { currentView } = useNavigation();
  const { viewHints } = useHintsContext();
  const { hints: universalHints } = useKeybindingHints(currentView, 'normal');

  // #1596: Memoize formatted hints string
  const formatted = useMemo(() => {
    const allHints = [...viewHints, ...universalHints];
    return allHints
      .map((h) => `[${h.key}] ${h.label}`)
      .join('  ');
  }, [viewHints, universalHints]);

  return (
    <Box marginTop={1} justifyContent="space-between">
      <Text dimColor>{formatted}</Text>
      <Text dimColor>Theme: {theme.name}</Text>
    </Box>
  );
});
