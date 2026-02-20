/**
 * useAnimation - Terminal-based animation hook
 * Issue #1024: Animations and visual effects
 * Issue #1210: Reduced-motion and animation accessibility support
 *
 * Provides animation primitives for terminal UI:
 * - Fade (dim/bright transitions)
 * - Pulse (periodic brightness changes)
 * - Blink (on/off visibility)
 * - Typewriter (character-by-character reveal)
 * - Spring (physics-based animations)
 * - Progress (smooth progress bar animations)
 * - Spinner (frame-based loading spinners)
 * - Counter (animated number display)
 *
 * Accessibility:
 * - Respects BC_NO_ANIMATIONS=1 environment variable
 * - Configurable frame rate via BC_ANIMATION_FPS
 * - useReducedMotion() hook for components to check preference
 */

import { useState, useEffect, useCallback, useRef, useMemo } from 'react';

// ============================================================================
// Accessibility Support (Issue #1210)
// ============================================================================

/**
 * Check if animations should be disabled globally.
 * Respects BC_NO_ANIMATIONS environment variable.
 */
export function shouldDisableAnimations(): boolean {
  return process.env.BC_NO_ANIMATIONS === '1' || process.env.BC_NO_ANIMATIONS === 'true';
}

/**
 * Get the configured animation frame rate.
 * Respects BC_ANIMATION_FPS environment variable.
 * Returns 60 by default, capped at 120fps max.
 */
export function getAnimationFps(): number {
  const envFps = process.env.BC_ANIMATION_FPS;
  if (envFps) {
    const parsed = parseInt(envFps, 10);
    if (!isNaN(parsed) && parsed > 0 && parsed <= 120) {
      return parsed;
    }
  }
  return 60;
}

/**
 * Hook to check if reduced motion is preferred.
 * Returns true if animations should be disabled or reduced.
 */
export function useReducedMotion(): boolean {
  return useMemo(() => shouldDisableAnimations(), []);
}

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
  /** Frame rate in fps (default: 60) */
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
 *
 * Respects reduced motion preference - skips to completion when animations are disabled.
 */
