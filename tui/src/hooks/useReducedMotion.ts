/**
 * useReducedMotion - Accessibility hook for reduced motion preference
 * Issue #1210: Add reduced-motion and animation accessibility support
 *
 * Checks for:
 * 1. BC_NO_ANIMATIONS environment variable
 * 2. Config setting (tui.animations)
 * 3. System prefers-reduced-motion (via env hint)
 */

/* eslint-disable @typescript-eslint/dot-notation */

import { useMemo } from 'react';

export interface ReducedMotionState {
  /** Whether animations should be disabled */
  prefersReducedMotion: boolean;
  /** Recommended frame rate (0 = static, 30 = reduced, 60 = full) */
  recommendedFps: number;
  /** Source of the preference */
  source: 'env' | 'config' | 'system' | 'default';
}

/**
 * Hook to check if reduced motion is preferred
 *
 * Priority:
 * 1. BC_NO_ANIMATIONS=1 environment variable
 * 2. BC_REDUCED_MOTION=1 environment variable
 * 3. BC_ANIMATION_FPS environment variable for custom fps
 * 4. Default: animations enabled at 60fps
 */
export function useReducedMotion(): ReducedMotionState {
  return useMemo(() => {
    // Check BC_NO_ANIMATIONS environment variable
    const noAnimations = process.env['BC_NO_ANIMATIONS'];
    if (noAnimations === '1' || noAnimations === 'true') {
      return {
        prefersReducedMotion: true,
        recommendedFps: 0,
        source: 'env' as const,
      };
    }

    // Check BC_REDUCED_MOTION for reduced (not disabled) animations
    const reducedMotion = process.env['BC_REDUCED_MOTION'];
    if (reducedMotion === '1' || reducedMotion === 'true') {
      return {
        prefersReducedMotion: true,
        recommendedFps: 30,
        source: 'env' as const,
      };
    }

    // Check BC_ANIMATION_FPS for custom frame rate
    const customFps = process.env['BC_ANIMATION_FPS'];
    if (customFps) {
      const fps = parseInt(customFps, 10);
      if (!isNaN(fps) && fps >= 0 && fps <= 60) {
        return {
          prefersReducedMotion: fps === 0,
          recommendedFps: fps,
          source: 'env' as const,
        };
      }
    }

    // Default: full animations at 60fps
    return {
      prefersReducedMotion: false,
      recommendedFps: 60,
      source: 'default' as const,
    };
  }, []);
}

/**
 * Get animation options adjusted for reduced motion preference
 */
export function getAccessibleAnimationOptions(
  reducedMotion: ReducedMotionState,
  options: { duration?: number; fps?: number } = {}
): { duration: number; fps: number; enabled: boolean } {
  if (reducedMotion.prefersReducedMotion && reducedMotion.recommendedFps === 0) {
    // Instant transitions
    return { duration: 0, fps: 0, enabled: false };
  }

  if (reducedMotion.prefersReducedMotion) {
    // Reduced animations
    return {
      duration: Math.min(options.duration ?? 300, 150), // Cap at 150ms
      fps: Math.min(options.fps ?? 60, reducedMotion.recommendedFps),
      enabled: true,
    };
  }

  // Full animations
  return {
    duration: options.duration ?? 300,
    fps: options.fps ?? 60,
    enabled: true,
  };
}

export default useReducedMotion;
