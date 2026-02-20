/**
 * useAnimation - Terminal-based animation hook
 * Issue #1024: Animations and visual effects
 * Issue #1210: Reduced motion accessibility support
 *
 * Provides animation primitives for terminal UI:
 * - Fade (dim/bright transitions)
 * - Pulse (periodic brightness changes)
 * - Blink (on/off visibility)
 * - Typewriter (character-by-character reveal)
 *
 * All animations respect reduced motion preferences via:
 * - BC_NO_ANIMATIONS=1 (disables all animations)
 * - BC_REDUCED_MOTION=1 (reduces animation intensity)
 * - BC_ANIMATION_FPS (custom frame rate limit)
 */

import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { useReducedMotion, getAccessibleAnimationOptions } from './useReducedMotion';

/** Animation easing functions */
export type EasingFunction = (t: number) => number;

export const easings: Record<string, EasingFunction> = {
  linear: (t) => t,
  easeIn: (t) => t * t,
  easeOut: (t) => t * (2 - t),
  easeInOut: (t) => (t < 0.5 ? 2 * t * t : -1 + (4 - 2 * t) * t),
  bounce: (t) => {
    const n1 = 7.5625;
    const d1 = 2.75;
    if (t < 1 / d1) return n1 * t * t;
    if (t < 2 / d1) return n1 * (t -= 1.5 / d1) * t + 0.75;
    if (t < 2.5 / d1) return n1 * (t -= 2.25 / d1) * t + 0.9375;
    return n1 * (t -= 2.625 / d1) * t + 0.984375;
  },
};

/** Animation state */
export interface AnimationState {
  /** Current progress 0-1 */
  progress: number;
  /** Whether animation is running */
  isRunning: boolean;
  /** Whether animation is complete */
  isComplete: boolean;
  /** Current iteration (for loops) */
  iteration: number;
}

/** Animation options */
export interface UseAnimationOptions {
  /** Duration in ms (default: 300) */
  duration?: number;
  /** Delay before start in ms (default: 0) */
  delay?: number;
  /** Easing function (default: 'easeOut') */
  easing?: keyof typeof easings | EasingFunction;
  /** Number of iterations, Infinity for endless (default: 1) */
  iterations?: number;
  /** Auto-start animation (default: true) */
  autoStart?: boolean;
  /** Callback on completion */
  onComplete?: () => void;
  /** Frame rate in fps (default: 30) */
  fps?: number;
}

export interface UseAnimationResult {
  /** Current animation state */
  state: AnimationState;
  /** Start animation */
  start: () => void;
  /** Stop animation */
  stop: () => void;
  /** Reset to initial state */
  reset: () => void;
  /** Pause animation */
  pause: () => void;
  /** Resume paused animation */
  resume: () => void;
}

/**
 * Core animation hook
 * Respects reduced motion preferences (#1210)
 */
