/**
 * AnimatedText - Terminal-based text animation components
 * Issue #1024: Animations and visual effects
 * Issue #1210: Reduced-motion and animation accessibility support
 *
 * Provides animated text effects for terminal UI:
 * - FadeText: Fade in/out effect
 * - PulseText: Pulsing brightness
 * - TypewriterText: Character-by-character reveal
 * - BlinkText: Simple blink effect
 * - Spinner: Smooth loading spinner
 * - AnimatedProgressBar: Smooth progress bar
 * - AnimatedCounter: Animated number display
 * - LoadingDots: Animated loading dots
 * - WaveText: Text with wave animation
 *
 * All components respect reduced-motion accessibility preferences.
 */

import { memo, useMemo } from 'react';
import { Text, Box } from 'ink';
import type { TextProps } from 'ink';
import {
  useFade,
  usePulse,
  useTypewriter,
  useBlink,
  useSpinner,
  useProgressAnimation,
  useCounter,
  SPINNER_FRAMES,
} from '../hooks/useAnimation';
import type { FadeDirection } from '../hooks/useAnimation';

/** Common props for all animated text components */
interface BaseAnimatedTextProps extends Omit<TextProps, 'children'> {
  /** Text content to animate */
  children: string;
}

/** FadeText props */
export interface FadeTextProps extends BaseAnimatedTextProps {
  /** Fade direction (default: 'in') */
  direction?: FadeDirection;
  /** Duration in ms (default: 200) */
  duration?: number;
  /** Callback when fade completes */
  onComplete?: () => void;
}

/**
 * FadeText - Text that fades in or out
 *
 * Uses dimColor prop to simulate opacity in terminal.
 */
export const FadeText = memo(function FadeText({
  children,
  direction = 'in',
  duration = 200,
  onComplete,
  ...textProps
}: FadeTextProps) {
  const { isDim, isComplete } = useFade({ direction, duration, onComplete });

  // In terminal, we can only do dim/not dim
  // For fade-out that's complete, we hide the text
  if (direction === 'out' && isComplete) {
    return null;
  }

  return (
    <Text {...textProps} dimColor={isDim}>
      {children}
    </Text>
  );
});

/** PulseText props */
export interface PulseTextProps extends BaseAnimatedTextProps {
  /** Pulse interval in ms (default: 1000) */
  interval?: number;
  /** Enable pulse (default: true) */
  enabled?: boolean;
}

/**
 * PulseText - Text that pulses between dim and bright
 *
 * Useful for indicating active/processing states.
 */
export const PulseText = memo(function PulseText({
  children,
  interval = 1000,
  enabled = true,
  ...textProps
}: PulseTextProps) {
  const { isDim } = usePulse({ interval, enabled });

  return (
    <Text {...textProps} dimColor={isDim}>
      {children}
    </Text>
  );
});

/** TypewriterText props */
export interface TypewriterTextProps extends BaseAnimatedTextProps {
  /** Characters per second (default: 30) */
  speed?: number;
  /** Delay before start in ms (default: 0) */
  delay?: number;
  /** Show cursor at end (default: true) */
  showCursor?: boolean;
  /** Cursor character (default: '▌') */
  cursor?: string;
  /** Callback when complete */
  onComplete?: () => void;
}

/**
 * TypewriterText - Text revealed character by character
 *
 * Classic typewriter effect for dramatic reveals.
 */
export const TypewriterText = memo(function TypewriterText({
  children,
  speed = 30,
  delay = 0,
  showCursor = true,
  cursor = '▌',
  onComplete,
  ...textProps
}: TypewriterTextProps) {
  const { displayText, isComplete } = useTypewriter({
    text: children,
    speed,
    delay,
    onComplete,
  });

  return (
    <Text {...textProps}>
      {displayText}
      {showCursor && !isComplete && (
        <Text color="cyan">{cursor}</Text>
      )}
    </Text>
  );
});

/** BlinkText props */
export interface BlinkTextProps extends BaseAnimatedTextProps {
  /** Blink interval in ms (default: 500) */
  interval?: number;
  /** Enable blink (default: true) */
  enabled?: boolean;
}

/**
 * BlinkText - Text that blinks on and off
 *
 * Use sparingly - blinking can be distracting.
 * Good for critical alerts or cursor effects.
 */
