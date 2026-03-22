/**
 * ProcessesView - Display background processes
 * Issue #1927 - k9s-style resource view for processes
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../theme';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { HeaderBar } from '../components/HeaderBar';
import { Footer } from '../components/Footer';
import { useDisableInput, useListNavigation, useLoadingTimeout } from '../hooks';
import { truncate } from '../utils';
import { getProcessList, type ProcessInfo } from '../services/bc';

export function ProcessesView(): React.ReactElement {
  const { theme } = useTheme();
  const { isDisabled: disableInput } = useDisableInput();
  const [processes, setProcesses] = useState<ProcessInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchProcesses = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getProcessList();
      setProcesses(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch processes');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchProcesses();
  }, [fetchProcesses]);

  const customKeys = useMemo(
    () => ({
      r: () => {
        void fetchProcesses();
      },
    }),
    [fetchProcesses]
  );

  const { selectedIndex } = useListNavigation({
    items: processes,
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
        <HeaderBar title="Processes" />
        <LoadingIndicator message="Loading processes..." />
        <Footer hints={viewHints} />
      </Box>
    );
  }

  if (error) {
    return (
      <Box flexDirection="column">
        <HeaderBar title="Processes" />
        <Box paddingLeft={1}>
          <Text color={theme.colors.error}>{error}</Text>
        </Box>
        <Footer hints={viewHints} />
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <HeaderBar title="Processes" count={processes.length} />

      {processes.length === 0 ? (
        <Box paddingLeft={1} paddingTop={1}>
          <Text dimColor>
            No background processes. Use &apos;bc process start&apos; to start one.
          </Text>
        </Box>
      ) : (
        <Box flexDirection="column" paddingTop={1}>
          <Box paddingLeft={1}>
            <Box width={20}>
              <Text bold>NAME</Text>
            </Box>
            <Box width={35}>
              <Text bold>COMMAND</Text>
            </Box>
            <Box width={12}>
              <Text bold>STATUS</Text>
            </Box>
            <Box width={10}>
              <Text bold>PID</Text>
            </Box>
          </Box>

          {processes.map((proc, index) => {
            const isSelected = index === selectedIndex;
            const statusColor =
              proc.status === 'running'
                ? theme.colors.success
                : proc.status === 'stopped'
                  ? theme.colors.error
                  : theme.colors.warning;
            return (
              <Box key={proc.name} paddingLeft={1}>
                <Box width={20}>
                  <Text inverse={isSelected} color={isSelected ? theme.colors.primary : undefined}>
                    {truncate(proc.name, 18)}
                  </Text>
                </Box>
                <Box width={35}>
                  <Text inverse={isSelected}>{truncate(proc.command, 33)}</Text>
                </Box>
                <Box width={12}>
                  <Text inverse={isSelected} color={statusColor}>
                    {proc.status}
                  </Text>
                </Box>
                <Box width={10}>
                  <Text inverse={isSelected} dimColor>
                    {proc.pid ?? '-'}
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
