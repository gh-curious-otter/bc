/**
 * ToolsView - Display installed tools and their status
 * Issue #1866 - Tools view for bc tool list
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Box, Text } from 'ink';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { HeaderBar } from '../components/HeaderBar';
import { Footer } from '../components/Footer';
import { useDisableInput, useListNavigation, useLoadingTimeout } from '../hooks';
import { truncate } from '../utils';
import type { ToolInfo } from '../types';
import { getToolList } from '../services/bc';

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface ToolsViewProps {}

export function ToolsView(_props: ToolsViewProps = {}): React.ReactElement {
  const { isDisabled: disableInput } = useDisableInput();
  const [tools, setTools] = useState<ToolInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchTools = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getToolList();
      setTools(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch tools');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchTools();
  }, [fetchTools]);

  const customKeys = useMemo(
    () => ({
      'r': () => { void fetchTools(); },
    }),
    [fetchTools]
  );

  const { selectedIndex } = useListNavigation({
    items: tools,
    disabled: disableInput,
    customKeys,
  });

  // Dynamic name column width
  const nameWidth = useMemo(() => {
    if (tools.length === 0) return 15;
    const maxLen = Math.max(...tools.map((t) => t.name.length));
    return Math.min(25, Math.max(15, maxLen + 3));
  }, [tools]);

  // #1898: Track loading duration for timeout messages
  const loadingElapsed = useLoadingTimeout(loading && tools.length === 0);

  if (loading && tools.length === 0) {
    // After 10s: timeout with retry
    if (loadingElapsed >= 10) {
      return (
        <Box flexDirection="column" width="100%" overflow="hidden">
          <HeaderBar title="Tools" count={0} loading={false} color="cyan" />
          <Box paddingX={1} marginTop={1} flexDirection="column">
            <Text color="yellow">Some tools could not be checked — press [r] to retry</Text>
          </Box>
          <Footer hints={[{ key: 'r', label: 'refresh' }]} />
        </Box>
      );
    }

    // Loading with progressive message
    const loadingMsg = loadingElapsed >= 5
      ? 'Tools check taking longer than expected...'
      : 'Loading tools...';

    return (
      <Box flexDirection="column" width="100%" overflow="hidden">
        <HeaderBar title="Tools" count={0} loading={true} color="cyan" />
        <Box paddingX={1} marginTop={1}>
          <LoadingIndicator message={loadingMsg} />
        </Box>
        <Footer hints={[{ key: 'r', label: 'refresh' }]} />
      </Box>
    );
  }

  if (error && tools.length === 0) {
    return (
      <Box flexDirection="column" width="100%" overflow="hidden">
        <HeaderBar title="Tools" count={0} loading={false} color="cyan" />
        <Box paddingX={1} marginTop={1}>
          <Text color="red">Error: {error}</Text>
          <Text dimColor> — Press r to retry</Text>
        </Box>
        <Footer hints={[{ key: 'r', label: 'refresh' }]} />
      </Box>
    );
  }

  return (
    <Box flexDirection="column" width="100%" overflow="hidden">
      <HeaderBar
        title="Tools"
        count={tools.length}
        loading={loading}
        color="cyan"
      />

      {/* Table header */}
      <Box flexDirection="column" marginBottom={1}>
        <Box paddingX={1}>
          <Box width={nameWidth}>
            <Text bold dimColor>NAME</Text>
          </Box>
          <Box width={14}>
            <Text bold dimColor>STATUS</Text>
          </Box>
          <Box width={16}>
            <Text bold dimColor>VERSION</Text>
          </Box>
          <Box flexGrow={1}>
            <Text bold dimColor>COMMAND</Text>
          </Box>
        </Box>

        {tools.length === 0 ? (
          <Box paddingX={1} marginTop={1}>
            <Text dimColor>No tools found.</Text>
          </Box>
        ) : (
          tools.map((tool, idx) => (
            <ToolRow
              key={tool.name}
              tool={tool}
              selected={idx === selectedIndex}
              nameWidth={nameWidth}
            />
          ))
        )}
      </Box>

      {error && (
        <Box marginBottom={1} paddingX={1}>
          <Text color="red">Error: {error}</Text>
        </Box>
      )}

      <Footer hints={[
        { key: 'j/k', label: 'nav' },
        { key: 'g/G', label: 'top/bottom' },
        { key: 'r', label: 'refresh' },
      ]} />
    </Box>
  );
}

interface ToolRowProps {
  tool: ToolInfo;
  selected: boolean;
  nameWidth: number;
}

function ToolRow({ tool, selected, nameWidth }: ToolRowProps): React.ReactElement {
  const isInstalled = tool.status === 'installed';
  const statusColor = isInstalled ? 'green' : 'red';
  const statusIcon = isInstalled ? '✓' : '✗';
  const truncateLen = nameWidth - 3;

  return (
    <Box paddingX={1}>
      <Box width={nameWidth}>
        <Text color={selected ? 'cyan' : undefined} bold={selected}>
          {selected ? '▸ ' : '  '}
          {truncate(tool.name, truncateLen)}
        </Text>
      </Box>
      <Box width={14}>
        <Text color={statusColor}>
          {statusIcon} {tool.status}
        </Text>
      </Box>
      <Box width={16}>
        <Text dimColor>{truncate(tool.version || '-', 14)}</Text>
      </Box>
      <Box flexGrow={1}>
        <Text dimColor>{truncate(tool.command, 40)}</Text>
      </Box>
    </Box>
  );
}

export default ToolsView;