export const BlinkText = memo(function BlinkText({
  children,
  interval = 500,
  enabled = true,
  ...textProps
}: BlinkTextProps) {
  const { isVisible } = useBlink({ interval, enabled });

  if (!isVisible) {
    // Return empty space to maintain layout
    return <Text>{' '.repeat(children.length)}</Text>;
  }

  return <Text {...textProps}>{children}</Text>;
});

/** StatusTransition props */
export interface StatusTransitionProps {
  /** Previous status text */
  from?: string;
  /** Current status text */
  to: string;
  /** Status color */
  color?: string;
  /** Show transition animation */
  animate?: boolean;
  /** Transition duration in ms (default: 300) */
  duration?: number;
}

/**
 * StatusTransition - Animated status change indicator
 *
 * Shows smooth transition when status changes.
 */
export const StatusTransition = memo(function StatusTransition({
  from,
  to,
  color,
  animate = true,
  duration = 300,
}: StatusTransitionProps) {
  const { isDim } = useFade({
    direction: 'in',
    duration,
    autoStart: animate && from !== undefined && from !== to,
  });

  // If no previous state or same state, just show current
  if (!from || from === to || !animate) {
    return <Text color={color}>{to}</Text>;
  }

  return (
    <Text color={color} dimColor={isDim}>
      {to}
    </Text>
  );
});

/** NotificationText props */
export interface NotificationTextProps extends BaseAnimatedTextProps {
  /** Notification type for color */
  type?: 'info' | 'success' | 'warning' | 'error';
  /** Auto-dismiss after ms (0 = no dismiss) */
  dismissAfter?: number;
  /** Callback when dismissed */
  onDismiss?: () => void;
}

const notificationColors: Record<string, string> = {
  info: 'blue',
  success: 'green',
  warning: 'yellow',
  error: 'red',
};

/**
 * NotificationText - Animated notification with auto-dismiss
 *
 * Fades in, optionally pulses, then fades out.
 */
export const NotificationText = memo(function NotificationText({
  children,
  type = 'info',
  dismissAfter = 0,
  onDismiss,
  ...textProps
}: NotificationTextProps) {
  const color = notificationColors[type];
  const { isDim, isComplete } = useFade({
    direction: dismissAfter > 0 ? 'out' : 'in',
    duration: 200,
    autoStart: dismissAfter > 0,
    onComplete: dismissAfter > 0 ? onDismiss : undefined,
  });

  if (dismissAfter > 0 && isComplete) {
    return null;
  }

  return (
    <Text {...textProps} color={color} dimColor={isDim}>
      {children}
    </Text>
  );
});

// ============================================================================
// Extended Animated Components (Issue #1210)
// ============================================================================

/** Spinner props */
export interface SpinnerProps {
  /** Spinner type (default: 'dots') */
  type?: keyof typeof SPINNER_FRAMES;
  /** Custom frames */
  frames?: string[];
  /** Frame interval in ms (default: 80) */
  interval?: number;
  /** Spinner color */
  color?: string;
  /** Text to show after spinner */
  text?: string;
  /** Enable spinner (default: true) */
  enabled?: boolean;
}

/**
 * Spinner - Smooth loading spinner animation
 *
 * Respects reduced-motion - shows static indicator when disabled.
 */
export const Spinner = memo(function Spinner({
  type = 'dots',
  frames,
  interval = 80,
  color = 'cyan',
  text,
  enabled = true,
}: SpinnerProps) {
  const spinnerFrames = frames ?? SPINNER_FRAMES[type];
  const { frame } = useSpinner({ frames: spinnerFrames, interval, enabled });

  return (
    <Box>
      <Text color={color}>{frame}</Text>
      {text && <Text> {text}</Text>}
    </Box>
  );
});

/** AnimatedProgressBar props */
export interface AnimatedProgressBarProps {
  /** Progress value 0-100 */
  value: number;
  /** Bar width in characters (default: 20) */
  width?: number;
  /** Filled character (default: '█') */
  filledChar?: string;
  /** Empty character (default: '░') */
  emptyChar?: string;
  /** Progress bar color */
  color?: string;
  /** Show percentage (default: true) */
  showPercent?: boolean;
  /** Animation duration in ms (default: 300) */
  duration?: number;
}

