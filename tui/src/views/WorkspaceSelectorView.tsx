/**
 * WorkspaceSelectorView - Workspace discovery and selection (#922)
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { getWorkspaces } from '../services/bc';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import type { DiscoveredWorkspace } from '../types';

interface WorkspaceSelectorViewProps {
  onSelect?: (workspace: DiscoveredWorkspace) => void;
}

/**
 * Format path for display - show shortened home path
 */
function formatPath(fullPath: string): string {
  const home = process.env.HOME ?? '';
  if (home && fullPath.startsWith(home)) {
    return '~' + fullPath.slice(home.length);
  }
  return fullPath;
}

export const WorkspaceSelectorView: React.FC<WorkspaceSelectorViewProps> = ({
  onSelect,
}) => {
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;
  const { setFocus } = useFocus();
  const { goHome } = useNavigation();

  const [workspaces, setWorkspaces] = useState<DiscoveredWorkspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showDetail, setShowDetail] = useState(false);
  const [filterV2Only, setFilterV2Only] = useState(false);

  // Set focus to view on mount for ESC hierarchy
  useEffect(() => {
    setFocus('view');
  }, [setFocus]);

  const fetchWorkspaces = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getWorkspaces();
      setWorkspaces(data.workspaces);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch workspaces');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchWorkspaces();
  }, [fetchWorkspaces]);

  // Filter workspaces based on view mode
  const filteredWorkspaces = useMemo(() => {
    if (!filterV2Only) return workspaces;
    return workspaces.filter((ws) => ws.is_v2);
  }, [workspaces, filterV2Only]);

  // Separate registered and discovered
  const registeredWorkspaces = useMemo(
    () => filteredWorkspaces.filter((ws) => ws.from_cache),
    [filteredWorkspaces]
  );
  const discoveredWorkspaces = useMemo(
    () => filteredWorkspaces.filter((ws) => !ws.from_cache),
    [filteredWorkspaces]
  );

  const selectedWorkspace = filteredWorkspaces[selectedIndex] as DiscoveredWorkspace | undefined;
  const v2Count = workspaces.filter((ws) => ws.is_v2).length;

  // Keyboard navigation
  useInput((input, key) => {
    if (showDetail) {
      if (key.escape || input === 'q' || key.return) {
        setShowDetail(false);
      }
      return;
    }

    // List navigation
    if (key.upArrow || input === 'k') {
      setSelectedIndex((i) => Math.max(0, i - 1));
    } else if (key.downArrow || input === 'j') {
      setSelectedIndex((i) => Math.min(filteredWorkspaces.length - 1, i + 1));
    } else if (input === 'g') {
      setSelectedIndex(0);
    } else if (input === 'G') {
      setSelectedIndex(Math.max(0, filteredWorkspaces.length - 1));
    } else if (key.return) {
      if (selectedWorkspace) {
        if (onSelect) {
          onSelect(selectedWorkspace);
        } else {
          setShowDetail(true);
        }
      }
    } else if (input === 'v') {
      setFilterV2Only(!filterV2Only);
      setSelectedIndex(0);
    } else if (input === 'r') {
      void fetchWorkspaces();
    } else if (input === 'q' || key.escape) {
      setFocus('main');
      goHome();
    }
  });

  // Detail view
  if (showDetail && selectedWorkspace) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text bold color="cyan">Workspace Details</Text>
        <Box marginTop={1} flexDirection="column" borderStyle="single" borderColor="gray" padding={1}>
          <Box>
            <Text bold>Name: </Text>
            <Text color="cyan">{selectedWorkspace.name}</Text>
          </Box>
          <Box>
            <Text bold>Path: </Text>
            <Text>{selectedWorkspace.path}</Text>
          </Box>
          <Box>
            <Text bold>Config: </Text>
            <Text color={selectedWorkspace.is_v2 ? 'green' : 'yellow'}>
              {selectedWorkspace.is_v2 ? 'v2 (TOML)' : 'v1 (JSON)'}
            </Text>
          </Box>
          <Box>
            <Text bold>Source: </Text>
            <Text color={selectedWorkspace.from_cache ? 'blue' : 'gray'}>
              {selectedWorkspace.from_cache ? 'Registered' : 'Discovered'}
            </Text>
          </Box>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Press any key to return</Text>
        </Box>
      </Box>
    );
  }

  if (loading && workspaces.length === 0) {
    return <LoadingIndicator message="Discovering workspaces..." />;
  }

  if (error) {
    return (
      <Box padding={1}>
        <Text color="red">Error: {error}</Text>
      </Box>
    );
  }

  // Calculate column widths
  const nameWidth = 20;
  const typeWidth = 8;
  const pathWidth = Math.min(50, terminalWidth - nameWidth - typeWidth - 10);

  const renderWorkspaceRow = (ws: DiscoveredWorkspace, actualIdx: number) => {
    const isSelected = actualIdx === selectedIndex;
    return (
      <Box key={ws.path}>
        <Text
          backgroundColor={isSelected ? 'blue' : undefined}
          color={isSelected ? 'white' : 'cyan'}
        >
          {ws.name.slice(0, nameWidth - 1).padEnd(nameWidth)}
        </Text>
        <Text
          backgroundColor={isSelected ? 'blue' : undefined}
          color={isSelected ? 'white' : ws.is_v2 ? 'green' : 'yellow'}
        >
          {(ws.is_v2 ? 'v2' : 'v1').padEnd(typeWidth)}
        </Text>
        <Text
          backgroundColor={isSelected ? 'blue' : undefined}
          color={isSelected ? 'white' : undefined}
          wrap="truncate"
        >
          {formatPath(ws.path).slice(0, pathWidth)}
        </Text>
      </Box>
    );
  };

  return (
    <Box flexDirection="column">
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="blue">Workspaces</Text>
        <Text dimColor> ({registeredWorkspaces.length} registered</Text>
        {discoveredWorkspaces.length > 0 && (
          <Text color="gray">, {discoveredWorkspaces.length} discovered</Text>
        )}
        <Text dimColor>)</Text>
        {loading && <Text color="gray"> (refreshing...)</Text>}
      </Box>

      {/* Filter indicator */}
      {filterV2Only && (
        <Box marginBottom={1}>
          <Text color="green">[Showing v2 only] ({v2Count} workspaces)</Text>
        </Box>
      )}

      {/* Workspace table */}
      <Box flexDirection="column" borderStyle="single" borderColor="gray">
        {/* Header */}
        <Box>
          <Text bold color="gray">
            {'NAME'.padEnd(nameWidth)}
            {'TYPE'.padEnd(typeWidth)}
            {'PATH'}
          </Text>
        </Box>

        {/* Registered workspaces */}
        {registeredWorkspaces.length > 0 && (
          <>
            <Box>
              <Text dimColor>Registered:</Text>
            </Box>
            {registeredWorkspaces.map((ws) => {
              const actualIdx = filteredWorkspaces.indexOf(ws);
              return renderWorkspaceRow(ws, actualIdx);
            })}
          </>
        )}

        {/* Separator if both types exist */}
        {registeredWorkspaces.length > 0 && discoveredWorkspaces.length > 0 && (
          <Box>
            <Text dimColor>{'─'.repeat(terminalWidth - 4)}</Text>
          </Box>
        )}

        {/* Discovered workspaces */}
        {discoveredWorkspaces.length > 0 && (
          <>
            <Box>
              <Text dimColor>Discovered:</Text>
            </Box>
            {discoveredWorkspaces.map((ws) => {
              const actualIdx = filteredWorkspaces.indexOf(ws);
              return renderWorkspaceRow(ws, actualIdx);
            })}
          </>
        )}

        {filteredWorkspaces.length === 0 && (
          <Box padding={1}>
            <Text dimColor>No workspaces found</Text>
          </Box>
        )}
      </Box>

      {/* Footer */}
      <Box marginTop={1}>
        <Text dimColor>
          j/k: nav | g/G: top/bottom | Enter: {onSelect ? 'select' : 'details'} | v: {filterV2Only ? 'show all' : 'v2 only'} | r: refresh | q/ESC: back
        </Text>
      </Box>
    </Box>
  );
};

export default WorkspaceSelectorView;
