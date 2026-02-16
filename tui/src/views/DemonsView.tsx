/**
 * DemonsView - Scheduled tasks list view
 * Issue #554 - Demons list view
 */

import React, { useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { useDemons } from '../hooks/useDemons';
import { StatusBadge } from '../components/StatusBadge';
import { Footer } from '../components/Footer';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { ErrorDisplay } from '../components/ErrorDisplay';
import type { Demon } from '../types';

/** Duration in ms to show action errors before auto-clearing */
const ERROR_DISPLAY_DURATION = 3000;

export interface DemonsViewProps {
  /** Callback when exiting the view */
  onExit?: () => void;
  /** Disable input handling (useful for testing) */
  disableInput?: boolean;
}

/**
 * Format cron schedule to human-readable string
 */
function formatSchedule(schedule: string): string {
  // Common patterns
  if (schedule === '* * * * *') return 'every minute';
  if (schedule === '0 * * * *') return 'every hour';
  if (schedule.startsWith('*/')) {
    const match = schedule.match(/^\*\/(\d+) \* \* \* \*$/);
    if (match) return `every ${match[1]} min`;
  }
  if (schedule.match(/^0 \d+ \* \* \*$/)) {
    const hour = schedule.split(' ')[1];
    return `daily at ${hour}:00`;
  }
  return schedule;
}

/**
 * Format relative time for last/next run
 */
function formatRelativeTime(timestamp?: string): string {
  if (!timestamp) return '-';
  try {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(Math.abs(diffMs) / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    const prefix = diffMs < 0 ? 'in ' : '';
    const suffix = diffMs >= 0 ? ' ago' : '';

    if (diffMins < 1) return 'now';
    if (diffMins < 60) return `${prefix}${String(diffMins)}m${suffix}`;
    if (diffHours < 24) return `${prefix}${String(diffHours)}h${suffix}`;
    return `${prefix}${String(diffDays)}d${suffix}`;
  } catch {
    return timestamp;
  }
}

/**
 * DemonsView - Display list of scheduled tasks (demons)
 *
 * Features:
 * - List all configured demons
 * - Show schedule, status, run history
 * - Keyboard navigation (j/k, e/d to enable/disable, r to run)
 */
export function DemonsView({
  onExit,
  disableInput = false,
}: DemonsViewProps): React.ReactElement {
  const { data: demons, loading, error, total, enabled, refresh, enable, disable, run } = useDemons();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [actionError, setActionError] = useState<string | null>(null);

  // Auto-clear action errors after a delay
  useEffect(() => {
    if (!actionError) return;
    const timer = setTimeout(() => { setActionError(null); }, ERROR_DISPLAY_DURATION);
    return () => { clearTimeout(timer); };
  }, [actionError]);

  useInput(
    (input, key) => {
      if (!demons || demons.length === 0) return;

      // Navigation
      if (input === 'j' || key.downArrow) {
        setSelectedIndex((prev) => Math.min(prev + 1, demons.length - 1));
      }
      if (input === 'k' || key.upArrow) {
        setSelectedIndex((prev) => Math.max(prev - 1, 0));
      }
      if (input === 'g') {
        setSelectedIndex(0);
      }
      if (input === 'G') {
        setSelectedIndex(demons.length - 1);
      }

      // Actions
      if (input === 'r') {
        void refresh();
      }
      if (input === 'q' && onExit) {
        onExit();
      }

      // Demon-specific actions
      const selectedDemon = demons[selectedIndex] as typeof demons[number] | undefined;
      if (selectedDemon) {
        if (input === 'e') {
          // Enable demon
          enable(selectedDemon.name).catch((err: unknown) => {
            const message = err instanceof Error ? err.message : String(err);
            setActionError(`Enable failed: ${message}`);
          });
        }
        if (input === 'd') {
          // Disable demon
          disable(selectedDemon.name).catch((err: unknown) => {
            const message = err instanceof Error ? err.message : String(err);
            setActionError(`Disable failed: ${message}`);
          });
        }
        if (input === 'x') {
          // Execute demon
          run(selectedDemon.name).catch((err: unknown) => {
            const message = err instanceof Error ? err.message : String(err);
            setActionError(`Run failed: ${message}`);
          });
        }
      }
    },
    { isActive: !disableInput }
  );

  if (error) {
    return <ErrorDisplay error={error} onRetry={() => { void refresh(); }} />;
  }

  if (loading && !demons) {
    return <LoadingIndicator message="Loading demons..." />;
  }

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="magenta">Demons</Text>
        <Text> · </Text>
        <Text dimColor>{total} total</Text>
        <Text dimColor> · </Text>
        <Text color={enabled > 0 ? 'green' : 'gray'}>{enabled} enabled</Text>
      </Box>

      {/* Action error feedback */}
      {actionError && (
        <Box marginBottom={1}>
          <Text color="red">{actionError}</Text>
        </Box>
      )}

      {/* Demon list */}
      {demons && demons.length > 0 ? (
        <Box flexDirection="column">
          {/* Header row */}
          <Box marginBottom={1}>
            <Box width={3}><Text> </Text></Box>
            <Box width={18}><Text bold dimColor>NAME</Text></Box>
            <Box width={16}><Text bold dimColor>SCHEDULE</Text></Box>
            <Box width={10}><Text bold dimColor>STATUS</Text></Box>
            <Box width={10}><Text bold dimColor>RUNS</Text></Box>
            <Box width={12}><Text bold dimColor>LAST RUN</Text></Box>
            <Box width={12}><Text bold dimColor>NEXT RUN</Text></Box>
          </Box>

          {/* Demon rows */}
          {demons.map((demon, index) => (
            <DemonRow
              key={demon.name}
              demon={demon}
              selected={index === selectedIndex}
            />
          ))}
        </Box>
      ) : (
        <Box flexDirection="column" paddingY={2}>
          <Text dimColor>No demons configured</Text>
          <Text dimColor>Create one with: bc demon create {'<name>'} --schedule {'\'<cron>\''} --cmd {'\'<command>\''}</Text>
        </Box>
      )}

      {/* Selected demon details */}
      {demons && demons.length > 0 && demons[selectedIndex] && (
        <Box marginTop={1} borderStyle="single" borderColor="gray" padding={1} flexDirection="column">
          <Text bold>{demons[selectedIndex].name}</Text>
          <Box marginTop={1}>
            <Text dimColor>Command: </Text>
            <Text>{demons[selectedIndex].command}</Text>
          </Box>
          {demons[selectedIndex].description && (
            <Box>
              <Text dimColor>Description: </Text>
              <Text>{demons[selectedIndex].description}</Text>
            </Box>
          )}
          {demons[selectedIndex].owner && (
            <Box>
              <Text dimColor>Owner: </Text>
              <Text>{demons[selectedIndex].owner}</Text>
            </Box>
          )}
        </Box>
      )}

      {/* Footer */}
      <Footer
        hints={[
          { key: 'j/k', label: 'navigate' },
          { key: 'e', label: 'enable' },
          { key: 'd', label: 'disable' },
          { key: 'x', label: 'run' },
          { key: 'r', label: 'refresh' },
          { key: 'q', label: 'back' },
        ]}
      />
    </Box>
  );
}

interface DemonRowProps {
  demon: Demon;
  selected: boolean;
}

function DemonRow({ demon, selected }: DemonRowProps): React.ReactElement {
  const statusText = demon.enabled ? 'enabled' : 'disabled';

  return (
    <Box>
      <Box width={3}>
        <Text color={selected ? 'cyan' : undefined}>
          {selected ? '▸ ' : '  '}
        </Text>
      </Box>
      <Box width={18}>
        <Text color={selected ? 'cyan' : undefined} bold={selected}>
          {demon.name.length > 16 ? demon.name.slice(0, 14) + '..' : demon.name}
        </Text>
      </Box>
      <Box width={16}>
        <Text dimColor>{formatSchedule(demon.schedule)}</Text>
      </Box>
      <Box width={10}>
        <StatusBadge state={statusText} showIcon={false} />
      </Box>
      <Box width={10}>
        <Text>{demon.run_count}</Text>
      </Box>
      <Box width={12}>
        <Text dimColor>{formatRelativeTime(demon.last_run)}</Text>
      </Box>
      <Box width={12}>
        <Text color={demon.enabled ? 'yellow' : 'gray'}>
          {formatRelativeTime(demon.next_run)}
        </Text>
      </Box>
    </Box>
  );
}

export default DemonsView;