export function useAnimation(options: UseAnimationOptions = {}): UseAnimationResult {
  const {
    duration = 300,
    delay = 0,
    easing = 'easeOut',
    iterations = 1,
    autoStart = true,
    onComplete,
    fps = getAnimationFps(),
  } = options;

  const reducedMotion = useReducedMotion();

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

  // Auto-start (respects reduced motion)
  useEffect(() => {
    if (autoStart) {
      if (reducedMotion) {
        // Skip to completion for reduced motion
        setState({
          progress: 1,
          isRunning: false,
          isComplete: true,
          iteration: 1,
        });
        onComplete?.();
      } else {
        start();
      }
    }
    return () => {
      if (animationRef.current) {
        clearInterval(animationRef.current);
      }
    };
  }, [autoStart, start, reducedMotion, onComplete]);

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
 *
 * Respects reduced motion - stays at full opacity when disabled.
 */
export function usePulse(options: UsePulseOptions = {}): UsePulseResult {
  const { interval = 1000, minOpacity = 0.3, maxOpacity = 1, enabled = true } = options;

  const reducedMotion = useReducedMotion();

  const { state } = useAnimation({
    duration: interval,
    iterations: Infinity,
    autoStart: enabled && !reducedMotion,
    easing: 'easeInOut',
  });

  // For reduced motion, stay at full opacity
  if (reducedMotion) {
    return { isDim: false, opacity: maxOpacity, progress: 0 };
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
 *
 * Respects reduced motion - stays visible when disabled.
 */
export function useBlink(options: UseBlinkOptions = {}): UseBlinkResult {
  const { interval = 500, enabled = true } = options;
  const [isVisible, setIsVisible] = useState(true);
  const reducedMotion = useReducedMotion();

  useEffect(() => {
    // Stay visible when animations are disabled
    if (!enabled || reducedMotion) {
      setIsVisible(true);
      return;
    }

    const timer = setInterval(() => {
      setIsVisible((v) => !v);
    }, interval);

    return () => {
      clearInterval(timer);
    };
  }, [interval, enabled, reducedMotion]);

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
 *
 * Respects reduced motion - shows full text immediately when disabled.
 */
export function useTypewriter(options: UseTypewriterOptions): UseTypewriterResult {
  const { text, speed = 30, delay = 0, autoStart = true, onComplete } = options;

  const reducedMotion = useReducedMotion();
  const [charIndex, setCharIndex] = useState(reducedMotion ? text.length : 0);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const charInterval = useMemo(() => Math.floor(1000 / speed), [speed]);

  const displayText = text.slice(0, charIndex);
  const isComplete = charIndex >= text.length;

  const start = useCallback(() => {
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
  }, [text.length, charInterval, delay, onComplete]);

  const reset = useCallback(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    setCharIndex(0);
  }, []);

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

  // Reset when text changes (respects reduced motion)
  useEffect(() => {
    if (reducedMotion) {
      // Show full text immediately for reduced motion
      setCharIndex(text.length);
      onComplete?.();
    } else {
      reset();
      if (autoStart) {
        start();
      }
    }
  }, [text, autoStart, reset, start, reducedMotion, onComplete]);

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
 *
 * Respects reduced motion via useAnimation.
 */
export function useFade(options: UseFadeOptions = {}): UseFadeResult {
  const { direction = 'in', duration = 200, autoStart = true, onComplete } = options;

  const { state, start } = useAnimation({
    duration,
    autoStart,
    onComplete,
    easing: 'easeOut',
  });

  const opacity = direction === 'in' ? state.progress : 1 - state.progress;
  const isDim = opacity < 0.5;

  return { isDim, opacity, start, isComplete: state.isComplete };
}

// ============================================================================
// Extended Animation Hooks (Issue #1210)
// ============================================================================

/** Spinner frame presets */
export const SPINNER_FRAMES = {
  dots: ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'],
  line: ['-', '\\', '|', '/'],
  circle: ['◐', '◓', '◑', '◒'],
  arrow: ['←', '↖', '↑', '↗', '→', '↘', '↓', '↙'],
  bounce: ['⠁', '⠂', '⠄', '⠂'],
  pulse: ['█', '▓', '▒', '░', '▒', '▓'],
};

/** Spinner animation options */
export interface UseSpinnerOptions {
  /** Spinner frames (default: dots) */
  frames?: string[];
  /** Frame interval in ms (default: 80) */
  interval?: number;
  /** Enable spinner (default: true) */
  enabled?: boolean;
}

export interface UseSpinnerResult {
  /** Current frame character */
  frame: string;
  /** Current frame index */
  frameIndex: number;
}

/**
 * Spinner animation hook - cycles through frames
 *
 * Respects reduced motion - shows static indicator when disabled.
 */
export function useSpinner(options: UseSpinnerOptions = {}): UseSpinnerResult {
  const { frames = SPINNER_FRAMES.dots, interval = 80, enabled = true } = options;

  const reducedMotion = useReducedMotion();
  const [frameIndex, setFrameIndex] = useState(0);

  useEffect(() => {
    // Show static indicator for reduced motion
    if (!enabled || reducedMotion) {
      setFrameIndex(0);
      return;
    }

    const timer = setInterval(() => {
      setFrameIndex((i) => (i + 1) % frames.length);
    }, interval);

    return () => {
      clearInterval(timer);
    };
  }, [frames.length, interval, enabled, reducedMotion]);

  // For reduced motion, show static indicator
  if (reducedMotion) {
    return { frame: '...', frameIndex: 0 };
  }

  return { frame: frames[frameIndex], frameIndex };
}

/** Progress animation options */
export interface UseProgressAnimationOptions {
  /** Target progress value 0-1 */
  progress: number;
  /** Animation duration in ms (default: 300) */
  duration?: number;
}

export interface UseProgressAnimationResult {
  /** Current animated progress value 0-1 */
  animatedProgress: number;
}

/**
 * Progress animation hook - smoothly animates between progress values
 *
 * Respects reduced motion - snaps to target value when disabled.
 */
export function useProgressAnimation(
  options: UseProgressAnimationOptions
): UseProgressAnimationResult {
  const { progress, duration = 300 } = options;

  const reducedMotion = useReducedMotion();
  const [animatedProgress, setAnimatedProgress] = useState(progress);
  const startValueRef = useRef(progress);
  const animationRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    // Snap to target for reduced motion
    if (reducedMotion) {
      setAnimatedProgress(progress);
      return;
    }

    const startValue = animatedProgress;
    const targetValue = progress;
    const startTime = Date.now();

    startValueRef.current = startValue;

    if (animationRef.current) {
      clearInterval(animationRef.current);
    }

    const fps = getAnimationFps();
    const frameInterval = Math.floor(1000 / fps);

    animationRef.current = setInterval(() => {
      const elapsed = Date.now() - startTime;
      const rawProgress = Math.min(elapsed / duration, 1);
      const easedProgress = easings.easeOut(rawProgress);

      const current = startValue + (targetValue - startValue) * easedProgress;
      setAnimatedProgress(current);

      if (rawProgress >= 1) {
        if (animationRef.current) {
          clearInterval(animationRef.current);
          animationRef.current = null;
        }
      }
    }, frameInterval);

    return () => {
      if (animationRef.current) {
        clearInterval(animationRef.current);
        animationRef.current = null;
      }
    };
  }, [progress, duration, reducedMotion]); // eslint-disable-line react-hooks/exhaustive-deps

  return { animatedProgress };
}

/** Counter animation options */
export interface UseCounterOptions {
  /** Target value */
  value: number;
  /** Animation duration in ms (default: 500) */
  duration?: number;
  /** Number of decimal places (default: 0) */
  decimals?: number;
}

export interface UseCounterResult {
  /** Current display value (formatted string) */
  displayValue: string;
  /** Current numeric value */
  numericValue: number;
}

/**
 * Counter animation hook - smoothly counts up/down to target value
 *
 * Respects reduced motion - shows target value immediately when disabled.
 */
export function useCounter(options: UseCounterOptions): UseCounterResult {
  const { value, duration = 500, decimals = 0 } = options;

  const reducedMotion = useReducedMotion();
  const [numericValue, setNumericValue] = useState(value);
  const animationRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    // Show target immediately for reduced motion
    if (reducedMotion) {
      setNumericValue(value);
      return;
    }

    const startValue = numericValue;
    const targetValue = value;
    const startTime = Date.now();

    if (animationRef.current) {
      clearInterval(animationRef.current);
    }

    const fps = getAnimationFps();
    const frameInterval = Math.floor(1000 / fps);

    animationRef.current = setInterval(() => {
      const elapsed = Date.now() - startTime;
      const rawProgress = Math.min(elapsed / duration, 1);
      const easedProgress = easings.easeOut(rawProgress);

      const current = startValue + (targetValue - startValue) * easedProgress;
      setNumericValue(current);

      if (rawProgress >= 1) {
        if (animationRef.current) {
          clearInterval(animationRef.current);
          animationRef.current = null;
        }
      }
    }, frameInterval);

    return () => {
      if (animationRef.current) {
        clearInterval(animationRef.current);
        animationRef.current = null;
      }
    };
  }, [value, duration, reducedMotion]); // eslint-disable-line react-hooks/exhaustive-deps

  const displayValue = numericValue.toFixed(decimals);

  return { displayValue, numericValue };
}