export function useAnimation(options: UseAnimationOptions = {}): UseAnimationResult {
  const reducedMotion = useReducedMotion();
  const accessibleOptions = getAccessibleAnimationOptions(reducedMotion, {
    duration: options.duration ?? 300,
    fps: options.fps ?? 30,
  });

  const {
    duration = accessibleOptions.duration,
    delay = 0,
    easing = 'easeOut',
    iterations = 1,
    autoStart = true,
    onComplete,
    fps = accessibleOptions.fps,
  } = options;

  const [state, setState] = useState<AnimationState>({
    progress: 0,
    isRunning: false,
    isComplete: false,
    iteration: 0,
  });

  const startTimeRef = useRef<number>(0);
  const pausedAtRef = useRef<number>(0);
  const animationRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const easingFn = useMemo(
    () => (typeof easing === 'function' ? easing : easings[easing]),
    [easing]
  );

  const frameInterval = useMemo(() => Math.floor(1000 / fps), [fps]);

  const stop = useCallback(() => {
    if (animationRef.current) {
      clearInterval(animationRef.current);
      animationRef.current = null;
    }
    setState((s) => ({ ...s, isRunning: false }));
  }, []);

  const reset = useCallback(() => {
    stop();
    setState({
      progress: 0,
      isRunning: false,
      isComplete: false,
      iteration: 0,
    });
  }, [stop]);

  const start = useCallback(() => {
    stop();

    const startAfterDelay = () => {
      startTimeRef.current = Date.now();
      setState((s) => ({ ...s, isRunning: true, isComplete: false }));

      animationRef.current = setInterval(() => {
        const elapsed = Date.now() - startTimeRef.current;
        const rawProgress = Math.min(elapsed / duration, 1);
        const progress = easingFn(rawProgress);

        setState((s) => {
          if (rawProgress >= 1) {
            const nextIteration = s.iteration + 1;
            if (iterations !== Infinity && nextIteration >= iterations) {
              // Animation complete
              if (animationRef.current) {
                clearInterval(animationRef.current);
                animationRef.current = null;
              }
              onComplete?.();
              return {
                progress: 1,
                isRunning: false,
                isComplete: true,
                iteration: nextIteration,
              };
            }
            // Start next iteration
            startTimeRef.current = Date.now();
            return { ...s, progress: 0, iteration: nextIteration };
          }
          return { ...s, progress };
        });
      }, frameInterval);
    };

    if (delay > 0) {
      setTimeout(startAfterDelay, delay);
    } else {
      startAfterDelay();
    }
  }, [stop, duration, delay, iterations, easingFn, frameInterval, onComplete]);

  const pause = useCallback(() => {
    if (state.isRunning && animationRef.current) {
      pausedAtRef.current = Date.now() - startTimeRef.current;
      clearInterval(animationRef.current);
      animationRef.current = null;
      setState((s) => ({ ...s, isRunning: false }));
    }
  }, [state.isRunning]);

  const resume = useCallback(() => {
    if (!state.isRunning && !state.isComplete && pausedAtRef.current > 0) {
      startTimeRef.current = Date.now() - pausedAtRef.current;
      setState((s) => ({ ...s, isRunning: true }));

      animationRef.current = setInterval(() => {
        const elapsed = Date.now() - startTimeRef.current;
        const rawProgress = Math.min(elapsed / duration, 1);
        const progress = easingFn(rawProgress);

        setState((s) => {
          if (rawProgress >= 1) {
            if (animationRef.current) {
              clearInterval(animationRef.current);
              animationRef.current = null;
            }
            onComplete?.();
            return { ...s, progress: 1, isRunning: false, isComplete: true };
          }
          return { ...s, progress };
        });
      }, frameInterval);
    }
  }, [state.isRunning, state.isComplete, duration, easingFn, frameInterval, onComplete]);

  // Auto-start (skip if animations disabled)
  useEffect(() => {
    // If animations disabled, immediately complete
    if (!accessibleOptions.enabled) {
      setState({
        progress: 1,
        isRunning: false,
        isComplete: true,
        iteration: 1,
      });
      onComplete?.();
      return;
    }

    if (autoStart) {
      start();
    }
    return () => {
      if (animationRef.current) {
        clearInterval(animationRef.current);
      }
    };
  }, [autoStart, start, accessibleOptions.enabled, onComplete]);

  return { state, start, stop, reset, pause, resume };
}

/** Pulse animation options */
export interface UsePulseOptions {
  /** Pulse interval in ms (default: 1000) */
  interval?: number;
  /** Minimum opacity 0-1 (default: 0.3) */
  minOpacity?: number;
  /** Maximum opacity 0-1 (default: 1) */
  maxOpacity?: number;
  /** Enable pulse (default: true) */
  enabled?: boolean;
}

export interface UsePulseResult {
  /** Whether currently dim (low opacity phase) */
  isDim: boolean;
  /** Current opacity value 0-1 */
  opacity: number;
  /** Current animation progress 0-1 */
  progress: number;
}

/**
 * Pulse animation hook - oscillates between dim and bright
 * Respects reduced motion preferences (#1210)
 */
export function usePulse(options: UsePulseOptions = {}): UsePulseResult {
  const { interval = 1000, minOpacity = 0.3, maxOpacity = 1, enabled = true } = options;

  const reducedMotion = useReducedMotion();
  const animationsDisabled =
    reducedMotion.prefersReducedMotion && reducedMotion.recommendedFps === 0;

  // Adjust interval for reduced motion (slower pulse, or disabled)
  const adjustedInterval = animationsDisabled
    ? interval // Won't animate anyway
    : reducedMotion.prefersReducedMotion
      ? Math.max(interval, 2000) // Slower pulse for reduced motion
      : interval;

  const { state } = useAnimation({
    duration: adjustedInterval,
    iterations: Infinity,
    autoStart: enabled && !animationsDisabled,
    easing: 'easeInOut',
  });

  // If animations disabled, return static max opacity
  if (animationsDisabled) {
    return { isDim: false, opacity: maxOpacity, progress: 1 };
  }

  // Oscillate between min and max using sine wave
  const sineProgress = Math.sin(state.progress * Math.PI);
  const opacity = minOpacity + sineProgress * (maxOpacity - minOpacity);
  const isDim = state.progress > 0.5;

  return { isDim, opacity, progress: state.progress };
}

/** Blink animation options */
export interface UseBlinkOptions {
  /** Blink interval in ms (default: 500) */
  interval?: number;
  /** Enable blink (default: true) */
  enabled?: boolean;
}

export interface UseBlinkResult {
  /** Whether currently visible */
  isVisible: boolean;
}

/**
 * Blink animation hook - simple on/off toggle
 * Respects reduced motion preferences (#1210)
 */
