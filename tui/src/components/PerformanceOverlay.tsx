/**
 * PerformanceOverlay - Dev mode performance monitoring display
 * Issue #1025: Performance monitoring dashboard
 *
 * Shows FPS counter, render times, and performance warnings.
 * Only visible when BC_TUI_DEBUG=1 or debugEnabled is true.
 */

import React, { useState, useEffect, useCallback, memo } from 'react';
import { Box, Text } from 'ink';
import { usePerformanceOptional } from '../hooks/PerformanceContext';

/** Target frame rate - 24fps for TUI (41.67ms per frame) */
const TARGET_FPS = 24;
const TARGET_FRAME_TIME_MS = 1000 / TARGET_FPS; // ~41.67ms

/** Warning threshold - warn if average frame time exceeds this */
const WARNING_THRESHOLD_MS = TARGET_FRAME_TIME_MS * 1.5; // ~62.5ms (16fps)

export interface PerformanceOverlayProps {
  /** Force show overlay regardless of debug mode */
  forceShow?: boolean;
  /** Position of the overlay */
  position?: 'top-right' | 'bottom-right' | 'top-left' | 'bottom-left';
  /** Show detailed metrics */
  detailed?: boolean;
}

interface FrameStats {
  fps: number;
  frameTime: number;
  avgFrameTime: number;
  minFrameTime: number;
  maxFrameTime: number;
  frameCount: number;
}

/**
 * Performance status indicator
 */
function getPerformanceStatus(avgFrameTime: number): {
  status: 'good' | 'warning' | 'critical';
  color: string;
  label: string;
} {
  if (avgFrameTime <= TARGET_FRAME_TIME_MS) {
    return { status: 'good', color: 'green', label: 'OK' };
  } else if (avgFrameTime <= WARNING_THRESHOLD_MS) {
    return { status: 'warning', color: 'yellow', label: 'SLOW' };
  } else {
    return { status: 'critical', color: 'red', label: 'CRIT' };
  }
}

/**
 * FPS Counter component - tracks frame timing
 */
function FPSCounter({
  onStats,
}: {
  onStats: (stats: FrameStats) => void;
}): null {
  const frameTimesRef = React.useRef<number[]>([]);
  const lastFrameTimeRef = React.useRef<number>(performance.now());
  const frameCountRef = React.useRef<number>(0);

  useEffect(() => {
    const updateFPS = () => {
      const now = performance.now();
      const frameTime = now - lastFrameTimeRef.current;
      lastFrameTimeRef.current = now;
      frameCountRef.current++;

      // Keep last 60 frame times for averaging
      frameTimesRef.current.push(frameTime);
      if (frameTimesRef.current.length > 60) {
        frameTimesRef.current.shift();
      }

      const times = frameTimesRef.current;
      const avgFrameTime = times.reduce((a, b) => a + b, 0) / times.length;
      const minFrameTime = Math.min(...times);
      const maxFrameTime = Math.max(...times);
      const fps = 1000 / avgFrameTime;

      onStats({
        fps,
        frameTime,
        avgFrameTime,
        minFrameTime,
        maxFrameTime,
        frameCount: frameCountRef.current,
      });
    };

    // Update at target frame rate
    const interval = setInterval(updateFPS, TARGET_FRAME_TIME_MS);
    return () => {
      clearInterval(interval);
    };
  }, [onStats]);

  return null;
}

/**
 * Performance Overlay Component
 * Displays FPS counter and performance metrics in dev mode
 */
export const PerformanceOverlay = memo(function PerformanceOverlay({
  forceShow = false,
  position = 'top-right',
  detailed = false,
}: PerformanceOverlayProps): React.ReactElement | null {
  const perf = usePerformanceOptional();
  const [stats, setStats] = useState<FrameStats>({
    fps: 0,
    frameTime: 0,
    avgFrameTime: 0,
    minFrameTime: 0,
    maxFrameTime: 0,
    frameCount: 0,
  });

  const handleStats = useCallback((newStats: FrameStats) => {
    setStats(newStats);

    // Record frame time metric if performance context available
    if (perf) {
      perf.recordMetric('frame:time', newStats.frameTime);
    }
  }, [perf]);

  // Check if we should show the overlay
  const debugEnabled = perf?.debugEnabled ?? false;
  const shouldShow = forceShow || debugEnabled || process.env.BC_TUI_DEBUG === '1';

  if (!shouldShow) {
    return null;
  }

  const { status, color, label } = getPerformanceStatus(stats.avgFrameTime);
  const fpsDisplay = Math.round(stats.fps);
  const avgMs = stats.avgFrameTime.toFixed(1);

  // Position styles
  const positionStyle: Record<string, string | number> = {};
  if (position.includes('right')) {
    positionStyle.alignSelf = 'flex-end';
  }

  return (
    <>
      <FPSCounter onStats={handleStats} />
      <Box
        flexDirection="column"
        borderStyle="single"
        borderColor={color}
        paddingX={1}
        {...positionStyle}
      >
        {/* FPS Display */}
        <Box>
          <Text color={color} bold>
            {fpsDisplay} FPS
          </Text>
          <Text dimColor> | </Text>
          <Text color={color}>{label}</Text>
        </Box>

        {/* Frame Time */}
        <Box>
          <Text dimColor>Frame: </Text>
          <Text>{avgMs}ms</Text>
          <Text dimColor> (target: {TARGET_FRAME_TIME_MS.toFixed(1)}ms)</Text>
        </Box>

        {/* Detailed Metrics */}
        {detailed && (
          <>
            <Box>
              <Text dimColor>Min/Max: </Text>
              <Text>
                {stats.minFrameTime.toFixed(1)}ms / {stats.maxFrameTime.toFixed(1)}ms
              </Text>
            </Box>
            <Box>
              <Text dimColor>Frames: </Text>
              <Text>{stats.frameCount}</Text>
            </Box>
            {perf && (
              <Box>
                <Text dimColor>Metrics: </Text>
                <Text>{perf.totalMeasurements}</Text>
              </Box>
            )}
          </>
        )}

        {/* Performance Warning */}
        {status !== 'good' && (
          <Box marginTop={1}>
            <Text color={color} bold>
              {status === 'critical' ? '! PERF CRITICAL' : '! PERF WARNING'}
            </Text>
          </Box>
        )}
      </Box>
    </>
  );
});

export default PerformanceOverlay;
