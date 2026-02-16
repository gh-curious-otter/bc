import React from 'react';
import { Box, Text } from 'ink';

// Braille spinner frames for smooth animation
// Issue #974 - Visual design improvements
const SPINNER_FRAMES = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];

export interface LoadingIndicatorProps {
  message?: string;
  /** Spinner color (default: 'cyan') */
  color?: string;
  /** Animation interval in ms (default: 80) */
  interval?: number;
}

/**
 * LoadingIndicator - Loading state with animated spinner
 * Shared component with Braille spinner animation
 */
export function LoadingIndicator({
  message = 'Loading...',
  color = 'cyan',
  interval = 80,
}: LoadingIndicatorProps) {
  const [frameIndex, setFrameIndex] = React.useState(0);

  React.useEffect(() => {
    const timer = setInterval(() => {
      setFrameIndex((i) => (i + 1) % SPINNER_FRAMES.length);
    }, interval);
    return () => { clearInterval(timer); };
  }, [interval]);

  return (
    <Box>
      <Text color={color}>{SPINNER_FRAMES[frameIndex]}</Text>
      <Text> {message}</Text>
    </Box>
  );
}

export default LoadingIndicator;
