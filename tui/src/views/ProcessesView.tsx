/**
 * ProcessesView - View for displaying managed processes
 * Issue #555: Processes view with list, details, and log viewer
 */

import { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import { useProcesses, useProcessLogs } from '../hooks';
import { Table } from '../components/Table';
import type { Column } from '../components/Table';
import { StatusBadge } from '../components/StatusBadge';
import type { Process } from '../types';

interface ProcessesViewProps {
  onBack?: () => void;
}

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

export function ProcessesView({ onBack }: ProcessesViewProps) {
  const { data: processes, loading, error, refresh } = useProcesses();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showLogs, setShowLogs] = useState(false);
  const processList = processes ?? [];

  const selectedProcess = processList[selectedIndex];

  // Keyboard navigation
  useInput((input, key) => {
    if (showLogs) {
      // Log viewer mode
      if (input === 'q' || key.escape) {
        setShowLogs(false);
      }
      return;
    }

    // List navigation mode
    if (key.upArrow || input === 'k') {
      setSelectedIndex((i) => Math.max(0, i - 1));
    } else if (key.downArrow || input === 'j') {
      setSelectedIndex((i) => Math.min(processList.length - 1, i + 1));
    } else if (key.return || input === 'l') {
      setShowLogs(true);
    } else if (input === 'r') {
      void refresh();
    } else if (input === 'q' || key.escape) {
      onBack?.();
    }
  });

  const columns: Column<Process>[] = [
    {
      key: 'name',
      header: 'Name',
      width: 20,
    },
    {
      key: 'running',
      header: 'Status',
      width: 10,
      render: (proc) => (
        <StatusBadge state={proc.running ? 'working' : 'stopped'} />
      ),
    },
    {
      key: 'pid',
      header: 'PID',
      width: 8,
      render: (proc) => <Text>{proc.pid > 0 ? proc.pid : '-'}</Text>,
    },
    {
      key: 'port',
      header: 'Port',
      width: 8,
      render: (proc) => <Text>{proc.port ?? '-'}</Text>,
    },
    {
      key: 'started_at',
      header: 'Uptime',
      width: 10,
      render: (proc) => (
        <Text>{proc.running ? formatUptime(proc.started_at) : '-'}</Text>
      ),
    },
    {
      key: 'command',
      header: 'Command',
      width: 30,
      render: (proc) => (
        <Text wrap="truncate">{proc.command ? proc.command.slice(0, 28) : '-'}</Text>
      ),
    },
  ];

  if (loading && processList.length === 0) {
    return (
      <Box padding={1}>
        <Text color="yellow">Loading processes...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Box padding={1}>
        <Text color="red">Error: {error}</Text>
      </Box>
    );
  }

  // Show log viewer
  if (showLogs) {
    return (
      <ProcessLogViewer
        process={selectedProcess}
        onBack={() => { setShowLogs(false); }}
      />
    );
  }

  return (
    <Box flexDirection="column">
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="magenta">
          Processes ({processList.length})
        </Text>
        {loading && <Text color="gray"> (refreshing...)</Text>}
      </Box>

      {processList.length === 0 ? (
        <Box padding={1}>
          <Text dimColor>No processes running</Text>
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

      {/* Footer with keybindings */}
      <Box marginTop={1}>
        <Text color="gray">
          j/k: navigate | Enter/l: logs | r: refresh | q: back
        </Text>
      </Box>
    </Box>
  );
}

interface ProcessLogViewerProps {
  process: Process;
  onBack: () => void;
}

function ProcessLogViewer({ process, onBack }: ProcessLogViewerProps) {
  const { data: logs, loading, error, refresh } = useProcessLogs({
    name: process.name,
    lines: 50,
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
          j/k: scroll | g/G: top/bottom | r: refresh | q: back
        </Text>
      </Box>
    </Box>
  );
}

export default ProcessesView;
