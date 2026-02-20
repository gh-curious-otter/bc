/**
 * useAnimation - Terminal-based animation hook
 * Issue #1024: Animations and visual effects
 * Issue #1198: TUI animations at 60fps
 *
 * Provides animation primitives for terminal UI:
 * - Fade (dim/bright transitions)
 * - Pulse (periodic brightness changes)
 * - Blink (on/off visibility)
 * - Typewriter (character-by-character reveal)
 * - Spring (physics-based smooth animations)
 * - Progress (smooth progress bar animations)
 *
 * Accessibility:
 * - Respects BC_NO_ANIMATIONS=1 environment variable
 * - Respects BC_ANIMATION_FPS for custom frame rate (default: 60)
 * - When animations disabled, shows final state immediately
 */

import { useState, useEffect, useCallback, useRef, useMemo } from 'react';

// ============================================================================
// Animation Configuration (Accessibility)
// ============================================================================

/**
 * Check if animations should be disabled.
 * Respects BC_NO_ANIMATIONS=1 environment variable.
 */
export function shouldDisableAnimations(): boolean {
  return process.env.BC_NO_ANIMATIONS === '1' || process.env.BC_NO_ANIMATIONS === 'true';
}

/**
 * Get configured animation FPS.
 * Respects BC_ANIMATION_FPS environment variable (default: 60).
 */
export function getAnimationFps(): number {
  const envFps = process.env.BC_ANIMATION_FPS;
  if (envFps) {
    const parsed = parseInt(envFps, 10);
    if (!isNaN(parsed) && parsed > 0 && parsed <= 120) {
      return parsed;
    }
  }
  return 60; // Default 60fps
}

/**
 * Hook to check if animations should be reduced/disabled.
 * Returns true if animations should be skipped.
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
  /** Frame rate in fps (default: 60 for smooth animations, #1198) */
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
 * Accessibility: When BC_NO_ANIMATIONS=1 is set, animations complete
 * immediately showing the final state.
 */
