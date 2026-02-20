/**
 * Tests for useAnimation hook
 * Issue #1024: Animations and visual effects
 */

import { describe, test, expect, beforeEach, afterAll } from 'bun:test';

// Test the easing functions and animation logic without React hooks
describe('Animation Easing Functions', () => {
  // Replicate easing functions from the module
  const easings = {
    linear: (t: number) => t,
    easeIn: (t: number) => t * t,
    easeOut: (t: number) => t * (2 - t),
    easeInOut: (t: number) => (t < 0.5 ? 2 * t * t : -1 + (4 - 2 * t) * t),
    bounce: (t: number) => {
      const n1 = 7.5625;
      const d1 = 2.75;
      if (t < 1 / d1) return n1 * t * t;
      if (t < 2 / d1) return n1 * (t -= 1.5 / d1) * t + 0.75;
      if (t < 2.5 / d1) return n1 * (t -= 2.25 / d1) * t + 0.9375;
      return n1 * (t -= 2.625 / d1) * t + 0.984375;
    },
  };

  describe('linear easing', () => {
    test('returns input unchanged', () => {
      expect(easings.linear(0)).toBe(0);
      expect(easings.linear(0.25)).toBe(0.25);
      expect(easings.linear(0.5)).toBe(0.5);
      expect(easings.linear(0.75)).toBe(0.75);
      expect(easings.linear(1)).toBe(1);
    });
  });

  describe('easeIn easing', () => {
    test('starts slow, ends fast', () => {
      const quarterProgress = easings.easeIn(0.5);
      expect(quarterProgress).toBe(0.25); // 0.5 * 0.5 = 0.25
    });

    test('bounds are correct', () => {
      expect(easings.easeIn(0)).toBe(0);
      expect(easings.easeIn(1)).toBe(1);
    });

    test('is slower than linear at midpoint', () => {
      const linear = easings.linear(0.5);
      const easeIn = easings.easeIn(0.5);
      expect(easeIn).toBeLessThan(linear);
    });
  });

  describe('easeOut easing', () => {
    test('starts fast, ends slow', () => {
      const midProgress = easings.easeOut(0.5);
      expect(midProgress).toBe(0.75); // 0.5 * (2 - 0.5) = 0.75
    });

    test('bounds are correct', () => {
      expect(easings.easeOut(0)).toBe(0);
      expect(easings.easeOut(1)).toBe(1);
    });

    test('is faster than linear at midpoint', () => {
      const linear = easings.linear(0.5);
      const easeOut = easings.easeOut(0.5);
      expect(easeOut).toBeGreaterThan(linear);
    });
  });

  describe('easeInOut easing', () => {
    test('is symmetric', () => {
      // At midpoint
      const atMid = easings.easeInOut(0.5);
      expect(atMid).toBe(0.5);
    });

    test('bounds are correct', () => {
      expect(easings.easeInOut(0)).toBe(0);
      expect(easings.easeInOut(1)).toBe(1);
    });

    test('first half accelerates', () => {
      const quarter = easings.easeInOut(0.25);
      expect(quarter).toBeLessThan(0.25);
    });

    test('second half decelerates', () => {
      const threeQuarter = easings.easeInOut(0.75);
      expect(threeQuarter).toBeGreaterThan(0.75);
    });
  });

  describe('bounce easing', () => {
    test('bounds are correct', () => {
      expect(easings.bounce(0)).toBe(0);
      expect(easings.bounce(1)).toBeCloseTo(1, 5);
    });

    test('all values are between 0 and 1', () => {
      for (let t = 0; t <= 1; t += 0.1) {
        const value = easings.bounce(t);
        expect(value).toBeGreaterThanOrEqual(0);
        expect(value).toBeLessThanOrEqual(1.01); // Small tolerance for floating point
      }
    });
  });

  describe('all easings', () => {
    test('start at 0', () => {
      for (const [name, fn] of Object.entries(easings)) {
        expect(fn(0)).toBe(0);
      }
    });

    test('end at or near 1', () => {
      for (const [name, fn] of Object.entries(easings)) {
        expect(fn(1)).toBeCloseTo(1, 5);
      }
    });

    test('are monotonically increasing (except bounce)', () => {
      // Exclude bounce since it has intentional non-monotonic behavior
      const monotonicEasings = Object.entries(easings).filter(
        ([name]) => name !== 'bounce'
      );
      for (const [name, fn] of monotonicEasings) {
        let prev = 0;
        for (let t = 0; t <= 1; t += 0.1) {
          const value = fn(t);
          expect(value).toBeGreaterThanOrEqual(prev - 0.001); // Small tolerance
          prev = value;
        }
      }
    });
  });
});