/**
 * AnimatedProgressBar - Smooth progress bar animation
 *
 * Smoothly animates between progress values.
 * Respects reduced-motion - snaps to value when disabled.
 */
export const AnimatedProgressBar = memo(function AnimatedProgressBar({
  value,
  width = 20,
  filledChar = '█',
  emptyChar = '░',
  color = 'green',
  showPercent = true,
  duration = 300,
}: AnimatedProgressBarProps) {
  // Normalize to 0-1 range
  const normalizedProgress = Math.max(0, Math.min(1, value / 100));

  const { animatedProgress } = useProgressAnimation({
    progress: normalizedProgress,
    duration,
  });

  const filledWidth = Math.round(animatedProgress * width);
  const emptyWidth = width - filledWidth;

  const filled = filledChar.repeat(filledWidth);
  const empty = emptyChar.repeat(emptyWidth);
  const percent = Math.round(animatedProgress * 100);

  return (
    <Box>
      <Text color={color}>{filled}</Text>
      <Text dimColor>{empty}</Text>
      {showPercent && <Text> {percent}%</Text>}
    </Box>
  );
});

/** AnimatedCounter props */
export interface AnimatedCounterProps {
  /** Target value */
  value: number;
  /** Animation duration in ms (default: 500) */
  duration?: number;
  /** Number of decimal places (default: 0) */
  decimals?: number;
  /** Prefix text (e.g., '$') */
  prefix?: string;
  /** Suffix text (e.g., '%') */
  suffix?: string;
  /** Text color */
  color?: string;
  /** Text props */
  bold?: boolean;
}

/**
 * AnimatedCounter - Smooth number animation
 *
 * Smoothly counts up/down to target value.
 * Respects reduced-motion - shows final value when disabled.
 */
export const AnimatedCounter = memo(function AnimatedCounter({
  value,
  duration = 500,
  decimals = 0,
  prefix = '',
  suffix = '',
  color,
  bold,
}: AnimatedCounterProps) {
  const { displayValue } = useCounter({ value, duration, decimals });

  return (
    <Text color={color} bold={bold}>
      {prefix}
      {displayValue}
      {suffix}
    </Text>
  );
});

/** LoadingDots props */
export interface LoadingDotsProps {
  /** Number of dots (default: 3) */
  count?: number;
  /** Interval between dots in ms (default: 300) */
  interval?: number;
  /** Dot character (default: '.') */
  dot?: string;
  /** Color */
  color?: string;
}

/**
 * LoadingDots - Animated loading dots
 *
 * Shows progressive dots animation: . .. ... . ..
 * Respects reduced-motion - shows static dots when disabled.
 */
export const LoadingDots = memo(function LoadingDots({
  count = 3,
  interval = 300,
  dot = '.',
  color,
}: LoadingDotsProps) {
  const frames = useMemo(() => {
    const result: string[] = [];
    for (let i = 1; i <= count; i++) {
      result.push(dot.repeat(i));
    }
    return result;
  }, [count, dot]);

  const { frame } = useSpinner({ frames, interval });

  // Pad to maintain consistent width
  const paddedFrame = frame.padEnd(count, ' ');

  return <Text color={color}>{paddedFrame}</Text>;
});

/** WaveText props */
export interface WaveTextProps {
  /** Text to animate */
  children: string;
  /** Wave interval in ms (default: 150) */
  interval?: number;
  /** Color */
  color?: string;
}

/**
 * WaveText - Text with wave animation effect
 *
 * Each character cycles through dim/bright creating a wave effect.
 * Respects reduced-motion - shows static text when disabled.
 */
export const WaveText = memo(function WaveText({
  children,
  interval = 150,
  color,
}: WaveTextProps) {
  const { frameIndex } = useSpinner({
    frames: Array.from({ length: children.length }, (_, i) => String(i)),
    interval,
  });

  const chars = children.split('');

  return (
    <Text color={color}>
      {chars.map((char, i) => {
        // Create wave pattern - character is bright when wave passes through
        const distance = Math.abs(i - (frameIndex % children.length));
        const isDim = distance > 1;
        return (
          <Text key={i} dimColor={isDim}>
            {char}
          </Text>
        );
      })}
    </Text>
  );
});

export default FadeText;