export function useAnimation(options: UseAnimationOptions = {}): UseAnimationResult {
  const {
    duration = 300,
    delay = 0,
    easing = 'easeOut',
    iterations = 1,
    autoStart = true,
    onComplete,
    fps = getAnimationFps(), // #1198: Configurable fps (default 60)
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

  // Auto-start (skip animation if reduced motion enabled)
  useEffect(() => {
    if (autoStart) {
      if (reducedMotion) {
        // Skip animation, go directly to complete state
        setState({
          progress: 1,
          isRunning: false,
          isComplete: true,
          iteration: iterations === Infinity ? 0 : iterations,
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
  }, [autoStart, start, reducedMotion, iterations, onComplete]);

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
 * Accessibility: When reduced motion is enabled, returns full opacity (no pulsing).
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

  // When reduced motion enabled, stay at full opacity
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
 * Accessibility: When reduced motion is enabled, stays visible (no blinking).
 */
export function useBlink(options: UseBlinkOptions = {}): UseBlinkResult {
  const { interval = 500, enabled = true } = options;
  const reducedMotion = useReducedMotion();
  const [isVisible, setIsVisible] = useState(true);

  useEffect(() => {
    // No blinking when disabled or reduced motion
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
 */
export function useTypewriter(options: UseTypewriterOptions): UseTypewriterResult {
  const { text, speed = 30, delay = 0, autoStart = true, onComplete } = options;

  const [charIndex, setCharIndex] = useState(0);
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
// Issue #1198: 60fps smooth animations
// ============================================================================

/** Spring animation options for physics-based motion */
export interface UseSpringOptions {
  /** Target value to animate towards */
  target: number;
  /** Tension/stiffness (default: 170) - higher = faster */
  tension?: number;
  /** Friction/damping (default: 26) - higher = more damping */
  friction?: number;
  /** Mass (default: 1) - higher = more inertia */
  mass?: number;
  /** Velocity threshold to stop (default: 0.01) */
  threshold?: number;
  /** Frame rate in fps (default: 60) */
  fps?: number;
}

export interface UseSpringResult {
  /** Current animated value */
  value: number;
  /** Current velocity */
  velocity: number;
  /** Whether animation is complete (settled) */
  isSettled: boolean;
}

/**
 * useSpring - Physics-based spring animation
 *
 * Creates smooth, natural-feeling animations that overshoot and settle.
 * Perfect for panel resizing, counters, and value transitions.
 *
 * Accessibility: When reduced motion is enabled, snaps to target immediately.
 */
export function useSpring(options: UseSpringOptions): UseSpringResult {
  const {
    target,
    tension = 170,
    friction = 26,
    mass = 1,
    threshold = 0.01,
    fps = getAnimationFps(),
  } = options;

  const reducedMotion = useReducedMotion();

  const [state, setState] = useState({
    value: target,
    velocity: 0,
    isSettled: true,
  });

  // When reduced motion enabled, snap to target immediately
  useEffect(() => {
    if (reducedMotion) {
      setState({ value: target, velocity: 0, isSettled: true });
    }
  }, [reducedMotion, target]);

  const frameInterval = useMemo(() => 1000 / fps, [fps]);
  const dt = frameInterval / 1000; // Convert to seconds

  const animationRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const targetRef = useRef(target);

  useEffect(() => {
    // If target changed, start animating (unless reduced motion)
    if (targetRef.current !== target) {
      targetRef.current = target;
      if (reducedMotion) {
        // Snap immediately when reduced motion enabled
        setState({ value: target, velocity: 0, isSettled: true });
      } else {
        setState((s) => ({ ...s, isSettled: false }));
      }
    }
  }, [target, reducedMotion]);

  useEffect(() => {
    // Skip animation loop when reduced motion enabled
    if (reducedMotion || state.isSettled) {
      if (animationRef.current) {
        clearInterval(animationRef.current);
        animationRef.current = null;
      }
      return;
    }

    animationRef.current = setInterval(() => {
      setState((s) => {
        // Spring physics: F = -kx - cv
        // where k = tension, c = friction, x = displacement
        const displacement = s.value - targetRef.current;
        const springForce = -tension * displacement;
        const dampingForce = -friction * s.velocity;
        const acceleration = (springForce + dampingForce) / mass;

        const newVelocity = s.velocity + acceleration * dt;
        const newValue = s.value + newVelocity * dt;

        // Check if settled (both position and velocity near target)
        const isSettled =
          Math.abs(newValue - targetRef.current) < threshold &&
          Math.abs(newVelocity) < threshold;

        if (isSettled) {
          return {
            value: targetRef.current,
            velocity: 0,
            isSettled: true,
          };
        }

        return {
          value: newValue,
          velocity: newVelocity,
          isSettled: false,
        };
      });
    }, frameInterval);

    return () => {
      if (animationRef.current) {
        clearInterval(animationRef.current);
      }
    };
  }, [state.isSettled, tension, friction, mass, dt, frameInterval, threshold, reducedMotion]);

  return state;
}

/** Progress animation options */
export interface UseProgressAnimationOptions {
  /** Current progress value 0-1 */
  progress: number;
  /** Animation duration in ms (default: 300) */
  duration?: number;
  /** Easing function (default: 'easeOut') */
  easing?: keyof typeof easings | EasingFunction;
  /** Frame rate in fps (default: 60) */
  fps?: number;
}

export interface UseProgressAnimationResult {
  /** Smoothly animated progress value 0-1 */
  animatedProgress: number;
  /** Whether animation is in progress */
  isAnimating: boolean;
}

/**
 * useProgressAnimation - Smooth progress bar animation
 *
 * Smoothly animates between progress values for loading bars.
 *
 * Accessibility: When reduced motion is enabled, snaps to progress immediately.
 */
export function useProgressAnimation(
  options: UseProgressAnimationOptions
): UseProgressAnimationResult {
  const { progress, duration = 300, easing = 'easeOut', fps = getAnimationFps() } = options;

  const reducedMotion = useReducedMotion();
  const [animatedProgress, setAnimatedProgress] = useState(progress);
  const [isAnimating, setIsAnimating] = useState(false);

  const animationRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const startValueRef = useRef(progress);
  const startTimeRef = useRef(0);
  const targetRef = useRef(progress);

  const easingFn = useMemo(
    () => (typeof easing === 'function' ? easing : easings[easing]),
    [easing]
  );

  const frameInterval = useMemo(() => Math.floor(1000 / fps), [fps]);

  useEffect(() => {
    if (targetRef.current === progress) {
      return;
    }

    // When reduced motion, snap immediately
    if (reducedMotion) {
      targetRef.current = progress;
      setAnimatedProgress(progress);
      setIsAnimating(false);
      return;
    }

    // Clear any existing animation
    if (animationRef.current) {
      clearInterval(animationRef.current);
    }

    // Start new animation
    startValueRef.current = animatedProgress;
    targetRef.current = progress;
    startTimeRef.current = Date.now();
    setIsAnimating(true);

    animationRef.current = setInterval(() => {
      const elapsed = Date.now() - startTimeRef.current;
      const rawProgress = Math.min(elapsed / duration, 1);
      const easedProgress = easingFn(rawProgress);

      const start = startValueRef.current;
      const end = targetRef.current;
      const current = start + (end - start) * easedProgress;

      setAnimatedProgress(current);

      if (rawProgress >= 1) {
        if (animationRef.current) {
          clearInterval(animationRef.current);
          animationRef.current = null;
        }
        setIsAnimating(false);
      }
    }, frameInterval);

    return () => {
      if (animationRef.current) {
        clearInterval(animationRef.current);
      }
    };
  }, [progress, duration, easingFn, frameInterval, animatedProgress, reducedMotion]);

  return { animatedProgress, isAnimating };
}

/** Spinner animation options */
export interface UseSpinnerOptions {
  /** Frames to cycle through */
  frames?: string[];
  /** Frame interval in ms (default: 80 for ~12fps spinner) */
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

/** Default spinner frames for smooth rotation */
export const SPINNER_FRAMES = {
  dots: ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'],
  line: ['|', '/', '-', '\\'],
  circle: ['◐', '◓', '◑', '◒'],
  arc: ['◜', '◠', '◝', '◞', '◡', '◟'],
  bouncing: ['⠁', '⠂', '⠄', '⠂'],
  growing: ['▁', '▃', '▄', '▅', '▆', '▇', '▆', '▅', '▄', '▃'],
  pulse: ['●', '●', '●', '○', '○', '○'],
};

/**
 * useSpinner - Smooth spinner animation
 *
 * Cycles through spinner frames for loading indicators.
 *
 * Accessibility: When reduced motion is enabled, shows static '...' indicator.
 */
export function useSpinner(options: UseSpinnerOptions = {}): UseSpinnerResult {
  const {
    frames = SPINNER_FRAMES.dots,
    interval = 80,
    enabled = true,
  } = options;

  const reducedMotion = useReducedMotion();
  const [frameIndex, setFrameIndex] = useState(0);

  useEffect(() => {
    // No spinning when disabled or reduced motion
    if (!enabled || reducedMotion) {
      return;
    }

    const timer = setInterval(() => {
      setFrameIndex((i) => (i + 1) % frames.length);
    }, interval);

    return () => {
      clearInterval(timer);
    };
  }, [frames.length, interval, enabled, reducedMotion]);

  // When reduced motion, show static indicator
  if (reducedMotion) {
    return {
      frame: '...',
      frameIndex: 0,
    };
  }

  return {
    frame: frames[frameIndex],
    frameIndex,
  };
}

/** Counter animation options */
export interface UseCounterOptions {
  /** Target value */
  value: number;
  /** Animation duration in ms (default: 500) */
  duration?: number;
  /** Number of decimal places (default: 0) */
  decimals?: number;
  /** Format function */
  format?: (value: number) => string;
}

export interface UseCounterResult {
  /** Formatted display value */
  displayValue: string;
  /** Raw animated value */
  rawValue: number;
  /** Whether animating */
  isAnimating: boolean;
}

/**
 * useCounter - Animated number counter
 *
 * Smoothly counts up/down to target value.
 */
export function useCounter(options: UseCounterOptions): UseCounterResult {
  const { value, duration = 500, decimals = 0, format } = options;

  const { animatedProgress, isAnimating } = useProgressAnimation({
    progress: value,
    duration,
  });

  const displayValue = format
    ? format(animatedProgress)
    : animatedProgress.toFixed(decimals);

  return {
    displayValue,
    rawValue: animatedProgress,
    isAnimating,
  };
}

export default useAnimation;
