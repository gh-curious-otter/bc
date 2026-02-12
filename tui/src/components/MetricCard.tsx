import React from 'react';
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
 */
export function MetricCard({
  value,
  label,
  color = 'white',
  prefix = '',
  suffix = '',
}: MetricCardProps) {
  return (
    <Box flexDirection="column" paddingX={2} borderStyle="single" borderColor="gray">
      <Text bold color={color}>
        {prefix}{value}{suffix}
      </Text>
      <Text dimColor>{label}</Text>
    </Box>
  );
}

export default MetricCard;
