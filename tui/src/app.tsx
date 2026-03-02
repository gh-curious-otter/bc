/**
 * App - Main TUI application component
 * k9s-style navigation with :command bar and /filter bar
 */

import React, { useState, useCallback, memo } from 'react';
import { Box, Text, useStdout } from 'ink';
import {
  NavigationProvider,
  useNavigation,
  useKeyboardNavigation,
  Breadcrumb,
  FocusProvider,
  type View,
} from './navigation';
import { useTheme } from './theme';
import { UnreadProvider, HintsProvider, DisableInputProvider, useDisableInput } from './hooks';
import { useThemeConfig } from './config';
import { RootProvider } from './providers';
import { Dashboard } from './views/Dashboard';
import { AgentsView } from './views/AgentsView';
import { RolesView } from './views/RolesView';
import { ChannelsView } from './views/ChannelsView';
import { CostsView } from './views/CostsView';
import { LogsView } from './views/LogsView';
import { WorktreesView } from './views/WorktreesView';
import { MemoryView } from './views/MemoryView';
import { HelpView } from './views/HelpView';
import { ToolsView } from './views/ToolsView';
import { CommandBar } from './components/CommandBar';
import { FilterBar } from './components/FilterBar';
import { FilterProvider } from './hooks/useFilter';
import { ViewErrorBoundary } from './components/ErrorBoundary';

interface AppProps {
  disableInput?: boolean;
  initialView?: View;
}

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

function AppWithFeatureProviders({
  disableInput,
  initialView,
}: AppWithFeatureProvidersProps): React.ReactElement {
  const themeConfig = useThemeConfig();

  return (
    <NavigationProvider initialView={initialView}>
      <FocusProvider>
        <UnreadProvider>
          <HintsProvider>
            <DisableInputProvider disabled={disableInput}>
              <FilterProvider>
                <AppContent themeConfig={themeConfig} />
              </FilterProvider>
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
  const { isDisabled: disableInput } = useDisableInput();
  const [showCommandBar, setShowCommandBar] = useState(false);
  const [showFilterBar, setShowFilterBar] = useState(false);

  React.useEffect(() => {
    if (themeConfig.theme !== 'dark' && themeConfig.theme !== 'light') {
      setThemeName(themeConfig.theme as Parameters<typeof setThemeName>[0]);
    }
  }, [themeConfig.theme, setThemeName]);

  const openCommandBar = useCallback(() => {
    setShowCommandBar(true);
    setShowFilterBar(false);
  }, []);

  const closeCommandBar = useCallback(() => {
    setShowCommandBar(false);
  }, []);

  const handleCommandSelect = useCallback((view: View) => {
    navigate(view);
    setShowCommandBar(false);
  }, [navigate]);

  const openFilterBar = useCallback(() => {
    setShowFilterBar(true);
    setShowCommandBar(false);
  }, []);

  const closeFilterBar = useCallback(() => {
    setShowFilterBar(false);
  }, []);

  useKeyboardNavigation({
    disabled: disableInput || showCommandBar || showFilterBar,
    onCommandBar: openCommandBar,
    onFilterBar: openFilterBar,
  });

  const terminalHeight = stdout.rows;
  const terminalWidth = stdout.columns;

  return (
    <Box flexDirection="column" padding={1} width={terminalWidth} height={terminalHeight} overflow="hidden">
      {/* Breadcrumb - always shows current view */}
      <Breadcrumb />

      {/* Main content area - full width, no sidebar */}
      <Box flexDirection="column" flexGrow={1} overflow="hidden">
        <ViewErrorBoundary key={currentView} viewName={currentView}>
          <ViewContent view={currentView} />
        </ViewErrorBoundary>
      </Box>

      {/* Command bar overlay (k9s-style :command) */}
      {showCommandBar && (
        <CommandBar
          onSelect={handleCommandSelect}
          onClose={closeCommandBar}
        />
      )}

      {/* Filter bar overlay (k9s-style /filter) */}
      {showFilterBar && (
        <FilterBar
          onClose={closeFilterBar}
        />
      )}

      {/* Footer with static hints - only when no overlay is active */}
      {!showCommandBar && !showFilterBar && <Footer />}
    </Box>
  );
}

interface ViewContentProps {
  view: View;
}

const ViewContent = memo(function ViewContent({ view }: ViewContentProps): React.ReactElement {
  switch (view) {
    case 'dashboard':
      return <Dashboard />;
    case 'agents':
      return <AgentsView />;
    case 'channels':
      return <ChannelsView />;
    case 'costs':
      return <CostsView />;
    case 'logs':
      return <LogsView />;
    case 'roles':
      return <RolesView />;
    case 'worktrees':
      return <WorktreesView />;
    case 'memory':
      return <MemoryView />;
    case 'tools':
      return <ToolsView />;
    case 'help':
      return <HelpView />;
    default:
      return <Text>Unknown view</Text>;
  }
});

/**
 * Footer - single line with static k9s-style hints
 */
const Footer = memo(function Footer(): React.ReactElement {
  return (
    <Box marginTop={1}>
      <Text dimColor>
        [<Text bold>:</Text>] command  [<Text bold>/</Text>] filter  [<Text bold>?</Text>] help  [<Text bold>Tab</Text>] next  [<Text bold>q</Text>] quit
      </Text>
    </Box>
  );
});
