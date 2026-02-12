/**
 * Default theme definitions for bc TUI
 *
 * Provides dark and light themes optimized for terminal display.
 */

import type { Theme, ThemeColors } from './types';

/**
 * Dark theme - default for most terminals
 */
export const darkTheme: Theme = {
  name: 'dark',
  mode: 'dark',
  colors: {
    // Primary colors
    primary: 'cyan',
    secondary: 'blue',
    accent: 'magenta',

    // Text colors
    text: 'white',
    textMuted: 'gray',
    textInverse: 'black',

    // Status colors
    success: 'green',
    warning: 'yellow',
    error: 'red',
    info: 'cyan',

    // Agent state colors
    agentIdle: 'gray',
    agentWorking: 'blue',
    agentDone: 'green',
    agentStuck: 'red',
    agentError: 'red',

    // UI element colors
    border: 'gray',
    borderFocused: 'cyan',
    selection: 'cyan',
    highlight: 'yellow',

    // Component-specific
    headerTitle: 'cyan',
    footerText: 'gray',
    badge: 'magenta',
  },
};

/**
 * Light theme - for light terminal backgrounds
 */
export const lightTheme: Theme = {
  name: 'light',
  mode: 'light',
  colors: {
    // Primary colors
    primary: 'blue',
    secondary: 'cyan',
    accent: 'magenta',

    // Text colors
    text: 'black',
    textMuted: 'gray',
    textInverse: 'white',

    // Status colors
    success: 'green',
    warning: 'yellow',
    error: 'red',
    info: 'blue',

    // Agent state colors
    agentIdle: 'gray',
    agentWorking: 'blue',
    agentDone: 'green',
    agentStuck: 'red',
    agentError: 'red',

    // UI element colors
    border: 'gray',
    borderFocused: 'blue',
    selection: 'blue',
    highlight: 'yellow',

    // Component-specific
    headerTitle: 'blue',
    footerText: 'gray',
    badge: 'magenta',
  },
};

/**
 * Get theme by name
 */
export function getTheme(name: 'dark' | 'light'): Theme {
  return name === 'light' ? lightTheme : darkTheme;
}

/**
 * Apply color overrides to a theme
 */
export function applyOverrides(
  theme: Theme,
  overrides: Partial<ThemeColors>
): Theme {
  return {
    ...theme,
    colors: {
      ...theme.colors,
      ...overrides,
    },
  };
}
