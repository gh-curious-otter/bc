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
 * Matrix theme - Classic green terminal aesthetic
 * Inspired by The Matrix with phosphor green accents
 */
export const matrixTheme: Theme = {
  name: 'matrix',
  mode: 'dark',
  colors: {
    // Primary colors - all green variants
    primary: 'green',
    secondary: 'greenBright',
    accent: 'greenBright',

    // Text colors
    text: 'green',
    textMuted: 'gray',
    textInverse: 'black',

    // Status colors
    success: 'greenBright',
    warning: 'yellow',
    error: 'red',
    info: 'green',

    // Agent state colors
    agentIdle: 'gray',
    agentWorking: 'greenBright',
    agentDone: 'green',
    agentStuck: 'yellow',
    agentError: 'red',

    // UI element colors
    border: 'green',
    borderFocused: 'greenBright',
    selection: 'greenBright',
    highlight: 'greenBright',

    // Component-specific
    headerTitle: 'greenBright',
    footerText: 'green',
    badge: 'greenBright',
  },
};

/**
 * Synthwave theme - Retro-futuristic cyberpunk aesthetics
 * Purple, pink, and cyan neon colors
 */
export const synthwaveTheme: Theme = {
  name: 'synthwave',
  mode: 'dark',
  colors: {
    // Primary colors - neon palette
    primary: 'magenta',
    secondary: 'cyan',
    accent: 'magentaBright',

    // Text colors
    text: 'white',
    textMuted: 'gray',
    textInverse: 'black',

    // Status colors
    success: 'cyanBright',
    warning: 'yellow',
    error: 'redBright',
    info: 'magenta',

    // Agent state colors
    agentIdle: 'gray',
    agentWorking: 'cyanBright',
    agentDone: 'cyan',
    agentStuck: 'yellow',
    agentError: 'redBright',

    // UI element colors
    border: 'magenta',
    borderFocused: 'magentaBright',
    selection: 'cyan',
    highlight: 'magentaBright',

    // Component-specific
    headerTitle: 'magentaBright',
    footerText: 'magenta',
    badge: 'cyanBright',
  },
};

/**
 * High contrast theme - Accessibility-focused
 * Bold, clear colors for maximum readability
 */
export const highContrastTheme: Theme = {
  name: 'high-contrast',
  mode: 'dark',
  colors: {
    // Primary colors - bright and bold
    primary: 'whiteBright',
    secondary: 'cyanBright',
    accent: 'yellowBright',

    // Text colors
    text: 'whiteBright',
    textMuted: 'white',
    textInverse: 'black',

    // Status colors - high visibility
    success: 'greenBright',
    warning: 'yellowBright',
    error: 'redBright',
    info: 'cyanBright',

    // Agent state colors
    agentIdle: 'white',
    agentWorking: 'cyanBright',
    agentDone: 'greenBright',
    agentStuck: 'yellowBright',
    agentError: 'redBright',

    // UI element colors
    border: 'whiteBright',
    borderFocused: 'yellowBright',
    selection: 'yellowBright',
    highlight: 'yellowBright',

    // Component-specific
    headerTitle: 'whiteBright',
    footerText: 'white',
    badge: 'cyanBright',
  },
};

/** Available theme names */
export type ThemeName = 'dark' | 'light' | 'matrix' | 'synthwave' | 'high-contrast';

/** All available themes */
export const themes: Record<ThemeName, Theme> = {
  dark: darkTheme,
  light: lightTheme,
  matrix: matrixTheme,
  synthwave: synthwaveTheme,
  'high-contrast': highContrastTheme,
};

/**
 * Get theme by name
 */
export function getTheme(name: ThemeName): Theme {
  return themes[name];
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
