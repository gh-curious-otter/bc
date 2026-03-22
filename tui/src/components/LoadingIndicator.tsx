import React, { memo } from 'react';
import { Box, Text } from 'ink';

// Braille spinner frames for smooth animation
// Issue #974 - Visual design improvements
// Issue #1198 - 60fps baseline
const SPINNER_FRAMES = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];

// Alternative spinner styles for variety
const SPINNER_DOTS = ['⣾', '⣽', '⣻', '⢿', '⡿', '⣟', '⣯', '⣷'];
const SPINNER_LINE = ['|', '/', '-', '\\'];
const SPINNER_CIRCLE = ['◐', '◓', '◑', '◒'];

export type SpinnerStyle = 'braille' | 'dots' | 'line' | 'circle';

const SPINNER_STYLES: Record<SpinnerStyle, string[]> = {
  braille: SPINNER_FRAMES,
  dots: SPINNER_DOTS,
  line: SPINNER_LINE,
  circle: SPINNER_CIRCLE,
};

export interface LoadingIndicatorProps {
  message?: string;
  /** Spinner color (default: 'cyan') */
  color?: string;
  /** Animation interval in ms (default: 50 for ~60fps visual smoothness) */
  interval?: number;
  /** Spinner style (default: 'braille') */
  style?: SpinnerStyle;
}

/**
 * LoadingIndicator - Loading state with animated spinner
 * Shared component with smooth 60fps-targeted animation.
 * Issue #1198: 60fps baseline for animations.
 *
 * Memoized for performance - Issue #1003 Phase 3 optimization.
 */
export const LoadingIndicator = memo(function LoadingIndicator({
  message = 'Loading...',
  color = 'cyan',
  interval = 50, // ~20fps with 10 frames = 2 full cycles/sec (visually smooth)
  style = 'braille',
}: LoadingIndicatorProps) {
  const [frameIndex, setFrameIndex] = React.useState(0);
  const frames = SPINNER_STYLES[style];

  React.useEffect(() => {
    const timer = setInterval(() => {
      setFrameIndex((i) => (i + 1) % frames.length);
    }, interval);
    return () => {
      clearInterval(timer);
    };
  }, [interval, frames.length]);

  return (
    <Box>
      <Text color={color}>{frames[frameIndex]}</Text>
      <Text> {message}</Text>
    </Box>
  );
});

/**
 * Spinner - Standalone spinner without message
 * For inline loading states.
 */
export const Spinner = memo(function Spinner({
  color = 'cyan',
  interval = 50,
  style = 'braille',
}: Omit<LoadingIndicatorProps, 'message'>) {
  const [frameIndex, setFrameIndex] = React.useState(0);
  const frames = SPINNER_STYLES[style];

  React.useEffect(() => {
    const timer = setInterval(() => {
      setFrameIndex((i) => (i + 1) % frames.length);
    }, interval);
    return () => {
      clearInterval(timer);
    };
  }, [interval, frames.length]);

  return <Text color={color}>{frames[frameIndex]}</Text>;
});

export default LoadingIndicator;