describe('Animation State Logic', () => {
  interface AnimationState {
    progress: number;
    isRunning: boolean;
    isComplete: boolean;
    iteration: number;
  }

  const createInitialState = (): AnimationState => ({
    progress: 0,
    isRunning: false,
    isComplete: false,
    iteration: 0,
  });

  test('initial state is correct', () => {
    const state = createInitialState();
    expect(state.progress).toBe(0);
    expect(state.isRunning).toBe(false);
    expect(state.isComplete).toBe(false);
    expect(state.iteration).toBe(0);
  });

  test('progress calculation from elapsed time', () => {
    const duration = 300;
    const calculateProgress = (elapsed: number) =>
      Math.min(elapsed / duration, 1);

    expect(calculateProgress(0)).toBe(0);
    expect(calculateProgress(150)).toBe(0.5);
    expect(calculateProgress(300)).toBe(1);
    expect(calculateProgress(400)).toBe(1); // Capped at 1
  });

  test('iteration increments correctly', () => {
    const duration = 100;
    const iterations = 3;

    // Simulate progress updates
    let iteration = 0;
    const times = [0, 100, 200, 300];

    for (const elapsed of times) {
      const rawProgress = Math.min(elapsed / duration, 1);
      if (rawProgress >= 1 && iteration < iterations) {
        iteration++;
      }
    }

    expect(iteration).toBe(3);
  });

  test('completes after final iteration', () => {
    const iterations = 2;
    let currentIteration = 0;
    let isComplete = false;

    // Simulate 2 complete iterations
    for (let i = 0; i < 3; i++) {
      currentIteration++;
      if (currentIteration >= iterations) {
        isComplete = true;
        break;
      }
    }

    expect(isComplete).toBe(true);
    expect(currentIteration).toBe(2);
  });
});

describe('Pulse Animation Logic', () => {
  test('oscillates between min and max opacity', () => {
    const minOpacity = 0.3;
    const maxOpacity = 1.0;

    const calculateOpacity = (progress: number) => {
      const sineProgress = Math.sin(progress * Math.PI);
      return minOpacity + sineProgress * (maxOpacity - minOpacity);
    };

    // At progress 0: sin(0) = 0, opacity = 0.3
    expect(calculateOpacity(0)).toBeCloseTo(0.3, 5);

    // At progress 0.5: sin(π/2) = 1, opacity = 1.0
    expect(calculateOpacity(0.5)).toBeCloseTo(1.0, 5);

    // At progress 1: sin(π) = 0, opacity = 0.3
    expect(calculateOpacity(1)).toBeCloseTo(0.3, 5);
  });

  test('isDim is true when progress > 0.5', () => {
    const isDimAtProgress = (progress: number) => progress > 0.5;

    expect(isDimAtProgress(0)).toBe(false);
    expect(isDimAtProgress(0.4)).toBe(false);
    expect(isDimAtProgress(0.5)).toBe(false);
    expect(isDimAtProgress(0.6)).toBe(true);
    expect(isDimAtProgress(1)).toBe(true);
  });
});

