import { Box, Text } from 'ink';
import { useTheme } from '../theme';

export interface ErrorDisplayProps {
  error: Error | string;
  onRetry?: () => void;
}

/**
 * ErrorDisplay - Error message display with retry option
 * Shared component
 */
export function ErrorDisplay({ error, onRetry }: ErrorDisplayProps) {
  const { theme } = useTheme();
  const message = typeof error === 'string' ? error : error.message;

  return (
    <Box flexDirection="column" borderStyle="single" borderColor={theme.colors.error} padding={1}>
      <Text color={theme.colors.error} bold>
        Error
      </Text>
      <Text color={theme.colors.error}>{message}</Text>
      {onRetry && (
        <Box marginTop={1}>
          <Text dimColor>Press &apos;r&apos; to retry</Text>
        </Box>
      )}
    </Box>
  );
}

export default ErrorDisplay;
