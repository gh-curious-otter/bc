/**
 * Timeline - Activity timeline visualization
 * Issue #1046 - Data visualization components
 */

import React, { memo, useMemo } from 'react';
import { Box, Text } from 'ink';

export interface TimelineSegment {
  /** Start time (ISO string or timestamp) */
  start: string | number;
  /** End time (ISO string or timestamp) */
  end: string | number;
  /** Status during this period */
  status: 'working' | 'idle' | 'stuck' | 'error' | 'done';
  /** Label for the segment (e.g., agent name or task) */
  label?: string;
}

export interface TimelineProps {
  /** Array of timeline segments */
  segments: TimelineSegment[];
  /** Width in characters (default: 40) */
  width?: number;
  /** Show time labels (default: true) */
  showTimeLabels?: boolean;
  /** Label for the timeline */
  label?: string;
  /** Time range to display (auto-calculated if not provided) */
  timeRange?: { start: number; end: number };
}

// Status characters for different states
const STATUS_CHARS: Record<TimelineSegment['status'], string> = {
  working: '█',
  idle: '░',
  stuck: '▓',
  error: '▒',
  done: '▄',
};

// Colors for different states
const STATUS_COLORS: Record<TimelineSegment['status'], string> = {
  working: 'green',
  idle: 'gray',
  stuck: 'yellow',
  error: 'red',
  done: 'cyan',
};

/**
 * Parse timestamp to milliseconds
 */
function parseTime(time: string | number): number {
  if (typeof time === 'number') return time;
  return new Date(time).getTime();
}

/**
 * Format timestamp for display
 */
function formatTimeLabel(ms: number): string {
  const date = new Date(ms);
  return date.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
}

/**
 * Timeline component showing activity periods
 */
export const Timeline = memo(function Timeline({
  segments,
  width = 40,
  showTimeLabels = true,
  label,
  timeRange,
}: TimelineProps): React.ReactElement {
  // Calculate time range from segments if not provided
  const range = useMemo(() => {
    if (timeRange) return timeRange;
    if (segments.length === 0) {
      const now = Date.now();
      return { start: now - 3600000, end: now }; // Last hour
    }

    let minTime = Infinity;
    let maxTime = -Infinity;

    for (const seg of segments) {
      const start = parseTime(seg.start);
      const end = parseTime(seg.end);
      if (start < minTime) minTime = start;
      if (end > maxTime) maxTime = end;
    }

    return { start: minTime, end: maxTime };
  }, [segments, timeRange]);

  // Build the timeline string
  const { timelineChars, timelineColors } = useMemo(() => {
    const chars: string[] = Array.from({ length: width }, () => '░');
    const colors: string[] = Array.from({ length: width }, () => 'gray');
    const duration = range.end - range.start;

    if (duration <= 0) {
      return { timelineChars: chars, timelineColors: colors };
    }

    // Sort segments by start time and process
    const sortedSegments = [...segments].sort(
      (a, b) => parseTime(a.start) - parseTime(b.start)
    );

    for (const seg of sortedSegments) {
      const start = parseTime(seg.start);
      const end = parseTime(seg.end);

      // Calculate positions in the timeline
      const startPos = Math.floor(((start - range.start) / duration) * width);
      const endPos = Math.ceil(((end - range.start) / duration) * width);

      // Fill in the segment
      for (let i = Math.max(0, startPos); i < Math.min(width, endPos); i++) {
        chars[i] = STATUS_CHARS[seg.status];
        colors[i] = STATUS_COLORS[seg.status];
      }
    }

    return { timelineChars: chars, timelineColors: colors };
  }, [segments, range, width]);

  if (segments.length === 0) {
    return (
      <Box>
        {label && <Text dimColor>{label}: </Text>}
        <Text dimColor>{'─'.repeat(width)} (no data)</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <Box>
        {label && <Text dimColor>{label}: </Text>}
        {timelineChars.map((char, idx) => (
          <Text key={idx} color={timelineColors[idx]}>{char}</Text>
        ))}
      </Box>
      {showTimeLabels && (
        <Box>
          {label && <Text dimColor>{''.padEnd(label.length + 2)}</Text>}
          <Text dimColor>
            {formatTimeLabel(range.start)}{''.padEnd(width - 10)}{formatTimeLabel(range.end)}
          </Text>
        </Box>
      )}
    </Box>
  );
});

/**
 * AgentTimeline - Timeline for a single agent showing work periods
 */
export interface AgentTimelineProps {
  /** Agent name */
  agent: string;
  /** Array of timeline segments */
  segments: TimelineSegment[];
  /** Width in characters (default: 30) */
  width?: number;
}

export const AgentTimeline = memo(function AgentTimeline({
  agent,
  segments,
  width = 30,
}: AgentTimelineProps): React.ReactElement {
  const agentLabel = agent.padEnd(10);

  // Build timeline chars without time labels
  const { timelineChars, timelineColors } = useMemo(() => {
    if (segments.length === 0) {
      return {
        timelineChars: Array.from({ length: width }, () => '░'),
        timelineColors: Array.from({ length: width }, () => 'gray'),
      };
    }

    let minTime = Infinity;
    let maxTime = -Infinity;

    for (const seg of segments) {
      const start = parseTime(seg.start);
      const end = parseTime(seg.end);
      if (start < minTime) minTime = start;
      if (end > maxTime) maxTime = end;
    }

    const timeRange = { start: minTime, end: maxTime };
    const duration = timeRange.end - timeRange.start;
    const chars: string[] = Array.from({ length: width }, () => '░');
    const colors: string[] = Array.from({ length: width }, () => 'gray');

    if (duration > 0) {
      for (const seg of segments) {
        const start = parseTime(seg.start);
        const end = parseTime(seg.end);
        const startPos = Math.floor(((start - timeRange.start) / duration) * width);
        const endPos = Math.ceil(((end - timeRange.start) / duration) * width);

        for (let i = Math.max(0, startPos); i < Math.min(width, endPos); i++) {
          chars[i] = STATUS_CHARS[seg.status];
          colors[i] = STATUS_COLORS[seg.status];
        }
      }
    }

    return { timelineChars: chars, timelineColors: colors };
  }, [segments, width]);

  return (
    <Box>
      <Text color="cyan">{agentLabel}</Text>
      <Text dimColor>|</Text>
      {timelineChars.map((char, idx) => (
        <Text key={idx} color={timelineColors[idx] ?? 'gray'}>{char}</Text>
      ))}
      <Text dimColor>|</Text>
    </Box>
  );
});

/**
 * TimelineLegend - Legend for timeline status colors
 */
export const TimelineLegend = memo(function TimelineLegend(): React.ReactElement {
  return (
    <Box>
      <Text dimColor>Legend: </Text>
      <Text color={STATUS_COLORS.working}>{STATUS_CHARS.working}</Text>
      <Text dimColor>=working </Text>
      <Text color={STATUS_COLORS.idle}>{STATUS_CHARS.idle}</Text>
      <Text dimColor>=idle </Text>
      <Text color={STATUS_COLORS.stuck}>{STATUS_CHARS.stuck}</Text>
      <Text dimColor>=stuck </Text>
      <Text color={STATUS_COLORS.done}>{STATUS_CHARS.done}</Text>
      <Text dimColor>=done</Text>
    </Box>
  );
});

export default Timeline;
