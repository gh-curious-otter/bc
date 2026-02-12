import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';

// View types for navigation
type View = 'dashboard' | 'agents' | 'channels' | 'costs' | 'help';

// Tab configuration
const TABS: { key: string; view: View; label: string }[] = [
  { key: '1', view: 'dashboard', label: 'Dashboard' },
  { key: '2', view: 'agents', label: 'Agents' },
  { key: '3', view: 'channels', label: 'Channels' },
  { key: '4', view: 'costs', label: 'Costs' },
  { key: '?', view: 'help', label: 'Help' },
];

interface AppProps {
  /** Disable input handling (useful for testing) */
  disableInput?: boolean;
}

export function App({ disableInput = false }: AppProps): React.ReactElement {
  const [currentView, setCurrentView] = useState<View>('dashboard');

  // Handle keyboard input for navigation
  // Use isActive option to conditionally enable input handling
  useInput(
    (input, key) => {
      // Tab navigation with number keys
      const tab = TABS.find((t) => t.key === input);
      if (tab) {
        setCurrentView(tab.view);
        return;
      }

      // ESC to go back to dashboard
      if (key.escape) {
        setCurrentView('dashboard');
      }

      // q to quit
      if (input === 'q') {
        process.exit(0);
      }
    },
    { isActive: !disableInput }
  );

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Header currentView={currentView} />

      {/* Main content area */}
      <Box flexDirection="column" marginTop={1}>
        <ViewContent view={currentView} />
      </Box>

      {/* Footer with navigation hints */}
      <Footer />
    </Box>
  );
}

// Header component with tabs
function Header({ currentView }: { currentView: View }): React.ReactElement {
  return (
    <Box>
      <Text bold color="cyan">
        bc{' '}
      </Text>
      <Text dimColor>|</Text>
      {TABS.map((tab, index) => (
        <React.Fragment key={tab.view}>
          <Text> </Text>
          <Text
            bold={currentView === tab.view}
            color={currentView === tab.view ? 'green' : undefined}
            dimColor={currentView !== tab.view}
          >
            [{tab.key}] {tab.label}
          </Text>
          {index < TABS.length - 1 && <Text dimColor> </Text>}
        </React.Fragment>
      ))}
    </Box>
  );
}

// Main content router
function ViewContent({ view }: { view: View }): React.ReactElement {
  switch (view) {
    case 'dashboard':
      return <DashboardView />;
    case 'agents':
      return <AgentsView />;
    case 'channels':
      return <ChannelsView />;
    case 'costs':
      return <CostsView />;
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

function ChannelsView(): React.ReactElement {
  return (
    <Box flexDirection="column">
      <Text bold>Channels</Text>
      <Text dimColor>Loading channels...</Text>
    </Box>
  );
}

function CostsView(): React.ReactElement {
  return (
    <Box flexDirection="column">
      <Text bold>Costs</Text>
      <Text dimColor>Loading cost data...</Text>
    </Box>
  );
}

function HelpView(): React.ReactElement {
  return (
    <Box flexDirection="column">
      <Text bold>Keyboard Shortcuts</Text>
      <Box marginTop={1} flexDirection="column">
        <Text>
          <Text color="yellow">1-4</Text> Switch tabs
        </Text>
        <Text>
          <Text color="yellow">?</Text>   Show help
        </Text>
        <Text>
          <Text color="yellow">ESC</Text> Back to dashboard
        </Text>
        <Text>
          <Text color="yellow">q</Text>   Quit
        </Text>
      </Box>
    </Box>
  );
}

// Footer with hints
function Footer(): React.ReactElement {
  return (
    <Box marginTop={1}>
      <Text dimColor>Press [?] for help, [q] to quit</Text>
    </Box>
  );
}