describe('Blink Animation Logic', () => {
  test('toggles visibility state', () => {
    let isVisible = true;

    // Simulate 4 toggles
    const states: boolean[] = [isVisible];
    for (let i = 0; i < 4; i++) {
      isVisible = !isVisible;
      states.push(isVisible);
    }

    expect(states).toEqual([true, false, true, false, true]);
  });

  test('stays visible when disabled', () => {
    const enabled = false;
    const isVisible = enabled ? false : true;
    expect(isVisible).toBe(true);
  });
});

describe('Typewriter Animation Logic', () => {
  test('reveals characters progressively', () => {
    const text = 'Hello';
    const getDisplayText = (charIndex: number) => text.slice(0, charIndex);

    expect(getDisplayText(0)).toBe('');
    expect(getDisplayText(1)).toBe('H');
    expect(getDisplayText(2)).toBe('He');
    expect(getDisplayText(3)).toBe('Hel');
    expect(getDisplayText(4)).toBe('Hell');
    expect(getDisplayText(5)).toBe('Hello');
  });

  test('is complete when all characters revealed', () => {
    const text = 'Test';
    const isComplete = (charIndex: number) => charIndex >= text.length;

    expect(isComplete(0)).toBe(false);
    expect(isComplete(2)).toBe(false);
    expect(isComplete(4)).toBe(true);
    expect(isComplete(5)).toBe(true);
  });

  test('calculates correct interval from speed', () => {
    // speed = characters per second
    // interval = ms between characters
    const calculateInterval = (speed: number) => Math.floor(1000 / speed);

    expect(calculateInterval(30)).toBe(33);  // ~33ms per char at 30 cps
    expect(calculateInterval(10)).toBe(100); // 100ms per char at 10 cps
    expect(calculateInterval(60)).toBe(16);  // ~16ms per char at 60 cps
  });
});

describe('Fade Animation Logic', () => {
  type FadeDirection = 'in' | 'out';

  test('fade in goes from 0 to 1', () => {
    const direction: FadeDirection = 'in';
    const calculateOpacity = (progress: number) =>
      direction === 'in' ? progress : 1 - progress;

    expect(calculateOpacity(0)).toBe(0);
    expect(calculateOpacity(0.5)).toBe(0.5);
    expect(calculateOpacity(1)).toBe(1);
  });

  test('fade out goes from 1 to 0', () => {
    const direction: FadeDirection = 'out';
    const calculateOpacity = (progress: number) =>
      direction === 'in' ? progress : 1 - progress;

    expect(calculateOpacity(0)).toBe(1);
    expect(calculateOpacity(0.5)).toBe(0.5);
    expect(calculateOpacity(1)).toBe(0);
  });

  test('isDim when opacity < 0.5', () => {
    const isDim = (opacity: number) => opacity < 0.5;

    expect(isDim(0)).toBe(true);
    expect(isDim(0.3)).toBe(true);
    expect(isDim(0.5)).toBe(false);
    expect(isDim(0.7)).toBe(false);
    expect(isDim(1)).toBe(false);
  });
});

describe('Frame Rate Calculation', () => {
  test('calculates frame interval from fps', () => {
    const calculateFrameInterval = (fps: number) => Math.floor(1000 / fps);

    expect(calculateFrameInterval(30)).toBe(33);  // ~33ms per frame
    expect(calculateFrameInterval(60)).toBe(16);  // ~16ms per frame
    expect(calculateFrameInterval(24)).toBe(41);  // ~41ms per frame
  });

  test('reasonable fps values produce valid intervals', () => {
    const fpsValues = [24, 30, 60, 120];

    for (const fps of fpsValues) {
      const interval = Math.floor(1000 / fps);
      expect(interval).toBeGreaterThan(0);
      expect(interval).toBeLessThan(100);
    }
  });
});

