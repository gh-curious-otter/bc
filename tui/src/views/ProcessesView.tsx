/**
 * ProcessesView - View for displaying managed processes
 * Issue #555: Processes view with list, details, and log viewer
 * Issue #1723: Migrated to useListNavigation hook
 */

import { useState, useMemo, useCallback, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { useProcesses, useProcessLogs, useDebounce, useListNavigation } from '../hooks';
import { useFocus } from '../navigation/FocusContext';
import { Table } from '../components/Table';
import type { Column } from '../components/Table';
import { StatusBadge } from '../components/StatusBadge';
import { HeaderBar } from '../components/HeaderBar';
import { ViewWrapper } from '../components/ViewWrapper';
import { DATA_LIMITS } from '../constants';
import type { Process } from '../types';

/**
 * Calculate uptime string from started_at timestamp
 */
function formatUptime(startedAt: string): string {
  const start = new Date(startedAt);
  const now = new Date();
  const diffMs = now.getTime() - start.getTime();

  const seconds = Math.floor(diffMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) {
    return `${String(days)}d ${String(hours % 24)}h`;
  } else if (hours > 0) {
    return `${String(hours)}h ${String(minutes % 60)}m`;
  } else if (minutes > 0) {
    return `${String(minutes)}m ${String(seconds % 60)}s`;
  } else {
    return `${String(seconds)}s`;
  }
}

export function ProcessesView(): React.ReactElement {
  const { data: processes, loading, error, refresh } = useProcesses();
  const { setFocus } = useFocus();

  // #1723: View-specific modal state (not part of list navigation)
  const [showLogs, setShowLogs] = useState(false);

  // #1723: Search query state managed separately for debounce
  const [searchQuery, setSearchQuery] = useState('');

  // Debounce search query for filtering (issue #1602)
  const debouncedSearchQuery = useDebounce(searchQuery, 300);

  // Filter processes by search query (using debounced query for performance)
  const processList = useMemo(() => {
    const list = processes ?? [];
    if (!debouncedSearchQuery) return list;
    const query = debouncedSearchQuery.toLowerCase();
    return list.filter(
      (proc) =>
        proc.name.toLowerCase().includes(query) ||
        proc.command.toLowerCase().includes(query) ||
        (proc.owner?.toLowerCase().includes(query) ?? false)
    );
  }, [processes, debouncedSearchQuery]);

  // Callbacks for list navigation
  const handleSelect = useCallback((_process: Process) => {
    setShowLogs(true);
  }, []);

  const handleRefresh = useCallback(() => {
    void refresh();
  }, [refresh]);

  // #1723: Use useListNavigation hook for vim-style navigation
  const {
    selectedIndex,
    selectedItem: selectedProcess,
    search,
  } = useListNavigation({
    items: processList,
    enableSearch: true,
    onSearchChange: setSearchQuery,
    onSelect: handleSelect,
    customKeys: {
      'l': () => { if (processList.length > 0) setShowLogs(true); },
      'r': handleRefresh,
    },
    // Disable navigation when showing logs (handled by ProcessLogViewer)
    isActive: !showLogs,
  });

  // Manage focus state for nested view navigation (#1692)
  // When in search mode, set focus='input' to allow typing special chars
  useEffect(() => {
    if (showLogs) {
      setFocus('view');
    } else if (search.isActive) {
      setFocus('input');
    } else {
      setFocus('main');
    }
  }, [showLogs, search.isActive, setFocus]);

  // Column widths: 14+9+7+6+8+22 = 66 (fits 80-col terminal)
  const columns: Column<Process>[] = [
    {
      key: 'name',
      header: 'Name',
      width: 14,
      render: (proc) => (
        <Text>{proc.name.length > 12 ? proc.name.slice(0, 11) + '…' : proc.name}</Text>
      ),
    },
    {
      key: 'running',
      header: 'Status',
      width: 9,
      render: (proc) => (
        <StatusBadge state={proc.running ? 'working' : 'stopped'} />
      ),
    },
    {
      key: 'pid',
      header: 'PID',
      width: 7,
      render: (proc) => <Text>{proc.pid > 0 ? proc.pid : '-'}</Text>,
    },
    {
      key: 'port',
      header: 'Port',
      width: 6,
      render: (proc) => <Text>{proc.port ?? '-'}</Text>,
    },
    {
      key: 'started_at',
      header: 'Uptime',
      width: 8,
      render: (proc) => (
        <Text>{proc.running ? formatUptime(proc.started_at) : '-'}</Text>
      ),
    },
    {
      key: 'command',
      header: 'Command',
      width: 22,
      render: (proc) => (
        <Text wrap="truncate">{proc.command ? proc.command.slice(0, 20) : '-'}</Text>
      ),
    },
  ];

  // Search mode overlay
  if (search.isActive) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text bold>Search Processes</Text>
        <Box marginTop={1} borderStyle="single" borderColor="cyan" paddingX={1}>
          <Text color="cyan">{'> '}</Text>
          <Text>{search.query}</Text>
          <Text color="cyan">|</Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Enter to confirm, Esc to cancel</Text>
        </Box>
      </Box>
    );
  }

  // Show log viewer
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- defensive check for empty list
  if (showLogs && selectedProcess) {
    return (
      <ProcessLogViewer
        process={selectedProcess}
        onBack={() => { setShowLogs(false); }}
      />
    );
  }

  // Build hints array dynamically
  const hints = [
    { key: 'j/k', label: 'nav' },
    { key: 'g/G', label: 'top/bottom' },
    { key: '/', label: 'search' },
    ...(search.query ? [{ key: 'c', label: 'clear' }] : []),
    { key: 'Enter/l', label: 'logs' },
    { key: 'r', label: 'refresh' },
    { key: 'q/ESC', label: 'back' },
  ];

  return (
    <ViewWrapper
      loading={loading && processList.length === 0}
      loadingMessage="Loading processes..."
      error={error}
      onRetry={() => { void refresh(); }}
      hints={hints}
    >
      {/* Header with count (#1446) */}
      <HeaderBar
        title="Processes"
        count={processList.length}
        loading={loading && processList.length > 0}
        subtitle={search.query ? `[/] "${search.query}"` : undefined}
        color="blue"
      />
      {processList.length === 0 ? (
        <Box padding={1} flexDirection="column">
          <Text dimColor>{search.query ? 'No processes match search' : 'No processes running.'}</Text>
          {!search.query && <Text dimColor>Start one with: bc process start &lt;name&gt; &lt;command&gt;</Text>}
        </Box>
      ) : (
        <>
          {/* Process Table */}
          <Table
            data={processList}
            columns={columns}
            selectedIndex={selectedIndex}
          />

          {/* Process Details */}
          {/* eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- defensive check for empty list */}
          {selectedProcess && (
            <Box marginTop={1} flexDirection="column">
              <Text bold color="cyan">Details</Text>
              <Box marginLeft={1} flexDirection="column">
                <Text>
                  <Text dimColor>Owner: </Text>
                  {selectedProcess.owner ?? 'system'}
                </Text>
                <Text>
                  <Text dimColor>Work Dir: </Text>
                  {selectedProcess.work_dir ?? '-'}
                </Text>
                <Text>
                  <Text dimColor>Log File: </Text>
                  {selectedProcess.log_file ?? '-'}
                </Text>
              </Box>
            </Box>
          )}
        </>
      )}
    </ViewWrapper>
  );
}

