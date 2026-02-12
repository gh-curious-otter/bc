import React from 'react';
import { Box, Text } from 'ink';

export interface ErrorDisplayProps {
  error: Error | string;
  onRetry?: () => void;
}

/**
 * ErrorDisplay - Error message display with retry option
 * Shared component
 */
export function ErrorDisplay({ error, onRetry }: ErrorDisplayProps) {
  const message = typeof error === 'string' ? error : error.message;

  return (
    <Box
      flexDirection="column"
      borderStyle="single"
      borderColor="red"
      padding={1}
    >
      <Text color="red" bold>
        Error
      </Text>
      <Text color="red">{message}</Text>
      {onRetry && (
        <Box marginTop={1}>
          <Text dimColor>Press 'r' to retry</Text>
        </Box>
      )}
    </Box>
  );
}

export default ErrorDisplay;
