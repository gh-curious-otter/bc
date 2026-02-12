import React from 'react';
import { Box, Text } from 'ink';
// Note: ink-spinner will be available after Phase 1 PRs merge
// import Spinner from 'ink-spinner';

export interface LoadingIndicatorProps {
  message?: string;
}

/**
 * LoadingIndicator - Loading state with spinner
 * Shared component
 */
export function LoadingIndicator({ message = 'Loading...' }: LoadingIndicatorProps) {
  // Simple dots animation until ink-spinner is available
  const [dots, setDots] = React.useState('');

  React.useEffect(() => {
    const interval = setInterval(() => {
      setDots((d) => (d.length >= 3 ? '' : d + '.'));
    }, 300);
    return () => clearInterval(interval);
  }, []);

  return (
    <Box>
      <Text color="cyan">⠋</Text>
      <Text> {message}{dots}</Text>
    </Box>
  );
}

export default LoadingIndicator;
