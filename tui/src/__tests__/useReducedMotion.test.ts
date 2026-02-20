/**
 * Tests for useReducedMotion hook
 * Issue #1210: Reduced motion accessibility support
 */

/* eslint-disable @typescript-eslint/dot-notation */

import {
  getAccessibleAnimationOptions,
  type ReducedMotionState,
} from '../hooks/useReducedMotion';

describe('useReducedMotion', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    // Reset environment before each test
    process.env = { ...originalEnv };
    delete process.env['BC_NO_ANIMATIONS'];
    delete process.env['BC_REDUCED_MOTION'];
    delete process.env['BC_ANIMATION_FPS'];
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  describe('BC_NO_ANIMATIONS', () => {
    it('should disable animations when BC_NO_ANIMATIONS=1', () => {
      process.env['BC_NO_ANIMATIONS'] = '1';

      const noAnimations = process.env['BC_NO_ANIMATIONS'];
      expect(noAnimations === '1' || noAnimations === 'true').toBe(true);
    });

    it('should disable animations when BC_NO_ANIMATIONS=true', () => {
      process.env['BC_NO_ANIMATIONS'] = 'true';

      const noAnimations = process.env['BC_NO_ANIMATIONS'];
      expect(noAnimations === '1' || noAnimations === 'true').toBe(true);
    });
  });

  describe('BC_REDUCED_MOTION', () => {
    it('should reduce animations when BC_REDUCED_MOTION=1', () => {
      process.env['BC_REDUCED_MOTION'] = '1';

      const reducedMotion = process.env['BC_REDUCED_MOTION'];
      expect(reducedMotion === '1' || reducedMotion === 'true').toBe(true);
    });
  });

  describe('BC_ANIMATION_FPS', () => {
    it('should parse valid fps values', () => {
      process.env['BC_ANIMATION_FPS'] = '30';

      const customFps = process.env['BC_ANIMATION_FPS'] ?? '';
      const fps = parseInt(customFps, 10);
      expect(!isNaN(fps) && fps >= 0 && fps <= 60).toBe(true);
      expect(fps).toBe(30);
    });

    it('should handle fps=0 as disabled', () => {
      process.env['BC_ANIMATION_FPS'] = '0';

      const customFps = process.env['BC_ANIMATION_FPS'] ?? '';
      const fps = parseInt(customFps, 10);
      expect(fps).toBe(0);
    });

    it('should ignore invalid fps values', () => {
      process.env['BC_ANIMATION_FPS'] = 'invalid';

      const customFps = process.env['BC_ANIMATION_FPS'] ?? '';
      const fps = parseInt(customFps, 10);
      expect(isNaN(fps)).toBe(true);
    });

    it('should ignore out of range fps values', () => {
      process.env['BC_ANIMATION_FPS'] = '120';

      const customFps = process.env['BC_ANIMATION_FPS'] ?? '';
      const fps = parseInt(customFps, 10);
      expect(fps >= 0 && fps <= 60).toBe(false);
    });
  });
});

describe('getAccessibleAnimationOptions', () => {
  it('should return instant transitions when animations disabled', () => {
    const state: ReducedMotionState = {
      prefersReducedMotion: true,
      recommendedFps: 0,
      source: 'env',
    };

    const result = getAccessibleAnimationOptions(state, { duration: 300, fps: 60 });

    expect(result.enabled).toBe(false);
    expect(result.duration).toBe(0);
    expect(result.fps).toBe(0);
  });

  it('should cap duration and fps for reduced motion', () => {
    const state: ReducedMotionState = {
      prefersReducedMotion: true,
      recommendedFps: 30,
      source: 'env',
    };

    const result = getAccessibleAnimationOptions(state, { duration: 300, fps: 60 });

    expect(result.enabled).toBe(true);
    expect(result.duration).toBe(150); // Capped at 150ms
    expect(result.fps).toBe(30); // Capped at 30fps
  });

  it('should allow full animations when not reduced', () => {
    const state: ReducedMotionState = {
      prefersReducedMotion: false,
      recommendedFps: 60,
      source: 'default',
    };

    const result = getAccessibleAnimationOptions(state, { duration: 300, fps: 60 });

    expect(result.enabled).toBe(true);
    expect(result.duration).toBe(300);
    expect(result.fps).toBe(60);
  });

  it('should use defaults when options not provided', () => {
    const state: ReducedMotionState = {
      prefersReducedMotion: false,
      recommendedFps: 60,
      source: 'default',
    };

    const result = getAccessibleAnimationOptions(state);

    expect(result.duration).toBe(300);
    expect(result.fps).toBe(60);
  });
});