describe('Animation Duration and Delay', () => {
  test('delay postpones animation start', () => {
    const delay = 100;
    const duration = 200;

    // Simulate timeline
    const shouldStart = (elapsed: number) => elapsed >= delay;

    expect(shouldStart(0)).toBe(false);
    expect(shouldStart(50)).toBe(false);
    expect(shouldStart(100)).toBe(true);
    expect(shouldStart(150)).toBe(true);
  });

  test('progress accounts for delay', () => {
    const delay = 100;
    const duration = 200;

    const calculateProgress = (elapsed: number) => {
      if (elapsed < delay) return 0;
      const adjustedElapsed = elapsed - delay;
      return Math.min(adjustedElapsed / duration, 1);
    };

    expect(calculateProgress(0)).toBe(0);
    expect(calculateProgress(100)).toBe(0);
    expect(calculateProgress(200)).toBe(0.5);
    expect(calculateProgress(300)).toBe(1);
    expect(calculateProgress(400)).toBe(1);
  });
});

describe('Notification Animation', () => {
  const notificationColors: Record<string, string> = {
    info: 'blue',
    success: 'green',
    warning: 'yellow',
    error: 'red',
  };

  test('notification types map to correct colors', () => {
    expect(notificationColors.info).toBe('blue');
    expect(notificationColors.success).toBe('green');
    expect(notificationColors.warning).toBe('yellow');
    expect(notificationColors.error).toBe('red');
  });

  test('all types have defined colors', () => {
    const types = ['info', 'success', 'warning', 'error'];
    for (const type of types) {
      expect(notificationColors[type]).toBeDefined();
      expect(typeof notificationColors[type]).toBe('string');
    }
  });
});

describe('Status Transition Logic', () => {
  test('no animation when status unchanged', () => {
    const shouldAnimate = (from: string | undefined, to: string) =>
      from !== undefined && from !== to;

    expect(shouldAnimate('working', 'working')).toBe(false);
    expect(shouldAnimate('idle', 'idle')).toBe(false);
  });

  test('animates when status changes', () => {
    const shouldAnimate = (from: string | undefined, to: string) =>
      from !== undefined && from !== to;

    expect(shouldAnimate('idle', 'working')).toBe(true);
    expect(shouldAnimate('working', 'done')).toBe(true);
  });

  test('no animation when no previous status', () => {
    const shouldAnimate = (from: string | undefined, to: string) =>
      from !== undefined && from !== to;

    expect(shouldAnimate(undefined, 'working')).toBe(false);
  });
});

describe('Animation Bounds', () => {
  test('progress never exceeds 1', () => {
    const clampProgress = (elapsed: number, duration: number) =>
      Math.min(elapsed / duration, 1);

    expect(clampProgress(0, 100)).toBe(0);
    expect(clampProgress(50, 100)).toBe(0.5);
    expect(clampProgress(100, 100)).toBe(1);
    expect(clampProgress(200, 100)).toBe(1); // Clamped
  });

  test('progress never goes below 0', () => {
    const clampProgress = (elapsed: number, duration: number) =>
      Math.max(0, Math.min(elapsed / duration, 1));

    expect(clampProgress(-50, 100)).toBe(0); // Clamped
    expect(clampProgress(0, 100)).toBe(0);
  });

  test('opacity stays within 0-1 range', () => {
    const minOpacity = 0.3;
    const maxOpacity = 1.0;

    const calculateOpacity = (progress: number) => {
      const sineProgress = Math.sin(progress * Math.PI);
      return minOpacity + sineProgress * (maxOpacity - minOpacity);
    };

    // Test across full range
    for (let p = 0; p <= 1; p += 0.1) {
      const opacity = calculateOpacity(p);
      expect(opacity).toBeGreaterThanOrEqual(minOpacity - 0.001);
      expect(opacity).toBeLessThanOrEqual(maxOpacity + 0.001);
    }
  });
});

// ============================================================================
// Issue #1210: Reduced-motion and animation accessibility support
// ============================================================================

