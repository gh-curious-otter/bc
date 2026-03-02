import { memo } from 'react';
import { Box, Text } from 'ink';

export interface MetricCardProps {
  value: number | string;
  label: string;
  color?: string;
  prefix?: string;
  suffix?: string;
}

/**
 * MetricCard - Compact metric display for summary dashboards
 * Shared component
 *
 * Memoized for performance - Issue #1003 Phase 3 optimization.
 */
export const MetricCard = memo(function MetricCard({
  value,
  label,
  color = 'white',
  prefix = '',
  suffix = '',
}: MetricCardProps) {
  return (
    <Box flexDirection="column" paddingX={1} marginRight={1} borderStyle="single" borderColor="gray" minHeight={4}>
      <Text dimColor>{label}</Text>
      <Text bold color={color}>
        {prefix}{value}{suffix}
      </Text>
    </Box>
  );
});

export default MetricCard;