interface ProcessLogViewerProps {
  process: Process;
  onBack: () => void;
}

function ProcessLogViewer({ process, onBack }: ProcessLogViewerProps) {
  const { data: logs, loading, error, refresh } = useProcessLogs({
    name: process.name,
    lines: DATA_LIMITS.PROCESS_LINES,
  });
  const [scrollOffset, setScrollOffset] = useState(0);
  const maxVisibleLines = 15;

  const logLines = logs ?? [];
  const visibleLogs = logLines.slice(
    scrollOffset,
    scrollOffset + maxVisibleLines
  );

  // Keyboard navigation for log scrolling
  useInput((input, key) => {
    if (key.upArrow || input === 'k') {
      setScrollOffset((o) => Math.max(0, o - 1));
    } else if (key.downArrow || input === 'j') {
      setScrollOffset((o) =>
        Math.min(Math.max(0, logLines.length - maxVisibleLines), o + 1)
      );
    } else if (input === 'g') {
      setScrollOffset(0);
    } else if (input === 'G') {
      setScrollOffset(Math.max(0, logLines.length - maxVisibleLines));
    } else if (input === 'r') {
      void refresh();
    } else if (input === 'q' || key.escape) {
      onBack();
    }
  });

  return (
    <Box flexDirection="column">
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="magenta">
          Logs: {process.name}
        </Text>
        {loading && <Text color="gray"> (loading...)</Text>}
        <Text dimColor> [{String(scrollOffset + 1)}-{String(Math.min(scrollOffset + maxVisibleLines, logLines.length))}/{String(logLines.length)}]</Text>
      </Box>

      {error ? (
        <Box padding={1}>
          <Text color="red">Error: {error}</Text>
        </Box>
      ) : logLines.length === 0 ? (
        <Box padding={1}>
          <Text dimColor>No logs available</Text>
        </Box>
      ) : (
        <Box
          flexDirection="column"
          borderStyle="single"
          borderColor="gray"
          padding={1}
          height={maxVisibleLines + 2}
        >
          {visibleLogs.map((line, idx) => (
            <Text key={scrollOffset + idx} wrap="truncate">
              {line}
            </Text>
          ))}
        </Box>
      )}

      {/* Footer with keybindings */}
      <Box marginTop={1}>
        <Text color="gray">
          j/k: scroll | g/G: top/bottom | r: refresh | q/ESC: back
        </Text>
      </Box>
    </Box>
  );
}

export default ProcessesView;