export function useBlink(options: UseBlinkOptions = {}): UseBlinkResult {
  const { interval = 500, enabled = true } = options;
  const [isVisible, setIsVisible] = useState(true);
  const reducedMotion = useReducedMotion();

  // If animations disabled, always visible (no blinking)
  const shouldBlink =
    enabled && !(reducedMotion.prefersReducedMotion && reducedMotion.recommendedFps === 0);

  // Slower blink for reduced motion
  const adjustedInterval = reducedMotion.prefersReducedMotion
    ? Math.max(interval, 1500)
    : interval;

  useEffect(() => {
    if (!shouldBlink) {
      setIsVisible(true);
      return;
    }

    const timer = setInterval(() => {
      setIsVisible((v) => !v);
    }, adjustedInterval);

    return () => {
      clearInterval(timer);
    };
  }, [adjustedInterval, shouldBlink]);

  return { isVisible };
}

/** Typewriter animation options */
export interface UseTypewriterOptions {
  /** Text to reveal */
  text: string;
  /** Characters per second (default: 30) */
  speed?: number;
  /** Delay before start in ms (default: 0) */
  delay?: number;
  /** Auto-start (default: true) */
  autoStart?: boolean;
  /** Callback when complete */
  onComplete?: () => void;
}

export interface UseTypewriterResult {
  /** Currently visible text */
  displayText: string;
  /** Whether animation is complete */
  isComplete: boolean;
  /** Start animation */
  start: () => void;
  /** Reset to beginning */
  reset: () => void;
}

/**
 * Typewriter animation hook - reveals text character by character
 * Respects reduced motion preferences (#1210)
 */
export function useTypewriter(options: UseTypewriterOptions): UseTypewriterResult {
  const { text, speed = 30, delay = 0, autoStart = true, onComplete } = options;

  const reducedMotion = useReducedMotion();

  // If animations disabled, show full text immediately
  const animationsDisabled =
    reducedMotion.prefersReducedMotion && reducedMotion.recommendedFps === 0;

  const [charIndex, setCharIndex] = useState(animationsDisabled ? text.length : 0);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Slower typing for reduced motion (capped at 15 cps)
  const adjustedSpeed = reducedMotion.prefersReducedMotion ? Math.min(speed, 15) : speed;
  const charInterval = useMemo(() => Math.floor(1000 / adjustedSpeed), [adjustedSpeed]);

  const displayText = text.slice(0, charIndex);
  const isComplete = charIndex >= text.length;

  const start = useCallback(() => {
    // If animations disabled, complete immediately
    if (animationsDisabled) {
      setCharIndex(text.length);
      onComplete?.();
      return;
    }

    const startTyping = () => {
      timerRef.current = setInterval(() => {
        setCharIndex((i) => {
          const next = i + 1;
          if (next >= text.length) {
            if (timerRef.current) {
              clearInterval(timerRef.current);
              timerRef.current = null;
            }
            onComplete?.();
            return text.length;
          }
          return next;
        });
      }, charInterval);
    };

    if (delay > 0) {
      setTimeout(startTyping, delay);
    } else {
      startTyping();
    }
  }, [text.length, charInterval, delay, onComplete, animationsDisabled]);

  const reset = useCallback(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    setCharIndex(animationsDisabled ? text.length : 0);
  }, [animationsDisabled, text.length]);

  useEffect(() => {
    if (autoStart) {
      start();
    }
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, [autoStart, start]);

  // Reset when text changes
  useEffect(() => {
    reset();
    if (autoStart) {
      start();
    }
  }, [text, autoStart, reset, start]);

  return { displayText, isComplete, start, reset };
}

/** Fade direction */
export type FadeDirection = 'in' | 'out';

/** Fade animation options */
export interface UseFadeOptions {
  /** Fade direction (default: 'in') */
  direction?: FadeDirection;
  /** Duration in ms (default: 200) */
  duration?: number;
  /** Auto-start (default: true) */
  autoStart?: boolean;
  /** Callback when complete */
  onComplete?: () => void;
}

export interface UseFadeResult {
  /** Whether element should be dimmed */
  isDim: boolean;
  /** Current opacity 0-1 */
  opacity: number;
  /** Start fade animation */
  start: () => void;
  /** Whether animation is complete */
  isComplete: boolean;
}

/**
 * Fade animation hook - fade in or out
 * Respects reduced motion preferences (#1210)
 */
export function useFade(options: UseFadeOptions = {}): UseFadeResult {
  const { direction = 'in', duration = 200, autoStart = true, onComplete } = options;

  const reducedMotion = useReducedMotion();
  const animationsDisabled =
    reducedMotion.prefersReducedMotion && reducedMotion.recommendedFps === 0;

  // useAnimation will handle instant completion when animations are disabled
  const { state, start } = useAnimation({
    duration: animationsDisabled ? 0 : duration,
    autoStart,
    onComplete,
    easing: 'easeOut',
  });

  // For disabled animations, return final state; otherwise compute from progress
  const opacity = animationsDisabled
    ? direction === 'in'
      ? 1
      : 0
    : direction === 'in'
      ? state.progress
      : 1 - state.progress;
  const isDim = opacity < 0.5;

  return { isDim, opacity, start, isComplete: animationsDisabled || state.isComplete };
}

export default useAnimation;
