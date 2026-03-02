/**
 * Shared output colorization utilities
 *
 * #1840: Extracted from AgentDetailView for reuse in AgentPeekPanel
 * #1161: Semantic colors for agent output readability
 * #1844: ANSI passthrough for log streaming output
 */

import React from 'react';
import { Text } from 'ink';

// ANSI escape code regex - detects SGR (Select Graphic Rendition) sequences
// eslint-disable-next-line no-control-regex
const ANSI_REGEX = /\x1b\[[0-9;]*m/;

/**
 * Check if a line contains ANSI escape codes.
 * #1844: Log streaming backend preserves ANSI codes in output.
 */
export function hasAnsiCodes(line: string): boolean {
  return ANSI_REGEX.test(line);
}

/**
 * Check if a line is a peek header (e.g., "=== agent-name (last 50 lines) ===").
 * #1844: Strip these headers from displayed output.
 */
export function isPeekHeader(line: string): boolean {
  return /^=== .+ \(last \d+ lines\) ===$/.test(line.trim());
}

/**
 * Colorize output line based on content patterns.
 * #1161: Apply semantic colors to agent output for better readability.
 * #1844: Pass through lines that already contain ANSI escape codes from log streaming.
 *
 * Patterns: errors (red), warnings (yellow), success (green), info (cyan)
 */
export function colorizeOutputLine(line: string): React.ReactElement {
  // #1844: If line already has ANSI codes from log streaming, render as-is.
  // Ink 4.x renders embedded ANSI escape sequences in Text content.
  if (hasAnsiCodes(line)) {
    return <Text>{line}</Text>;
  }

  const trimmed = line.trim().toLowerCase();

  // Error patterns
  if (
    trimmed.includes('error') ||
    trimmed.includes('failed') ||
    trimmed.includes('exception') ||
    trimmed.startsWith('✗') ||
    trimmed.startsWith('x ')
  ) {
    return <Text color="red">{line}</Text>;
  }

  // Warning patterns
  if (
    trimmed.includes('warning') ||
    trimmed.includes('warn') ||
    trimmed.includes('deprecated') ||
    trimmed.startsWith('⚠')
  ) {
    return <Text color="yellow">{line}</Text>;
  }

  // Success patterns
  if (
    trimmed.includes('success') ||
    trimmed.includes('passed') ||
    trimmed.includes('complete') ||
    trimmed.startsWith('✓') ||
    trimmed.startsWith('✔')
  ) {
    return <Text color="green">{line}</Text>;
  }

  // Tool/command patterns (cyan for actions)
  if (
    trimmed.startsWith('>') ||
    trimmed.startsWith('$') ||
    trimmed.includes('running') ||
    trimmed.includes('executing')
  ) {
    return <Text color="cyan">{line}</Text>;
  }

  // File paths (dim white)
  if (trimmed.match(/^[./~].*\.(tsx?|jsx?|go|py|md|json)$/)) {
    return <Text color="white">{line}</Text>;
  }

  // Default: dimmed text
  return <Text dimColor>{line}</Text>;
}
