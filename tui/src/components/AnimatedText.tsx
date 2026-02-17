/**
 * AnimatedText - Terminal-based text animation components
 * Issue #1024: Animations and visual effects
 *
 * Provides animated text effects for terminal UI:
 * - FadeText: Fade in/out effect
 * - PulseText: Pulsing brightness
 * - TypewriterText: Character-by-character reveal
 * - BlinkText: Simple blink effect
 */

import { memo } from 'react';
import { Text } from 'ink';
import type { TextProps } from 'ink';
import { useFade, usePulse, useTypewriter, useBlink } from '../hooks/useAnimation';
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

export default FadeText;