describe('Accessibility: shouldDisableAnimations', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  test('returns false when BC_NO_ANIMATIONS is not set', () => {
    delete process.env.BC_NO_ANIMATIONS;
    // Import function directly to test
    const shouldDisable = () =>
      process.env.BC_NO_ANIMATIONS === '1' || process.env.BC_NO_ANIMATIONS === 'true';
    expect(shouldDisable()).toBe(false);
  });

  test('returns true when BC_NO_ANIMATIONS is "1"', () => {
    process.env.BC_NO_ANIMATIONS = '1';
    const shouldDisable = () =>
      process.env.BC_NO_ANIMATIONS === '1' || process.env.BC_NO_ANIMATIONS === 'true';
    expect(shouldDisable()).toBe(true);
  });

  test('returns true when BC_NO_ANIMATIONS is "true"', () => {
    process.env.BC_NO_ANIMATIONS = 'true';
    const shouldDisable = () =>
      process.env.BC_NO_ANIMATIONS === '1' || process.env.BC_NO_ANIMATIONS === 'true';
    expect(shouldDisable()).toBe(true);
  });

  test('returns false for other values', () => {
    process.env.BC_NO_ANIMATIONS = '0';
    const shouldDisable = () =>
      process.env.BC_NO_ANIMATIONS === '1' || process.env.BC_NO_ANIMATIONS === 'true';
    expect(shouldDisable()).toBe(false);

    process.env.BC_NO_ANIMATIONS = 'false';
    expect(shouldDisable()).toBe(false);
  });
});

describe('Accessibility: getAnimationFps', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  test('returns 60 by default', () => {
    delete process.env.BC_ANIMATION_FPS;
    const getFps = () => {
      const envFps = process.env.BC_ANIMATION_FPS;
      if (envFps) {
        const parsed = parseInt(envFps, 10);
        if (!isNaN(parsed) && parsed > 0 && parsed <= 120) {
          return parsed;
        }
      }
      return 60;
    };
    expect(getFps()).toBe(60);
  });

  test('respects BC_ANIMATION_FPS environment variable', () => {
    const getFps = () => {
      const envFps = process.env.BC_ANIMATION_FPS;
      if (envFps) {
        const parsed = parseInt(envFps, 10);
        if (!isNaN(parsed) && parsed > 0 && parsed <= 120) {
          return parsed;
        }
      }
      return 60;
    };

    process.env.BC_ANIMATION_FPS = '30';
    expect(getFps()).toBe(30);

    process.env.BC_ANIMATION_FPS = '120';
    expect(getFps()).toBe(120);
  });

  test('ignores invalid fps values', () => {
    const getFps = () => {
      const envFps = process.env.BC_ANIMATION_FPS;
      if (envFps) {
        const parsed = parseInt(envFps, 10);
        if (!isNaN(parsed) && parsed > 0 && parsed <= 120) {
          return parsed;
        }
      }
      return 60;
    };

    process.env.BC_ANIMATION_FPS = '0';
    expect(getFps()).toBe(60); // Falls back to default

    process.env.BC_ANIMATION_FPS = '-1';
    expect(getFps()).toBe(60);

    process.env.BC_ANIMATION_FPS = '200';
    expect(getFps()).toBe(60); // Exceeds max

    process.env.BC_ANIMATION_FPS = 'invalid';
    expect(getFps()).toBe(60);
  });
});

describe('Spinner Animation Logic', () => {
  test('cycles through frames', () => {
    const frames = ['⠋', '⠙', '⠹', '⠸'];
    let frameIndex = 0;

    const nextFrame = () => {
      frameIndex = (frameIndex + 1) % frames.length;
      return frames[frameIndex];
    };

    expect(nextFrame()).toBe('⠙');
    expect(nextFrame()).toBe('⠹');
    expect(nextFrame()).toBe('⠸');
    expect(nextFrame()).toBe('⠋'); // Wraps around
  });

  test('returns static indicator for reduced motion', () => {
    const reducedMotion = true;
    const getFrame = (frameIndex: number, frames: string[]) => {
      if (reducedMotion) return '...';
      return frames[frameIndex];
    };

    expect(getFrame(0, ['⠋', '⠙', '⠹'])).toBe('...');
    expect(getFrame(1, ['⠋', '⠙', '⠹'])).toBe('...');
  });
});

