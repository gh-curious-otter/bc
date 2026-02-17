/**
 * PerformanceOverlay Tests
 * Issue #1025: Performance monitoring dashboard
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, beforeEach, afterEach } from 'bun:test';
import { PerformanceOverlay } from '../PerformanceOverlay';

describe('PerformanceOverlay', () => {
  const originalEnv = process.env.BC_TUI_DEBUG;

  beforeEach(() => {
    // Reset env before each test
    delete process.env.BC_TUI_DEBUG;
  });

  afterEach(() => {
    // Restore original env
    if (originalEnv !== undefined) {
      process.env.BC_TUI_DEBUG = originalEnv;
    } else {
      delete process.env.BC_TUI_DEBUG;
    }
  });

  describe('visibility', () => {
    it('hides by default when debug mode disabled', () => {
      const { lastFrame } = render(<PerformanceOverlay />);
      expect(lastFrame()).toBe('');
    });

    it('shows when forceShow is true', () => {
      const { lastFrame } = render(<PerformanceOverlay forceShow />);
      const output = lastFrame() ?? '';
      expect(output).toContain('FPS');
    });

    it('shows when BC_TUI_DEBUG=1', () => {
      process.env.BC_TUI_DEBUG = '1';
      const { lastFrame } = render(<PerformanceOverlay />);
      const output = lastFrame() ?? '';
      expect(output).toContain('FPS');
    });
  });

  describe('display', () => {
    it('shows FPS counter', () => {
      const { lastFrame } = render(<PerformanceOverlay forceShow />);
      const output = lastFrame() ?? '';
      expect(output).toContain('FPS');
    });

    it('shows frame time with target', () => {
      const { lastFrame } = render(<PerformanceOverlay forceShow />);
      const output = lastFrame() ?? '';
      expect(output).toContain('Frame:');
      expect(output).toContain('target:');
    });

    it('shows detailed metrics when detailed=true', () => {
      const { lastFrame } = render(<PerformanceOverlay forceShow detailed />);
      const output = lastFrame() ?? '';
      expect(output).toContain('Min/Max:');
      expect(output).toContain('Frames:');
    });
  });

  describe('performance targets', () => {
    it('uses 24fps as target (41.67ms frame time)', () => {
      const { lastFrame } = render(<PerformanceOverlay forceShow />);
      const output = lastFrame() ?? '';
      // Target should be approximately 41.7ms
      expect(output).toContain('41.');
    });
  });
});

describe('PerformanceOverlay - Performance Status', () => {
  it('renders without performance context', () => {
    const { lastFrame } = render(<PerformanceOverlay forceShow />);
    const output = lastFrame() ?? '';
    // Should render even without PerformanceProvider
    expect(output).toContain('FPS');
  });
});
