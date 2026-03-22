/**
 * MCPView - Display MCP server configurations
 * Issue #1927 - k9s-style resource view for MCP servers
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../theme';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { HeaderBar } from '../components/HeaderBar';
import { Footer } from '../components/Footer';
import { useDisableInput, useListNavigation, useLoadingTimeout } from '../hooks';
import { truncate } from '../utils';
import { getMCPList, type MCPServer } from '../services/bc';

export function MCPView(): React.ReactElement {
  const { theme } = useTheme();
  const { isDisabled: disableInput } = useDisableInput();
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchServers = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getMCPList();
      setServers(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch MCP servers');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchServers();
  }, [fetchServers]);

  const customKeys = useMemo(
    () => ({
      r: () => {
        void fetchServers();
      },
    }),
    [fetchServers]
  );

  const { selectedIndex } = useListNavigation({
    items: servers,
    disabled: disableInput,
    customKeys,
  });

  const showTimeout = useLoadingTimeout(loading);

  const viewHints = [
    { key: 'r', label: 'refresh', priority: 10 },
    { key: 'j/k', label: 'navigate', priority: 11 },
  ];

  if (loading && showTimeout) {
    return (
      <Box flexDirection="column">
        <HeaderBar title="MCP Servers" />
        <LoadingIndicator message="Loading MCP servers..." />
        <Footer hints={viewHints} />
      </Box>
    );
  }

  if (error) {
    return (
      <Box flexDirection="column">
        <HeaderBar title="MCP Servers" />
        <Box paddingLeft={1}>
          <Text color={theme.colors.error}>{error}</Text>
        </Box>
        <Footer hints={viewHints} />
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <HeaderBar title="MCP Servers" count={servers.length} />

      {servers.length === 0 ? (
        <Box paddingLeft={1} paddingTop={1}>
          <Text dimColor>No MCP servers configured. Use &apos;bc mcp add&apos; to add one.</Text>
        </Box>
      ) : (
        <Box flexDirection="column" paddingTop={1}>
          <Box paddingLeft={1}>
            <Box width={20}>
              <Text bold>NAME</Text>
            </Box>
            <Box width={10}>
              <Text bold>TRANSPORT</Text>
            </Box>
            <Box width={40}>
              <Text bold>COMMAND/URL</Text>
            </Box>
            <Box width={10}>
              <Text bold>ENABLED</Text>
            </Box>
          </Box>

          {servers.map((server, index) => {
            const isSelected = index === selectedIndex;
            const target = server.transport === 'sse' ? (server.url ?? '') : (server.command ?? '');
            return (
              <Box key={server.name} paddingLeft={1}>
                <Box width={20}>
                  <Text inverse={isSelected} color={isSelected ? theme.colors.primary : undefined}>
                    {truncate(server.name, 18)}
                  </Text>
                </Box>
                <Box width={10}>
                  <Text inverse={isSelected}>{server.transport}</Text>
                </Box>
                <Box width={40}>
                  <Text inverse={isSelected}>{truncate(target, 38)}</Text>
                </Box>
                <Box width={10}>
                  <Text
                    inverse={isSelected}
                    color={server.enabled ? theme.colors.success : theme.colors.error}
                  >
                    {server.enabled ? 'yes' : 'no'}
                  </Text>
                </Box>
              </Box>
            );
          })}
        </Box>
      )}

      <Footer hints={viewHints} />
    </Box>
  );
}