/** Spring animation options */
export interface UseSpringOptions {
  /** Target value */
  to: number;
  /** Starting value (default: 0) */
  from?: number;
  /** Spring tension (default: 170) */
  tension?: number;
  /** Spring friction (default: 26) */
  friction?: number;
  /** Mass (default: 1) */
  mass?: number;
  /** Precision threshold (default: 0.01) */
  precision?: number;
}

export interface UseSpringResult {
  /** Current animated value */
  value: number;
  /** Current velocity */
  velocity: number;
  /** Whether animation is complete */
  isComplete: boolean;
}

/**
 * Calculate spring force for physics-based animation
 */
export function calculateSpringForce(
  displacement: number,
  velocity: number,
  tension = 170,
  friction = 26
): number {
  return -tension * displacement - friction * velocity;
}

/**
 * Spring animation hook - physics-based spring animation
 *
 * Respects reduced motion - snaps to target value when disabled.
 */
export function useSpring(options: UseSpringOptions): UseSpringResult {
  const {
    to,
    from = 0,
    tension = 170,
    friction = 26,
    mass = 1,
    precision = 0.01,
  } = options;

  const reducedMotion = useReducedMotion();
  const [state, setState] = useState({
    value: reducedMotion ? to : from,
    velocity: 0,
    isComplete: reducedMotion,
  });
  const animationRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    // Snap to target for reduced motion
    if (reducedMotion) {
      setState({ value: to, velocity: 0, isComplete: true });
      return;
    }

    // Start spring animation
    let currentValue = state.value;
    let currentVelocity = state.velocity;
    let lastTime = Date.now();

    if (animationRef.current) {
      clearInterval(animationRef.current);
    }

    const fps = getAnimationFps();
    const frameInterval = Math.floor(1000 / fps);

    animationRef.current = setInterval(() => {
      const now = Date.now();
      const dt = Math.min((now - lastTime) / 1000, 0.064); // Cap delta time
      lastTime = now;

      const displacement = currentValue - to;
      const force = calculateSpringForce(displacement, currentVelocity, tension, friction);
      const acceleration = force / mass;

      currentVelocity += acceleration * dt;
      currentValue += currentVelocity * dt;

      // Check if spring has settled
      const isSettled =
        Math.abs(displacement) < precision && Math.abs(currentVelocity) < precision;

      if (isSettled) {
        if (animationRef.current) {
          clearInterval(animationRef.current);
          animationRef.current = null;
        }
        setState({ value: to, velocity: 0, isComplete: true });
      } else {
        setState({ value: currentValue, velocity: currentVelocity, isComplete: false });
      }
    }, frameInterval);

    return () => {
      if (animationRef.current) {
        clearInterval(animationRef.current);
        animationRef.current = null;
      }
    };
  }, [to, tension, friction, mass, precision, reducedMotion]); // eslint-disable-line react-hooks/exhaustive-deps

  return state;
}

export default useAnimation;