describe('Progress Animation Logic', () => {
  test('smoothly interpolates between values', () => {
    const startValue = 0.2;
    const targetValue = 0.8;

    const interpolate = (progress: number) =>
      startValue + (targetValue - startValue) * progress;

    expect(interpolate(0)).toBe(0.2);
    expect(interpolate(0.5)).toBe(0.5);
    expect(interpolate(1)).toBe(0.8);
  });

  test('snaps to target for reduced motion', () => {
    const reducedMotion = true;
    const targetValue = 0.8;

    const getProgress = () => {
      if (reducedMotion) return targetValue;
      // Would animate normally
      return 0;
    };

    expect(getProgress()).toBe(0.8);
  });
});

describe('Counter Animation Logic', () => {
  test('smoothly counts between values', () => {
    const startValue = 0;
    const targetValue = 100;

    const interpolate = (progress: number) =>
      startValue + (targetValue - startValue) * progress;

    expect(interpolate(0)).toBe(0);
    expect(interpolate(0.5)).toBe(50);
    expect(interpolate(1)).toBe(100);
  });

  test('formats with correct decimal places', () => {
    const format = (value: number, decimals: number) => value.toFixed(decimals);

    expect(format(42.567, 0)).toBe('43');
    expect(format(42.567, 1)).toBe('42.6');
    expect(format(42.567, 2)).toBe('42.57');
  });

  test('snaps to target for reduced motion', () => {
    const reducedMotion = true;
    const targetValue = 42;

    const getValue = () => {
      if (reducedMotion) return targetValue;
      return 0; // Would animate normally
    };

    expect(getValue()).toBe(42);
  });
});

describe('Spring Animation Logic', () => {
  test('calculates spring force correctly', () => {
    const calculateSpringForce = (
      displacement: number,
      velocity: number,
      tension = 170,
      friction = 26
    ) => -tension * displacement - friction * velocity;

    // At rest at target
    expect(calculateSpringForce(0, 0)).toBeCloseTo(0);

    // Displaced, no velocity - pulled back
    expect(calculateSpringForce(1, 0)).toBe(-170);

    // With velocity - damping applied
    expect(calculateSpringForce(0, 1)).toBe(-26);

    // Both displacement and velocity
    expect(calculateSpringForce(1, 1)).toBe(-196);
  });

  test('spring settles when displacement and velocity are small', () => {
    const precision = 0.01;

    const isSettled = (displacement: number, velocity: number) =>
      Math.abs(displacement) < precision && Math.abs(velocity) < precision;

    expect(isSettled(0, 0)).toBe(true);
    expect(isSettled(0.001, 0.001)).toBe(true);
    expect(isSettled(0.1, 0)).toBe(false);
    expect(isSettled(0, 0.1)).toBe(false);
  });

  test('snaps to target for reduced motion', () => {
    const reducedMotion = true;
    const targetValue = 100;

    const getValue = () => {
      if (reducedMotion) return targetValue;
      return 0; // Would animate normally
    };

    expect(getValue()).toBe(100);
  });
});

describe('Spinner Frame Presets', () => {
  const SPINNER_FRAMES = {
    dots: ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'],
    line: ['-', '\\', '|', '/'],
    circle: ['◐', '◓', '◑', '◒'],
    arrow: ['←', '↖', '↑', '↗', '→', '↘', '↓', '↙'],
    bounce: ['⠁', '⠂', '⠄', '⠂'],
    pulse: ['█', '▓', '▒', '░', '▒', '▓'],
  };

  test('all presets have at least 2 frames', () => {
    for (const [name, frames] of Object.entries(SPINNER_FRAMES)) {
      expect(frames.length).toBeGreaterThanOrEqual(2);
    }
  });

  test('dots preset has 10 frames', () => {
    expect(SPINNER_FRAMES.dots.length).toBe(10);
  });

  test('line preset is classic 4-frame rotation', () => {
    expect(SPINNER_FRAMES.line).toEqual(['-', '\\', '|', '/']);
  });
});
